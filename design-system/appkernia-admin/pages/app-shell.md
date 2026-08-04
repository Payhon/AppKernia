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
- Desktop sidebar collapse uses Ant Design's nested popup behavior. The control is not placed in the sidebar footer: hovering the desktop sidebar reveals a white left/right chevron handle at the vertical midpoint of the sidebar/content boundary. The control remains reachable by keyboard, is persistently visible on non-hover pointers, and reverses direction after collapse.
- At mobile widths, the same hierarchy is rendered in a drawer and the drawer closes after leaf navigation.
- Browser history and deep links remain intact.
- Header utilities are icon-first and ordered as fullscreen, language, account. Each 40 px icon button has a translated accessible name, stable hover treatment, and visible keyboard focus.
- Language opens a compact dropdown with the active locale selected. The login/auth surface retains the explicit text select because it is a standalone preference control rather than a dense application-header utility.
- Account is represented by a 36 px circular Avatar. Use the authenticated protected avatar Blob when available; otherwise render the uppercase first grapheme of display name/email. The click dropdown separates non-interactive user/role context from Personal Center and Sign Out actions.
- Fullscreen state follows the browser Fullscreen API and `fullscreenchange`, including Escape-driven exits. The icon and accessible label reverse with state; unsupported browsers expose a disabled control.

## Menu iconography and spacing

- Sidebar 使用 `#111111`，选中项反相为白底 ink 文字；hover 为 `#242424`，不得使用大面积品牌渐变。
- 选中叶子的 System 与功能目录祖先保持至少 `rgba(255,255,255,.92)` 的浅色文字和图标，不得继承叶子白底所用的 ink 文字色。
- 品牌区使用生成的 36px AppKernia 图标和 600 weight wordmark；折叠时仅保留图标并水平居中。
- Header 使用 sticky 64px、半透明白色、hairline 和背景 blur；内容滚动时仍保持上下文与操作可见。
- Header utility icons use neutral ink on transparent surfaces and `#F0F0F0` hover/focus fill. Account and language popups use white, 10 px radius, hairline border, and the shared stacked shadow; no marketing gradient is introduced.
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
