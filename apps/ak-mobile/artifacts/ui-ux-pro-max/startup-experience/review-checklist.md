# Review Checklist

- [x] Consent renders entirely from packaged icon/name/subtitle and local bilingual Catalog text.
- [x] No public config, image request, session restore or sensitive capability starts before consent.
- [x] Legal pages stay in the current container and return to consent.
- [x] Cancel behavior is isolated behind the Android/iOS/Harmony platform port.
- [x] Onboarding is image-only, has no skip, preloads all images and requires every position.
- [x] Completion is versioned by App UUID and monotonically stores the highest completed revision.
- [x] Network/preload errors fail open without writing completion; App ID mismatch remains closed.
- [x] All visible copy and image descriptions are bilingual semantic keys/data.
- [ ] Capture both locales on Android, iOS and Harmony physical devices with dynamic type and safe-area variants.
- [ ] Verify Android/Harmony exit, iOS blocked cancel and system-back behavior on physical devices.
