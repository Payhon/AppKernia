---
title: 仓库结构
description: 快速定位 AppKernia 的三端代码、契约与治理文件。
---

# 仓库结构

```text
AppKernia/
├── apps/
│   ├── ak-admin/      # React 管理平台
│   ├── ak-mobile/     # uni-app x 移动端
│   └── ak-docs/       # Rspress 官网与文档
├── server/            # Go API、Worker、CLI、迁移、sqlc、OpenAPI
├── blueprint/         # 三端蓝图与机器可读契约
├── docs/              # 项目实施状态、ADR 与交付报告
├── compose.yaml       # 本地容器开发栈
├── Makefile           # 跨项目常用入口
└── AGENTS.md          # Agent 与贡献者必须遵守的根规则
```

## 事实源

- 服务端 API：`server/openapi/openapi.yaml`
- 数据库：已发布迁移与 sqlc SQL
- Mobile 路由/页面/组件准入：`blueprint/mobile/spec/*.json`
- Admin 路由/权限/页面：`blueprint/admin-frontend/spec/*.json`
- 国际化：`blueprint/I18N_CONTRACT.md` 与 `blueprint/i18n-contract.json`

修改 API 时，需要同步 Go、OpenAPI、权限、审计、生成客户端与测试；不能只改其中一端。
