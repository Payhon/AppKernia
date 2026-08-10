# Decisions

- Retain the existing AK neutral palette, system font stack and Ant Design semantic tokens.
- Place one `GlobalAppSelector` in `AppShell`, immediately before `FullscreenToggle`.
- Show the selector only on `/app/users`, `/app/upgrade-center`, `/app/content/*` and the compatibility route `/system/mobile/releases`.
- Keep UUID `app_id` in URL search parameters; App list and other tenant/system pages do not show the selector.
- Persist only a `tenant UUID → App UUID` map under `ak.admin.app-selection.v1`; never persist App records, permissions, tokens or Query Cache.
- Resolve selection as explicit URL `app_id` first, then the active tenant's remembered App. A validated URL selection updates memory; a scoped route without a URL value writes remembered selection back to the URL.
- Clearing the selector removes both current URL selection and the active tenant's remembered value. Invalid/corrupt storage and remembered Apps missing from the current tenant list fail closed to no selection.
- Remove page-level context Cards and informational Alerts.
- Use one `AppSelectionRequiredState` inside each data Card. It has centered muted text, a responsive minimum height, `role=status`, and no spinner, filters or table.
- Hide create/publish controls until an App is selected. Disabled App status remains textual in both the dropdown label and desktop status tag.
- At mobile widths, preserve the selector in the header, compact its width and hide only secondary decorations. The accessible label remains available.
