---
title: Run AppKernia from zero
description: A first-time path from clone to a running AppKernia system.
---

# Run AppKernia from zero

This path only requires Git and Docker Desktop. Go, Node, and PostgreSQL stay inside containers, making it the easiest first run.

## 1. Install the prerequisites

- [Git](https://git-scm.com/downloads)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) or Docker Engine with Compose v2

```bash
git --version
docker version
docker compose version
```

All three commands should print a version.

## 2. Clone the repository

```bash
git clone https://github.com/Payhon/AppKernia.git
cd AppKernia
```

If you plan to contribute, fork the repository on GitHub and clone your fork instead.

## 3. Create local configuration

```bash
cp .env.example .env
```

The included values are development-only. Never commit `.env`, administrator passwords, tokens, private keys, or third-party credentials.

## 4. Start the stack

```bash
make docker-up
```

The first run builds images, starts PostgreSQL 18, applies migrations, seeds core permissions and menus, and starts the Go API and React Admin.

## 5. Create the first administrator

```bash
make docker-bootstrap-admin
```

Enter a password of at least 12 characters when prompted. The password is read from the interactive terminal and is not written to Git or shell history. The development email comes from `AK_BOOTSTRAP_EMAIL` in `.env`.

Core Seed does not contain a fixed administrator password. Running `make docker-bootstrap-admin` again reconciles the existing administrator's role, permissions, and menus, but **does not overwrite the existing password**, so it is not a password-reset command. The Docker build context also excludes `.secrets/`. For repeatable local initialization from a password file, use the development-only procedure in [source development](./source-development).

## 6. Open the system

- Admin: <http://localhost:4174>
- Source-mode API readiness: <http://localhost:8080/internal/v1/health/ready>

If the Docker API port is not published to the host, inspect health through Compose:

```bash
docker compose ps
docker compose logs api --tail=100
```

The first run is complete when `postgres`, `api`, and `admin` are healthy and the browser displays the login page.

## 7. Stop the stack

```bash
make docker-down
```

The normal command keeps the PostgreSQL Docker volume. Back up data before deliberately deleting volumes.

## Next

- For hot reload and debugging, continue with [source development](./source-development).
- To run the mobile app, see [mobile development](./mobile-development).
- If anything fails, use [troubleshooting](./troubleshooting).
