# Decisions — Collapsed sidebar flyouts

Date: 2026-08-05

## Interaction

- Preserve expanded sidebar `openKeys` in React state, but omit the controlled prop while the desktop Sider is collapsed. Ant Design can then open and close popup submenus from its native hover/focus state.
- Pass the collapsed state into the desktop navigation renderer instead of reading the global state inside the shared renderer. The mobile Drawer remains controlled even if the hidden desktop Sider is collapsed.
- Retain `triggerSubMenuAction="hover"` explicitly for desktop flyout intent; mobile inline mode still uses click disclosure.
- Tag every directory popup with a shared class so both second- and third-level surfaces receive the same connected treatment.

## Visual treatment

- Flyout background: `rgb(10 10 10 / 86%)` plus 14 px backdrop blur and 150% saturation.
- Flyout geometry: square parent-facing edge, 10 px outward radius, subtle light hairline, two-layer shadow.
- Collapse handle: fixed viewport midpoint, 22 × 40 px, no horizontal padding, flush at 248 px expanded / 80 px collapsed sidebar boundary.
- Keep the selected leaf opaque and high contrast through existing Ant Design tokens; do not use brand gradients.

## Accessibility and resilience

- Keep Ant Design Menu semantics instead of replacing it with mouse-only custom flyouts.
- Keep the collapse button focusable with its translated accessible name and current expanded state.
- Preserve the existing non-hover-pointer rule that leaves the collapse button visible.
- Disable transition under `prefers-reduced-motion` through the existing media rule.
