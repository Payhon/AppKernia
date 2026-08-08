---
title: Repository structure
description: Locate AppKernia code, contracts, and governance files.
---

# Repository structure

```text
AppKernia/
├── apps/
│   ├── ak-admin/      # React administration console
│   ├── ak-mobile/     # uni-app x mobile app
│   └── ak-docs/       # Rspress website and documentation
├── server/            # Go API, worker, CLI, migrations, sqlc, OpenAPI
├── blueprint/         # Three-surface blueprints and machine contracts
├── docs/              # Implementation status, ADRs, delivery reports
├── compose.yaml       # Local container stack
├── Makefile           # Cross-project commands
└── AGENTS.md          # Mandatory contributor and agent rules
```

Sources of truth:

- Server API: `server/openapi/openapi.yaml`
- Database: released migrations and sqlc SQL
- Mobile routes, pages, and component admission: `blueprint/mobile/spec/*.json`
- Admin routes, permissions, and pages: `blueprint/admin-frontend/spec/*.json`
- Internationalization: `blueprint/I18N_CONTRACT.md` and `blueprint/i18n-contract.json`

An API change must update Go, OpenAPI, permissions, audit, generated clients, and tests together.
