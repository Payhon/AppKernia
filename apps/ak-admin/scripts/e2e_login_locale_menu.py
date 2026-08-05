from __future__ import annotations

import json
import os
from pathlib import Path

from playwright.sync_api import Page, sync_playwright


ROOT = Path(__file__).resolve().parents[3]
BASE_URL = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174").rstrip("/")
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
SCREENSHOTS = (
    ROOT
    / "artifacts"
    / "ui-ux-pro-max"
    / "AKADM-login-locale-icon-menu"
    / "screenshots"
)
RESULTS = ROOT / "output" / "playwright" / "login-locale-icon-menu-e2e-results.json"


def assert_no_overflow(page: Page) -> None:
    dimensions = page.evaluate(
        "() => ({ client: document.documentElement.clientWidth, scroll: document.documentElement.scrollWidth })"
    )
    assert dimensions["scroll"] <= dimensions["client"], dimensions


def run_axe(page: Page) -> list[dict[str, object]]:
    if not page.evaluate("() => Boolean(window.axe)"):
        page.add_script_tag(path=str(AXE))
    result = page.evaluate(
        "async () => await axe.run(document.body, { resultTypes: ['violations'] })"
    )
    violations = [
        {
            "id": item["id"],
            "impact": item["impact"],
            "nodes": len(item["nodes"]),
            "targets": [node["target"] for node in item["nodes"]],
            "html": [node["html"] for node in item["nodes"]],
            "details": [node["any"] for node in item["nodes"]],
        }
        for item in result["violations"]
    ]
    popup = page.get_by_role("menu").evaluate(
        "(element) => element.parentElement?.outerHTML"
    )
    assert not violations, {"violations": violations, "popup": popup}
    return violations


def open_language_menu(page: Page, name: str) -> None:
    trigger = page.get_by_role("button", name=name, exact=True)
    trigger.focus()
    page.keyboard.press("Enter")
    page.get_by_role("menu").wait_for()


def main() -> None:
    SCREENSHOTS.mkdir(parents=True, exist_ok=True)
    RESULTS.parent.mkdir(parents=True, exist_ok=True)
    console_errors: list[str] = []
    evidence: dict[str, object] = {}
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)
        context = browser.new_context(viewport={"width": 1440, "height": 900})
        context.add_init_script(
            "window.localStorage.setItem('ak.admin.locale', 'zh-CN')"
        )
        page = context.new_page()
        page.on(
            "console",
            lambda message: console_errors.append(message.text)
            if message.type == "error"
            else None,
        )
        page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        page.get_by_role("heading", name="登录 AppKernia", exact=True).wait_for()
        assert page.locator(".ak-auth-toolbar select").count() == 0
        trigger = page.get_by_role("button", name="显示语言", exact=True)
        assert trigger.locator(".anticon-translation").count() == 1
        initial_time_origin = page.evaluate("performance.timeOrigin")

        open_language_menu(page, "显示语言")
        selected_zh = page.get_by_role("menuitem", name="简体中文", exact=True)
        assert "ant-dropdown-menu-item-selected" in (selected_zh.get_attribute("class") or "")
        assert_no_overflow(page)
        page.wait_for_timeout(300)
        page.screenshot(path=SCREENSHOTS / "zh-CN-menu-1440.png", full_page=True)
        evidence["zh-CN.1440.axe"] = run_axe(page)

        english = page.get_by_role("menuitem", name="English", exact=True)
        english.focus()
        page.keyboard.press("Enter")
        page.wait_for_function("document.documentElement.lang === 'en-US'")
        assert page.evaluate("performance.timeOrigin") == initial_time_origin
        page.get_by_role("heading", name="Sign in to AppKernia", exact=True).wait_for()

        viewport_results: dict[str, object] = {}
        for width, height in ((1024, 768), (768, 900), (375, 812)):
            page.set_viewport_size({"width": width, "height": height})
            assert_no_overflow(page)
            viewport_results[str(width)] = {"no_horizontal_overflow": True}

        open_language_menu(page, "Display language")
        selected_en = page.get_by_role("menuitem", name="English", exact=True)
        assert "ant-dropdown-menu-item-selected" in (selected_en.get_attribute("class") or "")
        page.wait_for_timeout(300)
        page.screenshot(path=SCREENSHOTS / "en-US-menu-375.png", full_page=True)
        evidence["en-US.375.axe"] = run_axe(page)

        evidence["workflow"] = {
            "native_select_removed": True,
            "translation_icon_present": True,
            "keyboard_open_and_select": True,
            "selected_state_zh_CN": True,
            "selected_state_en_US": True,
            "locale_changed_without_reload": True,
            "viewports": {"1440": {"no_horizontal_overflow": True}, **viewport_results},
            "unexpected_console_errors": console_errors,
        }
        assert not console_errors, console_errors
        browser.close()

    RESULTS.write_text(
        json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
