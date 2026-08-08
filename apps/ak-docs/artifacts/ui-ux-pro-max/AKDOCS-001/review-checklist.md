# Review checklist

- [x] No emoji used as UI icons.
- [x] Brand mark uses an original SVG; GitHub is a text-labelled external link.
- [x] Hover styles do not move layout.
- [x] Keyboard focus remains visible.
- [x] Light and dark theme content meets WCAG AA in the sampled axe runs.
- [x] Images include meaningful alternative text.
- [x] Home, docs tables, code blocks, and screenshots do not cause page-level horizontal scrolling.
- [x] `prefers-reduced-motion` disables decorative animation.
- [x] 375, 768, 1024, and 1440 px screenshots are captured after implementation.
- [x] Production 375 and 1440 px review has no orphaned single-character hero line.
- [x] Chinese and English routes both build and navigate.
- [x] Search, sidebar, locale switch, appearance switch, previous/next page, and edit link render in the production preview.
- [x] Product claims distinguish implemented, planned, and platform-unverified work.

Evidence: [screenshots/INDEX.md](screenshots/INDEX.md). Chromium checks reported HTTP 200, one H1, no broken images, no page-level overflow, no console errors, and no 4xx/5xx responses for every captured route. Eight axe WCAG 2/2.1 A/AA runs reported zero serious/critical findings.
