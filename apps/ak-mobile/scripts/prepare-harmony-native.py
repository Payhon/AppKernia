#!/usr/bin/env python3
"""Overlay AppKernia identity and resources onto HBuilderX's generated Harmony project."""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
SOURCE = ROOT / "harmony-configs/AppScope"
NATIVE_ROOT = ROOT / "unpackage/dist/dev/app-harmony"
NATIVE = NATIVE_ROOT / "AppScope"
BUILD_PROFILE = NATIVE_ROOT / "build-profile.json5"
SIGN_TOOL = Path(
    "/Applications/DevEco-Studio.app/Contents/sdk/default/openharmony/"
    "toolchains/lib/hap-sign-tool.jar"
)
APP_ID = json.loads((ROOT / "manifest.json").read_text(encoding="utf-8"))["appid"]
APP_RESOURCES = (
    ROOT
    / "unpackage/dist/dev/app-harmony/entry/src/main/resources/resfile/uni-app-x/apps"
)


def disable_generated_signing() -> None:
    """Remove stale HBuilderX debug signing only from the generated build tree."""

    if not BUILD_PROFILE.is_file():
        raise SystemExit("generated Harmony build-profile.json5 is missing")
    profile = json.loads(BUILD_PROFILE.read_text(encoding="utf-8"))
    profile.setdefault("app", {})["signingConfigs"] = []
    for product in profile["app"].get("products", []):
        product.pop("signingConfig", None)
    BUILD_PROFILE.write_text(
        json.dumps(profile, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )


def material_path(material: dict, key: str) -> Path:
    value = material.get(key)
    if not isinstance(value, str) or not value:
        raise SystemExit(f"Harmony signing material is missing {key}")
    path = Path(value).expanduser()
    if not path.is_absolute():
        path = NATIVE_ROOT / path
    if not path.is_file():
        raise SystemExit(f"Harmony signing material file is missing: {key}")
    return path


def require_appkernia_signing() -> None:
    """Validate a DevEco signing config without printing credentials or paths."""

    if not BUILD_PROFILE.is_file():
        raise SystemExit("generated Harmony build-profile.json5 is missing")
    try:
        profile = json.loads(BUILD_PROFILE.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise SystemExit(f"generated Harmony build profile is not strict JSON: {exc}") from exc

    app = profile.get("app", {})
    configs = app.get("signingConfigs", [])
    named_configs = {
        item.get("name"): item
        for item in configs
        if isinstance(item, dict) and isinstance(item.get("name"), str) and item.get("name")
    }
    if not named_configs:
        raise SystemExit(
            "Harmony signing is not configured; use DevEco Project Structure > Signing Configs first"
        )

    product = next(
        (item for item in app.get("products", []) if item.get("name") == "default"),
        None,
    )
    if not product or product.get("signingConfig") not in named_configs:
        raise SystemExit("Harmony default product does not reference a valid signing config")

    config = named_configs[product["signingConfig"]]
    if config.get("type") not in (None, "HarmonyOS"):
        raise SystemExit("Harmony signing config has an unexpected platform type")
    material = config.get("material")
    if not isinstance(material, dict):
        raise SystemExit("Harmony signing config has no material block")
    for secret_key in ("storePassword", "keyPassword", "keyAlias"):
        if not isinstance(material.get(secret_key), str) or not material[secret_key]:
            raise SystemExit(f"Harmony signing material is missing {secret_key}")

    material_path(material, "certpath")
    material_path(material, "storeFile")
    provision = material_path(material, "profile")
    if not SIGN_TOOL.is_file():
        raise SystemExit("DevEco hap-sign-tool.jar is missing")

    with tempfile.TemporaryDirectory(prefix="appkernia-profile-") as temp_dir:
        output = Path(temp_dir) / "profile.json"
        result = subprocess.run(
            [
                "java",
                "-jar",
                str(SIGN_TOOL),
                "verify-profile",
                "-inFile",
                str(provision),
                "-outFile",
                str(output),
            ],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0 or not output.is_file():
            raise SystemExit("Harmony Provision Profile signature verification failed")
        verified_text = output.read_text(encoding="utf-8")
        if "com.appkernia.mobile" not in verified_text:
            raise SystemExit("Harmony Provision Profile is not bound to com.appkernia.mobile")
        if "io.dcloud.uniappx" in verified_text:
            raise SystemExit("Harmony Provision Profile still contains the DCloud default bundle")

    print("Harmony signing config: PASS (credentials redacted)")


def normalize_runtime_app_id() -> None:
    """Keep the generated uni-app x resource directory aligned with manifest.appid."""

    if not APP_RESOURCES.is_dir():
        raise SystemExit("generated Harmony uni-app x resource directory is missing")
    app_dirs = [item for item in APP_RESOURCES.iterdir() if item.is_dir()]
    if len(app_dirs) != 1:
        raise SystemExit("expected exactly one generated Harmony runtime app directory")
    current = app_dirs[0]
    old_app_id = current.name
    if old_app_id != APP_ID:
        destination = APP_RESOURCES / APP_ID
        if destination.exists():
            raise SystemExit(f"cannot replace existing Harmony runtime app directory: {destination}")
        current.rename(destination)

        native_sources = ROOT / "unpackage/dist/dev/app-harmony/entry/src/main"
        old_bytes = old_app_id.encode()
        new_bytes = APP_ID.encode()
        candidates = [item for item in native_sources.rglob("*") if item.is_file()]
        candidates.append(ROOT / "unpackage/dist/dev/app-harmony/entry/build-profile.json5")
        for candidate in candidates:
            data = candidate.read_bytes()
            if old_bytes in data:
                candidate.write_bytes(data.replace(old_bytes, new_bytes))

    final_dirs = [item.name for item in APP_RESOURCES.iterdir() if item.is_dir()]
    if final_dirs != [APP_ID]:
        raise SystemExit("Harmony runtime AppID normalization failed")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--unsigned",
        action="store_true",
        help="remove generated signing references so hvigor emits an unsigned HAP",
    )
    parser.add_argument(
        "--require-signing",
        action="store_true",
        help="require and validate a DevEco signing config for com.appkernia.mobile",
    )
    parser.add_argument(
        "--check-signing",
        action="store_true",
        help="validate signing without modifying the generated native project",
    )
    args = parser.parse_args()
    selected_modes = sum((args.unsigned, args.require_signing, args.check_signing))
    if selected_modes > 1:
        parser.error("--unsigned, --require-signing and --check-signing are mutually exclusive")

    if args.check_signing:
        require_appkernia_signing()
        return

    if not (NATIVE / "app.json5").is_file():
        raise SystemExit("generated Harmony native project is missing; run HBuilderX compilation first")

    normalize_runtime_app_id()
    source_config = json.loads((SOURCE / "app.json5").read_text(encoding="utf-8"))
    native_config = json.loads((NATIVE / "app.json5").read_text(encoding="utf-8"))
    native_config["app"].update(source_config["app"])
    (NATIVE / "app.json5").write_text(
        json.dumps(native_config, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )

    source_resources = SOURCE / "resources"
    native_resources = NATIVE / "resources"
    for source in source_resources.rglob("*"):
        if source.is_file():
            destination = native_resources / source.relative_to(source_resources)
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)

    verified = json.loads((NATIVE / "app.json5").read_text(encoding="utf-8"))
    if verified["app"]["bundleName"] != "com.appkernia.mobile":
        raise SystemExit("Harmony native bundleName overlay failed")
    if args.unsigned:
        disable_generated_signing()
    elif args.require_signing:
        require_appkernia_signing()
    print("Prepared Harmony native project for com.appkernia.mobile")
    print(f"Harmony runtime AppID: {APP_ID}")
    if args.unsigned:
        print("Harmony signing: disabled in generated build tree (unsigned HAP)")


if __name__ == "__main__":
    main()
