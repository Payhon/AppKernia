---
title: Troubleshooting
description: Fix common first-run problems.
---

# Troubleshooting

## Port already in use

Source Admin uses `4173`, Docker Admin uses `4174`, and PostgreSQL maps to `55432` by default.

```bash
lsof -nP -iTCP:4173 -sTCP:LISTEN
lsof -nP -iTCP:4174 -sTCP:LISTEN
lsof -nP -iTCP:55432 -sTCP:LISTEN
```

Override Docker ports with `AK_ADMIN_PORT` or `AK_POSTGRES_PORT` in `.env`.

## Database is not ready

```bash
docker compose ps postgres
docker compose logs postgres --tail=100
make db-setup
```

## Cannot sign in

Core Seed never creates a fixed password:

```bash
make -C server bootstrap-admin
# Docker mode
make docker-bootstrap-admin
```

Do not look for a default password in issues, examples, or screenshots.

## Wrong Node or pnpm version

```bash
corepack enable
make toolchain
```

The current baseline is Node 24 and pnpm 11. Do not bypass the frozen lockfile to hide a toolchain mismatch.

## Ask for help

Include the OS, target commit, exact command, exit code, and redacted logs in an issue. Never publish `.env`, tokens, cookies, passwords, OTPs, or presigned URLs.
