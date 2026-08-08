---
title: Admin 核心资源 API
description: 用户、组织、权限、内容、文件、通知、任务、配置与审计接口索引。
---

# Admin 核心资源 API

以下为常用资源面。每个路由的请求/响应 Schema、权限码和状态码以 [OpenAPI YAML](/openapi.yaml) 为准。

## 身份、组织与授权

| 资源     | 核心路径                            | 说明                                               |
| -------- | ----------------------------------- | -------------------------------------------------- |
| 用户     | `/users`、`/users/{id}`             | 查询、创建、更新、启停、解锁、重置密码、角色与会话 |
| 租户     | `/tenants`、`/tenants/{id}/members` | 租户资料与成员生命周期                             |
| 组织     | `/org/units`、`/org/positions`      | 组织树移动、岗位与成员归属                         |
| 角色     | `/roles/{id}/permissions`、`/menus` | 权限、菜单与数据范围分别管理                       |
| 在线会话 | `/online-sessions`                  | 查询并强制撤销有权限的会话                         |

所有列表和详情在服务端/SQL 层执行租户与数据范围过滤。

## App 与内容

- `/apps`：管理租户下的 App 与启停状态。
- `/apps/{app_id}/users`：App-scoped 用户管理。
- `/apps/{app_id}/content/categories`、`/apps/{app_id}/content/articles`、`/apps/{app_id}/content/pages`：App-scoped 内容。
- `/apps/{app_id}/notices`、`/apps/{app_id}/messages`：App-scoped 受众预览、发布与收件人。
- `/mobile/releases` 与 `/apps/{app_id}/mobile/releases`：三平台版本策略。

发布、取消和状态迁移使用明确的命令路由；客户端不能直接伪造持久化状态。

## 平台能力

| 资源       | 路径前缀                                                       | 安全重点                                        |
| ---------- | -------------------------------------------------------------- | ----------------------------------------------- |
| 文件       | `/files`                                                       | 随机 Object Key、上传会话、扫描状态、跨租户拒绝 |
| 通知       | `/notification-templates`、`/notification-deliveries`          | 模板变量、目标加密、重试分类                    |
| 任务       | `/job-handlers`、`/job-schedules`                              | 只能选择编译期 Handler，不执行任意代码          |
| 配置       | `/configs`                                                     | Secret 密文保存，列表不返回明文                 |
| 字典       | `/dict-types`、`/dictionaries/{code}`                          | fixed/open/registered 策略由后端强制            |
| API Client | `/api-clients`                                                 | Secret 只在创建时显示一次，数据库只存 Hash      |
| Webhook    | `/webhooks`                                                    | HMAC、重放保护、URL SSRF 防护                   |
| 审计       | `/audit/operations`、`/audit/logins`、`/audit/security-events` | 字段级脱敏、不可由普通管理员修改                |
| 运维       | `/ops/health`、`/ops/runtime-summary`                          | 运行状态与编译期模块目录                        |

## 健康检查

- `GET /internal/v1/health/live`：只判断 API 进程存活。
- `GET /internal/v1/health/ready`：依赖不可用时返回 `503`。
