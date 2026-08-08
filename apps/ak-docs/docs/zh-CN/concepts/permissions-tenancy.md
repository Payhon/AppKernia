---
title: 权限与多租户
description: 理解 AppKernia 的 RBAC、菜单、数据范围与租户隔离。
---

# 权限与多租户

AppKernia 把“看得见”和“做得到”分开：

- `sys.menus` 决定 Admin 导航。
- `iam.permissions.code` 是后端授权事实，例如 `iam.user.read`。
- 页面按钮只改善体验，不能替代 API 授权。
- `tenant_id` 只能来自已验证 Session Context，不能相信请求 Body 或 Query。

## 数据范围

角色可以使用 `all`、`tenant`、`department`、`department_tree`、`self` 或 `custom`。Repository 必须把有效范围转成 SQL 条件；禁止全量读出后在 Go 或前端过滤。

## 两条 API 边界

| 客户端 | 唯一路由前缀    | Audience    |
| ------ | --------------- | ----------- |
| Mobile | `/api/v1`       | `ak-mobile` |
| Admin  | `/admin-api/v1` | `ak-admin`  |

同一个用户也不能拿 Admin Token 访问 Mobile-only 会话，反之亦然。
