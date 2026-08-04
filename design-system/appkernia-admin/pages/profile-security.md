# Profile Security Override

- Route: `/profile/security`
- Task: `AKADM-060` password, device and session-management subtasks
- Keep the existing AppKernia Admin shell, tokens, typography and spacing from `MASTER.md`.
- Present active sessions as compact responsive cards with the current session visually and semantically identified.
- Place an accessible password-change card before the session list with current/new/confirm fields and explicit other-session revocation guidance.
- Place registered devices before active sessions. Device cards show latest browser, platform, IP, last activity, first registration and active-session count without rendering the device key.
- Identify the current device with a textual tag. Every device removal requires confirmation, and current-device removal must warn about immediate sign-out.
- Show browser/user-agent, IP, last activity and expiry without exposing refresh/access tokens.
- Every revoke action requires a confirmation dialog. The current-session dialog must state that success immediately signs the user out.
- Use loading, empty, success and recoverable error states with `aria-live` or `role=alert`.
- Preserve keyboard order and visible focus at 375, 768, 1024 and 1440 px.
- Do not suggest that MFA management is complete in this subtask.
