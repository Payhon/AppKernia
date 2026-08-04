#!/usr/bin/env bash
set -euo pipefail

plist_version() {
  /usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$1"
}

printf 'HBuilderX=%s\n' "$(plist_version /Applications/HBuilderX.app/Contents/Info.plist)"
printf 'HBuilderX CLI=%s\n' "$(/Applications/HBuilderX.app/Contents/MacOS/cli help 2>&1 | sed -n '1p')"
printf 'Xcode=%s\n' "$(xcodebuild -version | tr '\n' ' ')"
printf 'DevEco Studio=%s\n' "$(plist_version /Applications/DevEco-Studio.app/Contents/Info.plist)"
/Applications/HBuilderX.app/Contents/MacOS/cli devices list --platform android || true
/Applications/HBuilderX.app/Contents/MacOS/cli devices list --platform ios-simulator || true
/Applications/HBuilderX.app/Contents/MacOS/cli devices list --platform app-harmony || true

