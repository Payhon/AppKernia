# AKDOCS-006 — Hero gradient and product focus

## Hero override

- Render the `MIT open source · Mobile + Admin + Server` eyebrow as compact
  mono text with no border, pill radius, or padded shell.
- Override Rspress's stock 1152 px flex Hero with the AppKernia 1488 px grid.
  At desktop widths, the product screenshot column must occupy at least 45% of
  the Hero and use the full available width.
- Use a full-bleed blue–cyan–green gradient spanning the Hero from viewport edge
  to viewport edge. Keep saturation restrained, preserve WCAG AA text contrast,
  and provide a darker equivalent rather than reusing the light gradient.
- Below 1100 px, stack copy and product surfaces; below 768 px, keep the Admin
  window and one phone without page-level horizontal overflow.

## Preserved AKDOCS-005 rules

## Page intent

- Speak as the author to developers: explain the problem, the product benefit,
  the path to start, and the invitation to contribute.
- Remove `HONEST MATURITY` and all delivery-report language from public copy.
  Version and adoption guidance may remain when it helps a developer make a
  decision, but it must be written as guidance rather than acceptance evidence.
- Preserve the existing feature, technology, product-tour, quick-start, FAQ,
  and community content while tightening hierarchy and repetition.

## Layout

- The homepage content rail is 1240 px wide with continuous left/right rules
  and one border between sections.
- Desktop sections use 12 columns: the small label sits in columns 1–3 and the
  headline/lead begin at column 4. Below 768 px the section becomes one column.
- Value, belief, surface, feature, technology, step, FAQ, and slider groups use
  one outer border with 1 px internal grid lines. They have no floating shadow,
  gradient top marker, or independent rounded shell.
- Product screenshot devices may retain modest depth because the shadow
  belongs to the mock device, not the page layout. Admin and Mobile sliders
  stack below 1100 px.

## Slider interaction

- No timer and no automatic advancement.
- Previous/next buttons, dot buttons, and Left/Right Arrow all change slides.
- Each slider is a named carousel region with a polite live region and visible
  counter. Icon-only controls have localized accessible names.
- Motion is limited to existing 180 ms state transitions and is removed by the
  global reduced-motion rule.

## Copy and asset boundaries

- Public captions describe product capability and workflow; they do not read
  like a test report. Runtime provenance remains in the AKDOCS evidence folder.
- DCloud's official uni-app image is self-hosted; the remaining marks come from
  `simple-icons@16.27.0`. Technology marks identify dependencies and do not
  imply endorsement.

## AKDOCS-007 installation quick start

- Replace the source-only three-card quick start with one five-option install
  tab set followed by the shared administrator bootstrap and serve commands.
- Keep source build selected while no public release exists. Shell, npm,
  Homebrew, and manual Release tabs must state their actual availability rather
  than presenting future channels as live.
- Tabs use real buttons, one visible panel, tab/tabpanel relationships, visible
  focus, Arrow/Home/End navigation, and 44 px minimum targets on narrow screens.
- Keep the existing section rail, typography, colors, and square connected-grid
  treatment; this is a content hierarchy change, not a new visual language.
