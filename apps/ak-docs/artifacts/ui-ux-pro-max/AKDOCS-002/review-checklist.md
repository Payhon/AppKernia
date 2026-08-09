# Review checklist

- [x] Admin identity assets are the only AppKernia marks referenced by docs.
- [x] The former documentation SVG marks and generated ecosystem Hero are gone.
- [x] Wide-screen shell is 1488 px and centered within 2 CSS px at 1920 px.
- [x] Prose stays within 72ch while tables and code may fill the content column.
- [x] Homepage uses one real Admin image and two real iOS simulator images.
- [x] Mobile/tablet Hero composition remains readable without page overflow.
- [x] Value cards use one border and visible hover/focus without geometry shift.
- [x] Homepage and five core entry pages are expanded in both locales.
- [x] All product evidence and platform boundaries are stated accurately.
- [x] Every sampled page has one H1, valid images and links, and no console error.
- [x] 375/768/1024/1440/1920 samples have no page-level horizontal overflow.
- [x] Sampled axe runs have no serious or critical findings.
- [x] Light/dark and `zh-CN`/`en-US` routes pass production-preview review.
- [x] Blueprint, i18n, docs checks, production build, and `git diff --check` pass.

The eight Chromium samples all returned HTTP 200 with one H1, valid images, no
page-level overflow, no console error, and zero axe serious/critical findings.
At 1920 px the documentation shell measured 1488 px with 216 px on both sides;
the sampled prose paragraph measured 725.625 px. Evidence is recorded in
[screenshots/INDEX.md](screenshots/INDEX.md).
