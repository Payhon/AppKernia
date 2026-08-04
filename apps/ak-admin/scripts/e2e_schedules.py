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
    violations = [
        {
            "id": item["id"],
            "impact": item["impact"],
            "nodes": [
                {"target": node["target"], "html": node["html"], "summary": node["failureSummary"]}
                for node in item["nodes"]
            ],
        }
        for item in result["violations"]
    ]
    evidence[name] = {"violations": violations}
    assert not violations, violations
    dimensions = page.evaluate("()=>({client:document.documentElement.clientWidth,scroll:document.documentElement.scrollWidth})")
    assert dimensions["scroll"] <= dimensions["client"], dimensions


def shot(page: Page, name: str, evidence: dict[str, object]) -> None:
    audit(page, name, evidence)
    page.evaluate("document.activeElement?.blur()")
    page.screenshot(path=OUTPUT / f"admin-schedules.{name}.png", full_page=True)


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

        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/job-handlers")) as handlers_response:
            page.locator('.ak-desktop-sider a[href="/system/integrations/schedules"]').click()
        page.wait_for_url(f"{BASE}/system/integrations/schedules")
        assert handlers_response.value.status == 200, handlers_response.value.text()
        handlers = handlers_response.value.json()["data"]
        assert [item["key"] for item in handlers] == ["system.health.snapshot"], handlers
        page.get_by_role("heading", name="Scheduled jobs", exact=True).wait_for()
        shot(page, "en-US.1440.empty", evidence)

        page.get_by_role("button", name="Create job", exact=True).click()
        drawer = page.get_by_role("dialog", name="Create scheduled job", exact=True)
        drawer.get_by_label("Job code", exact=True).fill(f"e2e.health.{suffix}")
        drawer.get_by_label("Job name", exact=True).fill(f"Health snapshot {suffix}")
        drawer.get_by_label("Cron expression", exact=True).fill("30 2 * * *")
        choose(page, "IANA time zone", "America/New_York")
        with page.expect_response(
            lambda response: response.url.endswith("/admin-api/v1/job-schedules/preview")
            and "America/New_York" in (response.request.post_data or "")
        ) as preview_response:
            page.locator(".ak-schedule-preview button").click()
        preview_body = preview_response.value.json()["data"]
        assert preview_response.value.status == 200 and len(preview_body["next_runs"]) == 5, preview_body
        assert preview_body["time_zone"] == "America/New_York"
        shot(page, "en-US.1440.editor-preview", evidence)
        with page.expect_response(lambda response: response.url.endswith("/admin-api/v1/job-schedules") and response.request.method == "POST") as created_response:
            drawer.get_by_role("button", name="Save", exact=True).click()
        created = created_response.value.json()["data"]
        assert created_response.value.status == 201, created_response.value.text()
        schedule_id = created["id"]
        assert created["handler_key"] == "system.health.snapshot" and created["status"] == "active"
        page.get_by_text("Scheduled job saved.", exact=True).wait_for()
        row = page.get_by_role("row").filter(has_text=f"Health snapshot {suffix}")
        row.wait_for()
        shot(page, "en-US.1440.saved", evidence)

        row.get_by_role("button", name="Run now", exact=True).click()
        confirm = page.get_by_role("dialog", name="Run this job now?", exact=True)
        with page.expect_response(lambda response: response.url.endswith(f"/job-schedules/{schedule_id}/execute")) as execute_response:
            confirm.get_by_role("button", name="Run now", exact=True).click()
        run = execute_response.value.json()["data"]
        assert execute_response.value.status == 202 and run["status"] == "queued", run
        history = page.get_by_role("dialog", name=f"Run history for “Health snapshot {suffix}”", exact=True)
        history.wait_for()
        history.get_by_text("Succeeded", exact=True).wait_for(timeout=15_000)
        assert history.locator(".ak-schedule-run-output").filter(has_text="JOBS.HEALTH.OK").count() == 1
        shot(page, "en-US.1440.run-succeeded", evidence)
        history.get_by_role("button", name="View", exact=True).click()
        page.wait_for_url(f"{BASE}/system/integrations/schedules/{schedule_id}/runs")
        page.get_by_role("heading", name="Execution history", exact=True).wait_for()
        page.get_by_text(schedule_id, exact=True).wait_for()
        shot(page, "en-US.1440.run-deep-link", evidence)
        page.get_by_label("Display language", exact=True).select_option(label="简体中文")
        page.get_by_role("heading", name="执行记录", exact=True).wait_for()
        page.set_viewport_size({"width": 375, "height": 812})
        shot(page, "zh-CN.375.run-deep-link", evidence)
        page.set_viewport_size({"width": 1440, "height": 900})
        page.get_by_label("显示语言", exact=True).select_option(label="English")
        page.get_by_role("button", name="Back to scheduled jobs", exact=True).click()
        page.wait_for_url(lambda value: value.startswith(f"{BASE}/system/integrations/schedules"))
        page.get_by_role("heading", name="Scheduled jobs", exact=True).wait_for()

        row.get_by_role("button", name="Pause", exact=True).click()
        pause_confirm = page.get_by_role("dialog", name="Pause scheduled job?", exact=True)
        with page.expect_response(lambda response: response.url.endswith(f"/job-schedules/{schedule_id}/pause")) as pause_response:
            pause_confirm.get_by_role("button", name="Pause", exact=True).click()
        assert pause_response.value.status == 200 and pause_response.value.json()["data"]["status"] == "paused"
        row.get_by_role("button", name="Resume", exact=True).click()
        resume_confirm = page.get_by_role("dialog", name="Resume scheduled job?", exact=True)
        with page.expect_response(lambda response: response.url.endswith(f"/job-schedules/{schedule_id}/resume")) as resume_response:
            resume_confirm.get_by_role("button", name="Resume", exact=True).click()
        assert resume_response.value.status == 200 and resume_response.value.json()["data"]["status"] == "active"

        page.get_by_label("Search name or code", exact=True).fill(suffix)
        page.wait_for_timeout(500)
        assert f"q={suffix}" in page.url
        evidence["url_state"] = page.url
        page.get_by_label("Display language", exact=True).select_option(label="简体中文")
        page.get_by_role("heading", name="定时任务", exact=True).wait_for()
        assert page.get_by_text(f"Health snapshot {suffix}", exact=True).count() == 1
        shot(page, "zh-CN.1440.saved", evidence)
        page.set_viewport_size({"width": 1024, "height": 768})
        shot(page, "zh-CN.1024.saved", evidence)
        page.set_viewport_size({"width": 768, "height": 1024})
        shot(page, "zh-CN.768.saved", evidence)
        page.set_viewport_size({"width": 375, "height": 812})
        page.wait_for_timeout(500)
        shot(page, "zh-CN.375.saved", evidence)

        database = sql(f"""SELECT run.status || '|' || COALESCE(run.output->>'result_code','') || '|' || schedule.status
FROM jobs.schedule_runs run JOIN jobs.schedules schedule ON schedule.id=run.schedule_id
WHERE schedule.id='{schedule_id}' ORDER BY run.created_at DESC LIMIT 1;""")
        assert database == "succeeded|JOBS.HEALTH.OK|active", database
        audits = int(sql(f"SELECT count(*) FROM audit.operation_logs WHERE resource_id='{schedule_id}' AND action_name IN ('jobs.schedule.create','jobs.schedule.execute','jobs.schedule.pause','jobs.schedule.resume');"))
        assert audits == 4, audits
        evidence["runtime"] = {
            "schedule_id": schedule_id,
            "run_id": run["id"],
            "database_state": database,
            "audit_count": audits,
            "registered_handlers": [item["key"] for item in handlers],
            "preview_count": len(preview_body["next_runs"]),
        }
        evidence["unexpected_console_errors"] = console_errors
        assert not console_errors, console_errors
        (OUTPUT / "admin-schedules-e2e-results.json").write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        context.close()
        browser.close()
    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
