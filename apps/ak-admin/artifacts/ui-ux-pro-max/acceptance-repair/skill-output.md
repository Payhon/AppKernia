# ui-ux-pro-max output

Queries executed:

1. `SaaS admin legal content table empty translation fallback accessibility --design-system`
2. `React data table empty state localized status tag accessibility --stack react`
3. `forms tables accessibility empty state responsive --domain ux`

Relevant guidance:

- Preserve a flat, data-dense surface with high text contrast and visible focus states.
- Derive display fallbacks during render instead of introducing effect-driven duplicate state.
- Never leave an empty or crashed surface when content is missing; show a helpful localized fallback.
- Keep wide data tables inside their existing responsive scrolling container.
- Use text together with status color; status must never be an untranslated protocol token.

The generated font/color suggestions were not adopted because the repository Master remains the visual source of truth and forbids runtime web fonts.
