# ui-ux-pro-max output

## Design-system query

- Query: `cross-platform developer platform mobile home profile article reading clean blue professional`
- Pattern result: App Store Style Landing. Only its screenshot-led hierarchy is relevant; store ratings, QR codes, and download CTAs are rejected because this is an in-app product surface.
- Style: Minimalism and Swiss Style; clean, spacious, functional, high contrast, grid-based.
- Suggested colors: primary `#1E40AF`, secondary `#3B82F6`, background `#EFF6FF`, text `#1E3A8A`.
- Suggested typography: Poppins/Open Sans. Runtime font downloads are rejected in favor of platform system fonts.
- Effects: subtle 200-250 ms transitions, restrained shadows, clear hierarchy.
- Avoid: clutter and slow loading.

## UX query

- Use correct mobile keyboard/input mode.
- Search empty state must give a recovery action or suggestion.
- Sticky navigation/action bars must not obscure content.
- Use a consistent type scale.
- Mutations need explicit success feedback.
- Destructive actions require confirmation.
- Async buttons disable repeated submission and expose loading.
- Build mobile-first and provide meaningful image alternatives.

## Vue stack query

- Centralize route guards instead of checking authentication in every page.
- Normalize and validate form input centrally rather than parsing ad hoc in each page.

The repository's uni-app x and AK UI constraints override generic Vue library suggestions.
