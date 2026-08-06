from __future__ import annotations

import json
import os
from pathlib import Path

from playwright.sync_api import Page, Route, sync_playwright

ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "output/playwright/service-status"
AXE = ROOT / "apps/ak-admin/node_modules/axe-core/axe.min.js"
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174").rstrip("/")
EMAIL = os.environ["AK_E2E_EMAIL"]
PASSWORD = os.environ["AK_E2E_PASSWORD"]
EXPECTED_CODES = {"iam", "org", "sys", "storage", "notify", "jobs", "audit", "ops"}
MOCK_AUTH = os.environ.get("AK_E2E_MOCK_AUTH") == "1"


def ok(data: object) -> dict[str, object]:
    return {"code": "OK", "message": "OK", "data": data, "request_id": "service-status-e2e"}


def install_mock_api(page: Page) -> None:
    catalog = json.loads(
        (ROOT / "blueprint/backend/spec/core-modules.json").read_text(encoding="utf-8")
    )
    module_data = [
        {
            "code": module["code"],
            "name_key": module["name_key"],
            "description_key": module["description_key"],
            "version": "dev",
            "capabilities": module["capabilities"],
            "status": module["status"],
        }
        for module in sorted(catalog["modules"], key=lambda value: value["code"])
    ]
    menu_data = [
        {"id": "dashboard", "parent_id": None, "code": "dashboard", "i18n_key": "menu.dashboard", "title": "Dashboard", "type": "page", "path": "/dashboard", "component_key": "dashboard", "icon": "DashboardOutlined", "affix": True, "sort": 10, "feature_flag": ""},
        {"id": "system", "parent_id": None, "code": "system", "i18n_key": "menu.system", "title": "System", "type": "directory", "path": "/system", "component_key": None, "icon": "SettingOutlined", "affix": False, "sort": 20, "feature_flag": ""},
        {"id": "monitoring", "parent_id": "system", "code": "system.monitoring", "i18n_key": "menu.system.monitoring", "title": "Monitoring", "type": "directory", "path": "/system/monitoring", "component_key": None, "icon": "FundProjectionScreenOutlined", "affix": False, "sort": 80, "feature_flag": ""},
        {"id": "health", "parent_id": "monitoring", "code": "system.monitoring.health", "i18n_key": "menu.system.monitoring.health", "title": "Service Status", "type": "page", "path": "/system/monitoring/health", "component_key": "system.monitoring.health", "icon": "HeartOutlined", "affix": False, "sort": 20, "feature_flag": ""},
    ]
    auth_context = {
        "user": {"id": "e2e-user", "email": EMAIL, "display_name": "Service Status E2E", "locale": "zh-CN", "time_zone": "Asia/Shanghai", "avatar_url": None},
        "active_tenant": {"id": "e2e-tenant", "code": "e2e", "name": "E2E"},
        "available_tenants": [{"id": "e2e-tenant", "code": "e2e", "name": "E2E", "status": "active"}],
        "roles": ["super-admin"],
        "permissions": ["ops.health.read"],
        "menus": menu_data,
        "feature_flags": {},
        "menu_revision": 1,
        "permission_revision": 1,
        "server_time": "2026-08-05T12:00:00Z",
    }
    health = {
        "status": "ready",
        "dependencies": [
            {"code": "api", "status": "ready", "latency_ms": 1, "checked_at": "2026-08-05T12:00:00Z"},
            {"code": "postgresql", "status": "ready", "latency_ms": 3, "checked_at": "2026-08-05T12:00:00Z"},
            {"code": "redis", "status": "not_configured", "latency_ms": 0, "checked_at": "2026-08-05T12:00:00Z"},
            {"code": "object_storage", "status": "ready", "latency_ms": 5, "checked_at": "2026-08-05T12:00:00Z"},
        ],
        "checked_at": "2026-08-05T12:00:00Z",
    }
    runtime = {
        "app_version": "dev",
        "go_version": "go1.26.5",
        "uptime_seconds": 3600,
        "modules": module_data,
        "queue": {"status": "ready", "available": 2, "retryable": 0, "scheduled": 1, "last_heartbeat_at": "2026-08-05T12:00:00Z"},
        "schedule_runs_24h": {"succeeded": 24, "failed": 1},
        "generated_at": "2026-08-05T12:00:00Z",
    }

    def handler(route: Route) -> None:
        url = route.request.url
        method = route.request.method
        if url.endswith("/auth/public-config"):
            route.fulfill(json=ok({"locale": "zh-CN", "default_locale": "zh-CN", "supported_locales": ["zh-CN", "en-US"], "feature_flags": {}, "settings": {}}))
        elif url.endswith("/auth/login"):
            route.fulfill(json=ok({"access_token": "mock-access", "token_type": "Bearer", "expires_in": 900, "csrf_token": "mock-csrf-token-long-enough"}))
        elif url.endswith("/auth/context"):
            route.fulfill(json=ok(auth_context))
        elif url.endswith("/me") and method == "PATCH":
            requested = route.request.post_data_json
            route.fulfill(json=ok({**auth_context["user"], "locale": requested.get("locale", "zh-CN")}))
        elif "/dashboard/summary" in url:
            route.fulfill(json=ok({"range": "30d", "start_at": "2026-07-07T00:00:00Z", "end_at": "2026-08-05T23:59:59Z", "metrics": []}))
        elif "/dashboard/trends" in url:
            route.fulfill(json=ok({"range": "30d", "start_at": "2026-07-07T00:00:00Z", "end_at": "2026-08-05T23:59:59Z", "series": []}))
        elif "/dashboard/activity" in url:
            route.fulfill(json=ok({"range": "30d", "start_at": "2026-07-07T00:00:00Z", "end_at": "2026-08-05T23:59:59Z", "operations": [], "failed_jobs": [], "security_events": []}))
        elif url.endswith("/ops/health"):
            route.fulfill(json=ok(health))
        elif url.endswith("/ops/runtime-summary"):
            route.fulfill(json=ok(runtime))
        else:
            route.fallback()

    page.route("**/admin-api/v1/**", handler)


def login(page: Page) -> None:
    page.goto(f"{BASE}/login", wait_until="networkidle")
    page.get_by_label("账号", exact=True).fill(EMAIL)
    password_input = page.get_by_label("密码", exact=True)
    password_input.fill(PASSWORD)
    assert password_input.input_value() == PASSWORD
    with page.expect_request(
        lambda request: request.url.endswith("/admin-api/v1/auth/login")
    ) as login_request:
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/auth/login")
        ) as login_response:
            page.locator('button[type="submit"]').click()
    request_payload = login_request.value.post_data_json
    assert request_payload["email"] == EMAIL
    assert request_payload["password"] == PASSWORD
    response = login_response.value
    if response.status != 200:
        payload = response.json()
        raise AssertionError(
            f"login failed: status={response.status} payload={payload}"
        )
    page.wait_for_url(lambda value: value.startswith(f"{BASE}/dashboard"))


def switch_language(page: Page, locale: str) -> None:
    current = page.locator("html").get_attribute("lang")
    if current == locale:
        return
    label = "显示语言" if current == "zh-CN" else "Display language"
    option = "English" if locale == "en-US" else "简体中文"
    page.get_by_role("button", name=label, exact=True).click()
    page.get_by_role("menuitem", name=option, exact=True).click()
    page.locator(f'html[lang="{locale}"]').wait_for()


def audit_and_shot(
    page: Page,
    locale: str,
    width: int,
    height: int,
    evidence: dict[str, object],
) -> None:
    page.set_viewport_size({"width": width, "height": height})
    page.wait_for_timeout(300)
    page.add_script_tag(path=str(AXE))
    result = page.evaluate(
        "async()=>await axe.run({exclude:[['.ant-table-measure-row']]},"
        "{resultTypes:['violations']})"
    )
    violations = [
        {"id": item["id"], "impact": item["impact"], "nodes": len(item["nodes"])}
        for item in result["violations"]
    ]
    dimensions = page.evaluate(
        "()=>({client:document.documentElement.clientWidth,"
        "scroll:document.documentElement.scrollWidth})"
    )
    scroll_regions = page.locator(".ak-ops-table-scroll")
    assert scroll_regions.count() == 2
    for index in range(scroll_regions.count()):
        scroll_regions.nth(index).focus()
        assert scroll_regions.nth(index).evaluate("el=>el===document.activeElement")
    assert not violations, violations
    assert dimensions["scroll"] <= dimensions["client"], dimensions
    key = f"{locale}.{width}"
    evidence[key] = {
        "axe_violations": violations,
        "dimensions": dimensions,
        "focusable_scroll_regions": 2,
    }
    page.screenshot(path=OUTPUT / f"service-status.{key}.png", full_page=True)


def main() -> None:
    OUTPUT.mkdir(parents=True, exist_ok=True)
    evidence: dict[str, object] = {}
    console_errors: list[str] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
        page = context.new_page()
        if MOCK_AUTH:
            install_mock_api(page)
        page.on(
            "console",
            lambda message: console_errors.append(message.text)
            if message.type == "error"
            else None,
        )
        login(page)
        assert page.get_by_role("link", name="模块信息", exact=True).count() == 0

        if MOCK_AUTH:
            page.evaluate(
                "()=>{history.pushState({},'', '/system/monitoring/health');"
                "window.dispatchEvent(new PopStateEvent('popstate'))}"
            )
            page.wait_for_url(f"{BASE}/system/monitoring/health")
            runtime = page.evaluate(
                "async()=>await (await fetch('/admin-api/v1/ops/runtime-summary')).json()"
            )["data"]
        else:
            raise AssertionError("live mode requires externally supplied valid credentials")
        modules = runtime["modules"]
        assert len(modules) == 8
        assert {module["code"] for module in modules} == EXPECTED_CODES
        assert all(module["version"] == runtime["app_version"] for module in modules)
        assert all(module["name_key"] and module["description_key"] for module in modules)
        assert all(module["capabilities"] for module in modules)

        for locale in ("zh-CN", "en-US"):
            switch_language(page, locale)
            heading = "服务状态" if locale == "zh-CN" else "Service Status"
            try:
                page.get_by_role("heading", name=heading, exact=True).wait_for()
            except Exception as error:
                raise AssertionError(
                    f"service-status heading missing: url={page.url} "
                    f"lang={page.locator('html').get_attribute('lang')} "
                    f"body={page.locator('body').inner_text()[:1000]} "
                    f"console={console_errors}"
                ) from error
            body = page.locator("body").inner_text()
            for code in EXPECTED_CODES:
                assert code in body
            for width, height in ((375, 812), (768, 1024), (1024, 768), (1440, 900)):
                audit_and_shot(page, locale, width, height, evidence)

        old_api = page.request.get(f"{BASE}/admin-api/v1/modules")
        assert old_api.status == 404, old_api.text()
        evidence["runtime_contract"] = {
            "module_count": len(modules),
            "codes": sorted(EXPECTED_CODES),
            "app_version": runtime["app_version"],
            "all_versions_match": True,
            "old_modules_api_status": old_api.status,
            "browser_data_source": "mocked-contract" if MOCK_AUTH else "live-backend",
        }
        evidence["unexpected_console_errors"] = console_errors
        assert not console_errors, console_errors
        (OUTPUT / "e2e-results.json").write_text(
            json.dumps(evidence, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
        context.close()
        browser.close()

    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
