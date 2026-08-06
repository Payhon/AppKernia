# Request — Sidebar three-state controls

Date: 2026-08-05

## User request

- Keep the desktop sidebar collapsed after navigating from a collapsed flyout; route changes must never expand it.
- Narrow the midpoint collapse/expand handle and preserve a visible surface, border, and intentional icon/background color switch on hover.
- Add a fully hidden state, entered from a separate close icon above the midpoint handle while collapsed.
- In fully hidden state, keep a semi-transparent right-chevron restore tab at the far-left viewport edge; make it opaque on hover.
- In fully hidden state, also show a restore icon as the leftmost utility in the top-right Header. Either restore action must fully expand the sidebar and disappear afterward.

## Scope

- Authenticated Admin App Shell desktop behavior and styling.
- Preserve collapsed nested flyouts, mobile Drawer, route selection, permissions, feature flags, and current content.
- Persist only the non-sensitive sidebar display preference; no auth, API, backend, database, or permission changes.
