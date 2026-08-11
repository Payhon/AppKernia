from __future__ import annotations

import json
import os
import re
from pathlib import Path
from urllib.parse import urlparse

from playwright.sync_api import Page, Request, expect, sync_playwright


ROOT = Path(__file__).resolve().parents[3]
BASE_URL = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4174").rstrip("/")
EMAIL = os.environ.get("AK_E2E_EMAIL")
PASSWORD = os.environ.get("AK_E2E_PASSWORD")
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
ARTIFACTS = ROOT / "apps" / "ak-admin" / "artifacts" / "ui-ux-pro-max" / "AKADM-openapi-reference-navigation-i18n"
SCREENSHOTS = ARTIFACTS / "screenshots"
EVIDENCE = ROOT / "output" / "playwright" / "openapi-reference-navigation-i18n.evidence.json"


def severe_axe_violations(page: Page) -> list[dict[str, object]]:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate("async () => await axe.run({ resultTypes: ['violations'] })")
    return [
        {
            "id": item["id"],
            "impact": item["impact"],
            "nodes": len(item["nodes"]),
            "targets": [node["target"] for node in item["nodes"]],
        }
        for item in result["violations"]
        if item["impact"] in ("critical", "serious")
    ]


def external_request_urls(requests: list[str]) -> list[str]:
    expected = urlparse(BASE_URL).netloc
    return sorted({url for url in requests if urlparse(url).netloc != expected})


def exercise_openapi(page: Page, locale: str) -> dict[str, object]:
    labels = {
        "zh-CN": {
            "module": "健康检查",
            "module_search": "健康检查",
            "mobile_menu": "打开菜单",
            "notice": "交互式接口测试提示",
            "notice_close": "关闭接口测试提示",
            "models": re.compile(r"^模型"),
            "operation": "检查 API 进程存活状态",
            "operation_search": "存活",
            "search": "打开搜索",
            "search_input": "输入搜索查询",
            # Scalar localizes the reference shell but its embedded API client
            # keeps this action in English in 1.64.1.
            "send": re.compile(r"^Send Request"),
            "surfaces": ["平台与公共接口", "移动端接口", "管理端接口"],
            "title": "AppKernia OpenAPI 文档",
        },
        "en-US": {
            "module": "Platform health",
            "module_search": "Platform health",
            "mobile_menu": "Open Menu",
            "notice": "Interactive API testing notice",
            "notice_close": "Dismiss interactive API testing notice",
            "models": re.compile(r"^Models"),
            "operation": "Check API process liveness",
            "operation_search": "liveness",
            "search": "Open Search",
            "search_input": "Enter search query",
            "send": re.compile(r"^Send Request"),
            "surfaces": ["Platform and Public APIs", "Mobile APIs", "Admin APIs"],
            "title": "AppKernia OpenAPI Documentation",
        },
    }[locale]
    requests: list[str] = []
    page.on("request", lambda request: requests.append(request.url))
    response = page.goto(f"{BASE_URL}/openapi/?lang={locale}", wait_until="networkidle", timeout=120_000)
    assert response is not None and response.status == 200
    headers = response.headers
    assert "no-cache" in headers.get("cache-control", ""), headers
    assert "frame-ancestors 'none'" in headers.get("content-security-policy", ""), headers
    assert headers.get("referrer-policy") == "no-referrer", headers
    assert headers.get("x-content-type-options") == "nosniff", headers
    assert headers.get("x-frame-options") == "DENY", headers
    expect(page.locator("html")).to_have_attribute("lang", locale)
    assert page.title() == labels["title"]
    expect(page.get_by_text(labels["notice"], exact=True)).to_be_visible()
    expect(page.get_by_role("button", name=labels["notice_close"], exact=True)).to_be_visible()
    expect(page.get_by_role("button", name=labels["models"])).to_be_visible()
    for surface in labels["surfaces"]:
        expect(page.get_by_text(surface, exact=True).first).to_be_visible()

    module = page.get_by_role("button", name=re.compile(rf"^{re.escape(labels['module'])}"), exact=False)
    operation = page.get_by_role("button", name=re.compile(rf"^{re.escape(labels['operation'])}"), exact=False)
    expect(module).to_be_visible()
    expect(operation).to_have_count(0)
    module.click()
    expect(operation).to_be_visible()
    page.screenshot(path=SCREENSHOTS / f"openapi.{locale}.1440.module-expanded.png", full_page=False)

    page.get_by_role("button", name=re.compile(labels["search"]), exact=False).click()
    search = page.get_by_role("combobox", name=labels["search_input"], exact=True)
    search.fill(labels["module_search"])
    expect(page.get_by_role("dialog")).to_contain_text(labels["module"])
    search.fill(labels["operation_search"])
    expect(page.get_by_role("dialog")).to_contain_text(labels["operation"])
    expect(page.locator(".scalar-modal-layout")).to_have_css("opacity", "1")
    page.screenshot(path=SCREENSHOTS / f"openapi.{locale}.1440.search.png", full_page=False)
    page.keyboard.press("Escape")

    operation.click()
    expect(page.get_by_role("heading", name=labels["operation"], exact=True)).to_be_visible()
    assert page.url.endswith("#tag/platform-health/GET/internal/v1/health/live"), page.url
    test_request = page.get_by_role("button").filter(has_text=re.compile("/internal/v1/health/live"))
    expect(test_request).to_be_visible()
    test_request.click()
    send = page.get_by_role("button", name=labels["send"])
    expect(send).to_be_visible()
    with page.expect_response(lambda item: item.url.endswith("/internal/v1/health/live")) as response_info:
        send.click()
    health_response = response_info.value
    assert health_response.status == 200, health_response.status
    health_request: Request = health_response.request
    request_headers = health_request.all_headers()
    assert "cookie" not in request_headers, request_headers
    assert request_headers.get("accept-language") == locale, request_headers
    expect(page.get_by_text("200", exact=True).last).to_be_visible()
    page.get_by_role("button", name="Close Client", exact=True).click()

    page.get_by_role("button", name=labels["notice_close"], exact=True).click()
    expect(page.locator("#ak-openapi-notice")).to_be_hidden()
    page.screenshot(path=SCREENSHOTS / f"openapi.{locale}.1440.notice-dismissed.png", full_page=False)

    internal_response = page.request.get(f"{BASE_URL}/internal/v1/metrics")
    assert internal_response.status == 404, internal_response.status
    yaml_response = page.request.get(f"{BASE_URL}/openapi/openapi.yaml")
    assert yaml_response.status == 200
    assert "no-cache" in yaml_response.headers.get("cache-control", "")
    assert yaml_response.body() == (ROOT / "server" / "openapi" / "openapi.yaml").read_bytes()

    # Scalar keeps its off-canvas request client mounted after close. Reload to
    # audit the stable public documentation surface and the non-persistent-auth
    # boundary rather than counting hidden third-party client controls.
    page.reload(wait_until="networkidle")
    expect(page.locator("#ak-openapi-notice")).to_be_hidden()
    expect(page.locator(".scalar-container.scalar-client--open")).to_have_count(0)
    expect(page.locator(".scalar-mcp-layer")).to_have_count(0)
    expect(page.get_by_text(re.compile(r"生成 MCP|Generate MCP"))).to_have_count(0)
    for component_title in ["Enable a suspended tenant membership", "Suspend a tenant membership and revoke its tenant sessions", "Clear credential lock state for a tenant user"]:
        if locale == "zh-CN":
            expect(page.get_by_text(component_title, exact=True)).to_have_count(0)

    page.set_viewport_size({"width": 375, "height": 812})
    page.get_by_role("button", name=labels["mobile_menu"], exact=True).click()
    mobile_module = page.get_by_role("button", name=re.compile(rf"^{re.escape(labels['module'])}"), exact=False)
    mobile_module.click()
    expect(page.get_by_role("button", name=re.compile(rf"^{re.escape(labels['operation'])}"), exact=False)).to_be_visible()
    assert page.evaluate("document.documentElement.scrollWidth <= document.documentElement.clientWidth")
    page.screenshot(path=SCREENSHOTS / f"openapi.{locale}.375.module-expanded.png", full_page=False)
    violations = severe_axe_violations(page)
    external = external_request_urls(requests)
    forbidden = [
        url for url in requests
        if re.search(r"(?:agent|posthog|telemetry|fonts\.(?:googleapis|gstatic)|proxy\.scalar)", url, re.I)
    ]
    assert not external, external
    assert not forbidden, forbidden
    assert not violations, violations
    return {
        "status": response.status,
        "lang": locale,
        "title": page.title(),
        "interface_surfaces": labels["surfaces"],
        "expanded_module": labels["module"],
        "notice_dismissed_in_session": True,
        "localized_operation_title": labels["operation"],
        "stable_anchor": "#tag/platform-health/GET/internal/v1/health/live",
        "health_status": health_response.status,
        "health_accept_language": request_headers.get("accept-language"),
        "health_cookie_sent": "cookie" in request_headers,
        "internal_metrics_status": internal_response.status,
        "canonical_yaml_bytes": len(yaml_response.body()),
        "external_requests": external,
        "forbidden_requests": forbidden,
        "mobile_viewport": {"width": 375, "height": 812, "horizontal_overflow": False},
        "axe_serious_or_critical": violations,
    }


def login(page: Page) -> None:
    assert EMAIL and PASSWORD, "AK_E2E_EMAIL and AK_E2E_PASSWORD are required for Admin shell coverage"
    page.goto(f"{BASE_URL}/login", wait_until="networkidle")
    if page.locator("html").get_attribute("lang") != "zh-CN":
        page.get_by_label("Display language", exact=True).click()
        page.get_by_role("menuitem", name="简体中文", exact=True).click()
    page.get_by_label("账号", exact=True).fill(EMAIL)
    page.get_by_label("密码", exact=True).fill(PASSWORD)
    page.locator('button[type="submit"]').click()
    page.wait_for_url(re.compile(rf"{re.escape(BASE_URL)}/dashboard"), timeout=30_000)
    page.get_by_role("heading", name="Dashboard", exact=True).wait_for()
    if page.locator("html").get_attribute("lang") != "zh-CN":
        page.get_by_label("Display language", exact=True).click()
        page.get_by_role("menuitem", name="简体中文", exact=True).click()
        expect(page.locator("html")).to_have_attribute("lang", "zh-CN")


def exercise_shell(page: Page) -> dict[str, object]:
    console_errors: list[str] = []
    page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
    login(page)
    primary_labels = page.locator("#ak-primary-navigation .ant-menu-title-content").all_inner_texts()
    assert "系统" not in primary_labels, primary_labels
    docs = page.get_by_role("link", name="OpenAPI 文档", exact=True)
    system = page.get_by_role("button", name="打开系统菜单", exact=True)
    expect(docs).to_be_visible()
    expect(system).to_be_visible()
    docs_box = docs.bounding_box()
    system_box = system.bounding_box()
    sider_box = page.locator(".ak-desktop-sider").bounding_box()
    assert docs_box and system_box and sider_box
    assert docs_box["x"] < system_box["x"]
    assert abs((system_box["y"] + system_box["height"]) - (sider_box["y"] + sider_box["height"] - 6)) <= 8

    current_url = page.url
    with page.expect_popup() as popup_info:
        docs.click()
    docs_popup = popup_info.value
    docs_popup.wait_for_load_state("networkidle")
    assert docs_popup.url == f"{BASE_URL}/openapi/?lang=zh-CN", docs_popup.url
    assert page.url == current_url
    # rel=noopener deliberately severs the opener while Playwright still
    # observes the newly-created browsing context through expect_popup.
    assert docs_popup.evaluate("window.opener === null") is True
    docs_popup.close()
    page.mouse.move(800, 80)
    page.keyboard.press("Escape")
    page.evaluate("document.activeElement instanceof HTMLElement && document.activeElement.blur()")
    page.wait_for_timeout(300)

    page.screenshot(path=SCREENSHOTS / "admin-navigation-openapi.zh-CN.1440-expanded.png")
    system.click()
    popover = page.locator(".ak-system-menu-popover")
    expect(popover).to_be_visible()
    popover_style = popover.locator(".ant-popover-container").evaluate(
        """element => {
          const style = getComputedStyle(element)
          return { border: style.border, borderRadius: style.borderRadius, boxShadow: style.boxShadow }
        }"""
    )
    assert popover_style["border"] != "0px none rgb(0, 0, 0)", popover_style
    assert popover_style["borderRadius"] == "12px", popover_style
    assert popover_style["boxShadow"] != "none", popover_style
    group = page.locator("#ak-desktop-system-navigation .ant-menu-submenu-title").filter(has_text="系统设置")
    group.focus()
    page.keyboard.press("ArrowRight")
    expect(group).to_have_attribute("aria-expanded", "true")
    dictionaries = page.locator(".ak-navigation-submenu-popup:visible").get_by_role(
        "link", name="字典管理", exact=True
    )
    expect(dictionaries).to_be_visible()
    expect(popover).to_be_visible()
    page.wait_for_timeout(250)
    page.screenshot(path=SCREENSHOTS / "admin-system-popover.zh-CN.1440.png")
    dictionaries.click()
    page.wait_for_url(re.compile(r"/system/settings/dictionaries(?:\?.*)?$"))
    expect(page.get_by_role("button", name="打开系统菜单", exact=True)).to_have_attribute("aria-current", "page")
    expect(popover).to_have_count(0)

    page.get_by_label("显示语言", exact=True).click()
    page.get_by_role("menuitem", name="English", exact=True).click()
    expect(page.locator("html")).to_have_attribute("lang", "en-US")
    page.get_by_role("button", name="Collapse navigation", exact=True).click()
    expect(page.locator(".ak-desktop-sider")).to_have_class(re.compile("ant-layout-sider-collapsed"))
    collapsed_docs = page.get_by_role("link", name="OpenAPI documentation", exact=True)
    collapsed_system = page.get_by_role("button", name="Open system menu", exact=True)
    expect(collapsed_docs).to_have_css("width", "40px")
    expect(collapsed_system).to_have_css("width", "40px")
    collapsed_docs_box = collapsed_docs.bounding_box()
    collapsed_system_box = collapsed_system.bounding_box()
    assert collapsed_docs_box and collapsed_system_box
    assert round(collapsed_docs_box["width"]) == 40, collapsed_docs_box
    assert round(collapsed_system_box["width"]) == 40, collapsed_system_box
    page.screenshot(path=SCREENSHOTS / "admin-navigation-openapi.en-US.1440-collapsed.png")
    collapsed_system.click()
    expect(page.locator(".ak-system-menu-popover")).to_be_visible()
    page.keyboard.press("Escape")
    expect(collapsed_system).to_be_focused()
    expect(page.locator(".ak-system-menu-popover")).to_have_count(0)

    page.emulate_media(reduced_motion="reduce")
    duration = collapsed_docs.evaluate("element => getComputedStyle(element).transitionDuration")
    assert duration in ("0s", "0.00001s", "1e-05s"), duration
    page.emulate_media(reduced_motion="no-preference")

    page.set_viewport_size({"width": 375, "height": 812})
    page.get_by_role("button", name="Open navigation", exact=True).click()
    drawer = page.locator("#ak-mobile-navigation")
    expect(drawer).to_be_visible()
    mobile_docs = page.get_by_role("link", name="OpenAPI documentation", exact=True)
    mobile_system = page.get_by_role("button", name="Open system menu", exact=True)
    docs_mobile_box = mobile_docs.bounding_box()
    system_mobile_box = mobile_system.bounding_box()
    assert docs_mobile_box and system_mobile_box
    assert docs_mobile_box["height"] >= 44 and system_mobile_box["height"] >= 44
    assert docs_mobile_box["x"] < system_mobile_box["x"]
    mobile_system.click()
    expect(page.locator("#ak-mobile-system-navigation")).to_be_visible()
    page.get_by_text("System Settings", exact=True).last.click()
    mobile_dictionaries = page.locator("#ak-mobile-system-navigation").get_by_role(
        "link", name="Dictionaries", exact=True
    )
    expect(mobile_dictionaries).to_be_visible()
    mobile_dictionaries.scroll_into_view_if_needed()
    mobile_dictionaries.focus()
    expect(mobile_dictionaries).to_be_focused()
    page.wait_for_timeout(300)
    page.screenshot(path=SCREENSHOTS / "admin-system-drawer.en-US.375.png")
    assert page.evaluate("document.documentElement.scrollWidth <= document.documentElement.clientWidth")

    violations = severe_axe_violations(page)
    assert not violations, violations
    assert not console_errors, console_errors
    return {
        "primary_labels": primary_labels,
        "desktop_utility_order": ["documentation", "system"],
        "collapsed_button_widths": [collapsed_docs_box["width"], collapsed_system_box["width"]],
        "mobile_touch_heights": [docs_mobile_box["height"], system_mobile_box["height"]],
        "system_popover_style": popover_style,
        "reduced_motion_transition_duration": duration,
        "horizontal_overflow": False,
        "axe_serious_or_critical": violations,
        "console_errors": console_errors,
    }


def main() -> None:
    SCREENSHOTS.mkdir(parents=True, exist_ok=True)
    EVIDENCE.parent.mkdir(parents=True, exist_ok=True)
    evidence: dict[str, object] = {}
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        # The response CSP is asserted above. Playwright bypasses it only in
        # this isolated context so the local axe bundle can be injected.
        for locale, evidence_key in (("zh-CN", "openapi_zh_CN"), ("en-US", "openapi_en_US")):
            docs_context = browser.new_context(bypass_csp=True, viewport={"width": 1440, "height": 900})
            docs_context.add_cookies([
                {"name": "ak_admin_probe", "value": "must-not-leave-docs", "url": BASE_URL}
            ])
            evidence[evidence_key] = exercise_openapi(docs_context.new_page(), locale)
            docs_context.close()

        if os.environ.get("AK_E2E_SKIP_SHELL") != "1":
            shell_context = browser.new_context(
                bypass_csp=True,
                locale="zh-CN",
                viewport={"width": 1440, "height": 900},
            )
            evidence["shell"] = exercise_shell(shell_context.new_page())
            shell_context.close()
        browser.close()

    EVIDENCE.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"status": "passed", "evidence": str(EVIDENCE)}, ensure_ascii=False))


if __name__ == "__main__":
    main()
