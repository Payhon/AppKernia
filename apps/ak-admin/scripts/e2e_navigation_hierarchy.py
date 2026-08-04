from __future__ import annotations

import json
import os
from pathlib import Path

from playwright.sync_api import Page, expect, sync_playwright


ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "output" / "playwright"
AXE = ROOT / "apps" / "ak-admin" / "node_modules" / "axe-core" / "axe.min.js"
BASE_URL = os.environ.get("AK_E2E_BASE_URL", "http://127.0.0.1:4173").rstrip("/")
EMAIL = os.environ["AK_E2E_EMAIL"]
PASSWORD = os.environ["AK_E2E_PASSWORD"]


def direct_labels(page: Page, selector: str) -> list[str]:
    return page.locator(selector).evaluate_all(
        """elements => elements.map(element => element.textContent?.trim() || '')"""
    )


def assert_desktop_tree(page: Page, locale: str) -> dict[str, list[str]]:
    if locale == "zh-CN":
        root = ["Dashboard", "系统"]
        groups = ["系统设置", "用户管理", "权限设置", "文件存储", "通知中心", "任务集成", "审计安全", "运行监控"]
        system_settings = ["系统配置", "字典管理", "地区管理", "模块信息"]
        users = ["部门", "用户", "岗位"]
        access = ["角色", "权限目录", "菜单"]
    else:
        root = ["Dashboard", "System"]
        groups = ["System Settings", "User Management", "Access Control", "File Storage", "Notification Center", "Jobs & Integrations", "Audit & Security", "Monitoring"]
        system_settings = ["System Configuration", "Dictionaries", "Regions", "Modules"]
        users = ["Departments", "Users", "Positions"]
        access = ["Roles", "Permission Catalog", "Menus"]

    root_selector = ".ak-desktop-sider .ant-menu-root > li > .ant-menu-title-content, .ak-desktop-sider .ant-menu-root > li > .ant-menu-submenu-title > .ant-menu-title-content"
    expect(page.locator(".ak-desktop-sider .ant-menu-root")).to_be_visible()
    actual_root = direct_labels(page, root_selector)
    assert actual_root == root, actual_root

    system_title = page.locator(".ak-desktop-sider .ant-menu-root > .ant-menu-submenu > .ant-menu-submenu-title")
    if system_title.get_attribute("aria-expanded") != "true":
        system_title.click()
    group_selector = ".ak-desktop-sider .ant-menu-root > .ant-menu-submenu > ul > .ant-menu-submenu > .ant-menu-submenu-title > .ant-menu-title-content"
    expect(page.locator(group_selector).first).to_be_visible()
    actual_groups = direct_labels(page, group_selector)
    assert actual_groups == groups, actual_groups

    def expand_and_read(group: str) -> list[str]:
        group_title = page.locator(".ak-desktop-sider .ant-menu-submenu-title").filter(has_text=group).last
        if group_title.get_attribute("aria-expanded") != "true":
            group_title.click()
        group_item = group_title.locator("xpath=..")
        return direct_labels(group_item, ":scope > ul > .ant-menu-item > .ant-menu-title-content")

    actual_settings = expand_and_read(groups[0])
    actual_users = expand_and_read(groups[1])
    actual_access = expand_and_read(groups[2])
    assert actual_settings == system_settings, actual_settings
    assert actual_users == users, actual_users
    assert actual_access == access, actual_access

    return {
        "root": actual_root,
        "groups": actual_groups,
        "system_settings": actual_settings,
        "user_management": actual_users,
        "access_control": actual_access,
    }


def inspect_menu_icons(page: Page, scope: str) -> dict[str, object]:
    group_titles = page.locator(
        f"{scope} .ant-menu-root > .ant-menu-submenu > ul > .ant-menu-submenu > .ant-menu-submenu-title"
    )
    for index in range(group_titles.count()):
        title = group_titles.nth(index)
        if title.get_attribute("aria-expanded") != "true":
            title.click()
    page.wait_for_timeout(500)

    rows = page.locator(f"{scope} .ant-menu-root").evaluate(
        """root => [...root.querySelectorAll('li')]
          .filter(row => row.offsetParent !== null)
          .map(row => {
            const target = row.matches('.ant-menu-item')
              ? row
              : row.querySelector(':scope > .ant-menu-submenu-title')
            const icon = target?.querySelector(':scope > .anticon')
            const label = target?.querySelector(':scope > .ant-menu-title-content')
            const iconRect = icon?.getBoundingClientRect()
            const labelRect = label?.getBoundingClientRect()
            return {
              code: target?.getAttribute('data-menu-id') || '',
              label: label?.textContent?.trim() || '',
              iconCount: icon ? 1 : 0,
              iconWidth: iconRect ? Math.round(iconRect.width * 100) / 100 : 0,
              gap: iconRect && labelRect ? Math.round((labelRect.left - iconRect.right) * 100) / 100 : -1,
            }
          })"""
    )
    assert 32 <= len(rows) <= 35, rows
    assert all(row["iconCount"] == 1 for row in rows), rows
    assert all(abs(row["iconWidth"] - 16) <= 0.5 for row in rows), rows
    assert all(abs(row["gap"] - 8) <= 0.5 for row in rows), rows
    return {
        "visible_items": len(rows),
        "items_with_icons": sum(row["iconCount"] for row in rows),
        "icon_widths": sorted(set(row["iconWidth"] for row in rows)),
        "icon_label_gaps": sorted(set(row["gap"] for row in rows)),
        "labels": [row["label"] for row in rows],
    }


def run_axe(page: Page) -> list[dict[str, object]]:
    page.wait_for_timeout(500)
    page.add_script_tag(path=str(AXE))
    result = page.evaluate("async () => await axe.run({ resultTypes: ['violations'] })")
    severe = [
        {
            "id": item["id"],
            "impact": item["impact"],
            "nodes": len(item["nodes"]),
            "targets": [node["target"] for node in item["nodes"]],
            "summaries": [node["failureSummary"] for node in item["nodes"]],
        }
        for item in result["violations"]
        if item["impact"] in ("critical", "serious")
    ]
    assert not severe, severe
    return severe


def main() -> None:
    OUTPUT.mkdir(parents=True, exist_ok=True)
    evidence: dict[str, object] = {}
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        context = browser.new_context(locale="zh-CN", viewport={"width": 1440, "height": 900})
        page = context.new_page()
        page.goto(f"{BASE_URL}/login", wait_until="networkidle")
        if page.locator("html").get_attribute("lang") != "zh-CN":
            page.get_by_label("Display language").select_option(label="简体中文")
        page.get_by_label("账号").fill(EMAIL)
        page.get_by_label("密码").fill(PASSWORD)
        page.locator('button[type="submit"]').click()
        page.wait_for_url(lambda url: url.startswith(f"{BASE_URL}/dashboard"))
        page.get_by_role("heading", name="Dashboard").wait_for()
        if page.locator("html").get_attribute("lang") != "zh-CN":
            page.get_by_label("Display language").select_option(label="简体中文")
            expect(page.locator("html")).to_have_attribute("lang", "zh-CN")

        evidence["zh-CN"] = assert_desktop_tree(page, "zh-CN")
        evidence["zh-CN_icons"] = inspect_menu_icons(page, ".ak-desktop-sider")
        evidence["zh-CN_axe_serious_or_critical"] = run_axe(page)
        page.screenshot(path=OUTPUT / "admin-navigation-hierarchy.zh-CN.1440.png", full_page=True)
        page.evaluate("window.scrollTo(0, 0)")
        page.screenshot(path=OUTPUT / "admin-navigation-icons.zh-CN.1440.png")

        time_origin = page.evaluate("performance.timeOrigin")
        page.get_by_label("显示语言").select_option(label="English")
        expect(page.locator("html")).to_have_attribute("lang", "en-US")
        assert page.evaluate("performance.timeOrigin") == time_origin
        evidence["en-US"] = assert_desktop_tree(page, "en-US")
        evidence["en-US_icons"] = inspect_menu_icons(page, ".ak-desktop-sider")
        page.screenshot(path=OUTPUT / "admin-navigation-hierarchy.en-US.1440.png", full_page=True)
        page.evaluate("window.scrollTo(0, 0)")
        page.screenshot(path=OUTPUT / "admin-navigation-icons.en-US.1440.png")

        page.set_viewport_size({"width": 375, "height": 812})
        page.get_by_label("Open navigation").click()
        expect(page.locator(".ak-mobile-drawer .ant-menu-root")).to_be_visible()
        mobile_system = page.locator(".ak-mobile-drawer .ant-menu-root > .ant-menu-submenu > .ant-menu-submenu-title")
        if mobile_system.get_attribute("aria-expanded") != "true":
            mobile_system.click()
        expect(page.get_by_text("System Settings", exact=True).last).to_be_visible()
        page.wait_for_timeout(500)
        page.screenshot(path=OUTPUT / "admin-navigation-hierarchy.en-US.375.png")
        page.screenshot(path=OUTPUT / "admin-navigation-icons.en-US.375.png")
        evidence["mobile"] = {"width": 375, "drawer_tree_visible": True}
        browser.close()

    evidence_path = OUTPUT / "admin-navigation-hierarchy.evidence.json"
    evidence_path.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"evidence": str(evidence_path), "locales": ["zh-CN", "en-US"], "status": "passed"}, ensure_ascii=False))


if __name__ == "__main__":
    main()
