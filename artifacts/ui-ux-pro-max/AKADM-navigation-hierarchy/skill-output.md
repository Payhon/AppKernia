# ui-ux-pro-max Output

All commands below were executed from the repository root on 2026-08-04 and returned exit code `0`.

## Design-system query

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "enterprise admin dashboard hierarchical sidebar navigation three levels information architecture accessibility" --design-system -p "AppKernia Admin Navigation Hierarchy" -f markdown
```

The generated direction selected a data-dense enterprise dashboard, WCAG AA contrast, visible focus states, 150–300 ms interaction feedback, reduced-motion support, and responsive checks at 375/768/1024/1440 px. It also returned landing-page suggestions (hero, logo carousel, CTA, external fonts) that are outside the Admin-shell request and are explicitly rejected in `decisions.md`.

## UX query

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "sidebar hierarchical menu keyboard focus disclosure navigation" --domain ux -n 8
```

Relevant high/medium findings:

- Preserve visible keyboard focus indicators.
- Keep keyboard tab order aligned with visual order and avoid keyboard traps.
- Provide a skip-to-content link for navigation-heavy pages.
- Keep browser back navigation predictable.
- Visually identify the active page and section.
- Breadcrumbs are suitable for navigation with three or more levels.

## React query

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "nested navigation menu aria keyboard responsive" --stack react -n 8
```

Relevant findings:

- Prefer semantic navigation controls over click handlers on generic elements.
- Use React context/composition for shared state instead of deep prop drilling.
- Announce genuinely dynamic status content when required.

## Web query

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "nested sidebar navigation focus keyboard aria" --domain web -n 8
```

Relevant findings:

- All interactive controls require keyboard support.
- Never remove focus outlines without a visible replacement.
- Prefer semantic HTML and hide decorative icons from assistive technology.
