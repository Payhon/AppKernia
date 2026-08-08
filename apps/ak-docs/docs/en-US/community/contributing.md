---
title: Contributing
description: The AppKernia path from issue to pull request.
---

# Contributing

Thank you for giving your time to AppKernia. We value contributions that are verifiable, scoped, and easy for the next maintainer to continue.

Good first contributions include clearer documentation, equivalent `zh-CN` / `en-US` copy, missing unit or accessibility tests, complete platform evidence, and minimal issue reproductions.

1. Fork [Payhon/AppKernia](https://github.com/Payhon/AppKernia).
2. Create a branch from your fork.
3. Read the root and relevant subproject `AGENTS.md` files.
4. Reproduce the problem in [source mode](../guide/source-development).
5. Make the smallest complete change without unrelated rewrites.

An API change updates Go route/application/repository, OpenAPI, migrations/sqlc when relevant, permission seeds, audit/security events, generated Admin/Mobile clients, and tests.

A pull request should explain why, what changed, API/database/permission/security impact, exact commands and exit codes, screenshot/device evidence for UI changes, unverified platforms, risks, and rollback.

```bash
make check
```

If a platform cannot be run, say “not verified” or “blocked.” Never present a static check as physical-device acceptance.
