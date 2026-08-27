---
title: Mobile custom-base and release packaging
description: Build AppKernia Android, iOS, and HarmonyOS custom playgrounds and signed release artifacts with cross-platform scripts.
---

# Mobile custom-base and release packaging

AppKernia provides one Node.js orchestrator for the HBuilderX and DevEco toolchains on macOS and Windows. Android and iOS custom bases always use a custom playground. HarmonyOS produces a local HAP with the AppKernia native identity instead of the DCloud package name or default icon.

## Custom bases

```bash
pnpm build:mobile:base:preflight
pnpm build:mobile:base:dry-run
pnpm build:mobile:base
pnpm build:mobile:base:verify
```

Per-platform commands:

```bash
pnpm build:mobile:base:android
pnpm build:mobile:base:ios:simulator
pnpm build:mobile:base:ios:device
pnpm build:mobile:base:harmony
pnpm build:mobile:base:harmony:signed
```

The `all` command defaults to an iOS simulator base on macOS. On Windows it defaults to an iOS device base and therefore requires an Apple Development p12 and provisioning profile. Override it with `AK_CUSTOM_IOS_TARGET=simulator|device`.

## Production releases

```bash
pnpm build:mobile:release:dry-run
pnpm build:mobile:release:harmony:prepare
# Configure the release Signing Config in DevEco Studio.
pnpm build:mobile:release:preflight
pnpm build:mobile:release
pnpm build:mobile:release:verify
```

Per-platform entry points are `build:mobile:release:android`, `build:mobile:release:ios`, `build:mobile:release:harmony:prepare`, and `build:mobile:release:harmony`.

Android release packaging uses the project's own Keystore. iOS uses an Apple Distribution p12 and App Store provisioning profile. The orchestrator passes signing values to HBuilderX through a restricted temporary configuration file, so passwords are not written to command-line arguments or the repository.

HarmonyOS release packaging is intentionally two-stage:

1. `release:harmony:prepare` regenerates the native project, overlays the AppKernia AppScope, and removes stale signing references.
2. Configure a release Signing Config for `com.appkernia.mobile` in DevEco Studio, then run `release:harmony` to build the signed APP.

`release:preflight` checks the Android and iOS signing environment variables together with the Harmony Signing Config. It fails if any required input is absent.

## Tool paths

Common installation locations are detected automatically. Override them with:

```text
HBUILDERX_CLI
DEVECO_STUDIO_HOME
PYTHON_BIN
```

Windows PowerShell example:

```powershell
$env:HBUILDERX_CLI = 'C:\Tools\HBuilderX\cli.exe'
$env:DEVECO_STUDIO_HOME = 'C:\Program Files\Huawei\DevEco Studio'
pnpm build:mobile:base:dry-run
```

## Signing environment variables

Android release:

```text
AK_ANDROID_CERT_FILE
AK_ANDROID_CERT_ALIAS
AK_ANDROID_CERT_PASSWORD
AK_ANDROID_STORE_PASSWORD
AK_ANDROID_CHANNELS (optional)
```

iOS development and release:

```text
AK_IOS_DEV_PROFILE / AK_IOS_DEV_CERT_FILE / AK_IOS_DEV_CERT_PASSWORD
AK_IOS_DIST_PROFILE / AK_IOS_DIST_CERT_FILE / AK_IOS_DIST_CERT_PASSWORD
```

Never commit these values through an `.env` file. Use a CI secret store or temporary variables in the current local shell.

## Evidence boundaries

- A dry run validates orchestration but does not package an app.
- A successful build is not an installation or physical-device result.
- A signed artifact has not necessarily been uploaded or approved by a store.
- An iOS simulator result does not prove iOS physical-device behavior.
- An unsigned Harmony HAP is for the official emulator and is not a production store APP.

The complete repository manuals are:

- `docs/manual/mobile-custom-base-build.md`
- `docs/manual/mobile-production-release.md`

Continue with [Mobile development](./mobile-development.md) and [Project structure](./project-structure.md).
