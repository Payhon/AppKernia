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

## 从哪里开始

- [响应、错误与幂等](./conventions)
- [Mobile 认证 API](./mobile-auth)
- [Mobile 用户与公共资源](./mobile-resources)
- [Admin 认证 API](./admin-auth)
- [Admin 核心资源 API](./admin-core)

<div class="ak-doc-callout"><strong>版本状态</strong>当前 API 版本为 0.1.0，项目尚未发布稳定版。不要从本页复制字段后长期手写客户端；应从 OpenAPI 生成并通过 Schema Hash 检查漂移。</div>
