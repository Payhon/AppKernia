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

<div className="ak-diagram" role="group" aria-label="AppKernia 权限与多租户决策链">

```mermaid
flowchart LR
  accTitle: AppKernia 权限与多租户决策链
  accDescr: 客户端可见菜单和按钮只提供体验提示，请求进入 API 后从会话解析用户和租户，后端检查权限并把数据范围编译为 SQL 条件。
  Context["GET /auth/context"] --> Menu["Admin 菜单 / Route Guard"]
  Context --> Button["页面按钮 / AkCan"]
  Menu --> Request["请求 /api/v1 或 /admin-api/v1"]
  Button --> Request
  Request --> Session["验证 Session + Audience"]
  Session --> Tenant["服务端解析 active tenant"]
  Tenant --> Permission["检查稳定 permission code"]
  Permission --> Scope["合并角色与组织 data scope"]
  Scope --> SQL["Repository / sqlc 注入 SQL 条件"]
  SQL --> Result["仅返回授权租户与数据范围"]
```

</div>

<p className="ak-diagram-summary">菜单和按钮可以提前避免无效操作，但真正的授权从会话、Audience 和租户上下文开始，最后必须落到服务端权限判断与 SQL 数据范围。</p>

## 数据范围

角色可以使用 `all`、`tenant`、`department`、`department_tree`、`self` 或 `custom`。Repository 必须把有效范围转成 SQL 条件；禁止全量读出后在 Go 或前端过滤。

当用户同时拥有多个角色时，Application 先根据当前租户、角色有效期和组织关系计算有效范围，再交给 Repository 构造参数化 SQL。切换租户后必须清理客户端受保护缓存，避免上一租户的数据快照被错误复用。

## 两条 API 边界

| 客户端 | 唯一路由前缀    | Audience    |
| ------ | --------------- | ----------- |
| Mobile | `/api/v1`       | `ak-mobile` |
| Admin  | `/admin-api/v1` | `ak-admin`  |

同一个用户也不能拿 Admin Token 访问 Mobile-only 会话，反之亦然。
