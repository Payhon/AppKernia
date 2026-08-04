from __future__ import annotations

from pathlib import Path
from urllib.parse import urlparse

from playwright.sync_api import Route, sync_playwright


BASE_URL = "http://127.0.0.1:4173"
ROOT = Path(__file__).resolve().parents[3]
OUTPUT = Path(__file__).resolve().parent / "screenshots"
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
REQUEST_ID = "019f0000-0000-7000-8000-000000000001"
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
    "active_tenant": {"id": "t1", "code": "visual", "name": "AppKernia Workspace"},
    "available_tenants": [
        {"id": "t1", "code": "visual", "name": "AppKernia Workspace", "status": "active"}
    ],
    "roles": ["platform_admin"],
    "permissions": [
        "iam.user.read",
        "audit.operation.read",
        "audit.security.read",
        "jobs.run.read",
    ],
    "menus": [
        menu("1", None, "dashboard", "menu.dashboard", "Dashboard", "page", "/dashboard", "dashboard", "DashboardOutlined", 10),
        menu("2", None, "system", "menu.system", "系统", "directory", "/system", None, "SettingOutlined", 20),
        menu("3", "2", "system.users", "menu.system.users", "用户管理", "directory", "/system/users", None, "TeamOutlined", 20),
        menu("4", "3", "system.users.accounts", "menu.system.users.accounts", "用户", "page", "/system/users/accounts", "system.users.accounts", "UserOutlined", 10),
    ],
    "feature_flags": {"multi_tenant": False},
    "menu_revision": 1,
    "permission_revision": 1,
    "server_time": NOW,
}

WINDOW = {"range": "30d", "start_at": "2026-07-06T00:00:00Z", "end_at": NOW}
SUMMARY = {
    **WINDOW,
    "metrics": [
        {"key": "users.total", "value": 12840},
        {"key": "users.new", "value": 286},
        {"key": "sessions.active", "value": 942},
        {"key": "jobs.failed", "value": 7},
        {"key": "security.open", "value": 3},
        {"key": "messages.published", "value": 124},
    ],
}
DAYS = [f"2026-08-0{day}" for day in range(1, 8)]
TRENDS = {
    **WINDOW,
    "series": [
        {
            "key": key,
            "points": [{"day": day, "value": value} for day, value in zip(DAYS, values)],
        }
        for key, values in {
            "logins.success": [382, 441, 406, 518, 492, 564, 602],
            "logins.failure": [12, 18, 9, 21, 15, 11, 8],
            "users.new": [22, 31, 28, 46, 40, 51, 68],
            "jobs.failed": [1, 0, 2, 0, 3, 1, 0],
            "security.events": [0, 1, 0, 2, 1, 0, 1],
        }.items()
    ],
}
ACTIVITY = {
    **WINDOW,
    "operations": [
        {
            "id": "o1",
            "module_code": "iam",
            "action_name": "更新角色授权",
            "resource_type": "role",
            "succeeded": True,
            "error_code": "",
            "occurred_at": NOW,
        }
    ],
    "failed_jobs": [
        {
            "id": "j1",
            "schedule_code": "sync",
            "schedule_name": "数据同步检查",
            "error_code": "JOB.TIMEOUT",
            "occurred_at": "2026-08-04T09:30:00Z",
        }
    ],
    "security_events": [
        {
            "id": "s1",
            "event_type": "异常登录速率",
            "severity": "medium",
            "source": "Admin Gateway",
            "occurred_at": "2026-08-04T08:45:00Z",
        }
    ],
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
        route.fulfill(json=envelope(SUMMARY), headers=headers)
    elif path.endswith("/dashboard/trends"):
        route.fulfill(json=envelope(TRENDS), headers=headers)
    elif path.endswith("/dashboard/activity"):
        route.fulfill(json=envelope(ACTIVITY), headers=headers)
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
        page.get_by_role("heading", name="关键指标").wait_for()
        page.get_by_role("img", name="Dashboard 每日趋势折线图").wait_for()
        page.add_script_tag(path=str(AXE))
        audit = page.evaluate("async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })")
        severe = [item for item in audit["violations"] if item["impact"] in ("critical", "serious")]
        if severe:
            raise AssertionError(severe)
        dimensions = page.evaluate("() => ({scrollWidth: document.documentElement.scrollWidth, clientWidth: document.documentElement.clientWidth})")
        if dimensions["scrollWidth"] > dimensions["clientWidth"]:
            raise AssertionError(dimensions)
        page.screenshot(path=OUTPUT / "dashboard-1440x900-light.png", full_page=True)
        page.set_viewport_size({"width": 768, "height": 1024})
        page.get_by_role("button", name="打开导航").click()
        page.get_by_role("dialog").wait_for()
        page.wait_for_timeout(500)
        page.screenshot(path=OUTPUT / "dashboard-768x1024-navigation.png", full_page=False)
        print(f"dashboard mock-contract visual: axe violations={len(audit['violations'])}, critical_or_serious={len(severe)}")
        context.close()
        browser.close()


if __name__ == "__main__":
    main()
