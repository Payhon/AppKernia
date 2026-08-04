# Authentication Recovery Page Override

Extends `../MASTER.md` for `/register`, `/forgot-password` and `/reset-password`.

- Layout: reuse the authentication split shell and single focused card; at narrow widths the brand block remains compact above the form.
- Hierarchy: one H1 brand landmark, one H2 task heading, concise explanatory text, form, primary action, then safe navigation back to login.
- Width: keep the existing authentication card measure so password guidance wraps without horizontal scrolling at 375 px.
- Forms: explicit labels/IDs; `email`, `new-password` and `new-password` autocomplete semantics; visible password toggles; agreement checkbox never prechecked.
- Feedback: errors use `role=alert`; neutral request completion and cooldown use `aria-live=polite`; do not reveal account existence.
- Security: never render the recovery token, email provider diagnostics or credential material. Remove token-bearing URL state as early as the flow permits.
- States: loading, feature-disabled, validation, generic service failure, neutral forgot completion, invalid/expired token and reset success.
- Accessibility: ≥4.5:1 body text contrast, visible focus, keyboard order follows the form, no color-only meaning and reduced-motion support.
- Localization: all copy is semantic i18n keys with complete `zh-CN` and `en-US` packs; long English copy must fit 375/768/1024/1440.
