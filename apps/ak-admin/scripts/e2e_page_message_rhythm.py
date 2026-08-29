from __future__ import annotations

import json
import os
from pathlib import Path

from playwright.sync_api import Page, sync_playwright

from e2e_mobile_releases import APP_ID, route_handler


ROOT = Path(__file__).resolve().parents[3]
BASE = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4173").rstrip("/")
AXE = ROOT / "apps/ak-admin/node_modules/axe-core/axe.min.js"
OUT = ROOT / "apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-page-message-rhythm/screenshots"
OUT.mkdir(parents=True, exist_ok=True)


def login(page: Page) -> None:
    page.goto(f"{BASE}/login", wait_until="domcontentloaded")
    page.locator("#login-email").fill("e2e@app.test")
    page.locator("#login-password").fill("password")
    page.locator('button[type="submit"]').click()
    page.wait_for_url("**/dashboard**")


def measure_message_gaps(page: Page) -> list[dict[str, object]]:
    measurements = page.evaluate(
        """() => Array.from(document.querySelectorAll('.ak-page-container .ant-alert')).flatMap((alert) => {
            const next = alert.nextElementSibling
            if (!next || alert.parentElement?.classList.contains('ant-space-item')) return []
            const alertRect = alert.getBoundingClientRect()
            const nextRect = next.getBoundingClientRect()
            return [{
                title: alert.querySelector('.ant-alert-title')?.textContent?.trim() ?? '',
                nextClass: String(next.className),
                gap: Math.round((nextRect.top - alertRect.bottom) * 100) / 100,
            }]
        })"""
    )
    assert measurements, "expected a visible page Alert followed by a content surface"
    too_close = [item for item in measurements if item["gap"] < 15.5]
    assert not too_close, too_close
    return measurements


def assert_no_page_overflow(page: Page) -> None:
    size = page.evaluate(
        "() => ({ scrollWidth: document.documentElement.scrollWidth, clientWidth: document.documentElement.clientWidth })"
    )
    assert size["scrollWidth"] <= size["clientWidth"], size


def axe_result(page: Page) -> dict[str, object]:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate(
        "async () => await axe.run({ exclude: [['.ant-table-measure-row']] }, { resultTypes: ['violations'] })"
    )
    severe = [item for item in result["violations"] if item["impact"] in ("critical", "serious")]
    assert not severe, severe
    return {
        "serious_critical": len(severe),
        "violations": [
            {"id": item["id"], "impact": item["impact"], "nodes": len(item["nodes"])}
            for item in result["violations"]
        ],
    }


def main() -> None:
    evidence: dict[str, object] = {
        "source_audit": {
            "page_files_with_alerts": 27,
            "page_alerts": 66,
            "page_scoped_non_space_alerts": 54,
        }
    }
    console_errors: list[str] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
        page = context.new_page()
        page.set_default_timeout(15_000)
        page.route("**/admin-api/**", route_handler)
        page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
        login(page)
        route = f"/app/upgrade-center?app_id={APP_ID}&q=&package_type=&platform=&publish_status=&page=1&page_size=20"
        page.evaluate(
            "path => { window.history.pushState({}, '', path); window.dispatchEvent(new PopStateEvent('popstate')); }",
            route,
        )
        page.get_by_role("heading", name="App升级中心").wait_for()
        page.get_by_text("uni-app x 仅支持原生版本升级", exact=False).wait_for()

        for width, height in ((1440, 900), (375, 812)):
            page.set_viewport_size({"width": width, "height": height})
            page.wait_for_timeout(300)
            assert_no_page_overflow(page)
            evidence[f"zh-CN.light.{width}"] = {
                "message_gaps": measure_message_gaps(page),
                "axe": axe_result(page),
                "page_overflow": False,
            }
            page.screenshot(path=OUT / f"upgrade-center.zh-CN.light.{width}.png", full_page=True)

        assert not console_errors, console_errors
        evidence["console_errors"] = console_errors
        (OUT / "e2e-results.json").write_text(
            json.dumps(evidence, ensure_ascii=False, indent=2), encoding="utf-8"
        )
        context.close()
        browser.close()

    print(json.dumps(evidence, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
