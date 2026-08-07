# Decisions

- Reuse `brand.primary #2563EB`, card and spacing tokens from the Mobile Master.
- Use only `ak-*` controls for business interaction; native text views only render safe static/dynamic text.
- Registration is hidden unless server public config enables it. Its server response never establishes a session.
- Both privacy policy and terms are available before login, including bundled snapshots for offline first-launch consent.
- CMS content passes the existing text-only `ArticleBodyRenderer`; no HTML renderer is introduced.
