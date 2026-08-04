# AKADM-200 UI Decisions

- Reuse the established AppKernia navy/blue tokens and system font; do not import external Google Fonts.
- Configurations are grouped by module/group with URL-backed filters and a dense accessible table.
- Secrets expose only `configured/not configured`, key version, and updated time. Create/replace forms accept a new value once; list/detail never receive ciphertext or plaintext.
- Secret replacement uses a separate destructive-confirm modal and stable success/error live feedback. “Keep unchanged” is represented by omitting the secret field, never by sending a masked placeholder.
- Dictionaries use a two-pane master-detail layout on desktop and stacked selector/detail layout on narrow screens. No swipe-only interaction.
- System dictionary types and their items show a lock label and disable mutation controls. Tenant-owned types use exact action permissions.
- Locale labels are explicit; fallback from requested locale to `zh-CN`, then locale-neutral, is rendered as metadata rather than silently changing the stored locale.
- Loading, empty, retry, stale/refetch, keyboard focus and reduced-motion follow the existing Admin primitives.
