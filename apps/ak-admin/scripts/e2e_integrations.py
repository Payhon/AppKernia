from __future__ import annotations

import json
import os
import subprocess
import uuid
from pathlib import Path

from playwright.sync_api import Page, sync_playwright

from e2e_navigation_helpers import open_system_page

ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "output" / "playwright"
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174").rstrip("/")
EMAIL = os.environ["AK_E2E_EMAIL"]
PASSWORD = os.environ["AK_E2E_PASSWORD"]
API_CONTAINER = os.environ.get("AK_E2E_API_CONTAINER", "appkernia-api-1")


def audit(page: Page, name: str, evidence: dict[str, object]) -> None:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate("async()=>await axe.run({exclude:[['.ant-table-measure-row']]},{resultTypes:['violations']})")
    violations = [{"id": item["id"], "impact": item["impact"], "nodes": len(item["nodes"])} for item in result["violations"]]
    severe = [item for item in violations if item["impact"] in ("critical", "serious")]
    dimensions = page.evaluate("()=>({client:document.documentElement.clientWidth,scroll:document.documentElement.scrollWidth})")
    evidence[name] = {"violations": violations, "critical_or_serious": len(severe), "dimensions": dimensions}
    assert not severe, severe
    assert dimensions["scroll"] <= dimensions["client"], dimensions


def shot(page: Page, name: str, evidence: dict[str, object]) -> None:
    audit(page, name, evidence)
    page.evaluate("()=>{if(document.activeElement instanceof HTMLElement)document.activeElement.blur();document.querySelector('.ak-skip-link')?.blur()}")
    page.wait_for_timeout(50)
    page.screenshot(path=OUTPUT / f"admin-integrations.{name}.png", full_page=True)


def internal_post(path: str, body: dict[str, object], token: str | None = None) -> subprocess.CompletedProcess[str]:
    command = ["docker", "exec", API_CONTAINER, "wget", "-qO-", "--header", "Content-Type: application/json"]
    if token:
        command += ["--header", f"Authorization: Bearer {token}"]
    command += ["--post-data", json.dumps(body, separators=(",", ":")), f"http://127.0.0.1:8080{path}"]
    return subprocess.run(command, capture_output=True, text=True, check=False)


def internal_get(path: str, token: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["docker", "exec", API_CONTAINER, "wget", "-S", "-qO-", "--header", f"Authorization: Bearer {token}", f"http://127.0.0.1:8080{path}"],
        capture_output=True,
        text=True,
        check=False,
    )


def login(page: Page) -> None:
    page.goto(f"{BASE}/login", wait_until="networkidle")
    page.get_by_label("账号", exact=True).fill(EMAIL)
    page.get_by_label("密码", exact=True).fill(PASSWORD)
    page.locator('button[type="submit"]').click()
    page.wait_for_url(lambda value: value.startswith(f"{BASE}/dashboard"))
    if page.locator("html").get_attribute("lang") == "zh-CN":
        page.get_by_label("显示语言", exact=True).select_option(label="English")
    else:
        page.get_by_label("Display language", exact=True).wait_for()


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

        open_system_page(page, "/system/integrations/api-clients", "Jobs & Integrations")
        page.wait_for_url(f"{BASE}/system/integrations/api-clients")
        page.get_by_role("heading", name="API Clients", exact=True).wait_for()
        shot(page, "api-clients.en-US.1440.empty", evidence)
        page.get_by_role("button", name="Create client", exact=True).click()
        page.get_by_label("Name", exact=True).fill(f"Machine client {suffix}")
        page.get_by_label("Description", exact=True).fill(f"E2E {suffix}")
        page.get_by_label("Allowed CIDRs", exact=True).fill("127.0.0.1/32")
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/api-clients") and response.request.method == "POST") as client_created:
            page.get_by_role("button", name="Save", exact=True).click()
        assert client_created.value.status == 201, client_created.value.text()
        client = client_created.value.json()["data"]
        assert client["secrets"] == []
        row = page.get_by_role("row").filter(has_text=f"Machine client {suffix}")
        row.wait_for()
        client_id = row.locator("code").inner_text()
        with page.expect_response(
            lambda response: f"/admin-api/v1/api-clients/{client['id']}" in response.url
            and response.request.method == "GET"
        ) as client_detail:
            row.get_by_role("button", name="View", exact=True).click()
        detail_body = client_detail.value.json()["data"]
        assert client_detail.value.status == 200 and detail_body["id"] == client["id"]
        assert "secret" not in detail_body and all("secret" not in item for item in detail_body.get("secrets", []))
        page.wait_for_url(f"{BASE}/system/integrations/api-clients/{client['id']}")
        page.get_by_role("heading", name="API Client Details", exact=True).wait_for()
        shot(page, "api-client-detail.en-US.1440", evidence)
        page.get_by_label("Display language", exact=True).select_option(label="简体中文")
        page.get_by_role("heading", name="API 客户端详情", exact=True).wait_for()
        page.set_viewport_size({"width": 375, "height": 812})
        shot(page, "api-client-detail.zh-CN.375", evidence)
        page.set_viewport_size({"width": 1440, "height": 900})
        page.get_by_label("显示语言", exact=True).select_option(label="English")
        page.get_by_role("button", name="Back to API Clients", exact=True).click()
        page.wait_for_url(lambda value: value.startswith(f"{BASE}/system/integrations/api-clients"))
        page.get_by_role("heading", name="API Clients", exact=True).wait_for()
        row = page.get_by_role("row").filter(has_text=f"Machine client {suffix}")
        row.wait_for()
        row.get_by_role("button", name="Manage secrets", exact=True).click()
        page.get_by_role("button", name="Create secret", exact=True).click()
        secret_value = page.get_by_label("Client secret", exact=True).input_value()
        assert secret_value.startswith("aks_")
        page.get_by_role("checkbox", name="I have stored this secret securely", exact=True).check()
        page.get_by_role("button", name="Saved and close", exact=True).click()
        token_response = internal_post("/api/v1/auth/client-token", {"client_id": client_id, "client_secret": secret_value})
        assert token_response.returncode == 0, token_response.stderr
        machine_token = json.loads(token_response.stdout)["data"]["access_token"]
        admin_rejection = internal_get("/admin-api/v1/api-clients", machine_token)
        assert admin_rejection.returncode != 0 and "401" in admin_rejection.stderr
        page.get_by_role("button", name="Revoke", exact=True).click()
        confirm = page.get_by_role("dialog", name="Revoke client secret", exact=True)
        with page.expect_response(lambda response: "/secrets/" in response.url and response.request.method == "DELETE") as secret_revoked:
            confirm.get_by_role("button", name="Revoke", exact=True).click()
        assert secret_revoked.value.status == 200
        rejected_secret = internal_post("/api/v1/auth/client-token", {"client_id": client_id, "client_secret": secret_value})
        assert rejected_secret.returncode != 0
        secret_value = ""
        machine_token = ""
        page.get_by_text("Client secret revoked", exact=True).wait_for()
        shot(page, "api-clients.en-US.1440.revoked", evidence)
        page.get_by_label("Display language", exact=True).select_option(label="简体中文")
        page.get_by_role("heading", name="API 客户端", exact=True).wait_for()
        for width, height in ((1024, 768), (768, 1024), (375, 812)):
            page.set_viewport_size({"width": width, "height": height})
            page.wait_for_timeout(250)
            shot(page, f"api-clients.zh-CN.{width}.revoked", evidence)
        page.set_viewport_size({"width": 1440, "height": 900})
        page.get_by_label("显示语言", exact=True).select_option(label="English")

        open_system_page(page, "/system/integrations/webhooks", "Jobs & Integrations")
        page.wait_for_url(f"{BASE}/system/integrations/webhooks")
        page.get_by_role("heading", name="Webhooks", exact=True).wait_for()
        shot(page, "webhooks.en-US.1440.empty", evidence)
        page.get_by_role("button", name="Create endpoint", exact=True).click()
        page.get_by_label("Name", exact=True).fill(f"Order receiver {suffix}")
        page.get_by_label("HTTPS endpoint URL", exact=True).fill("https://hooks.example.test/events")
        page.get_by_label("Event codes", exact=True).fill("order.created")
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/webhooks") and response.request.method == "POST") as webhook_created:
            page.get_by_role("button", name="Save", exact=True).click()
        assert webhook_created.value.status == 201, webhook_created.value.text()
        webhook = webhook_created.value.json()["data"]
        assert webhook["endpoint"]["event_types"] == ["order.created"]
        signing_secret = page.get_by_label("Webhook signing secret", exact=True).input_value()
        assert signing_secret.startswith("whsec_")
        page.get_by_role("checkbox", name="I have stored the signing secret securely", exact=True).check()
        page.get_by_role("button", name="Saved and close", exact=True).click()
        signing_secret = ""
        webhook_row = page.get_by_role("row").filter(has_text=f"Order receiver {suffix}")
        webhook_row.wait_for()
        webhook_row.get_by_role("button", name="Send test", exact=True).click()
        test_confirm = page.get_by_role("dialog", name="Send signed test event", exact=True)
        with page.expect_response(lambda response: response.url.endswith("/test") and response.request.method == "POST") as test_response:
            test_confirm.get_by_role("button", name="Send test", exact=True).click()
        delivery = test_response.value.json()["data"]
        assert test_response.value.status == 200 and delivery["status"] == "succeeded" and delivery["response_status"] == 204
        page.get_by_text("Test delivery succeeded", exact=True).wait_for()
        webhook_row.get_by_role("button", name="View deliveries", exact=True).click()
        page.get_by_role("dialog", name=f"Delivery history for “Order receiver {suffix}”", exact=True).get_by_text("Succeeded", exact=True).wait_for()
        shot(page, "webhooks.en-US.1440.delivery", evidence)
        page.keyboard.press("Escape")
        page.get_by_label("Display language", exact=True).select_option(label="简体中文")
        page.get_by_role("heading", name="Webhook", exact=True).wait_for()
        shot(page, "webhooks.zh-CN.1440.saved", evidence)
        for width, height in ((1024, 768), (768, 1024), (375, 812)):
            page.set_viewport_size({"width": width, "height": height})
            page.wait_for_timeout(250)
            shot(page, f"webhooks.zh-CN.{width}.saved", evidence)

        page.set_viewport_size({"width": 1440, "height": 900})
        page.goto(f"{BASE}/dashboard", wait_until="networkidle")
        page.get_by_label("显示语言", exact=True).select_option(label="English")
        page.goto(f"{BASE}/404", wait_until="networkidle")
        page.get_by_role("heading", name="Page Not Found", exact=True).wait_for()
        shot(page, "errors.en-US.1440.not-found", evidence)
        page.goto(f"{BASE}/offline", wait_until="networkidle")
        page.get_by_role("heading", name="Offline", exact=True).wait_for()
        page.get_by_role("button", name="Retry connection", exact=True).wait_for()
        shot(page, "errors.en-US.1440.offline", evidence)
        page.goto(f"{BASE}/dashboard", wait_until="networkidle")
        page.get_by_label("Display language", exact=True).select_option(label="简体中文")
        page.set_viewport_size({"width": 375, "height": 812})
        page.goto(f"{BASE}/404", wait_until="networkidle")
        page.get_by_role("heading", name="页面不存在", exact=True).wait_for()
        shot(page, "errors.zh-CN.375.not-found", evidence)
        page.goto(f"{BASE}/offline", wait_until="networkidle")
        page.get_by_role("heading", name="网络不可用", exact=True).wait_for()
        page.get_by_role("button", name="重试连接", exact=True).wait_for()
        shot(page, "errors.zh-CN.375.offline", evidence)

        evidence["runtime"] = {
            "api_client_id": client["id"],
            "machine_token_audience": "ak-api",
            "admin_api_rejected_machine_token": True,
            "revoked_secret_rejected": True,
            "webhook_id": webhook["endpoint"]["id"],
            "delivery_id": delivery["id"],
            "delivery_status": delivery["status"],
            "delivery_response_status": delivery["response_status"],
            "api_client_detail_get": True,
            "explicit_404_route": True,
            "offline_route": True,
            "locales": ["zh-CN", "en-US"],
        }
        unexpected = [message for message in console_errors if "401 (Unauthorized)" not in message]
        evidence["unexpected_console_errors"] = unexpected
        assert not unexpected, unexpected
        (OUTPUT / "admin-integrations-e2e-results.json").write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        context.close()
        browser.close()
    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
