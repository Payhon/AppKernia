# ui-ux-pro-max output

Date: 2026-08-11

The repository-local `ui-ux-pro-max` Skill was read and run before changing the visible reference. Searches covered:

1. Design system: API documentation, developer portal, structured navigation, dense technical content, and responsive documentation layouts.
2. UX: hierarchical disclosure, search, keyboard navigation, visible focus, 44 px mobile targets, reduced motion, and overflow prevention.
3. React: localized source transformation, controlled loading/error states, stable identifiers, and isolation of a large documentation dependency graph.

Applied findings:

- Use progressive disclosure: interface surfaces establish the broad API area, modules provide the expandable unit, and operations remain hidden until needed.
- Keep search continuously available and index the same localized title object used by navigation and body content.
- Preserve semantic identifiers and anchors while translating presentation-only labels.
- Verify keyboard focus, compact mobile navigation, reduced motion, and horizontal overflow rather than introducing new decorative interactions.
- Keep the documentation runtime independent so YAML parsing and large catalogs do not affect the Admin application entry.
- For persistent warning content, use a compact neutral close control with a visible focus ring and a short-lived session scope; do not permanently hide an important write-risk reminder.

Rejected or overridden by the AK master system and security constraints: vibrant block styling, marketing-page treatments, remote Noto Sans Hebrew typography, external assets, a custom replacement sidebar, and translation of canonical schemas or descriptions.
