#!/usr/bin/env python3
"""Verify that the offline startup snapshot still matches the native manifest."""

from __future__ import annotations

import json
import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "manifest.json"
SNAPSHOT = ROOT / "src/generated/startup-snapshot.uts"


def snapshot_value(source: str, field: str) -> str:
    match = re.search(rf"\b{re.escape(field)}:\s*'([^']+)'", source)
    if match is None:
        raise SystemExit(f"bundled startup snapshot is missing {field}")
    return match.group(1).strip()


def main() -> None:
    manifest = json.loads(MANIFEST.read_text(encoding="utf-8"))
    manifest_app_id = str(manifest.get("akRuntime", {}).get("appId", "")).strip()
    source = SNAPSHOT.read_text(encoding="utf-8")
    snapshot_app_id = snapshot_value(source, "appId")
    icon_path = snapshot_value(source, "iconPath")

    if not manifest_app_id:
        raise SystemExit("manifest akRuntime.appId is required")
    if snapshot_app_id != manifest_app_id:
        raise SystemExit(
            "bundled startup snapshot appId drifted from manifest.json: "
            f"snapshot={snapshot_app_id}, manifest={manifest_app_id}; "
            "rerun ak-cli app-startup export before packaging"
        )
    if not icon_path.startswith("/static/"):
        raise SystemExit("bundled startup iconPath must reference a packaged /static/ asset")
    icon_file = ROOT / icon_path.removeprefix("/")
    if not icon_file.is_file():
        raise SystemExit(f"bundled startup icon is missing: {icon_file.relative_to(ROOT)}")

    for locale_field in ("zhCN", "enUS"):
        block = re.search(rf"\b{locale_field}:\s*\{{([^}}]+)\}}", source)
        if block is None:
            raise SystemExit(f"bundled startup snapshot is missing {locale_field}")
        for text_field in ("displayName", "subtitle"):
            value = snapshot_value(block.group(1), text_field)
            if not value:
                raise SystemExit(f"bundled startup snapshot has an empty {locale_field}.{text_field}")

    print("bundled startup snapshot matches manifest App ID and packaged assets")


if __name__ == "__main__":
    main()
