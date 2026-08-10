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
```

`make setup` installs the locked pnpm dependencies, starts PostgreSQL, applies migrations, and writes the core seed data.

## Create the administrator interactively

For a first operator-led initialization, use the interactive command:

```bash
make -C server bootstrap-admin
```

The email, tenant, display name, and locale come from `AK_BOOTSTRAP_EMAIL`, `AK_BOOTSTRAP_TENANT_CODE`, `AK_BOOTSTRAP_TENANT_NAME`, `AK_BOOTSTRAP_DISPLAY_NAME`, and `AK_BOOTSTRAP_LOCALE` in `.env`. The password must contain at least 12 characters and is read only from the current terminal, so it does not appear in command arguments or shell history.

## Initialize through Core Seed with a password file (development only)

For repeatable local database rebuilds, `seed core` can read the initial password from a protected local file. The following flow does not put the password itself in an environment variable, command argument, or terminal output:

```bash
mkdir -p .secrets
chmod 700 .secrets
printf 'Seed administrator password: '
read -r -s AK_LOCAL_SEED_PASSWORD; printf '\n'
printf '%s\n' "$AK_LOCAL_SEED_PASSWORD" > .secrets/seed-admin-password
unset AK_LOCAL_SEED_PASSWORD
chmod 600 .secrets/seed-admin-password

AK_SEED_ADMIN_PASSWORD_FILE="$PWD/.secrets/seed-admin-password" \
  make -C server seed-core
```

The default email is `admin@appkernia.local`; set `AK_SEED_ADMIN_EMAIL` on the same command to override it. `development_admin=true` in the successful output confirms that the administrator branch ran. Without `AK_SEED_ADMIN_PASSWORD_FILE`, Core Seed still writes permissions, menus, dictionaries, and configuration, but reports `development_admin=false` and does not create an administrator.

Observe these requirements:

- This path is accepted only when `AK_ENV=development`. Other environments reject it; initialize production interactively from a controlled operator terminal.
- `.secrets/` is excluded from Git and the Docker build context, but the directory must remain `0700` and the file `0600`. Never copy the value into `.env`, issues, logs, screenshots, or chat messages.
- The file is used only when the account is missing. Re-running Seed reconciles roles, permissions, and menus, but **does not change an existing password**.
- An existing account must be active, have a usable credential, and be an active member of the target tenant. Otherwise Seed fails instead of granting cross-tenant access.
- Remove the local password file after initialization when repeatable provisioning is no longer needed. If retained, continue to treat it as a secret and restrict backups, synchronization, and read access.
- Do not copy the password file into a Docker image. Use `make docker-bootstrap-admin` for Docker mode.

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
