#!/usr/bin/env python3
"""Verify the native TabBar source canvases and optical glyph bounds."""

from pathlib import Path

from PIL import Image


ROOT = Path(__file__).resolve().parents[1] / "static" / "ak-tabbar"
TABS = ("home", "browse", "topics", "profile")
STATES = ("default", "selected", "selected-dark")
CANVAS = (81, 81)
TARGET_EDGE = 66
MAX_CENTER_OFFSET = 0.5


def verify(path: Path) -> None:
    image = Image.open(path).convert("RGBA")
    if image.size != CANVAS:
        raise SystemExit(f"{path.name}: expected {CANVAS[0]}x{CANVAS[1]} canvas, got {image.size}")

    bounds = image.getchannel("A").getbbox()
    if bounds is None:
        raise SystemExit(f"{path.name}: icon has no visible pixels")

    width = bounds[2] - bounds[0]
    height = bounds[3] - bounds[1]
    if max(width, height) != TARGET_EDGE:
        raise SystemExit(
            f"{path.name}: expected {TARGET_EDGE}px optical maximum edge, got {width}x{height}"
        )

    canvas_center_x = CANVAS[0] / 2
    canvas_center_y = CANVAS[1] / 2
    glyph_center_x = (bounds[0] + bounds[2]) / 2
    glyph_center_y = (bounds[1] + bounds[3]) / 2
    if abs(glyph_center_x - canvas_center_x) > MAX_CENTER_OFFSET or abs(glyph_center_y - canvas_center_y) > MAX_CENTER_OFFSET:
        raise SystemExit(f"{path.name}: glyph is not optically centered, bounds={bounds}")

    print(f"{path.name}: {width}x{height}, bounds={bounds}")


def main() -> None:
    for tab in TABS:
        for state in STATES:
            verify(ROOT / f"{tab}-{state}.png")
    print("AK Mobile TabBar icon geometry passed: 12 assets.")


if __name__ == "__main__":
    main()
