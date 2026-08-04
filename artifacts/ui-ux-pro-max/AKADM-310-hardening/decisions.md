# AKADM-310 Hardening Decisions

- Keep the established AppKernia light enterprise tokens and system font stack; reject the search result's video hero, marketing CTA, external font, scroll-snap, and animation patterns.
- Keep TanStack file-route lazy chunks and add a machine-enforced Vite manifest budget: initial static JS gzip at most 300 KiB and every JS chunk at most 180 KiB.
- Treat on-demand ECharts as a route-level chunk, while still enforcing the 180 KiB cap; do not exempt it from the implemented checker.
- Run axe and no-horizontal-overflow gates at 375/768/1024/1440, with both `zh-CN` and `en-US` represented in critical journeys.
- Stabilize full-page screenshots by auditing the keyboard skip link first, then hiding it only during screenshot capture to avoid Playwright's fixed-element stitching artifact.
- Expected 401/403/422 browser resource errors are counted exactly by their negative test; any other console error remains a hard failure.
- Persist only safe JSON evidence. One-time TOTP, recovery, OAuth, password, access/refresh, and CSRF values are excluded and scanned.
