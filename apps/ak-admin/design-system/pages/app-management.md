# App Management

This page family inherits `../MASTER.md`.

- The application list is the tenant-scoped inventory. Show manifest AppID, fixed App type, name, description, remark, state, creation time and explicit destination actions.
- Search, type and status filters stay in one dense row on desktop and stack below 768px. The table owns its horizontal overflow; the page never does.
- Multi-select only enables rows that are both disabled and non-default. Single and batch deletion use the same destructive confirmation language and all-or-nothing server command.
- Application selection is required for App users, App content, releases and notifications. Persist it as the typed `app_id` URL search parameter and as a tenant-keyed, non-sensitive UUID preference so navigation and refresh restore the same context.
- Render the shared global application selector in the shell header immediately before fullscreen, and only on App-scoped routes. The tenant application inventory route does not show it.
- URL selection has precedence over memory and refreshes the remembered value after the current tenant's App list validates it. Clearing the selector removes both values; an unavailable remembered App is discarded rather than used as an unscoped fallback.
- If no application is selected, do not render page filters, a table, mobile data cards or their loading indicators. Use the shared minimum-height centered prompt inside the data card and do not issue an unscoped request.
- Use a two-column detail drawer only at desktop widths. At 768px and below, use a full-screen drawer; grouped localized content stays inside the shared horizontally navigable locale Tabs region.
- Keep article/category tables horizontally scrollable inside their own focusable container. Article, category and single-page editors share dictionary-driven locale tabs; the page editor keeps its version-history table below the draft and does not become a free-form JSON editor.
- Disable reserved-page deletion controls and explain their protected status inline.
- The application editor is one long, sectioned Drawer: basic/registration, icon and ordered screenshots, owner/team, channels, H5 and stores. Use two columns only when the Drawer has room; otherwise stack fields.
- Display internal file UUIDs as secondary technical references after selection. Do not expose provider bucket or object key.
- AppID is editable only while a migrated application is pending configuration. App type is immutable after creation.
