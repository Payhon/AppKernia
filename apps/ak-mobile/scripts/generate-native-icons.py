#!/usr/bin/env python3
"""Generate deterministic Android, iOS and HarmonyOS icons from the AK brand master."""

from __future__ import annotations

import json
from pathlib import Path

from PIL import Image


MOBILE_ROOT = Path(__file__).resolve().parent.parent
REPO_ROOT = MOBILE_ROOT.parents[1]
SOURCE = REPO_ROOT / "apps/ak-admin/public/brand/appkernia-mark.png"
OUTPUT = MOBILE_ROOT / "static/app-icons"
HARMONY = MOBILE_ROOT / "harmony-configs/AppScope"
BRAND_BLUE = (36, 107, 254, 255)


def trimmed_master() -> Image.Image:
    source = Image.open(SOURCE).convert("RGBA")
    bounds = source.getchannel("A").getbbox()
    if bounds is None:
        raise SystemExit(f"brand master is fully transparent: {SOURCE}")
    return source.crop(bounds)


def square_icon(master: Image.Image, size: int, *, opaque: bool, fill_ratio: float = 0.92) -> Image.Image:
    background = BRAND_BLUE if opaque else (0, 0, 0, 0)
    canvas = Image.new("RGBA", (size, size), background)
    limit = max(1, round(size * fill_ratio))
    scale = min(limit / master.width, limit / master.height)
    resized = master.resize(
        (max(1, round(master.width * scale)), max(1, round(master.height * scale))),
        Image.Resampling.LANCZOS,
    )
    offset = ((size - resized.width) // 2, (size - resized.height) // 2)
    canvas.alpha_composite(resized, offset)
    return canvas.convert("RGB") if opaque else canvas


def save_png(image: Image.Image, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    image.save(path, format="PNG", optimize=True)


def write_harmony_app_config() -> None:
    manifest = json.loads((MOBILE_ROOT / "manifest.json").read_text(encoding="utf-8"))
    config = {
        "app": {
            "bundleName": "com.appkernia.mobile",
            "vendor": "AppKernia",
            "versionCode": int(manifest["versionCode"]),
            "versionName": manifest["versionName"],
            "icon": "$media:layered_image",
            "label": "$string:app_name",
        }
    }
    path = HARMONY / "app.json5"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(config, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def main() -> None:
    master = trimmed_master()

    for size in (72, 96, 144, 192):
        save_png(square_icon(master, size, opaque=False), OUTPUT / f"android/icon-{size}.png")
    for size in (480, 720, 960):
        save_png(
            square_icon(master, size, opaque=False, fill_ratio=0.60),
            OUTPUT / f"android/splash-icon-{size}.png",
        )

    # iOS applies its own rounded mask. Bleed the existing rounded-square mark
    # past the canvas edge to avoid a second visible corner/background ring.
    save_png(square_icon(master, 1024, opaque=True, fill_ratio=1.12), OUTPUT / "ios/appstore-1024.png")

    # DevEco's HarmonyOS ability template uses 288px foreground/background
    # layers and a 144px start icon. Keep the complete AppKernia mark on the
    # foreground layer so its white A and green orbital stroke survive HDS
    # masking instead of being upscaled from an undersized bitmap.
    save_png(Image.new("RGB", (288, 288), BRAND_BLUE[:3]), HARMONY / "resources/base/media/background.png")
    save_png(square_icon(master, 288, opaque=False), HARMONY / "resources/base/media/foreground.png")
    save_png(square_icon(master, 144, opaque=False), HARMONY / "resources/base/media/startIcon.png")
    save_png(square_icon(master, 144, opaque=False), MOBILE_ROOT / "nativeResources/android/res/drawable-xxhdpi/icon_round.png")
    write_harmony_app_config()

    print(f"Generated native icon assets from {SOURCE.relative_to(REPO_ROOT)}")


if __name__ == "__main__":
    main()
