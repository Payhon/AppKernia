from __future__ import annotations

from pathlib import Path
from urllib.parse import urlparse

from playwright.sync_api import Route, sync_playwright


BASE_URL = "http://127.0.0.1:4173"
ROOT = Path(__file__).resolve().parents[3]
OUTPUT = Path(__file__).resolve().parent / "screenshots"
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
REQUEST_ID = "019f0000-0000-7000-8000-000000000003"
NOW = "2026-08-04T12:00:00Z"


def envelope(data: object) -> dict[str, object]:
    return {"code": "OK", "message": "ok", "data": data, "request_id": REQUEST_ID}


USER = {
    "id": "u1",
    "email": "visual@appkernia.test",
    "display_name": "Visual Admin",
    "locale": "zh-CN",
    "time_zone": "Asia/Shanghai",
    "avatar_url": None,
}

CONTEXT = {
    "user": USER,
    "active_tenant": {"id": "t1", "code": "visual", "name": "Local Default"},
    "available_tenants": [{"id": "t1", "code": "visual", "name": "Local Default", "status": "active"}],
    "roles": ["platform_admin", "tenant_admin"],
    "permissions": [],
    "menus": [{
        "id": "1", "parent_id": None, "code": "dashboard", "i18n_key": "menu.dashboard",
        "title": "Dashboard", "type": "page", "path": "/dashboard", "component_key": "dashboard",
        "icon": "DashboardOutlined", "affix": True, "sort": 10, "feature_flag": "",
    }],
    "feature_flags": {"multi_tenant": False, "avatar_upload": True},
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
    elif path.endswith("/dashboard/summary"):
        route.fulfill(json=envelope({"range": "30d", "start_at": NOW, "end_at": NOW, "metrics": []}), headers=headers)
    elif path.endswith("/dashboard/trends"):
        route.fulfill(json=envelope({"range": "30d", "start_at": NOW, "end_at": NOW, "series": []}), headers=headers)
    elif path.endswith("/dashboard/activity"):
        route.fulfill(json=envelope({"range": "30d", "start_at": NOW, "end_at": NOW, "operations": [], "failed_jobs": [], "security_events": []}), headers=headers)
    elif path.endswith("/me"):
        locale = "en-US" if route.request.method == "PATCH" else USER["locale"]
        route.fulfill(json=envelope({**USER, "locale": locale}), headers={"Content-Language": locale})
    else:
        route.fulfill(status=404, json={"code": "NOT_FOUND", "message": "not found", "request_id": REQUEST_ID})


def assert_no_horizontal_overflow(page) -> None:
    dimensions = page.evaluate("() => ({scrollWidth: document.documentElement.scrollWidth, clientWidth: document.documentElement.clientWidth})")
    if dimensions["scrollWidth"] > dimensions["clientWidth"]:
        raise AssertionError(dimensions)


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

        header = page.locator(".ak-shell-header")
        header.get_by_role("button", name="进入全屏").wait_for()
        header.get_by_role("button", name="显示语言").wait_for()
        account = header.get_by_role("button", name="打开用户菜单")
        if account.inner_text().strip() != "V":
            raise AssertionError({"avatar_initial": account.inner_text()})
        if header.get_by_text("Visual Admin").count() or header.get_by_text("退出登录").count():
            raise AssertionError("username or sign-out text leaked into the header")

        account.click()
        page.wait_for_timeout(500)
        role_line = page.locator(".ak-account-summary-text > span")
        if "platform_admin · tenant_admin" not in role_line.inner_text():
            raise AssertionError({"role_line": role_line.inner_text()})
        page.get_by_role("menuitem", name="个人中心").wait_for()
        page.get_by_role("menuitem", name="退出登录").wait_for()
        page.screenshot(path=OUTPUT / "header-account-1440x900.png", full_page=False)
        page.get_by_role("menuitem", name="个人中心").click()
        page.wait_for_url("**/profile/basic")

        language = header.get_by_role("button", name="显示语言")
        language.click()
        page.locator(".ak-language-dropdown").wait_for(state="visible")
        page.wait_for_timeout(200)
        selected = page.locator(".ak-language-dropdown .ant-dropdown-menu-item-selected")
        if selected.inner_text().strip() != "简体中文":
            raise AssertionError({"selected_locale": selected.inner_text()})
        page.screenshot(path=OUTPUT / "header-language-1440x900.png", full_page=False)
        page.get_by_role("menuitem", name="English").click()
        page.wait_for_function("document.documentElement.lang === 'en-US'")
        page.locator(".ak-language-dropdown").wait_for(state="hidden")
        header.get_by_role("button", name="Display language").wait_for()

        fullscreen = header.get_by_role("button", name="Enter fullscreen")
        if fullscreen.is_disabled():
            raise AssertionError("Fullscreen API is unexpectedly unavailable")
        fullscreen.click()
        page.wait_for_function("Boolean(document.fullscreenElement)")
        header.get_by_role("button", name="Exit fullscreen").wait_for()
        page.screenshot(path=OUTPUT / "header-fullscreen-1440x900.png", full_page=False)
        header.get_by_role("button", name="Exit fullscreen").click()
        page.wait_for_function("!document.fullscreenElement")

        page.set_viewport_size({"width": 375, "height": 812})
        header.get_by_role("button", name="Open account menu").click()
        page.wait_for_timeout(500)
        if "platform_admin · tenant_admin" not in role_line.inner_text():
            raise AssertionError({"mobile_role_line": role_line.inner_text()})
        assert_no_horizontal_overflow(page)
        page.screenshot(path=OUTPUT / "header-account-375x812.png", full_page=False)

        page.add_script_tag(path=str(AXE))
        audit = page.evaluate("async () => await axe.run(document, { resultTypes: ['violations'] })")
        severe = [item for item in audit["violations"] if item["impact"] in ("critical", "serious")]
        if severe:
            raise AssertionError(severe)

        violation_summary = [f"{item['id']}:{item['impact']}" for item in audit["violations"]]
        print(f"header utilities visual: axe={violation_summary}, critical_or_serious={len(severe)}")
        context.close()
        browser.close()


if __name__ == "__main__":
    main()
