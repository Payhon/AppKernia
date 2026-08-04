# AKADM-150 Design Decisions

- Inherit `design-system/appkernia-admin/MASTER.md`; retain the skill's data-dense, high-contrast and responsive table direction.
- Use the existing system font stack and approved primary/danger tokens. Do not load Fira/Google Fonts, use amber as the main CTA, or introduce marketing content.
- Persist user, platform, status, IP hint, page and sort in URL search params.
- Present server-provided email, IP and device hints only; the client never reconstructs hashes, token material, full user-agent or refresh-token state.
- A current session is identified in text as well as color. Its revoke confirmation has an extra warning and successful self-revoke clears authentication and returns to login.
- Every revoke is permission-gated in the UI and authorized again by the Backend; writes are never automatically replayed after a 401.
- Use Ant Design Modal focus management and live success/error feedback; keep the table scroll container keyboard reachable through real row actions.
