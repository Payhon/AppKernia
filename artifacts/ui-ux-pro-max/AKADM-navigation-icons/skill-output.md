# ui-ux-pro-max Output

All four commands were executed from the repository root on 2026-08-04 and returned exit code `0`.

## Design-system query

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "enterprise admin dashboard sidebar navigation consistent iconography compact spacing accessible React" --design-system -p "AppKernia Admin Navigation Icons" -f markdown
```

Relevant output: data-dense enterprise dashboard, consistent SVG iconography, WCAG AA contrast, visible focus, reduced-motion support, stable hover feedback, bundle-size awareness, and responsive checks at 375/768/1024/1440 px.

The command also returned landing-page Hero, CTA, logo-carousel, external Fira fonts, and a new dark/green palette. These are outside this App Shell refinement and are rejected in `decisions.md`.

## UX query

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "sidebar menu icons consistent size label spacing hierarchy" --domain ux -n 8
```

Relevant output: consistent typography improves scanning; preserve at least 8 px separation between adjacent touch targets; retain 44 px mobile touch targets; monitor bundle growth.

## React query

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "navigation icons semantic keyboard accessible compact" --stack react -n 8
```

Relevant output: retain semantic navigation elements and use accessible role/name queries in tests.

## Web query

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "icon label spacing sidebar focus contrast" --domain web -n 8
```

Relevant output: decorative icons must be hidden from assistive technology; visible focus must remain; semantic elements are preferred; icon-only controls require accessible names.
