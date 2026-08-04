# AKADM-250 Screenshot Index

Final screenshots are stored under `output/playwright/` with the prefix `admin-integrations`.

- API Client: `en-US` empty/revoked at 1440px; `zh-CN` revoked at 1024/768/375px.
- Webhook: `en-US` empty/delivery at 1440px; `zh-CN` saved at 1440/1024/768/375px.
- Final 375px API Client review uses an explicit name/action compact table; no secondary column is clipped.
- `admin-integrations-e2e-results.json` records 11 axe checks with zero critical/serious violations, zero page overflow, no console errors, `ak-api` audience isolation, revoked-secret rejection, and a successful 204 local-mock webhook delivery.

No one-time client or webhook secret is present in any screenshot or persisted result.
