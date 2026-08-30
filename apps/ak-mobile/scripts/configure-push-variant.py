#!/usr/bin/env python3
"""Generate the native Android push dependency boundary without credentials."""

from __future__ import annotations

import hashlib
import json
import os
import shutil
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
LOCK_PATH = ROOT / "config" / "push-sdk-lock.json"
PLUGIN_ANDROID = ROOT / "uni_modules" / "ak-push" / "utssdk" / "app-android"
OUTPUT_CONFIG = PLUGIN_ANDROID / "config.json"
OUTPUT_MANIFEST = PLUGIN_ANDROID / "AndroidManifest.xml"
SUMMARY = ROOT / ".push-build" / "android-push-summary.json"


def required(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        raise SystemExit(f"{name} is required for this push build")
    return value


def pinned_dependency(name: str) -> str:
    value = required(name)
    normalized = value.lower()
    floating_markers = ("+", "snapshot", "latest", "release")
    if any(marker in normalized for marker in floating_markers):
        raise SystemExit(f"{name} must identify an exact, immutable SDK version")
    return value


def metadata(name: str, value: str) -> str:
    escaped = value.replace("&", "&amp;").replace('"', "&quot;").replace("<", "&lt;")
    return f'        <meta-data android:name="{name}" android:value="{escaped}" />'


def main() -> None:
    lock = json.loads(LOCK_PATH.read_text(encoding="utf-8"))
    variant = os.environ.get("AK_ANDROID_PUSH_VARIANT", "disabled").strip().lower()
    require_push = os.environ.get("AK_REQUIRE_PUSH_CONFIG", "0") == "1"
    if variant not in {"disabled", "google", "china"}:
        raise SystemExit("AK_ANDROID_PUSH_VARIANT must be google or china (disabled is allowed only for non-push development builds)")
    if require_push and variant == "disabled":
        raise SystemExit("a production push build requires AK_ANDROID_PUSH_VARIANT=google|china")

    dependencies: list[object] = []
    project: dict[str, list[str]] = {"plugins": [], "dependencies": [], "repositories": []}
    meta = [
        metadata("com.appkernia.push.variant", variant),
        metadata("dcloud_push_auto_request_permission", "false"),
    ]
    public_fields: list[str] = []
    for filename in ("google-services.json", "agconnect-services.json", "mcs-services.json"):
        target = ROOT / "nativeResources" / "android" / filename
        if target.exists():
            target.unlink()

    if variant == "google":
        google = lock["google"]
        dependencies.append({"id": "firebase-bom", "source": f"implementation platform('com.google.firebase:firebase-bom:{google['firebase_bom']}')"})
        dependencies.extend(google["dependencies"])
        project["plugins"].append("com.google.gms.google-services")
        project["dependencies"].append(f"com.google.gms:google-services:{google['google_services_plugin']}")
        source = Path(required("AK_FCM_CONFIG_FILE")).expanduser().resolve()
        if not source.is_file():
            raise SystemExit("AK_FCM_CONFIG_FILE does not name a readable google-services.json")
        shutil.copyfile(source, ROOT / "nativeResources" / "android" / "google-services.json")
        public_fields.append("AK_FCM_CONFIG_FILE")
        # Firebase's content provider is still packaged, but Messaging must not
        # acquire a registration token before the user has opted in to push.
        meta.extend((
            metadata("firebase_messaging_auto_init_enabled", "false"),
            metadata("firebase_analytics_collection_enabled", "false"),
        ))
    elif variant == "china":
        china = lock["china"]
        honor = china["honor"]
        dependencies.append(honor["dependency"])
        project["plugins"].append(honor["plugin"])
        project["dependencies"].append(honor["plugin_dependency"])
        project["repositories"].append(honor["repository"])
        for provider in ("huawei_android", "xiaomi", "oppo", "vivo", "meizu"):
            item = china[provider]
            dependencies.append(pinned_dependency(item["dependency_env"]))
            if item.get("repository"):
                project["repositories"].append(item["repository"])
        public = {
            "com.appkernia.push.huawei.app_id": required("AK_HUAWEI_PUSH_APP_ID"),
            "com.hihonor.push.app_id": required("AK_HONOR_PUSH_APP_ID"),
            "com.appkernia.push.xiaomi.app_id": required("AK_XIAOMI_PUSH_APP_ID"),
            "com.appkernia.push.xiaomi.app_key": required("AK_XIAOMI_PUSH_APP_KEY"),
            "com.appkernia.push.oppo.app_key": required("AK_OPPO_PUSH_APP_KEY"),
            "com.appkernia.push.oppo.app_secret": required("AK_OPPO_PUSH_APP_SECRET"),
            "com.appkernia.push.meizu.app_id": required("AK_MEIZU_PUSH_APP_ID"),
            "com.appkernia.push.meizu.app_key": required("AK_MEIZU_PUSH_APP_KEY"),
        }
        # Huawei Push Kit must remain inert until AppKernia's consent and Push
        # preference gates explicitly request registration.
        meta.append(metadata("push_kit_auto_init_enabled", "false"))
        meta.extend(metadata(key, value) for key, value in public.items())
        public_fields.extend(key.split(".")[-2] + "." + key.split(".")[-1] for key in public)
        for env_name, filename in (("AK_HUAWEI_CONFIG_FILE", "agconnect-services.json"), ("AK_HONOR_CONFIG_FILE", "mcs-services.json")):
            source = Path(required(env_name)).expanduser().resolve()
            if not source.is_file():
                raise SystemExit(f"{env_name} does not name a readable provider configuration file")
            shutil.copyfile(source, ROOT / "nativeResources" / "android" / filename)
            public_fields.append(env_name)

    config: dict[str, object] = {"minSdkVersion": 26}
    if dependencies:
        config["dependencies"] = dependencies
    if any(project.values()):
        config["project"] = {key: value for key, value in project.items() if value}
    OUTPUT_CONFIG.write_text(json.dumps(config, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    manifest = """<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android">
    <uses-permission android:name="android.permission.POST_NOTIFICATIONS" />
    <application>
{metadata}
    </application>
</manifest>
""".format(metadata="\n".join(meta))
    OUTPUT_MANIFEST.write_text(manifest, encoding="utf-8")
    SUMMARY.parent.mkdir(parents=True, exist_ok=True)
    digest = hashlib.sha256((OUTPUT_CONFIG.read_bytes() + OUTPUT_MANIFEST.read_bytes())).hexdigest()
    SUMMARY.write_text(json.dumps({"variant": variant, "config_sha256": digest, "public_field_names": sorted(public_fields), "contains_server_secret_values": False}, indent=2) + "\n", encoding="utf-8")
    print(f"configured Android push variant={variant} sha256={digest}")


if __name__ == "__main__":
    main()
