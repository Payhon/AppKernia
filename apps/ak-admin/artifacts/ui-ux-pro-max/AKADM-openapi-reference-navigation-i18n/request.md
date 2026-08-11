# Request

Date: 2026-08-11

Improve the independent OpenAPI reference without replacing Scalar or creating a second specification:

- Group hundreds of operations by interface surface and business module instead of rendering a flat navigation list.
- Localize the three surface labels, 31 module names, and every rendered operation title for `zh-CN` and `en-US`.
- Keep canonical English parameters, responses, schemas, examples, paths, methods, security, and operation identifiers unchanged.
- Ensure the same localized title appears in navigation, search results, and the operation body.
- Keep initial module lists collapsed and preserve public interactive testing, credential omission, self-hosting, CSP, reduced motion, keyboard navigation, and responsive 1440/375 verification.

Constraints: one canonical `server/openapi/openapi.yaml`, no locale-specific specs, no custom sidebar rewrite, no Admin main-bundle import, no backend/database/permission/client signature changes. The interactive-testing risk notice must be dismissible with a bilingual accessible close control, scoped to the current browser tab.
