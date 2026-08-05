#!/usr/bin/env python3
"""Convert HotGo's PostgreSQL province seed into AppKernia's region catalog."""

from __future__ import annotations

import argparse
import ast
import json
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path


START_MARKER = "-- 转存表中的数据 `hg_sys_provinces`"
END_MARKER = "-- 转存表中的数据 hg_sys_serve_license"
COORDINATE_SCALE = Decimal("0.0000001")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--source-commit", required=True)
    parser.add_argument("--output", type=Path, required=True)
    return parser.parse_args()


def coordinate(value: str) -> str | None:
    if not value:
        return None
    normalized = Decimal(value).quantize(COORDINATE_SCALE, rounding=ROUND_HALF_UP)
    return format(normalized, "f")


def load_rows(source: Path) -> list[tuple[object, ...]]:
    sql = source.read_text(encoding="utf-8")
    try:
        block = sql[sql.index(START_MARKER) : sql.index(END_MARKER)]
    except ValueError as exc:
        raise SystemExit(f"HotGo province seed markers were not found in {source}") from exc
    rows: list[tuple[object, ...]] = []
    for line in block.splitlines():
        candidate = line.strip().rstrip(",;")
        if candidate.startswith("("):
            row = ast.literal_eval(candidate)
            if not isinstance(row, tuple) or len(row) != 12:
                raise SystemExit(f"Unexpected province row: {candidate[:120]}")
            rows.append(row)
    return rows


def build_catalog(rows: list[tuple[object, ...]], commit: str) -> dict[str, object]:
    by_code = {int(row[0]): row for row in rows}
    if len(by_code) != len(rows):
        raise SystemExit("HotGo province seed contains duplicate region codes")

    full_names: dict[int, str] = {}

    def full_name(code: int) -> str:
        if code in full_names:
            return full_names[code]
        row = by_code[code]
        parent = int(row[5])
        if parent and parent not in by_code:
            raise SystemExit(f"Region {code} references missing parent {parent}")
        value = str(row[1]) if not parent else f"{full_name(parent)} / {row[1]}"
        full_names[code] = value
        return value

    regions: list[dict[str, object]] = []
    for row in sorted(rows, key=lambda item: (int(item[6]), int(item[0]))):
        code = int(row[0])
        parent = int(row[5])
        source_level = int(row[6])
        if source_level not in (1, 2, 3):
            raise SystemExit(f"Region {code} has unsupported level {source_level}")
        if bool(parent) != (source_level > 1):
            raise SystemExit(f"Region {code} has inconsistent parent and level")
        if parent and int(by_code[parent][6]) != source_level - 1:
            raise SystemExit(f"Region {code} is not directly below parent {parent}")
        regions.append(
            {
                "code": str(code),
                "parent_code": str(parent) if parent else None,
                "level": source_level - 1,
                "name": str(row[1]),
                "full_name": full_name(code),
                "postal_code": None,
                "longitude": coordinate(str(row[3])),
                "latitude": coordinate(str(row[4])),
                "status": "active" if int(row[9]) == 1 else "disabled",
                "metadata": {
                    "pinyin_initial": str(row[2]),
                    "source_level": source_level,
                    "sort_order": int(row[8]),
                },
            }
        )

    return {
        "version": 1,
        "source": {
            "project": "HotGo",
            "repository": "https://github.com/bufanyun/hotgo",
            "commit": commit,
            "path": "server/storage/data/hotgo-pg.sql",
            "license": "MIT",
            "copyright": "Copyright (c) 2021-present HotGo",
            "note": "HotGo snapshot; not asserted to be the latest official administrative division catalog.",
        },
        "level_mapping": {"HotGo 1": 0, "HotGo 2": 1, "HotGo 3": 2},
        "regions": regions,
    }


def main() -> None:
    args = parse_args()
    catalog = build_catalog(load_rows(args.source), args.source_commit)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(
        json.dumps(catalog, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    levels: dict[int, int] = {}
    for region in catalog["regions"]:
        level = int(region["level"])
        levels[level] = levels.get(level, 0) + 1
    print(f"wrote {len(catalog['regions'])} regions to {args.output}; levels={levels}")


if __name__ == "__main__":
    main()
