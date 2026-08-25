#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "$project_root/../.." && pwd)"

python3 "$repo_root/blueprint/mobile/scripts/validate_blueprint_specs.py"
python3 "$repo_root/blueprint/scripts/validate_i18n_contract.py"
python3 "$project_root/scripts/check-i18n-catalogs.py"
python3 "$project_root/scripts/generate-mobile-client.py"
python3 "$project_root/scripts/check-startup-snapshot.py"
python3 "$project_root/scripts/test_upgrade_contract.py"
jq -e '."uni-app-x".renderer == "vdom"' "$project_root/manifest.json" >/dev/null
rg -n --glob '*.uts' --glob '*.uvue' '\bany\b|sslVerify\s*:\s*false|uni\.(setStorage|setStorageSync)\([^\n]*(token|password|otp)' "$project_root" && {
  printf 'mobile source contains a forbidden pattern\n' >&2
  exit 1
} || true

if rg -n --glob '*.uvue' 'min-height\s*:\s*[0-9.]+%' "$project_root"; then
  printf 'mobile source contains an iOS-unsupported percentage min-height value\n' >&2
  exit 1
fi

if rg -n 'getString\(' "$project_root/src/core/network/ak-http-client.uts"; then
  printf 'mobile HTTP client must not invoke UTSJSONObject methods on uni.request response headers\n' >&2
  exit 1
fi
if rg -n 'request\.headers\.forEach' "$project_root/src/core/network/ak-http-client.uts"; then
  printf 'mobile HTTP client must not dynamically bridge request header Map entries on iOS\n' >&2
  exit 1
fi
if rg -n "headers\['(Authorization|X-AK-Device-Key)'\]" "$project_root/src/core/network/ak-http-client.uts"; then
  printf 'mobile HTTP client must declare dynamic iOS request headers in the UTSJSONObject literal\n' >&2
  exit 1
fi
if rg -n "getStorageSync\('ak.auth.device-key'\)|setStorageSync\('ak.auth.device-key'" "$project_root/src"; then
  printf 'mobile authentication must not bridge the installation UUID through plain iOS storage\n' >&2
  exit 1
fi
if ! rg -Fq "'X-AK-Device-Key': this.deviceKey" "$project_root/src/core/network/ak-http-client.uts"; then
  printf 'mobile HTTP client must inject the primitive runtime device UUID directly into request headers\n' >&2
  exit 1
fi
if ! rg -Fq 'JSON.parseObject(JSON.stringify(headerSource))' "$project_root/src/core/network/ak-http-client.uts"; then
  printf 'mobile HTTP headers must materialize primitive values before crossing the iOS request bridge\n' >&2
  exit 1
fi

login_page="$project_root/pages/auth/login/index.uvue"
if rg -n 'variant="secondary"' "$login_page"; then
  printf 'login page must keep recovery and registration as text links, not secondary buttons\n' >&2
  exit 1
fi
for required_login_pattern in 'class="auth-link-row"' 'class="auth-link"' 'class="auth-link auth-link-register"' 'height:44px' 'margin-left:8px'; do
  if ! rg -q "$required_login_pattern" "$login_page"; then
    printf 'login page is missing the required accessible secondary auth-link layout: %s\n' "$required_login_pattern" >&2
    exit 1
  fi
done

auth_runtime="$project_root/src/features/auth/auth-runtime.uts"
if rg -n "ak-mobile-' \+ Date\.now|ak-mobile-.*Date\.now" "$auth_runtime"; then
  printf 'mobile authentication device keys must use the backend UUID contract\n' >&2
  exit 1
fi
for uuid_pattern in "randomHex(8)" "'-4' + this.randomHex(3)" "this.randomHex(12)"; do
  if ! rg -Fq "$uuid_pattern" "$auth_runtime"; then
    printf 'mobile authentication device key generator is missing UUID v4 structure: %s\n' "$uuid_pattern" >&2
    exit 1
  fi
done
if ! rg -Fq 'activeClient.setDeviceKey(value)' "$auth_runtime" || ! rg -Fq 'this.resetDeviceKey()' "$auth_runtime" || ! rg -Fq '!this.isDeviceKey(credential.deviceKey)' "$auth_runtime"; then
  printf 'mobile authentication must keep fresh and restored secure device UUIDs aligned with the HTTP client\n' >&2
  exit 1
fi

back_button="$project_root/components/ak-ui/ak-back-button/ak-back-button.uvue"
if rg -n '<navigator|open-type="navigateBack"' "$back_button"; then
  printf 'AK back button must use explicit uni.navigateBack handling instead of navigator navigateBack\n' >&2
  exit 1
fi
if ! rg -q 'uni\.navigateBack\(' "$back_button" || ! rg -q 'fallbackUrl' "$back_button"; then
  printf 'AK back button must support explicit history navigation with a fallback URL\n' >&2
  exit 1
fi
for guest_page in auth/forgot-password auth/register auth/reset-password auth/verify-contact legal/privacy legal/terms legal/page; do
  guest_page_file="$project_root/pages/$guest_page/index.uvue"
  if ! rg -q 'fallback-url="/pages/auth/login/index"' "$guest_page_file"; then
    printf 'guest or legal page is missing the login fallback: %s\n' "$guest_page" >&2
    exit 1
  fi
done

theme_root="$project_root/components/ak-ui/ak-theme-root/ak-theme-root.uvue"
theme_tokens="$project_root/design-system/tokens.scss"
if ! rg -q 'padding-top:var\(--ak-safe-top\)' "$theme_root" || ! rg -q -- '--ak-safe-top:var\(--status-bar-height\)' "$theme_tokens"; then
  printf 'mobile pages must receive the native status-bar inset from the shared theme root\n' >&2
  exit 1
fi

button_component="$project_root/components/ak-ui/ak-button/ak-button.uvue"
for label_class in ak-button__label-primary ak-button__label-secondary ak-button__label-danger; do
  if ! rg -q "$label_class" "$button_component"; then
    printf 'AK button is missing an explicit slot-safe text style: %s\n' "$label_class" >&2
    exit 1
  fi
done

for tab_icon in \
  home-default.png home-selected.png \
  notifications-default.png notifications-selected.png \
  profile-default.png profile-selected.png; do
  if [[ ! -f "$project_root/static/ak-tabbar/$tab_icon" ]] || ! rg -q "static/ak-tabbar/$tab_icon" "$project_root/pages.json"; then
    printf 'mobile TabBar icon is missing or not declared: %s\n' "$tab_icon" >&2
    exit 1
  fi
done

if rg -n --glob '*.uvue' 'height\s*:\s*(620|640)px' "$project_root/pages"; then
  printf 'mobile pages must not use fixed 620/640px content heights\n' >&2
  exit 1
fi

for target in android ios harmony; do
  test -x "$project_root/scripts/build-platform.sh"
  rg -q "$target" "$project_root/scripts/build-platform.sh"
done

printf 'AK Mobile static project checks passed. Platform builds are separate evidence gates.\n'
