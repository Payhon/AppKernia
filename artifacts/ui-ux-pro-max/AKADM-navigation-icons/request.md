# AK Admin Navigation Icons Request

## User request

Add a configured icon to every visible Admin sidebar menu item. The current navigation only visibly distinguishes User Management with an icon. Reduce the excessive horizontal space between every icon and its label.

## Product and stack

- Enterprise B2B administration dashboard.
- React 19 + Ant Design 6 Menu.
- Three-level navigation: Dashboard, System, functional group, page.

## Constraints

- Render only icons from a compile-time Ant Design Icons allowlist.
- Use the backend `menu.icon` field as the configured source; never execute or dynamically import a server-provided value.
- Configure every core menu row with an icon and retain a safe fallback for custom rows without a valid icon.
- Keep a consistent 16 px icon box and an 8 px icon-to-label gap at all depths.
- Preserve permissions, feature flags, active states, keyboard behavior, collapsed sidebar popups, mobile Drawer behavior, and both supported locales.
