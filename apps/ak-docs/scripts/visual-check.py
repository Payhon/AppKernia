#!/usr/bin/env python3
"""Production-preview visual, responsive, and accessibility checks."""

from __future__ import annotations

import asyncio
import json
import os
from pathlib import Path

from playwright.async_api import BrowserContext, async_playwright


ROOT = Path(__file__).resolve().parents[3]
EVIDENCE = ROOT / "apps/ak-docs/artifacts/ui-ux-pro-max/AKDOCS-002/screenshots"
AXE_PATH = ROOT / "apps/ak-admin/node_modules/axe-core/axe.min.js"
BASE_URL = os.environ.get("AK_DOCS_PREVIEW_URL", "http://127.0.0.1:4175/AppKernia").rstrip("/")

SAMPLES = (
    ("home.zh-CN.light.375", "/", 375, 812, "light"),
    ("home.zh-CN.light.768", "/", 768, 1024, "light"),
    ("home.zh-CN.light.1024", "/", 1024, 900, "light"),
    ("home.zh-CN.light.1440", "/", 1440, 900, "light"),
    ("home.zh-CN.light.1920", "/", 1920, 1080, "light"),
    ("home.en-US.dark.1440", "/en-US/", 1440, 900, "dark"),
    ("concepts.en-US.light.1920", "/en-US/concepts/", 1920, 1080, "light"),
    ("guide.en-US.dark.1440", "/en-US/guide/", 1440, 900, "dark"),
)


async def configure_theme(context: BrowserContext, theme: str) -> None:
    await context.add_init_script(
        f"""
        localStorage.setItem('rspress-theme-appearance', {json.dumps(theme)});
        localStorage.setItem('rspress-visited', '1');
        document.documentElement.classList.toggle('dark', {json.dumps(theme)} === 'dark');
        """
    )


async def inspect_sample(browser, name: str, route: str, width: int, height: int, theme: str):
    locale = "zh-CN" if ".zh-CN." in name else "en-US"
    context = await browser.new_context(
        viewport={"width": width, "height": height}, locale=locale
    )
    await configure_theme(context, theme)
    page = await context.new_page()
    console_errors: list[str] = []
    failed_responses: list[dict[str, object]] = []
    failed_requests: list[dict[str, object]] = []
    navigations: list[str] = []
    page.on("console", lambda message: console_errors.append(message.text) if message.type == "error" else None)
    page.on(
        "response",
        lambda response: failed_responses.append({"url": response.url, "status": response.status})
        if response.status >= 400
        else None,
    )
    page.on(
        "requestfailed",
        lambda request: failed_requests.append(
            {
                "url": request.url,
                "failure": request.failure,
                "resourceType": request.resource_type,
                "navigation": request.is_navigation_request(),
            }
        ),
    )
    page.on(
        "framenavigated",
        lambda frame: navigations.append(frame.url) if frame == page.main_frame else None,
    )

    response = await page.goto(f"{BASE_URL}{route}", wait_until="networkidle")
    # Rspress loads the local search index after hydration. Let that request
    # settle before assertions so closing the isolated context cannot cancel it.
    await page.wait_for_timeout(1000)
    status = response.status if response else 0
    metrics = await page.evaluate(
        """
        () => {
          const images = [...document.images];
          const sidebar = document.querySelector('.rp-doc-layout__sidebar');
          const outline = document.querySelector('.rp-doc-layout__outline');
          const paragraph = document.querySelector('.rp-doc p');
          const sidebarRect = sidebar?.getBoundingClientRect();
          const outlineRect = outline?.getBoundingClientRect();
          return {
            h1Count: document.querySelectorAll('h1').length,
            viewportWidth: window.innerWidth,
            scrollWidth: document.documentElement.scrollWidth,
            brokenImages: images
              .filter(image => !image.complete || image.naturalWidth === 0)
              .map(image => image.currentSrc || image.src),
            overflowElements: [...document.querySelectorAll('body *')]
              .map(element => {
                const rect = element.getBoundingClientRect();
                return {
                  selector: element.id ? `#${element.id}` : `.${[...element.classList].join('.')}`,
                  left: rect.left,
                  right: rect.right,
                  width: rect.width,
                };
              })
              .filter(item => item.left < -1 || item.right > window.innerWidth + 1)
              .slice(0, 12),
            shell: sidebarRect && outlineRect ? {
              left: sidebarRect.left,
              right: window.innerWidth - outlineRect.right,
              width: outlineRect.right - sidebarRect.left,
            } : null,
            paragraphWidth: paragraph?.getBoundingClientRect().width ?? null,
          };
        }
        """
    )
    await page.add_script_tag(path=str(AXE_PATH))
    axe = await page.evaluate(
        """
        async () => {
          const result = await axe.run(document, {
            runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'] },
          });
          return result.violations
            .filter(item => item.impact === 'serious' || item.impact === 'critical')
            .map(item => ({
              id: item.id,
              impact: item.impact,
              nodes: item.nodes.map(node => ({
                target: node.target,
                html: node.html,
                failureSummary: node.failureSummary,
              })),
            }));
        }
        """
    )

    await page.screenshot(path=str(EVIDENCE / f"{name}.png"), full_page=False)
    errors: list[str] = []
    if status >= 400 or status == 0:
        errors.append(f"navigation status {status}")
    if metrics["h1Count"] != 1:
        errors.append(f"expected one H1, got {metrics['h1Count']}")
    if metrics["scrollWidth"] > metrics["viewportWidth"] + 1:
        errors.append(f"horizontal overflow {metrics['scrollWidth']} > {metrics['viewportWidth']}")
    if metrics["brokenImages"]:
        errors.append(f"broken images: {metrics['brokenImages']}")
    if console_errors:
        errors.append(f"console errors: {console_errors}")
    if failed_responses:
        errors.append(f"failed responses: {failed_responses}")
    non_aborted_requests = [
        request for request in failed_requests if request["failure"] != "net::ERR_ABORTED"
    ]
    if non_aborted_requests:
        errors.append(f"failed requests: {non_aborted_requests}")
    if axe:
        errors.append(f"axe serious/critical: {axe}")
    if width == 1920 and "/concepts/" in route:
        shell = metrics["shell"]
        if not shell:
            errors.append("wide documentation shell not measurable")
        else:
            if abs(shell["left"] - shell["right"]) > 2:
                errors.append(f"wide shell is not centered: {shell}")
            if abs(shell["width"] - 1488) > 2:
                errors.append(f"wide shell is not 1488px: {shell}")
        if metrics["paragraphWidth"] and metrics["paragraphWidth"] > 820:
            errors.append(f"prose line is too wide: {metrics['paragraphWidth']}px")

    result = {
        "name": name,
        "url": page.url,
        "status": status,
        "theme": theme,
        "metrics": metrics,
        "axeSeriousCritical": axe,
        "abortedRequests": failed_requests,
        "navigations": navigations,
        "errors": errors,
    }
    await context.close()
    return result


async def main() -> int:
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    if not AXE_PATH.exists():
        raise FileNotFoundError(f"axe-core is missing: {AXE_PATH}")

    async with async_playwright() as playwright:
        browser = await playwright.chromium.launch()
        results = []
        sample_filter = os.environ.get("AK_DOCS_SAMPLE", "")
        for sample in SAMPLES:
            if sample_filter and sample_filter not in sample[0]:
                continue
            results.append(await inspect_sample(browser, *sample))
        await browser.close()

    print(json.dumps(results, indent=2, ensure_ascii=False))
    return 1 if any(result["errors"] for result in results) else 0


if __name__ == "__main__":
    raise SystemExit(asyncio.run(main()))
