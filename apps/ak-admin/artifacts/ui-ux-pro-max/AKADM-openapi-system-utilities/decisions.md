# Decisions

## Information architecture

- Partition only after backend menus pass Feature Flag, permission, implementation-registry, and empty-directory pruning.
- Treat `system` as a data-level root and a visual shell utility. Do not mutate `sys.menus`, Seed data, route registration, or API authorization.
- Hide the gear when no System leaf survives filtering. Documentation remains public and visible whenever the sidebar itself is visible.

## OpenAPI page

- Use a Vite MPA entry at `/openapi/` with Scalar `1.64.1`, built from the canonical YAML bytes and isolated from the Admin initial dependency graph.
- Self-host all bundled assets. Disable default fonts, Agent, telemetry, developer tools, remote proxy, remote plugins, and authentication persistence.
- Force all interactive requests through a custom fetch with `credentials: omit` and the current canonical `Accept-Language`. Do not inspect or copy Admin auth state.
- Show a bilingual warning above the reference because write operations affect the active environment.

## Interaction

- Desktop: controlled Popover above the gear; second-level groups in a vertical menu and third-level pages in cascading submenus.
- Mobile: controlled Popover inside the Drawer with inline expandable levels and bounded scrolling.
- Close on outside click, Escape, or page selection. Escape schedules focus restoration to the invoking trigger.
- Mark the gear selected only for `/system` and `/system/*`; retain the existing three-state sidebar behavior.

## Visual and responsive

- Two equal icon columns; 40 px allocation per icon in the 80 px collapsed rail, at least 44 px actual button/touch height.
- Reuse the existing near-black shell palette, focus treatment, and Ant semantic structure. No layout-shifting hover transform.
- Panel uses a 1 px subtle border, 12 px radius, layered shadow, viewport-relative maximum height, and internal scrolling.
