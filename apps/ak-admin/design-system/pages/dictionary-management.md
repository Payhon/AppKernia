# Dictionary management page override

- Keep the desktop master/detail layout, with an approximately 320 px type navigator and a flexible item table.
- Build the type navigator as a two-level hierarchy: collapsible namespace categories first, selectable dictionary types second. Categories are presentation-only and come from the stable prefix before the first `.` in each dictionary code; they are never sent to dictionary consumers.
- Category headers show a localized category name, the namespace code, and the visible type count. Type selection remains a real button with `aria-pressed`, a visible focus state, and URL-persisted selection.
- Keep all matching categories expanded after filtering unless the user explicitly collapses them. On tablet and mobile, stack the navigator above the detail card.
- The item table gives the dictionary key/value its own column. Do not hide the key under the display label.
- Every writable row exposes a text Edit action. Editing a built-in row creates or updates a tenant override; the key, locale, and registered capability metadata remain immutable. Tenant rows use the normal update flow.
- The override drawer must explain the scope before submission, retain explicit field labels, show loading and success/error feedback, and never imply that the global seed was changed.
- Color and CSS class use single-value creatable selects: each starts with an explicit default choice, exposes bilingual named presets, remains searchable, and accepts a custom value with Enter. Color options show a swatch plus name and value; style options show a controlled preview plus name and class value.
- Custom CSS class values may be stored but must not be applied to the dictionary-management DOM. Only the management UI's compiled preset allowlist receives a live preview; custom values remain visible as code text.
- Keep status, built-in/override origin, and extension policy readable in text. Color may reinforce state but cannot be the only signal.
- On narrow screens, retain contained horizontal table scrolling and ensure the table scroll region is keyboard focusable.
