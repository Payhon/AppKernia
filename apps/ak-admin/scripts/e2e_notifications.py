from __future__ import annotations

import json
import os
import subprocess
import uuid
from pathlib import Path

from playwright.sync_api import Page, sync_playwright

ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "output" / "playwright"
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174").rstrip("/")
EMAIL = os.environ["AK_E2E_EMAIL"]
PASSWORD = os.environ["AK_E2E_PASSWORD"]
CONTAINER = os.environ.get("AK_E2E_POSTGRES_CONTAINER", "appkernia-postgres-1")


def sql(statement: str) -> str:
    result = subprocess.run(
        ["docker", "exec", CONTAINER, "psql", "-U", "appkernia", "-d", "appkernia", "-At", "-v", "ON_ERROR_STOP=1", "-c", statement],
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def audit(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate("async()=>await axe.run({exclude:[['.ant-table-measure-row']]},{resultTypes:['violations']})")
    evidence[name] = {"violations": [{"id": item["id"], "impact": item["impact"], "nodes": len(item["nodes"])} for item in result["violations"]]}
    assert not result["violations"], result["violations"]
    dimensions = page.evaluate("()=>({client:document.documentElement.clientWidth,scroll:document.documentElement.scrollWidth})")
    assert dimensions["scroll"] <= dimensions["client"], dimensions


def shot(page: Page, name: str, evidence: dict[str, object]) -> None:
    audit(page, name, evidence)
    page.screenshot(path=OUTPUT / f"admin-{name}.png", full_page=True)


def choose(page: Page, label: str, option: str) -> None:
    page.get_by_label(label, exact=True).last.click()
    dropdown = page.locator(".ant-select-dropdown:visible")
    dropdown.locator(".ant-select-item-option").filter(has_text=option).click()
    dropdown.wait_for(state="hidden")


def login(page: Page) -> None:
    page.goto(f"{BASE}/login", wait_until="networkidle")
    page.get_by_label("账号", exact=True).fill(EMAIL)
    page.get_by_label("密码", exact=True).fill(PASSWORD)
    page.locator('button[type="submit"]').click()
    page.wait_for_url(lambda value: value.startswith(f"{BASE}/dashboard"))


def main() -> None:
    OUTPUT.mkdir(parents=True, exist_ok=True)
    suffix = uuid.uuid4().hex[:10]
    evidence: dict[str, object] = {}
    console_errors: list[str] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
        page = context.new_page()
        page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
        login(page)
        page.get_by_label("显示语言", exact=True).select_option(label="English")

        page.locator('.ak-desktop-sider a[href="/system/notifications/notices"]').click()
        page.wait_for_url(f"{BASE}/system/notifications/notices")
        page.get_by_role("heading", name="Announcements", exact=True).wait_for()
        shot(page, "notifications.notices.en-US.1440.empty", evidence)
        page.get_by_role("button", name="Create draft", exact=True).click()
        drawer = page.get_by_role("dialog", name="Create notification draft", exact=True)
        drawer.get_by_label("Title", exact=True).fill(f"Maintenance {suffix}")
        choose(page, "Body format", "Sanitized HTML")
        drawer.get_by_label("Body", exact=True).fill('<p>Service <strong>ready</strong>.</p><script>window.bad=true</script><a href="javascript:bad()">unsafe</a>')
        assert "window.bad" not in drawer.locator(".ak-notification-preview").inner_text()
        shot(page, "notifications.notices.en-US.1440.editor", evidence)
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/notices") and response.request.method == "POST") as created:
            drawer.get_by_role("button", name="Save", exact=True).click()
        created_body = created.value.json()
        assert created.value.status == 201, created.value.text()
        notice_id = created_body["data"]["id"]
        assert "script" not in created_body["data"]["body"] and "javascript" not in created_body["data"]["body"]
        page.get_by_text("Notification saved.", exact=True).wait_for()
        page.locator("#notification-title").wait_for(state="detached")
        row = page.get_by_role("row").filter(has_text=f"Maintenance {suffix}")
        with page.expect_response(lambda response: response.url.endswith(f"/notices/{notice_id}/recipient-preview")) as preview:
            row.get_by_role("button", name="Publish", exact=True).click()
        assert preview.value.status == 200
        preview_count = preview.value.json()["data"]["count"]
        page.get_by_role("dialog", name="Confirm recipients and publish?", exact=True).wait_for()
        shot(page, "notifications.notices.en-US.1440.confirm", evidence)
        with page.expect_response(lambda response: response.url.endswith(f"/notices/{notice_id}/publish")) as published:
            page.get_by_role("dialog", name="Confirm recipients and publish?", exact=True).get_by_role("button", name="Publish", exact=True).click()
        assert published.value.status == 200
        assert published.value.json()["data"]["recipients"]["count"] == preview_count
        page.get_by_text("Notification published or scheduled.", exact=True).wait_for()
        row.get_by_role("button", name="View", exact=True).click()
        page.wait_for_url(f"{BASE}/system/notifications/notices/{notice_id}")
        page.get_by_role("heading", name=f"Maintenance {suffix}", exact=True).wait_for()
        assert page.locator("script").filter(has_text="window.bad").count() == 0
        shot(page, "notifications.notices.en-US.1440.detail", evidence)

        page.locator('.ak-desktop-sider a[href="/system/notifications/messages"]').click()
        page.wait_for_url(f"{BASE}/system/notifications/messages")
        page.get_by_role("heading", name="In-app messages", exact=True).wait_for()
        page.get_by_role("button", name="Create draft", exact=True).click()
        message_drawer = page.get_by_role("dialog", name="Create notification draft", exact=True)
        message_drawer.get_by_label("Title", exact=True).fill(f"Security {suffix}")
        choose(page, "Message type", "Security")
        message_drawer.get_by_label("Body", exact=True).fill("Review your active sessions.")
        choose(page, "Audience scope", "0 selected member(s)")
        message_drawer.get_by_label("Recipients", exact=True).click()
        page.locator(".ant-select-dropdown:visible .ant-select-item-option").first.click()
        page.keyboard.press("Escape")
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/messages") and response.request.method == "POST") as message_created:
            message_drawer.get_by_role("button", name="Save", exact=True).click()
        assert message_created.value.status == 201, message_created.value.text()
        message_id = message_created.value.json()["data"]["id"]
        message_row = page.get_by_role("row").filter(has_text=f"Security {suffix}")
        with page.expect_response(lambda response: response.url.endswith(f"/messages/{message_id}/recipient-preview")) as message_preview:
            message_row.get_by_role("button", name="Publish", exact=True).click()
        assert message_preview.value.json()["data"]["count"] == 1
        page.get_by_role("dialog", name="Confirm recipients and publish?", exact=True).get_by_role("button", name="Cancel", exact=True).click()
        evidence["recipient_confirmation"] = {"notice_count": preview_count, "selected_message_count": 1}

        page.locator('.ak-desktop-sider a[href="/system/notifications/templates"]').click()
        page.wait_for_url(f"{BASE}/system/notifications/templates")
        page.get_by_role("heading", name="Notification templates", exact=True).wait_for()
        page.get_by_role("button", name="Create template", exact=True).click()
        template_drawer = page.get_by_role("dialog", name="Create notification template", exact=True)
        template_drawer.get_by_label("Template code", exact=True).fill(f"e2e.welcome.{suffix}")
        template_drawer.get_by_label("Template name", exact=True).fill(f"E2E Welcome {suffix}")
        choose(page, "Channel", "Email")
        template_drawer.get_by_label("Subject template", exact=True).fill("Hello {{name}}")
        template_drawer.get_by_label("Body template", exact=True).fill("Welcome {{name}}")
        template_drawer.get_by_label("Variable JSON Schema", exact=True).fill('{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}')
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/notification-templates") and response.request.method == "POST") as template_created:
            template_drawer.get_by_role("button", name="Save", exact=True).click()
        assert template_created.value.status == 201, template_created.value.text()
        template_id = template_created.value.json()["data"]["id"]
        page.get_by_text("Notification saved.", exact=True).wait_for()
        page.locator("#notification-template-code").wait_for(state="detached")
        shot(page, "notifications.templates.en-US.1440", evidence)
        page.set_viewport_size({"width": 1024, "height": 768})
        shot(page, "notifications.templates.en-US.1024", evidence)
        page.set_viewport_size({"width": 768, "height": 1024})
        shot(page, "notifications.templates.en-US.768", evidence)
        page.set_viewport_size({"width": 1440, "height": 900})

        delivery_id = sql(f"""INSERT INTO notify.deliveries(tenant_id,message_id,user_id,template_id,channel,target_ciphertext,target_hash,target_hint,target_key_version,provider,status,attempt_count,max_attempts,last_error)
SELECT tm.tenant_id,'{notice_id}',tm.user_id,'{template_id}','email',decode('01','hex'),decode(repeat('01',32),'hex'),'e***@example.test',1,'local-mock','failed',1,3,'temporary failure' FROM iam.tenant_members tm JOIN iam.users u ON u.id=tm.user_id WHERE u.email='{EMAIL}' LIMIT 1 RETURNING id;""").splitlines()[0]
        assert delivery_id
        page.locator('.ak-desktop-sider a[href="/system/notifications/deliveries"]').click()
        page.wait_for_url(f"{BASE}/system/notifications/deliveries")
        page.get_by_role("heading", name="Delivery records", exact=True).wait_for()
        delivery_row = page.get_by_role("row").filter(has_text="e***@example.test")
        delivery_row.get_by_role("button", name="View", exact=True).click()
        page.get_by_role("dialog", name="Delivery details", exact=True).wait_for()
        page.get_by_text("PROVIDER_DELIVERY_FAILED", exact=True).wait_for()
        page.wait_for_timeout(500)
        shot(page, "notifications.deliveries.en-US.1440.failed", evidence)
        page.get_by_role("dialog", name="Delivery details", exact=True).get_by_role("button", name="Retry", exact=True).click()
        retry_dialog = page.get_by_role("dialog", name="Retry this delivery?", exact=True)
        with page.expect_response(lambda response: response.url.endswith(f"/notification-deliveries/{delivery_id}/retry")) as retried:
            retry_dialog.get_by_role("button", name="Retry", exact=True).click()
        assert retried.value.status == 200 and retried.value.json()["data"]["status"] == "pending"
        page.get_by_text("Delivery queued for retry.", exact=True).wait_for()
        evidence["delivery_retry"] = {"status": retried.value.status, "delivery_id": delivery_id}

        page.get_by_label("Display language", exact=True).select_option(label="简体中文")
        page.get_by_role("heading", name="投递记录", exact=True).wait_for()
        shot(page, "notifications.deliveries.zh-CN.1440", evidence)
        page.set_viewport_size({"width": 375, "height": 812})
        page.wait_for_timeout(500)
        shot(page, "notifications.deliveries.zh-CN.375", evidence)
        audits = int(sql("SELECT count(*) FROM audit.operation_logs WHERE module_code='notify' AND action_name IN ('notify.notice.create','notify.notice.publish','notify.message.create','notify.template.create','notify.delivery.retry');"))
        assert audits >= 5
        evidence["notification_audits"] = audits
        evidence["unexpected_console_errors"] = console_errors
        assert not console_errors, console_errors
        (OUTPUT / "admin-notifications-e2e-results.json").write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        context.close()
        browser.close()
    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
