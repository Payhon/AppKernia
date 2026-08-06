# ui-ux-pro-max output — Sidebar three-state controls

Date: 2026-08-05

## Commands

```bash
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "enterprise admin dashboard three-state sidebar expanded collapsed hidden edge restore control compact accessible" --design-system -p "AppKernia Admin Sidebar Three State"
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "sidebar collapse hidden restore persistent state progressive disclosure keyboard focus hover controls" --domain ux -n 10
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "compact edge tab control opacity hover contrast border icon" --domain web -n 10
python3 .codex/skills/ui-ux-pro-max/scripts/search.py "persistent client UI preference routing remount state local storage" --stack react
```

## Relevant results

- Data-dense Admin navigation should maximize content visibility without sacrificing current-location indication.
- Keyboard focus, logical tab order, and visible focus rings are high-severity requirements.
- Hover states must provide visible feedback; primary actions cannot depend on hover alone.
- Icon-only buttons require accessible names, and decorative SVGs must remain hidden from assistive technology.
- State should have one source of truth; derived booleans should come from the stored mode instead of becoming independent state.
- Transitions should remain 150–300 ms and avoid scale/layout movement.

## Rejected recommendations

- Enterprise marketing gateway sections, Google-hosted Fira fonts, green CTA colors, and broad dark/OLED theming conflict with the approved AppKernia Admin system.
- Emoji close icons, scale animation, and an invisible hover-only hide action were rejected for consistency and accessibility.
