from __future__ import annotations

import json
import os
import uuid
from pathlib import Path

from playwright.sync_api import Page, expect, sync_playwright

from e2e_navigation_helpers import open_system_page


ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "output" / "playwright"
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
BASE_URL = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174").rstrip("/")
EMAIL = os.environ["AK_E2E_EMAIL"]
PASSWORD = os.environ["AK_E2E_PASSWORD"]
GLOBAL_DICTIONARY = os.environ["AK_E2E_GLOBAL_DICTIONARY"]


def run_axe(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate("async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })")
    violations = result["violations"]
    evidence[name] = {
        "violations": [{"id": item["id"], "impact": item["impact"], "nodes": len(item["nodes"]), "targets": [node["target"] for node in item["nodes"]], "html": [node["html"] for node in item["nodes"]]} for item in violations],
        "critical_or_serious": sum(item["impact"] in ("critical", "serious") for item in violations),
    }
    assert evidence[name]["critical_or_serious"] == 0, violations


def no_overflow(page: Page) -> None:
    dimensions = page.evaluate("() => ({ width: document.documentElement.clientWidth, scroll: document.documentElement.scrollWidth })")
    assert dimensions["scroll"] <= dimensions["width"], dimensions


def select_ant(page: Page, label: str, option: str) -> None:
    page.get_by_label(label, exact=True).click()
    page.locator(".ant-select-dropdown:visible").get_by_role("option", name=option, exact=True).click()


def main() -> None:
    OUTPUT.mkdir(parents=True, exist_ok=True)
    suffix = uuid.uuid4().hex[:10]
    setting_key = f"e2e.display.{suffix}"
    secret_key = f"e2e.secret.{suffix}"
    dictionary_code = f"e2e.status.{suffix}"
    secret_value = f"one-time-secret-{uuid.uuid4()}"
    rotated_secret = f"rotated-secret-{uuid.uuid4()}"
    evidence: dict[str, object] = {}
    console_errors: list[str] = []

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
        page = context.new_page()
        page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
        page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        page.get_by_label("账号", exact=True).fill(EMAIL)
        page.get_by_label("密码", exact=True).fill(PASSWORD)
        page.locator('button[type="submit"]').click()
        page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))
        page.get_by_label("显示语言", exact=True).select_option(label="English")

        open_system_page(page, "/system/settings/configs", "System Settings")
        page.wait_for_url(f"{BASE_URL}/system/settings/configs")
        page.get_by_role("heading", name="System Configuration", exact=True).wait_for()
        run_axe(page, "configs.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-configs.en-US.1440.png", full_page=True)

        page.get_by_role("button", name="Create setting", exact=True).click()
        config_drawer = page.get_by_role("dialog", name="Create setting", exact=True)
        config_drawer.get_by_label("Module code", exact=True).fill("core")
        config_drawer.get_by_label("Configuration group", exact=True).fill("e2e")
        config_drawer.get_by_label("Configuration key", exact=True).fill(setting_key)
        config_drawer.get_by_label("Display name", exact=True).fill(f"E2E Display {suffix}")
        config_drawer.get_by_label("Current value", exact=True).fill("alpha")
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/configs") and response.request.method == "POST") as create_config:
            config_drawer.get_by_role("button", name="Create", exact=True).click()
        assert create_config.value.status == 201, create_config.value.text()
        page.get_by_text("Setting created", exact=True).wait_for()
        page.get_by_label("Search names or keys", exact=True).fill(setting_key)
        page.wait_for_url(lambda url: "q=e2e.display" in url)
        page.get_by_text(setting_key, exact=False).wait_for()

        page.get_by_role("button", name="Create setting", exact=True).click()
        secret_drawer = page.get_by_role("dialog", name="Create setting", exact=True)
        secret_drawer.get_by_label("Module code", exact=True).fill("core")
        secret_drawer.get_by_label("Configuration group", exact=True).fill("e2e")
        secret_drawer.get_by_label("Configuration key", exact=True).fill(secret_key)
        secret_drawer.get_by_label("Display name", exact=True).fill(f"E2E Secret {suffix}")
        secret_drawer.get_by_role("checkbox", name="Secret setting", exact=True).check()
        secret_drawer.get_by_label("New secret", exact=True).fill(secret_value)
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/configs") and response.request.method == "POST") as create_secret:
            secret_drawer.get_by_role("button", name="Create", exact=True).click()
        secret_body = create_secret.value.json()
        assert create_secret.value.status == 201, create_secret.value.text()
        assert secret_body["data"]["secret_configured"] is True
        assert secret_body["data"]["value"] is None and "secret_value" not in secret_body["data"]
        assert secret_value not in create_secret.value.text()
        page.get_by_label("Search names or keys", exact=True).fill(secret_key)
        page.get_by_text(secret_key, exact=False).wait_for()
        assert page.get_by_text(secret_value, exact=True).count() == 0
        page.get_by_role("button", name="Replace secret", exact=True).click()
        rotate_dialog = page.get_by_role("dialog", name="Replace secret", exact=True)
        rotate_dialog.get_by_label("New secret", exact=True).fill(rotated_secret)
        with page.expect_response(lambda response: response.url.endswith("/rotate-secret") and response.request.method == "POST") as rotate_secret:
            rotate_dialog.get_by_role("button", name="Replace secret", exact=True).click()
        assert rotate_secret.value.status == 200, rotate_secret.value.text()
        assert rotated_secret not in rotate_secret.value.text()
        page.get_by_text("Secret safely replaced", exact=True).wait_for()
        rotate_dialog.wait_for(state="detached")

        time_origin = page.evaluate("performance.timeOrigin")
        page.get_by_label("Display language", exact=True).select_option(label="简体中文")
        page.get_by_role("heading", name="系统配置", exact=True).wait_for()
        page.get_by_text("密钥已安全替换", exact=True).wait_for()
        assert page.evaluate("performance.timeOrigin") == time_origin
        run_axe(page, "configs.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-configs.zh-CN.1440.png", full_page=True)
        page.set_viewport_size({"width": 375, "height": 812})
        no_overflow(page)
        run_axe(page, "configs.zh-CN.375", evidence)
        page.screenshot(path=OUTPUT / "admin-configs.zh-CN.375.png", full_page=True)

        page.set_viewport_size({"width": 1440, "height": 900})
        page.get_by_label("显示语言", exact=True).select_option(label="English")
        open_system_page(page, "/system/settings/dictionaries", "System Settings")
        page.wait_for_url(f"{BASE_URL}/system/settings/dictionaries")
        page.get_by_role("heading", name="Dictionaries", exact=True).wait_for()
        page.get_by_role("button", name="Create dictionary", exact=True).click()
        type_drawer = page.get_by_role("dialog", name="Create dictionary", exact=True)
        type_drawer.get_by_label("Dictionary code", exact=True).fill(dictionary_code)
        type_drawer.get_by_label("Dictionary name", exact=True).fill(f"E2E Status {suffix}")
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/dict-types") and response.request.method == "POST") as create_type:
            type_drawer.get_by_role("button", name="Create", exact=True).click()
        assert create_type.value.status == 201, create_type.value.text()
        page.get_by_text("Dictionary created", exact=True).wait_for()
        page.get_by_text(f"E2E Status {suffix}", exact=True).first.wait_for()
        page.get_by_role("button", name="Add dictionary item", exact=True).click()
        item_drawer = page.get_by_role("dialog", name="Add dictionary item", exact=True)
        item_drawer.get_by_label("Item value", exact=True).fill("ready")
        item_drawer.get_by_label("Display label", exact=True).fill("Ready")
        item_drawer.get_by_label("Language", exact=True).click()
        page.locator('.ant-select-dropdown:visible .ant-select-item-option[title="English"]').click()
        with page.expect_response(lambda response: "/dict-types/" in response.url and response.url.endswith("/items") and response.request.method == "POST") as create_item_en:
            item_drawer.get_by_role("button", name="Create", exact=True).click()
        assert create_item_en.value.status == 201, create_item_en.value.text()
        page.get_by_text("Dictionary item created", exact=True).wait_for()
        page.get_by_role("button", name="Add dictionary item", exact=True).click()
        zh_item_drawer = page.get_by_role("dialog", name="Add dictionary item", exact=True)
        zh_item_drawer.get_by_label("Item value", exact=True).fill("ready")
        zh_item_drawer.get_by_label("Display label", exact=True).fill("就绪")
        zh_item_drawer.get_by_label("Language", exact=True).click()
        page.locator('.ant-select-dropdown:visible .ant-select-item-option[title="简体中文"]').click()
        with page.expect_response(lambda response: "/dict-types/" in response.url and response.url.endswith("/items") and response.request.method == "POST") as create_item_zh:
            zh_item_drawer.get_by_role("button", name="Create", exact=True).click()
        assert create_item_zh.value.status == 201, create_item_zh.value.text()
        expect(page.get_by_text("Ready", exact=True)).to_be_visible()
        expect(page.get_by_text("就绪", exact=True)).to_be_visible()

        page.get_by_label("Search dictionary names or codes", exact=True).fill(GLOBAL_DICTIONARY)
        page.get_by_text(GLOBAL_DICTIONARY, exact=True).click()
        page.get_by_text("System dictionary locked", exact=True).wait_for()
        assert page.get_by_role("button", name="Add dictionary item", exact=True).count() == 0
        run_axe(page, "dictionaries.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-dictionaries.en-US.1440.png", full_page=True)
        dictionary_time_origin = page.evaluate("performance.timeOrigin")
        page.get_by_label("Display language", exact=True).select_option(label="简体中文")
        page.get_by_role("heading", name="字典管理", exact=True).wait_for()
        page.get_by_text("系统字典已锁定", exact=True).wait_for()
        assert page.evaluate("performance.timeOrigin") == dictionary_time_origin
        run_axe(page, "dictionaries.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-dictionaries.zh-CN.1440.png", full_page=True)
        page.set_viewport_size({"width": 375, "height": 812})
        no_overflow(page)
        run_axe(page, "dictionaries.zh-CN.375", evidence)
        page.screenshot(path=OUTPUT / "admin-dictionaries.zh-CN.375.png", full_page=True)

        anonymous = browser.new_context(locale="zh-CN", viewport={"width": 768, "height": 900})
        anonymous_page = anonymous.new_page()
        anonymous_page.goto(f"{BASE_URL}/system/settings/configs")
        anonymous_page.wait_for_url(lambda url: "/login" in url and "redirect=%2Fsystem%2Fsettings%2Fconfigs" in url)
        anonymous.close()

        evidence["system_settings"] = {
            "config_create": create_config.value.status,
            "secret_create": create_secret.value.status,
            "secret_rotate": rotate_secret.value.status,
            "secret_never_echoed": True,
            "dictionary_type_create": create_type.value.status,
            "dictionary_item_locales": [create_item_en.value.status, create_item_zh.value.status],
            "system_dictionary_locked": True,
            "url_state": True,
            "locale_switch_without_reload": True,
            "anonymous_redirect": True,
        }
        unexpected = [message for message in console_errors if "401 (Unauthorized)" not in message]
        evidence["unexpected_console_errors"] = unexpected
        assert not unexpected, unexpected
        (OUTPUT / "admin-system-settings-e2e-results.json").write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        context.close()
        browser.close()

    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
