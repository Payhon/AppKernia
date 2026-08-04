# Navigation Hierarchy Review Checklist

- [x] `ui-ux-pro-max` design-system, UX, React, and Web searches executed with exit code 0.
- [x] App Shell page override records the two-root, three-level information architecture.
- [x] Dashboard is the only root page; System is the only root directory.
- [x] System Settings contains System Configuration and Dictionaries (plus Regions and Modules).
- [x] User Management contains Departments, Users, and Positions.
- [x] Access Control contains Roles and Menus (plus the existing Permission Catalog).
- [x] All remaining implemented menu pages are below a functional group under System.
- [x] Permission, route implementation, and feature-flag filtering prune inaccessible leaves and empty groups.
- [x] Current leaf selection and active ancestor disclosure are derived from the router location.
- [x] `zh-CN` and `en-US` labels switch without reload (`performance.timeOrigin` unchanged).
- [x] Desktop 1440 px screenshots captured for both locales.
- [x] Mobile 375 px drawer screenshot captured with nested System navigation visible.
- [x] Chromium axe reports zero serious/critical violations after disclosure animations settle.
- [x] Dark-navigation leaf links meet contrast requirements.
- [x] Admin strict typecheck, lint, 42 tests, production build, bundle budget, and blueprint validator pass.
- [ ] Firefox and Safari visual/browser verification was not run in this task.
