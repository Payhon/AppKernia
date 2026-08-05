# ak-secure-storage

Only sensitive, short-lived credential material belongs here. The plugin has no
web implementation and deliberately provides no ordinary-storage fallback.

- Android: an AES-256-GCM key is created in Android Keystore. The plugin-owned,
  app-private preference file contains only `nonce.ciphertext`, never plaintext.
- iOS: Keychain generic-password entries use
  `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`, so values are unavailable
  before first unlock and excluded from backup migration.
- HarmonyOS NEXT: Asset Store Kit holds each value as a critical asset at
  `DEVICE_FIRST_UNLOCKED`. Values are limited to 900 UTF-8 bytes because Asset
  Store is intended for short sensitive data; the session adapter stores each
  credential field separately.

The public UTS surface is `availability`, `set`, `get`, `remove`, and
`clearNamespace`. It returns stable error codes only and never logs values,
aliases, crypto material, or native error text.

Native verification is required on Android API 26+, iOS 13+, and Harmony API
14+. A platform compilation is not evidence that a device Keychain/Keystore
round-trip has occurred.
