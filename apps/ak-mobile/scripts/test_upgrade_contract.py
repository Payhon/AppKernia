#!/usr/bin/env python3
"""Fast upgrade decision and wiring checks that do not claim native runtime evidence."""

from __future__ import annotations

import json
import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parents[1]


def parse(value: str) -> tuple[int, int, int] | None:
    if not re.fullmatch(r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)", value):
        return None
    return tuple(int(part) for part in value.split("."))  # type: ignore[return-value]


def compare(left: str, right: str) -> int | None:
    a, b = parse(left), parse(right)
    if a is None or b is None:
        return None
    return (a > b) - (a < b)


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(f"FAIL: {message}")


def main() -> None:
    cases = {
        ("0.2.0", "0.2.0"): 0,
        ("0.10.0", "0.2.0"): 1,
        ("2.0.0", "10.0.0"): -1,
        ("1.0", "1.0.0"): None,
        ("01.0.0", "1.0.0"): None,
        ("1.0.0-rc.1", "1.0.0"): None,
    }
    for values, expected in cases.items():
        require(compare(*values) == expected, f"SemVer decision mismatch for {values}")

    module = ROOT / "uni_modules/ak-upgrade"
    source = "\n".join(path.read_text(encoding="utf-8") for path in module.rglob("*") if path.suffix in {".uts", ".uvue"})
    require("uniCloud" not in source, "ak-upgrade must not depend on uniCloud")
    require("uts-progressNotification" not in source and "uts-openSchema" not in source, "ak-upgrade must remain dependency-free")
    require("'X-AppID': akUpgradeCoordinator.appId()" in source, "APK download must carry X-AppID")
    require("Authorization" not in (module / "pages/upgrade-dialog.uvue").read_text(encoding="utf-8"), "APK download must not carry a user token")
    require("uni.installApk" in source and "#ifdef APP-ANDROID" in source, "APK installation must be Android-only")
    require("config_mismatch" in source and "compareUpgradeSemver" in source, "upgrade gate must validate AppID and SemVer")

    pages = json.loads((ROOT / "pages.json").read_text(encoding="utf-8"))
    registered = {page["path"] for page in pages["pages"]}
    require("uni_modules/ak-upgrade/pages/upgrade-dialog" in registered, "upgrade dialog is not registered")
    openapi = (REPO / "server/openapi/openapi.yaml").read_text(encoding="utf-8")
    for fragment in ("delivery_mode:", "store_list:", "MobileAppVersionStore:"):
        require(fragment in openapi, f"OpenAPI upgrade projection missing {fragment}")
    print(f"AK upgrade static tests passed: {len(cases)} SemVer cases plus module/contract wiring.")


if __name__ == "__main__":
    main()
