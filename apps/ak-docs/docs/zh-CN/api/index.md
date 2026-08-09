---
title: 服务端 API
description: AppKernia API 的入口、鉴权边界与 OpenAPI 事实源。
---

# 服务端 API

AppKernia 使用 OpenAPI 3.1 描述服务端契约。本文档提供最常用路径、示例和安全约束；字段、枚举、响应码与 Schema 的最终事实源始终是当前仓库的 `server/openapi/openapi.yaml`。

[下载当前 OpenAPI YAML](/openapi.yaml)

## API 面

| 面           | 前缀            | 使用者                    | 身份边界                              |
| ------------ | --------------- | ------------------------- | ------------------------------------- |
| App API      | `/api/v1`       | uni-app x 与普通 App 用户 | `ak-mobile` Bearer Token              |
| Admin API    | `/admin-api/v1` | React 管理平台            | `ak-admin` Access Token + 安全 Cookie |
| Internal API | `/internal/v1`  | 探活与内部监控            | 部署网络边界                          |

Admin 与 Mobile Token 不能互换。`X-AppID` 只选择一个公开、启用的 App；租户和用户范围必须来自已验证 Session，不能由该 Header 越权指定。

## 认证流程

```text
登录 / 注册
    ↓
服务端验证身份 + Audience + App membership
    ↓
Access Token（短期） + Refresh Token（轮换）
    ↓
受保护请求 → 过期时仅通过明确刷新流程恢复
    ↓
刷新成功撤销旧链；退出或风险事件撤销 Session
```

Admin 和 Mobile 使用不同入口与 Audience。客户端不能把一次失败写请求盲目重放；只有接口明确具备幂等语义并满足[幂等约定](./conventions)时才可自动重试。

## API 家族

| 家族        | 常见资源                             | 从这里开始                        |
| ----------- | ------------------------------------ | --------------------------------- |
| 公共 App    | 配置、地区、字典、版本、法律文档     | [Mobile 资源](./mobile-resources) |
| Mobile 身份 | 登录、刷新、退出、找回、注册与验证码 | [Mobile 认证](./mobile-auth)      |
| Mobile 用户 | Profile、偏好、通知、会话与安全事件  | [Mobile 资源](./mobile-resources) |
| Admin 身份  | 登录、刷新、MFA 与 Session           | [Admin 认证](./admin-auth)        |
| Admin 业务  | 用户、组织、权限、内容、文件与任务   | [Admin 核心资源](./admin-core)    |
| Internal    | Live、Ready 与 Metrics               | 仅部署网络内使用                  |

## 所有客户端都应发送

```http
Accept: application/json
Accept-Language: zh-CN
X-Request-ID: 019…
```

App 相关接口还需要：

```http
X-AppID: 01900000-0000-7000-8000-000000000001
```

受保护的 Mobile 请求：

```http
Authorization: Bearer YOUR_ACCESS_TOKEN
```

## 第一个无需登录的请求

在本机 API 已启动后，用仓库清单里的 App ID 请求公共配置：

```bash
curl --fail-with-body \
  -H 'Accept: application/json' \
  -H 'Accept-Language: zh-CN' \
  -H 'X-AppID: 00000000-0000-4000-8000-000000000001' \
  http://127.0.0.1:8080/api/v1/public/config
```

成功响应应同时满足 HTTP 2xx、稳定 `code`、与请求语言匹配的 `Content-Language`，并且不包含任何服务器 Secret。

## 集成检查表

- 从 OpenAPI 生成 Client，并在 CI 中校验 Schema Hash。
- Mobile 只调用 `/api/v1`，Admin 只调用 `/admin-api/v1`。
- 每个请求传递 `Accept-Language`；日志保留 Request ID，但不记录 Token。
- 用后端权限测试覆盖拒绝路径，不把菜单或按钮当授权证据。
- 写请求明确幂等键、重试边界和审计行为。
- 多租户接口用两个 Tenant 的集成数据验证 SQL 隔离。

## 从哪里开始

- [响应、错误与幂等](./conventions)
- [Mobile 认证 API](./mobile-auth)
- [Mobile 用户与公共资源](./mobile-resources)
- [Admin 认证 API](./admin-auth)
- [Admin 核心资源 API](./admin-core)

<div class="ak-doc-callout"><strong>版本状态</strong>当前 API 版本为 0.1.0，项目尚未发布稳定版。不要从本页复制字段后长期手写客户端；应从 OpenAPI 生成并通过 Schema Hash 检查漂移。</div>
