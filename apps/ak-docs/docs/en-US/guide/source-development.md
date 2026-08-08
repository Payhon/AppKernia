---
title: Source development
description: Run the Go API, worker, and React Admin on the host.
---

# Source development

Source mode keeps PostgreSQL in Docker while Go and Vite run on the host.

| Tool    | Version                 |
| ------- | ----------------------- |
| Go      | 1.26.5 / 1.26.x         |
| Node.js | 24.x                    |
| pnpm    | 11.x                    |
| Docker  | Compose v2              |
| Python  | 3.x for contract checks |

```bash
corepack enable
make toolchain
cp .env.example .env
make setup
make -C server bootstrap-admin
```

Start the backend in terminal 1:

```bash
make dev-backend
```

Start Admin in terminal 2:

```bash
make dev-admin
```

Open <http://localhost:4173>. Vite proxies `/admin-api` to `127.0.0.1:8080`.

Run the repository gates:

```bash
make check
```

This checks blueprints, i18n contracts, backend, Admin, and Mobile static rules. It is not Android, iOS, or HarmonyOS compilation or physical-device acceptance.

Scoped checks:

```bash
make -C server check
pnpm --filter @appkernia/admin check
pnpm --filter @appkernia/docs check
apps/ak-mobile/scripts/check-project.sh
```
