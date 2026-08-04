# Review checklist

## Visual quality

- [x] Generated brand mark inspected at master and compact sizes.
- [x] Final brand PNG files have alpha channels.
- [x] App Shell, authentication and browser favicon use generated assets.
- [x] Primary action, link, status and gradient roles are separated.
- [x] Cards use hairline plus stacked small shadows; no layout-shifting hover transform.
- [x] Dashboard chart palette/grid/tooltip align with the updated system.

## Interaction and accessibility

- [x] Existing semantic Ant Design controls and visible labels preserved.
- [x] Visible `:focus-visible` treatment retained and strengthened.
- [x] `prefers-reduced-motion` global rule retained; ECharts animation remains disabled for reduced motion.
- [x] Login 1440/768/375: no horizontal overflow and axe 0 violations.
- [x] Dashboard 1440: no horizontal overflow and axe 0 serious/critical violations.
- [x] Dashboard 768 Drawer screenshot captured after transition completion.
- [x] Chart table alternative retained.

## Responsive screenshots

- [x] `screenshots/1440x900-light.png`
- [x] `screenshots/768x1024.png`
- [x] `screenshots/375x812.png`
- [x] `screenshots/dashboard-1440x900-light.png`
- [x] `screenshots/dashboard-768x1024-navigation.png`
- [ ] Dark theme screenshot: not applicable; product does not currently implement a dark theme and none is claimed.

## Engineering gates

- [x] ESLint.
- [x] TypeScript strict check.
- [x] Vitest: 10 files / 45 tests.
- [x] Vite production build.
- [x] Bundle budget.
- [x] Admin blueprint validator.
- [x] Mobile blueprint validator.
- [x] Cross-surface i18n validator.
- [x] UI Skill availability check.
- [x] `git diff --check`.

## Verification boundary

- Chromium only for this turn's new screenshots.
- Login screenshots use the current local Docker Admin/API public surface.
- Dashboard visuals use `visual_check.py` deterministic API-boundary fixtures and do not prove PostgreSQL or production deployment.
- Firefox, Safari, production deployment, Android/iOS/Harmony builds and physical devices were not run.
