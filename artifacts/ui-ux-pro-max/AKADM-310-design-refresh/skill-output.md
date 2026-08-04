# ui-ux-pro-max output

## Design system query

Command:

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "enterprise B2B developer platform admin dashboard RBAC system management data dense stark ink near-white calm professional accessible Vercel inspired React" --design-system -p "AppKernia Admin" -f markdown
```

Result:

- Pattern: Enterprise Gateway. Its marketing sections and Contact Sales CTA were rejected as out of scope for the authenticated Admin.
- Style: Data-Dense Dashboard; multiple charts/widgets, tables, compact grid, strong data visibility; excellent performance and WCAG AA orientation.
- Suggested colors: navy/blue with amber CTA. This was superseded by the user's `DESIGN.md` ink-primary direction and the generated blue/cyan/green brand identity.
- Suggested typography: Fira Code/Fira Sans via Google Fonts. External fonts were rejected by repository policy; local Geist/Inter/system fallbacks are used.
- Effects: tooltips, chart interaction, row hover, smooth filters and loading indicators.
- Anti-patterns: ornate design and missing filters.
- Checklist: no emoji icons, consistent icon sizing, 150–300ms transitions, 4.5:1 contrast, visible focus, reduced motion, 375/768/1024/1440 review.

## React stack query

Command:

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "table form drawer keyboard focus accessibility reduced motion responsive" --stack react
```

Result:

- High: manage focus for modal/dialog and return focus on close.
- High: associate visible labels with form controls; placeholders are not labels.
- Medium: use lazy state initialization for expensive initialization.

## Chart query

Command:

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "data dense admin dashboard charts operational monitoring" --domain chart -n 6
```

Result:

- Complex flow, hierarchy, network, 3D and geographic visualizations all require table/text alternatives because of weak accessibility.
- Dashboard implementation keeps the existing line chart and its expandable data table alternative; it does not introduce an unsuitable chart type.

## UX query

Command:

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "enterprise admin accessibility focus responsive tables forms" --domain ux -n 10
```

Result:

- Tables need controlled horizontal scrolling or responsive transformation.
- Interactive controls need visible focus rings.
- Avoid page-level horizontal overflow.
- Validate forms on blur, use semantic input types/autofill, mark required fields, and expose submit feedback.
- Verify common breakpoints including 375/768/1024/1440.
