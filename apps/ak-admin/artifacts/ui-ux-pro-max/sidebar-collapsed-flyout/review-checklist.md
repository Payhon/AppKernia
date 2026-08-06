# Review checklist — Collapsed sidebar flyouts

Date: 2026-08-05

- [x] Root cause identified: expanded-state `openKeys` was still controlled in collapsed mode.
- [x] Expanded desktop navigation keeps current-route ancestors open.
- [x] Collapsed desktop navigation delegates submenu visibility to native hover/focus state.
- [x] Mobile Drawer remains click-driven and controlled.
- [x] Second- and third-level popups use the same connected translucent surface.
- [x] Parent-facing popup corners are square; outward corners remain rounded.
- [x] Collapse handle is fixed at viewport midpoint and flush with both sidebar widths.
- [x] Collapse button keeps translated `aria-label`, `aria-controls`, and `aria-expanded`.
- [x] Reduced-motion override remains in effect.
- [x] Browser interaction verified at expanded and collapsed widths.
- [x] Flyouts verified hidden after pointer leaves the active hierarchy.
- [x] Progressive second- then third-level hover verified.
- [x] `zh-CN` and `en-US` screenshots captured.
- [x] Axe, overflow, console error, and production build checks recorded.
