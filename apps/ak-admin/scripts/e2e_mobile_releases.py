from __future__ import annotations

import json
import os
import re
from pathlib import Path
from urllib.parse import urlparse

from playwright.sync_api import Page, Route, sync_playwright


ROOT = Path(__file__).resolve().parents[3]
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4173").rstrip("/")
AXE = ROOT / "apps/ak-admin/node_modules/axe-core/axe.min.js"
OUT = ROOT / "output/playwright/app-upgrade-center"
OUT.mkdir(parents=True, exist_ok=True)
STARTUP_OUT = ROOT / "output/playwright/app-startup-experience"
STARTUP_OUT.mkdir(parents=True, exist_ok=True)

TENANT_ID = "523e4567-e89b-12d3-a456-426614174000"
APP_ID = "623e4567-e89b-12d3-a456-426614174000"
USER_ID = "423e4567-e89b-12d3-a456-426614174000"
ICON_FILE_ID = "a23e4567-e89b-12d3-a456-426614174001"
ZH_SLIDE_ONE = "a23e4567-e89b-12d3-a456-426614174002"
EN_SLIDE_ONE = "a23e4567-e89b-12d3-a456-426614174003"
ZH_SLIDE_TWO = "a23e4567-e89b-12d3-a456-426614174004"
EN_SLIDE_TWO = "a23e4567-e89b-12d3-a456-426614174005"
STARTUP_IMAGE = (ROOT / "output/playwright/admin-login.zh-CN.768.png").read_bytes()
PERMISSIONS = [
    "app.application.read",
    "app.application.create",
    "app.application.update",
    "app.application.disable",
    "app.application.delete",
    "app.onboarding.publish",
    "app.user.read",
    "app.content.read",
    "app.content.create",
    "app.content.update",
    "app.content.publish",
    "app.content.delete",
    "mobile.release.read",
    "mobile.release.create",
    "mobile.release.update",
    "mobile.release.publish",
    "mobile.release.delete",
    "storage.file.read",
]
APP = {
    "id": APP_ID,
    "tenant_id": TENANT_ID,
    "appid": "__UNI__APPKERNIA",
    "appid_pending": False,
    "app_type": "uni_app_x",
    "code": "default-app",
    "name": "AppKernia",
    "description": "跨平台应用开发基座",
    "introduction": "Build once and operate safely across supported platforms.",
    "remark": "Production application boundary",
    "status": "active",
    "default_locale": "zh-CN",
    "registration_enabled": True,
    "registration_verification_mode": "email_otp",
    "creator_user_id": USER_ID,
    "owner_type": "tenant",
    "owner_id": TENANT_ID,
    "icon_file_id": ICON_FILE_ID,
    "managers": [USER_ID],
    "members": [],
    "screenshots": [],
    "channels": [
        {"id": "723e4567-e89b-12d3-a456-426614174001", "channel_code": "android", "name": "AppKernia Android", "url": "https://example.test/app.apk", "abm_url": None, "qrcode_file_id": None, "enabled": True},
        {"id": "723e4567-e89b-12d3-a456-426614174002", "channel_code": "h5", "name": "AppKernia Web", "url": "https://example.test", "abm_url": None, "qrcode_file_id": None, "enabled": True},
    ],
    "store_listings": [
        {"id": "823e4567-e89b-12d3-a456-426614174000", "name": "Official", "scheme": "appkernia://", "enabled": True, "priority": 100}
    ],
    "startup": {
        "translations": {
            "zh-CN": {"display_name": "AppKernia", "subtitle": "安全一致的跨端应用基座"},
            "en-US": {"display_name": "AppKernia", "subtitle": "A consistent and secure cross-platform foundation"},
        },
        "onboarding_enabled": True,
        "draft_slides": [
            {"id": "b23e4567-e89b-12d3-a456-426614174001", "position": 0, "assets": {"zh-CN": {"file_id": ZH_SLIDE_ONE, "accessibility_label": "第一张中文启动介绍"}, "en-US": {"file_id": EN_SLIDE_ONE, "accessibility_label": "First English onboarding image"}}},
            {"id": "b23e4567-e89b-12d3-a456-426614174002", "position": 1, "assets": {"zh-CN": {"file_id": ZH_SLIDE_TWO, "accessibility_label": "第二张中文启动介绍"}, "en-US": {"file_id": EN_SLIDE_TWO, "accessibility_label": "Second English onboarding image"}}},
        ],
        "published_version": 3,
        "published_at": "2026-08-24T02:00:00Z",
        "draft_changed": True,
    },
    "is_default": True,
    "lock_version": 3,
    "created_at": "2026-08-05T02:00:00Z",
    "updated_at": "2026-08-10T02:00:00Z",
}

APP_PAGE = {
    "id": "a23e4567-e89b-12d3-a456-426614174000",
    "slug": "about-us",
    "page_type": "about-us",
    "status": "draft",
    "lock_version": 2,
    "current_revision_id": None,
    "updated_at": "2026-08-10T02:00:00Z",
    "translations": {
        "zh-CN": {"title": "关于我们", "body_format": "markdown", "body": "中文正文"},
        "en-US": {"title": "About us", "body_format": "markdown", "body": "English body"},
    },
    "revisions": [],
}


def release(
    release_id: str,
    version: str,
    package_type: str,
    platforms: list[str],
    publish_status: str,
    published_platforms: list[str],
    lock_version: int,
) -> dict[str, object]:
    ever = None if publish_status == "draft" else "2026-08-08T02:00:00Z"
    external = "https://example.test/download/" + version
    return {
        "id": release_id,
        "tenant_id": TENANT_ID,
        "app_id": APP_ID,
        "package_type": package_type,
        "version": version,
        "minimum_native_version": "2.0.0" if package_type == "wgt" else None,
        "package_file_id": None,
        "external_url": external,
        "create_env": "upgrade_center",
        "is_silently": package_type == "wgt",
        "is_mandatory": False,
        "ever_published_at": ever,
        "last_published_at": ever,
        "unpublished_at": "2026-08-09T02:00:00Z" if publish_status == "offline" else None,
        "publish_status": publish_status,
        "platforms": platforms,
        "published_platforms": published_platforms,
        "titles": {"zh-CN": f"版本 {version}", "en-US": f"Version {version}"},
        "contents": {"zh-CN": "提升稳定性并修复已知问题。", "en-US": "Improves stability and fixes known issues."},
        "store_listing_ids": [APP["store_listings"][0]["id"]],
        "lock_version": lock_version,
        "created_at": "2026-08-08T02:00:00Z",
        "updated_at": "2026-08-10T02:00:00Z",
        "platform": platforms[0],
        "current_version": version,
        "minimum_version": "2.0.0" if package_type == "wgt" else "0.0.0",
        "upgrade_url": external,
        "release_notes": {"zh-CN": "提升稳定性并修复已知问题。", "en-US": "Improves stability and fixes known issues."},
        "active": bool(published_platforms),
    }


RELEASES = [
    release("923e4567-e89b-12d3-a456-426614174001", "3.0.0", "wgt", ["android", "ios", "harmony"], "partial", ["android", "ios"], 6),
    release("923e4567-e89b-12d3-a456-426614174002", "2.9.0", "native_app", ["android"], "online", ["android"], 4),
    release("923e4567-e89b-12d3-a456-426614174003", "3.1.0", "native_app", ["ios"], "draft", [], 1),
    release("923e4567-e89b-12d3-a456-426614174004", "2.8.0", "native_app", ["harmony"], "offline", [], 5),
]


def success(route: Route, data: object, status: int = 200) -> None:
    route.fulfill(
        status=status,
        content_type="application/json",
        body=json.dumps({"code": "OK", "message": "OK", "data": data, "request_id": "app-upgrade-e2e"}),
    )


def conflict(route: Route) -> None:
    route.fulfill(
        status=409,
        content_type="application/json",
        body=json.dumps({"error": {"code": "SYS.MOBILE_RELEASE.CONFLICT", "message": "conflict", "message_key": "errors.common.conflict", "details": {}}, "request_id": "app-upgrade-e2e"}),
    )


def route_handler(route: Route) -> None:
    parsed = urlparse(route.request.url)
    path = parsed.path
    method = route.request.method
    if path.endswith("/auth/public-config"):
        return success(route, {"locale": "zh-CN", "default_locale": "zh-CN", "supported_locales": ["zh-CN", "en-US"], "feature_flags": {}, "settings": {}})
    if path.endswith("/auth/login"):
        return success(route, {"access_token": "e2e", "token_type": "Bearer", "expires_in": 900, "csrf_token": "csrf-e2e-value-long-enough"})
    if path.endswith("/auth/context"):
        return success(route, {"user": {"id": USER_ID, "email": "e2e@app.test", "display_name": "E2E", "locale": "zh-CN", "time_zone": "UTC", "avatar_url": None}, "active_tenant": {"id": TENANT_ID, "name": "E2E", "code": "e2e"}, "available_tenants": [], "roles": [], "permissions": PERMISSIONS, "menus": [], "feature_flags": {}, "menu_revision": 1, "permission_revision": 1, "server_time": "2026-08-10T02:00:00Z"})
    if path.endswith("/me") and method == "PATCH":
        return success(route, {"id": USER_ID, "email": "e2e@app.test", "display_name": "E2E", "locale": route.request.post_data_json.get("locale", "zh-CN"), "time_zone": "UTC", "avatar_url": None})
    if path.endswith("/apps") and method == "GET":
        return success(route, {"items": [APP], "total": 1})
    if path.endswith("/apps") and method == "POST":
        return success(route, APP, 201)
    if path.endswith("/apps/batch-delete"):
        return success(route, {"deleted_count": len(route.request.post_data_json["ids"])})
    if path.endswith(f"/apps/{APP_ID}") and method == "GET":
        return success(route, APP)
    if path.endswith(f"/apps/{APP_ID}") and method == "PATCH":
        return success(route, {**APP, "lock_version": APP["lock_version"] + 1})
    if path.endswith(f"/apps/{APP_ID}/startup/onboarding/publish") and method == "POST":
        return success(route, {**APP, "startup": {**APP["startup"], "published_version": 4, "published_at": "2026-08-25T02:00:00Z", "draft_changed": False}})
    if path.endswith(f"/apps/{APP_ID}") and method == "DELETE":
        return success(route, {"deleted": True})
    if path.endswith("/dictionaries/system.language"):
        english = route.request.headers.get("accept-language", "").startswith("en-US")
        return success(
            route,
            {
                "code": "system.language",
                "locale": "en-US" if english else "zh-CN",
                "extension_policy": "fixed",
                "items": [
                    {
                        "value": "zh-CN",
                        "label": "Simplified Chinese" if english else "简体中文",
                        "is_default": True,
                        "extra": {},
                    },
                    {"value": "en-US", "label": "English", "is_default": False, "extra": {}},
                ],
            },
        )
    page_root = f"/apps/{APP_ID}/content/pages"
    if path.endswith(page_root) and method == "GET":
        return success(route, {"items": [APP_PAGE], "page": 1, "page_size": 100, "total": 1})
    if path.endswith(page_root) and method == "POST":
        return success(route, APP_PAGE, 201)
    if path.endswith(page_root + "/about-us") and method == "PATCH":
        return success(route, {**APP_PAGE, "lock_version": 3})
    if path.endswith(page_root + "/about-us/publish"):
        return success(route, {**APP_PAGE, "status": "published", "lock_version": 3})
    release_root = f"/apps/{APP_ID}/mobile/releases"
    if path.endswith(release_root) and method == "GET":
        return success(route, {"items": RELEASES, "page": 1, "page_size": 20, "total": len(RELEASES)})
    if path.endswith(release_root) and method == "POST":
        return success(route, RELEASES[2], 201)
    if path.endswith(release_root + "/batch-delete"):
        return success(route, {"deleted_count": len(route.request.post_data_json["ids"])})
    if release_root + "/" in path and method == "PATCH":
        return conflict(route)
    if release_root + "/" in path and method == "DELETE":
        return success(route, {"deleted": True})
    if path.endswith("/publish") or path.endswith("/unpublish"):
        return success(route, RELEASES[0])
    if "/files/" in path and path.endswith("/content"):
        return route.fulfill(status=200, content_type="image/png", body=STARTUP_IMAGE)
    if "/files" in path:
        return success(route, {"items": [], "page": 1, "page_size": 50, "total": 0})
    return success(route, {})


def run_axe(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate("async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })")
    severe = [item for item in result["violations"] if item["impact"] in ("critical", "serious")]
    evidence[name] = {
        "serious_critical": len(severe),
        "violations": [{"id": item["id"], "impact": item["impact"], "nodes": len(item["nodes"])} for item in result["violations"]],
    }
    assert not severe, severe


def assert_no_page_overflow(page: Page) -> None:
    dimensions = page.evaluate("() => ({ scrollWidth: document.documentElement.scrollWidth, clientWidth: document.documentElement.clientWidth })")
    if dimensions["scrollWidth"] > dimensions["clientWidth"]:
        dimensions["table_scroll"] = page.evaluate("""() => Array.from(document.querySelectorAll('.ak-table-scroll, .ant-table-content, .ant-card-body')).map((element) => { const rect=element.getBoundingClientRect(); return {className:String(element.className), left:Math.round(rect.left), right:Math.round(rect.right), width:Math.round(rect.width), clientWidth:element.clientWidth, scrollWidth:element.scrollWidth, overflowX:getComputedStyle(element).overflowX}; })""")
        dimensions["offenders"] = page.evaluate("""() => Array.from(document.querySelectorAll('body *')).map((element) => { const rect = element.getBoundingClientRect(); return { tag: element.tagName, className: String(element.className).slice(0, 120), left: Math.round(rect.left), right: Math.round(rect.right), width: Math.round(rect.width) }; }).filter((item) => item.right > document.documentElement.clientWidth + 1 || item.left < -1).sort((left, right) => right.width - left.width).slice(0, 12)""")
        raise AssertionError(dimensions)


def login(page: Page) -> None:
    page.goto(f"{BASE}/login", wait_until="domcontentloaded")
    page.locator("#login-email").fill("e2e@app.test")
    page.locator("#login-password").fill("password")
    page.locator('button[type="submit"]').click()
    page.wait_for_url("**/dashboard**")


def navigate(page: Page, path: str) -> None:
    page.evaluate("path => { window.history.pushState({}, '', path); window.dispatchEvent(new PopStateEvent('popstate')); }", path)


def main() -> None:
    print("app-upgrade-e2e:start", flush=True)
    evidence: dict[str, object] = {}
    console_errors: list[str] = []
    expected_conflicts: list[str] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900}, color_scheme="light")
        page = context.new_page()
        page.set_default_timeout(15_000)
        page.route("**/admin-api/**", route_handler)

        def collect_console(message: object) -> None:
            if getattr(message, "type", None) != "error":
                return
            message_text = getattr(message, "text", "")
            if "409 (Conflict)" in message_text:
                expected_conflicts.append(message_text)
            else:
                console_errors.append(message_text)

        page.on("console", collect_console)
        login(page)

        print("app-upgrade-e2e:applications", flush=True)
        navigate(page, "/app/applications")
        page.get_by_role("heading", name="应用管理").wait_for()
        page.get_by_text("__UNI__APPKERNIA", exact=True).wait_for()
        assert page.locator(".ak-global-app-context").count() == 0
        assert page.get_by_role("button", name="升级中心").count() == 1
        assert_no_page_overflow(page)
        run_axe(page, "applications.zh-CN.light.1440", evidence)
        page.screenshot(path=OUT / "applications.zh-CN.light.1440.png", full_page=True)

        page.locator("button:has-text('编 辑')").first.click()
        drawer = page.get_by_role("dialog")
        drawer.wait_for()
        assert drawer.locator('input[placeholder="__UNI__APPKERNIA"]').is_disabled()
        drawer.get_by_text("启动体验", exact=True).scroll_into_view_if_needed()
        assert drawer.get_by_text("已发布版本 v3", exact=True).is_visible()
        assert drawer.get_by_text("草稿有变更", exact=True).is_visible()
        assert drawer.get_by_role("button", name="发布新版本").is_enabled()
        chinese_descriptions = drawer.get_by_label("简体中文 无障碍说明")
        assert chinese_descriptions.count() == 2
        assert chinese_descriptions.nth(0).input_value() == "第一张中文启动介绍"
        drawer.locator("button").filter(has_text=re.compile(r"下\s*移")).first.focus()
        page.keyboard.press("Enter")
        assert chinese_descriptions.nth(0).input_value() == "第二张中文启动介绍"
        drawer.locator("button").filter(has_text=re.compile(r"上\s*移")).nth(1).focus()
        page.keyboard.press("Enter")
        assert chinese_descriptions.nth(0).input_value() == "第一张中文启动介绍"
        page.wait_for_timeout(400)
        for width in (375, 768, 1440):
            page.set_viewport_size({"width": width, "height": 900 if width != 375 else 812})
            drawer.get_by_text("启动体验", exact=True).scroll_into_view_if_needed()
            assert_no_page_overflow(page)
            run_axe(page, f"startup.drawer.zh-CN.light.{width}", evidence)
            page.screenshot(path=STARTUP_OUT / f"startup.drawer.zh-CN.light.{width}.png", full_page=True)
        page.keyboard.press("Escape")
        drawer.wait_for(state="hidden")

        for width in (375, 768, 1024, 1440):
            page.set_viewport_size({"width": width, "height": 900})
            page.mouse.move(1, 1)
            page.wait_for_timeout(300)
            assert_no_page_overflow(page)
            run_axe(page, f"applications.zh-CN.light.{width}", evidence)

        page.get_by_label("显示语言").click()
        page.get_by_role("menuitem", name="English").click()
        page.get_by_role("heading", name="Application management").wait_for()
        page.set_viewport_size({"width": 375, "height": 812})
        page.mouse.move(1, 1)
        page.wait_for_timeout(500)
        assert_no_page_overflow(page)
        run_axe(page, "applications.en-US.light.375", evidence)
        page.screenshot(path=OUT / "applications.en-US.light.375.png", full_page=True)

        page.get_by_role("button", name="Edit").first.click()
        drawer = page.get_by_role("dialog")
        drawer.wait_for()
        for width in (375, 768, 1440):
            page.set_viewport_size({"width": width, "height": 900 if width != 375 else 812})
            drawer.get_by_text("Startup experience", exact=True).scroll_into_view_if_needed()
            assert drawer.get_by_label("English Accessibility description").count() == 2
            assert_no_page_overflow(page)
            run_axe(page, f"startup.drawer.en-US.light.{width}", evidence)
            page.screenshot(path=STARTUP_OUT / f"startup.drawer.en-US.light.{width}.png", full_page=True)
        page.keyboard.press("Escape")
        drawer.wait_for(state="hidden")

        print("app-upgrade-e2e:upgrade-center", flush=True)
        page.set_viewport_size({"width": 1440, "height": 900})
        unselected_routes = [
            "/app/users",
            "/app/content/articles",
            "/app/content/pages",
            "/app/upgrade-center",
            "/system/mobile/releases",
        ]
        for path in unselected_routes:
            navigate(page, path)
            prompt = page.get_by_test_id("app-selection-required")
            prompt.wait_for()
            selector = page.get_by_role("combobox", name="Current application")
            selector.wait_for()
            assert page.locator(".ak-global-app-context").count() == 1
            assert page.locator(".ak-app-scope-context, .ak-app-scope-alert").count() == 0
            assert page.locator(".ak-table-scroll, .ak-release-filters, .ak-content-filters").count() == 0
            assert page.locator(".ant-spin-spinning").count() == 0
            layout = prompt.evaluate("""element => {
                const style = getComputedStyle(element)
                const rect = element.getBoundingClientRect()
                const text = element.firstElementChild.getBoundingClientRect()
                return {
                    minHeight: parseFloat(style.minHeight),
                    horizontalDelta: Math.abs((rect.left + rect.width / 2) - (text.left + text.width / 2)),
                    verticalDelta: Math.abs((rect.top + rect.height / 2) - (text.top + text.height / 2)),
                }
            }""")
            assert layout["minHeight"] >= 280, layout
            assert layout["horizontalDelta"] <= 2, layout
            assert layout["verticalDelta"] <= 2, layout
            assert_no_page_overflow(page)

        navigate(page, "/app/upgrade-center")
        page.get_by_role("heading", name="App upgrade center").wait_for()
        page.get_by_test_id("app-selection-required").wait_for()
        app_context = page.locator(".ak-global-app-context")
        fullscreen = page.get_by_role("button", name="Enter fullscreen")
        app_context_box = app_context.bounding_box()
        fullscreen_box = fullscreen.bounding_box()
        assert app_context_box and fullscreen_box
        assert app_context_box["x"] + app_context_box["width"] <= fullscreen_box["x"] + 1
        run_axe(page, "upgrade.unselected.en-US.light.1440", evidence)
        page.screenshot(path=OUT / "upgrade.unselected.en-US.light.1440.png", full_page=True)

        page.set_viewport_size({"width": 768, "height": 900})
        page.wait_for_timeout(300)
        assert_no_page_overflow(page)
        page.screenshot(path=OUT / "upgrade.unselected.en-US.light.768.png", full_page=True)

        page.set_viewport_size({"width": 375, "height": 812})
        page.wait_for_timeout(300)
        assert_no_page_overflow(page)
        page.screenshot(path=OUT / "upgrade.unselected.en-US.light.375.png", full_page=True)

        selector = page.get_by_role("combobox", name="Current application")
        selector.click()
        page.locator(".ant-select-item-option").filter(has_text="AppKernia").click()
        page.wait_for_url(lambda url: f"app_id={APP_ID}" in url)
        stored_selection = page.evaluate("key => localStorage.getItem(key)", "ak.admin.app-selection.v1")
        assert stored_selection and APP_ID in stored_selection and TENANT_ID in stored_selection

        navigate(page, "/app/users")
        page.wait_for_url(lambda url: "/app/users" in str(url) and f"app_id={APP_ID}" in str(url))
        page.get_by_role("combobox", name="Current application").wait_for()
        page.get_by_role("group", name="Current application").get_by_text("AppKernia", exact=True).wait_for()
        assert page.get_by_test_id("app-selection-required").count() == 0
        assert page.locator(".ak-table-scroll").count() == 1

        persisted_page = context.new_page()
        persisted_page.set_default_timeout(15_000)
        persisted_page.route("**/admin-api/**", route_handler)
        persisted_page.on("console", collect_console)
        login(persisted_page)
        navigate(persisted_page, "/app/users")
        persisted_page.wait_for_url(lambda url: "/app/users" in str(url) and f"app_id={APP_ID}" in str(url))
        persisted_context = persisted_page.locator(".ak-global-app-context")
        persisted_context.get_by_role("combobox").wait_for()
        persisted_context.get_by_text("AppKernia", exact=True).wait_for()
        assert persisted_page.get_by_test_id("app-selection-required").count() == 0
        persisted_page.screenshot(path=OUT / "users.remembered.zh-CN.light.1440.png", full_page=True)
        persisted_page.close()

        navigate(page, "/app/upgrade-center")
        page.wait_for_url(lambda url: "/app/upgrade-center" in str(url) and f"app_id={APP_ID}" in str(url))
        page.get_by_role("heading", name="App upgrade center").wait_for()
        page.get_by_role("combobox", name="Current application").wait_for()
        assert page.get_by_test_id("app-selection-required").count() == 0
        page.get_by_text("3.0.0", exact=True).wait_for()
        assert page.get_by_text("Partially online", exact=True).count() >= 1
        assert page.get_by_role("button", name="Edit").count() == 1
        assert page.get_by_role("button", name="Delete").count() == 1
        assert_no_page_overflow(page)
        run_axe(page, "upgrade.en-US.light.375", evidence)
        page.screenshot(path=OUT / "upgrade.en-US.light.375.png", full_page=True)

        page.set_viewport_size({"width": 1440, "height": 900})
        page.wait_for_timeout(300)
        page.get_by_role("button", name="Publish new version").click()
        page.get_by_role("menuitem", name="WGT resource package").click()
        wgt_drawer = page.get_by_role("dialog", name="Publish WGT resource package")
        wgt_drawer.wait_for()
        page.wait_for_timeout(400)
        assert wgt_drawer.get_by_role("tab", name="Simplified Chinese").count() == 1
        assert wgt_drawer.get_by_role("tab", name="English").count() == 1
        assert wgt_drawer.get_by_role("tab", name="English").get_attribute("aria-selected") == "true"
        english_title = wgt_drawer.get_by_label("English Update title")
        english_title.fill("Preserved WGT title")
        wgt_drawer.get_by_role("tab", name="Simplified Chinese").click()
        wgt_drawer.get_by_role("tab", name="English").click()
        assert english_title.input_value() == "Preserved WGT title"
        run_axe(page, "upgrade.wgt-drawer.en-US.light.1440", evidence)
        page.screenshot(path=OUT / "upgrade.wgt-drawer.en-US.light.1440.png", full_page=True)
        page.keyboard.press("Escape")
        wgt_drawer.wait_for(state="hidden")

        page.get_by_role("button", name="Publish new version").click()
        page.get_by_role("menuitem", name="Native app package").click()
        native_drawer = page.get_by_role("dialog", name="Publish native app package")
        native_drawer.wait_for()
        assert native_drawer.get_by_role("tab", name="Simplified Chinese").count() == 1
        assert native_drawer.get_by_role("tab", name="English").get_attribute("aria-selected") == "true"
        native_drawer.get_by_label("Publish immediately").click()
        page.locator(".ant-drawer-extra button").last.click()
        native_drawer.get_by_text("Correct the highlighted fields.", exact=True).wait_for()
        native_drawer.get_by_role("tab", name="Simplified Chinese").wait_for()
        assert native_drawer.get_by_role("tab", name="Simplified Chinese").get_attribute("aria-selected") == "true"
        page.screenshot(path=OUT / "upgrade.native-drawer.validation.en-US.light.1440.png", full_page=True)
        page.keyboard.press("Escape")
        native_drawer.wait_for(state="hidden")

        navigate(page, f"/app/content/pages?app_id={APP_ID}")
        page.get_by_role("heading", name="Single pages").wait_for()
        page.get_by_text("About us", exact=True).wait_for()
        page.get_by_role("button", name="Edit").first.click()
        app_page_drawer = page.get_by_role("dialog", name="Edit single page")
        app_page_drawer.wait_for()
        assert app_page_drawer.get_by_role("tab", name="Simplified Chinese").count() == 1
        assert app_page_drawer.get_by_role("tab", name="English").get_attribute("aria-selected") == "true"
        page.screenshot(path=OUT / "app-page.localized-tabs.en-US.light.1440.png", full_page=True)
        page.keyboard.press("Escape")
        app_page_drawer.wait_for(state="hidden")

        page.get_by_role("button", name="Edit").click()
        edit_drawer = page.get_by_role("dialog", name="Edit version draft")
        edit_drawer.wait_for()
        title_input = edit_drawer.get_by_label("Update title").first
        title_input.fill("Preserved after conflict")
        edit_drawer.get_by_role("button", name="Save").click()
        edit_drawer.get_by_text("Another administrator updated this version. Your form input has been kept; refresh and try again.", exact=True).wait_for()
        assert title_input.input_value() == "Preserved after conflict"
        page.keyboard.press("Escape")
        edit_drawer.wait_for(state="hidden")

        for width in (768, 1024, 1440):
            page.set_viewport_size({"width": width, "height": 900})
            page.mouse.move(1, 1)
            page.wait_for_timeout(300)
            assert_no_page_overflow(page)
            run_axe(page, f"upgrade.en-US.light.{width}", evidence)

        dark = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900}, color_scheme="dark")
        dark_page = dark.new_page()
        dark_page.route("**/admin-api/**", route_handler)
        login(dark_page)
        navigate(dark_page, f"/app/upgrade-center?app_id={APP_ID}&page=1&page_size=20")
        dark_page.get_by_role("heading", name="App升级中心").wait_for()
        assert dark_page.evaluate("matchMedia('(prefers-color-scheme: dark)').matches") is True
        assert_no_page_overflow(dark_page)
        run_axe(dark_page, "upgrade.zh-CN.preferred-dark.1440", evidence)
        dark_page.screenshot(path=OUT / "upgrade.zh-CN.preferred-dark.1440.png", full_page=True)
        dark.close()

        assert len(expected_conflicts) == 1, expected_conflicts
        assert not console_errors, console_errors
        evidence["console"] = {"unexpected_errors": console_errors, "expected_conflicts": len(expected_conflicts)}
        evidence["coverage"] = {"locales": ["zh-CN", "en-US"], "viewports": [375, 768, 1024, 1440], "color_schemes": ["light", "preferred-dark"]}
        serialized = json.dumps(evidence, ensure_ascii=False, indent=2)
        (OUT / "e2e-results.json").write_text(serialized, encoding="utf-8")
        startup_evidence = {key: value for key, value in evidence.items() if key.startswith("startup.") or key in ("console", "coverage")}
        (STARTUP_OUT / "e2e-results.json").write_text(json.dumps(startup_evidence, ensure_ascii=False, indent=2), encoding="utf-8")
        context.close()
        browser.close()
    print("app-upgrade-e2e:done", flush=True)


if __name__ == "__main__":
    main()
