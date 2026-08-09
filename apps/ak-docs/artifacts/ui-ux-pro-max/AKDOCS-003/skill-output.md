# AKDOCS-003 ui-ux-pro-max output

Queries executed on 2026-08-09:

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py \
  "open source developer documentation architecture diagrams product story screenshot gallery professional technical" \
  --design-system -p "AppKernia Docs Content" -f markdown
python3 .codex/skills/ui-ux-pro-max/scripts/search.py \
  "software architecture flow diagram sequence diagram documentation" \
  --domain chart -n 10
python3 .codex/skills/ui-ux-pro-max/scripts/search.py \
  "technical documentation diagrams accessibility dark mode mobile responsive" \
  --domain ux -n 10
python3 .codex/skills/ui-ux-pro-max/scripts/search.py \
  "MDX diagram component screenshot gallery responsive" --stack react
```

Applied guidance:

- Use a story-led documentation entry with clear categories and a concrete next
  step for new contributors.
- Prefer high readability, 65–75 character prose, responsive images, meaningful
  alternative text, mobile-first containment, and keyboard-visible focus.
- Keep React components small and focused; use repository evidence instead of
  abstract marketing art for product proof.
- Use rendered flow/sequence diagrams for architecture and lifecycle questions,
  then retain prose summaries for accessibility and export paths.
- Respect reduced motion and avoid gratuitous diagram or gallery animation.

Not applied:

- Sankey charts were suggested for generic flows but are less suitable for
  layered software architecture, sequencing, and accessible text export.
- Motion-heavy storytelling, glass effects, ratings, testimonials, and download
  counters would weaken credibility at the current project stage.
