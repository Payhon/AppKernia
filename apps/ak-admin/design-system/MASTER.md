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
- Group editable translations in one shared line-style locale Tabs region. Source locale order, default and labels from the locked `system.language` dictionary; do not stack one card per locale or maintain a page-local language list.
- Keep visited locale panels mounted so form values and dirty state survive switching. Mark invalid locale tabs with a semantic icon and accessible text, then focus the first invalid locale after submit.
- While the language dictionary is loading or invalid, render a contained loading/error state in the multilingual region and disable saving; do not silently fall back to a page-local array.

## Responsive layout

- Desktop: category navigation and content use an approximately 280 px / flexible two-column layout.
- Tablet: category navigation becomes a horizontally scrollable selector or a compact select; content remains single-column.
- Mobile: stack filters, actions, descriptions, and upload feedback with no horizontal page overflow.
- Service status follows `pages/service-status.md`: 24/16 px section rhythm, equal-height summary cards, semantic module identity, and focusable contained tables.

## App management scope

- App-scoped administration uses one global application selector in the shell header, immediately before the fullscreen control. It is visible only on routes that require an App and keeps the selection in the URL as `app_id`; never make a destructive or publishing action appear to target an unnamed application.
- Remember only the selected App UUID in non-sensitive browser workspace state, keyed by active tenant. An explicit valid URL `app_id` takes precedence and updates the memory; a scoped route without `app_id` restores the tenant's latest App and writes it back to the URL. Clearing the selector clears both URL and memory.
- Do not repeat an application context strip inside page content. Until an App is selected, replace the scoped table area with the shared neutral prompt: a minimum-height container with centered muted text and no loading spinner, filter controls or informational Alert.
- On narrow screens, keep the selector in the shell header, hide its secondary label/status decoration when space is constrained, and retain its accessible name and searchable control.
- Treat pending verification, enabled and disabled membership states as separate textual status tags. Legal-page publishing must show the document type and immutable published version beside the action.

## Delivery checks

- Verify 1440 px light/dark and 768 px layouts, both locales, loading/error/empty states, keyboard focus, and upload progress/cancel/resume.
- Keep screenshots and the UI Skill decision trail under `artifacts/ui-ux-pro-max/`.
