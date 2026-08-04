# AKADM-260 Screenshot Evidence

Runtime: Docker production Admin image, Go API/Worker, PostgreSQL 18, Playwright Chromium.

## Access Rules

- `output/playwright/admin-operations.block-rules.en-US.1440.empty.png`
- `output/playwright/admin-operations.block-rules.en-US.1440.saved.png`
- `output/playwright/admin-operations.block-rules.zh-CN.1024.saved.png`
- `output/playwright/admin-operations.block-rules.zh-CN.768.saved.png`
- `output/playwright/admin-operations.block-rules.zh-CN.375.saved.png`

## Service Status

- `output/playwright/admin-operations.health.en-US.1440.png`
- `output/playwright/admin-operations.health.zh-CN.768.png`
- `output/playwright/admin-operations.health.zh-CN.375.png`

The 375 px Access Rules and Service Status screenshots were manually inspected after the final passing run. All eight scenes passed axe with zero serious/critical violations and had no horizontal document overflow. The result file is `output/playwright/admin-operations-e2e-results.json`.
