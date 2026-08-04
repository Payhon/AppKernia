from __future__ import annotations

import base64
import hashlib
import hmac
import json
import os
import re
import struct
import time
from pathlib import Path
from typing import Any

from playwright.sync_api import Page, sync_playwright


ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "output" / "playwright"
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4173").rstrip("/")
EMAIL = os.environ["AK_E2E_EMAIL"]
PASSWORD = os.environ["AK_E2E_PASSWORD"]


def current_totp(secret: str) -> str:
    key = base64.b32decode(secret.upper() + "=" * ((8 - len(secret) % 8) % 8))
    counter = int(time.time()) // 30
    digest = hmac.new(key, struct.pack(">Q", counter), hashlib.sha1).digest()
    offset = digest[-1] & 0x0F
    value = struct.unpack(">I", digest[offset : offset + 4])[0] & 0x7FFFFFFF
    return f"{value % 1_000_000:06d}"


def audit(page: Page, name: str, evidence: dict[str, Any]) -> None:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate(
        "async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })"
    )
    violations = [
        {"id": item["id"], "impact": item["impact"], "nodes": len(item["nodes"])}
        for item in result["violations"]
    ]
    severe = [item for item in violations if item["impact"] in ("critical", "serious")]
    dimensions = page.evaluate(
        "() => ({ client: document.documentElement.clientWidth, scroll: document.documentElement.scrollWidth })"
    )
    evidence[name] = {
        "violations": violations,
        "critical_or_serious": len(severe),
        "dimensions": dimensions,
    }
    assert not severe, severe
    assert dimensions["scroll"] <= dimensions["client"], dimensions


def shot(page: Page, name: str, evidence: dict[str, Any]) -> None:
    audit(page, name, evidence)
    page.evaluate(
        "() => { if (document.activeElement instanceof HTMLElement) document.activeElement.blur(); document.querySelector('.ak-skip-link')?.blur() }"
    )
    page.wait_for_timeout(200)
    skip_link = page.locator(".ak-skip-link")
    if skip_link.count():
        skip_link.evaluate("element => { element.style.display = 'none' }")
    page.screenshot(path=OUTPUT / f"admin-identity-security.{name}.png", full_page=True)
    if skip_link.count():
        skip_link.evaluate("element => { element.style.display = '' }")


def login(page: Page) -> tuple[str, str]:
    page.goto(f"{BASE}/login", wait_until="networkidle")
    page.get_by_label("账号", exact=True).fill(EMAIL)
    page.get_by_label("密码", exact=True).fill(PASSWORD)
    with page.expect_response(
        lambda response: response.url.endswith("/admin-api/v1/auth/login")
        and response.request.method == "POST"
    ) as login_response:
        page.locator('button[type="submit"]').click()
    response = login_response.value
    assert response.status == 200, response.text()
    payload = response.json()["data"]
    page.wait_for_url(lambda value: value.startswith(f"{BASE}/dashboard"))
    if page.locator("html").get_attribute("lang") == "zh-CN":
        page.get_by_label("显示语言", exact=True).select_option(label="English")
    page.get_by_label("Display language", exact=True).wait_for()
    return payload["access_token"], payload["csrf_token"]


def open_security(page: Page) -> None:
    page.locator(".ak-user-name").click()
    page.wait_for_url(f"{BASE}/profile/basic")
    page.locator('.ak-profile-navigation a[href="/profile/security"]').click()
    page.wait_for_url(f"{BASE}/profile/security")
    page.get_by_role("heading", name="Security Settings", exact=True).wait_for()


def main() -> None:
    OUTPUT.mkdir(parents=True, exist_ok=True)
    evidence: dict[str, Any] = {}
    console_errors: list[str] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
        page = context.new_page()
        page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
        access_token, csrf_token = login(page)
        open_security(page)

        page.get_by_role("button", name="Set up authenticator", exact=True).click()
        enrollment = page.get_by_role("dialog", name="Set up authenticator", exact=True)
        enrollment.wait_for()
        secret = enrollment.locator("code").first.inner_text().strip()
        assert secret and " " not in secret
        code = current_totp(secret)
        secret = ""
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/me/mfa/totp/verify")
            and response.request.method == "POST"
        ) as verified:
            enrollment.get_by_label("6-digit code", exact=True).fill(code)
            enrollment.get_by_role("button", name="Verify and enable", exact=True).click()
        code = ""
        assert verified.value.status == 200, verified.value.text()
        recovery = page.get_by_role("dialog", name="One-time recovery codes", exact=True)
        recovery.wait_for()
        assert recovery.locator(".ak-recovery-code-list li").count() == 10
        recovery.get_by_role("checkbox", name="I saved the recovery codes in a secure location", exact=True).check()
        recovery.get_by_role("button", name="I saved them securely", exact=True).click()
        page.get_by_text("Enabled", exact=True).wait_for()
        shot(page, "security.en-US.1440.enabled", evidence)

        page.get_by_role("button", name="Regenerate recovery codes", exact=True).click()
        rotate = page.get_by_role("dialog", name="Regenerate recovery codes", exact=True)
        rotate.get_by_label("Verification details", exact=True).fill("incorrect-password-value")
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/me/mfa/recovery-codes/rotate")
            and response.request.method == "POST"
        ) as rejected_rotate:
            rotate.get_by_role("button", name="Confirm", exact=True).click()
        assert rejected_rotate.value.status == 403, rejected_rotate.value.text()
        rotate.get_by_label("Verification details", exact=True).fill(PASSWORD)
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/me/mfa/recovery-codes/rotate")
            and response.request.method == "POST"
        ) as accepted_rotate:
            rotate.get_by_role("button", name="Confirm", exact=True).click()
        assert accepted_rotate.value.status == 200, accepted_rotate.value.text()
        recovery = page.get_by_role("dialog", name="One-time recovery codes", exact=True)
        recovery.wait_for()
        assert recovery.locator(".ak-recovery-code-list li").count() == 10
        recovery.get_by_role("checkbox", name="I saved the recovery codes in a secure location", exact=True).check()
        recovery.get_by_role("button", name="I saved them securely", exact=True).click()

        page.locator('.ak-profile-navigation a[href="/profile/connections"]').click()
        page.wait_for_url(f"{BASE}/profile/connections")
        page.get_by_role("heading", name="Connected Accounts", exact=True).wait_for()
        shot(page, "connections.en-US.1440.empty", evidence)
        with page.expect_request(
            lambda request: request.url.endswith("/admin-api/v1/me/oauth/local/callback")
            and request.method == "POST"
        ) as callback_request:
            page.get_by_role("button", name="Connect account", exact=True).click()
        callback_payload = callback_request.value.post_data_json
        page.get_by_role("heading", name="Third-party account connected", exact=True).wait_for()
        assert page.url == f"{BASE}/auth/callback/local", page.url
        shot(page, "callback.en-US.1440.success", evidence)

        replay_response = context.request.post(
            f"{BASE}/admin-api/v1/me/oauth/local/callback",
            headers={
                "Authorization": f"Bearer {access_token}",
                "X-CSRF-Token": csrf_token,
                "Accept-Language": "en-US",
                "Content-Type": "application/json",
            },
            data=callback_payload,
        )
        assert replay_response.status == 422, replay_response.text()
        callback_payload = {}
        access_token = ""
        csrf_token = ""

        page.get_by_role("link", name="Return to connected accounts", exact=True).click()
        page.wait_for_url(f"{BASE}/profile/connections")
        page.get_by_text(re.compile(r"Account hint: local-[0-9a-f]{8}"), exact=True).wait_for()
        body = page.locator("body").inner_text()
        assert "authorization_code" not in body and "access_token" not in body and "subject" not in body.lower()
        page.get_by_role("button", name="Disconnect", exact=True).click()
        confirm = page.locator(".ant-popconfirm")
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/me/oauth/local")
            and response.request.method == "DELETE"
        ) as unbound:
            confirm.get_by_role("button", name="Disconnect", exact=True).click()
        assert unbound.value.status == 200, unbound.value.text()
        page.get_by_text("The third-party account was disconnected.", exact=True).wait_for()

        page.locator('.ak-profile-navigation a[href="/profile/security"]').click()
        page.wait_for_url(f"{BASE}/profile/security")
        page.get_by_role("button", name="Disable", exact=True).click()
        disable = page.get_by_role("dialog", name="Disable multi-factor authentication", exact=True)
        disable.get_by_label("Verification details", exact=True).fill(PASSWORD)
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/me/mfa/totp")
            and response.request.method == "DELETE"
        ) as disabled:
            disable.get_by_role("button", name="Confirm", exact=True).click()
        assert disabled.value.status == 200, disabled.value.text()
        page.get_by_text("Multi-factor authentication is disabled.", exact=True).wait_for()

        page.get_by_label("Display language", exact=True).select_option(label="简体中文")
        page.get_by_role("heading", name="安全设置", exact=True).wait_for()
        for width, height in ((1024, 768), (768, 1024), (375, 812)):
            page.set_viewport_size({"width": width, "height": height})
            page.wait_for_timeout(250)
            shot(page, f"security.zh-CN.{width}.disabled", evidence)

        page.locator('.ak-profile-navigation a[href="/profile/connections"]').click()
        page.wait_for_url(f"{BASE}/profile/connections")
        page.get_by_role("heading", name="第三方绑定", exact=True).wait_for()
        page.set_viewport_size({"width": 375, "height": 812})
        shot(page, "connections.zh-CN.375.empty", evidence)

        evidence["runtime"] = {
            "mfa_verify_status": verified.value.status,
            "rotate_wrong_proof_status": rejected_rotate.value.status,
            "rotate_status": accepted_rotate.value.status,
            "oauth_callback_replay_status": replay_response.status,
            "oauth_unbind_status": unbound.value.status,
            "mfa_disable_status": disabled.value.status,
            "one_time_values_persisted": False,
            "locales": ["zh-CN", "en-US"],
        }
        expected_forbidden_console = [
            message
            for message in console_errors
            if message == "Failed to load resource: the server responded with a status of 403 (Forbidden)"
        ]
        unexpected_console_errors = [message for message in console_errors if message not in expected_forbidden_console]
        evidence["expected_wrong_proof_console_errors"] = len(expected_forbidden_console)
        evidence["unexpected_console_errors"] = unexpected_console_errors
        assert len(expected_forbidden_console) == 1, console_errors
        assert not unexpected_console_errors, unexpected_console_errors
        (OUTPUT / "admin-identity-security-e2e-results.json").write_text(
            json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
        )
        context.close()
        browser.close()
    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
