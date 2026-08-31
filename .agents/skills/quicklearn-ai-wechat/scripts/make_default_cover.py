#!/usr/bin/env python3
"""Generate the deterministic 快学AI fallback cover asset."""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


FONT_CANDIDATES = (
    Path("/System/Library/Fonts/PingFang.ttc"),
    Path("/System/Library/Fonts/STHeiti Medium.ttc"),
    Path("/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"),
    Path("/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc"),
)


def load_font(size: int) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    for candidate in FONT_CANDIDATES:
        if candidate.is_file():
            return ImageFont.truetype(str(candidate), size=size)
    return ImageFont.load_default(size=size)


def generate(output: Path) -> None:
    width, height = 900, 383
    image = Image.new("RGB", (width, height), "#07142f")
    draw = ImageDraw.Draw(image)
    top = (7, 20, 47)
    bottom = (14, 72, 112)
    for y in range(height):
        ratio = y / max(height - 1, 1)
        color = tuple(round(top[index] * (1 - ratio) + bottom[index] * ratio) for index in range(3))
        draw.line((0, y, width, y), fill=color)

    draw.ellipse((650, -150, 1020, 220), fill="#0d6f9e")
    draw.ellipse((720, 205, 980, 465), fill="#123f72")
    draw.rounded_rectangle((70, 62, 166, 158), radius=24, fill="#22d3ee")
    draw.line((94, 111, 141, 111), fill="#07142f", width=8)
    draw.line((117, 87, 117, 135), fill="#07142f", width=8)

    title_font = load_font(68)
    subtitle_font = load_font(28)
    label_font = load_font(18)
    draw.text((70, 184), "快学AI", font=title_font, fill="#ffffff")
    draw.text((72, 281), "把复杂技术讲明白", font=subtitle_font, fill="#c8f4ff")
    draw.rounded_rectangle((657, 160, 826, 204), radius=22, fill="#ffffff")
    draw.text((684, 171), "TECH NOTES", font=label_font, fill="#0b527c")

    output.parent.mkdir(parents=True, exist_ok=True)
    image.save(output, format="JPEG", quality=92, optimize=True, progressive=True)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    generate(args.output.expanduser().resolve())
    print(args.output.expanduser().resolve())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
