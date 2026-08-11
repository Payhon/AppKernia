from __future__ import annotations

import json
import os
import re
from pathlib import Path

from playwright.sync_api import Page, expect, sync_playwright


ROOT = Path(__file__).resolve().parents[3]
BASE_URL = os.environ.get("AK_E2E_BASE_URL", "http://localhost:4174").rstrip("/")
EMAIL = os.environ.get("AK_E2E_EMAIL", "codex-e2e@appkernia.test")
PASSWORD = os.environ["AK_E2E_PASSWORD"]
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
ARTIFACTS = ROOT / "apps" / "ak-admin" / "artifacts" / "ui-ux-pro-max" / "sidebar-three-state-controls"
SCREENSHOTS = ARTIFACTS / "screenshots"
OUTPUT = ROOT / "output" / "playwright" / "sidebar-three-state-controls.evidence.json"


def visible_popup_count(page: Page) -> int:
    return page.locator(".ak-navigation-submenu-popup").evaluate_all(
        """elements => elements.filter(element => {
          const style = getComputedStyle(element)
          const rect = element.getBoundingClientRect()
          return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0
        }).length"""
    )


def wait_for_popup_count(page: Page, count: int) -> None:
    page.wait_for_function(
        """expected => [...document.querySelectorAll('.ak-navigation-submenu-popup')].filter(element => {
          const style = getComputedStyle(element)
          const rect = element.getBoundingClientRect()
          return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0
        }).length === expected""",
        arg=count,
    )


def shell_metrics(page: Page) -> dict[str, object]:
    return page.evaluate(
        """() => {
          const sider = document.querySelector('.ak-desktop-sider')
          const handle = document.querySelector('.ak-sider-collapse-handle')
          const hideHandle = document.querySelector('.ak-sider-hide-handle')
          const edgeRestore = document.querySelector('.ak-sider-hidden-restore')
          const headerRestore = document.querySelector('.ak-sidebar-header-restore')
          const siderRect = sider?.getBoundingClientRect()
          const handleRect = handle?.getBoundingClientRect()
          const hideRect = hideHandle?.getBoundingClientRect()
          const edgeRect = edgeRestore?.getBoundingClientRect()
          return {
            viewport: { width: innerWidth, height: innerHeight },
            collapsed: sider?.classList.contains('ant-layout-sider-collapsed') ?? null,
            storedMode: localStorage.getItem('ak.admin.sidebar-mode.v1'),
            sider: siderRect ? { left: siderRect.left, right: siderRect.right, width: siderRect.width } : null,
            handle: handleRect ? {
              left: handleRect.left,
              right: handleRect.right,
              top: handleRect.top,
              bottom: handleRect.bottom,
              width: handleRect.width,
              height: handleRect.height,
              centerY: handleRect.top + handleRect.height / 2,
            } : null,
            hideHandle: hideRect ? {
              left: hideRect.left,
              right: hideRect.right,
              top: hideRect.top,
              bottom: hideRect.bottom,
              width: hideRect.width,
              height: hideRect.height,
            } : null,
            edgeRestore: edgeRect ? {
              left: edgeRect.left,
              right: edgeRect.right,
              top: edgeRect.top,
              bottom: edgeRect.bottom,
              width: edgeRect.width,
              height: edgeRect.height,
              centerY: edgeRect.top + edgeRect.height / 2,
              opacity: getComputedStyle(edgeRestore).opacity,
            } : null,
            headerRestore: headerRestore ? {
              visible: headerRestore.getBoundingClientRect().width > 0,
              isFirstAction: headerRestore.parentElement?.firstElementChild === headerRestore,
            } : null,
          }
        }"""
    )


def control_style(page: Page, selector: str) -> dict[str, str]:
    return page.locator(selector).evaluate(
        """element => {
          const style = getComputedStyle(element)
          return {
            backgroundColor: style.backgroundColor,
            borderRightColor: style.borderRightColor,
            color: style.color,
            opacity: style.opacity,
            hovered: element.matches(':hover'),
            hitClass: document.elementFromPoint(
              element.getBoundingClientRect().left + element.getBoundingClientRect().width / 2,
              element.getBoundingClientRect().top + element.getBoundingClientRect().height / 2,
            )?.getAttribute('class') || '',
          }
        }"""
    )


def popup_metrics(page: Page) -> list[dict[str, object]]:
    return page.locator(".ak-navigation-submenu-popup").evaluate_all(
        """elements => elements.filter(element => {
          const style = getComputedStyle(element)
          const rect = element.getBoundingClientRect()
          return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0
        }).map(element => {
          const menu = element.querySelector(':scope > .ant-menu')
          const style = menu ? getComputedStyle(menu) : null
          const rect = menu?.getBoundingClientRect()
          return {
            text: (element.textContent || '').trim().slice(0, 240),
            rect: rect ? { left: rect.left, right: rect.right, top: rect.top, bottom: rect.bottom, width: rect.width, height: rect.height } : null,
            backgroundColor: style?.backgroundColor || '',
            borderRadius: style?.borderRadius || '',
            backdropFilter: style?.backdropFilter || '',
            boxShadow: style?.boxShadow || '',
          }
        })"""
    )


def run_axe(page: Page) -> list[dict[str, object]]:
    page.add_script_tag(path=str(AXE))
    result = page.evaluate("async () => await axe.run({ resultTypes: ['violations'] })")
    severe = [
        {"id": item["id"], "impact": item["impact"], "nodes": len(item["nodes"])}
        for item in result["violations"]
        if item["impact"] in ("critical", "serious")
    ]
    assert not severe, severe
    return severe


def verify_three_state_sidebar(
    page: Page,
    locale: str,
    *,
    verify_route_persistence: bool,
    restore_via: str,
) -> dict[str, object]:
    labels = {
        "zh-CN": {
            "collapse": "收起导航",
            "expand": "展开导航",
            "hide": "完全隐藏导航",
            "system": "打开系统菜单",
            "system_settings": "系统设置",
            "leaf": "字典管理",
        },
        "en-US": {
            "collapse": "Collapse navigation",
            "expand": "Expand navigation",
            "hide": "Hide navigation completely",
            "system": "Open system menu",
            "system_settings": "System Settings",
            "leaf": "Dictionaries",
        },
    }[locale]
    sider = page.locator(".ak-desktop-sider")
    handle = page.get_by_role("button", name=labels["collapse"], exact=True)
    sider.hover(position={"x": 120, "y": 360})
    expect(handle).to_be_visible()
    handle.click()
    expect(sider).to_have_class(re.compile(r".*ant-layout-sider-collapsed.*"))
    page.wait_for_timeout(250)

    collapsed = shell_metrics(page)
    assert collapsed["collapsed"] is True
    assert collapsed["sider"]["right"] == 80
    assert collapsed["handle"]["left"] == collapsed["sider"]["right"]
    assert collapsed["handle"]["width"] == 18
    assert abs(collapsed["handle"]["centerY"] - collapsed["viewport"]["height"] / 2) <= 0.5
    assert collapsed["storedMode"] == "collapsed"
    assert visible_popup_count(page) == 0

    expand = page.get_by_role("button", name=labels["expand"], exact=True)
    expect(expand).to_be_visible()
    normal_handle_style = control_style(page, ".ak-sider-collapse-handle")
    expand.hover()
    page.wait_for_timeout(200)
    hover_handle_style = control_style(page, ".ak-sider-collapse-handle")
    assert normal_handle_style["backgroundColor"] == "rgba(255, 255, 255, 0.94)"
    assert hover_handle_style["backgroundColor"] == "rgb(23, 23, 23)", hover_handle_style
    assert hover_handle_style["color"] == "rgb(255, 255, 255)", hover_handle_style
    assert hover_handle_style["borderRightColor"] == "rgb(23, 23, 23)", hover_handle_style

    system = page.get_by_role("button", name=labels["system"], exact=True)
    system.click()
    expect(page.locator(".ak-system-menu-popover")).to_be_visible()
    second_level = page.locator(".ak-system-menu-popover").evaluate(
        """element => {
          const rect = element.getBoundingClientRect()
          const style = getComputedStyle(element.querySelector('.ant-popover-inner'))
          return [{
            text: (element.textContent || '').trim().slice(0, 240),
            rect: { left: rect.left, right: rect.right, top: rect.top, bottom: rect.bottom, width: rect.width, height: rect.height },
            backgroundColor: style.backgroundColor,
            borderRadius: style.borderRadius,
            backdropFilter: style.backdropFilter,
            boxShadow: style.boxShadow,
          }]
        }"""
    )

    group = page.locator("#ak-desktop-system-navigation .ant-menu-submenu-title").filter(
        has_text=labels["system_settings"]
    )
    expect(group).to_be_visible()
    group.hover()
    wait_for_popup_count(page, 1)
    third_level = popup_metrics(page)
    assert third_level, third_level

    leaf = page.locator(".ak-navigation-submenu-popup .ant-menu-item").filter(has_text=labels["leaf"])
    expect(leaf).to_be_visible()
    leaf.hover()
    wait_for_popup_count(page, 1)
    page.screenshot(path=SCREENSHOTS / f"{locale}-collapsed-third-level-1440.png")

    if verify_route_persistence:
        leaf.get_by_role("link", name=labels["leaf"], exact=True).click()
        page.wait_for_url(re.compile(r"/system/settings/dictionaries(?:\?.*)?$"))
        page.wait_for_timeout(250)
        route_navigation = shell_metrics(page)
        assert route_navigation["collapsed"] is True
        assert route_navigation["sider"]["right"] == 80
        assert route_navigation["storedMode"] == "collapsed"
    else:
        page.keyboard.press("Escape")
        expect(system).to_be_focused()
        wait_for_popup_count(page, 0)
        route_navigation = None

    sider.hover(position={"x": 40, "y": 360})
    hide = page.get_by_role("button", name=labels["hide"], exact=True)
    expect(hide).to_be_visible()
    hide.click()
    expect(page.locator(".ak-desktop-sider")).to_have_count(0)
    edge_restore = page.locator(".ak-sider-hidden-restore")
    header_restore = page.locator(".ak-sidebar-header-restore")
    expect(edge_restore).to_be_visible()
    expect(header_restore).to_be_visible()
    hidden = shell_metrics(page)
    assert hidden["storedMode"] == "hidden"
    assert hidden["edgeRestore"]["left"] == 0
    assert hidden["edgeRestore"]["width"] == 18
    assert hidden["edgeRestore"]["opacity"] == "0.48"
    assert abs(hidden["edgeRestore"]["centerY"] - hidden["viewport"]["height"] / 2) <= 0.5
    assert hidden["headerRestore"]["isFirstAction"] is True
    edge_restore.hover()
    page.wait_for_timeout(200)
    edge_hover_style = control_style(page, ".ak-sider-hidden-restore")
    assert edge_hover_style["opacity"] == "1"
    page.screenshot(path=SCREENSHOTS / f"{locale}-hidden-1440.png")

    restore = header_restore if restore_via == "header" else edge_restore
    restore.click()
    restored_sider = page.locator(".ak-desktop-sider")
    expect(restored_sider).to_be_visible()
    expect(restored_sider).not_to_have_class(re.compile(r".*ant-layout-sider-collapsed.*"))
    expect(page.locator(".ak-sider-hidden-restore")).to_have_count(0)
    expect(page.locator(".ak-sidebar-header-restore")).to_have_count(0)
    page.wait_for_timeout(250)
    expanded = shell_metrics(page)
    assert expanded["storedMode"] == "expanded"
    assert expanded["sider"]["right"] == 248
    assert expanded["handle"]["left"] == expanded["sider"]["right"]
    assert abs(expanded["handle"]["centerY"] - expanded["viewport"]["height"] / 2) <= 0.5

    return {
        "collapsed": collapsed,
        "route_navigation": route_navigation,
        "normal_handle_style": normal_handle_style,
        "hover_handle_style": hover_handle_style,
        "hidden": hidden,
        "edge_hover_style": edge_hover_style,
        "expanded": expanded,
        "popup_counts": {"initial": 0, "utility_click": 1, "group_hover": 1, "escape": 0},
        "second_level": second_level,
        "third_level": third_level,
    }


def verify_mobile_drawer_when_desktop_hidden(page: Page) -> dict[str, object]:
    sider = page.locator(".ak-desktop-sider")
    sider.hover(position={"x": 120, "y": 360})
    page.get_by_role("button", name="Collapse navigation", exact=True).click()
    expect(sider).to_have_class(re.compile(r".*ant-layout-sider-collapsed.*"))
    sider.hover(position={"x": 40, "y": 360})
    page.get_by_role("button", name="Hide navigation completely", exact=True).click()
    expect(page.locator(".ak-desktop-sider")).to_have_count(0)

    page.set_viewport_size({"width": 375, "height": 812})
    expect(page.locator(".ak-sider-hidden-restore")).to_be_hidden()
    expect(page.locator(".ak-sidebar-header-restore")).to_be_hidden()
    mobile_toggle = page.get_by_role("button", name="Open navigation", exact=True)
    expect(mobile_toggle).to_be_visible()
    mobile_toggle.click()
    drawer = page.locator("#ak-mobile-navigation")
    expect(drawer).to_be_visible()
    page.wait_for_timeout(350)
    drawer_rect = drawer.evaluate(
        """element => {
          const wrapper = element.closest('.ant-drawer-content-wrapper')
          const rect = wrapper?.getBoundingClientRect()
          return rect ? { left: rect.left, right: rect.right, width: rect.width } : null
        }"""
    )
    assert drawer_rect is not None
    assert drawer_rect["left"] == 0
    assert drawer_rect["width"] == 280
    page.get_by_role("button", name="Open system menu", exact=True).click()
    expect(page.locator("#ak-mobile-system-navigation")).to_be_visible()
    page.get_by_text("System Settings", exact=True).last.click()
    expect(page.get_by_role("link", name="Dictionaries", exact=True)).to_be_visible()
    page.screenshot(path=SCREENSHOTS / "en-US-hidden-mobile-drawer-375.png")
    overflow = page.evaluate(
        "() => document.documentElement.scrollWidth > document.documentElement.clientWidth"
    )
    assert overflow is False
    page.locator(".ant-drawer-close").click()
    expect(drawer).to_be_hidden()

    result = {
        "viewport": {"width": 375, "height": 812},
        "storedMode": page.evaluate("localStorage.getItem('ak.admin.sidebar-mode.v1')"),
        "edgeRestoreHidden": True,
        "headerRestoreHidden": True,
        "drawerOpened": True,
        "drawerRect": drawer_rect,
        "horizontalOverflow": overflow,
    }
    page.set_viewport_size({"width": 1440, "height": 900})
    return result


def main() -> None:
    SCREENSHOTS.mkdir(parents=True, exist_ok=True)
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    console_errors: list[str] = []
    evidence: dict[str, object] = {}

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
        page = context.new_page()
        page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
        page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        page.evaluate("localStorage.setItem('ak.admin.sidebar-mode.v1', 'expanded')")
        page.reload(wait_until="networkidle")
        page.get_by_label("账号", exact=True).fill(EMAIL)
        page.get_by_label("密码", exact=True).fill(PASSWORD)
        page.locator('button[type="submit"]').click()
        page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))
        page.get_by_role("heading", name="Dashboard", exact=True).wait_for()
        if page.locator("html").get_attribute("lang") != "zh-CN":
            page.get_by_label("Display language", exact=True).click()
            page.get_by_role("menuitem", name="简体中文", exact=True).click()
            expect(page.locator("html")).to_have_attribute("lang", "zh-CN")

        evidence["zh-CN"] = verify_three_state_sidebar(
            page,
            "zh-CN",
            verify_route_persistence=True,
            restore_via="header",
        )
        page.get_by_label("显示语言", exact=True).click()
        page.get_by_role("menuitem", name="English", exact=True).click()
        expect(page.locator("html")).to_have_attribute("lang", "en-US")
        evidence["en-US"] = verify_three_state_sidebar(
            page,
            "en-US",
            verify_route_persistence=False,
            restore_via="edge",
        )
        evidence["axe_serious_or_critical"] = run_axe(page)
        evidence["mobile_hidden_drawer"] = verify_mobile_drawer_when_desktop_hidden(page)
        evidence["horizontal_overflow"] = page.evaluate(
            "() => document.documentElement.scrollWidth > document.documentElement.clientWidth"
        )
        evidence["console_errors"] = console_errors
        assert evidence["horizontal_overflow"] is False
        assert not console_errors, console_errors

        page.get_by_label("Display language", exact=True).click()
        page.get_by_role("menuitem", name="简体中文", exact=True).click()
        expect(page.locator("html")).to_have_attribute("lang", "zh-CN")
        page.locator(".ak-account-trigger").click()
        page.locator(".ak-account-dropdown .ant-dropdown-menu-item-danger").click()
        page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/login"))
        page.wait_for_timeout(250)
        evidence["console_errors"] = console_errors
        assert not console_errors, console_errors
        browser.close()

    OUTPUT.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"status": "passed", "evidence": str(OUTPUT)}, ensure_ascii=False))


if __name__ == "__main__":
    main()
