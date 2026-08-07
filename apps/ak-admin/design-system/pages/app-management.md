# App Management

This page family inherits `../MASTER.md`.

- Application selection is required for App users, App content, releases and notifications. Persist it as the typed `app_id` URL search parameter.
- Keep the application selector visible above list filters. If no application is selected, show an empty-state call to select or create an application; do not issue an unscoped request.
- Use a two-column detail drawer only at desktop widths. At 768px and below, use a full-screen drawer and stack bilingual editor fields.
- Keep article/category tables horizontally scrollable inside their own focusable container. Single-page editor uses locale tabs and a version-history table below the draft, not a free-form JSON editor.
- Disable reserved-page deletion controls and explain their protected status inline.
