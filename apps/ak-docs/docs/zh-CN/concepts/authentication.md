---
title: 认证与会话
description: AppKernia 的 Token、会话、刷新轮换与客户端存储边界。
---

# 认证与会话

## Token 分工

| 类型          | 作用          | 客户端存储                                        | 服务端存储         |
| ------------- | ------------- | ------------------------------------------------- | ------------------ |
| Access Token  | 短期 API 身份 | Admin / Mobile 只在内存                           | 不作为长期会话事实 |
| Refresh Token | 轮换会话      | Admin Secure HttpOnly Cookie；Mobile 系统安全存储 | 仅 SHA-256 Hash    |

Access Token 使用 Ed25519/EdDSA，默认 15 分钟，并通过 `aud` 隔离 `ak-mobile`、`ak-admin` 与 `ak-api`。

## 登录如何形成会话

<div className="ak-diagram" role="group" aria-label="AppKernia 登录与会话建立流程">

```mermaid
sequenceDiagram
  accTitle: AppKernia 登录与会话建立流程
  accDescr: 客户端向对应 API 面登录，服务端验证 App、用户、密码和 Audience，写入会话与刷新令牌哈希，再返回短期访问令牌和一次性刷新令牌。
  participant C as Mobile / Admin client
  participant A as Auth API
  participant D as PostgreSQL
  C->>A: credentials + App context + intended audience
  A->>D: load active user and membership
  D-->>A: password hash + tenant membership
  A->>A: verify password, status, audience, risk policy
  A->>D: create session + SHA-256 refresh hash
  D-->>A: committed session family
  A-->>C: short-lived access + one-time refresh
```

</div>

<p className="ak-diagram-summary">登录入口先确定 Mobile 或 Admin Audience。服务端验证账号、App 与租户成员关系后才创建 Session Family；明文密码和 Refresh Token 都不会写入数据库或日志。</p>

## Refresh Rotation

<div className="ak-diagram" role="group" aria-label="AppKernia Refresh Token 轮换与重放检测">

```mermaid
sequenceDiagram
  accTitle: AppKernia Refresh Token 轮换与重放检测
  accDescr: 服务端锁定刷新令牌记录并验证哈希；活动令牌被消费并替换，已消费令牌重用会撤销整个会话家族并记录安全事件。
  participant C as Mobile / Admin client
  participant A as AK API
  participant D as PostgreSQL
  C->>A: old refresh token + expected audience
  A->>D: SELECT FOR UPDATE + verify token hash
  alt token is active and session is valid
    D-->>A: active token and session family
    A->>D: consume old + insert new hash atomically
    A-->>C: new access + one-time refresh token
  else token was consumed, revoked, or mismatched
    D-->>A: reuse / revoked / audience mismatch
    A->>D: revoke session family + security event
    A-->>C: stable unauthorized error, sign in again
  end
```

</div>

<p className="ak-diagram-summary">每次刷新都会原子消费旧令牌并生成新令牌；同一个旧 Token 并发出现时只有一个请求能成功，后续重用会触发整条会话链撤销。</p>

同一旧 Token 并发使用时只能一个成功。已消费 Token 再次出现会被视为重放，服务端撤销整个 Session Family 并创建安全事件。

## 会话在哪里失效

<div className="ak-diagram" role="group" aria-label="AppKernia 会话失效与客户端清理流程">

```mermaid
flowchart LR
  accTitle: AppKernia 会话失效与客户端清理流程
  accDescr: 主动退出、管理员撤销、密码或 MFA 安全事件、刷新重放和过期都使会话失效，客户端随后清理令牌、受保护缓存和用户上下文。
  Active["Active session"] --> Logout["用户主动退出"]
  Active --> AdminRevoke["管理员或用户撤销设备"]
  Active --> Risk["密码 / MFA / 风险安全事件"]
  Active --> Replay["Refresh 重放检测"]
  Active --> Expired["绝对或空闲过期"]
  Logout --> Revoked["Session revoked"]
  AdminRevoke --> Revoked
  Risk --> Revoked
  Replay --> Revoked
  Expired --> Revoked
  Revoked --> Clear["清理 Access Token、Refresh、受保护 Cache"]
  Clear --> SignIn["返回登录或账号恢复"]
```

</div>

<p className="ak-diagram-summary">会话失效是服务端事实。客户端收到稳定的未授权结果后清理本地身份与租户相关缓存，不尝试用隐藏页面或旧快照继续授权操作。</p>

## 客户端规则

- 401 Refresh 使用 single-flight，原请求最多重试一次。
- 403 不触发 Refresh。
- 没有幂等保证的写请求不自动重放。
- 登出、切换租户或 Session 失效后清理受保护缓存。
- Admin Refresh Token 只使用 Secure HttpOnly Cookie；Mobile Refresh Token 只进入系统安全存储。
- Access Token 不进入 localStorage、sessionStorage、IndexedDB 或普通 `uni` storage。
