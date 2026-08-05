# Review checklist

- [x] Actual ui-ux-pro-max design-system, UX, and React searches executed.
- [x] Existing Admin Master retained without overwrite.
- [x] View/action permission mapping and runtime gates reviewed; authenticated E2E exercises read/create/update.
- [x] Loading, empty, error, stale, save, and validation implementations reviewed; 409 conflict is exercised at runtime.
- [x] URL platform filter restores after leaving the route and using browser Back. A hard refresh returns to login because the access token is intentionally memory-only.
- [x] Both release-note locales are required.
- [x] Core SemVer ordering, HTTPS URL, active URL, PATCH lock-version, and response mapping are covered by unit tests.
- [x] 1440 px zh-CN list and Drawer screenshots reviewed.
- [x] 375 px en-US card layout reviewed with no page overflow or clipped action.
- [x] axe serious/critical violations equal zero for all three required views.
- [x] Typecheck, lint, 62 tests, build, bundle budget, blueprint, i18n, and diff checks pass.
