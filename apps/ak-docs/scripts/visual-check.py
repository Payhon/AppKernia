#!/usr/bin/env python3
"""Production-preview visual, responsive, and accessibility checks."""

from __future__ import annotations

import asyncio
import json
import os
from pathlib import Path

from playwright.async_api import (
    BrowserContext,
    TimeoutError as PlaywrightTimeoutError,
    async_playwright,
)


ROOT = Path(__file__).resolve().parents[3]
EVIDENCE_ID = os.environ.get("AK_DOCS_EVIDENCE_ID", "AKDOCS-005")
EVIDENCE = ROOT / f"apps/ak-docs/artifacts/ui-ux-pro-max/{EVIDENCE_ID}/screenshots"
AXE_PATH = ROOT / "apps/ak-admin/node_modules/axe-core/axe.min.js"
BASE_URL = os.environ.get("AK_DOCS_PREVIEW_URL", "http://127.0.0.1:4175/AppKernia").rstrip("/")

SAMPLES = (
    ("home.zh-CN.light.375", "/", 375, 812, "light"),
    ("home.zh-CN.light.768", "/", 768, 1024, "light"),
    ("home.zh-CN.light.1024", "/", 1024, 900, "light"),
    ("home.zh-CN.light.1440", "/", 1440, 900, "light"),
    ("home.zh-CN.light.1920", "/", 1920, 1080, "light"),
    ("home.en-US.dark.1440", "/en-US/", 1440, 900, "dark"),
    ("what-is-appkernia.zh-CN.light.1440", "/guide/what-is-appkernia", 1440, 1000, "light"),
    ("what-is-appkernia.en-US.dark.375", "/en-US/guide/what-is-appkernia", 375, 812, "dark"),
    ("architecture.zh-CN.light.1920", "/concepts/architecture", 1920, 1080, "light"),
    ("architecture.en-US.dark.1024", "/en-US/concepts/architecture", 1024, 900, "dark"),
    ("authentication.zh-CN.light.768", "/concepts/authentication", 768, 1024, "light"),
    ("concepts-index.zh-CN.light.1440", "/concepts/", 1440, 900, "light"),
    ("concepts-index.en-US.dark.1440", "/en-US/concepts/", 1440, 900, "dark"),
    ("authentication.en-US.dark.1440", "/en-US/concepts/authentication", 1440, 900, "dark"),
    ("permissions-tenancy.zh-CN.light.1440", "/concepts/permissions-tenancy", 1440, 900, "light"),
    ("permissions-tenancy.en-US.dark.1440", "/en-US/concepts/permissions-tenancy", 1440, 900, "dark"),
    ("internationalization.zh-CN.light.1440", "/concepts/internationalization", 1440, 900, "light"),
    ("internationalization.en-US.dark.1440", "/en-US/concepts/internationalization", 1440, 900, "dark"),
    ("api-index.zh-CN.light.1440", "/api/", 1440, 900, "light"),
    ("api-index.en-US.dark.1440", "/en-US/api/", 1440, 900, "dark"),
    ("mobile-components-index.zh-CN.light.1440", "/mobile-components/", 1440, 900, "light"),
    ("mobile-components-index.en-US.dark.1440", "/en-US/mobile-components/", 1440, 900, "dark"),
)

EXPECTED_DIAGRAMS = {
    "/concepts/": 2,
    "/en-US/concepts/": 2,
    "/concepts/architecture": 4,
    "/en-US/concepts/architecture": 4,
    "/concepts/authentication": 3,
    "/en-US/concepts/authentication": 3,
    "/concepts/permissions-tenancy": 1,
    "/en-US/concepts/permissions-tenancy": 1,
    "/concepts/internationalization": 1,
    "/en-US/concepts/internationalization": 1,
    "/api/": 1,
    "/en-US/api/": 1,
    "/mobile-components/": 1,
    "/en-US/mobile-components/": 1,
}


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
    diagram_render_timeout = False
    diagram_shells = await page.locator(".ak-diagram").count()
    if diagram_shells:
        try:
            await page.wait_for_function(
                """
                expected => document.querySelectorAll('.ak-diagram svg').length === expected
                """,
                arg=diagram_shells,
                timeout=15000,
            )
        except PlaywrightTimeoutError:
            diagram_render_timeout = True
    status = response.status if response else 0
    metrics = await page.evaluate(
        """
        () => {
          const images = [...document.images];
          const sidebar = document.querySelector('.rp-doc-layout__sidebar');
          const outline = document.querySelector('.rp-doc-layout__outline');
          const paragraph = document.querySelector('.rp-doc p');
          const hero = document.querySelector('.rp-home-hero');
          const heroImage = document.querySelector('.rp-home-hero__image');
          const heroLabel = document.querySelector('.rp-home-hero__title-brand');
          const sidebarRect = sidebar?.getBoundingClientRect();
          const outlineRect = outline?.getBoundingClientRect();
          const heroRect = hero?.getBoundingClientRect();
          const heroImageRect = heroImage?.getBoundingClientRect();
          const heroLabelRect = heroLabel?.getBoundingClientRect();
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
            diagramShells: document.querySelectorAll('.ak-diagram').length,
            diagramSvgs: document.querySelectorAll('.ak-diagram svg').length,
            diagramTitles: document.querySelectorAll('.ak-diagram svg title').length,
            diagramDescriptions: document.querySelectorAll('.ak-diagram svg desc').length,
            rawMermaidBlocks: document.querySelectorAll('pre code.language-mermaid, .language-mermaid').length,
            galleryImages: document.querySelectorAll('.ak-product-gallery img').length,
            homeSections: document.querySelectorAll('.ak-home-section').length,
            featureCards: document.querySelectorAll('.ak-feature-card').length,
            technologyCards: document.querySelectorAll('.ak-tech-logo-card').length,
            productSliders: document.querySelectorAll('.ak-product-slider').length,
            maturitySections: document.querySelectorAll('#maturity-title, .ak-maturity-grid').length,
            hero: heroRect && heroImageRect && heroLabelRect ? {
              width: heroRect.width,
              imageWidth: heroImageRect.width,
              imageRatio: heroImageRect.width / heroRect.width,
              labelHeight: heroLabelRect.height,
              labelBorderWidth: Number.parseFloat(getComputedStyle(heroLabel).borderTopWidth),
              backgroundImage: getComputedStyle(hero, '::before').backgroundImage,
            } : null,
          };
        }
        """
    )
    slider_interactions = None
    delivery_copy_hits: list[str] = []
    if route in {"/", "/en-US/"}:
        sliders = page.locator(".ak-product-slider")
        admin_slider = sliders.nth(0)
        mobile_slider = sliders.nth(1)
        admin_initial = await admin_slider.locator(".ak-product-slider__viewport img").get_attribute("alt")
        mobile_initial = await mobile_slider.locator(".ak-product-slider__viewport img").get_attribute("alt")
        await admin_slider.locator(".ak-product-slider__controls > button").last.click()
        admin_after_click = await admin_slider.locator(".ak-product-slider__viewport img").get_attribute("alt")
        mobile_keyboard_control = mobile_slider.locator(
            ".ak-product-slider__controls > button"
        ).first
        await mobile_keyboard_control.focus()
        await mobile_keyboard_control.press("ArrowRight")
        mobile_after_key = await mobile_slider.locator(".ak-product-slider__viewport img").get_attribute("alt")
        stable_admin = admin_after_click
        stable_mobile = mobile_after_key
        await page.wait_for_timeout(1200)
        slider_interactions = {
            "adminChangedAfterClick": admin_initial != admin_after_click,
            "mobileChangedAfterKeyboard": mobile_initial != mobile_after_key,
            "noAutomaticAdvance": stable_admin
            == await admin_slider.locator(".ak-product-slider__viewport img").get_attribute("alt")
            and stable_mobile
            == await mobile_slider.locator(".ak-product-slider__viewport img").get_attribute("alt"),
            "liveRegions": await page.locator(".ak-product-slider [aria-live='polite']").count(),
        }
        await page.evaluate(
            "document.activeElement instanceof HTMLElement && document.activeElement.blur()"
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

    await page.screenshot(
        path=str(EVIDENCE / f"{name}.png"),
        full_page=route in {"/", "/en-US/"},
    )
    if route in {"/", "/en-US/"} and width == 1440:
        await page.locator(".rp-home-hero").screenshot(
            path=str(EVIDENCE / f"{name}.hero.png")
        )
        await page.add_style_tag(content=".rp-nav { display: none !important; }")
        await page.locator(".ak-tech-stack-section").screenshot(
            path=str(EVIDENCE / f"{name}.technology.png")
        )
        await page.locator(".ak-product-tour").screenshot(
            path=str(EVIDENCE / f"{name}.product-tour.png")
        )
    errors: list[str] = []
    if status >= 400 or status == 0:
        errors.append(f"navigation status {status}")
    if metrics["h1Count"] != 1:
        errors.append(f"expected one H1, got {metrics['h1Count']}")
    if metrics["scrollWidth"] > metrics["viewportWidth"] + 1:
        errors.append(f"horizontal overflow {metrics['scrollWidth']} > {metrics['viewportWidth']}")
    if metrics["brokenImages"]:
        errors.append(f"broken images: {metrics['brokenImages']}")
    if diagram_render_timeout:
        errors.append(
            f"Mermaid render timeout: {metrics['diagramSvgs']} of {metrics['diagramShells']} diagrams"
        )
    if metrics["diagramShells"] != metrics["diagramSvgs"]:
        errors.append(
            f"expected one SVG per diagram shell: {metrics['diagramSvgs']} of {metrics['diagramShells']}"
        )
    if metrics["diagramSvgs"]:
        if metrics["diagramTitles"] != metrics["diagramSvgs"]:
            errors.append(
                f"diagram titles missing: {metrics['diagramTitles']} of {metrics['diagramSvgs']}"
            )
        if metrics["diagramDescriptions"] != metrics["diagramSvgs"]:
            errors.append(
                f"diagram descriptions missing: {metrics['diagramDescriptions']} of {metrics['diagramSvgs']}"
            )
    if metrics["rawMermaidBlocks"]:
        errors.append(f"raw Mermaid source blocks remain: {metrics['rawMermaidBlocks']}")
    expected_diagrams = EXPECTED_DIAGRAMS.get(route)
    if expected_diagrams is not None and metrics["diagramShells"] != expected_diagrams:
        errors.append(
            f"expected {expected_diagrams} diagram shells, got {metrics['diagramShells']}"
        )
    if "what-is-appkernia" in route and metrics["galleryImages"] != 8:
        errors.append(f"expected 8 product gallery images, got {metrics['galleryImages']}")
    if route in {"/", "/en-US/"}:
        hero = metrics["hero"]
        if not hero:
            errors.append("hero metrics are not measurable")
        else:
            if hero["labelBorderWidth"] != 0:
                errors.append(f"hero label border remains: {hero['labelBorderWidth']}px")
            if hero["labelHeight"] > 32:
                errors.append(f"hero label is too tall: {hero['labelHeight']}px")
            if hero["backgroundImage"] == "none":
                errors.append("hero gradient background is missing")
            if width >= 1200 and hero["imageRatio"] < 0.45:
                errors.append(f"hero product image is too small: {hero['imageRatio']:.3f}")
        if metrics["homeSections"] != 9:
            errors.append(f"expected 9 home sections, got {metrics['homeSections']}")
        if metrics["maturitySections"]:
            errors.append(f"maturity section remains: {metrics['maturitySections']}")
        if metrics["featureCards"] != 6:
            errors.append(f"expected 6 feature cards, got {metrics['featureCards']}")
        if metrics["technologyCards"] != 9:
            errors.append(f"expected 9 technology cards, got {metrics['technologyCards']}")
        if metrics["productSliders"] != 2:
            errors.append(f"expected 2 product sliders, got {metrics['productSliders']}")
        if not slider_interactions or not all(
            value is True
            for key, value in slider_interactions.items()
            if key != "liveRegions"
        ):
            errors.append(f"slider interaction failed: {slider_interactions}")
        if not slider_interactions or slider_interactions["liveRegions"] != 2:
            errors.append(f"expected 2 slider live regions: {slider_interactions}")
        home_text = (await page.locator(".ak-home-main").inner_text()).lower()
        delivery_phrases = (
            "honest maturity",
            "本机隔离",
            "运行证据",
            "验收",
            "不等同",
            "不代表",
            "local test environment",
            "physical-device acceptance",
            "platform evidence",
            "separate delivery from aspiration",
        )
        delivery_copy_hits = [phrase for phrase in delivery_phrases if phrase.lower() in home_text]
        if delivery_copy_hits:
            errors.append(f"delivery-report copy remains: {delivery_copy_hits}")
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
        "sliderInteractions": slider_interactions,
        "deliveryCopyHits": delivery_copy_hits,
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

    serialized_results = json.dumps(results, indent=2, ensure_ascii=False)
    (EVIDENCE / "results.json").write_text(
        f"{serialized_results}\n", encoding="utf-8"
    )
    print(serialized_results)
    return 1 if any(result["errors"] for result in results) else 0


if __name__ == "__main__":
    raise SystemExit(asyncio.run(main()))
