# Service status UI decisions

- Remove the duplicate module-management surface instead of moving or redirecting it.
- Retain the current service-status route and append semantic module information to its existing runtime summary.
- Introduce a single `.ak-ops-stack` for predictable 24/16 section rhythm and equal-height summary cards.
- Separate runtime facts and modules with a level-three heading and 16 px internal spacing.
- Display localized module name, stable code, localized description, localized capability tags, build version, and text status.
- Keep safe fallback behavior for unknown module or capability codes.
- Make both tables labeled focusable regions with contained horizontal scrolling.
