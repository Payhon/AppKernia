# AppKernia 后端与后台管理功能矩阵

> 本清单供 Coding Agent 拆分任务、生成权限种子、实现 API 和验收功能。  
> React 管理端只消费这里定义的 API、菜单和权限；不采用 HotGo 的 Vue 前端代码。

优先级：

- **P0**：没有它无法形成安全可运行的基础框架。
- **P1**：后台管理核心，AK 第一版正式交付范围。
- **P2**：通用平台能力，建议紧随第一版完成。
- **P3**：高级或按需模块。

---

## 1. 总览

| 模块 | 优先级 | App API | Admin API | Worker | 核心表 |
|---|---:|---:|---:|---:|---|
| 工程底座 | P0 | ✓ | ✓ | ✓ | — |
| 国际化与语言协商 | P0 | ✓ | ✓ | ✓ | `iam.users.locale`、字典/模板 locale |
| 注册/登录/Token | P0 | ✓ | ✓ |  | IAM Auth 表 |
| 当前用户/个人中心 | P0 | ✓ | ✓ |  | `iam.users`、`tenant_members` |
| 租户与成员 | P1 | ✓ | ✓ |  | `tenants`、`tenant_members` |
| 用户管理 | P1 |  | ✓ | 导入导出 | `users`、`credentials` |
| 会话与设备 | P0/P1 | ✓ | ✓ | 清理 | `devices`、`sessions`、`refresh_tokens` |
| RBAC 权限 | P1 |  | ✓ |  | `roles`、`permissions`、关系表 |
| 组织与岗位 | P1 |  | ✓ |  | `org.*`、`role_scope_units` |
| 菜单与路由上下文 | P1 |  | ✓ |  | `sys.menus`、`role_menus` |
| 字典 | P2 | 公共读取 | ✓ |  | `dict_types`、`dict_items` |
| 配置 | P2 | 公共读取 | ✓ |  | `config_items` |
| 文件存储 | P2 | ✓ | ✓ | 校验/扫描 | `storage.*` |
| 站内消息/公告 | P2 | ✓ | ✓ | 发布 | `notify.messages`、`recipients` |
| 邮件/SMS/Push | P2 | OTP | ✓ | 投递 | `notify.templates/deliveries/push_devices` |
| 定时任务 | P2 |  | ✓ | ✓ | `jobs.schedules/runs` + River |
| Outbox/Webhook | P2 |  | ✓ | ✓ | `outbox_events`、`webhook_*` |
| API Client | P2 | Machine | ✓ |  | `api_clients*` |
| 审计与安全事件 | P0/P1 | 我的登录记录可选 | ✓ | 归档 | `audit.*` |
| 地区数据 | P2 | ✓ | ✓/管理 | 导入 | `sys.regions` |
| MFA/OAuth | P3 | ✓ | ✓ |  | `mfa_*`、`oauth_accounts` |
| 代码生成 CLI | P3 |  |  | CLI | — |
| Billing | P3/可选 | ✓ | ✓ | 回调/对账 | `billing.*` |

---

## 2. P0 工程底座

### 2.1 服务与配置

**功能**

- `ak-api`、`ak-worker`、`ak-cli` 三个可执行程序。
- 环境分层配置、秘密注入、配置校验。
- PostgreSQL 连接池和健康检查。
- Redis/MinIO 可选依赖探活。
- SIGTERM 优雅关闭。
- Request ID、Trace ID、结构化日志。
- OpenAPI 3.1 导出。

**内部端点**

```text
GET /internal/v1/health/live
GET /internal/v1/health/ready
GET /internal/v1/metrics
```

**验收**

- PostgreSQL 不可用时 readiness 失败，liveness 不误判进程死亡。
- SIGTERM 后停止接收新请求并等待在途请求。
- 配置缺少必填秘密时启动失败且不打印秘密。
- 每个响应包含 Request ID。

### 2.2 错误、分页与幂等基础

**功能**

- 稳定业务错误码。
- 统一成功/错误响应。
- page/size 与 cursor 分页公共类型。
- `Idempotency-Key` 中间件和 `sys.idempotency_keys`。
- 统一 Validation Error 详情。

**验收**

- 相同身份、相同 Key、相同请求返回同一结果。
- 相同 Key、不同请求哈希返回冲突错误。
- 不能把内部 SQL/堆栈返回给客户端。


### 2.3 国际化与语言协商

**功能**

- `zh-CN`、`en-US` 两套完整 Backend Catalog。
- 默认/最终回退 `zh-CN`，平台别名规范化。
- `Accept-Language` Middleware、用户 locale 和 Request Context。
- 错误 `code + message_key + localized message`。
- Auth Context/Public Config 返回当前语言与支持语言。
- 字典、通知模板、法律文档、版本提示按语言选择。

**验收**

- 匿名请求按 `Accept-Language` 解析，登录请求优先用户偏好。
- 每个响应返回正确 `Content-Language`；可缓存本地化响应包含 `Vary`。
- `zh-CN`/`en-US` key 和占位符完全一致。
- 缺失目标翻译时确定性回退到 `zh-CN` 并记录指标，不出现空白文案。
- 同一错误在两种语言下 `code` 完全相同，仅展示文案变化。
- API 中时间、数字和金额保持语言无关格式。

---

## 3. P0 认证闭环

### 3.1 注册与密码登录

**App API**

```text
POST /api/v1/auth/register
POST /api/v1/auth/login/password
POST /api/v1/auth/login/otp
POST /api/v1/auth/token/refresh
POST /api/v1/auth/logout
POST /api/v1/auth/logout-all
```

**Admin API**

```text
POST /admin-api/v1/auth/login
POST /admin-api/v1/auth/login/captcha
POST /admin-api/v1/auth/token/refresh
POST /admin-api/v1/auth/logout
GET  /admin-api/v1/auth/context
```

**功能**

- 用户名/邮箱/手机号规范化。
- Argon2id 密码哈希。
- 登录失败统一文案，防账号枚举。
- Admin 同一 HMAC 保护范围在 30 分钟内第三次失败后强制短时、一次性 PNG 图形验证码；刷新客户端不能绕过。
- Access Token + 旋转 Refresh Token。
- Audience 隔离：`ak-mobile`、`ak-admin`、`ak-api`。
- 登录风控、失败次数和临时锁定。
- 登录事件记录。

**权限**

登录端点匿名；`auth/context` 要求有效 Admin Session。

**验收**

- 不同大小写邮箱不能重复注册。
- 并发刷新同一 Refresh Token 只能一个成功。
- 已消费 Token 重用撤销 Session Family，并创建高等级安全事件。
- Admin Token 不能访问 Mobile-only Token Audience，反之亦然。
- 密码、Token 和 OTP 不出现在日志。
- 已知与未知账号前两次均返回稳定凭据错误，第三次均返回 `IAM.AUTH.CAPTCHA_REQUIRED`；挑战与标识、Audience、来源绑定，错误/过期/已消费挑战不可复用，成功登录清零该范围失败状态。

### 3.2 密码与验证

**App API**

```text
POST /api/v1/auth/password/forgot
POST /api/v1/auth/password/reset
POST /api/v1/auth/password/change
POST /api/v1/auth/email/send-code
POST /api/v1/auth/email/verify
POST /api/v1/auth/mobile/send-code
POST /api/v1/auth/mobile/verify
```

**功能**

- OTP/Token 仅存哈希。
- 过期、尝试次数、消费状态。
- 修改密码后可选撤销其他会话。
- 密码历史和强度策略。
- 发送限流和目标级冷却。

**验收**

- 同一 Challenge 只能成功消费一次。
- 错误次数达到上限后即使验证码正确也不能使用。
- 重置密码 Token 泄漏不暴露用户是否存在。

---

## 4. P0/P1 当前用户、会话与设备

### 4.1 个人中心

**App API**

```text
GET   /api/v1/me
PATCH /api/v1/me
POST  /api/v1/me/avatar/upload-session
POST  /api/v1/me/avatar/upload-sessions/{id}/content
GET   /api/v1/me/avatar/content
```

**功能**

- 昵称、头像、性别、生日、简介、语言、时区。
- 邮箱和手机号变更走重新验证流程。
- 乐观锁更新资料。

**验收**

- 普通资料更新不能直接把邮箱/手机号标记为已验证。
- 头像由浏览器裁剪为受控正方形后，通过当前租户配置的 local/S3/MinIO ObjectStore Adapter 上传；服务端仍校验当前用户、租户、MIME 魔数、尺寸、大小和随机对象 Key，客户端不能指定 Bucket 或对象路径。

### 4.2 我的会话和设备

**App API**

```text
GET    /api/v1/me/sessions
DELETE /api/v1/me/sessions/{session_id}
GET    /api/v1/me/devices
DELETE /api/v1/me/devices/{device_id}
```

**Admin API**

```text
GET    /admin-api/v1/online-sessions
DELETE /admin-api/v1/online-sessions/{id}
GET    /admin-api/v1/users/{id}/sessions
DELETE /admin-api/v1/users/{id}/sessions/{session_id}
```

**权限码**

```text
iam.session.read_self
iam.session.revoke_self
iam.session.read
iam.session.revoke
iam.device.read_self
iam.device.revoke_self
```

**验收**

- 用户不能撤销他人的会话。
- 管理员强制下线必须有权限并写操作审计。
- 删除设备时撤销或解绑关联会话/Push Token 的策略明确。

### 4.3 当前 App 账号删除

**App API**

```text
POST /api/v1/me/account-deletion/verification-code
POST /api/v1/me/account-deletion/confirm
```

**权限/审计动作码**

```text
iam.user.delete_self
```

**验收**

- 邮箱和用户只能来自已验证 Mobile Session 与 `X-AppID`，客户端不能指定删除目标。
- 6 位邮箱验证码绑定 `app_id + user_id`，哈希存储，10 分钟有效、60 秒冷却、最多 5 次尝试且只能消费一次。
- 确认接口不可自动重试；成功后当前 App 会话、成员、偏好、Push、通知和业务关联立即失效，无冷静期、查询或撤销接口。
- 其他 App、Admin、组织、角色、计费或资源所有关系存在时保留共享身份；没有其他关系时物理删除全局身份和凭据。
- 法律同意与安全/登录/操作审计只保留不可关联个人的事实；独占对象通过事务内可靠队列幂等清除。

---

## 5. P1 用户、租户与成员管理

### 5.1 用户管理

**Admin API**

```text
GET    /admin-api/v1/users
POST   /admin-api/v1/users
GET    /admin-api/v1/users/{id}
PATCH  /admin-api/v1/users/{id}
POST   /admin-api/v1/users/{id}/enable
POST   /admin-api/v1/users/{id}/disable
POST   /admin-api/v1/users/{id}/unlock
POST   /admin-api/v1/users/{id}/reset-password
PUT    /admin-api/v1/users/{id}/roles
POST   /admin-api/v1/users/import
POST   /admin-api/v1/users/export
```

**权限码**

```text
iam.user.read
iam.user.create
iam.user.update
iam.user.enable
iam.user.disable
iam.user.unlock
iam.user.reset_password
iam.user.assign_role
iam.user.import
iam.user.export
```

**后台页面**

- 用户列表与条件筛选。
- 用户详情/编辑。
- 角色、组织和岗位分配。
- 会话和登录记录。
- 批量导入/导出任务。

**验收**

- 查询受租户和角色数据范围限制。
- 禁用用户后新请求和刷新被拒绝。
- 重置密码生成一次性临时密码或安全重置链接，不在日志中输出。
- 导入逐行返回结果，不能因单行失败造成不可恢复的全量状态。

### 5.2 租户和成员

**Admin API**

```text
GET   /admin-api/v1/tenants
POST  /admin-api/v1/tenants
PATCH /admin-api/v1/tenants/{id}
GET   /admin-api/v1/tenants/{id}/members
POST  /admin-api/v1/tenants/{id}/members
PATCH /admin-api/v1/tenants/{id}/members/{user_id}
```

**权限码**

```text
iam.tenant.read
iam.tenant.create
iam.tenant.update
iam.tenant.member.read
iam.tenant.member.invite
iam.tenant.member.update
iam.tenant.member.remove
```

**验收**

- 普通租户管理员不能读取其他租户。
- 租户停用后其成员不能建立新的租户 Session。
- 成员离开采用状态变化并保留审计历史。

---

## 6. P1 RBAC、数据范围与菜单

### 6.1 角色与权限

**Admin API**

```text
GET    /admin-api/v1/roles
POST   /admin-api/v1/roles
PATCH  /admin-api/v1/roles/{id}
DELETE /admin-api/v1/roles/{id}
PUT    /admin-api/v1/roles/{id}/permissions
PUT    /admin-api/v1/roles/{id}/data-scope
GET    /admin-api/v1/permissions
```

**权限码**

```text
iam.role.read
iam.role.create
iam.role.update
iam.role.delete
iam.role.assign_permission
iam.role.update_data_scope
iam.permission.read
```

**验收**

- 系统角色不能被普通租户管理员删除或改 code。
- 角色继承不能形成环。
- 过期 User Role 不参与授权。
- 修改权限后缓存有明确失效策略。
- API 权限矩阵覆盖无角色、单角色、多角色和禁用角色。

### 6.2 组织、岗位和数据范围

**Admin API**

```text
GET    /admin-api/v1/org/units/tree
POST   /admin-api/v1/org/units
PATCH  /admin-api/v1/org/units/{id}
POST   /admin-api/v1/org/units/{id}/move
DELETE /admin-api/v1/org/units/{id}
GET    /admin-api/v1/org/positions
POST   /admin-api/v1/org/positions
PATCH  /admin-api/v1/org/positions/{id}
DELETE /admin-api/v1/org/positions/{id}
PUT    /admin-api/v1/org/users/{user_id}/assignments
```

**权限码**

```text
org.unit.read
org.unit.create
org.unit.update
org.unit.move
org.unit.delete
org.position.read
org.position.create
org.position.update
org.position.delete
org.assignment.update
```

**验收**

- 节点移动不能形成环或跨租户。
- 删除有子节点/成员的组织默认拒绝。
- `department_tree` 使用 Recursive CTE。
- 数据权限在 SQL 层限制，不允许读出后过滤。

### 6.3 菜单

**Admin API**

```text
GET    /admin-api/v1/menus/tree
POST   /admin-api/v1/menus
PATCH  /admin-api/v1/menus/{id}
POST   /admin-api/v1/menus/{id}/move
DELETE /admin-api/v1/menus/{id}
PUT    /admin-api/v1/roles/{id}/menus
GET    /admin-api/v1/auth/context
```

**权限码**

```text
sys.menu.read
sys.menu.create
sys.menu.update
sys.menu.move
sys.menu.delete
iam.role.assign_menu
```

**验收**

- 返回菜单树只包含角色可见、状态有效和前端已注册的 `component_key`。
- 菜单隐藏不影响后端权限判定。
- 服务器不能下发可执行脚本作为组件。

---

## 7. P1 审计与安全管理

**Admin API**

```text
GET  /admin-api/v1/audit/operations
GET  /admin-api/v1/audit/logins
GET  /admin-api/v1/audit/security-events
POST /admin-api/v1/audit/security-events/{id}/resolve
```

**权限码**

```text
audit.operation.read
audit.login.read
audit.security.read
audit.security.resolve
```

**功能**

- 管理写操作审计。
- 登录成功/失败/阻断。
- Token 重放、越权尝试、异常风控事件。
- 按租户、用户、IP、资源、Request ID、时间过滤。

**验收**

- 审计不能包含密码、Token、验证码、Cookie、完整 API Secret。
- 登录标识只保存 SHA-256/HMAC Hash 和脱敏 Hint。
- 普通管理员不能删除或修改审计记录。

---

## 8. P2 字典与配置

### 8.1 字典

**App API**

```text
GET /api/v1/public/dictionaries/{code}
```

**Admin API**

```text
GET   /admin-api/v1/dictionaries/{code}
GET   /admin-api/v1/dict-types
POST  /admin-api/v1/dict-types
PATCH /admin-api/v1/dict-types/{id}
GET   /admin-api/v1/dict-types/{id}/items
POST  /admin-api/v1/dict-types/{id}/items
PATCH /admin-api/v1/dict-items/{id}
DELETE /admin-api/v1/dict-items/{id}
```

**权限码**

```text
sys.dictionary.read
sys.dictionary.create
sys.dictionary.update
sys.dictionary.delete
```

**验收**

- 同一字典 value 在同一 locale 中唯一，`NULL locale` 也唯一。
- 系统类型与系统项不可被租户修改或删除；租户项按 value 覆盖全局项。
- 消费解析顺序为请求 locale、neutral、`zh-CN`，同级优先租户项；禁用覆盖项会隐藏同 value 的全局项。
- `fixed/open/registered/s3_compatible` 扩展策略由后端强制执行；驱动字典只绑定编译期能力，不加载动态代码。
- 核心种子包含固定的 `system.language`、`storage.driver`、`sms.provider` 与 SMS/Email 模板用途，并提供 `zh-CN`、`en-US` 标签；`system.language` 只负责受支持语言的展示顺序和名称，不扩大 `SupportedLocale` 协议。
- 公开接口只能读取 `visibility=public` 的字典，Admin 消费接口不暴露租户覆盖内部结构。

### 8.2 配置

**App API**

```text
GET /api/v1/public/config
```

**Admin API**

```text
GET   /admin-api/v1/configs
POST  /admin-api/v1/configs
PATCH /admin-api/v1/configs/{id}
POST  /admin-api/v1/configs/{id}/rotate-secret
GET   /admin-api/v1/files/upload-policy
GET   /admin-api/v1/apps/{app_id}/scanner-config
PUT   /admin-api/v1/apps/{app_id}/scanner-config
```

**权限码**

```text
sys.config.read
sys.config.create
sys.config.update
sys.config.rotate_secret
app.scanner_config.read
app.scanner_config.update
```

**验收**

- 公开接口只返回 `is_public=true` 且非秘密配置。
- `/api/v1/public/config` 的 `scanner.webview` 默认关闭且白名单为空；配置缺失和解析失败均不得放宽客户端行为。
- 扫码配置更新使用独立乐观锁、租户范围查询和专用审计事件；审计只记录开关、域名摘要和锁版本，不记录任何扫码内容。
- 域名规则由服务端统一小写、去尾点、去重和排序；只接受 ASCII/Punycode 精确域名与非公共后缀通配符，启用时至少一条、最多 100 条。
- 秘密配置加密存储；列表和详情不返回明文。
- 配置按 value_type 和 JSON Schema 校验。
- 修改后缓存精确失效。
- `seed core` 为活动租户幂等补齐 9 类系统目录配置，并保留现有值和密封 Secret。
- 目录配置只允许修改当前值或轮换 Secret，目录元数据和默认值不可变。

---

## 9. P2 文件存储

**App/Admin API**

```text
POST /api/v1/me/avatar/upload-session
POST /api/v1/me/avatar/upload-sessions/{id}/content
GET  /api/v1/me/avatar/content
GET  /admin-api/v1/files/upload-policy
POST /admin-api/v1/files/upload-sessions
GET  /admin-api/v1/files/upload-sessions/{id}
PUT  /admin-api/v1/files/upload-sessions/{id}/parts/{partNumber}
POST /admin-api/v1/files/upload-sessions/{id}/complete
DELETE /admin-api/v1/files/upload-sessions/{id}
GET  /admin-api/v1/files
POST /admin-api/v1/files/{id}/presign-download
GET  /admin-api/v1/files/{id}/content
DELETE /admin-api/v1/files/{id}
```

**权限码**

```text
storage.file.upload_self
storage.file.read
storage.file.upload
storage.file.download
storage.file.delete
```

**Worker**

- 完成后 HEAD/大小校验。
- SHA-256 和 MIME/魔数检测。
- 病毒扫描。
- 隔离、清理过期 Upload Session、孤儿对象回收。

**验收**

- Object Key 随机生成。
- 扩展名/MIME 欺骗、超限、跨租户下载被拒绝。
- 只有 ready 且 clean/skipped 文件可引用。
- V1 不接受任意远程 URL 转存。
- 上传策略和对象路由由租户的 local/S3-compatible/MinIO 配置决定；生产禁止 local 和非 TLS 远端 Endpoint。
- Admin 响应可展示 provider，但不得返回 Bucket、Object Key 或存储凭据。

---

## 10. P2 消息与通知

### 10.1 站内消息/公告

**App API**

```text
GET   /api/v1/me/notifications
PATCH /api/v1/me/notifications/{message_id}/read
POST  /api/v1/me/notifications/read-all
```

**Admin API**

```text
GET   /admin-api/v1/notices
POST  /admin-api/v1/notices
PATCH /admin-api/v1/notices/{id}
POST  /admin-api/v1/notices/{id}/publish
POST  /admin-api/v1/notices/{id}/cancel
```

**权限码**

```text
notify.message.read_self
notify.message.mark_read_self
notify.notice.read
notify.notice.create
notify.notice.update
notify.notice.publish
notify.notice.cancel
```

**验收**

- 用户只能读取自己的收件记录。
- 发布动作幂等，不重复生成收件人。
- WebSocket 只发送新消息提示，不作为最终已读状态事实源。

### 10.2 渠道模板和投递

**Admin API**

```text
GET /admin-api/v1/notification-templates
POST /admin-api/v1/notification-templates
PATCH /admin-api/v1/notification-templates/{id}
GET /admin-api/v1/notification-templates/{id}/sms-bindings
PUT /admin-api/v1/notification-templates/{id}/sms-bindings/{provider}
DELETE /admin-api/v1/notification-templates/{id}/sms-bindings/{provider}
POST /admin-api/v1/notification-templates/{id}/test
GET /admin-api/v1/notification-deliveries
POST /admin-api/v1/notification-deliveries/{id}/retry
```

**权限码**

```text
notify.template.read
notify.template.create
notify.template.update
notify.template.test
notify.delivery.read
notify.delivery.retry
```

**验收**

- 模板变量按 Schema 校验。
- HTML 保存时净化，渲染只允许声明变量并执行 HTML 转义；短信外部模板 ID 按租户和供应商绑定。
- Delivery 与 River Job 在同一事务创建，以 `dedupe_key` 去重；投递目标与变量载荷使用信封加密。
- 腾讯云与阿里云短信使用编译期官方 SDK Adapter；SMTP 支持 STARTTLS/implicit TLS、context 与超时。
- Provider 错误分类为可重试/永久失败/结果不确定；不确定短信不自动重放，人工重试必须确认重复扣费风险。
- 找回密码按注册租户解析 `email/password_reset` 模板后异步入队，匿名响应继续保持账号枚举防护。
- 不在日志中记录完整短信验证码、邮件重置链接或 Push Token。

---

## 11. P2 任务调度与事件

### 11.1 定时计划

**Admin API**

```text
GET   /admin-api/v1/job-schedules
POST  /admin-api/v1/job-schedules
PATCH /admin-api/v1/job-schedules/{id}
POST  /admin-api/v1/job-schedules/{id}/pause
POST  /admin-api/v1/job-schedules/{id}/resume
POST  /admin-api/v1/job-schedules/{id}/execute
GET   /admin-api/v1/job-schedules/{id}/runs
```

**权限码**

```text
jobs.schedule.read
jobs.schedule.create
jobs.schedule.update
jobs.schedule.pause
jobs.schedule.execute
jobs.run.read
```

**验收**

- `handler_key` 必须存在于编译期注册表。
- Cron 和 IANA 时区合法。
- overlap/misfire/timeout/retry 策略有效。
- 不能通过 Payload 执行 Shell、SQL 或任意代码。

### 11.2 Outbox 和 Webhook

**Admin API**

```text
GET   /admin-api/v1/webhooks
POST  /admin-api/v1/webhooks
PATCH /admin-api/v1/webhooks/{id}
POST  /admin-api/v1/webhooks/{id}/test
GET   /admin-api/v1/webhooks/{id}/deliveries
```

**权限码**

```text
sys.webhook.read
sys.webhook.create
sys.webhook.update
sys.webhook.test
sys.webhook.delivery.read
```

**验收**

- 业务事务和 Outbox 写入原子。
- Webhook 使用时间戳、事件 ID 和 HMAC 签名。
- 接收方重放可识别；发送方重试幂等。
- Endpoint URL 防 SSRF，响应体只保存限制长度摘要。

---

## 12. P2 API Client

**Admin API**

```text
GET    /admin-api/v1/api-clients
POST   /admin-api/v1/api-clients
GET    /admin-api/v1/api-clients/{id}
PATCH  /admin-api/v1/api-clients/{id}
POST   /admin-api/v1/api-clients/{id}/secrets
DELETE /admin-api/v1/api-clients/{id}/secrets/{secret_id}
PUT    /admin-api/v1/api-clients/{id}/permissions
```

**权限码**

```text
sys.api_client.read
sys.api_client.create
sys.api_client.update
sys.api_client.rotate_secret
sys.api_client.revoke_secret
sys.api_client.assign_permission
```

**验收**

- 原始 Secret 只在创建时返回一次。
- 数据库只存 Secret Hash 和 Prefix。
- CIDR Allowlist、过期时间和禁用状态生效。
- Machine Token 使用 `ak-api` Audience，不能登录 Admin。

---

## 13. P2 地区与编译模块目录

**API**

```text
GET /api/v1/regions
GET /admin-api/v1/regions
POST /admin-api/v1/regions
PATCH /admin-api/v1/regions/{code}
DELETE /admin-api/v1/regions/{code}
GET /admin-api/v1/ops/runtime-summary
```

**权限码**

```text
sys.region.read
sys.region.create
sys.region.update
sys.region.delete
ops.health.read
```

**验收**

- `blueprint/backend/spec/core-regions.json` 固定记录来源提交、许可证和层级映射；当前 HotGo 快照含 3,663 条地区编码，规范化为 34/357/3,272 条 0/1/2 级记录。
- `ak-cli seed core` 按父级优先顺序幂等 upsert，稳定 code 不随名称改变；缺失经纬度保留为 `NULL`，不伪造邮编或坐标，且不覆盖后台标记为手工维护或软删除的记录。
- Admin 可在 level 0/1 节点下新增直接下级，编辑非结构字段，并软删除无下级的叶子节点；所有写入使用精确权限、乐观锁和操作审计。
- 地区目录是带来源版本的初始化快照，不宣称等同于最新官方行政区划；升级必须重新生成目录、审查 diff 并重跑完整性测试。
- `blueprint/backend/spec/core-modules.json` 固定登记 8 个领域模块；初始化按稳定 code 幂等 upsert，并清理目录外测试残留。
- `sys.modules` 只由服务状态运行摘要读取，不提供独立管理 API 或页面，也不支持上传、安装或执行二进制插件。
- API、CLI、Worker、种子和运行摘要共用构建注入版本；本地未注入时明确为 `dev`。

---

## 14. P3 MFA 与 OAuth

**功能**

- TOTP 注册、验证、禁用。
- WebAuthn 凭据注册和验证。
- MFA 恢复码。
- Apple、Google、微信等 OAuth/OIDC Adapter。
- 高风险操作 Step-up Authentication。

**权限码**

```text
iam.mfa.manage_self
iam.oauth.manage_self
iam.mfa.reset
```

**验收**

- TOTP Secret 加密存储。
- WebAuthn Sign Count 与 Challenge 防重放。
- OAuth `state`、PKCE、Nonce 校验。
- 管理员重置 MFA 必须强审计并要求 Step-up。

---

## 15. P3 代码生成 CLI

**命令**

```text
ak gen module <name>
ak gen crud --module <name> --table <schema.table>
ak gen permission --module <name>
ak openapi export
ak seed core
ak bootstrap-admin
```

**验收**

- 支持 dry-run diff。
- 不覆盖手写文件。
- 生成后自动 gofmt、sqlc、测试和 OpenAPI 检查。
- CLI 可在开发/CI 使用，生产 HTTP API 不暴露生成能力。

---

## 16. P3 Optional Billing

**范围**

- 支付业务单、渠道交易、异步回调。
- 退款。
- 余额/积分账户和不可变账本。
- 提现。
- 对账和差错处理。

**建议权限码**

```text
billing.payment.read
billing.payment.create
billing.payment.close
billing.refund.read
billing.refund.create
billing.wallet.read
billing.wallet.adjust
billing.withdrawal.read
billing.withdrawal.review
```

**验收**

- 金额一律使用最小货币单位 bigint。
- 回调验签、金额/币种/商户校验和幂等。
- 钱包快照和账本同事务更新。
- 账本不 UPDATE/DELETE，错误通过补偿流水修正。
- 提现目标加密，列表只返回脱敏信息。

---

## 17. React Admin 页面建议

以下仅定义后端所需数据和权限，不改变已确定的 React 技术栈：

```text
/dashboard
/system/users
/system/tenants
/system/org-units
/system/positions
/system/roles
/system/permissions
/system/menus
/system/dictionaries
/system/configs
/system/files
/system/notices
/system/notification-templates
/system/job-schedules
/system/online-sessions
/system/audit/operations
/system/audit/logins
/system/audit/security-events
/system/api-clients
/system/webhooks
/system/modules
/profile
/profile/security
```

`component_key` 必须对应 React 代码中静态注册的键，例如：

```text
system.users
system.roles
system.menus
system.audit.operations
profile.security
```

---

## 18. 第一版发布边界

### AK Core 1.0 必须完成

- P0 全部。
- P1 全部。
- P2 中的字典、配置、文件、站内通知、定时任务、审计查询。
- OpenAPI、权限种子、菜单种子、迁移和 Agent 规范。

### 可以延后

- 多渠道短信/邮件供应商全部实现；先提供接口和一个开发 Adapter。
- Webhook、API Client、MFA/OAuth。
- 外部 Kafka/NATS。
- Billing。
- PostgreSQL RLS 和审计分区。

---

## 19. 功能完成判定

每个矩阵条目只有同时具备以下产物才算完成：

1. 数据库迁移和 sqlc Query。
2. Domain/Application/Repository/Transport 实现。
3. OpenAPI 契约和错误码。
4. 权限码及路由绑定。
5. 菜单/字典/配置种子（需要时）。
6. 审计、幂等、事务和数据范围策略。
7. 单元、集成、契约、租户越权和关键并发测试。
8. React/uni-app x 可消费的生成客户端更新。
9. 文档、回滚说明和真实测试证据。

## App help and feedback

- CMS: published bilingual `faq`, `contact-support`, `about-us` reuse `/api/v1/public/pages/{slug}` and existing admin pages/revisions/publishing. No hardcoded support address is seeded.
- Mobile: authenticated `/api/v1/me/feedbacks` collection/detail plus dedicated upload/cancel/private-content routes. Identity comes from an `ak-mobile` token and matching App header; body fields cannot select user or tenant.
- Admin: `/admin-api/v1/apps/{app_id}/feedbacks` collection/detail/status/reply/private-content endpoints. Permissions `app.feedback.read`, `.update`, `.reply` are checked on the server. Replies may choose the resulting processing state; independent status changes require `.update`.
- Writes: UUID `Idempotency-Key` plus payload hash for submission/reply, row version conflict for status/reply, no automatic unprotected write retries. Failures retain client in-memory drafts.
- Images: JPEG/PNG/WebP only, up to 5 MiB and any stricter configured image policy, signature/type/dimension/full-decode validation; at most three attachments, purpose/owner/App checks. Only `ready + clean` can attach or preview; `skipped`, `pending`, `infected`, `failed` and quarantined/deleted rows are denied. Feedback requires the ClamD Unix INSTREAM adapter (`AK_FEEDBACK_CLAMD_SOCKET`); unset/unavailable scanners fail closed. Protocol mocks exist only in tests. Live signature-engine acceptance remains an operational gate.
- Excluded: live chat, user follow-up messages, support assignment and push reminders.

## 内置公开 H5（ADR-0030）

Server 新增 `publicweb` HTML Transport；GoFrame + html/template + go:embed，无 Node 运行依赖。模板固定在 `resource/tpl`，业务读取经过 Application 接口。公众号式资讯/单页阅读；统一 App 下载页，平台识别全部在原生 JS。旧 `/s/{slug}?app_id=` 使用同一模板与新 canonical。公开入口忽略管理端 Cookie/Token，不拓宽读取范围。稳定平台枚举沿用 android/ios/harmony。

新增 app.public_web.read/update 与乐观锁配置；存量默认不公开下载页。同一配置还管理资讯/单页下载推广的显示开关及双语标题、说明、按钮文字，关闭后服务端不输出推广 DOM；空文案使用内置双语回退。公开域名由 AK_PUBLIC_WEB_BASE_URL 指定，生产必须 HTTPS origin，不能信任 Host。APK 资源入口每次重新检查 App、开关、当前发布及文件状态，并复用短期签名。视频布局使用播放器内浮动图标切换，语言图标位于页头。完整契约见 server/openapi/openapi.yaml，运维和验收见 docs/manual/public-web.md。
