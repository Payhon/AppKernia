# App Shell — Navigation Override

This page override extends `design-system/appkernia-admin/MASTER.md` for the authenticated Admin application shell.

## Information architecture

- The sidebar has exactly two root nodes: `navigation.dashboard` and `navigation.system`.
- `navigation.dashboard` is a directly navigable page.
- `navigation.system` is a directory containing functional second-level directories.
- Business pages are third-level leaves. No business page other than Dashboard may render at the root.
- Functional directories with no accessible implemented leaves are removed instead of rendering empty disclosure controls.
- Backend ordering, permission filtering, feature flags, and route implementation checks remain authoritative.

## Core functional groups

- `navigation.system.settings`: system configuration, dictionaries, regions, and modules.
- `navigation.system.users`: departments, users, positions, and tenant management when enabled.
- `navigation.system.access`: roles, permission catalog, and menus.
- Storage, notification, integration, security/audit, and monitoring pages remain separate functional groups below System.

## Interaction

- Opening a directory must not navigate or reload the page.
- Navigating to a leaf selects that leaf and expands its System and functional-group ancestors.
- Desktop sidebar collapse uses Ant Design's nested popup behavior.
- At mobile widths, the same hierarchy is rendered in a drawer and the drawer closes after leaf navigation.
- Browser history and deep links remain intact.

## Menu iconography and spacing

- Every visible root, functional directory, and page leaf renders a configured icon.
- Core icon names come from the backend menu seed and resolve only through a compile-time Ant Design Icons allowlist.
- Unknown or empty custom-menu icon names use `AppstoreOutlined`; they never trigger dynamic imports.
- Icons use a consistent 16 × 16 px visual box and remain decorative because every item has a translated text label.
- Icon-to-label spacing is exactly 8 px. Hierarchy indentation remains controlled by Ant Design and must not be collapsed to compensate for spacing.
- Menu rows retain their existing target height and keyboard/focus behavior.

## Accessibility and localization

- Use Ant Design Menu semantic controls and keyboard behavior; do not replace them with click-only generic elements.
- Preserve visible `:focus-visible` treatment from the approved shell tokens.
- Decorative icons must not become the only accessible label.
- Every label is resolved through i18next; both `zh-CN` and `en-US` use identical keys and update without reload.
- The selected leaf and disclosed ancestors must communicate the current location visually.

## Responsive review sizes

- Desktop: 1440 × 900.
- Tablet: 768 × 1024.
- Mobile: 375 × 812.
