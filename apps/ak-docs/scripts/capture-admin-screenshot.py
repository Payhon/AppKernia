#!/usr/bin/env python3
"""Capture a local, authenticated AppKernia Admin dashboard screenshot.

Credentials are accepted only through environment variables so this utility can
be used with the ignored docs/LOCAL_TEST_ACCESS.md file without checking secrets
into the repository or printing them to the terminal.
"""

from __future__ import annotations

import os
from pathlib import Path

from playwright.sync_api import sync_playwright


def required_env(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        raise SystemExit(f"missing required environment variable: {name}")
    return value


def main() -> None:
    base_url = os.environ.get("AK_SCREENSHOT_BASE_URL", "http://localhost:4174").rstrip("/")
    email = required_env("AK_SCREENSHOT_EMAIL")
    password = required_env("AK_SCREENSHOT_PASSWORD")
    output = Path(required_env("AK_SCREENSHOT_OUTPUT")).expanduser().resolve()
    output.parent.mkdir(parents=True, exist_ok=True)

    console_errors: list[str] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)
        context = browser.new_context(
            locale="zh-CN",
            viewport={"width": 1440, "height": 900},
            device_scale_factor=1,
            color_scheme="light",
        )
        page = context.new_page()
        page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
        page.goto(f"{base_url}/login", wait_until="networkidle", timeout=30_000)
        page.locator("#login-email").fill(email)
        page.locator("#login-password").fill(password)
        page.locator('button[type="submit"]').click()
        page.wait_for_url("**/dashboard**", timeout=30_000)
        page.get_by_role("heading", name="Dashboard", exact=True).wait_for(state="visible", timeout=30_000)
        page.locator("#dashboard-summary-title").wait_for(state="visible", timeout=30_000)
        page.locator("#dashboard-trends-title").wait_for(state="visible", timeout=30_000)
        page.wait_for_load_state("networkidle", timeout=30_000)
        page.locator(".ak-dashboard-section .ant-skeleton").first.wait_for(state="detached", timeout=30_000)
        page.screenshot(path=str(output), full_page=False)
        browser.close()

    if console_errors:
        raise SystemExit(f"dashboard emitted {len(console_errors)} console error(s)")
    print(f"captured authenticated dashboard: {output}")


if __name__ == "__main__":
    main()
