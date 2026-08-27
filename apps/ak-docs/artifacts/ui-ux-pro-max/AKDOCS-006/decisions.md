# AKDOCS-006 decisions

## Label

- Keep the existing bilingual frontmatter text and remove only its visual
  frame. Explicitly reset line-height because Rspress's stock Hero inherits the
  large headline line box into the label.

## Product composition

- Override the stock 1152 px flex layout with a specific 1488 px grid rule.
  The text column remains readable while the showcase receives 1.12fr and at
  least 45% of the desktop Hero width.
- Increase the browser canvas by reducing right padding and enlarge phones from
  22% to 23% without changing the screenshot files or their accessible text.

## Gradient

- Use layered radial blue/cyan glows over a quiet blue-to-green linear field.
- Render the background through a Hero pseudo-element sized to `100vw`, so the
  visual is full bleed while the content remains centered.
- Use separate light and dark gradients. No animation or blur is required.
