# Design decisions
- Keep the existing dictionary master/detail information architecture; add text badges for built-in, tenant override, extension policy, and availability instead of relying on color.
- Dictionary-backed config fields use a normal labeled Select with loading, retry, empty, and disabled-current-value states.
- Provider-specific configuration stays in the existing category form and appears only when its provider is selected; hidden secrets are never copied between providers.
- SMS bindings live in the notification-template editing flow because the approved provider template belongs to a logical template and locale.
- Test delivery uses a dedicated drawer with target and declared variables. SMS test delivery displays a billable-action warning and requires explicit confirmation.
- Mobile widths stack actions and details; tables retain a contained horizontal scroll instead of overflowing the page.
