#!/usr/bin/env python3
"""Fail when an Android artifact contains SDK signatures from the other variant."""

from __future__ import annotations

import argparse
import zipfile
from pathlib import Path


GOOGLE_MARKERS = (b"com/google/firebase/messaging", b"com/google/android/gms")
CHINA_MARKERS = (b"com/huawei/hms/push", b"com/hihonor/push", b"com/xiaomi/mipush", b"com/heytap/msp/push", b"com/vivo/push", b"com/meizu/cloud/pushsdk")


def artifact_bytes(path: Path) -> bytes:
    with zipfile.ZipFile(path) as archive:
        chunks = []
        for item in archive.infolist():
            if item.file_size <= 64 * 1024 * 1024 and (item.filename.endswith(".dex") or item.filename.endswith(".jar")):
                chunks.append(archive.read(item))
        return b"".join(chunks)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("variant", choices=("google", "china"))
    parser.add_argument("artifact", type=Path)
    args = parser.parse_args()
    payload = artifact_bytes(args.artifact)
    forbidden = CHINA_MARKERS if args.variant == "google" else GOOGLE_MARKERS
    required = GOOGLE_MARKERS[:1] if args.variant == "google" else CHINA_MARKERS
    leaked = [marker.decode() for marker in forbidden if marker in payload]
    if leaked:
        raise SystemExit("forbidden push SDK markers: " + ", ".join(leaked))
    if not any(marker in payload for marker in required):
        raise SystemExit(f"artifact does not contain the expected {args.variant} push SDK marker")
    print(f"verified {args.variant} push dependency boundary: {args.artifact}")


if __name__ == "__main__":
    main()
