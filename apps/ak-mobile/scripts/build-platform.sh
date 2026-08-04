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
trap 'rm -f "$log_file"' EXIT
set +e
"${command[@]}" 2>&1 | tee "$log_file"
ak_build_exit=${PIPESTATUS[0]}
set -e
if rg -q '暂不支持|项目 .* 不存在|编译失败|compile failed|Error:|ERROR|错误|已停止运行' "$log_file"; then
  exit 1
fi
exit "$ak_build_exit"
