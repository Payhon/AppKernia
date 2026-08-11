---
title: 总体架构
description: AppKernia 三端一体的模块化架构。
---

# 总体架构

AppKernia 是一个模块化单体服务端驱动两类客户端的三端系统。Admin 与 Mobile 共享业务事实和 OpenAPI 契约，但使用不同的路由前缀、Token Audience、交互模型和本地存储策略。

<div className="ak-diagram" role="group" aria-label="AppKernia 三端总体架构">

```mermaid
flowchart TB
  accTitle: AppKernia 三端总体架构
  accDescr: Mobile 通过 App API，Admin 通过 Admin API 进入同一个模块化服务端；服务端使用 PostgreSQL、对象存储、任务队列和可观测能力。
  subgraph Clients["客户端产品面"]
    direction LR
    User["App 用户"] --> Mobile["AK Mobile<br/>uni-app x / UTS / UVue"]
    Operator["运营与开发者"] --> Admin["AK Admin<br/>React / Vite / TanStack"]
  end
  Mobile --> AppBoundary["/api/v1<br/>ak-mobile Audience"]
  Admin --> AdminBoundary["/admin-api/v1<br/>ak-admin Audience"]
  AppBoundary --> API["ak-api<br/>GoFrame HTTP boundary"]
  AdminBoundary --> API
  API --> Modules["模块化 Application 与 Domain"]
  subgraph Foundations["持久化与异步基础设施"]
    direction LR
    PG[("PostgreSQL 18")]
    Storage["S3-compatible<br/>Local adapter"]
    River["River 内部任务"]
    Outbox["Transactional Outbox"]
  end
  Modules --> PG
  Modules --> Storage
  Modules --> River
  Modules --> Outbox
  River --> Worker["ak-worker"]
  Outbox --> Integrations["Webhook / 外部消息适配器"]
  API --> OTel["OpenTelemetry"]
  Worker --> OTel
```

</div>

<p className="ak-diagram-summary">两类客户端进入不同的安全边界，但最终由同一组服务端模块、事务和数据约束维护业务事实；Worker、Outbox 和 OpenTelemetry 补齐异步与可观测链路。</p>

## Admin / Web 架构

<div className="ak-diagram" role="group" aria-label="AK Admin Web 前端架构">

```mermaid
flowchart TB
  accTitle: AK Admin Web 前端架构
  accDescr: 浏览器中的 React App Shell 组织静态路由、动态菜单、会话、查询、表单和 UI；生成客户端只访问 Admin API，后端仍是最终授权者。
  Browser["浏览器"] --> Shell["React App Shell"]
  subgraph SPA["AK Admin SPA"]
    direction TB
    Shell --> Router["TanStack Router<br/>静态 Route Registry"]
    Shell --> Session["Auth Session Manager<br/>Access Token 仅内存"]
    Shell --> Query["TanStack Query<br/>Server State"]
    Shell --> Store["Zustand<br/>主题 / 语言 / Shell 状态"]
    Router --> Features["Feature 页面与 Route Guard"]
    Features --> Forms["RHF + Zod"]
    Features --> UI["Ant Design + AK UI"]
    Query --> Client["Generated OpenAPI Client"]
  end
  Client -->|"/admin-api/v1"| AdminAPI["Go Admin API"]
  AdminAPI --> Authz["服务端授权与 SQL 数据范围"]
  AdminAPI --> Context["GET /auth/context<br/>菜单 / 权限 / 租户"]
  Context --> Router
```

</div>

<p className="ak-diagram-summary">Admin 的路由代码在构建时静态存在，后端菜单只引用允许的注册键。TanStack Query 管理服务端数据，Zustand 只保存客户端 Shell 状态；按钮和路由守卫改善体验，但不能替代 Go API 授权。</p>

## Server 架构

<div className="ak-diagram" role="group" aria-label="AK Server 模块化单体架构">

```mermaid
flowchart TB
  accTitle: AK Server 模块化单体架构
  accDescr: API、Worker 与 CLI 进入共享的应用和领域模块；Transport 依赖 Application，Application 依赖 Domain 和 Ports，基础设施适配器实现 Ports 并连接数据库、存储与外部系统。
  subgraph Entry["进程与入口"]
    API["ak-api<br/>HTTP / WebSocket"]
    Worker["ak-worker<br/>River / Outbox"]
    CLI["ak-cli<br/>migrate / seed / generate"]
  end
  API --> Transport["Transport<br/>解析 / 校验 / 响应"]
  Transport --> Application["Application<br/>Use Case / 事务 / 幂等"]
  Worker --> Application
  CLI --> Application
  Application --> Domain["Domain<br/>实体 / 规则 / 状态机"]
  Application --> Ports["Application 与 Domain Ports"]
  Infra["Infrastructure Adapters<br/>pgx / sqlc / S3 / Provider"] -.->|"实现"| Ports
  subgraph Backends["基础设施目标"]
    direction LR
    PG[("PostgreSQL 18")]
    Objects["S3-compatible storage"]
    External["通知 / OAuth / Webhook"]
  end
  Infra --> PG
  Infra --> Objects
  Infra --> External
  subgraph Effects["可信副作用"]
    direction LR
    Audit["审计与安全事件"]
    Jobs["River / Outbox 入队"]
  end
  Application --> Audit
  Application --> Jobs
```

</div>

<p className="ak-diagram-summary">模块化单体保留一个可事务协作的部署边界，同时用 Application、Domain 与 Port 隔离业务和基础设施。关键租户过滤、锁和约束留在 PostgreSQL/sqlc 可审计实现中。</p>

## Mobile 架构

<div className="ak-diagram" role="group" aria-label="AK Mobile 跨平台分层架构">

```mermaid
flowchart TB
  accTitle: AK Mobile 跨平台分层架构
  accDescr: UVue 页面使用 AK UI 并进入 Presentation 和 Application；Use Case 通过 Ports 访问网络、安全存储、推送和平台能力，Android、iOS 与 HarmonyOS 差异被 Adapter 隔离。
  Pages["pages/*.uvue<br/>Feature Page"] --> Presentation["Presentation<br/>View State / Navigation"]
  Presentation --> Application["Application Use Cases<br/>登录 / 刷新 / 资料 / 租户"]
  Application --> Repositories["Repository / Query Cache<br/>tenant + subject scoped"]
  Application --> Ports["Core Ports<br/>Network / SecureStorage / Push / OAuth / Device"]
  Repositories --> Client["Generated UTS API Client"]
  Client --> HTTP["AkHttpClient → /api/v1"]
  Ports --> Adapters["Platform Adapters"]
  Adapters --> Android["Android"]
  Adapters --> IOS["iOS"]
  Adapters --> Harmony["HarmonyOS NEXT"]
  Pages --> AKUI["AK UI · ak-*"]
  AKUI --> UView["uView Ultra / uni native / adapter"]
```

</div>

<p className="ak-diagram-summary">业务页面只组合 Feature 和 `ak-*` 组件，不直接拼 API URL、读写 Token 或调用平台 SDK。平台声称仍需要目标平台独立的构建、安装与设备证据。</p>

## 三个产品面的职责

### AK Mobile

使用 uni-app x、UTS/UVue 与 VDOM。业务页面通过 `ak-*` 组件、Application Use Case 与 Port 访问平台和网络能力，避免直接依赖原生 SDK 或 uView 细节。

### AK Admin

React SPA 只访问 `/admin-api/v1`。路由代码静态编译，菜单由服务端过滤，最终授权仍由 Go API 强制执行。

System 在数据层继续是一级菜单，但 Shell 将它放到侧栏底部齿轮；普通主菜单独立滚动。旁边的文档图标打开公开、独立构建的 OpenAPI 页面，既不读取 Admin 会话凭据，也不改变现有路由和权限语义。详见[在线 OpenAPI 文档与系统菜单](../api/online-reference)。

### AK Server

GoFrame 负责 HTTP 边界，pgx/v5 + sqlc 负责 PostgreSQL 数据访问。API、Worker 与 CLI 共享代码库；内部任务使用 River，外部事件使用 Transactional Outbox。

## 契约先行

`server/openapi/openapi.yaml` 是 API 最终事实源。Admin 和 Mobile 从契约生成类型；任何接口改动必须同时更新路由、用例、数据库/权限/审计（如涉及）和测试。

| 变化           | 必须一起检查                                                  |
| -------------- | ------------------------------------------------------------- |
| API 字段或响应 | OpenAPI、Go 实现、两个生成 Client、契约测试                   |
| 权限或数据范围 | 权限 Seed、后端中间件/Application、SQL 过滤、拒绝路径测试     |
| 用户可见文案   | 稳定错误码、`zh-CN`/`en-US` Catalog、`Content-Language` 与 UI |
| 异步副作用     | 事务边界、River/Outbox、幂等、重试、审计与可观测性            |
