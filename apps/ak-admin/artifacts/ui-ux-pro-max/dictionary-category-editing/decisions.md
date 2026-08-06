# Design decisions
- Use the stable prefix before the first `.` in a dictionary code as a presentation category. This reproduces HotGo's two-level discoverability without adding fake parent dictionary records or changing the consumption API.
- Render category headers as collapsible controls with localized known namespace labels, namespace code, and item count. Unknown namespaces display their stable code and remain fully usable.
- Split display label and dictionary key/value into separate columns for scanning and copying.
- Show Edit for tenant rows and extensible built-in rows. A built-in edit creates a tenant override; key, locale, and capability metadata are immutable, so global seeds stay locked.
- If the current page already contains the matching tenant override, editing the built-in row opens that override instead of attempting a duplicate create.
- Keep filters and selected type in the URL, preserve a focusable horizontal table region, and stack master/detail cards below 1024 px.
