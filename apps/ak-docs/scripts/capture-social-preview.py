#!/usr/bin/env python3
"""Capture the final homepage Hero as the social sharing image."""

from __future__ import annotations

import asyncio
import os
from pathlib import Path

from playwright.async_api import async_playwright


ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "apps/ak-docs/docs/public/social-preview.png"
BASE_URL = os.environ.get(
    "AK_DOCS_PREVIEW_URL", "http://127.0.0.1:4175/AppKernia"
).rstrip("/")


async def main() -> None:
    async with async_playwright() as playwright:
        browser = await playwright.chromium.launch()
        context = await browser.new_context(
            viewport={"width": 1200, "height": 630}, locale="zh-CN"
        )
        await context.add_init_script(
            """
            localStorage.setItem('rspress-theme-appearance', 'light');
            localStorage.setItem('rspress-visited', '1');
            """
        )
        page = await context.new_page()
        response = await page.goto(f"{BASE_URL}/", wait_until="networkidle")
        if response is None or response.status >= 400:
            raise RuntimeError("Failed to load the documentation homepage")
        await page.wait_for_timeout(1000)
        await page.screenshot(path=str(OUTPUT), full_page=False)
        await browser.close()

    print(f"Captured {OUTPUT} (1200x630)")


if __name__ == "__main__":
    asyncio.run(main())
