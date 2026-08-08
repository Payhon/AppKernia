# Admin acceptance repair screenshots

Runtime: local Chromium at 1440 px, final Admin source, local Go API and isolated PostgreSQL 18.

- `applications.zh-CN.1440.png`: application status renders as `启用`.
- `applications.en-US.1440.png`: application status renders as `Active`.
- `pages-empty-translations.zh-CN.1440.png`: reserved draft with no revision uses the Chinese page-type fallback.
- `pages-empty-translations.en-US.1440.png`: the same draft uses the English page-type fallback.
- `results.json`: both locales and both routes have zero axe serious/critical violations, no console errors, and successful API responses.

These screenshots are local browser evidence, not production deployment evidence.
