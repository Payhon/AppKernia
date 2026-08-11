# ui-ux-pro-max output

Date: 2026-08-11

The repository-local `ui-ux-pro-max` Skill was read and run before UI implementation. The following scoped searches were used:

1. Design-system search: SaaS Admin navigation, fixed bottom utilities, OpenAPI documentation entry, and a System settings popover.
2. UX search: 44 px mobile touch targets, keyboard navigation, visible focus, focus restoration, outside/Escape dismissal, and reduced motion.
3. React implementation search: controlled popover state, focus management, responsive desktop/mobile menu modes, and isolation of heavy documentation code from the main application entry.

Applied output:

- Use a fixed utility row separated from the independently scrolling primary navigation.
- Keep icon order deterministic and provide both tooltip and accessible name.
- Use a bounded, bordered, rounded and shadowed desktop panel above the gear; use inline hierarchy in the mobile Drawer.
- Preserve the native Ant Menu keyboard model, close on Escape/outside/navigation, and restore trigger focus.
- Keep motion subtle and disable non-essential transitions under `prefers-reduced-motion`.
- Lazy/isolate the heavy API reference as a separate Vite page so Scalar is not part of the Admin initial graph.

Rejected or overridden as out of scope: marketing navigation patterns, gradients, remote typography, broad redesign of the existing dark sidebar, and flattening the existing System hierarchy.
