# Review checklist — Sidebar three-state controls

Date: 2026-08-05

- [x] Existing dirty worktree audited and unrelated changes preserved.
- [x] `ui-ux-pro-max` design-system, UX, web, and React queries executed.
- [x] Master and App Shell override updated before implementation.
- [x] Collapsed state survives flyout leaf navigation and route component remount.
- [x] Main boundary handle is 18 px wide and retains visible hover/focus surface, border, and shadow.
- [x] Collapsed-only close control fully hides the sidebar.
- [x] Hidden edge restore is semi-transparent at rest and opaque on hover/focus.
- [x] Hidden Header restore is the leftmost right-side utility.
- [x] Either hidden restore action returns directly to fully expanded mode.
- [x] Desktop restore controls do not interfere with the mobile Drawer.
- [x] zh-CN/en-US screenshots, axe, overflow, console, build, and validators recorded.

Evidence: `output/playwright/sidebar-three-state-controls.evidence.json`; Chromium 1440×900 and 375×812; axe serious/critical 0; console errors 0; horizontal overflow false.
