from __future__ import annotations

import json
import os
import subprocess
from pathlib import Path

from playwright.sync_api import Page, sync_playwright


ROOT = Path(__file__).resolve().parents[3]
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174").rstrip("/")
EMAIL = os.environ["AK_E2E_EMAIL"]
PASSWORD = os.environ["AK_E2E_PASSWORD"]
POSTGRES = os.environ.get("AK_E2E_POSTGRES_CONTAINER", "appkernia-postgres-1")
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
ARTIFACTS = ROOT / "apps" / "ak-admin" / "artifacts" / "ui-ux-pro-max" / "dictionary-category-editing"
SCREENSHOTS = ARTIFACTS / "screenshots"
RESULTS = ROOT / "output" / "playwright" / "dictionary-category-editing-e2e-results.json"
APPEARANCE_ARTIFACTS = ROOT / "apps" / "ak-admin" / "artifacts" / "ui-ux-pro-max" / "dictionary-appearance-selects"
APPEARANCE_SCREENSHOTS = APPEARANCE_ARTIFACTS / "screenshots"
APPEARANCE_RESULTS = ROOT / "output" / "playwright" / "dictionary-appearance-selects-e2e-results.json"


def sql(statement: str) -> str:
    completed = subprocess.run(
        ["docker", "exec", POSTGRES, "psql", "-U", "appkernia", "-d", "appkernia", "-At", "-v", "ON_ERROR_STOP=1", "-c", statement],
        check=True,
        capture_output=True,
        text=True,
    )
    return completed.stdout.strip()


def quoted(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def cleanup() -> None:
    tenant_id = sql(
        "SELECT tm.tenant_id FROM iam.tenant_members tm JOIN iam.users u ON u.id=tm.user_id "
        f"WHERE u.email={quoted(EMAIL)} ORDER BY tm.created_at LIMIT 1"
    )
    if tenant_id:
        tenant = quoted(tenant_id.splitlines()[0])
        sql(
            "BEGIN; "
            f"DELETE FROM sys.role_menus WHERE role_id IN (SELECT id FROM iam.roles WHERE tenant_id={tenant}::uuid); "
            f"DELETE FROM iam.role_permissions WHERE role_id IN (SELECT id FROM iam.roles WHERE tenant_id={tenant}::uuid); "
            f"DELETE FROM iam.role_scope_units WHERE role_id IN (SELECT id FROM iam.roles WHERE tenant_id={tenant}::uuid); "
            f"DELETE FROM iam.user_roles WHERE tenant_id={tenant}::uuid; "
            f"DELETE FROM iam.sessions WHERE tenant_id={tenant}::uuid; "
            f"DELETE FROM iam.tenant_members WHERE tenant_id={tenant}::uuid; "
            f"DELETE FROM iam.roles WHERE tenant_id={tenant}::uuid; "
            f"DELETE FROM iam.tenants WHERE id={tenant}::uuid; COMMIT;"
        )
    sql(f"DELETE FROM iam.users WHERE email={quoted(EMAIL)}")


def audit(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate(
        "async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })"
    )
    violations = [
        {
            "id": item["id"],
            "impact": item["impact"],
            "nodes": len(item["nodes"]),
        }
        for item in result["violations"]
    ]
    serious = [item for item in violations if item["impact"] in ("critical", "serious")]
    dimensions = page.evaluate(
        "() => ({client: document.documentElement.clientWidth, scroll: document.documentElement.scrollWidth})"
    )
    evidence[name] = {
        "violations": violations,
        "critical_or_serious": len(serious),
        "viewport": dimensions,
    }
    assert not serious, serious
    assert dimensions["scroll"] <= dimensions["client"], dimensions


def screenshot(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.wait_for_timeout(600)
    audit(page, name, evidence)
    path = SCREENSHOTS / f"{name}.png"
    page.screenshot(path=path, full_page=True)
    evidence[f"{name}.screenshot"] = str(path.relative_to(ROOT))


def appearance_screenshot(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.wait_for_timeout(600)
    audit(page, name, evidence)
    path = APPEARANCE_SCREENSHOTS / f"{name}.png"
    page.screenshot(path=path, full_page=True)
    evidence[f"{name}.screenshot"] = str(path.relative_to(ROOT))


def switch_locale(page: Page, button_name: str, locale_name: str) -> None:
    page.get_by_role("button", name=button_name, exact=True).click()
    page.locator(".ak-language-dropdown:visible").get_by_text(locale_name, exact=True).click()


def navigate(page: Page, path: str, parents: tuple[str, ...]) -> None:
    sider = page.locator(".ak-desktop-sider")
    link = sider.locator(f'a[href="{path}"]')
    for index, parent in enumerate(parents):
        if link.count() > 0:
            break
        next_label = parents[index + 1] if index + 1 < len(parents) else None
        if next_label is None or sider.get_by_text(next_label, exact=True).count() == 0:
            sider.get_by_text(parent, exact=True).click()
    link.click()
    page.wait_for_url(lambda value: value.startswith(f"{BASE}{path}"))


def main() -> None:
    SCREENSHOTS.mkdir(parents=True, exist_ok=True)
    APPEARANCE_SCREENSHOTS.mkdir(parents=True, exist_ok=True)
    RESULTS.parent.mkdir(parents=True, exist_ok=True)
    evidence: dict[str, object] = {}
    console_errors: list[str] = []
    http_errors: list[dict[str, object]] = []
    try:
        with sync_playwright() as playwright:
            browser = playwright.chromium.launch()
            context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
            page = context.new_page()
            page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
            page.on("response", lambda response: http_errors.append({"status": response.status, "url": response.url}) if response.status >= 400 else None)

            page.goto(f"{BASE}/login", wait_until="networkidle")
            page.get_by_label("账号", exact=True).fill(EMAIL)
            page.get_by_label("密码", exact=True).fill(PASSWORD)
            page.locator('button[type="submit"]').click()
            page.wait_for_url(lambda value: value.startswith(f"{BASE}/dashboard"))

            navigate(page, "/system/settings/dictionaries", ("系统", "系统设置"))
            try:
                page.get_by_role("heading", name="字典管理", exact=True).wait_for()
            except Exception as error:
                debug_path = SCREENSHOTS / "debug-dictionary-route.png"
                page.screenshot(path=debug_path, full_page=True)
                raise AssertionError(
                    f"dictionary route did not render: url={page.url} body={page.locator('body').inner_text()[:2500]!r} "
                    f"console_errors={console_errors!r} http_errors={http_errors!r}"
                ) from error
            try:
                page.get_by_text("存储", exact=True).wait_for()
            except Exception as error:
                debug_path = SCREENSHOTS / "debug-dictionary-categories.png"
                page.screenshot(path=debug_path, full_page=True)
                raise AssertionError(
                    f"dictionary categories did not render: url={page.url} body={page.locator('body').inner_text()[:3500]!r} "
                    f"console_errors={console_errors!r} http_errors={http_errors!r}"
                ) from error
            page.get_by_text("短信", exact=True).wait_for()
            page.get_by_text("通知", exact=True).wait_for()
            assert page.get_by_role("columnheader", name="字典键值", exact=True).is_visible()

            page.get_by_text("storage.driver", exact=True).click()
            local_row = page.get_by_role("row").filter(has_text="本地存储").first
            assert local_row.get_by_role("button", name="删除", exact=True).count() == 0
            local_row.get_by_role("button", name="编辑", exact=True).click()
            drawer = page.get_by_role("dialog", name="编辑内置项（租户覆盖）", exact=True)
            drawer.get_by_text("将保存为当前租户覆盖", exact=True).wait_for()
            assert drawer.get_by_label("字典键值", exact=True).is_disabled()
            assert drawer.get_by_label("语言", exact=True).is_disabled()
            drawer.get_by_label("显示标签", exact=True).fill("本地存储（租户）")
            drawer.get_by_label("颜色", exact=True).click()
            color_menu = page.locator(".ant-select-dropdown:visible").last
            assert "默认（不指定）" in color_menu.inner_text(), color_menu.inner_text()
            page.get_by_text("成功绿", exact=True).last.click()
            drawer.get_by_label("样式类", exact=True).click()
            style_menu = page.locator(".ant-select-dropdown:visible").last
            assert "默认（不指定）" in style_menu.inner_text(), style_menu.inner_text()
            appearance_screenshot(page, "zh-CN-dictionary-appearance-presets-1440", evidence)
            page.get_by_text("成功", exact=True).last.click()
            with page.expect_response(
                lambda response: "/admin-api/v1/dict-types/" in response.url
                and response.url.endswith("/items")
                and response.request.method == "POST"
            ) as created:
                drawer.locator('button[type="submit"]').click()
            assert created.value.status == 201, created.value.text()
            create_body = created.value.request.post_data_json
            assert create_body["color"] == "#087a68", create_body
            assert create_body["css_class"] == "ak-dictionary-style-success", create_body
            page.get_by_text("租户覆盖已保存", exact=True).wait_for()
            tenant_row = page.get_by_role("row").filter(has_text="本地存储（租户）").filter(has_text="租户覆盖")
            tenant_row.wait_for()
            assert tenant_row.get_by_role("button", name="删除", exact=True).is_visible()

            tenant_row.get_by_role("button", name="编辑", exact=True).click()
            edit_drawer = page.get_by_role("dialog", name="编辑内置项（租户覆盖）", exact=True)
            assert edit_drawer.get_by_label("字典键值", exact=True).is_disabled()
            assert edit_drawer.get_by_label("语言", exact=True).is_disabled()
            edit_drawer.get_by_label("显示标签", exact=True).fill("本地存储（已编辑）")
            edit_drawer.get_by_label("颜色", exact=True).fill("#123456")
            edit_drawer.get_by_label("颜色", exact=True).press("Enter")
            edit_drawer.get_by_label("样式类", exact=True).fill("tenant-accent")
            edit_drawer.get_by_label("样式类", exact=True).press("Enter")
            appearance_screenshot(page, "zh-CN-dictionary-appearance-custom-1440", evidence)
            with page.expect_response(
                lambda response: "/admin-api/v1/dict-items/" in response.url
                and response.request.method == "PATCH"
            ) as updated:
                edit_drawer.locator('button[type="submit"]').click()
            assert updated.value.status == 200, updated.value.text()
            update_body = updated.value.request.post_data_json
            assert update_body["color"] == "#123456", update_body
            assert update_body["css_class"] == "tenant-accent", update_body
            page.get_by_text("字典项已更新", exact=True).wait_for()
            page.get_by_text("本地存储（已编辑）", exact=True).wait_for()
            screenshot(page, "zh-CN-dictionaries-category-edit-1440", evidence)

            switch_locale(page, "显示语言", "English")
            try:
                page.get_by_role("heading", name="Dictionaries", exact=True).wait_for()
            except Exception as error:
                debug_path = SCREENSHOTS / "debug-locale-switch.png"
                page.screenshot(path=debug_path, full_page=True)
                raise AssertionError(
                    f"English dictionary state did not render: url={page.url} body={page.locator('body').inner_text()[:3000]!r} "
                    f"console_errors={console_errors!r} http_errors={http_errors!r}"
                ) from error
            page.get_by_text("Storage", exact=True).wait_for()
            assert page.get_by_role("columnheader", name="Dictionary key/value", exact=True).is_visible()
            screenshot(page, "en-US-dictionaries-category-edit-1440", evidence)
            english_row = page.get_by_role("row").filter(has_text="本地存储（已编辑）").filter(has_text="Tenant override")
            english_row.get_by_role("button", name="Edit", exact=True).click()
            english_drawer = page.get_by_role("dialog", name="Edit built-in item (tenant override)", exact=True)
            assert english_drawer.get_by_text("#123456", exact=True).count() >= 1
            assert english_drawer.get_by_text("tenant-accent", exact=True).count() >= 1
            appearance_screenshot(page, "en-US-dictionary-appearance-custom-1440", evidence)
            english_drawer.get_by_role("button", name="Close", exact=True).click()

            page.set_viewport_size({"width": 375, "height": 812})
            screenshot(page, "en-US-dictionaries-category-edit-375", evidence)
            mobile_row = page.get_by_role("row").filter(has_text="本地存储（已编辑）").filter(has_text="Tenant override")
            mobile_row.get_by_role("button", name="Edit", exact=True).click()
            page.get_by_role("dialog", name="Edit built-in item (tenant override)", exact=True).wait_for()
            appearance_screenshot(page, "en-US-dictionary-appearance-custom-375", evidence)

            evidence["override_api"] = {
                "create_status": created.value.status,
                "update_status": updated.value.status,
            }
            evidence["appearance_api"] = {
                "preset_create": {
                    "color": create_body["color"],
                    "css_class": create_body["css_class"],
                },
                "custom_update": {
                    "color": update_body["color"],
                    "css_class": update_body["css_class"],
                },
            }
            evidence["console_errors"] = console_errors
            evidence["http_errors"] = http_errors
            assert not console_errors, console_errors
            assert not http_errors, http_errors
            RESULTS.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
            APPEARANCE_RESULTS.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
            context.close()
            browser.close()
    finally:
        cleanup()
    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
