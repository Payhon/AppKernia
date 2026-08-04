# Navigation Design Decisions

## Adopted

1. Keep exactly two root nodes: the Dashboard page and the System directory.
2. Preserve the backend menu tree and ordering instead of flattening page rows.
3. Use expandable second-level functional categories and prune categories that contain no accessible implemented page.
4. Keep the active page selected and automatically disclose every active ancestor.
5. Use Ant Design Menu's semantic, keyboard-accessible disclosure behavior and preserve visible focus styles.
6. Close the mobile drawer after a leaf navigation while keeping directory disclosure inside the drawer.
7. Resolve every label through semantic i18n keys, so a locale change updates the full tree without a reload.
8. Validate desktop and 375 px mobile layouts in both `zh-CN` and `en-US`.

## Rejected

- The generated hero, logo carousel, industry tabs, CTA, marketing sections, and sales-conversion guidance do not apply to an authenticated Admin application shell.
- The generated Fira/Google Fonts recommendation conflicts with the existing repository design system and the requirement to avoid an unnecessary external runtime dependency.
- The generated grey/orange palette conflicts with the approved AppKernia Admin token system; existing navigation tokens remain authoritative.
