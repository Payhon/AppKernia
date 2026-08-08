# Acceptance repair request

- Date: 2026-08-08
- Scope: repair the App single-page content list when a reserved draft has no translation, and repair the visible application status translation.
- Constraints: preserve the existing Ant Design layout, App scope selector, bilingual catalogs, table accessibility, and content-management design-system override.
- Acceptance: no error boundary for untranslated drafts; a stable localized fallback is visible; `active` status renders localized in `zh-CN` and `en-US`; responsive table behavior remains unchanged.
