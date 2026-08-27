# AppKernia Docs Design System

> Source: `ui-ux-pro-max` persisted output for `AppKernia Docs`, refreshed for
> `AKDOCS-006` on 2026-08-27.

## Product direction

- Product: open-source framework website and developer documentation.
- Voice: precise, generous, credible, optimistic, community-first.
- Style: Minimalism / Swiss grid with a restrained developer-tool atmosphere.
- Homepage pattern: author-led product story, connected capability grid, product screenshots, adoption path, community CTA.
- Public copy speaks from the project author to developers. Build logs, acceptance
  boundaries, screenshot provenance, and delivery-report language belong in
  internal evidence artifacts rather than the landing-page narrative.

## Color

| Role         | Light     | Dark      |
| ------------ | --------- | --------- |
| Brand        | `#1677ff` | `#69b1ff` |
| Brand strong | `#0958d9` | `#91caff` |
| Accent cyan  | `#13c2c2` | `#5cdbd3` |
| Accent green | `#52c41a` | `#95de64` |
| Page         | `#f8fbff` | `#07111f` |
| Surface      | `#ffffff` | `#0d1b2d` |
| Text         | `#0f172a` | `#e6f1ff` |
| Muted        | `#475569` | `#9fb1c6` |
| Border       | `#dbe7f3` | `#20334b` |

正文和交互控件至少满足 WCAG AA。状态不能只通过颜色表达。

## Typography

- Heading/code accent: JetBrains Mono, with system monospace fallback.
- Body/UI: IBM Plex Sans, with system sans-serif fallback.
- Avoid blocking rendering on third-party font CDNs; use local/system fallback first.
- Body line-height: 1.7–1.8. Prose line length: 65–75 characters (`72ch`
  default); code blocks and tables may use the full content column.

## Layout and interaction

- Documentation desktop shell maximum width: 1488 px, centered as
  `280 px sidebar + 960 px content + 248 px outline`.
- The content column uses 56 px horizontal padding on wide screens. From
  1280–1535 px it becomes fluid with 40 px padding; below 1280 px the outline
  collapses, and below 768 px the sidebar uses the Rspress drawer.
- At 1920 px, the left and right outer margins of the three-column shell must
  differ by no more than 2 CSS px.
- Home sections use a 1240 px, 12-column responsive grid and 80–112 px vertical
  rhythm. Eyebrows occupy columns 1–3 while the heading and lead begin at
  column 4 on desktop, then collapse to one reading column below 768 px.
- Homepage sections form one continuous bordered rail: neutral white/black
  surfaces, 1 px dividers, square outer edges, minimal shadow, and one blue
  accent. This follows the restrained grid logic of developer platforms such
  as Vercel without copying their assets or product identity.
- The homepage Hero uses a full-bleed, low-saturation blue–cyan–green gradient
  behind the centered content grid. Text remains on an optically quiet portion
  of the gradient, while the product showcase occupies at least 45% of the
  desktop Hero width.
- The custom home layout must render the authored MDX body after the Hero;
  frontmatter-only output is not an acceptable homepage implementation.
- Breakpoints are verified at 375, 768, 1024, 1440, and 1920 px.
- All links and buttons show visible hover and `:focus-visible` states without layout shift.
- Motion lasts 150–250 ms and is disabled by `prefers-reduced-motion`.
- Tables may scroll horizontally on small screens; the page itself must not.
- Rspress built-in search, locale navigation, sidebar, breadcrumbs, and heading hierarchy remain intact.

## Imagery

- Use the Admin assets in `apps/ak-admin/public/brand` as the only source of
  AppKernia identity marks for the documentation site.
- Use freshly captured, authentic product screenshots for the homepage Hero:
  one loaded Admin dashboard and two signed-in iOS simulator surfaces.
- Desktop uses an Admin browser window with two overlapping phones; tablet
  stacks the browser and phones; mobile shows the Admin window and one phone.
- The Hero eyebrow is compact plain text without a pill border or padded frame.
- No emoji icons, guessed brand logos, fake metrics, customer logos, star counts, or testimonials.
- Every image has meaningful alt text; decorative layers are hidden from assistive technology.
- Product galleries use repository screenshots only. Public captions explain
  what a developer can do; runtime provenance and test boundaries remain in
  the corresponding internal screenshot index.
- Homepage product tours use two manual sliders: one Admin/Web and one Mobile.
  They never auto-advance, support buttons, dots, and keyboard arrow keys, and
  announce slide changes without adding visible duplicate captions.
- Technology marks use official or version-locked SVG sources, are self-hosted
  in the production output, and do not imply vendor endorsement.

## Technical diagrams

- Render flowcharts and sequence diagrams with the Rspress-compatible Mermaid
  plugin; never publish Mermaid source as the primary reading experience.
- Every diagram supplies an accessible title and description plus a concise
  prose summary that remains useful when SVG rendering is unavailable.
- Architecture diagrams use a left-to-right flow on desktop, readable labels,
  no decorative animation, and horizontal containment on narrow screens.
- Use one visual question per diagram. Separate the overall system, Admin,
  Server, and Mobile diagrams instead of forcing every layer into one canvas.
- Diagrams must react to Rspress light/dark mode, keep text contrast at WCAG AA,
  and preserve keyboard and page-level horizontal-scroll behavior.

## Cards

- Homepage value cards are semantic links, not the Rspress `features` widget.
- Card groups share one outer border and 1 px internal dividers instead of
  floating rounded rectangles. Cards use flat surfaces, square edges, no
  decorative shadow, and a quiet background change on hover/focus.
- Core-feature cards use a six-card grid at desktop, two columns at tablet, and
  one column on narrow screens. Technology cards use the same connected
  three-column grid and preserve each mark's recognizable brand color.
- Do not add nested outlines, duplicated frames, tilt, shine, or moving hover
  geometry. Hover and keyboard focus may change color and shadow only.

## Anti-patterns

- Neon-heavy cyberpunk decoration, excessive glass blur, gradients behind long-form text.
- Hidden navigation or a marketing landing page that prevents access to docs.
- Claims that imply unverified Android/iOS/Harmony device acceptance.
- Copy that portrays planned capabilities as shipped.
- Generated ecosystem art in the product Hero when current real product
  evidence is available.
