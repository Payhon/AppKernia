from __future__ import annotations

import json
import os
import struct
import uuid
import zlib
from pathlib import Path
from urllib.parse import parse_qs, urlparse

from playwright.sync_api import Page, Response, expect, sync_playwright

from e2e_navigation_helpers import open_system_page


ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "output" / "playwright"
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
BASE_URL = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4173").rstrip("/")
EMAIL = os.environ.get("AK_E2E_EMAIL", "codex-e2e@appkernia.test")
DISPLAY_NAME = os.environ.get("AK_E2E_DISPLAY_NAME", "Codex E2E")
SOURCE_TENANT_CODE = os.environ.get("AK_E2E_TENANT_CODE", "docker-test")
PASSWORD = os.environ["AK_E2E_PASSWORD"]
AUDIT_EVENT_ID = os.environ["AK_E2E_AUDIT_EVENT_ID"]
AUDIT_EVENT_TYPE = os.environ["AK_E2E_AUDIT_EVENT_TYPE"]
AUDIT_REQUEST_ID = os.environ["AK_E2E_AUDIT_REQUEST_ID"]


def png_chunk(kind: bytes, payload: bytes) -> bytes:
    return struct.pack(">I", len(payload)) + kind + payload + struct.pack(">I", zlib.crc32(kind + payload) & 0xFFFFFFFF)


def avatar_png() -> bytes:
    """Build a tiny standards-compliant RGBA PNG that Chromium can decode."""
    signature = b"\x89PNG\r\n\x1a\n"
    header = struct.pack(">IIBBBBB", 2, 2, 8, 6, 0, 0, 0)
    row = b"\x00" + b"\x2f\x6f\xde\xff" * 2
    return signature + png_chunk(b"IHDR", header) + png_chunk(b"IDAT", zlib.compress(row * 2)) + png_chunk(b"IEND", b"")


def assert_no_horizontal_overflow(page: Page) -> None:
    dimensions = page.evaluate("""() => {
        const root = document.documentElement
        return {
            scrollWidth: root.scrollWidth,
            clientWidth: root.clientWidth,
            offenders: [...document.querySelectorAll('body *')].map((element) => {
                const rect = element.getBoundingClientRect()
                return { tag: element.tagName, className: String(element.className).slice(0, 120), left: Math.round(rect.left), right: Math.round(rect.right), width: Math.round(rect.width) }
            }).filter((item) => item.right > root.clientWidth + 1 || item.left < -1).sort((left, right) => right.right - left.right).slice(0, 8),
        }
    }""")
    assert dimensions["scrollWidth"] <= dimensions["clientWidth"], f"page has horizontal overflow: {dimensions}"


def run_axe(page: Page, name: str, evidence: dict[str, object]) -> None:
    if page.locator(".ant-message-fade-appear-active").count():
        page.wait_for_timeout(500)
    page.add_script_tag(path=str(AXE))
    # Ant Design inserts an aria-hidden zero-height table measurement clone. It is
    # never interactive; audit the rendered table and its real selection controls.
    result = page.evaluate("async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })")
    violations = result["violations"]
    severe = [item for item in violations if item["impact"] in ("critical", "serious")]
    evidence[name] = {
        "critical_or_serious": len(severe),
        "violations": [
            {"id": item["id"], "impact": item["impact"], "nodes": len(item["nodes"])}
            for item in violations
        ],
    }
    assert not severe, f"axe serious/critical violations: {severe}"


def choose_locale(page: Page, label: str, option: str) -> None:
    page.get_by_label(label).select_option(label=option)


def main() -> None:
    OUTPUT.mkdir(parents=True, exist_ok=True)
    avatar_fixture = OUTPUT / "avatar-e2e-fixture.png"
    avatar_invalid_fixture = OUTPUT / "avatar-e2e-invalid.txt"
    avatar_fixture.write_bytes(avatar_png())
    avatar_invalid_fixture.write_text("not an image\n", encoding="utf-8")
    evidence: dict[str, object] = {}
    console_errors: list[str] = []
    content_languages: list[str] = []
    loaded_scripts: list[str] = []
    avatar_content_responses: list[Response] = []
    user_status_responses: list[Response] = []

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
        page = context.new_page()
        page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
        page.on("request", lambda request: loaded_scripts.append(request.url) if request.resource_type == "script" else None)
        page.on(
            "response",
            lambda response: content_languages.append(response.headers.get("content-language", ""))
            if "/admin-api/" in response.url
            else None,
        )
        page.on(
            "response",
            lambda response: user_status_responses.append(response)
            if "/admin-api/v1/users/" in response.url
            and (response.url.endswith("/enable") or response.url.endswith("/disable"))
            else None,
        )
        page.on(
            "response",
            lambda response: avatar_content_responses.append(response)
            if "/admin-api/v1/me/avatar/content?v=" in response.url
            else None,
        )

        page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        assert page.locator("html").get_attribute("lang") == "zh-CN"
        assert page.get_by_role("heading", name="登录 AppKernia").is_visible()
        assert page.get_by_label("账号").get_attribute("autocomplete") == "username"
        assert page.get_by_label("密码").get_attribute("autocomplete") == "current-password"
        page.keyboard.press("Tab")
        assert page.evaluate("document.activeElement?.getAttribute('aria-label')") == "显示语言"
        page.keyboard.press("Tab")
        assert page.evaluate("document.activeElement?.id") == "login-email"
        login_time_origin = page.evaluate("performance.timeOrigin")
        assert_no_horizontal_overflow(page)
        run_axe(page, "login.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-login.zh-CN.1440.png", full_page=True)

        choose_locale(page, "显示语言", "English")
        assert page.locator("html").get_attribute("lang") == "en-US"
        assert page.get_by_role("heading", name="Sign in to AppKernia").is_visible()
        assert page.evaluate("performance.timeOrigin") == login_time_origin
        run_axe(page, "login.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-login.en-US.1440.png", full_page=True)

        page.set_viewport_size({"width": 375, "height": 812})
        assert_no_horizontal_overflow(page)
        run_axe(page, "login.en-US.375", evidence)
        page.screenshot(path=OUTPUT / "admin-login.en-US.375.png", full_page=True)

        choose_locale(page, "Display language", "简体中文")
        page.set_viewport_size({"width": 768, "height": 900})
        assert_no_horizontal_overflow(page)
        page.screenshot(path=OUTPUT / "admin-login.zh-CN.768.png", full_page=True)

        page.get_by_role("link", name="创建账号", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/register")
        page.get_by_role("heading", name="创建账号", exact=True).wait_for()
        registration_time_origin = page.evaluate("performance.timeOrigin")
        terms = page.get_by_role("checkbox", name="我已阅读并同意当前工作区的使用条款与隐私规则。")
        assert terms.is_checked() is False
        assert page.get_by_label("密码", exact=True).get_attribute("autocomplete") == "new-password"
        assert page.get_by_label("确认密码", exact=True).get_attribute("autocomplete") == "new-password"
        run_axe(page, "register.zh-CN.768", evidence)
        page.screenshot(path=OUTPUT / "admin-register.zh-CN.768.png", full_page=True)
        choose_locale(page, "显示语言", "English")
        assert page.evaluate("performance.timeOrigin") == registration_time_origin
        page.get_by_role("heading", name="Create an account", exact=True).wait_for()
        run_axe(page, "register.en-US.768", evidence)
        registered_email = f"browser-register-{uuid.uuid4()}@example.test"
        page.get_by_label("Display name").fill("Browser Registered Member")
        page.get_by_label("Email").fill(registered_email)
        page.get_by_label("Password", exact=True).fill("Browser registration password 2026!")
        page.get_by_label("Confirm password", exact=True).fill("Browser registration password 2026!")
        page.get_by_role("button", name="Create account", exact=True).click()
        page.get_by_text("Agree to the terms of use and privacy rules first.", exact=True).wait_for()
        page.get_by_role("checkbox", name="I have read and agree to this workspace's terms of use and privacy rules.").check()
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/auth/register")) as registration_response:
            page.get_by_role("button", name="Create account", exact=True).click()
        assert registration_response.value.status == 202, registration_response.value.text()
        page.get_by_role("heading", name="Registration request accepted", exact=True).wait_for()
        page.screenshot(path=OUTPUT / "admin-register-success.en-US.768.png", full_page=True)
        page.get_by_role("link", name="Back to sign in", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/login")
        choose_locale(page, "Display language", "简体中文")

        page.get_by_role("link", name="忘记密码？", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/forgot-password")
        page.get_by_role("heading", name="找回密码", exact=True).wait_for()
        run_axe(page, "forgot-password.zh-CN.768", evidence)
        page.screenshot(path=OUTPUT / "admin-forgot-password.zh-CN.768.png", full_page=True)
        page.get_by_label("邮箱").fill(f"unknown-{uuid.uuid4()}@example.test")
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/auth/password/forgot")) as unknown_forgot_response:
            page.get_by_role("button", name="发送重置指引", exact=True).click()
        assert unknown_forgot_response.value.status == 202
        unknown_forgot_data = unknown_forgot_response.value.json()["data"]
        page.get_by_role("heading", name="请求已受理", exact=True).wait_for()
        choose_locale(page, "显示语言", "English")
        page.get_by_role("heading", name="Request accepted", exact=True).wait_for()
        run_axe(page, "forgot-password.en-US.768", evidence)
        page.screenshot(path=OUTPUT / "admin-forgot-password-success.en-US.768.png", full_page=True)

        page.goto(f"{BASE_URL}/forgot-password", wait_until="networkidle")
        page.get_by_label("Email").fill(EMAIL)
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/auth/password/forgot")) as known_forgot_response:
            page.get_by_role("button", name="Send reset instructions", exact=True).click()
        assert known_forgot_response.value.status == 202
        assert known_forgot_response.value.json()["data"] == unknown_forgot_data
        page.get_by_role("heading", name="Request accepted", exact=True).wait_for()

        invalid_reset_token = "invalid-browser-reset-token-that-is-long-enough"
        page.goto(f"{BASE_URL}/reset-password?token={invalid_reset_token}", wait_until="networkidle")
        page.wait_for_url(f"{BASE_URL}/reset-password")
        page.set_viewport_size({"width": 375, "height": 812})
        assert page.get_by_label("New password", exact=True).get_attribute("autocomplete") == "new-password"
        assert page.get_by_label("Confirm new password", exact=True).get_attribute("autocomplete") == "new-password"
        run_axe(page, "reset-password.en-US.375", evidence)
        page.get_by_label("New password", exact=True).fill("Browser reset password 2026!")
        page.get_by_label("Confirm new password", exact=True).fill("Browser reset password 2026!")
        page.get_by_role("button", name="Reset password", exact=True).click()
        page.get_by_role("alert").get_by_text("The reset link is invalid or has expired. Request a new one.", exact=True).wait_for()
        page.screenshot(path=OUTPUT / "admin-reset-password-invalid.en-US.375.png", full_page=True)

        page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        choose_locale(page, "Display language", "简体中文")
        page.set_viewport_size({"width": 768, "height": 900})
        page.get_by_label("账号").fill("unknown@appkernia.test")
        page.get_by_label("密码").fill("wrong-password-value")
        page.locator('button[type="submit"]').click()
        unknown_error = page.get_by_role("alert").inner_text()
        page.get_by_label("账号").fill(EMAIL)
        page.get_by_label("密码").fill("wrong-password-value")
        page.locator('button[type="submit"]').click()
        known_error = page.get_by_role("alert").inner_text()
        assert known_error == unknown_error == "账号或密码不正确"

        page.set_viewport_size({"width": 1440, "height": 900})
        page.get_by_label("账号").fill(EMAIL)
        page.get_by_label("密码").fill(PASSWORD)
        page.locator('button[type="submit"]').click()
        page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))
        page.get_by_role("heading", name="Dashboard").wait_for()
        if page.locator("html").get_attribute("lang") != "zh-CN":
            choose_locale(page, "Display language", "简体中文")
        page.get_by_role("heading", name="关键指标").wait_for()
        page.get_by_text("用户总数", exact=True).wait_for()
        page.get_by_role("img", name="Dashboard 每日趋势折线图").wait_for()
        assert page.get_by_text("查看趋势数据表", exact=True).is_visible()
        assert page.get_by_text(DISPLAY_NAME, exact=True).is_visible()
        assert "zh-CN" in content_languages
        run_axe(page, "dashboard.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-dashboard.zh-CN.1440.png", full_page=True)

        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/dashboard/summary?range=7d")
        ) as range_response:
            with page.expect_response(
                lambda response: response.url.endswith("/admin-api/v1/dashboard/trends?range=7d")
            ) as trends_response:
                page.get_by_text("7 天", exact=True).click()
        assert range_response.value.status == 200
        assert trends_response.value.status == 200
        page.wait_for_url(f"{BASE_URL}/dashboard?range=7d")
        assert range_response.value.json()["data"]["range"] == "7d"
        trend_series = {
            series["key"]: sum(point["value"] for point in series["points"])
            for series in trends_response.value.json()["data"]["series"]
        }
        assert trend_series["logins.success"] > 0
        assert trend_series["logins.failure"] > 0

        dashboard_time_origin = page.evaluate("performance.timeOrigin")
        choose_locale(page, "显示语言", "English")
        assert page.locator("html").get_attribute("lang") == "en-US"
        assert page.get_by_text("Current workspace", exact=True).is_visible()
        assert page.get_by_role("heading", name="Key metrics").is_visible()
        assert page.get_by_text("Total users", exact=True).is_visible()
        assert page.evaluate("performance.timeOrigin") == dashboard_time_origin
        run_axe(page, "dashboard.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-dashboard.en-US.1440.png", full_page=True)

        page.get_by_role("link", name=DISPLAY_NAME, exact=True).click()
        page.wait_for_url(f"{BASE_URL}/profile/basic")
        page.get_by_role("heading", name="Basic Settings").wait_for()
        assert page.get_by_label("Sign-in email").is_disabled()
        avatar_input = page.locator("#profile-avatar-file")
        assert avatar_input.get_attribute("accept") == "image/jpeg,image/png,image/webp"
        avatar_input.set_input_files(avatar_invalid_fixture)
        page.get_by_text("Choose a JPEG, PNG, or WebP image up to 5 MB.", exact=True).wait_for()
        avatar_input.set_input_files(avatar_fixture)
        page.get_by_text("Choose a JPEG, PNG, or WebP image up to 5 MB.", exact=True).wait_for(state="detached")
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/me/avatar/upload-session")
            and response.request.method == "POST"
        ) as avatar_session_response:
            with page.expect_response(
                lambda response: "/admin-api/v1/me/avatar/upload-sessions/" in response.url
                and response.url.endswith("/content")
                and response.request.method == "PUT"
            ) as avatar_upload_response:
                page.get_by_role("button", name="Upload avatar", exact=True).click()
        assert avatar_session_response.value.status == 201
        assert avatar_upload_response.value.status == 200
        page.get_by_text("Your avatar has been updated.", exact=True).wait_for()
        uploaded_avatar = page.locator(".ak-avatar-preview .ant-avatar img")
        uploaded_avatar.wait_for()
        assert uploaded_avatar.get_attribute("alt")
        assert (uploaded_avatar.get_attribute("src") or "").startswith("blob:")
        assert avatar_content_responses, "authenticated avatar content was not loaded"
        avatar_content_response = avatar_content_responses[-1]
        assert avatar_content_response.status == 200
        assert avatar_content_response.headers.get("content-type") == "image/png"
        avatar_file_id = parse_qs(urlparse(avatar_content_response.url).query)["v"][0]
        uuid.UUID(avatar_file_id)
        run_axe(page, "profile-avatar.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-profile-avatar.en-US.1440.png", full_page=True)
        run_axe(page, "profile.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-profile.en-US.1440.png", full_page=True)
        for width in (1024, 768, 375):
            page.set_viewport_size({"width": width, "height": 900 if width > 375 else 812})
            assert_no_horizontal_overflow(page)
            page.screenshot(path=OUTPUT / f"admin-profile.en-US.{width}.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})
        page.get_by_label("Display name").fill(f"{DISPLAY_NAME} Test")
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/me") and response.request.method == "PATCH") as first_save:
            page.get_by_role("button", name="Save").click()
        assert first_save.value.status == 200
        page.get_by_text("Your profile settings have been saved.", exact=True).wait_for()
        page.get_by_label("Display name").fill(DISPLAY_NAME)
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/me") and response.request.method == "PATCH") as second_save:
            page.get_by_role("button", name="Save").click()
        assert second_save.value.status == 200
        page.get_by_text("Your profile settings have been saved.", exact=True).wait_for()
        choose_locale(page, "Display language", "简体中文")
        page.get_by_role("heading", name="基本设置").wait_for()
        page.get_by_text("头像已更新。", exact=True).wait_for()
        page.get_by_role("img", name=f"{DISPLAY_NAME} 的头像", exact=True).wait_for()
        page.locator("#profile-avatar-file").set_input_files(avatar_invalid_fixture)
        page.get_by_text("请选择不超过 5 MB 的 JPEG、PNG 或 WebP 图片。", exact=True).wait_for()
        run_axe(page, "profile-avatar.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-profile-avatar.zh-CN.1440.png", full_page=True)
        run_axe(page, "profile.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-profile.zh-CN.1440.png", full_page=True)
        choose_locale(page, "显示语言", "English")

        org_suffix = uuid.uuid4().hex[:8]
        root_name = f"Platform {org_suffix}"
        child_name = f"Engineering {org_suffix}"
        page.get_by_role("link", name="Departments", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/users/departments")
        page.get_by_role("heading", name="Departments", exact=True).wait_for()
        page.get_by_role("button", name="Create department", exact=True).click()
        department_modal = page.locator(".ant-modal:visible")
        department_modal.get_by_label("Department code").fill(f"ROOT-{org_suffix}")
        department_modal.get_by_label("Department name").fill(root_name)
        department_modal.get_by_label("Organization type").click()
        page.get_by_title("Company", exact=True).click()
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/org/units") and response.request.method == "POST") as root_create_response:
            department_modal.get_by_role("button", name="Save", exact=True).click()
        assert root_create_response.value.status == 201
        page.get_by_text(root_name, exact=False).first.wait_for()
        page.get_by_role("button", name="Create department", exact=True).click()
        department_modal = page.locator(".ant-modal:visible")
        department_modal.get_by_label("Department code").fill(f"ENG-{org_suffix}")
        department_modal.get_by_label("Department name").fill(child_name)
        with page.expect_response(lambda response: "/admin-api/v1/org/units/tree" in response.url and response.request.method == "GET") as child_tree_response:
            with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/org/units") and response.request.method == "POST") as child_create_response:
                department_modal.get_by_role("button", name="Save", exact=True).click()
        assert child_create_response.value.status == 201
        assert child_tree_response.value.status == 200
        root_tree_node = page.locator(".ant-tree-treenode").filter(has_text=root_name).first
        if root_tree_node.get_attribute("aria-expanded") == "false":
            root_tree_node.locator(".ant-tree-switcher").click()
        page.get_by_text(child_name, exact=False).click()
        move_button = page.get_by_role("button", name="Move", exact=True)
        move_button.focus()
        page.keyboard.press("Enter")
        move_dialog = page.get_by_role("dialog", name="Move department")
        move_dialog.wait_for()
        move_dialog.locator('input[role="spinbutton"]').fill("10")
        with page.expect_response(lambda response: "/admin-api/v1/org/units/" in response.url and response.url.endswith("/move") and response.request.method == "POST") as move_response:
            page.get_by_role("button", name="OK", exact=True).click()
        assert move_response.value.status == 200, move_response.value.text()
        page.get_by_text("Moved successfully", exact=True).wait_for()
        assert_no_horizontal_overflow(page)
        run_axe(page, "departments.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-departments.en-US.1440.png", full_page=True)
        for width in (1024, 768, 375):
            page.set_viewport_size({"width": width, "height": 900 if width > 375 else 812})
            assert_no_horizontal_overflow(page)
            if width == 375:
                run_axe(page, "departments.en-US.375", evidence)
            page.screenshot(path=OUTPUT / f"admin-departments.en-US.{width}.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})

        page.get_by_role("link", name="Positions", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/users/positions")
        page.get_by_role("heading", name="Positions", exact=True).wait_for()
        page.get_by_role("button", name="Create position", exact=True).click()
        position_modal = page.locator(".ant-modal:visible")
        position_modal.get_by_label("Position code").fill(f"DEV-{org_suffix}")
        position_modal.get_by_label("Position name").fill(f"Developer {org_suffix}")
        position_modal.get_by_label("Description").fill("E2E organization contract fixture")
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/org/positions") and response.request.method == "POST") as position_create_response:
            position_modal.get_by_role("button", name="Save", exact=True).click()
        assert position_create_response.value.status == 201
        page.get_by_text(f"Developer {org_suffix}", exact=True).wait_for()
        run_axe(page, "positions.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-positions.en-US.1440.png", full_page=True)
        for width in (1024, 768, 375):
            page.set_viewport_size({"width": width, "height": 900 if width > 375 else 812})
            assert_no_horizontal_overflow(page)
            if width == 375:
                run_axe(page, "positions.en-US.375", evidence)
            page.screenshot(path=OUTPUT / f"admin-positions.en-US.{width}.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})

        managed_email = f"managed-{org_suffix}@example.test"
        managed_name = f"Managed {org_suffix}"
        page.get_by_role("link", name="Users", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/users/accounts")
        page.get_by_role("heading", name="Users", exact=True).wait_for()
        page.get_by_role("button", name="Create user", exact=True).click()
        user_drawer = page.locator(".ant-drawer:visible")
        user_drawer.get_by_label("Email", exact=True).fill(managed_email)
        user_drawer.get_by_label("Display name", exact=True).fill(managed_name)
        user_drawer.get_by_label("Temporary password", exact=True).fill("Managed temporary password 2026!")
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/users")
            and response.request.method == "POST"
        ) as user_create_response:
            user_drawer.get_by_role("button", name="Save", exact=True).click()
        assert user_create_response.value.status == 201, user_create_response.value.text()
        page.get_by_text(managed_name, exact=True).wait_for()
        user_search = page.get_by_label("Search name, email, or username", exact=True)
        user_search.fill(managed_name)
        page.wait_for_url(lambda url: "/system/users/accounts?q=" in url)
        page.get_by_text(managed_name, exact=True).click()
        page.wait_for_url(lambda url: "/system/users/accounts/" in url)
        page.get_by_role("heading", name=managed_name, exact=True).wait_for()
        page.get_by_role("tab", name="Roles and organization", exact=True).click()
        role_card = page.locator(".ant-card").filter(has=page.get_by_text("Roles", exact=True)).first
        role_card.get_by_label("Roles", exact=True).click()
        role_option = page.locator(".ant-select-dropdown:visible .ant-select-item-option").first
        role_option.wait_for()
        role_option.click()
        page.keyboard.press("Escape")
        with page.expect_response(
            lambda response: response.url.endswith("/roles") and response.request.method == "PUT"
        ) as role_replace_response:
            role_card.get_by_role("button", name="Save", exact=True).click()
        assert role_replace_response.value.status == 200, role_replace_response.value.text()
        assignment_card = page.locator(".ant-card").filter(has=page.get_by_text("Departments and positions", exact=True)).first
        department_select = assignment_card.get_by_label("Department", exact=True)
        department_select.click()
        department_select.fill(root_name)
        department_select.press("Enter")
        page.keyboard.press("Escape")
        position_select = assignment_card.get_by_label("Positions", exact=True)
        position_select.click()
        position_select.fill(f"Developer {org_suffix}")
        position_select.press("Enter")
        page.keyboard.press("Escape")
        with page.expect_response(
            lambda response: "/admin-api/v1/org/users/" in response.url
            and response.url.endswith("/assignments")
            and response.request.method == "PUT"
        ) as assignment_replace_response:
            assignment_card.get_by_role("button", name="Save", exact=True).click()
        assert assignment_replace_response.value.status == 200, assignment_replace_response.value.text()
        page.get_by_role("tab", name="Login sessions", exact=True).click()
        page.get_by_text("No login sessions", exact=True).wait_for()
        page.wait_for_timeout(500)
        run_axe(page, "user-detail.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-user-detail.en-US.1440.png", full_page=True)
        page.go_back()
        page.wait_for_url(lambda url: "/system/users/accounts?q=" in url)
        assert page.get_by_label("Search name, email, or username", exact=True).input_value() == managed_name
        page.get_by_text(managed_name, exact=True).wait_for()

        bulk_one = f"Bulk One {org_suffix}"
        bulk_two = f"Bulk Two {org_suffix}"
        import_csv = (
            "email,display_name,locale,time_zone,temporary_password\n"
            f"bulk-one-{org_suffix}@example.test,{bulk_one},en-US,UTC,Bulk temporary password 2026!\n"
            f"bulk-two-{org_suffix}@example.test,{bulk_two},zh-CN,Asia/Shanghai,Bulk temporary password 2026!\n"
        )
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/users/import")
            and response.request.method == "POST"
        ) as user_import_response:
            page.locator('input[type="file"][accept*=".csv"]').set_input_files({
                "name": "users.csv",
                "mimeType": "text/csv",
                "buffer": import_csv.encode("utf-8"),
            })
        assert user_import_response.value.status == 200, user_import_response.value.text()
        assert user_import_response.value.json()["data"]["created"] == 2
        page.wait_for_timeout(500)
        user_search.fill(org_suffix)
        page.get_by_text(bulk_one, exact=True).wait_for()
        page.get_by_text(bulk_two, exact=True).wait_for()
        page.locator(".ant-table-row").filter(has_text=bulk_one).get_by_role("checkbox").check()
        page.locator(".ant-table-row").filter(has_text=bulk_two).get_by_role("checkbox").check()
        page.get_by_text("2 selected", exact=True).wait_for()
        status_start = len(user_status_responses)
        page.locator(".ak-bulk-bar").get_by_role("button", name="Disable", exact=True).click()
        bulk_dialog = page.get_by_role("dialog", name="Disable 2 users?")
        bulk_dialog.wait_for()
        bulk_dialog.get_by_role("button", name="Disable", exact=True).click()
        page.get_by_text("Updated 2 users", exact=True).wait_for()
        bulk_dialog.wait_for(state="detached")
        assert len(user_status_responses) >= status_start + 2
        assert [response.status for response in user_status_responses[-2:]] == [200, 200]
        with page.expect_download() as export_download:
            page.get_by_role("button", name="Export CSV", exact=True).click()
        assert export_download.value.suggested_filename == "users.csv"
        assert_no_horizontal_overflow(page)
        run_axe(page, "users.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-users.en-US.1440.png", full_page=True)
        for width in (1024, 768, 375):
            page.set_viewport_size({"width": width, "height": 900 if width > 375 else 812})
            page.wait_for_timeout(250)
            assert_no_horizontal_overflow(page)
            if width == 375:
                run_axe(page, "users.en-US.375", evidence)
            page.screenshot(path=OUTPUT / f"admin-users.en-US.{width}.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})
        choose_locale(page, "Display language", "简体中文")
        page.get_by_role("heading", name="用户", exact=True).wait_for()
        assert page.get_by_text("已选择 0 项", exact=True).count() == 0
        run_axe(page, "users.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-users.zh-CN.1440.png", full_page=True)
        choose_locale(page, "显示语言", "English")

        tenant_suffix = uuid.uuid4().hex[:8]
        target_tenant_name = f"Workspace {tenant_suffix}"
        target_tenant_code = f"workspace-{tenant_suffix}"
        page.get_by_role("link", name="Tenants", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/users/tenants")
        page.get_by_role("heading", name="Tenants", exact=True).wait_for()
        tenant_search = page.get_by_label("Search name or code", exact=True)
        tenant_search.fill(SOURCE_TENANT_CODE)
        page.wait_for_url(lambda url: f"/system/users/tenants?q={SOURCE_TENANT_CODE}" in url)
        current_tenant_name = page.locator(".ant-table-row").first.get_by_role("link").first.inner_text()
        page.locator(".ant-table-row").first.get_by_role("link").first.click()
        page.wait_for_url(lambda url: "/system/users/tenants/" in url)
        page.get_by_role("tab", name="Members", exact=True).click()
        page.get_by_text(EMAIL, exact=True).wait_for()
        run_axe(page, "tenant-detail.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-tenant-detail.en-US.1440.png", full_page=True)
        page.get_by_role("link", name="Back to tenants", exact=True).click()
        page.wait_for_url(lambda url: f"/system/users/tenants?q={SOURCE_TENANT_CODE}" in url)
        assert tenant_search.input_value() == SOURCE_TENANT_CODE
        tenant_search.fill("")
        page.get_by_role("button", name="Create tenant", exact=True).click()
        page.get_by_label("Tenant code", exact=True).fill(target_tenant_code)
        page.get_by_label("Tenant name", exact=True).fill(target_tenant_name)
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/tenants")
            and response.request.method == "POST"
        ) as tenant_create_response:
            page.get_by_role("button", name="Create", exact=True).click()
        assert tenant_create_response.value.status == 201, tenant_create_response.value.text()
        page.get_by_text(
            f"Tenant {target_tenant_name} was created and is available from the workspace switcher",
            exact=True,
        ).wait_for()
        assert_no_horizontal_overflow(page)
        run_axe(page, "tenants.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-tenants.en-US.1440.png", full_page=True)
        for width in (1024, 768, 375):
            page.set_viewport_size({"width": width, "height": 900 if width > 375 else 812})
            page.wait_for_timeout(250)
            assert_no_horizontal_overflow(page)
            if width == 375:
                run_axe(page, "tenants.en-US.375", evidence)
            page.screenshot(path=OUTPUT / f"admin-tenants.en-US.{width}.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})
        tenant_time_origin = page.evaluate("performance.timeOrigin")
        switcher = page.get_by_label("Switch workspace", exact=True)
        switcher.click()
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/auth/switch-tenant")
            and response.request.method == "POST"
        ) as tenant_switch_response:
            page.get_by_role("option", name=target_tenant_name, exact=True).click()
        assert tenant_switch_response.value.status == 200, tenant_switch_response.value.text()
        page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))
        page.get_by_role("heading", name="Key metrics", exact=True).wait_for()
        assert page.evaluate("performance.timeOrigin") == tenant_time_origin
        assert page.locator(".ak-tenant-context").get_by_text(target_tenant_name, exact=True).is_visible()
        page.get_by_label("Switch workspace", exact=True).click()
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/auth/switch-tenant")
            and response.request.method == "POST"
        ) as tenant_switch_back_response:
            page.get_by_role("option", name=current_tenant_name, exact=True).click()
        assert tenant_switch_back_response.value.status == 200, tenant_switch_back_response.value.text()
        page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))
        page.get_by_label("Switch workspace", exact=True).wait_for()
        choose_locale(page, "Display language", "简体中文")
        open_system_page(page, "/system/users/tenants", "用户管理")
        page.wait_for_url(f"{BASE_URL}/system/users/tenants")
        page.get_by_role("heading", name="租户", exact=True).wait_for()
        run_axe(page, "tenants.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-tenants.zh-CN.1440.png", full_page=True)
        choose_locale(page, "显示语言", "English")

        access_suffix = uuid.uuid4().hex[:8]
        role_name = f"Support {access_suffix}"
        role_code = f"support.{access_suffix}"
        page.get_by_role("link", name="Roles", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/access/roles")
        page.get_by_role("heading", name="Roles", exact=True).wait_for()
        page.get_by_role("button", name="Create role", exact=True).click()
        page.get_by_label("Role code", exact=True).fill(role_code)
        page.get_by_label("Role name", exact=True).fill(role_name)
        page.get_by_label("Description", exact=True).fill("E2E custom role")
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/roles")
            and response.request.method == "POST"
        ) as role_create_response:
            page.get_by_role("button", name="Create", exact=True).click()
        assert role_create_response.value.status == 201, role_create_response.value.text()
        role_search = page.get_by_label("Search role name or code", exact=True)
        role_search.fill(role_code)
        page.wait_for_url(lambda url: f"/system/access/roles?q={role_code}" in url)
        page.get_by_role("link", name=role_name, exact=True).click()
        page.wait_for_url(lambda url: "/system/access/roles/" in url)
        page.get_by_role("tab", name="Permissions", exact=True).click()
        permission_option = page.locator(".ak-permission-grid .ant-checkbox-wrapper").filter(has_text="iam.user.read")
        permission_option.click()
        expect(permission_option.get_by_role("checkbox")).to_be_checked()
        with page.expect_response(
            lambda response: "/admin-api/v1/roles/" in response.url
            and response.url.endswith("/permissions")
            and response.request.method == "PUT"
        ) as role_permissions_response:
            page.locator(".ant-card").filter(has=page.get_by_text("Permissions", exact=True)).get_by_role("button", name="Save", exact=True).click()
        assert role_permissions_response.value.status == 200, role_permissions_response.value.text()
        page.get_by_role("tab", name="Menus", exact=True).click()
        page.locator(".ant-tree-checkbox").first.click()
        with page.expect_response(
            lambda response: "/admin-api/v1/roles/" in response.url
            and response.url.endswith("/menus")
            and response.request.method == "PUT"
        ) as role_menus_response:
            page.locator(".ant-card").filter(has=page.get_by_text("Menus", exact=True)).get_by_role("button", name="Save", exact=True).click()
        assert role_menus_response.value.status == 200, role_menus_response.value.text()
        page.get_by_role("tab", name="Data scope", exact=True).click()
        page.get_by_role("radio", name="Custom units", exact=True).check()
        scope_units = page.get_by_label("Custom organization units", exact=True)
        scope_units.click()
        scope_units.press_sequentially(root_name)
        expect(page.locator(".ant-select-dropdown:visible")).to_be_visible()
        page.locator(".ant-select-dropdown:visible .ant-select-item-option").first.click()
        expect(page.locator(".ant-select-selection-item").filter(has_text=root_name)).to_be_visible()
        page.get_by_role("heading", name=role_name, exact=True).click()
        expect(page.locator(".ant-select-dropdown:visible")).to_have_count(0)
        with page.expect_response(
            lambda response: "/admin-api/v1/roles/" in response.url
            and response.url.endswith("/data-scope")
            and response.request.method == "PUT"
        ) as role_scope_response:
            page.locator(".ant-card").filter(has=page.get_by_text("Data scope", exact=True)).get_by_role("button", name="Save", exact=True).click()
        assert role_scope_response.value.status == 200, role_scope_response.value.text()
        assert_no_horizontal_overflow(page)
        run_axe(page, "role-detail.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-role-detail.en-US.1440.png", full_page=True)
        page.get_by_role("link", name="Back to roles", exact=True).click()
        page.wait_for_url(lambda url: f"/system/access/roles?q={role_code}" in url)
        assert role_search.input_value() == role_code
        role_search.fill("")
        run_axe(page, "roles.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-roles.en-US.1440.png", full_page=True)
        for width in (1024, 768, 375):
            page.set_viewport_size({"width": width, "height": 900 if width > 375 else 812})
            page.wait_for_timeout(250)
            assert_no_horizontal_overflow(page)
            if width == 375:
                run_axe(page, "roles.en-US.375", evidence)
            page.screenshot(path=OUTPUT / f"admin-roles.en-US.{width}.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})
        page.get_by_role("link", name="Permission Catalog", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/access/permissions")
        page.get_by_role("heading", name="Permission Catalog", exact=True).wait_for()
        page.get_by_label("Search permission code or name", exact=True).fill("iam.user.read")
        page.get_by_text("iam.user.read", exact=True).wait_for()
        assert_no_horizontal_overflow(page)
        run_axe(page, "permissions.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-permissions.en-US.1440.png", full_page=True)
        custom_menu_code = f"custom.page.{access_suffix}"
        page.get_by_role("link", name="Menus", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/access/menus")
        page.get_by_role("heading", name="Menus", exact=True).wait_for()
        page.get_by_role("button", name="Create menu", exact=True).click()
        page.get_by_label("Menu code", exact=True).fill(custom_menu_code)
        page.get_by_label("Title", exact=True).fill(f"Custom {access_suffix}")
        page.get_by_label("Translation key", exact=True).fill("menu.dashboard")
        page.get_by_label("Menu type", exact=True).click()
        page.locator('.ant-select-dropdown:visible .ant-select-item-option[title="Page"]').click()
        page.get_by_label("Route path", exact=True).fill(f"/custom/{access_suffix}")
        component_select = page.get_by_label("Static component", exact=True)
        component_select.click()
        component_select.press_sequentially("dashboard")
        page.locator('.ant-select-dropdown:visible .ant-select-item-option[title="dashboard — /dashboard"]').click()
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/menus")
            and response.request.method == "POST"
        ) as menu_create_response:
            page.get_by_role("button", name="Save", exact=True).click()
        assert menu_create_response.value.status == 201, menu_create_response.value.text()
        page.locator(".ant-drawer").wait_for(state="detached")
        page.get_by_label("Search title or code", exact=True).fill(custom_menu_code)
        page.get_by_text(custom_menu_code, exact=True).wait_for()
        assert_no_horizontal_overflow(page)
        run_axe(page, "menus.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-menus.en-US.1440.png", full_page=True)
        for width in (1024, 768, 375):
            page.set_viewport_size({"width": width, "height": 900 if width > 375 else 812})
            page.wait_for_timeout(250)
            assert_no_horizontal_overflow(page)
            if width == 375:
                run_axe(page, "menus.en-US.375", evidence)
            page.screenshot(path=OUTPUT / f"admin-menus.en-US.{width}.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})
        choose_locale(page, "Display language", "简体中文")
        page.get_by_role("heading", name="菜单", exact=True).wait_for()
        run_axe(page, "menus.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-menus.zh-CN.1440.png", full_page=True)
        open_system_page(page, "/system/access/roles", "权限设置")
        page.get_by_role("heading", name="角色", exact=True).wait_for()
        run_axe(page, "roles.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-roles.zh-CN.1440.png", full_page=True)
        choose_locale(page, "显示语言", "English")

        page.get_by_role("link", name="Operation Logs", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/security/operation-logs")
        page.get_by_role("heading", name="Operation Logs", exact=True).wait_for()
        operation_search = page.get_by_label("Search request, action, event, or safe identifier hint", exact=True)
        operation_search.fill(AUDIT_REQUEST_ID)
        page.get_by_text("audit.e2e.fixture", exact=True).wait_for()
        assert f"q={AUDIT_REQUEST_ID}" in page.url
        page.get_by_role("button", name="View details", exact=True).click()
        page.get_by_text("Operation details", exact=True).wait_for()
        expect(page.locator(".ak-audit-json pre").first).to_contain_text("[REDACTED]")
        page.wait_for_timeout(350)
        run_axe(page, "operation-logs.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-operation-logs.en-US.1440.png", full_page=True)
        page.keyboard.press("Escape")
        page.locator(".ant-drawer").wait_for(state="detached")
        page.set_viewport_size({"width": 375, "height": 812})
        page.wait_for_timeout(250)
        assert_no_horizontal_overflow(page)
        run_axe(page, "operation-logs.en-US.375", evidence)
        page.screenshot(path=OUTPUT / "admin-operation-logs.en-US.375.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})

        page.get_by_role("link", name="Login Logs", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/security/login-logs")
        page.get_by_role("heading", name="Login Logs", exact=True).wait_for()
        page.get_by_label("Result", exact=True).click()
        page.locator('.ant-select-dropdown:visible .ant-select-item-option[title="Failure"]').click()
        page.wait_for_url(lambda url: "/system/security/login-logs?" in url and "result=failure" in url)
        page.locator(".ak-audit-result-failure").get_by_text("Failure", exact=True).first.wait_for()
        page.wait_for_timeout(350)
        assert_no_horizontal_overflow(page)
        run_axe(page, "login-logs.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-login-logs.en-US.1440.png", full_page=True)

        page.get_by_role("link", name="Security Events", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/security/events")
        page.get_by_role("heading", name="Security Events", exact=True).wait_for()
        security_search = page.get_by_label("Search request, action, event, or safe identifier hint", exact=True)
        security_search.fill(AUDIT_EVENT_TYPE)
        event_link = page.get_by_role("link", name=AUDIT_EVENT_TYPE, exact=True)
        event_link.wait_for()
        run_axe(page, "security-events.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-security-events.en-US.1440.png", full_page=True)
        event_link.click()
        page.wait_for_url(f"{BASE_URL}/system/security/events/{AUDIT_EVENT_ID}")
        page.get_by_role("heading", name="Security Event Details", exact=True).wait_for()
        expect(page.locator(".ak-audit-json pre").first).to_contain_text("[REDACTED]")
        run_axe(page, "security-event-detail.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-security-event-detail.en-US.1440.png", full_page=True)
        with page.expect_response(
            lambda response: response.url.endswith(f"/admin-api/v1/audit/security-events/{AUDIT_EVENT_ID}/resolve")
            and response.request.method == "POST"
        ) as audit_resolve_response:
            page.get_by_role("button", name="Resolve event", exact=True).click()
            page.locator(".ant-modal-confirm-title:visible", has_text="Resolve this security event?").wait_for()
            page.locator(".ant-modal-confirm-btns .ant-btn-primary:visible").click()
        assert audit_resolve_response.value.status == 200, audit_resolve_response.value.text()
        page.get_by_text("Security event resolved and audited.", exact=True).wait_for()
        expect(page.get_by_role("button", name="Resolve event", exact=True)).to_have_count(0)
        page.get_by_role("link", name="Back to security events", exact=True).click()
        page.wait_for_url(lambda url: "/system/security/events?" in url and f"q={AUDIT_EVENT_TYPE}" in url)
        security_search = page.get_by_label("Search request, action, event, or safe identifier hint", exact=True)
        assert security_search.input_value() == AUDIT_EVENT_TYPE
        page.locator(".ak-status-success").get_by_text("Resolved", exact=True).wait_for()
        page.set_viewport_size({"width": 375, "height": 812})
        page.wait_for_timeout(250)
        assert_no_horizontal_overflow(page)
        run_axe(page, "security-events.en-US.375", evidence)
        page.screenshot(path=OUTPUT / "admin-security-events.en-US.375.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})
        choose_locale(page, "Display language", "简体中文")
        page.get_by_role("heading", name="安全事件", exact=True).wait_for()
        run_axe(page, "security-events.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-security-events.zh-CN.1440.png", full_page=True)
        choose_locale(page, "显示语言", "English")

        page.get_by_role("link", name="Positions", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/users/positions")
        choose_locale(page, "Display language", "简体中文")
        page.get_by_role("heading", name="岗位", exact=True).wait_for()
        run_axe(page, "positions.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-positions.zh-CN.1440.png", full_page=True)
        page.get_by_role("link", name="部门", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/system/users/departments")
        page.get_by_role("heading", name="部门", exact=True).wait_for()
        run_axe(page, "departments.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-departments.zh-CN.1440.png", full_page=True)
        choose_locale(page, "显示语言", "English")
        page.get_by_role("link", name=DISPLAY_NAME, exact=True).click()
        page.wait_for_url(f"{BASE_URL}/profile/basic")

        secondary_user_agent = f"AppKernia E2E Secondary Session {os.getpid()}"
        secondary = browser.new_context(
            locale="en-US",
            user_agent=secondary_user_agent,
            viewport={"width": 1024, "height": 768},
        )
        secondary_page = secondary.new_page()
        secondary_page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        secondary_page.get_by_label("Account").fill(EMAIL)
        secondary_page.get_by_label("Password").fill(PASSWORD)
        secondary_page.locator('button[type="submit"]').click()
        secondary_page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))

        page.get_by_role("link", name="Security settings", exact=True).click()
        page.wait_for_url(f"{BASE_URL}/profile/security")
        page.get_by_role("heading", name="Security Settings").wait_for()
        devices_section = page.locator('section[aria-labelledby="registered-devices-title"]')
        sessions_section = page.locator('section[aria-labelledby="active-sessions-title"]')
        secondary_device_card = devices_section.locator(".ak-session-card").filter(has_text=secondary_user_agent)
        secondary_device_card.wait_for()
        current_device_card = devices_section.locator(".ak-session-card").filter(
            has=page.get_by_text("Current device", exact=True)
        )
        assert current_device_card.count() == 1
        current_device_card.get_by_role("button", name="Remove device").click()
        page.get_by_text(
            "Removing this current device will revoke all of its sessions and sign you out immediately. Continue?",
            exact=True,
        ).wait_for()
        page.get_by_role("button", name="Cancel", exact=True).click()
        page.wait_for_timeout(350)
        secondary_card = sessions_section.locator(".ak-session-card").filter(has_text=secondary_user_agent)
        secondary_card.wait_for()
        current_card = sessions_section.locator(".ak-session-card").filter(
            has=page.get_by_text("Current session", exact=True)
        )
        assert current_card.count() == 1
        current_card.get_by_role("button", name="Revoke session").click()
        page.get_by_text("Revoking this current session will sign you out immediately. Continue?", exact=True).wait_for()
        page.get_by_role("button", name="Cancel", exact=True).click()
        page.wait_for_timeout(350)
        run_axe(page, "profile-security.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-profile-security.en-US.1440.png", full_page=True)
        for width in (1024, 768, 375):
            page.set_viewport_size({"width": width, "height": 900 if width > 375 else 812})
            assert_no_horizontal_overflow(page)
            page.screenshot(path=OUTPUT / f"admin-profile-security.en-US.{width}.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})
        assert page.get_by_label("Current password", exact=True).get_attribute("autocomplete") == "current-password"
        assert page.get_by_label("New password", exact=True).get_attribute("autocomplete") == "new-password"
        assert page.get_by_label("Confirm new password", exact=True).get_attribute("autocomplete") == "new-password"
        page.get_by_label("Current password", exact=True).fill("wrong-current-password")
        page.get_by_label("New password", exact=True).fill("new-admin-password-2026")
        page.get_by_label("Confirm new password", exact=True).fill("new-admin-password-2026")
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/me/password/change")
            and response.request.method == "POST"
        ) as password_response:
            page.get_by_role("button", name="Change password").click()
        assert password_response.value.status == 422
        page.get_by_text("The current password is incorrect.", exact=True).wait_for()
        secondary_card.get_by_role("button", name="Revoke session").click()
        page.get_by_text("Revoke this session? It will no longer be able to access Admin.", exact=True).wait_for()
        with page.expect_response(
            lambda response: "/admin-api/v1/me/sessions/" in response.url
            and response.request.method == "DELETE"
        ) as revoke_response:
            page.locator(".ant-popconfirm-buttons .ant-btn-primary:visible").click()
        assert revoke_response.value.status == 200
        secondary_card.wait_for(state="detached")
        page.get_by_text("The session has been revoked.", exact=True).wait_for()
        secondary_device_card.get_by_role("button", name="Remove device").click()
        page.get_by_text("Remove this device? All sessions associated with it will be revoked.", exact=True).wait_for()
        with page.expect_response(
            lambda response: "/admin-api/v1/me/devices/" in response.url
            and response.request.method == "DELETE"
        ) as remove_device_response:
            page.locator(".ant-popconfirm-buttons .ant-btn-primary:visible").click()
        assert remove_device_response.value.status == 200
        secondary_device_card.wait_for(state="detached")
        page.get_by_text("The device was removed and its sessions were revoked.", exact=True).wait_for()
        secondary.close()
        choose_locale(page, "Display language", "简体中文")
        page.get_by_role("heading", name="安全设置").wait_for()
        run_axe(page, "profile-security.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-profile-security.zh-CN.1440.png", full_page=True)
        choose_locale(page, "显示语言", "English")
        page.get_by_role("link", name="Dashboard", exact=True).click()
        page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))

        page.set_viewport_size({"width": 1024, "height": 900})
        page.wait_for_timeout(350)
        assert_no_horizontal_overflow(page)
        page.screenshot(path=OUTPUT / "admin-dashboard.en-US.1024.png", full_page=True)

        page.set_viewport_size({"width": 375, "height": 812})
        page.wait_for_timeout(350)
        assert_no_horizontal_overflow(page)
        page.get_by_label("Open navigation").click()
        page.locator(".ant-drawer-content-wrapper").wait_for(state="visible")
        page.wait_for_timeout(350)
        page.screenshot(path=OUTPUT / "admin-dashboard.en-US.375-navigation.png", full_page=True)
        page.keyboard.press("Escape")

        monitoring_user_agent = f"AppKernia E2E Monitoring Session {os.getpid()}"
        monitoring = browser.new_context(locale="en-US", user_agent=monitoring_user_agent, viewport={"width": 1024, "height": 768})
        monitoring_page = monitoring.new_page()
        monitoring_page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        monitoring_page.get_by_label("Account").fill(EMAIL)
        monitoring_page.get_by_label("Password").fill(PASSWORD)
        monitoring_page.locator('button[type="submit"]').click()
        monitoring_page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))

        page.set_viewport_size({"width": 1440, "height": 900})
        page.get_by_role("menuitem", name="Online Sessions", exact=True).get_by_role("link").click()
        page.wait_for_url(f"{BASE_URL}/system/monitoring/sessions")
        page.get_by_role("heading", name="Online Sessions", exact=True).wait_for()
        page.get_by_text("This browser's current session is marked below. Forcing it offline will sign you out immediately.", exact=True).wait_for()
        session_search = page.get_by_label("Search user name or account", exact=True)
        session_search.fill(DISPLAY_NAME)
        page.wait_for_url(lambda url: "/system/monitoring/sessions?" in url and "q=" in url)
        target_session_row = page.locator("tbody tr").filter(
            has=page.get_by_role("button", name="Force offline", exact=True),
            has_not_text="Current session",
        ).first
        target_session_row.wait_for()
        target_session_id = target_session_row.get_attribute("data-row-key")
        assert target_session_id
        target_session_row.get_by_text("Unknown", exact=True).wait_for()
        page.locator("tr").filter(has=page.get_by_text("Current session", exact=True)).wait_for()
        run_axe(page, "online-sessions.en-US.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-online-sessions.en-US.1440.png", full_page=True)
        target_session_row.get_by_role("button", name="Force offline", exact=True).click()
        page.locator(".ant-modal-confirm-title:visible", has_text="Force this session offline?").wait_for()
        with page.expect_response(
            lambda response: "/admin-api/v1/online-sessions/" in response.url and response.request.method == "DELETE"
        ) as online_revoke_response:
            page.locator(".ant-modal-confirm-btns .ant-btn-dangerous:visible").click()
        assert online_revoke_response.value.status == 200
        revoked_session_row = page.locator(f'tbody tr[data-row-key="{target_session_id}"]')
        revoked_session_row.get_by_text("Revoked", exact=True).wait_for()
        assert revoked_session_row.get_by_role("button", name="Force offline", exact=True).count() == 0
        page.get_by_text("Session forced offline and audit recorded.", exact=True).wait_for()
        page.set_viewport_size({"width": 375, "height": 812})
        page.wait_for_timeout(250)
        assert_no_horizontal_overflow(page)
        run_axe(page, "online-sessions.en-US.375", evidence)
        page.screenshot(path=OUTPUT / "admin-online-sessions.en-US.375.png", full_page=True)
        page.set_viewport_size({"width": 1440, "height": 900})
        choose_locale(page, "Display language", "简体中文")
        page.get_by_role("heading", name="在线会话", exact=True).wait_for()
        page.get_by_text("会话已强制下线并完成审计。", exact=True).wait_for()
        run_axe(page, "online-sessions.zh-CN.1440", evidence)
        page.screenshot(path=OUTPUT / "admin-online-sessions.zh-CN.1440.png", full_page=True)
        choose_locale(page, "显示语言", "English")
        monitoring.close()

        anonymous = browser.new_context(locale="zh-CN", viewport={"width": 768, "height": 900})
        anonymous_page = anonymous.new_page()
        anonymous_page.goto(f"{BASE_URL}/dashboard")
        anonymous_page.wait_for_url(lambda url: "/login" in url)
        anonymous_page.get_by_role("heading", name="登录 AppKernia").wait_for()
        anonymous_page.screenshot(path=OUTPUT / "admin-protected-redirect.zh-CN.768.png", full_page=True)
        anonymous_profile = anonymous.new_page()
        anonymous_profile.goto(f"{BASE_URL}/profile/basic")
        anonymous_profile.wait_for_url(lambda url: "/login" in url and "redirect=%2Fprofile%2Fbasic" in url)
        anonymous_profile.get_by_role("heading", name="登录 AppKernia").wait_for()
        anonymous_security = anonymous.new_page()
        anonymous_security.goto(f"{BASE_URL}/profile/security")
        anonymous_security.wait_for_url(lambda url: "/login" in url and "redirect=%2Fprofile%2Fsecurity" in url)
        anonymous_security.get_by_role("heading", name="登录 AppKernia").wait_for()
        anonymous_org = anonymous.new_page()
        anonymous_org.goto(f"{BASE_URL}/system/users/departments")
        anonymous_org.wait_for_url(lambda url: "/login" in url and "redirect=%2Fsystem%2Fusers%2Fdepartments" in url)
        anonymous_org.get_by_role("heading", name="登录 AppKernia").wait_for()
        anonymous_users = anonymous.new_page()
        anonymous_users.goto(f"{BASE_URL}/system/users/accounts")
        anonymous_users.wait_for_url(lambda url: "/login" in url and "redirect=%2Fsystem%2Fusers%2Faccounts" in url)
        anonymous_users.get_by_role("heading", name="登录 AppKernia").wait_for()
        anonymous_tenants = anonymous.new_page()
        anonymous_tenants.goto(f"{BASE_URL}/system/users/tenants")
        anonymous_tenants.wait_for_url(lambda url: "/login" in url and "redirect=%2Fsystem%2Fusers%2Ftenants" in url)
        anonymous_tenants.get_by_role("heading", name="登录 AppKernia").wait_for()
        anonymous_roles = anonymous.new_page()
        anonymous_roles.goto(f"{BASE_URL}/system/access/roles")
        anonymous_roles.wait_for_url(lambda url: "/login" in url and "redirect=%2Fsystem%2Faccess%2Froles" in url)
        anonymous_roles.get_by_role("heading", name="登录 AppKernia").wait_for()
        anonymous_audit = anonymous.new_page()
        anonymous_audit.goto(f"{BASE_URL}/system/security/events")
        anonymous_audit.wait_for_url(lambda url: "/login" in url and "redirect=%2Fsystem%2Fsecurity%2Fevents" in url)
        anonymous_audit.get_by_role("heading", name="登录 AppKernia").wait_for()
        anonymous_sessions = anonymous.new_page()
        anonymous_sessions.goto(f"{BASE_URL}/system/monitoring/sessions")
        anonymous_sessions.wait_for_url(lambda url: "/login" in url and "redirect=%2Fsystem%2Fmonitoring%2Fsessions" in url)
        anonymous_sessions.get_by_role("heading", name="登录 AppKernia").wait_for()
        anonymous.close()

        reduced = browser.new_context(locale="zh-CN", reduced_motion="reduce", viewport={"width": 1024, "height": 768})
        reduced_page = reduced.new_page()
        reduced_page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        transition_seconds = reduced_page.locator('button[type="submit"]').evaluate(
            "element => Number.parseFloat(getComputedStyle(element).transitionDuration) || 0"
        )
        assert transition_seconds <= 0.001
        reduced.close()

        persisted = browser.new_context(locale="zh-CN", viewport={"width": 1024, "height": 768})
        persisted_page = persisted.new_page()
        persisted_page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        persisted_page.get_by_label("账号").fill(EMAIL)
        persisted_page.get_by_label("密码").fill(PASSWORD)
        persisted_page.locator('button[type="submit"]').click()
        persisted_page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))
        persisted_page.get_by_text("Current workspace", exact=True).wait_for()
        assert persisted_page.locator("html").get_attribute("lang") == "en-US"
        persisted.close()

        evidence["browser"] = {"engine": "Chromium", "viewports": [375, 768, 1024, 1440]}
        evidence["profile_basic"] = {"read": True, "save": True, "restored_fixture": True, "locales": ["zh-CN", "en-US"]}
        evidence["profile_avatar"] = {
            "feature_gate": True,
            "invalid_file_localized": True,
            "upload_session_status": avatar_session_response.value.status,
            "upload_status": avatar_upload_response.value.status,
            "private_content_status": avatar_content_response.status,
            "private_content_type": avatar_content_response.headers.get("content-type"),
            "file_id": avatar_file_id,
            "blob_rendered": True,
            "locales": ["zh-CN", "en-US"],
        }
        evidence["profile_self_only"] = {"anonymous_direct_url_redirected": True}
        evidence["profile_security_sessions"] = {
            "list": True,
            "current_session_warning": True,
            "other_session_revoked": True,
            "locales": ["zh-CN", "en-US"],
        }
        evidence["profile_security_devices"] = {
            "list": True,
            "current_device_warning": True,
            "other_device_removed": True,
            "associated_sessions_revoked": "backend_integration",
            "locales": ["zh-CN", "en-US"],
        }
        evidence["profile_password"] = {
            "password_manager_autocomplete": True,
            "invalid_current_password_localized": True,
            "successful_change": "backend_integration_only",
        }
        evidence["organization"] = {
            "unit_create_statuses": [root_create_response.value.status, child_create_response.value.status],
            "keyboard_move_status": move_response.value.status,
            "position_create_status": position_create_response.value.status,
            "anonymous_direct_url_redirected": True,
            "locales": ["zh-CN", "en-US"],
        }
        evidence["users"] = {
            "create_status": user_create_response.value.status,
            "role_replace_status": role_replace_response.value.status,
            "assignment_replace_status": assignment_replace_response.value.status,
            "import_status": user_import_response.value.status,
            "import_created": user_import_response.value.json()["data"]["created"],
            "bulk_disable_statuses": [response.status for response in user_status_responses[-2:]],
            "export_download": export_download.value.suggested_filename,
            "url_search_restored_after_detail": True,
            "anonymous_direct_url_redirected": True,
            "locales": ["zh-CN", "en-US"],
        }
        evidence["tenants"] = {
            "create_status": tenant_create_response.value.status,
            "switch_statuses": [tenant_switch_response.value.status, tenant_switch_back_response.value.status],
            "url_search_restored_after_detail": True,
            "switch_without_reload": True,
            "anonymous_direct_url_redirected": True,
            "locales": ["zh-CN", "en-US"],
        }
        evidence["access_control"] = {
            "role_create_status": role_create_response.value.status,
            "permission_replace_status": role_permissions_response.value.status,
            "menu_replace_status": role_menus_response.value.status,
            "data_scope_replace_status": role_scope_response.value.status,
            "menu_create_status": menu_create_response.value.status,
            "permission_menu_separation": True,
            "component_key_selector": True,
            "url_search_restored_after_detail": True,
            "anonymous_direct_url_redirected": True,
            "depth_and_cycle_rejection": "backend_integration",
            "locales": ["zh-CN", "en-US"],
        }
        evidence["audit_security"] = {
            "operation_redaction_fixture": True,
            "operation_url_filters": True,
            "login_result_url_filter": True,
            "security_event_detail": True,
            "resolve_status": audit_resolve_response.value.status,
            "resolve_audited": True,
            "security_url_restored_after_detail": True,
            "anonymous_direct_url_redirected": True,
            "locales": ["zh-CN", "en-US"],
        }
        evidence["online_sessions"] = {
            "safe_server_hints": True,
            "current_session_warning": True,
            "url_filters": True,
            "revoke_status": online_revoke_response.value.status,
            "list_refreshed_after_revoke": True,
            "refresh_family_and_audit": "backend_integration",
            "anonymous_direct_url_redirected": True,
            "locales": ["zh-CN", "en-US"],
        }
        evidence["account_enumeration"] = {"same_error": True}
        evidence["anonymous_auth"] = {
            "registration_feature_gate": True,
            "terms_not_preselected": True,
            "registration_postgresql": registration_response.value.status == 202,
            "forgot_known_unknown_same_response": known_forgot_response.value.json()["data"] == unknown_forgot_data,
            "reset_token_removed_from_url": True,
            "invalid_reset_localized": True,
            "successful_reset": "backend_postgresql_integration",
            "locales": ["zh-CN", "en-US"],
        }
        evidence["keyboard_navigation"] = {"language_then_account": True}
        evidence["locale_switch_without_reload"] = {"login": True, "dashboard": True}
        evidence["dashboard"] = {
            "postgresql_backed_metrics": True,
            "postgresql_backed_login_trends": {
                "success": trend_series["logins.success"],
                "failure": trend_series["logins.failure"],
            },
            "range_in_url": True,
            "permission_pruned_contract": "backend_unit_and_integration",
            "chart_table_alternative": True,
            "lazy_echarts_chunk": any("DashboardTrendChart" in url for url in loaded_scripts),
            "locales": ["zh-CN", "en-US"],
        }
        evidence["authenticated_locale_persisted"] = {"fresh_browser_session": True, "locale": "en-US"}
        evidence["reduced_motion"] = {"transition_seconds": transition_seconds}
        expected_auth_errors = [message for message in console_errors if "401 (Unauthorized)" in message]
        expected_validation_errors = [message for message in console_errors if "422 (Unprocessable Entity)" in message]
        unexpected_console_errors = [
            message for message in console_errors
            if "401 (Unauthorized)" not in message and "422 (Unprocessable Entity)" not in message
        ]
        evidence["expected_auth_401_console_entries"] = len(expected_auth_errors)
        evidence["expected_validation_422_console_entries"] = len(expected_validation_errors)
        evidence["unexpected_console_errors"] = unexpected_console_errors
        (OUTPUT / "admin-e2e-axe-results.json").write_text(
            json.dumps(evidence, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
        assert len(expected_auth_errors) == 2, f"unexpected authentication error count: {console_errors}"
        assert len(expected_validation_errors) == 2, f"unexpected validation error count: {console_errors}"
        assert not unexpected_console_errors, f"browser console errors: {unexpected_console_errors}"
        assert evidence["dashboard"]["lazy_echarts_chunk"] is True, "lazy ECharts chunk was not requested"
        avatar_fixture.unlink(missing_ok=True)
        avatar_invalid_fixture.unlink(missing_ok=True)
        context.close()
        browser.close()

    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
