---
title: Docker development stack
description: Understand AppKernia containers, ports, and data lifecycle.
---

# Docker development stack

| Service    | Purpose                   | Host port                  |
| ---------- | ------------------------- | -------------------------- |
| `postgres` | PostgreSQL 18             | `55432`                    |
| `migrate`  | One-shot migrations       | None                       |
| `seed`     | One-shot core seed        | None                       |
| `api`      | Go API                    | Compose network by default |
| `admin`    | React Admin and API proxy | `4174`                     |

Redis, MinIO, and the worker are optional or command-specific dependencies. PostgreSQL remains the source of truth.

```bash
make docker-up
make docker-logs
make docker-down
```

Override `AK_ADMIN_PORT`, `AK_ADMIN_ORIGIN`, and bootstrap values in `.env`. When changing the port, update the allowed Admin origin at the same time.
