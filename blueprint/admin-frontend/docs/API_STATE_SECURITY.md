# API、状态和安全契约

## API 事实源

OpenAPI 3.1 是唯一 DTO/错误码事实源；生成客户端放 `src/generated/api/`，禁止手改。Feature 层通过 query option factory 和 mutation hook 使用生成客户端，页面不直接调用 fetch。

## Token

Access Token 仅在内存；Refresh Token 使用 Secure + HttpOnly + SameSite Cookie。CSRF 使用受保护 Cookie + Header/Origin 策略。401 进入 single-flight refresh，所有并发请求共享一个 Promise，成功后各重试一次；失败清空会话并跳登录。403 不刷新。

## Auth Context

建议 `GET /admin-api/v1/auth/context` 返回：当前用户、活动租户、available_tenants、roles、permissions、menus、feature_flags、menu_revision、permission_revision、server_time。租户切换后签发新上下文并删除旧 tenant scope cache。

## Query 和 Zustand

Query Key 示例：`['tenant', tenantId, 'users', normalizedSearch]`、`['global','permission-catalog']`。Mutation 只精确失效相关 Key；快速筛选 debounce 并取消旧请求。Zustand 只保存主题、侧栏、语言、工作区和会话快照，不保存用户列表/角色详情等 Server State。

## 错误、并发和敏感数据

统一处理 400/401/403/404/409/422/429/5xx。409 显示刷新/对比，不静默覆盖。Secret 永不回显；一次性 Secret/恢复码只能展示一次。审计、日志、telemetry 不记录密码、Token、验证码、完整邮箱/手机、连接串或密钥。
