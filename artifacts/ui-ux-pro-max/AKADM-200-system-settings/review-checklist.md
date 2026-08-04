# AKADM-200 Review Checklist

- [ ] Config and dictionary routes enforce view permissions.
- [ ] Every action uses the exact create/update/delete/rotate permission.
- [ ] Secret plaintext/ciphertext never appears in list, detail, error, audit, screenshot or fixture.
- [ ] Config values are validated by value type and validation schema with optimistic version conflict handling.
- [ ] Dictionary locale fallback and system lock states are explicit.
- [ ] URL filters, selection, sort and pagination restore after reload.
- [ ] All visible copy uses matching zh-CN/en-US keys and switches without reload.
- [ ] Loading, empty, error, retry, success, conflict and destructive-confirm states are reviewed.
- [ ] 375/768/1024/1440 screenshots and axe checks are reviewed.
- [ ] PostgreSQL integration, OpenAPI, Docker API/Admin and Chromium E2E are verified.
