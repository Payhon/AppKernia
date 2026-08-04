# AKADM-050 Design Decisions

- Reuse the existing secure split authentication shell so login, registration and recovery have one visual and keyboard model.
- Use the repository Master tokens and system font. Do not import the suggested Google Font, avoiding a third-party request on an authentication surface.
- Registration is feature-gated by backend public config at route and entry level. The agreement checkbox starts unchecked.
- Forgot-password always presents the same completion copy, independently of account existence. Cooldown comes from the stable backend response and is announced with a live region.
- Reset accepts the one-time token only from the URL, never persists it, and removes it from browser history immediately after exchange or successful completion.
- Password fields use explicit labels, `new-password` autocomplete and visibility toggles. Validation and service errors are text, not color-only.
- Keep motion limited to existing subtle focus/loading transitions and preserve the repository reduced-motion override.
- External email delivery is represented by a backend Port. Development evidence may use a local adapter, but the normal browser UI never displays a recovery secret.
