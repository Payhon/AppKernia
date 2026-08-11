# Review checklist

## Structure and state

- [x] Permission and Feature Flag filtering happens before the System partition.
- [x] Primary menu does not render the System root.
- [x] Documentation remains available when System is absent.
- [x] System routes, Seed structure, route registry, and backend authorization remain unchanged.
- [x] Expanded, collapsed, hidden, and mobile Drawer states retain their existing semantics.

## Accessibility and interaction

- [x] Icon buttons have localized tooltip and ARIA names.
- [x] Documentation uses `_blank` with `noopener noreferrer`.
- [x] System trigger exposes `aria-haspopup`, `aria-expanded`, `aria-controls`, and active-route state.
- [x] Escape closes and restores trigger focus; outside click and navigation close the panel.
- [x] Mobile targets are at least 44 px and reduced-motion rules are present.
- [x] Real Chromium keyboard/axe/overflow verification recorded for the stable shell and documentation surface.

## OpenAPI security and packaging

- [x] Canonical YAML is copied byte-for-byte at build time and served by Vite during development.
- [x] Scalar package is exactly `1.64.1` and isolated from the Admin entry graph.
- [x] Authentication persistence, default fonts, Agent, telemetry, developer tools, remote proxy, and plugin URLs are disabled.
- [x] Interactive requests force `credentials: omit` and canonical `Accept-Language`.
- [x] CSP and security/cache headers are defined for the Nginx production route.
- [x] Docker/Nginx smoke test and public Chromium screenshots recorded.

## Localization and evidence

- [x] All AppKernia-owned visible strings use semantic `zh-CN`/`en-US` keys.
- [x] Desktop/mobile navigation override and architecture documents are updated.
- [x] Screenshot index contains expanded, collapsed, mobile, System panel, and both OpenAPI locales.

## Verification boundaries

- Chromium ran at 1440 px desktop and 375 px mobile viewport sizes; the latter is not physical-device evidence.
- The stable OpenAPI surface and Admin shell have no axe serious/critical findings. Scalar's third-party transient API client layer still exposes ARIA and contrast findings while open; this state is recorded as a remaining risk rather than marked passed.
- The public health request, cookie omission, language header, reload boundary, and protected internal-route rejection were automated. A manually entered Bearer token against a protected write endpoint was not executed.
