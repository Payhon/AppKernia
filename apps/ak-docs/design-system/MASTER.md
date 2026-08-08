# AppKernia Docs Design System

> Source: `ui-ux-pro-max` persisted output for `AppKernia Docs`, 2026-08-08.

## Product direction

- Product: open-source framework website and developer documentation.
- Voice: precise, generous, credible, optimistic, community-first.
- Style: Minimalism / Swiss grid with a restrained developer-tool atmosphere.
- Homepage pattern: product story, architecture proof, real screenshots, adoption path, community CTA.

## Color

| Role         | Light     | Dark      |
| ------------ | --------- | --------- |
| Brand        | `#0284c7` | `#38bdf8` |
| Brand strong | `#0369a1` | `#7dd3fc` |
| Accent       | `#7c3aed` | `#a78bfa` |
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
- Body line-height: 1.7–1.8. Content width: 760–820 px.

## Layout and interaction

- Shared maximum width: 1200 px.
- Home sections use 12-column responsive grid and 72–112 px vertical rhythm.
- Breakpoints verified at 375, 768, 1024, and 1440 px.
- All links and buttons show visible hover and `:focus-visible` states without layout shift.
- Motion lasts 150–250 ms and is disabled by `prefers-reduced-motion`.
- Tables may scroll horizontally on small screens; the page itself must not.
- Rspress built-in search, locale navigation, sidebar, breadcrumbs, and heading hierarchy remain intact.

## Imagery

- Use authentic product screenshots for trust.
- Use restrained editorial 3D illustration only for the ecosystem concept.
- No emoji icons, guessed brand logos, fake metrics, customer logos, star counts, or testimonials.
- Every image has meaningful alt text; decorative layers are hidden from assistive technology.

## Anti-patterns

- Neon-heavy cyberpunk decoration, excessive glass blur, gradients behind long-form text.
- Hidden navigation or a marketing landing page that prevents access to docs.
- Claims that imply unverified Android/iOS/Harmony device acceptance.
- Copy that portrays planned capabilities as shipped.
