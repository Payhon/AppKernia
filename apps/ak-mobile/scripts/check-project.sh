#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "$project_root/../.." && pwd)"

python3 "$repo_root/blueprint/mobile/scripts/validate_blueprint_specs.py"
python3 "$repo_root/blueprint/scripts/validate_i18n_contract.py"
jq -e '."uni-app-x".renderer == "vdom"' "$project_root/manifest.json" >/dev/null
rg -n --glob '*.uts' --glob '*.uvue' '\bany\b|sslVerify\s*:\s*false|uni\.(setStorage|setStorageSync)\([^\n]*(token|password|otp)' "$project_root" && {
  printf 'mobile source contains a forbidden pattern\n' >&2
  exit 1
} || true

for target in android ios harmony; do
  test -x "$project_root/scripts/build-platform.sh"
  rg -q "$target" "$project_root/scripts/build-platform.sh"
done

printf 'AK Mobile static project checks passed. Platform builds are separate evidence gates.\n'

