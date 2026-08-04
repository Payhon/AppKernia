# Navigation Icon Decisions

## Adopted

1. Use the existing backend `icon` field as the configured source for every core menu row.
2. Resolve icon names through a compile-time `@ant-design/icons` allowlist. Unknown or empty values use `AppstoreOutlined`; server values never become import paths or executable code.
3. Configure a distinct, semantically related icon for all 35 core rows, including every third-level page.
4. Render icons in a consistent 16 × 16 px box and use an 8 px icon-to-label gap at every depth.
5. Keep icons decorative (`aria-hidden`) because the adjacent translated label remains the accessible name.
6. Preserve Ant Design Menu semantics, 44 px row targets, visible focus, active state, collapsed popups, and mobile Drawer behavior.
7. Re-run the bundle budget because the complete Ant icon allowlist adds JavaScript.

## Rejected

- Emoji and text-symbol icons are inconsistent and can be announced unexpectedly.
- Dynamically importing a backend-provided icon name violates the static registry boundary.
- A second design palette, external fonts, Hero, CTA, and marketing sections conflict with the approved Admin Master.
- Removing indentation to solve perceived spacing would erase hierarchy; only the icon-label gap is reduced.
