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
make -C server bootstrap-admin
```

`make setup` 安装冻结的 pnpm 依赖、启动 PostgreSQL、执行迁移并写入核心种子。

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
