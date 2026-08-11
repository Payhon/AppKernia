# Request

Date: 2026-08-11

Implement a self-hosted public OpenAPI reference and reorganize Admin shell navigation:

- Keep `server/openapi/openapi.yaml` as the only OpenAPI source.
- Add fixed Documentation and System icons at the bottom of the sidebar and mobile Drawer.
- Remove the System data root from the scrolling primary menu without changing its routes, permissions, feature flags, database rows, or third-level hierarchy.
- Open Documentation in a new browsing context and open System in a bordered, shadowed panel above the gear.
- Cover expanded/collapsed desktop, mobile Drawer, `zh-CN`/`en-US`, keyboard/focus, reduced motion, and accessibility.

Constraints: existing Ant Design visual language, no external fonts/CDN, no hard-coded visible copy, no database/menu Seed changes, no flattening of System.
