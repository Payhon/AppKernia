---
title: 源码开发模式
description: 在宿主机运行 Go API、Worker 与 React Admin。
---

# 源码开发模式

源码模式让 PostgreSQL 运行在 Docker 中，Go 与 Vite 运行在宿主机，适合日常开发。

## 环境版本

| 工具    | 版本              |
| ------- | ----------------- |
| Go      | 1.26.5 / 1.26.x   |
| Node.js | 24.x              |
| pnpm    | 11.x              |
| Docker  | Compose v2        |
| Python  | 3.x，用于契约校验 |

```bash
corepack enable
make toolchain
```

`make toolchain` 会打印实际版本并在主版本不符时退出。

## 初始化

```bash
cp .env.example .env
make setup
```

`make setup` 安装冻结的 pnpm 依赖、启动 PostgreSQL、执行迁移并写入核心种子。

### 交互式创建管理员

首次人工初始化推荐使用交互式命令：

```bash
make -C server bootstrap-admin
```

邮箱、租户、显示名称和语言分别读取 `.env` 中的 `AK_BOOTSTRAP_EMAIL`、`AK_BOOTSTRAP_TENANT_CODE`、`AK_BOOTSTRAP_TENANT_NAME`、`AK_BOOTSTRAP_DISPLAY_NAME` 和 `AK_BOOTSTRAP_LOCALE`。密码至少 12 位，只从当前终端读取，不会出现在命令参数或 Shell 历史中。

### 使用密码文件随 Core Seed 初始化（仅开发环境）

需要反复重建本地开发库时，可以让 `seed core` 从受保护的本地文件读取初始密码。下面的命令不会把密码本身放入环境变量、命令参数或终端输出：

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

邮箱默认是 `admin@appkernia.local`，可以在同一条命令前通过 `AK_SEED_ADMIN_EMAIL` 覆盖。成功输出中的 `development_admin=true` 表示管理员初始化分支已经执行；未设置 `AK_SEED_ADMIN_PASSWORD_FILE` 时，Core Seed 仍会正常写入权限、菜单、字典和配置，但显示 `development_admin=false`，也不会创建管理员。

使用密码文件时必须注意：

- 只允许在 `AK_ENV=development` 中使用；其他环境会直接拒绝，生产环境应由受控运维终端交互初始化。
- `.secrets/` 已被 Git 和 Docker 构建上下文忽略，但仍应保持目录 `0700`、文件 `0600`，且不得把内容复制到 `.env`、Issue、日志、截图或聊天记录。
- 密码文件只用于创建缺失的账号。管理员已存在时，重复执行会幂等补齐角色、权限和菜单，**不会修改已有密码**。
- 已存在账号必须处于 active 状态、具有可用密码凭据，并且是目标租户的 active 成员；否则 Seed 会失败，不会跨租户提权。
- 初始化完成后，如不再需要自动重建本地管理员，应删除本机密码文件；保留时需继续按 Secret 管理并限制备份、同步和读取权限。
- Docker 模式不要把密码文件复制进镜像；使用 `make docker-bootstrap-admin` 交互初始化。

## 启动

终端 1：

```bash
make dev-backend
```

终端 2：

```bash
make dev-admin
```

打开 <http://localhost:4173>。Vite 会把 `/admin-api` 代理到 `127.0.0.1:8080`。

## 运行质量门禁

```bash
make check
```

它会执行三份蓝图、国际化契约、后端、Admin 与 Mobile 静态门禁。它不等于 Android、iOS、HarmonyOS 的编译或真机验收。

按范围运行：

```bash
make -C server check
pnpm --filter @appkernia/admin check
pnpm --filter @appkernia/docs check
apps/ak-mobile/scripts/check-project.sh
```
