# Push Channels Override

- Route: `/notifications/push-channels?app_id=...`; baseline: `../MASTER.md`.
- The selected App remains visible in the shell selector. Environment is an explicit filter and is repeated in the page heading summary.
- Desktop/tablet use provider summary cards followed by a compact provider table. At 375px, use stacked cards with full-width actions and no horizontal page overflow.
- Provider states are `unconfigured`, `draft`, `preflight failed`, `ready`, `active`, `disabled`, and `faulted`; pair every color with text and an icon.
- Public configuration uses a provider-specific labelled form. Secrets are write-only and are represented after save only by configured state, field names and fingerprint.
- Preflight, activation, disable, credential rotation and test notification are distinct actions. Testing selects a registered device; raw Token input is prohibited.
- Delivery results use the explicit labels provider accepted, failed, invalid Token and opened. Never label provider acceptance as device delivery.
- Drawers retain focus, validation uses `aria-live`, destructive/lifecycle confirmations name provider and environment, and long identifiers wrap safely.
- Place a tooltip-backed question-mark icon beside the environment selector in the page heading. It opens one bounded modal with nine provider tabs in the stable catalog order; use start-side tabs on desktop and horizontally scrollable top tabs below 768px.
- Every provider guide uses the same reading order: verified official-document notice, application steps, exact AppKernia field mapping, before-save warning, then official links. External links always open with `noopener noreferrer`; secrets remain write-only.
- Guide body text stays within 72 characters per line where practical, modal content scrolls independently, tab and dialog focus remain keyboard-operable, and 375px must not introduce page-level horizontal overflow.
