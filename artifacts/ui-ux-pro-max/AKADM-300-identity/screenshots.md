# AKADM-300 Screenshot Evidence

Runtime: Docker production Admin image, Go API/Worker, PostgreSQL 18, Playwright Chromium.

- `output/playwright/admin-identity-security.security.en-US.1440.enabled.png`
- `output/playwright/admin-identity-security.connections.en-US.1440.empty.png`
- `output/playwright/admin-identity-security.callback.en-US.1440.success.png`
- `output/playwright/admin-identity-security.security.zh-CN.1024.disabled.png`
- `output/playwright/admin-identity-security.security.zh-CN.768.disabled.png`
- `output/playwright/admin-identity-security.security.zh-CN.375.disabled.png`
- `output/playwright/admin-identity-security.connections.zh-CN.375.empty.png`

The 375px security/connections scenes and 1440px callback were manually inspected after the final passing run. All seven scenes passed axe with zero violations and had no horizontal document overflow. TOTP secrets, recovery codes, step-up proofs, OAuth state/code, access tokens, and raw provider subjects are absent from screenshots and `output/playwright/admin-identity-security-e2e-results.json`.
