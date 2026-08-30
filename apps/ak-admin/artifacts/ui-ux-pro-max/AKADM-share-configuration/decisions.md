# Decisions

1. Preserve the existing blue Admin Master and system type stack.
2. Use a dedicated configuration page because reusable multi-row Provider resources do not belong in scalar system settings.
3. Use table at `md+` and cards below `md`; keep Drawers full width on narrow screens.
4. Separate configuration lifecycle from App binding. A binding save always performs server preflight first.
5. Never request WeChat AppSecret. Mask the Android signature input and omit all platform identity details from audit payloads.
6. Show rebuild/export status at both configuration and binding entry points.
7. Treat server menus, permissions, registered routes, and implemented route keys as one navigation contract. The Share Configurations page must be allowlisted under its existing semantic System Settings hierarchy.
8. Reduce the Application Management operation column from 300px to 112px. Use Edit, Rocket, FileText, ShareAlt, Stop/CheckCircle and Delete icons for the corresponding menu entries; keep stop/delete danger styling and the existing delete confirmation.
9. Use the shared `.ak-page-container` instead of a page-local `Space` root. Keep 48px symmetric desktop padding and 16px symmetric mobile padding, and use the standard level-one Admin heading structure.
10. Keep the WeChat application guide separate from the editor: provider URLs live in `wechat-application-guide.ts`, user-facing instructions live in the `share_configs` bilingual catalogs, and the reusable modal component only maps those sources into UI.
11. Use five vertical steps rather than a large prose block. All eight external resources are HTTPS, use an external-link icon, and open with `target="_blank"` and `rel="noopener noreferrer"`; AppSecret remains explicitly excluded.
12. Apply contrast corrections only inside the guide modal. This preserves the global Admin theme while keeping waiting-step titles and links above the WCAG AA threshold.
