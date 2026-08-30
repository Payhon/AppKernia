---
title: Mobile permission center
description: Query notification permission, request it on user action, open system settings, and keep a stable foundation for future device capabilities.
---

# Mobile permission center

“Me → App permissions” lists only capabilities that are compiled into the current build and used by the product. Notification permission is fully wired in the first release. Camera, photos, file picking, microphone, location, and Bluetooth have stable reserved definitions but stay hidden and unrequested until a feature needs them.

Opening the page or returning from system settings only queries status; it never prompts. Denying notification permission does not block sign-in, in-app messages, or other features.

## Stable permission states

| State            | Meaning                                                                   |
| ---------------- | ------------------------------------------------------------------------- |
| `not_determined` | The OS has not asked the user                                             |
| `authorized`     | Granted                                                                   |
| `limited`        | A temporary or restricted capability is available                         |
| `denied`         | Denied; platform policy decides whether to request again or open settings |
| `restricted`     | Restricted by OS policy, parental controls, or device management          |
| `unavailable`    | Unsupported by this platform, OS version, or build                        |

Each capability also reports `can_request`, `can_open_settings`, a platform explanation, and the last checked time. The UI chooses “Enable” or “Open system settings” from these facts instead of guessing platform behavior.

## Push enablement order

1. Confirm that required privacy consent has been completed.
2. Request OS notification permission only after an explicit user action.
3. Initialize the single push adapter available to this build and device.
4. Obtain the provider's renewable token.
5. Idempotently register the server-side device binding.
6. Enable the push master switch while preserving separate service/security and news/operations subscriptions.

Any failure keeps or rolls back the switch and shows a recoverable error. Disabling push first deactivates the server binding, then best-effort unregisters the local SDK token to reduce cross-account delivery risk.

## Platform behavior

- Android 13+ handles `POST_NOTIFICATIONS`. Settings open the app's notification page first and fall back to app details.
- iOS distinguishes not determined, authorized, provisional/limited, and denied, and prefers the notification-specific settings destination.
- HarmonyOS NEXT queries, requests, and opens settings through notification management, with a controlled fallback only when required.
- `ak-push` delegates authorization and settings operations to `ak-permissions`, preventing two OS-status sources.

OS permission state is not uploaded in the first release. The server stores only notification preferences and push device bindings. On return from settings, the client refreshes in `onShow` and deactivates a binding that is no longer usable.

## PermissionPort

`uni_modules/ak-permissions` exposes:

- `listCapabilities()`
- `getStatus(key)`
- `request(key)`
- `openSettings(key)`
- `onStatusChanged(listener)`

A compile-time registry owns capability definitions. File access prefers the system picker; reserving a file capability never justifies broad Android storage permission.

## Public-config compatibility

Mobile normalizes the `share` and `push` sections of public app configuration at runtime. When an older server omits either section, both capabilities fail closed and the app can still start; missing fields are never interpreted as enabled.

This compatibility prevents a startup crash during version skew, but it is not a substitute for a server upgrade. Public configuration, build variant, provider channel, and device registration must all be ready before push can be enabled.

## Acceptance boundary

HBuilderX compilation or simulator startup does not replace physical-device acceptance. Before release, verify first prompt, denial, permanent denial, settings recovery, upgrade, reinstall, account switch, token refresh, and notification-open behavior on iOS, GMS Android, China-provider Android, and HarmonyOS NEXT devices.

Continue with [Push channel configuration](./push-channels) and the [Notification API](../api/mobile-notifications).
