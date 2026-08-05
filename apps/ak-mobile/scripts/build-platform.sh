#!/usr/bin/env bash
set -euo pipefail

platform="${1:-}"
project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cli=/Applications/HBuilderX.app/Contents/MacOS/cli

"$cli" project open --path "$project_root" >/dev/null

case "$platform" in
  android)
    command=("$cli" launch app-android --project "$project_root" --compile true --cleanCache true --continue-on-error false)
    ;;
  ios)
    command=("$cli" launch app-ios --project "$project_root" --iosTarget simulator --compile true --cleanCache true --continue-on-error false)
    ;;
  harmony)
    command=("$cli" launch app-harmony --project "$project_root" --compile true --cleanCache true --continue-on-error false)
    ;;
  *)
    printf 'usage: %s android|ios|harmony\n' "$0" >&2
    exit 2
    ;;
esac

log_file="$(mktemp -t appkernia-hbuilder.XXXXXX)"
cleanup() {
  "$cli" project close --path "$project_root" >/dev/null 2>&1 || true
  rm -f "$log_file"
}
trap cleanup EXIT
set +e
"${command[@]}" 2>&1 | tee "$log_file"
ak_build_exit=${PIPESTATUS[0]}
set -e
if rg -q '暂不支持|项目 .* 不存在|编译失败|compile failed|Error:|ERROR|错误|与主程序的连接已中断|运行状态错误' "$log_file"; then
  exit 1
fi
# `--compile true` intentionally ends with "已停止运行" after class generation.
# A compile-only Android pass is therefore CLI exit 0, no error marker, and this phase marker;
# it is not an install, launch, or device validation.
if [[ "$platform" == "android" ]] && ! rg -q '正在编译为android class' "$log_file"; then
  printf '%s\n' 'Android compile did not reach the page-to-class phase.' >&2
  exit 1
fi
# iOS and Harmony need an explicit UTS completion marker. A successful CLI
# transport response without it is not a compilation result. Harmony may then
# continue into native packaging; any later package/signing failure is caught
# by the error-marker check above and remains a non-zero script result.
if [[ "$platform" == "ios" || "$platform" == "harmony" ]] && ! rg -q 'UTS编译完毕' "$log_file"; then
  printf '%s UTS compilation did not complete.\n' "$platform" >&2
  exit 1
fi
exit "$ak_build_exit"
