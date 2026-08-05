# Review Checklist

- [x] Form mode is the default on a clean URL.
- [x] Top switch exposes labelled Form and Table modes.
- [x] Table filters and definition editing remain available in table mode.
- [x] Direct form uses definition-driven typed controls and persistent labels.
- [x] One action saves only changed values.
- [x] Secret fields never show existing values and blank means unchanged.
- [x] Dirty changes are protected across category/mode changes and browser unload.
- [x] Validation and async outcomes are announced.
- [x] Permissions, locked definitions, and disabled definitions prevent edits.
- [x] zh-CN and en-US keys/placeholders match.
- [x] 375 and 1440 layouts have no page-level horizontal overflow.
- [x] Interactive controls and automated accessibility checks pass.
- [x] Static checks, unit tests, production build, and repository validators pass.
- [x] Runtime screenshots are indexed under `screenshots/`.

## Runtime evidence

- Docker Chromium against `http://127.0.0.1:4174`, using the real Go API and PostgreSQL.
- axe: zero violations for form/table in English at 1440, form in Chinese at 1440, and form in Chinese at 375.
- Unified save and exact restore both returned HTTP 200; PostgreSQL was rechecked with `site.name = AppKernia`.
- The existing file picker opened from the Logo definition and exposed the shared cloud upload entry.
- Firefox, Safari, production deployment, and native mobile devices were not tested.
