# Skill output

The repository-mandated `ui-ux-pro-max` workflow was rerun before changing the
visible documentation UI.

Commands:

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py \
  "open source developer documentation real product screenshots admin mobile professional landing page" \
  --design-system --persist -p "AppKernia Docs AKDOCS-002" -f markdown

python3 .codex/skills/ui-ux-pro-max/scripts/search.py \
  "documentation content width line length three column centered wide screen accessibility" \
  --domain ux

python3 .codex/skills/ui-ux-pro-max/scripts/search.py \
  "feature cards single border hover focus no nested frame" \
  --domain style
```

Applied recommendations:

- prefer authentic product screenshots to abstract marketing artwork;
- keep long-form prose near 65–75 characters per line;
- use one visually clear card boundary with stable hover and focus states;
- preserve keyboard navigation, semantic headings, AA contrast, and reduced
  motion behavior;
- validate layout at the full set of target responsive widths.

Recommendations intentionally not adopted:

- ratings, download counters, customer logos, testimonials, and adoption
  metrics have no verified source;
- a download-first CTA is premature for a source-oriented project;
- heavy glass effects reduce long-form documentation legibility;
- generated illustration is unnecessary because current product evidence is
  available.
