from __future__ import annotations

import json
import os
import uuid
from pathlib import Path

from playwright.sync_api import Page, sync_playwright

from e2e_navigation_helpers import open_system_page


ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "output" / "playwright"
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
BASE_URL = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174").rstrip("/")
EMAIL = os.environ["AK_E2E_EMAIL"]
PASSWORD = os.environ["AK_E2E_PASSWORD"]


def run_axe(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.wait_for_timeout(500)
    if not page.evaluate("() => Boolean(window.axe)"):
        page.add_script_tag(path=str(AXE))
    result = page.evaluate(
        "async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })"
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
    severe = [
        item for item in violations if item["impact"] in ("critical", "serious")
    ]
    evidence[name] = {"violations": violations, "critical_or_serious": len(severe)}
    contrast_styles = page.evaluate(
        """() => [...document.querySelectorAll('.ant-menu-submenu-title, .ant-menu-item-selected a, .ant-switch-inner-unchecked')].map((element) => {
            const style = getComputedStyle(element)
            return { text: element.textContent?.trim(), color: style.color, background: style.backgroundColor, parentBackground: getComputedStyle(element.parentElement).backgroundColor }
        })"""
    )
    assert not severe, {"severe": severe, "styles": contrast_styles}


def assert_no_overflow(page: Page) -> None:
    dimensions = page.evaluate(
        "() => ({ client: document.documentElement.clientWidth, scroll: document.documentElement.scrollWidth })"
    )
    assert dimensions["scroll"] <= dimensions["client"], dimensions


def switch_header_locale(page: Page, button: str, option: str) -> None:
    page.get_by_role("button", name=button, exact=True).click()
    page.get_by_role("menuitem", name=option, exact=True).click()


def open_configs_from_sidebar(page: Page) -> None:
    settings_label = (
        "System Settings"
        if page.locator("html").get_attribute("lang") == "en-US"
        else "系统设置"
    )
    open_system_page(page, "/system/settings/configs", settings_label)
    page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/system/settings/configs"))


def main() -> None:
    OUTPUT.mkdir(parents=True, exist_ok=True)
    evidence: dict[str, object] = {}
    console_errors: list[str] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(
            locale="zh-CN", viewport={"width": 1440, "height": 900}
        )
        page = context.new_page()
        page.on(
            "console",
            lambda message: console_errors.append(message.text)
            if message.type == "error"
            else None,
        )
        page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        account_label = "Account" if page.locator("html").get_attribute("lang") == "en-US" else "账号"
        password_label = "Password" if account_label == "Account" else "密码"
        page.get_by_label(account_label, exact=True).fill(EMAIL)
        page.get_by_label(password_label, exact=True).fill(PASSWORD)
        page.locator('button[type="submit"]').click()
        page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))
        if page.locator("html").get_attribute("lang") != "en-US":
            switch_header_locale(page, "显示语言", "English")

        open_configs_from_sidebar(page)
        page.get_by_role("heading", name="System Configuration", exact=True).wait_for()
        assert page.get_by_role("radio", name="Form mode", exact=True).is_checked()
        assert page.get_by_role("button", name="Save all changes", exact=True).is_disabled()
        run_axe(page, "form.en-US.1440", evidence)

        site_name = page.get_by_label("Site name", exact=True)
        original_name = site_name.input_value()
        test_name = f"{original_name} E2E {uuid.uuid4().hex[:6]}"
        site_name.fill(test_name)
        assert page.get_by_text("1 changed", exact=True).is_visible()
        page.locator(".ant-segmented-item").filter(has_text="Table mode").click()
        discard_dialog = page.get_by_role(
            "dialog", name="Discard unsaved configuration changes?", exact=True
        )
        discard_dialog.wait_for()
        discard_dialog.get_by_role("button", name="Cancel", exact=True).click()

        with page.expect_response(
            lambda response: "/admin-api/v1/configs/" in response.url
            and response.request.method == "PATCH"
        ) as save_response:
            page.get_by_role("button", name="Save all changes", exact=True).click()
        assert save_response.value.status == 200, save_response.value.text()
        page.get_by_text("Saved 1 settings", exact=True).wait_for()
        assert site_name.input_value() == test_name

        site_name.fill(original_name)
        with page.expect_response(
            lambda response: "/admin-api/v1/configs/" in response.url
            and response.request.method == "PATCH"
        ) as restore_response:
            page.get_by_role("button", name="Save all changes", exact=True).click()
        assert restore_response.value.status == 200, restore_response.value.text()
        page.get_by_text("Saved 1 settings", exact=True).wait_for()
        assert site_name.input_value() == original_name

        page.locator(".ant-segmented-item").filter(has_text="Table mode").click()
        page.wait_for_url(lambda url: "mode=table" in url)
        page.get_by_role("table").wait_for()
        run_axe(page, "table.en-US.1440", evidence)

        switch_header_locale(page, "Display language", "简体中文")
        page.get_by_role("heading", name="系统配置", exact=True).wait_for()
        page.locator(".ant-segmented-item").filter(has_text="表单模式").click()
        page.wait_for_url(lambda url: "mode=table" not in url)
        run_axe(page, "form.zh-CN.1440", evidence)
        page.screenshot(
            path=OUTPUT / "admin-config-form.zh-CN.1440.png", full_page=True
        )

        logo_select = (
            page.locator(".ak-config-direct-field")
            .filter(has_text="site.logo_file_id")
            .get_by_role("button")
        )
        assert logo_select.count() == 1
        logo_select.click()
        picker = page.get_by_role("dialog", name="选择文件", exact=True)
        picker.get_by_role("button", name="上传文件", exact=True).wait_for()
        page.keyboard.press("Escape")
        picker.wait_for(state="hidden")

        page.set_viewport_size({"width": 375, "height": 812})
        assert_no_overflow(page)
        run_axe(page, "form.zh-CN.375", evidence)
        page.screenshot(
            path=OUTPUT / "admin-config-form.zh-CN.375.png", full_page=True
        )

        unexpected = [
            message for message in console_errors if "401 (Unauthorized)" not in message
        ]
        evidence["workflow"] = {
            "default_form_mode": True,
            "dirty_navigation_guard": True,
            "unified_save_status": save_response.value.status,
            "restored_original_status": restore_response.value.status,
            "table_mode_preserved": True,
            "file_picker_and_upload_entry": True,
            "mobile_no_overflow": True,
            "unexpected_console_errors": unexpected,
        }
        assert not unexpected, unexpected
        (OUTPUT / "admin-config-form-mode-e2e-results.json").write_text(
            json.dumps(evidence, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
        context.close()
        browser.close()
    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
