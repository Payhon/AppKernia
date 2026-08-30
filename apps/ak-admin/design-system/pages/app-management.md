# App Management

This page family inherits `../MASTER.md`.

- The application list is the tenant-scoped inventory. Show manifest AppID, fixed App type, name, description, remark, state, creation time and explicit destination actions.
- Keep the desktop operation column compact: expose one text-and-icon Dropdown trigger, then pair every menu entry with a semantic Ant Design icon. Preserve permission pruning, disabled states and destructive confirmation inside the menu; reuse the same action model on mobile cards.
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
- The startup experience is a distinct section inside the application Drawer. Localized display name and subtitle use `AkLocalizedFormTabs`; changing Admin locale must not discard the inactive locale values.
- Treat the packaged icon and the remotely published onboarding revision as separate release surfaces. Show an authenticated icon thumbnail and a package-export reminder, but never imply that saving the Drawer changes an already installed binary.
- Draft save and onboarding publish are separate actions. Show the current immutable published version, published time and draft-changed state next to the publish action; a publish conflict keeps the draft values and offers a reload/retry path.
- Each onboarding position contains paired `zh-CN` and `en-US` scanned image assets plus localized non-visual accessibility descriptions. The pair moves and deletes as one unit.
- Limit the draft to ten positions. Provide labelled Move up / Move down controls with disabled boundary states; drag-only ordering is not permitted.
- Keep the onboarding draft when the enabled switch is off. Re-enabling an unchanged published version must not be presented as a new user-visible revision.
- At 768 px and below the startup fields, asset pair editors and action row stack to one column. File UUIDs remain secondary technical text and thumbnails never replace accessible labels.
- Replace the App-row “Share configuration” destination with one “Client configuration” Modal. Keep the existing action order and show the destination when at least one registered client-config Tab is readable.
- The client configuration Modal is 760 px on desktop and full-screen at narrow widths. Tabs remain in code-defined order: Sharing, then Scanner; switching Tabs never saves or discards local state.
- Each client configuration Tab owns its loading, validation and save actions. Never add a global Save button. Confirm before closing when any mounted Tab is dirty.
- Scanner host rows provide inline, screen-reader-announced validation, 44 px remove controls and no horizontal overflow. The enable switch retains host rows when disabled and the security explanation stays visible in read-only mode.
