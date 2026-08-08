---
title: 故障排查
description: 处理第一次启动时最常见的问题。
---

# 故障排查

## 端口被占用

源码 Admin 默认使用 `4173`，Docker Admin 默认使用 `4174`，PostgreSQL 默认映射 `55432`。

```bash
lsof -nP -iTCP:4173 -sTCP:LISTEN
lsof -nP -iTCP:4174 -sTCP:LISTEN
lsof -nP -iTCP:55432 -sTCP:LISTEN
```

Docker 端口可在 `.env` 用 `AK_ADMIN_PORT` 或 `AK_POSTGRES_PORT` 覆盖。

## 数据库未就绪

```bash
docker compose ps postgres
docker compose logs postgres --tail=100
make db-setup
```

## 无法登录

Core Seed 不创建固定密码。请运行：

```bash
make -C server bootstrap-admin
# Docker 模式
make docker-bootstrap-admin
```

不要从 Issue、示例或截图中寻找“默认密码”。

## Node / pnpm 版本不符

```bash
corepack enable
make toolchain
```

当前要求 Node 24 与 pnpm 11。不要通过跳过冻结 lockfile 解决版本问题。

## 获取帮助

提交 Issue 前请附上操作系统、目标 commit、实际命令、退出码和已脱敏日志。不要公开 `.env`、Token、Cookie、密码、验证码或预签名 URL。
