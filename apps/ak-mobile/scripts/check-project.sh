#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "$project_root/../.." && pwd)"

python3 "$repo_root/blueprint/mobile/scripts/validate_blueprint_specs.py"
python3 "$repo_root/blueprint/scripts/validate_i18n_contract.py"
python3 "$project_root/scripts/check-i18n-catalogs.py"
python3 "$project_root/scripts/generate-oauth-snapshot-uts.py"
python3 "$project_root/scripts/test_notification_contract.py"
python3 "$project_root/scripts/test_bookmark_filter_contract.py"
python3 "$project_root/scripts/test_scanner_contract.py"
python3 "$project_root/scripts/generate-mobile-client.py"
python3 "$project_root/scripts/check-startup-snapshot.py"
python3 "$project_root/scripts/test_upgrade_contract.py"
node --test "$project_root/scripts/mobile-package.test.mjs"
node --test "$project_root/scripts/login-ui.test.mjs"
node --test "$project_root/scripts/sms-captcha.test.mjs"
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
for cancellation_pattern in \
  'private finished: boolean = false' \
  'if (this.cancelled || this.finished) return' \
  'this.task = null' \
  'cancellation.finish()' \
  'if (cancellation.isCancelled()) return'; do
  if ! rg -Fq "$cancellation_pattern" "$project_root/src/core/network/ak-http-client.uts"; then
    printf 'mobile HTTP cancellation must not abort a completed native RequestTask: %s\n' "$cancellation_pattern" >&2
    exit 1
  fi
done
if rg -Uq "if \(cancellation\.isCancelled\(\)\) \{[[:space:]]+onFailure" "$project_root/src/core/network/ak-http-client.uts"; then
  printf 'cancelled mobile requests must not callback into an unloaded page instance\n' >&2
  exit 1
fi

public_config_runtime="$project_root/src/features/app-config/public-config-runtime.uts"
if rg -n 'response\.data\.(share\.providers|push\.(enabled|providers|build_variants))' "$public_config_runtime"; then
  printf 'optional public share/push capabilities must not be dereferenced before compatibility normalization\n' >&2
  exit 1
fi
for compatibility_pattern in \
  'const shareRuntime: PublicShareRuntimeWire | null' \
  'if (shareRuntime != null)' \
  'let pushEnabled: boolean = false' \
  'const pushRuntime: PublicPushRuntimeWire | null' \
  'if (pushRuntime != null)'; do
  if ! rg -Fq "$compatibility_pattern" "$public_config_runtime"; then
    printf 'public config rolling-upgrade compatibility is missing: %s\n' "$compatibility_pattern" >&2
    exit 1
  fi
done

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
for installation_key_pattern in \
  'activeClient.setDeviceKey(value)' \
  'installationStorage.readDeviceKey' \
  'migrateSessionDeviceKeyOrCreate' \
  'writeDeviceKey(credential.deviceKey' \
  'credential.deviceKey != this.deviceKey' \
  'oauthCoordinator.configure'; do
  if ! rg -Fq "$installation_key_pattern" "$auth_runtime"; then
    printf 'mobile authentication is missing the stable secure installation UUID contract: %s\n' "$installation_key_pattern" >&2
    exit 1
  fi
done
if rg -Fq 'clearNamespace' "$project_root/src/core/stores/ak-secure-session-storage.uts"; then
  printf 'mobile sign-out must remove only session fields and preserve the installation UUID\n' >&2
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
  browse-default.png browse-selected.png \
  topics-default.png topics-selected.png \
  profile-default.png profile-selected.png; do
  if [[ ! -f "$project_root/static/ak-tabbar/$tab_icon" ]] || ! rg -q "static/ak-tabbar/$tab_icon" "$project_root/pages.json"; then
    printf 'mobile TabBar icon is missing or not declared: %s\n' "$tab_icon" >&2
    exit 1
  fi
done
python3 "$project_root/scripts/verify-tabbar-icons.py"

back_button_icon_size="$project_root/components/ak-ui/ak-back-button/ak-back-button.uvue"
if ! rg -Uq 'ak-back-button__icon \{[[:space:]]+width: 20px;[[:space:]]+height: 20px;' "$back_button_icon_size"; then
  printf 'AK back button must use the 20px top-navigation glyph size\n' >&2
  exit 1
fi

home_page="$project_root/pages/home/index.uvue"
for home_navigation_pattern in 'class="nav-title-row"' 'class="nav-actions"' 'name="search" :icon-size="20"' 'name="bell" :icon-size="20"' 'nav-action-last { margin-left: 8px; }'; do
  if ! rg -Fq "$home_navigation_pattern" "$home_page"; then
    printf 'home page is missing the aligned 20px navigation layout: %s\n' "$home_navigation_pattern" >&2
    exit 1
  fi
done

for top_navigation_spec in \
  'pages/browse/index.uvue|name="search" :icon-size="20"' \
  'pages/browse/index.uvue|name="filter" :icon-size="20"' \
  'pages/articles/detail/index.uvue|name="share" :icon-size="20"' \
  'pages/profile/basic/index.uvue|name="edit" :icon-size="20"' \
  'pages/profile/index.uvue|name="bell"' \
  'pages/profile/index.uvue|:icon-size="20"' \
  'components/ak-ui/ak-video-viewer/ak-video-viewer.uvue|name="layout-vertical" :icon-size="20"' \
  'components/ak-ui/ak-video-viewer/ak-video-viewer.uvue|name="layout-horizontal" :icon-size="20"'; do
  navigation_file="${top_navigation_spec%%|*}"
  navigation_pattern="${top_navigation_spec#*|}"
  if ! rg -Fq "$navigation_pattern" "$project_root/$navigation_file"; then
    printf 'mobile top navigation is missing the 20px icon contract: %s (%s)\n' "$navigation_file" "$navigation_pattern" >&2
    exit 1
  fi
done

if rg -n 'class="action-icon"[^>]*:icon-size="20"' "$project_root/pages/articles/detail/index.uvue" || rg -n 'class="ak-sheet-close"[^>]*:icon-size="20"' "$project_root/components/ak-ui/ak-bottom-sheet/ak-bottom-sheet.uvue"; then
  printf 'compact content and modal actions must not inherit the 20px top-navigation size\n' >&2
  exit 1
fi

empty_component="$project_root/components/ak-ui/ak-empty/ak-empty.uvue"
status_view="$project_root/components/ak-ui/ak-status-view/ak-status-view.uvue"
for empty_pattern in \
  ':name="icon" :size="48"' \
  'color: var(--ak-muted)' \
  'min-width: 44px' \
  'min-height: 44px' \
  'min-height: 32px' \
  ':name="actionIcon" :size="14"' \
  'margin-left: 8px'; do
  if ! rg -Fq "$empty_pattern" "$empty_component"; then
    printf 'AK empty component is missing the shared visual or touch contract: %s\n' "$empty_pattern" >&2
    exit 1
  fi
done
for empty_icon in empty.svg offline.svg error.svg retry.svg; do
  if [[ ! -f "$project_root/static/ak-icons/$empty_icon" ]]; then
    printf 'AK empty component is missing its semantic icon: %s\n' "$empty_icon" >&2
    exit 1
  fi
done
if ! rg -Fq '<ak-empty' "$status_view" || rg -Fq '<ak-empty-state' "$status_view" || rg -n '<ak-empty-state' "$project_root/pages"; then
  printf 'mobile empty content states must be routed through the shared ak-empty component\n' >&2
  exit 1
fi

profile_page="$project_root/pages/profile/index.uvue"
if ! rg -Fq '<ak-icon name="language"' "$profile_page" || ! rg -Fq '<ak-icon name="settings"' "$profile_page" || [[ ! -f "$project_root/static/ak-icons/language.svg" ]]; then
  printf 'profile language/appearance and application permissions must use distinct semantic icons\n' >&2
  exit 1
fi
if rg -Uq '<ak-avatar[^>]*?/>[[:space:]]+>' "$profile_page"; then
  printf 'profile avatar must not be followed by a standalone angle-bracket text node\n' >&2
  exit 1
fi

profile_edit_page="$project_root/pages/profile/edit/index.uvue"
for avatar_crop_pattern in \
  'this.openCrop(info.path,info.width,info.height)' \
  'class="crop-source"' \
  'class="crop-gesture-layer"' \
  'event.touches.length>=2' \
  'name="zoom-out"' \
  'name="zoom-in"' \
  'name="reset"' \
  'width:320px;height:320px'; do
  if ! rg -Fq "$avatar_crop_pattern" "$profile_edit_page"; then
    printf 'profile avatar crop is missing normalized iOS preview or pinch controls: %s\n' "$avatar_crop_pattern" >&2
    exit 1
  fi
done
if rg -Fq 'class="crop-controls"' "$profile_edit_page"; then
  printf 'profile avatar crop must keep zoom/reset controls as floating icons, not text buttons\n' >&2
  exit 1
fi
for avatar_crop_icon in zoom-out.svg zoom-in.svg reset.svg; do
  if [[ ! -f "$project_root/static/ak-icons/$avatar_crop_icon" ]]; then
    printf 'profile avatar crop icon is missing: %s\n' "$avatar_crop_icon" >&2
    exit 1
  fi
done
avatar_loader="$project_root/src/features/profile/infrastructure/profile-avatar-loader.uts"
for avatar_loader_pattern in 'private finished: boolean = false' 'if (this.cancelled || this.finished) return' 'cancellation.finish();onSuccess'; do
  if ! rg -Fq "$avatar_loader_pattern" "$avatar_loader"; then
    printf 'profile avatar loader cancellation must be idempotent after native task completion: %s\n' "$avatar_loader_pattern" >&2
    exit 1
  fi
done

bookmarks_page="$project_root/pages/bookmarks/index.uvue"
for bookmark_tab_pattern in 'filter-label-active' 'color: var(--ak-on-brand)' 'font-weight: 600'; do
  if ! rg -Fq "$bookmark_tab_pattern" "$bookmarks_page"; then
    printf 'bookmark tabs are missing the high-contrast selected label contract: %s\n' "$bookmark_tab_pattern" >&2
    exit 1
  fi
done

notifications_page="$project_root/pages/notifications/index.uvue"
for notification_nav_pattern in '<ak-back-button/>' 'class="nav-title"' '<view class="page">' 'class="notification-scroll"' '.page{height:100%;flex:1;overflow:hidden' '.nav{height:44px' 'flex-shrink:0' '.notification-scroll{height:0;flex:1;min-height:0}' '.nav-side{width:88px;height:44px' 'notifications.markAllRead'; do
  if ! rg -Fq "$notification_nav_pattern" "$notifications_page"; then
    printf 'notifications page is missing the fixed navigation and independent scroll contract: %s\n' "$notification_nav_pattern" >&2
    exit 1
  fi
done
if ! grep -Fq '"path": "pages/notifications/index", "style": { "navigationStyle": "custom", "disableScroll": true }' "$project_root/pages.json"; then
  echo "notifications page must disable native page scrolling" >&2
  exit 1
fi
python3 - "$notifications_page" <<'PY'
import re
import sys

source = open(sys.argv[1], encoding="utf-8").read()
template = source.split("</template>", 1)[0]
fixed_header = re.search(
    r'<view class="page">\s*<view class="nav">.*?</view>\s*<scroll-view class="notification-scroll"',
    template,
    re.S,
)
if fixed_header is None:
    raise SystemExit("notifications navigation must be a non-scrolling sibling before the message scroll-view")
PY

video_viewer="$project_root/components/ak-ui/ak-video-viewer/ak-video-viewer.uvue"
if rg -n 'uni\.getVideoInfo|getVideoInfo\(' "$video_viewer"; then
  printf 'content video viewer must not require the optional uni-media module at runtime\n' >&2
  exit 1
fi
for video_pattern in 'video-stage-vertical' 'justify-content: center' 'onCoverLoad' "mode === 'vertical' ? 'contain' : 'cover'"; do
  if ! rg -Fq "$video_pattern" "$video_viewer"; then
    printf 'content video viewer is missing the safe centered orientation fallback: %s\n' "$video_pattern" >&2
    exit 1
  fi
done

imagetext_viewer="$project_root/components/ak-ui/ak-imagetext-viewer/ak-imagetext-viewer.uvue"
if ! rg -Fq 'uni.previewImage({ current: current, urls: urls })' "$imagetext_viewer"; then
  printf 'image-text viewer must open the ordered gallery in the native full-screen preview\n' >&2
  exit 1
fi

bottom_sheet="$project_root/components/ak-ui/ak-bottom-sheet/ak-bottom-sheet.uvue"
if rg -n '^[[:space:]]*><|^[[:space:]]*>[[:space:]]*$' "$bottom_sheet"; then
  printf 'bottom sheet header must not contain standalone angle-bracket text nodes\n' >&2
  exit 1
fi
for sheet_pattern in '<view class="ak-sheet-header">' 'class="ak-sheet-title">{{ title }}</text>' 'name="close"' 'flex: 1' 'min-width: 44px' 'min-height: 44px'; do
  if ! rg -Fq "$sheet_pattern" "$bottom_sheet"; then
    printf 'bottom sheet is missing the required single close-action layout: %s\n' "$sheet_pattern" >&2
    exit 1
  fi
done
if [[ "$(rg -c 'name="close"' "$bottom_sheet")" -ne 1 ]]; then
  printf 'bottom sheet header must expose exactly one close action\n' >&2
  exit 1
fi

account_deletion_page="$project_root/pages/profile/account-deletion/index.uvue"
account_deletion_repository="$project_root/src/features/account-deletion/infrastructure/http-account-deletion-repository.uts"
for deletion_pattern in \
  "accountDeletion.warningTitle" \
  "accountDeletion.scopeOtherApps" \
  "input-type=\"number\"" \
  ":max-length=\"6\"" \
  "accountDeletion.acknowledgement" \
  "uni.showModal" \
  "pushCoordinator.unregisterLocal" \
  "authRuntime.clearDeletedAccount"; do
  if ! rg -Fq "$deletion_pattern" "$account_deletion_page"; then
    printf 'account deletion page is missing a required safety control: %s\n' "$deletion_pattern" >&2
    exit 1
  fi
done
for deletion_api_pattern in \
  "/me/account-deletion/verification-code" \
  "/me/account-deletion/confirm" \
  "retryOnUnauthorized: false"; do
  if ! rg -Fq "$deletion_api_pattern" "$account_deletion_repository"; then
    printf 'account deletion API contract is missing or replay-unsafe: %s\n' "$deletion_api_pattern" >&2
    exit 1
  fi
done
if rg -n 'email|user_id|userId' "$account_deletion_repository"; then
  printf 'account deletion requests must not accept an email or target user identifier\n' >&2
  exit 1
fi
if ! rg -Fq 'v-if="!guest" class="profile-actions"' "$project_root/pages/profile/index.uvue" || ! rg -Fq 'v-if="accountDeletionEnabled" class="delete-account-link"' "$project_root/pages/profile/index.uvue"; then
  printf 'profile account deletion entry must require authentication and the account_deletion feature flag\n' >&2
  exit 1
fi
auth_repository="$project_root/src/features/auth/auth-repository.uts"
if ! rg -Fq 'feature_flags.toMap().forEach((value, key)' "$auth_repository" || rg -Fq 'entry.value' "$auth_repository"; then
  printf 'mobile auth feature flags must use the cross-platform Map.forEach(value, key) contract\n' >&2
  exit 1
fi
if ! rg -Fq "authRuntime.refreshContext(()=>{this.accountDeletionEnabled=!this.guest&&authContextStore.hasEnabledFlag('account_deletion')}" "$project_root/pages/profile/index.uvue"; then
  printf 'profile must refresh the server feature snapshot before deciding account deletion visibility\n' >&2
  exit 1
fi
for profile_deletion_visibility_pattern in 'class="profile-actions"' 'flex: 1;' 'height: auto;' 'padding: 8px 20px 12px' 'flex-shrink: 0'; do
  if ! rg -Fq "$profile_deletion_visibility_pattern" "$project_root/pages/profile/index.uvue"; then
    printf 'profile account actions must remain above the native TabBar: %s\n' "$profile_deletion_visibility_pattern" >&2
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
