# Decisions

- Keep the current page structure and Ant Design tokens; this is a resilience repair, not a redesign.
- Resolve the title from the current locale, then the alternate supported locale, then a localized document-type/slug fallback.
- Keep translation data optional at the rendering boundary because newly seeded reserved drafts legitimately have no revision or translations.
- Align visible status keys with the stable API protocol values (`active`, `disabled`) in both catalogs.
- Add focused tests for empty translations and bilingual status output before browser verification.
