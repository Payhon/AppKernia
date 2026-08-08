---
title: Docker 开发栈
description: 理解 AppKernia 的容器、端口与数据生命周期。
---

# Docker 开发栈

默认启动的核心服务：

| 服务       | 作用                          | 宿主机端口          |
| ---------- | ----------------------------- | ------------------- |
| `postgres` | PostgreSQL 18                 | `55432`             |
| `migrate`  | 一次性迁移任务                | 无                  |
| `seed`     | 一次性核心种子任务            | 无                  |
| `api`      | Go API                        | 默认仅 Compose 网络 |
| `admin`    | React Admin 静态站与 API 反代 | `4174`              |

`redis`、`minio` 和 `worker` 是可选或按 profile/命令启动的依赖。核心事实仍在 PostgreSQL，不应把 Redis 当数据库使用。

```bash
make docker-up
make docker-logs
make docker-down
```

`.env` 中可以覆盖 `AK_ADMIN_PORT`、`AK_ADMIN_ORIGIN` 与 bootstrap 信息。修改端口时同时更新允许的 Admin Origin，避免浏览器请求被拒绝。
