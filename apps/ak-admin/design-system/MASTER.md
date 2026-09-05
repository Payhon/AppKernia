# AppKernia Admin Design System

## Foundation

- Keep the existing Ant Design semantic tokens in `src/app/theme.ts` as the visual source of truth.
- Use a neutral white/gray data-dense surface with near-black primary actions and blue informational links.
- Use the existing system font stack; do not fetch web fonts at runtime.
- Standard spacing is 4/8/16/24/32 px. Controls are 40 px high, cards use 12 px radius, and focus states remain visible.
- A page-level Alert/Message must keep at least 16 px of vertical separation from the next card, filter, table, description or other content surface. Consecutive messages use the same minimum rhythm. Apply this through the shared `.ak-page-container` rule; do not add blanket margins to Alerts inside Modal/Drawer or double the spacing already owned by Ant `Space`.
- Support `zh-CN` and `en-US` at every breakpoint. Labels must wrap without obscuring controls.

## Interaction

- Every operation longer than 300 ms shows loading or progress feedback.
- Icon-only actions require an accessible name and tooltip; prefer text actions in data tables.
- Do not use layout-shifting hover transforms. Respect reduced-motion preferences.
- Do not encode status using color alone. Pair color with text or an icon.
- Keep resource-picker tables data-dense without hiding asset identity: use compact rows, a small fixed-ratio thumbnail, separate uploaded-time and size columns, and server-backed date-range filters.
- In resource-picker modal footers, place upload and its policy hint on the left as a secondary workflow, and keep cancel/confirm grouped on the right; allow the groups to stack on narrow screens without changing action order.
- Treat selectable resource rows as one interaction target across every cell while retaining the explicit radio control; pair pointer affordance with Enter/Space support, visible focus, `aria-selected`, and unchanged disabled scan-gate behavior.
- Use a pale informational surface for selected resource rows and preserve dark body text at WCAG AA contrast; the radio remains the non-color selection indicator. Pair the footer upload label with the existing Ant Design upload icon.
- Prefer the native Ant Design Splitter and Modal extension points for adjustable resource browsers. Keep explicit minimum pane/window sizes, visible drag affordances, keyboard-operable window controls, viewport clamping, a reversible maximize state, and a compact stacked fallback below desktop width.
- Resource pickers expose grid, compact-table and thumbnail views from one icon dropdown whose menu pairs icons with localized labels. Grid cards keep filenames visible and reveal metadata on hover/focus; compact tables use 16px identities and approximately 24px rows; thumbnail view remains the default visual browser. Preserve one selection and scan gate across all views.
- When resource preview is collapsed, reopen it with a small edge-mounted directional icon rather than a text button. Present maximize and close as one aligned window-control group, and render selected-file feedback on a neutral gray surface without an outlined alert treatment.
- Keep light-theme Select and selectable Dropdown/Menu options readable with a pale selected background and dark semantic text; never derive popup selection contrast from the near-black primary action color.
- Expose dialog resizing through the shared `AkModal` `resizable` capability. Its bottom-right hit area stays visually quiet until hover/focus/drag, then shows an inset rounded corner parallel to the dialog radius; dragging highlights the corner and keyboard arrows remain supported.
- Preserve keyboard navigation and a visible focus indicator for scrollable tables, category navigation, dialogs, and upload controls.
- For a single-value field that supports both presets and custom values, use a searchable creatable select. Include an explicit default option, preserve Enter-to-create keyboard behavior, and show text/value alongside any visual swatch.
- Group editable translations in one shared line-style locale Tabs region. Source locale order, default and labels from the locked `system.language` dictionary; do not stack one card per locale or maintain a page-local language list.
- Keep visited locale panels mounted so form values and dirty state survive switching. Mark invalid locale tabs with a semantic icon and accessible text, then focus the first invalid locale after submit.
- While the language dictionary is loading or invalid, render a contained loading/error state in the multilingual region and disable saving; do not silently fall back to a page-local array.
- Treat System as a data-level root but a shell utility: remove it from the scrolling primary menu and expose it through the fixed bottom gear beside OpenAPI documentation. Keep permission and feature-flag pruning before this visual partition.
- Bottom shell utilities are icon-only controls with tooltip, semantic accessible name, visible focus ring, and at least a 44 px touch target. Documentation stays left, System stays right; both disappear only when the entire sidebar is hidden.
- Open the desktop System hierarchy in a bounded, bordered and shadowed panel above the gear. Preserve the second/third levels through cascading keyboard-capable menus; use an inline scrollable hierarchy in the mobile Drawer. Escape and outside click close the panel and restore focus to its trigger.
- Open OpenAPI documentation in a separate browsing context with `noopener noreferrer`; do not move the current Admin route or imply that a browser tab is guaranteed to be a separate operating-system window.
- Structure the OpenAPI reference as interface surface, business module, then operation. Keep module operation lists collapsed initially; localize surface/module labels and operation titles from the documentation-only `api_reference` namespace while retaining canonical English descriptions, schemas, parameters and examples.

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

- Public H5 previews follow `pages/public-web.md`: shared AkModal, an independently sized toolbar and a proportionally scaled phone shell. Only published public content is embedded; preview does not emulate a device browser or inherit administrative data access.

- Push channel administration additionally follows `pages/push-channels.md` for write-only credentials, provider lifecycle labels, registered-device testing and acceptance terminology.

- Sign-in administration additionally follows `pages/login-providers.md` for fixed password state, conditional OTP channels, compiled provider forms, write-only secrets, preflight lifecycle, help guidance, and atomic App bindings.

- API Client administration additionally follows `pages/api-clients.md` for optional delegated-user binding, explicit effective-permission guidance, searchable active-user selection, and non-disclosure of client Secrets.

- Authentication surfaces additionally follow `pages/login-auth.md` for progressive interactive CAPTCHA disclosure, keyboard-equivalent controls, responsive coordinate mapping, focus recovery, and reduced-motion behavior.

- System configuration additionally follows `pages/system-configs.md` for the localized fixed CAPTCHA type enum and its permission-gated global edit boundary.

- The information content workbench additionally follows `pages/content-management.md` for its five-tab taxonomy, editor, moderation and report workflow.

- Verify 1440 px light/dark and 768 px layouts, both locales, loading/error/empty states, keyboard focus, and upload progress/cancel/resume.
- Keep screenshots and the UI Skill decision trail under `artifacts/ui-ux-pro-max/`.
- For navigation utility changes, additionally verify 1440 px expanded/collapsed, 375 px Drawer, both locales, current-System selected state, no-System permission state, keyboard focus return, reduced motion, and absence of horizontal overflow.
- For OpenAPI reference changes, additionally verify both locales at 1440 × 900 and 375 × 812, module and operation search, title consistency between navigation/search/body, stable anchors, canonical download byte identity, and isolation from the Admin entry graph.

## Help and feedback

Use [feedback](pages/feedback.md) for the App-scoped feedback list and private detail drawer. This page follows the existing fixed light Admin tokens under either OS color-scheme preference; adding a global dark-theme switch is outside this change.
