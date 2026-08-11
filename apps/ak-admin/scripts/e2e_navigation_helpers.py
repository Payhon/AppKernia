from __future__ import annotations

from playwright.sync_api import Page, expect


def _is_english(page: Page) -> bool:
    return page.locator("html").get_attribute("lang") == "en-US"


def open_system_menu(page: Page, *, mobile: bool = False):
    label = "Open system menu" if _is_english(page) else "打开系统菜单"
    scope = page.locator(".ak-mobile-drawer") if mobile else page.locator(".ak-desktop-sider")
    trigger = scope.get_by_role("button", name=label, exact=True)
    expect(trigger).to_be_visible()
    if trigger.get_attribute("aria-expanded") != "true":
        trigger.click()
    menu_id = "#ak-mobile-system-navigation" if mobile else "#ak-desktop-system-navigation"
    menu = page.locator(menu_id)
    expect(menu).to_be_visible()
    return menu


def open_system_page(page: Page, path: str, group: str, *, mobile: bool = False) -> None:
    menu = open_system_menu(page, mobile=mobile)
    link = page.locator(f'.ak-system-menu-popover a[href="{path}"]:visible')
    if link.count() == 0:
        group_item = menu.get_by_role("menuitem", name=group, exact=True)
        expect(group_item).to_be_visible()
        if mobile:
            group_item.click()
        else:
            group_item.hover()
        expect(link).to_be_visible()
    link.click()
    page.wait_for_url(lambda value: value.split("?", 1)[0].endswith(path))
