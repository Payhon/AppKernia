# Navigation Utilities Override

## Scope

This override governs the fixed OpenAPI and System utilities at the bottom of the Admin sidebar and mobile Drawer.

## Layout

- Keep the brand fixed, the primary menu independently scrollable, and the utility row fixed to the bottom.
- Expanded desktop uses two equal columns inside the 248 px sidebar. The 80 px collapsed sidebar assigns 40 px to each icon column. Mobile targets are at least 44 px high.
- Documentation is always left and System is always right. Hide System when permission/feature filtering removes every System leaf; retain Documentation.

## Interaction and accessibility

- Documentation uses a file-document icon, localized tooltip/ARIA label, and opens `/openapi/?lang=<locale>` with `_blank`, `noopener`, and `noreferrer`.
- System uses a gear icon and selected state on `/system/*`. Its desktop panel opens above the trigger and retains second-level groups with cascading third-level pages. Mobile uses an inline expanded hierarchy bounded by the Drawer.
- Outside click and Escape close the panel. Escape returns focus to the trigger. Menu selection closes the panel; mobile selection also closes the Drawer.
- Preserve Ant Menu arrow-key behavior, visible focus outlines, reduced-motion preferences, and no horizontal page overflow.

## Visual treatment

- Utility row: near-black sidebar surface, subtle top divider, 8 px button radius when expanded.
- System panel: 12 px radius, 1 px low-contrast border, layered shadow, maximum viewport-relative height, and internal scrolling.
- Active System button reverses to a white surface with dark ink. Hover/focus never uses layout-shifting transforms.

## Verification matrix

- Desktop 1440 px expanded and 80 px collapsed.
- Mobile Chromium viewport 375 × 812.
- `zh-CN` and `en-US` runtime language switching.
- System route selected state, no-System permission state, third-level navigation, focus return, reduced motion, axe serious/critical, and horizontal overflow.
