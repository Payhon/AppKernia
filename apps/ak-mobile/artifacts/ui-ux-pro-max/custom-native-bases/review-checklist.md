# Review checklist

- [x] Canonical AppKernia brand master identified and visually inspected.
- [x] Android density icons and Android 12 safe-area splash icons generated.
- [x] iOS 1024 × 1024 icon generated without an alpha channel.
- [x] HarmonyOS 288 × 288 layered icon, 144 × 144 start icon, label and native bundle overlay added.
- [x] Android/iOS run paths force `--playground custom`.
- [x] Android custom-base APK native identity inspected (`com.appkernia.mobile`, `AppKernia`, `0.2.0`).
- [x] Android physical-device installer rendered the AppKernia icon/name/version.
- [x] Android physical-device installation, custom-playground resource sync and anonymous cold start completed.
- [x] iOS custom-base App inspected, installed and launcher/runtime captured on an iOS 18.6 simulator.
- [x] HarmonyOS unsigned HAP inspected with `com.appkernia.mobile` and `__UNI__196F2FC` resources.
- [x] HarmonyOS unsigned HAP installed and foregrounded on the official API 22 Phone emulator; first-launch AppKernia privacy surface captured at 1080 × 2340.
- [x] HarmonyOS API 22 Home Screen captured; complete repository-derived AppKernia layered launcher icon and label are visible.
- [ ] HarmonyOS signed HAP launcher/start-window captured on a physical device.
- [ ] Android/iOS/HarmonyOS secure-storage smoke completed on physical devices.
