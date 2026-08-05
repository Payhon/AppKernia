from __future__ import annotations

import json
import os
from pathlib import Path

from playwright.sync_api import Page, Route, sync_playwright


ROOT = Path(__file__).resolve().parents[3]
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4173").rstrip("/")
AXE = ROOT / "apps/ak-admin/node_modules/axe-core/axe.min.js"
OUT = ROOT / "apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-mobile-releases/screenshots"
OUT.mkdir(parents=True, exist_ok=True)

PERMISSIONS = ["mobile.release.read", "mobile.release.create", "mobile.release.update"]
RELEASES = [
    {
        "id": "123e4567-e89b-12d3-a456-426614174000",
        "platform": "android",
        "current_version": "2.4.0",
        "minimum_version": "2.1.0",
        "upgrade_url": "https://example.test/android",
        "release_notes": {"zh-CN": "提升启动速度并修复已知问题。", "en-US": "Improves startup performance and fixes known issues."},
        "active": True,
        "lock_version": 3,
        "updated_at": "2026-08-05T02:00:00Z",
    },
    {
        "id": "223e4567-e89b-12d3-a456-426614174000",
        "platform": "ios",
        "current_version": "2.3.1",
        "minimum_version": "2.0.0",
        "upgrade_url": "https://example.test/ios",
        "release_notes": {"zh-CN": "优化通知体验。", "en-US": "Improves notification delivery."},
        "active": True,
        "lock_version": 2,
        "updated_at": "2026-08-04T02:00:00Z",
    },
    {
        "id": "323e4567-e89b-12d3-a456-426614174000",
        "platform": "harmony",
        "current_version": "1.8.0",
        "minimum_version": "1.6.0",
        "upgrade_url": None,
        "release_notes": {"zh-CN": "下一版本准备中。", "en-US": "The next release is being prepared."},
        "active": False,
        "lock_version": 1,
        "updated_at": "2026-08-03T02:00:00Z",
    },
]


def success(route: Route, data: object, status: int = 200) -> None:
    route.fulfill(status=status, content_type="application/json", body=json.dumps({"code": "OK", "message": "OK", "data": data, "request_id": "mobile-release-e2e"}))


def conflict(route: Route) -> None:
    route.fulfill(status=409, content_type="application/json", body=json.dumps({"error": {"code": "MOBILE.RELEASE.CONFLICT", "message": "conflict", "message_key": "errors.common.conflict", "details": {}}, "request_id": "mobile-release-e2e"}))


def route_handler(route: Route) -> None:
    url = route.request.url
    method = route.request.method
    if url.endswith("/auth/public-config"):
        return success(route, {"locale": "zh-CN", "default_locale": "zh-CN", "supported_locales": ["zh-CN", "en-US"], "feature_flags": {}, "settings": {}})
    if url.endswith("/auth/login"):
        return success(route, {"access_token": "e2e", "token_type": "Bearer", "expires_in": 900, "csrf_token": "csrf-e2e-value-long-enough"})
    if url.endswith("/auth/context"):
        return success(route, {"user": {"id": "423e4567-e89b-12d3-a456-426614174000", "email": "e2e@app.test", "display_name": "E2E", "locale": "zh-CN", "time_zone": "UTC", "avatar_url": None}, "active_tenant": {"id": "523e4567-e89b-12d3-a456-426614174000", "name": "E2E", "code": "e2e"}, "available_tenants": [], "roles": [], "permissions": PERMISSIONS, "menus": [], "feature_flags": {}, "menu_revision": 1, "permission_revision": 1, "server_time": "2026-08-05T02:00:00Z"})
    if url.endswith("/me") and method == "PATCH":
        return success(route, {"id": "423e4567-e89b-12d3-a456-426614174000", "email": "e2e@app.test", "display_name": "E2E", "locale": "en-US", "time_zone": "UTC", "avatar_url": None})
    if url.endswith("/mobile/releases") and method == "GET":
        return success(route, RELEASES)
    if "/mobile/releases/" in url and method == "PATCH":
        body = route.request.post_data_json
        assert body["lock_version"] == 3, body
        return conflict(route)
    if url.endswith("/mobile/releases") and method == "POST":
        return success(route, RELEASES[0])
    return success(route, {})


def run_axe(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate("async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })")
    severe = [item for item in result["violations"] if item["impact"] in ("critical", "serious")]
    evidence[name] = {"serious_critical": len(severe), "violations": [{"id": item["id"], "impact": item["impact"], "nodes": len(item["nodes"])} for item in result["violations"]]}
    assert not severe, severe


def assert_no_page_overflow(page: Page) -> None:
    dimensions = page.evaluate("() => ({ scrollWidth: document.documentElement.scrollWidth, clientWidth: document.documentElement.clientWidth })")
    assert dimensions["scrollWidth"] <= dimensions["clientWidth"], dimensions


def main() -> None:
    print("mobile-release-e2e:start", flush=True)
    evidence: dict[str, object] = {}
    console_errors: list[str] = []
    expected_console_errors: list[str] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
        page = context.new_page()
        page.set_default_timeout(15_000)
        page.route("**/admin-api/**", route_handler)
        def collect_console(message: object) -> None:
            if getattr(message, "type", None) != "error":
                return
            text = getattr(message, "text", "")
            if text == "Failed to load resource: the server responded with a status of 409 (Conflict)":
                expected_console_errors.append(text)
                return
            console_errors.append(text)

        page.on("console", collect_console)

        print("mobile-release-e2e:login", flush=True)
        page.goto(f"{BASE}/login", wait_until="domcontentloaded")
        page.get_by_label("账号").fill("e2e@app.test")
        page.get_by_label("密码").fill("password")
        page.locator('button[type="submit"]').click()
        page.wait_for_url("**/dashboard**")

        print("mobile-release-e2e:list-zh", flush=True)
        page.evaluate("window.history.pushState({}, '', '/system/mobile/releases'); window.dispatchEvent(new PopStateEvent('popstate'))")
        page.get_by_role("heading", name="移动端发布策略").wait_for()
        page.get_by_text("2.4.0", exact=True).wait_for()
        page.get_by_role("combobox", name="平台").click()
        page.locator(".ant-select-dropdown:visible .ant-select-item-option").filter(has_text="Android").click()
        page.wait_for_url("**/system/mobile/releases?platform=android")
        page.evaluate("window.history.pushState({}, '', '/404'); window.dispatchEvent(new PopStateEvent('popstate'))")
        page.wait_for_url("**/404")
        page.go_back()
        page.get_by_role("heading", name="移动端发布策略").wait_for()
        page.wait_for_url("**/system/mobile/releases?platform=android")
        page.get_by_role("heading", name="移动端发布策略").click()
        page.wait_for_timeout(200)
        assert page.get_by_text("2.3.1", exact=True).count() == 0
        assert_no_page_overflow(page)
        run_axe(page, "zh-CN.1440", evidence)
        page.screenshot(path=OUT / "1440x900-light.png", full_page=True)

        print("mobile-release-e2e:drawer-conflict", flush=True)
        page.locator("button:has-text('编 辑')").click()
        page.get_by_role("dialog").wait_for()
        page.locator("button:has-text('保 存')").click()
        page.get_by_text("此策略已被其他管理员更新。请关闭编辑器、刷新列表后重新编辑。", exact=True).wait_for()
        run_axe(page, "zh-CN.drawer.1440", evidence)
        page.screenshot(path=OUT / "drawer-zh-CN-1440.png", full_page=True)
        page.keyboard.press("Escape")

        print("mobile-release-e2e:list-en-mobile", flush=True)
        page.get_by_label("显示语言").click()
        page.get_by_role("menuitem", name="English").click()
        page.get_by_role("heading", name="Mobile release policies").wait_for()
        page.get_by_role("heading", name="Mobile release policies").click()
        page.wait_for_timeout(200)
        page.set_viewport_size({"width": 375, "height": 812})
        assert_no_page_overflow(page)
        run_axe(page, "en-US.375", evidence)
        page.screenshot(path=OUT / "375x812-en-US.png", full_page=True)

        assert expected_console_errors == ["Failed to load resource: the server responded with a status of 409 (Conflict)"], expected_console_errors
        assert not console_errors, console_errors
        evidence["console"] = {"unexpected_errors": console_errors, "expected_conflict_errors": len(expected_console_errors)}
        (OUT / "axe-results.json").write_text(json.dumps(evidence, indent=2), encoding="utf-8")
        browser.close()
    print("mobile-release-e2e:done", flush=True)


if __name__ == "__main__":
    main()
