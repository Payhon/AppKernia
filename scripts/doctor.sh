#!/usr/bin/env bash
set -euo pipefail

required=(go node npm pnpm docker)
missing=0

for command_name in "${required[@]}"; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    printf 'required.%s=missing\n' "${command_name}"
    missing=1
  fi
done

if (( missing != 0 )); then
  exit 1
fi

printf 'go=%s\n' "$(GOTOOLCHAIN=go1.26.5 go version)"
printf 'node=%s\n' "$(node --version)"
printf 'npm=%s\n' "$(npm --version)"
printf 'pnpm=%s\n' "$(pnpm --version)"
printf 'docker=%s\n' "$(docker version --format '{{.Client.Version}}')"

version_mismatch=0
node_major="$(node --version)"
node_major="${node_major#v}"
node_major="${node_major%%.*}"
pnpm_major="$(pnpm --version)"
pnpm_major="${pnpm_major%%.*}"
npm_major="$(npm --version)"
npm_major="${npm_major%%.*}"

if [[ "${node_major}" != "24" ]]; then
  printf 'version.node=unsupported (expected 24.x)\n'
  version_mismatch=1
fi
if [[ "${pnpm_major}" != "11" ]]; then
  printf 'version.pnpm=unsupported (expected 11.x)\n'
  version_mismatch=1
fi
if (( npm_major < 11 )); then
  printf 'version.npm=unsupported (expected >=11)\n'
  version_mismatch=1
fi

if [[ "$(uname -s)" == "Darwin" && -x /usr/libexec/PlistBuddy ]]; then
  for app_spec in \
    "hbuilderx:/Applications/HBuilderX.app" \
    "xcode:/Applications/Xcode.app" \
    "deveco:/Applications/DevEco-Studio.app"; do
    app_name="${app_spec%%:*}"
    app_path="${app_spec#*:}"
    plist_path="${app_path}/Contents/Info.plist"
    if [[ -f "${plist_path}" ]]; then
      printf 'optional.%s=%s\n' "${app_name}" "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${plist_path}")"
    else
      printf 'optional.%s=not-installed\n' "${app_name}"
    fi
  done
else
  printf 'optional.mobile-ides=not-checked-on-this-platform\n'
fi

exit "${version_mismatch}"
