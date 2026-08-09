# AppKernia Docs Design System

> Source: `ui-ux-pro-max` persisted output for `AppKernia Docs`, refreshed for
> `AKDOCS-003` on 2026-08-09.

## Product direction

- Product: open-source framework website and developer documentation.
- Voice: precise, generous, credible, optimistic, community-first.
- Style: Minimalism / Swiss grid with a restrained developer-tool atmosphere.
- Homepage pattern: product story, architecture proof, real screenshots, adoption path, community CTA.

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
- Home sections use 12-column responsive grid and 72–112 px vertical rhythm.
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
- No emoji icons, guessed brand logos, fake metrics, customer logos, star counts, or testimonials.
- Every image has meaningful alt text; decorative layers are hidden from assistive technology.
- Product galleries use repository acceptance screenshots only. Each caption
  identifies the verified runtime and avoids implying coverage for another
  platform, device, locale, or deployment environment.

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
- Each card has one 1 px border and one lightweight brand-color top marker.
- Do not add nested outlines, duplicated frames, tilt, shine, or moving hover
  geometry. Hover and keyboard focus may change color and shadow only.

## Anti-patterns

- Neon-heavy cyberpunk decoration, excessive glass blur, gradients behind long-form text.
- Hidden navigation or a marketing landing page that prevents access to docs.
- Claims that imply unverified Android/iOS/Harmony device acceptance.
- Copy that portrays planned capabilities as shipped.
- Generated ecosystem art in the product Hero when current real product
  evidence is available.
