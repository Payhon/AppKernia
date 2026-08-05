# AKADM Profile Avatar Crop — ui-ux-pro-max Output

## Required design-system query

Query: `enterprise SaaS profile avatar image crop upload reusable component`

- Recommended style: Minimalism / Swiss, clean grid, high contrast, functional whitespace.
- Recommended effects: subtle 200–250 ms hover, clear hierarchy, fast loading.
- Required checks: visible focus, reduced motion, 375/768/1024/1440.
- The generated Enterprise Gateway marketing pattern, indigo/emerald palette and external Plus Jakarta Sans were rejected because this is an authenticated settings workflow governed by the existing AppKernia Master.

## UX query

Query: `avatar crop modal zoom rotate keyboard accessibility preview upload progress`

- All crop functions need keyboard alternatives and logical tab order.
- Show progress for the multi-stage flow.
- Keep visible focus indicators.
- Images require meaningful alt text.
- Errors need `role=alert`; state changes need an announced status.
- Icon-only actions require accessible names and cannot communicate state by color alone.

## React query

Query: `image crop canvas object URL reusable controlled component performance`

- Extract reusable stateful logic instead of duplicating it in the page.
- Keep components focused: crop dialog, upload controller and profile integration remain separate.
- Use controlled inputs and typed props; do not use `any`.
- Revoke object URLs and avoid persisting image binary in Zustand or browser storage.

## Web query

Query: `file input image preview dialog focus keyboard accessible progress`

- Use a semantic file input with an accessible name.
- Provide visible focus and button names for rotate/nudge controls.
- Inline validation belongs near the avatar control.
- Pointer dragging is optional enhancement; keyboard controls remain complete.
