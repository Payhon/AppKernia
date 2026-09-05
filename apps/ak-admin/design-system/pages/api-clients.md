# API Client administration override

- Keep the existing dense inventory, edit Drawer, separate permission/App assignment Drawers, and one-time Secret disclosure flow.
- Treat the bound user as optional delegation, not ownership: label it explicitly and explain that effective access is the intersection of API Client permissions and the bound user's current permissions.
- Use the existing searchable active-user Select. Options show display name and email; the value remains the stable user UUID. Loading, failure, empty, disabled-current-value, and unbound states must remain distinguishable without color alone.
- Changing or clearing a bound user is part of the ordinary API Client edit transaction. Do not expose the user's password, sessions, roles, or other security details in this surface.
- Show the current binding in the inventory/detail views using localized text. Long names, email addresses, client IDs, and permission codes must wrap or remain inside a focusable scroll region at 375 px.
- Preserve explicit labels, Drawer focus management, save loading feedback, visible keyboard focus, and `zh-CN`/`en-US` key parity.
- Keep Drawer helper text at `#536179` or an equivalent WCAG AA contrast; Drawer portals are outside `.ak-app-layout`, so shell-scoped overrides do not apply.
- The configurable Admin base path changes navigation mechanics only; it must not alter the visual hierarchy, logical route names, menu paths, or API paths.
