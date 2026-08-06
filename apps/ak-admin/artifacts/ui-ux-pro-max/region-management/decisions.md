# Region management UI decisions

- Keep the existing lazy tree table and URL-backed filters.
- Add a permission-aware text action column instead of global CRUD buttons.
- Show “Add child” only on level 0/1 rows; code, parent, and derived level remain immutable in edit mode.
- Use one large labeled drawer for add/edit with RHF + Zod validation and explicit loading/error/success feedback.
- Use a destructive confirmation only for leaf deletion; parent deletion produces a clear “handle children first” message.
- Refresh the affected parent branch after mutations and invalidate all region search queries.
- Retain a focusable horizontal scroll container for narrow screens and use numeric input modes for numeric data.
