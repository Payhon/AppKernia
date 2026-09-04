# System configurations override

This page inherits `../MASTER.md` and `system-settings.md`.

- Place `interactive_captcha.type` in the localized `iam / security` category. The protocol values are the fixed enum `click | slide | drag | rotate`; labels are translated without changing the submitted value.
- Treat CAPTCHA type as a stable security protocol option, not a runtime-extensible dictionary. Unknown values remain visible for diagnosis and must not be silently coerced.
- Reuse the generic enum field so plugins and future settings receive the same `settings.configOptions.<config_key>.<value>` localization convention.
- Preserve the server-provided permission and lock state. Global platform configuration remains read-only unless the backend marks it writable for the authorized platform administrator.
- Keep the current category-first, form/table, URL-addressable configuration workflow; do not introduce a CAPTCHA-specific settings page or bypass optimistic version checks.
- Validate both locales, all four labels, raw protocol-value submission, locked/forbidden states, narrow layout, keyboard selection, and save/conflict feedback.
