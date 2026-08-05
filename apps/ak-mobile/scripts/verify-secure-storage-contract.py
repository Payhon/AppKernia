#!/usr/bin/env python3
"""Static contract checks for the AK platform credential vault.

This deliberately verifies source-level invariants only. It does not claim a
Keychain, Keystore, or Asset Store device round trip.
"""

from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PLUGIN = ROOT / "uni_modules" / "ak-secure-storage"


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(f"FAIL: {message}")


def text(path: Path) -> str:
    require(path.exists(), f"missing {path.relative_to(ROOT)}")
    return path.read_text(encoding="utf-8")


def main() -> None:
    interface = text(PLUGIN / "utssdk/interface.uts")
    android = text(PLUGIN / "utssdk/app-android/index.uts")
    ios = text(PLUGIN / "utssdk/app-ios/index.uts")
    swift = text(PLUGIN / "utssdk/app-ios/hybrid.swift")
    harmony = text(PLUGIN / "utssdk/app-harmony/index.uts")
    port = text(ROOT / "src/core/stores/secure-session-storage-port.uts")
    runtime = text(ROOT / "src/core/stores/session-runtime.uts")
    adapter = text(ROOT / "src/core/stores/ak-secure-session-storage.uts")
    bootstrap = text(ROOT / "src/core/bootstrap/app-bootstrap.uts")

    for api in (
        "SecureStorageAvailabilityApi",
        "SecureStorageSet",
        "SecureStorageGet",
        "SecureStorageRemove",
        "SecureStorageClearNamespace",
    ):
        require(api in interface, f"vault API {api} is absent")
    require("accessToken" not in port.split("export type PersistedSessionCredential", 1)[1].split("}\n", 1)[0], "persisted DTO contains access token")
    require("this.session.setAccessToken" not in runtime.split("restore(", 1)[1].split("readCredential", 1)[0], "restore must not revive an old access token")
    require("authRuntime.restore(credential" in bootstrap, "bootstrap does not rotate from persisted refresh credential")
    require("uni.setStorage" not in adapter and "uni.getStorage" not in adapter, "adapter falls back to ordinary uni storage")
    require("KeyStore.getInstance('AndroidKeyStore')" in android, "Android does not use Android Keystore")
    require("AES/GCM/NoPadding" in android and "GCMParameterSpec" in android, "Android payload lacks AES-GCM")
    require("kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly" in swift, "iOS Keychain entry is not device-only")
    require("SecItemAdd" in swift and "SecItemCopyMatching" in swift and "SecItemDelete" in swift, "iOS Keychain CRUD is incomplete")
    require("@kit.AssetStoreKit" in harmony and "asset.Accessibility.DEVICE_FIRST_UNLOCKED" in harmony, "Harmony does not use Asset Store secure storage")
    require("asset.addSync" in harmony and "asset.querySync" in harmony and "asset.removeSync" in harmony, "Harmony Asset Store CRUD is incomplete")
    for native in (android, ios, swift, harmony):
        require("console.log" not in native and "console.error" not in native, "native vault must not log credential operations")
    print("secure storage static contract checks passed")


if __name__ == "__main__":
    main()
