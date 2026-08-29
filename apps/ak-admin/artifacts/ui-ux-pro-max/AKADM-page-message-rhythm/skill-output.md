# ui-ux-pro-max output

Executed on 2026-08-29:

- Design-system query: `enterprise admin data tables page message alert spacing vertical rhythm consistency`.
- UX query: `alert banner table card spacing vertical rhythm responsive`.
- React query: `shared page layout alert table spacing CSS selectors component audit`.

Applied findings:

- Keep the current data-dense Admin visual language and use the established 4/8/16/24 spacing scale.
- Use one shared page-container rule so information banners have a predictable 16px minimum separation from the following data surface.
- Scope the rule to content pages. Modal/Drawer messages remain local, and Alerts already wrapped by Ant `Space` retain the spacing owned by that component.
- Verify the representative page at desktop and mobile widths with computed geometry, not screenshot inspection alone.
