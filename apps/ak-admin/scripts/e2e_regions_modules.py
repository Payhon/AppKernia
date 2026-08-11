from __future__ import annotations

import json
import os
import uuid
from pathlib import Path

from playwright.sync_api import Page, Route, sync_playwright

from e2e_navigation_helpers import open_system_page

ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "output/playwright"
AXE = ROOT / "apps/ak-admin/node_modules/axe-core/axe.min.js"
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174").rstrip("/")
EMAIL = os.environ["AK_E2E_EMAIL"]
PASSWORD = os.environ["AK_E2E_PASSWORD"]
EXECUTABLE = os.environ.get("AK_E2E_BROWSER_EXECUTABLE")


def audit(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate(
        "async()=>await axe.run('main',"
        "{resultTypes:['violations']})"
    )
    evidence[name] = {
        "scope": "main",
        "violations": [
            {"id": value["id"], "impact": value["impact"], "nodes": len(value["nodes"])}
            for value in result["violations"]
        ]
    }
    assert not result["violations"], result["violations"]
    size = page.evaluate(
        "()=>({client:document.documentElement.clientWidth,"
        "scroll:document.documentElement.scrollWidth})"
    )
    assert size["scroll"] <= size["client"], size


def shot(page: Page, name: str, evidence: dict[str, object]) -> None:
    audit(page, name, evidence)
    page.screenshot(path=OUTPUT / f"admin-{name}.png", full_page=True)


def login(page: Page) -> None:
    page.goto(f"{BASE}/login", wait_until="networkidle")
    page.get_by_label("账号", exact=True).fill(EMAIL)
    page.get_by_label("密码", exact=True).fill(PASSWORD)
    page.locator('button[type="submit"]').click()
    page.wait_for_url(lambda url: url.startswith(f"{BASE}/dashboard"))


def select_language(page: Page, option: str) -> None:
    page.locator(
        'button[aria-label="显示语言"], button[aria-label="Display language"]'
    ).wait_for()
    for trigger in ("显示语言", "Display language"):
        button = page.get_by_role("button", name=trigger, exact=True)
        if button.count() == 1:
            button.click()
            break
    else:
        raise AssertionError("language switch is not available")
    page.get_by_role("menuitem", name=option, exact=True).click()


def open_regions(page: Page, system: str, settings: str, regions: str) -> None:
    del system, regions
    open_system_page(page, "/system/settings/regions", settings)
    page.wait_for_url(f"{BASE}/system/settings/regions")


def remove_region(page: Page, row_text: str) -> None:
    row = page.get_by_role("row").filter(has_text=row_text)
    row.get_by_role("button", name="Delete", exact=True).click()
    page.get_by_text("Delete region node?", exact=True).wait_for()
    page.locator(".ant-popconfirm").get_by_role(
        "button", name="Delete", exact=True
    ).click()


def strip_region_writes(route: Route) -> None:
    response = route.fetch()
    payload = response.json()
    payload["data"]["permissions"] = [
        value
        for value in payload["data"]["permissions"]
        if value not in {"sys.region.create", "sys.region.update", "sys.region.delete"}
    ]
    route.fulfill(response=response, json=payload)


def main() -> None:
    OUTPUT.mkdir(parents=True, exist_ok=True)
    suffix = uuid.uuid4().hex[:10]
    city_code = f"e2e-city-{suffix}"
    county_code = f"e2e-county-{suffix}"
    evidence: dict[str, object] = {}
    errors: list[str] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(
            executable_path=EXECUTABLE if EXECUTABLE else None,
        )
        context = browser.new_context(
            viewport={"width": 1440, "height": 900}, locale="zh-CN"
        )
        page = context.new_page()
        page.on(
            "console",
            lambda message: errors.append(message.text)
            if message.type == "error"
            else None,
        )
        login(page)
        select_language(page, "English")
        open_regions(page, "System", "System Settings", "Regions")
        page.get_by_role("heading", name="Regions", exact=True).wait_for()
        shot(page, "regions-manage.en-US.1440", evidence)

        first_root = page.get_by_role("row").nth(1)
        first_root.get_by_role("button", name="Delete", exact=True).click()
        page.get_by_text(
            "This region still has children. Handle the child regions first.",
            exact=True,
        ).wait_for()
        evidence["parent_delete_guard"] = True

        first_root.get_by_role("button", name="Add child", exact=True).click()
        page.locator(".ant-drawer-title").get_by_text(
            "Add child region", exact=True
        ).wait_for()
        page.get_by_label("Region code", exact=True).fill(city_code)
        page.get_by_label("Name", exact=True).fill("E2E City")
        assert page.get_by_label("Full name", exact=True).input_value().endswith(
            " / E2E City"
        )
        page.get_by_role("button", name="Save", exact=True).click()
        page.get_by_text("Child region added.", exact=True).wait_for()
        evidence["city_created"] = True

        first_root.get_by_role("button", name="Expand region", exact=True).click()
        city_row = page.get_by_role("row").filter(has_text="E2E City")
        city_row.get_by_role("button", name="Add child", exact=True).click()
        page.get_by_label("Region code", exact=True).fill(county_code)
        page.get_by_label("Name", exact=True).fill("E2E County")
        page.get_by_role("button", name="Save", exact=True).click()
        page.get_by_text("Child region added.", exact=True).wait_for()
        evidence["county_created"] = True

        city_row = page.get_by_role("row").filter(has_text="E2E City")
        city_row.get_by_role("button", name="Expand region", exact=True).click()
        county_row = page.get_by_role("row").filter(has_text="E2E County")
        county_row.get_by_role("button", name="Edit", exact=True).click()
        edit_drawer = page.get_by_role("dialog", name="Edit region", exact=True)
        edit_drawer.wait_for()
        assert edit_drawer.get_by_label("Region code", exact=True).is_disabled()
        assert edit_drawer.get_by_label("Parent code", exact=True).is_disabled()
        assert edit_drawer.get_by_label("Level", exact=True).is_disabled()
        shot(page, "regions-edit-drawer.en-US.1440", evidence)
        edit_drawer.get_by_label("Name", exact=True).fill("E2E County Updated")
        edit_drawer.get_by_label("Full name", exact=True).fill(
            "E2E Province / E2E City / E2E County Updated"
        )
        edit_drawer.get_by_role("button", name="Save", exact=True).click()
        page.get_by_text("Region updated.", exact=True).wait_for()
        evidence["county_updated"] = True

        remove_region(page, "E2E County Updated")
        page.get_by_text("Region deleted.", exact=True).wait_for()
        remove_region(page, "E2E City")
        page.get_by_text("Region deleted.", exact=True).wait_for()
        evidence["leaf_deletes"] = 2

        select_language(page, "简体中文")
        page.get_by_role("heading", name="地区管理", exact=True).wait_for()
        shot(page, "regions-manage.zh-CN.1440", evidence)
        page.set_viewport_size({"width": 375, "height": 812})
        shot(page, "regions-manage.zh-CN.375", evidence)
        context.close()

        read_only_context = browser.new_context(
            viewport={"width": 1440, "height": 900}, locale="zh-CN"
        )
        read_only_page = read_only_context.new_page()
        read_only_page.route("**/admin-api/v1/auth/context", strip_region_writes)
        login(read_only_page)
        select_language(read_only_page, "English")
        open_regions(
            read_only_page,
            "System",
            "System Settings",
            "Regions",
        )
        read_only_page.get_by_text("Read only", exact=True).wait_for()
        assert read_only_page.get_by_role("button", name="Add child").count() == 0
        assert read_only_page.get_by_role("button", name="Edit").count() == 0
        assert read_only_page.get_by_role("button", name="Delete").count() == 0
        shot(read_only_page, "regions-read-only.en-US.1440", evidence)
        evidence["read_only_actions_hidden"] = True
        read_only_context.close()
        browser.close()

    evidence["unexpected_console_errors"] = errors
    assert not errors, errors
    (OUTPUT / "admin-regions-modules-e2e-results.json").write_text(
        json.dumps(evidence, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )


if __name__ == "__main__":
    main()
