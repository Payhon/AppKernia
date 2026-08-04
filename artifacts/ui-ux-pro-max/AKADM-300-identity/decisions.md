# AKADM-300 Design Decisions

- Reuse the existing AppKernia light enterprise tokens, system font stack, Profile navigation, cards, alerts, drawers, and danger confirmations.
- Present enrollment as a bounded security task: one-time secret, copy action, numeric verification, then one-time recovery-code acknowledgement.
- Never persist TOTP secrets, step-up proofs, OAuth state/code, or recovery codes in Zustand, local/session storage, query persistence, logs, screenshots, or result JSON.
- Require explicit step-up proof for TOTP disable and recovery-code rotation. Destructive actions use a named dialog and do not auto-replay on 401.
- Connected accounts show only provider, provider account hint, status, and binding time. Unbinding explains impact and is disabled when the server rejects the last-login-method condition.
- OAuth callback has distinct processing, success, expired/invalid-state, feature-disabled, and retry/back states; it consumes URL parameters once and removes them from browser history.
- External fonts, marketing CTA patterns, badge animation, glow, dark gaming presentation, and hover movement are rejected for privacy, consistency, layout stability, and reduced-motion compliance.
