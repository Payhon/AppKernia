# Login Locale Icon Menu Request

## User request

- Replace the login page language select with a translation icon button.
- Open a language list popup when the icon is activated.
- Keep runtime `zh-CN` / `en-US` switching without a refresh.

## Constraints

- Reuse the shared `LocaleSwitcher` and existing Ant Design icon/menu behavior.
- Preserve anonymous local preference persistence, authenticated rollback behavior, visible focus, keyboard access, and responsive authentication layout.
- Keep all visible text in the bilingual blueprint catalogs.
