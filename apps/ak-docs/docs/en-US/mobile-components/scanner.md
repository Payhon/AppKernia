---
title: Scanner capability
description: Scan QR codes and barcodes through ak-scanner, then consume results through typed events, handlers, and a guarded WebView.
---

# Scanner capability

`uni_modules/ak-scanner` is AppKernia's only port to the system scanner API. It is camera-only, supports QR codes and one-dimensional barcodes in the first release, and never exposes photo-library input. Business pages must not call `uni.scanCode` directly.

Scan content stays in the current app process. It is never uploaded, persisted, logged, or written to audit records.

## Layers and public types

The native port opens the system scanner and normalizes platform results. The coordinator in `src/features/scanner` owns permission flow, events, handler priority, trusted pages, and the result fallback.

```ts
type AkScanFormat = 'qr_code' | 'bar_code' | 'unknown';
type AkScanSource = 'camera';

class AkScanResult {
  scanId: string;
  rawValue: string;
  format: AkScanFormat;
  source: AkScanSource;
  scannedAt: number;
}

type AkScanResolution = 'consumed' | 'open_webview' | 'present_result';
type AkScanEventName = 'captured' | 'parsed' | 'resolved' | 'cancelled' | 'failed';
```

`captured` means the platform returned a value, `parsed` means it has been normalized, and `resolved` carries the final resolution. Closing the system scanner emits `cancelled` without an error UI. Permission and platform failures emit `failed`.

## Calling and subscribing

Subscribe at page load and dispose the subscription at unload. The coordinator is single-flight and will not open a second scanner before the current attempt finishes.

```ts
import {
  scannerCoordinator,
  AkScanSubscription,
} from '@/src/features/scanner/application/scanner-coordinator.uts';
import { AkScanEvent } from '@/src/features/scanner/domain/models.uts';

let scanSubscription: AkScanSubscription | null = null;

export default {
  onLoad() {
    scanSubscription = scannerCoordinator.subscribe((event: AkScanEvent) => {
      if (event.name === 'resolved' && event.resolution === 'present_result') {
        // Present event.result in ak-bottom-sheet.
      }
    });
  },
  onUnload() {
    const current = scanSubscription;
    scanSubscription = null;
    if (current != null) current.dispose();
  },
  methods: {
    scan() {
      scannerCoordinator.scan();
    },
  },
};
```

## Registering a business handler

Handlers run from highest to lowest `priority`; the first matching `canHandle` owns the result. Future PC sign-in or device-binding modules can consume their protocols without changing the home page or native scanner port.

```ts
import { AkScanResult } from '@/uni_modules/ak-scanner';
import { AkScanHandler } from '@/src/features/scanner/domain/models.uts';

const loginHandler: AkScanHandler = {
  id: 'pc-login',
  priority: 100,
  canHandle: (result: AkScanResult) => result.rawValue.startsWith('ak://login/'),
  handle: (_result: AkScanResult) => 'consumed',
};

const registration = scannerCoordinator.registerHandler(loginHandler);
// Call registration.dispose() when the module unloads.
```

This is an extension-contract example only; PC sign-in and device binding are not implemented in this release. Handlers are compile-time client code. Server configuration cannot deliver executable handlers.

## Permission and error semantics

- Camera access is queried and requested only after the user taps Scan.
- `onlyFromCamera: true`; no photo-library permission is requested.
- After denial, the result sheet offers retry, system settings, and cancel.
- `cancelled` is a normal user path. Stable failure codes are `permission_denied`, `scanner_unavailable`, `native_failure`, and `busy`.
- The permission-center page only reads status and never prompts on open.

See the [Mobile permission center](../guide/mobile-permissions).

## Trusted WebView boundary

The coordinator opens the built-in WebView only for an absolute HTTPS URL with no credentials, no non-443 port, and a hostname matching the runtime allowlist. Missing, failed, or malformed configuration fails closed.

The target URL is never exposed as a route parameter. The coordinator issues a single-use in-memory token that expires after 60 seconds; the static WebView page consumes and deletes it. Initial open plus every `loading` and `load` event revalidates the destination. A redirect to an unapproved host, HTTP, or an external scheme closes the page and returns the original scan to the result sheet. The WebView exposes no scanner event or app message bridge.

The standard WebView guard closes a page after a load event reveals a violation; it does not promise zero network requests. A strict pre-request guarantee requires a future native navigation delegate on all three platforms.

## Result fallback and copy

An unapproved URL, plain text, numeric barcode, or public-config failure appears in `ak-bottom-sheet` with a wrapping raw value and normalized format. Clipboard access occurs only after the user taps Copy result, with separate success and failure feedback.

## Platform matrix

| Capability                         | Android                 | iOS                     | HarmonyOS NEXT          |
| ---------------------------------- | ----------------------- | ----------------------- | ----------------------- |
| QR code / barcode                  | uni-app x build support | uni-app x build support | uni-app x build support |
| Camera permission adapter          | Supported               | Supported               | Supported               |
| Built-in WebView guard             | Supported               | Supported               | Supported               |
| Physical-device release acceptance | Required per release    | Required per release    | Required per release    |

An iOS simulator has no usable camera, so tapping Scan fails safely with `scanner_unavailable` without entering the native scanner. A successful compile is not physical-device acceptance for scanning, permission recovery, or WebView redirects. Rebuild the custom base with `uni-scanCode` after adding scanner support or upgrading HBuilderX; do not reuse a base built before the feature existed. Refer to the official [uni.scanCode](https://doc.dcloud.net.cn/uni-app-x/api/scan-code.html), [web-view](https://doc.dcloud.net.cn/uni-app-x/component/web-view.html), and [clipboard](https://doc.dcloud.net.cn/uni-app-x/api/clipboard.html) documentation.
