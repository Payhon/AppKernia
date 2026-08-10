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

Core Seed 不内置固定密码。Docker 模式请运行：

```bash
make docker-bootstrap-admin
```

源码模式可以运行 `make -C server bootstrap-admin` 交互创建管理员，或按[源码开发模式](./source-development)中的步骤设置 `AK_SEED_ADMIN_PASSWORD_FILE` 后重新执行 Core Seed。

排查时注意：

- Seed 输出 `development_admin=false` 表示没有提供密码文件，因此没有执行开发管理员初始化。
- 重复运行 Seed 或 bootstrap 命令不会重置已有账号密码。不要从 Issue、示例或截图中寻找“默认密码”；应使用已有密码修改/找回流程。
- 源码 Admin 应通过 <http://localhost:4173> 的同源 `/admin-api` 代理访问 API。不要把开发前端临时改为跨域直连 `localhost:8080` 后误判为密码错误；浏览器控制台中的 CORS 错误说明请求尚未进入登录校验。
- 确认操作的是当前 Compose project 和 PostgreSQL volume；对另一个数据库执行 Seed 不会改变当前登录入口使用的账号。

## Node / pnpm 版本不符

```bash
corepack enable
make toolchain
```

当前要求 Node 24 与 pnpm 11。不要通过跳过冻结 lockfile 解决版本问题。

## 获取帮助

提交 Issue 前请附上操作系统、目标 commit、实际命令、退出码和已脱敏日志。不要公开 `.env`、Token、Cookie、密码、验证码或预签名 URL。
