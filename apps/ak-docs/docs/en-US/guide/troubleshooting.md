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

Core Seed does not contain a fixed password. In Docker mode, run:

```bash
make docker-bootstrap-admin
```

In source mode, run `make -C server bootstrap-admin` for interactive creation, or follow the `AK_SEED_ADMIN_PASSWORD_FILE` procedure in [source development](./source-development) before running Core Seed again.

When diagnosing a failed sign-in:

- `development_admin=false` in Seed output means no password file was supplied, so the development administrator branch did not run.
- Re-running Seed or bootstrap does not reset an existing account's password. Do not look for a default password in issues, examples, or screenshots; use the existing password-change or recovery flow.
- Source Admin should reach the API through the same-origin `/admin-api` proxy at <http://localhost:4173>. A browser CORS error after pointing the frontend directly at `localhost:8080` means the request did not reach credential validation.
- Confirm the active Compose project and PostgreSQL volume. Seeding a different database does not change the account used by the current login endpoint.

## Wrong Node or pnpm version

```bash
corepack enable
make toolchain
```

The current baseline is Node 24 and pnpm 11. Do not bypass the frozen lockfile to hide a toolchain mismatch.

## Ask for help

Include the OS, target commit, exact command, exit code, and redacted logs in an issue. Never publish `.env`, tokens, cookies, passwords, OTPs, or presigned URLs.
