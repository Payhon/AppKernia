# Decisions

- Keep `system.mobile.releases` as a compatibility route and add `app.upgrade-center`; both render the same page and call App-scoped endpoints.
- Persist every selector/filter/pagination value in the URL and keep the selected App visible.
- Treat platform values as protocol enums, not dictionaries, per the repository enumeration policy.
- Use one Drawer with native/WGT variants. Any record with `ever_published_at` is read-only; draft edits include `lock_version`.
- Keep a 409 conflict visible in the Drawer instead of closing it; offer a list refresh so the operator can reopen current state.
- Use a responsive table from 768px and compact cards below 768px; both expose the same detail action.
- Preserve the generic Master and save only this page override.
