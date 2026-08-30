---
title: Configure push channels
description: Apply for, configure, preflight, and safely enable APNs, FCM, China Android providers, and HarmonyOS Push Kit in Admin.
---

# Configure push channels

Admin's “Notification center → Push channels” manages offline push per app and environment. It never asks an administrator to paste a device token and never reveals server credentials. A test push can target only a safe summary of a device already registered to the selected app.

## Supported channels

| Target build     | Single client channel available at runtime                               | Server channel                    |
| ---------------- | ------------------------------------------------------------------------ | --------------------------------- |
| iOS              | Apple system notification capability                                     | APNs HTTP/2 + ES256 token         |
| `android_google` | FCM                                                                      | FCM HTTP v1 + OAuth 2.0           |
| `android_china`  | One available channel among Huawei, Honor, Xiaomi, OPPO, vivo, and Meizu | Each provider's official REST API |
| HarmonyOS NEXT   | HarmonyOS Push Kit                                                       | Push Kit server API               |

The Google and China Android variants are mutually exclusive. A Google build must not contain China-provider SDKs, and a China build must not depend on Firebase/GMS. Devices with no provider compiled into their build keep in-app notifications only.

## Configuration workflow

1. Select an app and the `development`, `staging`, or `production` environment.
2. Use the question-mark icon next to the page title. Its APNs, FCM, Huawei Android, Honor, Xiaomi, OPPO, vivo, Meizu, and Harmony tabs explain account application steps, field sources, risks, and official resources.
3. Create a draft and fill the strongly typed public fields. Store secret fields through the separate write-only action.
4. Run preflight and fix package name, bundle ID, environment, signing identity, or credential parsing failures.
5. Activate only after preflight reports ready. Use secret rotation instead of copying an old secret into a normal field.
6. Select a registered device whose provider matches the channel, send a test notification, and inspect its delivery result.

## Fields to prepare

| Channel        | Public or identity fields                                                 | Write-only secret    |
| -------------- | ------------------------------------------------------------------------- | -------------------- |
| APNs           | Team ID, Key ID, Bundle ID, Sandbox/Production environment                | `.p8` private key    |
| FCM            | Firebase Project ID, Android Package Name                                 | Service Account JSON |
| Huawei Android | Client ID, Package Name, notification Channel ID                          | Client Secret        |
| Honor          | App ID, Client ID, Package Name, notification Channel ID                  | Client Secret        |
| Xiaomi         | App ID, App Key, Package Name, Region, notification Channel ID            | App Secret           |
| OPPO           | App Key, Package Name, notification Channel ID                            | Master Secret        |
| vivo           | App ID, App Key, Package Name, service category, operations category      | App Secret           |
| Meizu          | App ID, App Key, Package Name                                             | App Secret           |
| Harmony        | Project ID, Client ID, Bundle Name, service category, operations category | Service Account JSON |

Links in the question-mark guide are restricted to official Apple, Firebase, Huawei, Honor, Xiaomi, OPPO, vivo, and Meizu domains. Provider consoles can move entry points; use the current official links in the page and the values shown in your provider console when applying.

## State and security semantics

- `draft`: editable but unavailable for sending.
- Preflight ready: required fields, key format, and known identity relationships are parseable; it does not prove the provider approved the account.
- `active`: eligible only when the global push switch, user subscription, device, and message expiry gates also pass.
- `faulted`: repeated authentication or configuration failures fault the channel; fix and preflight it before recovery.
- `disabled`: stop creating new provider deliveries without affecting in-app messages.

Credentials are isolated by tenant, app, environment, and provider, with key versions and fingerprints. Private keys, service accounts, client/master secrets, and device tokens must never enter logs, screenshots, fixtures, metrics, or Admin responses.

## Android build gates

Choose a build variant with `AK_ANDROID_PUSH_VARIANT=google|china`. A production build must also inject public client configuration that matches the app identity and pass final APK/AAB dependency and marker checks. Generated configuration is an ignored temporary artifact; server secrets never belong in an app package.

Every SDK must wait until the user completes privacy consent and actively enables push. Disable Firebase auto-initialization. Keep any provider disabled until delayed initialization and first-network behavior are proven compliant.

## Before production

- Provider account, push entitlement, message category, and rate-limit approval are complete.
- Package Name, Bundle ID, Bundle Name, signing fingerprint, and environment match the final artifact.
- SDK version, checksum, license, and privacy inventory are pinned and reviewed.
- Physical-device coverage includes foreground, background, terminated, offline recovery, token refresh, reinstall, account switch, and bilingual notification-open flows.

After configuration, use [Notification operations](./notification-operations) to inspect jobs and failures, and read [Notification and push architecture](../concepts/notification-architecture) for the asynchronous boundary.
