from __future__ import annotations

import json
import os
import subprocess
from pathlib import Path

from playwright.sync_api import Page, sync_playwright


ROOT = Path(__file__).resolve().parents[3]
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174").rstrip("/")
EMAIL = os.environ["AK_E2E_EMAIL"]
PASSWORD = os.environ["AK_E2E_PASSWORD"]
POSTGRES = os.environ.get("AK_E2E_POSTGRES_CONTAINER", "appkernia-postgres-1")
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
ARTIFACTS = ROOT / "apps" / "ak-admin" / "artifacts" / "ui-ux-pro-max" / "dictionary-notification-drivers"
SCREENSHOTS = ARTIFACTS / "screenshots"
RESULTS = ROOT / "output" / "playwright" / "dictionary-notification-drivers-e2e-results.json"


def sql(statement: str) -> str:
    completed = subprocess.run(
        ["docker", "exec", POSTGRES, "psql", "-U", "appkernia", "-d", "appkernia", "-At", "-v", "ON_ERROR_STOP=1", "-c", statement],
        check=True,
        capture_output=True,
        text=True,
    )
    return completed.stdout.strip()


def quoted(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def audit(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate(
        "async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })"
    )
    violations = [
        {
            "id": item["id"],
            "impact": item["impact"],
            "nodes": len(item["nodes"]),
            "targets": [node["target"] for node in item["nodes"]],
            "html": [node["html"] for node in item["nodes"]],
            "failure": [node.get("failureSummary", "") for node in item["nodes"]],
        }
        for item in result["violations"]
    ]
    serious = [item for item in violations if item["impact"] in ("critical", "serious")]
    dimensions = page.evaluate(
        "() => ({client: document.documentElement.clientWidth, scroll: document.documentElement.scrollWidth})"
    )
    evidence[name] = {"violations": violations, "critical_or_serious": len(serious), "viewport": dimensions}
    assert not serious, serious
    assert dimensions["scroll"] <= dimensions["client"], dimensions


def screenshot(page: Page, name: str, evidence: dict[str, object]) -> None:
    # Ant Drawer/Select transitions temporarily blend foreground and backdrop
    # colors. Audit the stable rendered state, not an animation frame.
    page.wait_for_timeout(600)
    audit(page, name, evidence)
    path = SCREENSHOTS / f"{name}.png"
    page.screenshot(path=path, full_page=True)
    evidence[f"{name}.screenshot"] = str(path.relative_to(ROOT))


def choose(page: Page, label: str, option: str) -> None:
    page.get_by_label(label, exact=True).last.click()
    dropdown = page.locator(".ant-select-dropdown:visible")
    target = dropdown.get_by_text(option, exact=True)
    if target.count() == 0:
        holder = dropdown.locator(".rc-virtual-list-holder")
        holder.evaluate("element => { element.scrollTop = element.scrollHeight; element.dispatchEvent(new Event('scroll')); }")
        page.wait_for_timeout(300)
    if target.count() == 0:
        raise AssertionError(f"select option {option!r} not rendered: {dropdown.inner_text()!r}")
    target.click()
    dropdown.wait_for(state="hidden")


def switch_locale(page: Page, button_name: str, locale_name: str) -> None:
    page.get_by_role("button", name=button_name, exact=True).click()
    page.get_by_text(locale_name, exact=True).last.click()


def login(page: Page) -> None:
    page.goto(f"{BASE}/login", wait_until="networkidle")
    page.get_by_label("账号", exact=True).fill(EMAIL)
    page.get_by_label("密码", exact=True).fill(PASSWORD)
    page.locator('button[type="submit"]').click()
    page.wait_for_url(lambda value: value.startswith(f"{BASE}/dashboard"))


def navigate(page: Page, path: str, parents: tuple[str, ...]) -> None:
    sider = page.locator(".ak-desktop-sider")
    link = sider.locator(f'a[href="{path}"]')
    for index, parent in enumerate(parents):
        if link.count() > 0:
            break
        next_label = parents[index + 1] if index + 1 < len(parents) else None
        if next_label is None or sider.get_by_text(next_label, exact=True).count() == 0:
            sider.get_by_text(parent, exact=True).click()
    link.click()
    page.wait_for_url(lambda value: value.startswith(f"{BASE}{path}"))


def cleanup() -> None:
    email = quoted(EMAIL)
    tenant_id = sql(
        f"SELECT tm.tenant_id FROM iam.tenant_members tm JOIN iam.users u ON u.id=tm.user_id WHERE u.email={email} ORDER BY tm.created_at LIMIT 1"
    )
    if tenant_id:
        tenant = quoted(tenant_id.splitlines()[0])
        sql(
            f"DELETE FROM river_job WHERE args->>'delivery_id' IN (SELECT id::text FROM notify.deliveries WHERE tenant_id={tenant}::uuid)"
        )
        sql(
            "BEGIN; "
            f"DELETE FROM sys.role_menus WHERE role_id IN (SELECT id FROM iam.roles WHERE tenant_id={tenant}::uuid); "
            f"DELETE FROM iam.role_permissions WHERE role_id IN (SELECT id FROM iam.roles WHERE tenant_id={tenant}::uuid); "
            f"DELETE FROM iam.role_scope_units WHERE role_id IN (SELECT id FROM iam.roles WHERE tenant_id={tenant}::uuid); "
            f"DELETE FROM iam.user_roles WHERE tenant_id={tenant}::uuid; "
            f"DELETE FROM iam.sessions WHERE tenant_id={tenant}::uuid; "
            f"DELETE FROM iam.tenant_members WHERE tenant_id={tenant}::uuid; "
            f"DELETE FROM iam.roles WHERE tenant_id={tenant}::uuid; "
            f"DELETE FROM iam.tenants WHERE id={tenant}::uuid; COMMIT;"
        )
    sql(f"DELETE FROM iam.users WHERE email={email}")


def main() -> None:
    SCREENSHOTS.mkdir(parents=True, exist_ok=True)
    RESULTS.parent.mkdir(parents=True, exist_ok=True)
    evidence: dict[str, object] = {}
    console_errors: list[str] = []
    http_errors: list[dict[str, object]] = []
    try:
        with sync_playwright() as playwright:
            browser = playwright.chromium.launch()
            context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
            page = context.new_page()
            page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
            page.on("response", lambda response: http_errors.append({"status": response.status, "url": response.url}) if response.status >= 400 else None)
            login(page)

            navigate(page, "/system/settings/dictionaries", ("系统", "系统设置"))
            page.wait_for_timeout(1500)
            if page.get_by_role("heading", name="字典管理", exact=True).count() == 0:
                page.screenshot(path=SCREENSHOTS / "debug-dictionaries.png", full_page=True)
                raise AssertionError(f"dictionary page did not render: url={page.url} body={page.locator('body').inner_text()[:2000]}")
            page.get_by_label("搜索字典名称或代码", exact=True).fill("storage.driver")
            storage_type = page.get_by_text("storage.driver", exact=True)
            try:
                storage_type.wait_for()
            except Exception as error:
                page.screenshot(path=SCREENSHOTS / "debug-dictionaries-list.png", full_page=True)
                raise AssertionError(f"storage dictionary did not load: body={page.locator('body').inner_text()[:2000]} http_errors={http_errors}") from error
            storage_type.click()
            for label in ("腾讯云 COS", "阿里云 OSS", "七牛云 Kodo"):
                page.get_by_text(label, exact=True).wait_for()
            screenshot(page, "zh-CN-dictionaries-1440", evidence)

            navigate(page, "/system/settings/configs", ("系统", "系统设置"))
            with page.expect_response(
                lambda response: "/admin-api/v1/dictionaries/storage.driver" in response.url
            ) as storage_dictionary_response:
                with page.expect_response(lambda response: "/admin-api/v1/configs?" in response.url and "module_code=storage" in response.url) as cloud_configs:
                    page.get_by_text("云存储", exact=True).click()
            cloud_payload = cloud_configs.value.json()
            evidence["cloud_config_count"] = len(cloud_payload["data"]["items"])
            assert any(item["config_key"] == "storage.driver" for item in cloud_payload["data"]["items"]), cloud_payload
            storage_select = page.get_by_label("存储驱动", exact=True)
            try:
                storage_select.wait_for()
            except Exception as error:
                page.screenshot(path=SCREENSHOTS / "debug-configs.png", full_page=True)
                raise AssertionError(f"storage driver field did not render: body={page.locator('body').inner_text()[:2500]} http_errors={http_errors}") from error
            resolved_storage = storage_dictionary_response.value
            assert resolved_storage.status == 200, resolved_storage.text()
            resolved_storage_labels = [item["label"] for item in resolved_storage.json()["data"]["items"]]
            assert "腾讯云 COS" in resolved_storage_labels, resolved_storage_labels
            assert "阿里云 OSS" in resolved_storage_labels, resolved_storage_labels
            assert "七牛云 Kodo" in resolved_storage_labels, resolved_storage_labels
            storage_select.click()
            dropdown = page.locator(".ant-select-dropdown:visible")
            storage_options = [item.inner_text() for item in dropdown.get_by_role("option").all()]
            page.keyboard.press("Escape")
            evidence["storage_dictionary_select"] = {
                "resolved_labels": resolved_storage_labels,
                "rendered_virtual_window": storage_options,
            }

            switch_locale(page, "显示语言", "English")
            page.get_by_role("heading", name="System Configuration", exact=True).wait_for()
            assert page.get_by_label("Storage driver", exact=True).is_visible()

            navigate(page, "/system/notifications/templates", ("System", "Notification Center"))
            page.get_by_role("heading", name="Notification templates", exact=True).wait_for()
            choose(page, "Channel", "SMS")
            sms_row = page.get_by_role("row").filter(has_text="General SMS verification code").first
            sms_row.get_by_role("button", name="SMS binding", exact=True).click()
            binding = page.get_by_role("dialog", name="SMS provider bindings", exact=True)
            provider = binding.get_by_label("SMS provider", exact=True)
            provider.click()
            provider_dropdown = page.locator(".ant-select-dropdown:visible")
            provider_labels = provider_dropdown.inner_text().splitlines()
            assert "Tencent Cloud SMS" in provider_labels and "Alibaba Cloud SMS" in provider_labels, provider_labels
            provider.press("Escape")
            provider_dropdown.wait_for(state="hidden")
            binding.get_by_label("Approved external template ID", exact=True).fill("SMS_E2E_APPROVED_001")
            binding.get_by_label("Tencent parameter order", exact=True).fill("code, expires_minutes")
            with page.expect_response(lambda response: "/sms-bindings/tencent" in response.url and response.request.method == "PUT") as binding_saved:
                binding.get_by_role("button", name="Save", exact=True).click()
            assert binding_saved.value.status == 200, binding_saved.value.text()
            screenshot(page, "en-US-sms-binding-1440", evidence)
            binding.locator(".ant-drawer-close").click()

            sms_row.get_by_role("button", name="Test send", exact=True).click()
            test_drawer = page.get_by_role("dialog", name="Real template test", exact=True)
            test_drawer.get_by_text("This sends a real, billable SMS", exact=True).wait_for()
            assert test_drawer.get_by_label("I understand the charge and duplicate-send risk.", exact=True).is_visible()
            screenshot(page, "en-US-sms-test-risk-1440", evidence)
            test_drawer.locator(".ant-drawer-close").click()

            choose(page, "Channel", "Email")
            email_row = page.get_by_role("row").filter(has_text="Password reset email").first
            email_row.get_by_role("button", name="Test send", exact=True).click()
            email_test = page.get_by_role("dialog", name="Real template test", exact=True)
            email_test.get_by_label("Recipient", exact=True).fill("delivery-check@example.test")
            email_test.get_by_label("Declared variables (JSON)", exact=True).fill(
                json.dumps({"reset_url": "https://admin.example.test/reset-password?token=redacted", "expires_minutes": "15"})
            )
            with page.expect_response(lambda response: response.url.endswith("/test") and response.request.method == "POST") as queued:
                email_test.get_by_role("button", name="Send now", exact=True).click()
            assert queued.value.status == 202, queued.value.text()
            delivery_id = queued.value.json()["data"]["id"]
            page.get_by_text("The test delivery was encrypted and queued.", exact=True).wait_for()
            evidence["email_test_delivery"] = {"id": delivery_id, "status": queued.value.status}

            switch_locale(page, "Display language", "简体中文")
            page.get_by_role("heading", name="通知模板", exact=True).wait_for()
            page.set_viewport_size({"width": 375, "height": 812})
            page.wait_for_timeout(500)
            screenshot(page, "zh-CN-templates-375", evidence)

            tenant_id = sql(
                f"SELECT tm.tenant_id FROM iam.tenant_members tm JOIN iam.users u ON u.id=tm.user_id WHERE u.email={quoted(EMAIL)} ORDER BY tm.created_at LIMIT 1"
            ).splitlines()[0]
            persisted = sql(
                "SELECT status||'|'||retry_risk||'|'||(position(convert_to('delivery-check@example.test','UTF8') in target_ciphertext)=0)::text "
                f"FROM notify.deliveries WHERE id={quoted(delivery_id)}::uuid AND tenant_id={quoted(tenant_id)}::uuid"
            )
            parts = persisted.split("|")
            assert len(parts) == 3 and parts[2] == "true", persisted
            audits = int(
                sql(
                    "SELECT count(*) FROM audit.operation_logs "
                    f"WHERE tenant_id={quoted(tenant_id)}::uuid AND action_name IN ('notify.template.binding.update','notify.template.test')"
                )
            )
            assert audits >= 2
            evidence["encrypted_delivery_persistence"] = {"status": parts[0], "retry_risk": parts[1], "plaintext_absent": True}
            evidence["notification_audits"] = audits
            evidence["console_errors"] = console_errors
            assert not console_errors, console_errors
            RESULTS.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
            context.close()
            browser.close()
    finally:
        cleanup()
    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
