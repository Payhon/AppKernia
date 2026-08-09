# AKDOCS-004 — Homepage content, sliders, and technology stack

## Page intent

- Make the existing long-form homepage content visible instead of stopping
  after the Hero.
- Give visitors three kinds of evidence in sequence: explicit capabilities,
  recognizable implementation technologies, and authentic product surfaces.
- Keep the marketing layer honest: no fake metrics, customer logos, reviews,
  auto-playing media, or unsupported platform claims.

## Layout

- `HomeLayout` places the Hero and MDX content in one main landmark and retains
  Rspress extension points and footer behavior.
- Six feature links use a 3/2/1-column grid with a single border and a 3 px
  brand marker. Cards change color and shadow on hover/focus without moving.
- Nine technology cards use three columns on desktop, two on tablets, and one
  below 420 px so names and marks remain readable.
- The product tour gives the wider Admin slider approximately 1.48 shares and
  the Mobile slider 0.52 shares on desktop; they stack below 1100 px.

## Slider interaction

- No timer and no automatic advancement.
- Previous/next buttons, dot buttons, and Left/Right Arrow all change slides.
- Each slider is a named carousel region with a polite live region and visible
  counter. Icon-only controls have localized accessible names.
- Motion is limited to existing 180 ms state transitions and is removed by the
  global reduced-motion rule.

## Evidence and logo boundaries

- Admin slides are repository screenshots from the isolated local Docker/API
  environment. Mobile slides are iPhone 16 Pro / iOS 18.6 simulator evidence.
- The gallery does not claim production, iOS hardware, Android runtime, or
  HarmonyOS runtime acceptance.
- DCloud's official uni-app image is self-hosted; the remaining marks come from
  `simple-icons@16.27.0`. Technology marks identify dependencies and do not
  imply endorsement.
