# Decisions

1. Use `apps/ak-admin/public/brand/appkernia-mark.png` as the canonical source and generate all mobile native assets deterministically.
2. Use `com.appkernia.mobile` as the Android package, iOS bundle identifier and HarmonyOS bundle name for the reusable AppKernia base.
3. Android and iOS build/run commands force the HBuilderX `custom` playground. The standard `io.dcloud.uniappx` playground is not an accepted fallback.
4. HarmonyOS has no playground concept; use `harmony-configs` to overlay AppKernia identity/resources before DevEco local packaging.
5. Keep signing material outside Git. Android custom debug packaging may use DCloud's public debug certificate; production signing remains a separate release gate. iOS physical-device packaging requires a matching AppKernia profile/certificate.
6. Do not infer physical-device acceptance from source checks, cloud packaging or a simulator artifact.
