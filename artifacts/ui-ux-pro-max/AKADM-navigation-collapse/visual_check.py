from __future__ import annotations

from pathlib import Path
from urllib.parse import urlparse

from playwright.sync_api import Route, sync_playwright


BASE_URL = "http://127.0.0.1:4173"
ROOT = Path(__file__).resolve().parents[3]
OUTPUT = Path(__file__).resolve().parent / "screenshots"
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
REQUEST_ID = "019f0000-0000-7000-8000-000000000002"
NOW = "2026-08-04T12:00:00Z"


def envelope(data: object) -> dict[str, object]:
    return {"code": "OK", "message": "ok", "data": data, "request_id": REQUEST_ID}


def menu(
    menu_id: str,
    parent_id: str | None,
    code: str,
    i18n_key: str,
    title: str,
    menu_type: str,
    path: str,
    component_key: str | None,
    icon: str,
    sort: int,
) -> dict[str, object]:
    return {
        "id": menu_id,
        "parent_id": parent_id,
        "code": code,
        "i18n_key": i18n_key,
        "title": title,
        "type": menu_type,
        "path": path,
        "component_key": component_key,
        "icon": icon,
        "affix": code == "dashboard",
        "sort": sort,
        "feature_flag": "",
    }


CONTEXT = {
    "user": {
        "id": "u1",
        "email": "visual@appkernia.test",
        "display_name": "视觉验收",
        "locale": "zh-CN",
        "time_zone": "Asia/Shanghai",
        "avatar_url": None,
    },
    "active_tenant": {"id": "t1", "code": "visual", "name": "Local Default"},
    "available_tenants": [{"id": "t1", "code": "visual", "name": "Local Default", "status": "active"}],
    "roles": ["platform_admin"],
    "permissions": ["sys.dictionary.read"],
    "menus": [
        menu("1", None, "dashboard", "menu.dashboard", "Dashboard", "page", "/dashboard", "dashboard", "DashboardOutlined", 10),
        menu("2", None, "system", "menu.system", "系统", "directory", "/system", None, "SettingOutlined", 20),
        menu("3", "2", "system.settings", "menu.system.settings", "系统设置", "directory", "/system/settings", None, "ControlOutlined", 10),
        menu("4", "3", "system.settings.dictionaries", "menu.system.settings.dictionaries", "字典管理", "page", "/system/settings/dictionaries", "system.settings.dictionaries", "BookOutlined", 20),
    ],
    "feature_flags": {"multi_tenant": False},
    "menu_revision": 1,
    "permission_revision": 1,
    "server_time": NOW,
}


def handle_api(route: Route) -> None:
    path = urlparse(route.request.url).path
    headers = {"Content-Language": "zh-CN"}
    if path.endswith("/auth/public-config"):
        route.fulfill(json=envelope({"feature_flags": {"admin_registration": False, "password_recovery": False}}), headers=headers)
    elif path.endswith("/auth/login"):
        route.fulfill(json=envelope({"access_token": "visual-token", "token_type": "Bearer", "expires_in": 900, "csrf_token": "visual-csrf"}), headers=headers)
    elif path.endswith("/auth/context"):
        route.fulfill(json=envelope(CONTEXT), headers=headers)
    elif path.endswith("/dict-types"):
        route.fulfill(json=envelope({"items": [], "total": 0, "page": 1, "page_size": 20}), headers=headers)
    else:
        route.fulfill(status=404, json={"code": "NOT_FOUND", "message": "not found", "request_id": REQUEST_ID})


def main() -> None:
    OUTPUT.mkdir(parents=True, exist_ok=True)
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900}, color_scheme="light")
        page = context.new_page()
        page.route("**/admin-api/v1/**", handle_api)
        page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        page.get_by_label("账号").fill("visual@appkernia.test")
        page.get_by_label("密码").fill("visual-only")
        page.locator('button[type="submit"]').click()
        page.wait_for_url("**/dashboard**")
        page.get_by_text("系统", exact=True).click()
        page.get_by_text("系统设置", exact=True).click()
        page.get_by_label("主导航").get_by_role("link", name="字典管理").click()
        page.wait_for_url("**/system/settings/dictionaries")
        page.get_by_role("heading", name="字典管理").wait_for()

        ancestor_colors = page.locator(".ak-desktop-sider .ant-menu-submenu-selected > .ant-menu-submenu-title:visible").evaluate_all(
            "els => els.map(el => getComputedStyle(el).color)"
        )
        if len(ancestor_colors) != 2 or any(color != "rgba(255, 255, 255, 0.92)" for color in ancestor_colors):
            raise AssertionError({"ancestor_colors": ancestor_colors})

        handle = page.get_by_role("button", name="收起导航")
        if handle.get_attribute("aria-expanded") != "true":
            raise AssertionError("expanded state is not exposed")
        page.mouse.move(900, 120)
        page.wait_for_timeout(200)
        if handle.evaluate("el => getComputedStyle(el).opacity") != "0":
            raise AssertionError("collapse handle should be hidden before sidebar hover")

        page.locator(".ak-desktop-sider").hover()
        page.wait_for_timeout(200)
        if handle.evaluate("el => getComputedStyle(el).opacity") != "1":
            raise AssertionError("collapse handle did not appear on hover")
        page.screenshot(path=OUTPUT / "dictionaries-expanded-hover-1440x900.png", full_page=True)

        handle.click()
        expand_handle = page.get_by_role("button", name="展开导航")
        if expand_handle.get_attribute("aria-expanded") != "false":
            raise AssertionError("collapsed state is not exposed")
        page.locator(".ak-desktop-sider").hover()
        page.wait_for_timeout(200)
        page.screenshot(path=OUTPUT / "dictionaries-collapsed-hover-1440x900.png", full_page=True)

        expand_handle.click()
        page.get_by_role("button", name="收起导航").wait_for()
        page.mouse.move(900, 120)
        page.wait_for_timeout(500)
        restored_colors = page.locator(".ak-desktop-sider .ant-menu-submenu-selected > .ant-menu-submenu-title:visible").evaluate_all(
            "els => els.map(el => getComputedStyle(el).color)"
        )
        if restored_colors != ancestor_colors:
            raise AssertionError({"restored_colors": restored_colors})

        page.add_script_tag(path=str(AXE))
        audit = page.evaluate("async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })")
        severe = [item for item in audit["violations"] if item["impact"] in ("critical", "serious")]
        if severe:
            raise AssertionError(severe)
        dimensions = page.evaluate("() => ({scrollWidth: document.documentElement.scrollWidth, clientWidth: document.documentElement.clientWidth})")
        if dimensions["scrollWidth"] > dimensions["clientWidth"]:
            raise AssertionError(dimensions)

        print(f"navigation visual: ancestor_colors={ancestor_colors}, axe={len(audit['violations'])}, critical_or_serious={len(severe)}")
        context.close()
        browser.close()


if __name__ == "__main__":
    main()
