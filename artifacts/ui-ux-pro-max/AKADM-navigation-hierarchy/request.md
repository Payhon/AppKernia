# AK Admin Navigation Hierarchy Request

## User request

Keep `Dashboard` as the only standalone page at the root of the Admin sidebar. Move every other Admin page below the `System` root directory and group pages into functional second-level directories.

Required core structure:

```text
Dashboard
System
├── System Settings
│   ├── System Config
│   └── Dictionary
├── User Management
│   ├── Departments
│   ├── Users
│   └── Positions
└── Permission Settings
    ├── Roles
    └── Menus
```

Other implemented pages must remain under `System` and be grouped by their functional domain.

## Constraints

- React + Ant Design Admin shell.
- Preserve backend permission and feature-flag filtering.
- Preserve the OpenAPI menu contract and backend-provided ordering.
- All visible labels must use existing semantic i18n keys for `zh-CN` and `en-US`.
- Locale switching must update the menu without reload.
- Desktop, collapsed sidebar, mobile drawer, keyboard access, active-page state, and active ancestor disclosure must remain usable.
