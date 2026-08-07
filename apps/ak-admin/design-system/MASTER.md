# AppKernia Admin Design System

## Foundation

- Keep the existing Ant Design semantic tokens in `src/app/theme.ts` as the visual source of truth.
- Use a neutral white/gray data-dense surface with near-black primary actions and blue informational links.
- Use the existing system font stack; do not fetch web fonts at runtime.
- Standard spacing is 4/8/16/24/32 px. Controls are 40 px high, cards use 12 px radius, and focus states remain visible.
- Support `zh-CN` and `en-US` at every breakpoint. Labels must wrap without obscuring controls.

## Interaction

- Every operation longer than 300 ms shows loading or progress feedback.
- Icon-only actions require an accessible name and tooltip; prefer text actions in data tables.
- Do not use layout-shifting hover transforms. Respect reduced-motion preferences.
- Do not encode status using color alone. Pair color with text or an icon.
- Preserve keyboard navigation and a visible focus indicator for scrollable tables, category navigation, dialogs, and upload controls.
- For a single-value field that supports both presets and custom values, use a searchable creatable select. Include an explicit default option, preserve Enter-to-create keyboard behavior, and show text/value alongside any visual swatch.

## Responsive layout

- Desktop: category navigation and content use an approximately 280 px / flexible two-column layout.
- Tablet: category navigation becomes a horizontally scrollable selector or a compact select; content remains single-column.
- Mobile: stack filters, actions, descriptions, and upload feedback with no horizontal page overflow.
- Service status follows `pages/service-status.md`: 24/16 px section rhythm, equal-height summary cards, semantic module identity, and focusable contained tables.

## App management scope

- App-scoped administration always shows the selected application in the page header, with the selection reflected in the URL as `app_id`; never make a destructive or publishing action appear to target an unnamed application.
- Use a compact application context strip (name, public ID, enabled/disabled status and registration policy) before scoped user, content, release and notification tables. On narrow screens it becomes a labelled select above the page title.
- Treat pending verification, enabled and disabled membership states as separate textual status tags. Legal-page publishing must show the document type and immutable published version beside the action.

## Delivery checks

- Verify 1440 px light/dark and 768 px layouts, both locales, loading/error/empty states, keyboard focus, and upload progress/cancel/resume.
- Keep screenshots and the UI Skill decision trail under `artifacts/ui-ux-pro-max/`.
