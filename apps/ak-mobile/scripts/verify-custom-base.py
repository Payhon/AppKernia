#!/usr/bin/env python3
"""Verify source configuration and optionally inspect produced custom-base artifacts."""

from __future__ import annotations

import argparse
import io
import json
import plistlib
import re
import subprocess
import tempfile
import zipfile
from pathlib import Path

from PIL import Image, ImageChops, ImageStat


ROOT = Path(__file__).resolve().parent.parent
EXPECTED_NATIVE_ID = "com.appkernia.mobile"
HAP_SIGN_TOOL = Path(
    "/Applications/DevEco-Studio.app/Contents/sdk/default/openharmony/"
    "toolchains/lib/hap-sign-tool.jar"
)


def source_manifest() -> dict:
    return json.loads((ROOT / "manifest.json").read_text(encoding="utf-8"))


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(message)


def image_contract(relative: str, size: int, *, opaque: bool | None = None) -> None:
    path = ROOT / relative
    require(path.is_file(), f"missing native icon: {relative}")
    with Image.open(path) as image:
        require(image.size == (size, size), f"{relative} must be {size}x{size}, got {image.size}")
        if opaque is True:
            require("A" not in image.getbands(), f"{relative} must not contain an alpha channel")


def image_bytes_match(actual: bytes, expected_path: Path) -> bool:
    with Image.open(io.BytesIO(actual)) as actual_image, Image.open(expected_path) as expected_image:
        return (
            actual_image.size == expected_image.size
            and actual_image.convert("RGBA").tobytes() == expected_image.convert("RGBA").tobytes()
        )


def resized_brand_rms(actual: bytes, expected_path: Path) -> tuple[float, ...]:
    with Image.open(io.BytesIO(actual)) as actual_image, Image.open(expected_path) as expected_image:
        branded = expected_image.convert("RGBA").resize(actual_image.size, Image.Resampling.LANCZOS)
        difference = ImageChops.difference(actual_image.convert("RGBA"), branded)
        return tuple(ImageStat.Stat(difference).rms)


def verify_sources() -> None:
    manifest = source_manifest()
    android = manifest["app-android"]["distribute"]
    ios = manifest["app-ios"]["distribute"]
    require(android.get("syncDebug") is True, "Android custom-base syncDebug must be true")
    require(ios.get("syncDebug") is True, "iOS custom-base syncDebug must be true")

    for size in (72, 96, 144, 192):
        image_contract(f"static/app-icons/android/icon-{size}.png", size)
    for size in (480, 720, 960):
        image_contract(f"static/app-icons/android/splash-icon-{size}.png", size)
    image_contract("static/app-icons/ios/appstore-1024.png", 1024, opaque=True)
    image_contract("harmony-configs/AppScope/resources/base/media/background.png", 288, opaque=True)
    image_contract("harmony-configs/AppScope/resources/base/media/foreground.png", 288)
    image_contract("harmony-configs/AppScope/resources/base/media/startIcon.png", 144)

    harmony = json.loads((ROOT / "harmony-configs/AppScope/app.json5").read_text(encoding="utf-8"))
    require(harmony["app"]["bundleName"] == EXPECTED_NATIVE_ID, "Harmony bundleName is not AppKernia-owned")
    require(harmony["app"]["versionName"] == manifest["versionName"], "Harmony versionName drifted from manifest")
    require(harmony["app"]["versionCode"] == int(manifest["versionCode"]), "Harmony versionCode drifted from manifest")

    build_script = (ROOT / "scripts/build-platform.sh").read_text(encoding="utf-8")
    require(build_script.count("--playground custom") == 2, "Android/iOS launch paths must force custom playground")
    harmony_builder = (ROOT / "scripts/build-custom-base.sh").read_text(encoding="utf-8")
    require("prepare-harmony-native.py" in harmony_builder, "Harmony build must apply AppKernia native overlay")

    print("Custom-base source contract: PASS")


def find_artifact(patterns: tuple[str, ...]) -> Path | None:
    matches: list[Path] = []
    for pattern in patterns:
        matches.extend(ROOT.glob(pattern))
    files = [path for path in matches if path.is_file()]
    return max(files, key=lambda path: path.stat().st_mtime) if files else None


def verify_apk(path: Path) -> None:
    aapt = Path(
        "/Applications/HBuilderX.app/Contents/HBuilderX/plugins/"
        "uts-development-android/static/macosx/aapt2"
    )
    require(aapt.is_file(), "cannot inspect APK: HBuilderX aapt2 not found")
    result = subprocess.run([str(aapt), "dump", "badging", str(path)], check=True, capture_output=True, text=True)
    require(f"package: name='{EXPECTED_NATIVE_ID}'" in result.stdout, "Android artifact still uses a non-AppKernia package")
    require("application-label:'AppKernia'" in result.stdout, "Android artifact label is not AppKernia")
    require(
        f"versionName='{source_manifest()['versionName']}'" in result.stdout,
        "Android artifact version drifted from manifest",
    )
    icon_entries = set(re.findall(r"application-icon-\d+:'([^']+)'", result.stdout))
    require(icon_entries, "Android artifact does not declare launcher icons")
    expected_icons = {
        size: ROOT / f"static/app-icons/android/icon-{size}.png"
        for size in (72, 96, 144, 192)
    }
    matched_sizes: set[int] = set()
    with zipfile.ZipFile(path) as archive:
        archive_names = set(archive.namelist())
        for entry in icon_entries:
            if entry not in archive_names:
                continue
            icon_bytes = archive.read(entry)
            with Image.open(io.BytesIO(icon_bytes)) as image:
                size = image.width if image.width == image.height else -1
            expected = expected_icons.get(size)
            if expected is not None and image_bytes_match(icon_bytes, expected):
                matched_sizes.add(size)
    require(
        matched_sizes == set(expected_icons),
        f"Android launcher icons are not the generated AppKernia assets: matched {sorted(matched_sizes)}",
    )
    print(f"Android custom base: PASS ({path.relative_to(ROOT)})")


def verify_ios_artifact(path: Path) -> None:
    icon_payloads: list[bytes] = []
    if path.is_dir():
        info = plistlib.loads((path / "Info.plist").read_bytes())
        icon_payloads = [icon.read_bytes() for icon in path.glob("AppIcon*.png")]
        icon_exists = bool(icon_payloads) or (path / "Assets.car").is_file()
    else:
        with zipfile.ZipFile(path) as archive:
            names = [name for name in archive.namelist() if name.startswith("Payload/") and name.endswith(".app/Info.plist")]
            require(len(names) == 1, "iOS artifact does not contain exactly one app Info.plist")
            info = plistlib.loads(archive.read(names[0]))
            icon_names = [name for name in archive.namelist() if "AppIcon" in name and name.endswith(".png")]
            icon_payloads = [archive.read(name) for name in icon_names]
            icon_exists = bool(icon_payloads) or any(name.endswith("Assets.car") for name in archive.namelist())
    require(info.get("CFBundleIdentifier") == EXPECTED_NATIVE_ID, "iOS artifact bundle id is not AppKernia-owned")
    display_name = info.get("CFBundleDisplayName", info.get("CFBundleName"))
    require(display_name == "AppKernia", "iOS artifact display name is not AppKernia")
    require(
        info.get("CFBundleShortVersionString") == source_manifest()["versionName"],
        "iOS artifact version drifted from manifest",
    )
    require(icon_exists, "iOS artifact does not contain an AppIcon asset")
    require(icon_payloads, "iOS artifact AppIcon cannot be compared with the AppKernia brand master")
    master = ROOT / "static/app-icons/ios/appstore-1024.png"
    best_rms = min(
        (resized_brand_rms(payload, master) for payload in icon_payloads),
        key=lambda values: max(values[:3]),
    )
    require(
        max(best_rms[:3]) <= 10.0,
        f"iOS packaged AppIcon does not match the generated AppKernia icon: RGB RMS {best_rms[:3]}",
    )
    print(f"iOS custom base: PASS ({path.relative_to(ROOT)})")


def verify_hap_signature(path: Path) -> bool:
    require(HAP_SIGN_TOOL.is_file(), "cannot verify HAP signature: DevEco hap-sign-tool.jar not found")
    with tempfile.TemporaryDirectory(prefix="appkernia-hap-") as temp_dir:
        temp = Path(temp_dir)
        certificate = temp / "certificate.cer"
        provision = temp / "profile.p7b"
        result = subprocess.run(
            [
                "java",
                "-jar",
                str(HAP_SIGN_TOOL),
                "verify-app",
                "-inFile",
                str(path),
                "-outCertChain",
                str(certificate),
                "-outProfile",
                str(provision),
            ],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            return False
        require(certificate.is_file(), "signed HAP verification did not emit a certificate chain")
        require(provision.is_file(), "signed HAP verification did not emit a Provision Profile")

        profile_json = temp / "profile.json"
        profile_result = subprocess.run(
            [
                "java",
                "-jar",
                str(HAP_SIGN_TOOL),
                "verify-profile",
                "-inFile",
                str(provision),
                "-outFile",
                str(profile_json),
            ],
            capture_output=True,
            text=True,
        )
        require(
            profile_result.returncode == 0 and profile_json.is_file(),
            "signed HAP Provision Profile verification failed",
        )
        verified_text = profile_json.read_text(encoding="utf-8")
        require(EXPECTED_NATIVE_ID in verified_text, "signed HAP profile is not bound to AppKernia")
        require("io.dcloud.uniappx" not in verified_text, "signed HAP profile contains DCloud identity")
        return True


def verify_hap(path: Path, *, require_signed: bool = False) -> None:
    with zipfile.ZipFile(path) as archive:
        module = json.loads(archive.read("module.json"))
        names = set(archive.namelist())
        branded_media = {
            name: archive.read(name)
            for name in (
                "resources/base/media/background.png",
                "resources/base/media/foreground.png",
                "resources/base/media/startIcon.png",
            )
            if name in names
        }
    app = module["app"]
    require(app.get("bundleName") == EXPECTED_NATIVE_ID, "Harmony artifact bundle name is not AppKernia-owned")
    require(
        app.get("versionName") == source_manifest()["versionName"],
        "Harmony artifact version drifted from manifest",
    )
    require(app.get("icon") == "$media:layered_image", "Harmony artifact does not use the layered AppKernia icon")
    require("resources/base/media/background.png" in names, "Harmony artifact background icon is missing")
    require("resources/base/media/foreground.png" in names, "Harmony artifact foreground icon is missing")
    require("resources/base/media/startIcon.png" in names, "Harmony artifact start icon is missing")
    for name, relative in {
        "resources/base/media/background.png": "harmony-configs/AppScope/resources/base/media/background.png",
        "resources/base/media/foreground.png": "harmony-configs/AppScope/resources/base/media/foreground.png",
        "resources/base/media/startIcon.png": "harmony-configs/AppScope/resources/base/media/startIcon.png",
    }.items():
        require(
            branded_media.get(name) == (ROOT / relative).read_bytes(),
            f"Harmony artifact {name} is not the generated AppKernia asset",
        )
    require(
        any(f"apps/{source_manifest()['appid']}/" in name for name in names),
        "Harmony artifact runtime AppID is stale",
    )
    signed = verify_hap_signature(path)
    if require_signed:
        require(signed, "Harmony HAP is unsigned and cannot be installed on a physical device")
    kind = "signed" if signed else "unsigned"
    print(f"Harmony local HAP ({kind}): PASS ({path.relative_to(ROOT)})")


def verify_artifacts(platform: str, *, require_harmony_signed: bool = False) -> None:
    if platform in ("all", "android"):
        apk = find_artifact(("unpackage/debug/*android*.apk",))
        require(apk is not None, "Android custom-base APK not found")
        verify_apk(apk)
    if platform in ("all", "ios"):
        ipa = find_artifact(("unpackage/debug/*.ipa",))
        apps = [item for item in ROOT.glob("unpackage/debug/*simulator*.app") if item.is_dir()]
        ios_artifact = ipa or (max(apps, key=lambda item: item.stat().st_mtime) if apps else None)
        require(ios_artifact is not None, "iOS custom-base IPA/App not found")
        verify_ios_artifact(ios_artifact)
    if platform in ("all", "harmony"):
        hap = find_artifact((
            "unpackage/dist/dev/app-harmony/entry/build/*/outputs/*/*.hap",
        ))
        require(hap is not None, "Harmony HAP not found")
        verify_hap(hap, require_signed=require_harmony_signed)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--artifacts", action="store_true")
    parser.add_argument("--require-harmony-signed", action="store_true")
    parser.add_argument("--platform", choices=("all", "android", "ios", "harmony"), default="all")
    args = parser.parse_args()
    verify_sources()
    if args.artifacts:
        verify_artifacts(args.platform, require_harmony_signed=args.require_harmony_signed)


if __name__ == "__main__":
    main()
