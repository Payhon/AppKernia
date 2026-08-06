# Request — Collapsed sidebar flyouts

Date: 2026-08-05

## User request

- Fix collapsed desktop navigation so second- and third-level flyouts do not remain permanently visible.
- Reveal each flyout progressively only while its corresponding parent icon or menu item is hovered.
- Make flyout surfaces feel connected instead of using four rounded corners on every layer.
- Add controlled translucency so the page below remains perceptible.
- Place the collapse control at the visible viewport midpoint, flush with the sidebar boundary, with icon-only width, no horizontal padding, and a subtle shadow.

## Scope

- Admin authenticated App Shell only.
- Preserve expanded desktop navigation, mobile Drawer navigation, route selection, permissions, feature flags, localization, and Ant Design keyboard semantics.
- No backend, API, permission, dictionary, or database changes.
