#!/usr/bin/env bash
set -euo pipefail

target="${1:-all}"
project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cli=/Applications/HBuilderX.app/Contents/MacOS/cli
ohpm=/Applications/DevEco-Studio.app/Contents/tools/ohpm/bin/ohpm
hvigor=/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw
android_package="${AK_ANDROID_PACKAGE:-com.appkernia.mobile}"
ios_bundle="${AK_IOS_BUNDLE_ID:-com.appkernia.mobile}"
log_dir="$project_root/unpackage/custom-base-logs"

usage() {
  printf 'usage: %s assets|android|ios-simulator|harmony|harmony-signed|all|verify|verify-installable\n' "$0" >&2
}

require_cli() {
  if [[ ! -x "$cli" ]]; then
    printf 'HBuilderX CLI not found: %s\n' "$cli" >&2
    exit 1
  fi
}

generate_assets() {
  python3 "$project_root/scripts/generate-native-icons.py"
  python3 "$project_root/scripts/verify-custom-base.py"
}

open_project() {
  "$cli" project open --path "$project_root" >/dev/null
}

close_project() {
  "$cli" project close --path "$project_root" >/dev/null 2>&1 || true
}

run_logged() {
  local log_file="$1"
  shift
  set +e
  "$@" 2>&1 | tee "$log_file"
  local command_exit=${PIPESTATUS[0]}
  set -e
  if [[ $command_exit -ne 0 ]] || rg -q '\[Error\]|编译失败|打包失败|错误：|程序已退出|安装鸿蒙工程依赖失败|ohpm ERROR' "$log_file"; then
    return 1
  fi
}

pack_android() {
  mkdir -p "$log_dir"
  run_logged "$log_dir/android.log" "$cli" pack \
    --project "$project_root" \
    --platform android \
    --iscustom true \
    --android.packagename "$android_package" \
    --android.androidpacktype 1 \
    --sourceMap false \
    --ignoreWarnings true
}

pack_ios_simulator() {
  mkdir -p "$log_dir"
  run_logged "$log_dir/ios-simulator.log" "$cli" pack \
    --project "$project_root" \
    --platform ios \
    --iscustom true \
    --ios.bundle "$ios_bundle" \
    --ios.supporteddevice iPhone \
    --ios.channels simulator \
    --sourceMap false \
    --ignoreWarnings true
}

generate_harmony_project() {
  local native_root="$project_root/unpackage/dist/dev/app-harmony"
  set +e
  env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY \
    -u http_proxy -u https_proxy -u all_proxy \
    "$cli" pack app-harmony \
    --project "$project_root" 2>&1 | tee "$log_dir/harmony-hbuilder.log"
  local hbuilder_exit=${PIPESTATUS[0]}
  set -e
  if [[ $hbuilder_exit -ne 0 ]] || ! rg -q '项目 .* 编译成功' "$log_dir/harmony-hbuilder.log"; then
    printf '%s\n' 'HBuilderX did not generate a compiled Harmony native project.' >&2
    return 1
  fi
}

build_harmony_native() {
  local signing_mode="$1"
  local native_root="$project_root/unpackage/dist/dev/app-harmony"
  if [[ ! -f "$native_root/build-profile.json5" ]]; then
    printf '%s\n' 'Generated Harmony native project is missing.' >&2
    return 1
  fi
  local prepare_args=()
  case "$signing_mode" in
    unsigned) prepare_args+=(--unsigned) ;;
    signed) prepare_args+=(--require-signing) ;;
    *) printf 'unknown Harmony signing mode: %s\n' "$signing_mode" >&2; return 2 ;;
  esac
  python3 "$project_root/scripts/prepare-harmony-native.py" "${prepare_args[@]}"
  (
    cd "$native_root"
    env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY \
      -u http_proxy -u https_proxy -u all_proxy \
      "$ohpm" install --all
  ) 2>&1 | tee "$log_dir/harmony-ohpm.log"
  (
    cd "$native_root"
    env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY \
      -u http_proxy -u https_proxy -u all_proxy \
      DEVECO_SDK_HOME=/Applications/DevEco-Studio.app/Contents/sdk \
      "$hvigor" --mode module \
      -p product=default \
      -p module=entry@default \
      -p buildMode=debug \
      assembleHap --no-daemon
  ) 2>&1 | tee "$log_dir/harmony-hvigor.log"
}

pack_harmony() {
  mkdir -p "$log_dir"
  generate_harmony_project
  build_harmony_native unsigned
}

pack_harmony_signed() {
  mkdir -p "$log_dir"
  build_harmony_native signed
}

require_cli
case "$target" in
  assets)
    generate_assets
    ;;
  verify)
    python3 "$project_root/scripts/verify-custom-base.py" --artifacts
    ;;
  verify-installable)
    python3 "$project_root/scripts/verify-custom-base.py" \
      --artifacts --require-harmony-signed
    ;;
  harmony-signed)
    generate_assets
    pack_harmony_signed
    ;;
  android|ios-simulator|harmony|all)
    generate_assets
    open_project
    trap close_project EXIT
    case "$target" in
      android) pack_android ;;
      ios-simulator) pack_ios_simulator ;;
      harmony) pack_harmony ;;
      all)
        pack_android
        pack_ios_simulator
        pack_harmony
        ;;
    esac
    ;;
  *)
    usage
    exit 2
    ;;
esac
