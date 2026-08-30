from __future__ import annotations

import json
import os
from pathlib import Path

from playwright.sync_api import Route, sync_playwright


ROOT = Path(__file__).resolve().parents[3]
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174")
AXE = ROOT / "apps/ak-admin/node_modules/axe-core/axe.min.js"
OUT = ROOT / "apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-share-configuration/screenshots"
OUT.mkdir(parents=True, exist_ok=True)

TENANT_ID = "423e4567-e89b-12d3-a456-426614174000"
APP_ID = "00000000-0000-4000-8000-000000000001"
CONFIG_ID = "923e4567-e89b-12d3-a456-426614174000"
PERMISSIONS = [
    "sys.share_config.read",
    "sys.share_config.create",
    "sys.share_config.update",
    "sys.share_config.delete",
    "app.application.read",
    "app.application.update",
    "app.application.disable",
    "app.application.delete",
    "app.share_binding.read",
    "app.share_binding.update",
]

CONFIG = {
    "id": CONFIG_ID,
    "tenant_id": TENANT_ID,
    "name": "应核正式微信分享",
    "description": "三端发布身份统一配置",
    "provider_code": "wechat",
    "external_app_id": "wx1234567890abcdef",
    "config_schema_version": 1,
    "secret_field_names": [],
    "has_secret": False,
    "status": "active",
    "binding_count": 1,
    "lock_version": 3,
    "created_at": "2026-08-29T00:00:00Z",
    "updated_at": "2026-08-29T06:00:00Z",
    "public_config": {
        "android": {"enabled": True, "package_name": "com.appkernia.mobile", "signature": "AABBCCDDEEFF0011"},
        "ios": {"enabled": True, "bundle_id": "com.appkernia.mobile", "universal_link": "https://share.example.com/app/"},
        "harmony": {"enabled": True, "bundle_name": "com.appkernia.mobile"},
    },
}

APP = {
    "id": APP_ID,
    "tenant_id": TENANT_ID,
    "appid": "__UNI__APPKERNIA",
    "appid_pending": False,
    "app_type": "uni_app_x",
    "code": "appkernia",
    "name": "应核 AppKernia",
    "description": "官方资讯样板 App",
    "introduction": "",
    "remark": "",
    "status": "active",
    "default_locale": "zh-CN",
    "registration_enabled": True,
    "registration_verification_mode": "email_otp",
    "owner_type": "tenant",
    "owner_id": TENANT_ID,
    "icon_file_id": None,
    "managers": [],
    "members": [],
    "screenshots": [],
    "channels": [],
    "store_listings": [],
    "is_default": True,
    "lock_version": 2,
    "created_at": "2026-08-01T00:00:00Z",
    "updated_at": "2026-08-29T06:00:00Z",
    "startup": {
        "translations": {
            "zh-CN": {"display_name": "应核 AppKernia", "subtitle": "开发者资讯与样板 App"},
            "en-US": {"display_name": "AppKernia", "subtitle": "Developer news and reference App"},
        },
        "onboarding_enabled": False,
        "draft_slides": [],
        "published_version": 1,
        "published_at": "2026-08-20T00:00:00Z",
        "draft_changed": False,
    },
}

BINDING = {
    "id": "a23e4567-e89b-12d3-a456-426614174000",
    "app_id": APP_ID,
    "provider_code": "wechat",
    "share_config_id": CONFIG_ID,
    "share_config_name": CONFIG["name"],
    "config_status": "active",
    "enabled": True,
    "scenes": ["session", "timeline", "favorite"],
    "share_origin": "https://share.example.com",
    "fallback_mode": "system",
    "lock_version": 1,
    "updated_at": "2026-08-29T06:00:00Z",
}


def respond(route: Route, data: object) -> None:
    route.fulfill(
        status=200,
        content_type="application/json",
        body=json.dumps({"code": "OK", "message": "OK", "data": data, "request_id": "share-e2e"}),
    )


def handler(route: Route) -> None:
    url = route.request.url
    method = route.request.method
    if url.endswith("/auth/public-config"):
        return respond(route, {"locale": "zh-CN", "default_locale": "zh-CN", "supported_locales": ["zh-CN", "en-US"], "feature_flags": {}, "settings": {}})
    if url.endswith("/auth/login"):
        return respond(route, {"access_token": "e2e", "token_type": "Bearer", "expires_in": 900, "csrf_token": "csrf-e2e-value-long-enough"})
    if url.endswith("/auth/context"):
        return respond(route, {"user": {"id": CONFIG_ID, "email": "e2e@app.test", "display_name": "E2E", "locale": "zh-CN", "time_zone": "UTC", "avatar_url": None}, "active_tenant": {"id": TENANT_ID, "name": "E2E", "code": "e2e"}, "available_tenants": [], "roles": [], "permissions": PERMISSIONS, "menus": [], "feature_flags": {}, "menu_revision": "1", "permission_revision": "1", "server_time": "2026-08-29T06:00:00Z"})
    if "/dashboard/summary" in url:
        return respond(route, {"range": "7d", "start_at": "2026-08-22T00:00:00Z", "end_at": "2026-08-29T00:00:00Z", "metrics": []})
    if "/dashboard/trends" in url:
        return respond(route, {"range": "7d", "start_at": "2026-08-22T00:00:00Z", "end_at": "2026-08-29T00:00:00Z", "series": []})
    if "/dashboard/activity" in url:
        return respond(route, {"range": "7d", "start_at": "2026-08-22T00:00:00Z", "end_at": "2026-08-29T00:00:00Z", "operations": [], "failed_jobs": [], "security_events": []})
    if url.endswith("/dictionaries/system.language"):
        english = route.request.headers.get("accept-language", "").startswith("en-US")
        return respond(route, {"code": "system.language", "locale": "en-US" if english else "zh-CN", "extension_policy": "fixed", "items": [{"value": "zh-CN", "label": "Simplified Chinese" if english else "简体中文", "is_default": True, "extra": {}}, {"value": "en-US", "label": "English", "is_default": False, "extra": {}}]})
    if url.endswith("/me") and method == "PATCH":
        return respond(route, {"id": CONFIG_ID, "email": "e2e@app.test", "display_name": "E2E", "locale": "en-US", "time_zone": "UTC", "avatar_url": None})
    if f"/apps/{APP_ID}/share-bindings/wechat/preflight" in url:
        return respond(route, {"ready": True, "provider_code": "wechat", "platforms": ["android", "ios", "harmony"], "scenes": ["session", "timeline", "favorite"], "issues": []})
    if f"/apps/{APP_ID}/share-bindings" in url:
        return respond(route, [BINDING])
    if "/share-configs" in url:
        return respond(route, {"items": [CONFIG], "page": 1, "page_size": 20, "total": 1})
    if "/apps?" in url:
        return respond(route, {"items": [APP], "page": 1, "page_size": 20, "total": 1})
    return respond(route, {})


def assert_accessible(page, name: str, evidence: dict[str, object]) -> None:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate("async () => await axe.run({exclude:[['.ant-table-measure-row']]},{resultTypes:['violations']})")
    severe = [item for item in result["violations"] if item["impact"] in ("serious", "critical")]
    current = evidence.get(name, {})
    evidence[name] = {**(current if isinstance(current, dict) else {}), "serious_critical": len(severe)}
    assert not severe, severe


def navigate(page, path: str) -> None:
    page.evaluate("path => { history.pushState({}, '', path); window.dispatchEvent(new PopStateEvent('popstate')); }", path)


def main() -> None:
    evidence: dict[str, object] = {}
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
        page = context.new_page()
        page.route("**/admin-api/**", handler)
        page.set_default_timeout(10_000)
        console_errors: list[str] = []
        page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)

        page.goto(BASE + "/login", wait_until="domcontentloaded")
        page.get_by_label("账号").fill("e2e@app.test")
        page.get_by_label("密码").fill("fixture-only")
        page.locator('button[type="submit"]').click()
        page.wait_for_url("**/dashboard**")

        navigate(page, "/system/settings/share-configs")
        page.get_by_role("heading", name="分享配置").wait_for()
        page.get_by_text(CONFIG["name"]).wait_for()
        desktop_padding = page.locator(".ak-page-container").evaluate("element => { const style = getComputedStyle(element); return { left: parseFloat(style.paddingLeft), right: parseFloat(style.paddingRight) }; }")
        alert_box = page.locator(".ak-page-container .ant-alert").bounding_box()
        desktop_right_gap = 1440 - (alert_box["x"] + alert_box["width"])
        assert desktop_padding["left"] == desktop_padding["right"] and desktop_padding["right"] >= 47
        assert desktop_right_gap >= 47
        evidence["share-configs.zh-CN.1440"] = {"padding_inline": desktop_padding, "right_gap": desktop_right_gap}
        assert_accessible(page, "share-configs.zh-CN.1440", evidence)
        page.screenshot(path=OUT / "share-configs-zh-CN-1440.png", full_page=True)

        page.get_by_role("button", name="新建配置").click()
        create_drawer = page.get_by_role("dialog", name="新建分享配置")
        create_drawer.wait_for()
        create_drawer.get_by_role("button", name="查看微信开放平台申请指引").click()
        guide_dialog = page.get_by_role("dialog", name="如何申请微信分享 AppID？")
        guide_dialog.wait_for()
        page.wait_for_timeout(400)
        guide_links = guide_dialog.get_by_role("link")
        assert guide_links.count() == 8
        for index in range(guide_links.count()):
            link = guide_links.nth(index)
            assert link.get_attribute("target") == "_blank"
            assert link.get_attribute("rel") == "noopener noreferrer"
            assert link.get_attribute("href").startswith("https://")
        evidence["share-guide.zh-CN.1440"] = {"steps": 5, "external_links": guide_links.count()}
        assert_accessible(page, "share-guide.zh-CN.1440", evidence)
        page.screenshot(path=OUT / "share-guide-zh-CN-1440.png", full_page=True)
        guide_dialog.get_by_role("button", name="我知道了").click()
        guide_dialog.wait_for(state="hidden")
        page.keyboard.press("Escape")
        create_drawer.wait_for(state="hidden")

        page.get_by_role("button", name="编辑").click()
        drawer = page.get_by_role("dialog", name="编辑分享配置")
        drawer.wait_for()
        assert drawer.get_by_label("Android 应用签名").get_attribute("type") == "password"
        page.screenshot(path=OUT / "share-config-editor-zh-CN-1440.png", full_page=True)
        page.keyboard.press("Escape")
        drawer.wait_for(state="hidden")

        page.set_viewport_size({"width": 375, "height": 812})
        page.get_by_text(CONFIG["name"]).wait_for()
        mobile_padding = page.locator(".ak-page-container").evaluate("element => { const style = getComputedStyle(element); return { left: parseFloat(style.paddingLeft), right: parseFloat(style.paddingRight) }; }")
        assert mobile_padding == {"left": 16, "right": 16}
        evidence["share-configs.zh-CN.375"] = {"padding_inline": mobile_padding}
        assert_accessible(page, "share-configs.zh-CN.375", evidence)
        page.screenshot(path=OUT / "share-configs-zh-CN-375.png", full_page=True)

        page.get_by_role("button", name="新建配置").click()
        create_drawer = page.get_by_role("dialog", name="新建分享配置")
        create_drawer.get_by_role("button", name="查看微信开放平台申请指引").click()
        guide_dialog = page.get_by_role("dialog", name="如何申请微信分享 AppID？")
        guide_dialog.wait_for()
        page.wait_for_timeout(400)
        guide_box = guide_dialog.bounding_box()
        assert guide_box["x"] >= 0 and guide_box["x"] + guide_box["width"] <= 375
        assert page.evaluate("document.documentElement.scrollWidth <= document.documentElement.clientWidth")
        evidence["share-guide.zh-CN.375"] = {"fits_viewport": True, "horizontal_overflow": False}
        assert_accessible(page, "share-guide.zh-CN.375", evidence)
        page.screenshot(path=OUT / "share-guide-zh-CN-375.png", full_page=True)
        guide_dialog.get_by_role("button", name="我知道了").click()
        guide_dialog.wait_for(state="hidden")
        page.keyboard.press("Escape")
        create_drawer.wait_for(state="hidden")

        page.set_viewport_size({"width": 1440, "height": 900})
        navigate(page, "/app/applications")
        page.get_by_role("heading", name="应用管理").wait_for()
        page.get_by_text(APP["name"]).wait_for()
        page.get_by_role("button", name=f"{APP['name']} 的操作").click()
        action_menu = page.locator(".ant-dropdown-menu")
        action_menu.wait_for()
        action_items = action_menu.get_by_role("menuitem")
        assert action_items.count() == 5
        for index in range(action_items.count()):
            assert action_items.nth(index).locator(".anticon").count() == 1
        action_width = page.get_by_role("columnheader", name="操作").bounding_box()["width"]
        assert action_width <= 120
        evidence["app-actions.zh-CN.1440"] = {"items": action_items.count(), "items_with_icons": action_items.count(), "column_width": action_width}
        assert_accessible(page, "app-actions.zh-CN.1440", evidence)
        page.screenshot(path=OUT / "app-actions-menu-zh-CN-1440.png", full_page=True)
        action_menu.get_by_role("menuitem", name="分享配置").click()
        page.set_viewport_size({"width": 768, "height": 900})
        binding_drawer = page.get_by_role("dialog", name=f"App 分享配置 · {APP['name']}")
        binding_drawer.wait_for()
        binding_drawer.get_by_role("button", name="执行预检").click()
        binding_drawer.get_by_text("预检通过，可以导出构建配置").wait_for()
        assert_accessible(page, "share-binding.zh-CN.768", evidence)
        page.screenshot(path=OUT / "app-share-binding-zh-CN-768.png", full_page=True)
        page.keyboard.press("Escape")
        binding_drawer.wait_for(state="hidden")

        navigate(page, "/system/settings/share-configs")
        page.get_by_role("heading", name="分享配置").wait_for()
        page.get_by_label("显示语言").click()
        page.get_by_role("menuitem", name="English").click()
        page.get_by_role("heading", name="Share configurations").wait_for()
        page.set_viewport_size({"width": 375, "height": 812})
        page.mouse.move(180, 360)
        page.wait_for_timeout(400)
        assert_accessible(page, "share-configs.en-US.375", evidence)
        page.screenshot(path=OUT / "share-configs-en-US-375.png", full_page=True)
        page.get_by_role("button", name="New configuration").click()
        create_drawer = page.get_by_role("dialog", name="New share configuration")
        create_drawer.get_by_role("button", name="View the WeChat Open Platform application guide").click()
        guide_dialog = page.get_by_role("dialog", name="How do I obtain a WeChat sharing AppID?")
        guide_dialog.wait_for()
        page.wait_for_timeout(400)
        assert guide_dialog.get_by_role("link").count() == 8
        evidence["share-guide.en-US.375"] = {"external_links": 8}
        assert_accessible(page, "share-guide.en-US.375", evidence)
        page.screenshot(path=OUT / "share-guide-en-US-375.png", full_page=True)

        evidence["console_errors"] = console_errors
        assert not console_errors, console_errors
        (OUT / "axe-results.json").write_text(json.dumps(evidence, ensure_ascii=False, indent=2), encoding="utf-8")
        browser.close()

    print(json.dumps(evidence, ensure_ascii=False))


if __name__ == "__main__":
    main()
