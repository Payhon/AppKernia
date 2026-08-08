# Decisions

1. Keep the Rspress documentation shell and override only `HomeHero` and `Nav`: the hero needs one semantic H1, and locale/version list items need a valid list parent. Search, routing, i18n, sidebar, and content layout remain stock Rspress behavior.
2. Use `zh-CN` and `en-US` as the canonical locale paths to match the repository-wide i18n contract.
3. Present the homepage as evidence-led: architecture, actual Admin/Mobile screenshots, and runnable commands before broad claims.
4. Use an original no-text ecosystem illustration generated for this task. It must not imply a platform partnership.
5. Do not display invented adoption metrics, testimonials, download numbers, or Star counts. Use an ordinary accessible GitHub link so anonymous API limits cannot create console errors.
6. Document OpenAPI as the server contract source and label current project maturity honestly.
7. Use system font fallbacks when IBM Plex Sans or JetBrains Mono is unavailable to avoid a blocking external font request.
8. Override Rspress's light/dark code token colors and low-contrast footer labels, and remove hidden heading anchors from keyboard order. Eight axe runs must remain at zero serious/critical findings.
9. Balance the two hero heading lines at all breakpoints. Production review showed the final Chinese character orphaned at 1440 px; native `text-wrap: balance` preserves the wording and hierarchy without breakpoint-specific copy or fixed widths.
