#!/usr/bin/env python3
"""Dependency-free static validator for the AppKernia database blueprint.

This does not replace executing migrations against PostgreSQL 18. It catches
common packaging, ordering, naming, pairing, and structural errors before the
real migration integration test runs.
"""
from __future__ import annotations

import json
import re
import sys
from collections import Counter
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MIGRATIONS = ROOT / "db" / "migrations"


def strip_sql_noise(text: str) -> str:
    # Good enough for these controlled migration files: remove comments,
    # single-quoted literals, and dollar-quoted function bodies.
    text = re.sub(r"/\*.*?\*/", "", text, flags=re.S)
    text = re.sub(r"--[^\n]*", "", text)
    text = re.sub(r"\$\$.*?\$\$", "''", text, flags=re.S)
    text = re.sub(r"'(?:''|[^'])*'", "''", text)
    return text


def balanced_parentheses(text: str) -> bool:
    level = 0
    for ch in strip_sql_noise(text):
        if ch == "(":
            level += 1
        elif ch == ")":
            level -= 1
            if level < 0:
                return False
    return level == 0


def extract(pattern: str, text: str) -> list[str]:
    return re.findall(pattern, strip_sql_noise(text), flags=re.I | re.M)


def main() -> int:
    errors: list[str] = []
    warnings: list[str] = []
    up_files = sorted(MIGRATIONS.glob("*.up.sql"))
    down_files = sorted(MIGRATIONS.glob("*.down.sql"))

    if not up_files:
        errors.append("No .up.sql migration files found")

    up_by_prefix = {p.name.split(".", 1)[0]: p for p in up_files}
    down_by_prefix = {p.name.split(".", 1)[0]: p for p in down_files}
    if set(up_by_prefix) != set(down_by_prefix):
        errors.append(
            f"Migration pair mismatch: up={sorted(up_by_prefix)}, down={sorted(down_by_prefix)}"
        )

    all_up = "\n".join(p.read_text(encoding="utf-8") for p in up_files)
    clean_all = strip_sql_noise(all_up)

    tables = extract(r"\bCREATE\s+TABLE\s+([A-Za-z_][\w]*\.[A-Za-z_][\w]*)", all_up)
    indexes = extract(
        r"\bCREATE\s+(?:UNIQUE\s+)?INDEX\s+([A-Za-z_][\w]*)", all_up
    )
    triggers = extract(r"\bCREATE\s+TRIGGER\s+([A-Za-z_][\w]*)", all_up)
    trigger_tables = set(
        extract(
            r"\bBEFORE\s+UPDATE\s+ON\s+([A-Za-z_][\w]*\.[A-Za-z_][\w]*)",
            all_up,
        )
    )
    refs = extract(
        r"\bREFERENCES\s+([A-Za-z_][\w]*\.[A-Za-z_][\w]*)", all_up
    )

    for label, values in (("table", tables), ("index", indexes), ("trigger", triggers)):
        dup = [name for name, count in Counter(values).items() if count > 1]
        if dup:
            errors.append(f"Duplicate {label} names: {dup}")

    missing_refs = sorted(set(refs) - set(tables))
    if missing_refs:
        errors.append(f"Foreign keys reference tables not created in this package: {missing_refs}")

    for p in up_files:
        text = p.read_text(encoding="utf-8")
        if len(re.findall(r"\bBEGIN\s*;", strip_sql_noise(text), flags=re.I)) != 1:
            errors.append(f"{p.name}: expected exactly one BEGIN")
        if len(re.findall(r"\bCOMMIT\s*;", strip_sql_noise(text), flags=re.I)) != 1:
            errors.append(f"{p.name}: expected exactly one COMMIT")
        if not balanced_parentheses(text):
            errors.append(f"{p.name}: unbalanced parentheses")

        prefix = p.name.split(".", 1)[0]
        down = down_by_prefix.get(prefix)
        if down:
            down_text = strip_sql_noise(down.read_text(encoding="utf-8"))
            created = extract(
                r"\bCREATE\s+TABLE\s+([A-Za-z_][\w]*\.[A-Za-z_][\w]*)", text
            )
            dropped = set(
                re.findall(
                    r"\bDROP\s+TABLE\s+IF\s+EXISTS\s+([A-Za-z_][\w]*\.[A-Za-z_][\w]*)",
                    down_text,
                    flags=re.I,
                )
            )
            missing_drops = [t for t in created if t not in dropped]
            if missing_drops:
                errors.append(f"{down.name}: missing DROP TABLE for {missing_drops}")
            if not balanced_parentheses(down.read_text(encoding="utf-8")):
                errors.append(f"{down.name}: unbalanced parentheses")

    # Identify tables containing updated_at and require a touch trigger, except
    # immutable/static tables intentionally maintained by import code.
    updated_at_tables: set[str] = set()
    for match in re.finditer(
        r"CREATE\s+TABLE\s+([\w]+\.[\w]+)\s*\((.*?)\n\);",
        all_up,
        flags=re.I | re.S,
    ):
        table, body = match.group(1), match.group(2)
        if re.search(r"\bupdated_at\b", body, flags=re.I):
            updated_at_tables.add(table)
    trigger_exceptions = {"sys.regions"}
    missing_touch = sorted(updated_at_tables - trigger_tables - trigger_exceptions)
    if missing_touch:
        errors.append(f"Tables with updated_at but no BEFORE UPDATE trigger: {missing_touch}")

    # Security-oriented schema checks for accidental plaintext credential fields.
    prohibited_exact_columns = {
        "password",
        "refresh_token",
        "access_token",
        "api_secret",
        "otp_code",
        "push_token",
    }
    column_candidates = re.findall(
        r"^\s{4}([A-Za-z_][\w]*)\s+[A-Za-z]", all_up, flags=re.M
    )
    bad_columns = sorted(prohibited_exact_columns.intersection(column_candidates))
    if bad_columns:
        errors.append(f"Prohibited plaintext credential-like columns found: {bad_columns}")

    # Validate machine-readable seeds.
    for name in ("core-permissions.json", "core-menus.json"):
        path = ROOT / "spec" / name
        try:
            payload = json.loads(path.read_text(encoding="utf-8"))
        except Exception as exc:  # noqa: BLE001 - validator output should include parser error
            errors.append(f"{name}: invalid JSON: {exc}")
            continue
        if payload.get("schema_version") != 1:
            warnings.append(f"{name}: unexpected schema_version")

    permission_path = ROOT / "spec" / "core-permissions.json"
    if permission_path.exists():
        payload = json.loads(permission_path.read_text(encoding="utf-8"))
        codes = [item["code"] for item in payload.get("permissions", [])]
        dup_codes = [code for code, count in Counter(codes).items() if count > 1]
        if dup_codes:
            errors.append(f"Duplicate permission codes: {dup_codes}")

    menu_path = ROOT / "spec" / "core-menus.json"
    if menu_path.exists():
        payload = json.loads(menu_path.read_text(encoding="utf-8"))
        menus = payload.get("menus", [])
        codes = [item["code"] for item in menus]
        known = set(codes)
        dup_codes = [code for code, count in Counter(codes).items() if count > 1]
        if dup_codes:
            errors.append(f"Duplicate menu codes: {dup_codes}")
        missing_parents = sorted(
            {item.get("parent") for item in menus if item.get("parent") and item.get("parent") not in known}
        )
        if missing_parents:
            errors.append(f"Menus reference missing parents: {missing_parents}")

    core_tables = [t for t in tables if not t.startswith("billing.")]
    billing_tables = [t for t in tables if t.startswith("billing.")]
    metrics = {
        "up_migrations": len(up_files),
        "down_migrations": len(down_files),
        "core_tables": len(core_tables),
        "optional_billing_tables": len(billing_tables),
        "total_tables": len(tables),
        "indexes": len(indexes),
        "triggers": len(triggers),
        "foreign_key_references": len(refs),
    }

    print("AppKernia blueprint static validation")
    print(json.dumps(metrics, ensure_ascii=False, indent=2))
    for warning in warnings:
        print(f"WARNING: {warning}")
    for error in errors:
        print(f"ERROR: {error}")

    if errors:
        print(f"FAILED: {len(errors)} error(s), {len(warnings)} warning(s)")
        return 1
    print(f"PASSED: 0 errors, {len(warnings)} warning(s)")
    print("NOTE: Static validation does not replace running migrations on PostgreSQL 18.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
