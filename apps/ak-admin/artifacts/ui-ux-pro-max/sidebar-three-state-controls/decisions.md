# Decisions — Sidebar three-state controls

Date: 2026-08-05

## State model

- Use one Zustand-backed `expanded | collapsed | hidden` state instead of independent booleans.
- Persist the mode to localStorage as a non-sensitive UI preference so TanStack route remounts and full reloads preserve explicit user choice.
- Only the midpoint toggle, collapsed close button, far-left restore tab, and Header restore button may change mode.
- Route navigation only closes the mobile Drawer; it never changes desktop sidebar mode.

## Controls

- Main boundary handle: 18 × 40 px, no horizontal padding, fixed viewport midpoint, left edge at 248 px expanded / 80 px collapsed.
- Hover/focus treatment: white/ink becomes ink/white, with border and stacked shadow retained.
- Collapsed close control: 18 × 28 px, 4 px above the main handle, SVG close icon, translated accessible name.
- Hidden edge restore: 18 × 40 px at x=0, 48% opacity at rest and 100% on hover/focus.
- Hidden Header restore: existing 40 px Header utility treatment, leftmost within the right action group.

## Responsive and accessibility

- Edge/close/desktop restore controls are hidden below 1024 px; mobile remains governed by the Drawer hamburger.
- Keep visible focus, translated labels, reduced-motion behavior, and Ant Design Button semantics.
- Use `MenuUnfoldOutlined`, `CloseOutlined`, and the existing chevron SVG; no emoji icon is rendered.
