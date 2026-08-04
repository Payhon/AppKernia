# Navigation Icons Review Checklist

- [x] Project-local `ui-ux-pro-max` design-system, UX, React, and Web searches ran with exit code 0.
- [x] App Shell override defines the configured-icon and 8 px spacing rules.
- [x] All 35 core menu seed rows have a non-empty icon.
- [x] All 35 core icon names resolve through the compile-time Ant Design Icons allowlist.
- [x] Missing or unknown custom icon names render the safe `AppstoreOutlined` fallback.
- [x] Server icon values are never used as dynamic import paths or executable code.
- [x] Every visible `zh-CN` item in the current environment has one icon: 32/32.
- [x] Every visible `en-US` item in the current environment has one icon: 32/32.
- [x] Computed icon box width is consistently 16 px.
- [x] Computed icon-to-label gap is consistently 8 px.
- [x] Root, functional-directory, and leaf icons remain visually distinct at 1440 px.
- [x] Mobile 375 px Drawer retains the same icon set and hierarchy.
- [x] Icons are decorative; translated labels remain the accessible names.
- [x] Chromium axe reports zero serious/critical violations.
- [x] Node 24 Admin check passes: lint, strict typecheck, 10 files/45 tests, production build, bundle budget, blueprint validator.
- [x] Backend check and all blueprint/i18n validators pass.
- [ ] Firefox and Safari visual verification were not run.
