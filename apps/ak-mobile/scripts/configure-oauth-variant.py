#!/usr/bin/env python3
"""Select the Android OAuth boundary for the existing google/china build variant."""

from __future__ import annotations

import hashlib
import json
import os
import shutil
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PLUGIN = ROOT / "uni_modules" / "ak-oauth" / "utssdk" / "app-android"
OUTPUT_CONFIG = PLUGIN / "config.json"
OUTPUT_MANIFEST = PLUGIN / "AndroidManifest.xml"
OUTPUT_BRIDGE = PLUGIN / "hybrid.kt"
SUMMARY = ROOT / ".push-build" / "android-oauth-summary.json"
GOOGLE_DEPENDENCIES = [
    "androidx.credentials:credentials:1.6.0",
    "androidx.credentials:credentials-play-services-auth:1.6.0",
    "com.google.android.libraries.identity.googleid:googleid:1.2.0",
]


def oauth_variant(push_variant: str) -> str:
    if push_variant not in {"disabled", "google", "china"}:
        raise SystemExit("AK_ANDROID_PUSH_VARIANT must be google or china (disabled is a development-only China OAuth boundary)")
    return "android_google" if push_variant == "google" else "android_china"


def oauth_config(variant: str) -> dict[str, object]:
    value: dict[str, object] = {"minSdkVersion": "26"}
    if variant == "android_google":
        value["dependencies"] = GOOGLE_DEPENDENCIES
    return value


def oauth_manifest(variant: str) -> str:
    return f'''<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android">
    <application>
        <meta-data android:name="com.appkernia.oauth.variant" android:value="{variant}" />
    </application>
</manifest>
'''


def main() -> None:
    variant = oauth_variant(os.environ.get("AK_ANDROID_PUSH_VARIANT", "disabled").strip().lower())
    template = PLUGIN / ("hybrid-google.kt.template" if variant == "android_google" else "hybrid-china.kt.template")
    if not template.is_file():
        raise SystemExit("Android OAuth bridge template is missing")
    OUTPUT_CONFIG.write_text(json.dumps(oauth_config(variant), ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    OUTPUT_MANIFEST.write_text(oauth_manifest(variant), encoding="utf-8")
    shutil.copyfile(template, OUTPUT_BRIDGE)
    SUMMARY.parent.mkdir(parents=True, exist_ok=True)
    digest = hashlib.sha256(OUTPUT_CONFIG.read_bytes() + OUTPUT_MANIFEST.read_bytes() + OUTPUT_BRIDGE.read_bytes()).hexdigest()
    SUMMARY.write_text(json.dumps({"variant": variant, "config_sha256": digest, "google_dependencies_packaged": variant == "android_google"}, indent=2) + "\n", encoding="utf-8")
    print(f"configured Android OAuth variant={variant} sha256={digest}")


if __name__ == "__main__":
    main()
