---
title: Mobile development
description: Run and verify the uni-app x mobile project with HBuilderX.
---

# Mobile development

AK Mobile lives in `apps/ak-mobile` and uses uni-app x, UTS/UVue, VDOM, and AK UI. A Node or Vite build is never a substitute for a mobile platform build.

## Toolchains

- Stable HBuilderX (currently the 5.15 stable line or a verified patch)
- Android 8 / API 26 or newer tooling
- Xcode, certificates, and provisioning for iOS 13+
- HBuilderX plus DevEco Studio for HarmonyOS NEXT API 14+

```bash
apps/ak-mobile/scripts/detect-toolchain.sh
apps/ak-mobile/scripts/check-project.sh
```

Open `apps/ak-mobile` in HBuilderX, configure signing locally, point the API base URL to a reachable Go API, and choose a target platform.

```bash
apps/ak-mobile/scripts/build-platform.sh android
apps/ak-mobile/scripts/build-platform.sh ios
apps/ak-mobile/scripts/build-platform.sh harmony
```

Completion depends on installed IDEs, SDKs, signing, and devices. A static check only proves static gates; do not report a platform as accepted without its build and device record.

See [Mobile custom-base and release packaging](./mobile-packaging.md) for custom playgrounds, signed release artifacts, macOS/Windows variables, and root-level automation commands.

Business pages must use `ak-*` components from `components/ak-ui`, never bind directly to `up-*`. Continue with the [mobile component overview](../mobile-components/).

## Push and system permissions

Mobile business code uses `uni_modules/ak-push` as its single push adapter and delegates authorization queries, requests, and system-settings navigation to `uni_modules/ak-permissions`. A page must not prompt on load. SDK initialization, token acquisition, and server registration start only after privacy consent and an explicit user action.

See the [Mobile permission center](./mobile-permissions), [Push channel configuration](./push-channels), and [Notification API](../api/mobile-notifications). The mutually exclusive Google/China Android variants and production build gates are covered by [Mobile packaging](./mobile-packaging.md).
