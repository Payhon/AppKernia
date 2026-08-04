# AKADM-240 UI Decisions

- Keep the established AppKernia Admin shell, semantic Ant Design tokens, system font stack and compact table rhythm.
- Use one static scheduled-jobs route with URL-backed filters and pagination; editing remains transient in an accessible Drawer.
- `handler_key` is a Select populated only from the server's compile-time registry. The UI never accepts a free-form executable handler, command, SQL or source code.
- Show a localized Cron explanation and the next five UTC instants formatted in the selected IANA time zone before save. DST effects remain visible as offsets rather than being hidden.
- Require confirmation for manual execution and status changes. Disable duplicate submits and announce success/error results.
- Run history opens in a detail Drawer, exposes safe status/error summaries only and never renders arbitrary output as executable content.
- At 375px, filters stack, primary actions remain visible and wide schedule/run tables use a focusable bounded scroll region.
