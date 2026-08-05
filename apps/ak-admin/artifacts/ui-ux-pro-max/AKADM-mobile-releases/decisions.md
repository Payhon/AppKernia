# Decisions

- Place the page under a new `system.mobile` directory as `system.mobile.releases`; mobile runtime governance is distinct from editorial content and general configuration.
- Use a client-side platform filter because the final OpenAPI declares no list query parameters and the endpoint returns a small global array.
- Treat platform values as protocol enums, not dictionaries, per the repository enumeration policy.
- Use one Drawer for both create and edit. Editing locks platform and includes the response `lock_version` in PATCH.
- Keep a 409 conflict visible in the Drawer instead of closing it; offer a list refresh so the operator can reopen current state.
- Use a responsive table with optional columns hidden at narrow widths; do not convert the page into a separate mobile UI architecture.
- Preserve the generic Master and save only this page override.
