# 前端蓝图要求的后端增量

机器可读待实现增量见 `admin-api-delta.json` 和 `core-permissions.delta.json`；第三方登录配置与 App binding 当前列为待实现增量。已落地接口和权限应归并到 Backend snapshot。每个增量仍必须同时提交 Go route、OpenAPI、SQL/sqlc（如需）、权限 Seed、审计、测试和前端 fixture。

## P0

- `GET /admin-api/v1/auth/public-config`：登录/注册页读取公开品牌、注册开关、验证码/MFA 方式。
- `GET /admin-api/v1/auth/csrf-token`：已落地；只返回当前同站 Admin Refresh Cookie Pair 绑定的双提交 CSRF Token，用于浏览器冷启动恢复，不返回 Refresh/Access Token。
- `POST /admin-api/v1/auth/register`：已落地；默认关闭，启用后只加入服务端配置租户并授予 `member`，重复邮箱保持相同 202 响应。
- `POST /admin-api/v1/auth/password/forgot`：已落地；已知/未知账号统一 202 与 cooldown，短期随机 Token 仅 Hash 入库，投递经 Port/Adapter。
- `POST /admin-api/v1/auth/password/reset`：已落地；一次性消费 challenge，事务更新密码历史、撤销跨 audience Session/Refresh family 并写脱敏审计。
- `GET /admin-api/v1/me`：当前管理端用户资料。
- `PATCH /admin-api/v1/me`：更新当前用户基本设置。
- `POST /admin-api/v1/me/avatar/upload-session`：已落地；创建本人/租户范围短期上传会话，服务端生成对象键。
- `PUT /admin-api/v1/me/avatar/upload-sessions/{id}/content`：已落地的 development-local Adapter 上传目标；校验真实图片后事务绑定头像、usage 与审计。
- `GET /admin-api/v1/me/avatar/content`：已落地；经本人、租户、文件状态和扫描门禁读取私有头像。
- `POST /admin-api/v1/me/password/change`：已落地；校验当前密码和最近历史，事务更新后保留当前会话并撤销其他会话/Refresh family，审计不含密码材料。
- `GET /admin-api/v1/me/sessions`：已落地；按当前 user、tenant、`ak-admin` audience 自作用域列出活动会话。
- `DELETE /admin-api/v1/me/sessions/{id}`：已落地；事务内撤销会话与 Refresh Token family，并写不可变操作审计。
- `GET /admin-api/v1/me/devices`：已落地；按本人、当前租户和 Admin audience 返回已关联设备及当前设备标记。
- `DELETE /admin-api/v1/me/devices/{id}`：已落地；撤销该设备关联的全部会话和 Refresh family、删除设备并写入审计。
- `GET /admin-api/v1/dashboard/summary`：已落地；按调用者权限省略无权 KPI，所有计数均按当前租户和时间范围查询。
- `GET /admin-api/v1/dashboard/trends`：已落地；按权限返回登录、用户、任务和安全日趋势，自动补零但不返回无权系列。
- `GET /admin-api/v1/dashboard/activity`：已落地；按权限返回最近操作、失败任务和安全事件，服务端不返回详情 JSON、Payload 或错误消息。

## P1

- `POST /admin-api/v1/auth/switch-tenant`：已落地；仅允许切换到活动成员租户，签发新租户会话并撤销旧会话。
- `DELETE /admin-api/v1/tenants/{id}/members/{user_id}`：已落地；保留成员历史、撤销租户会话并保护最后一个超级管理员。
- `GET /admin-api/v1/audit/security-events/{id}`：安全事件详情。

## P2

- 第三方登录配置：`GET /login-provider-catalog`、`GET/POST /login-provider-configs`、单项读取/更新/删除、Secret 轮换、预检、启用/停用，以及 App 范围 `GET/PUT /apps/{app_id}/login-provider-bindings`。列表/详情永不返回 Secret；binding 的四 Provider bulk PUT 必须在 serializable transaction 中全成全败并校验租户、active、preflight ready 与乐观锁。

- `GET /admin-api/v1/messages`：已落地；按当前租户、类型、状态和分页返回公告/站内消息。
- `POST /admin-api/v1/messages`：已落地；创建公告或站内消息，服务端解析收件范围并返回精确计数。
- `GET /admin-api/v1/messages/{id}`：已落地；返回租户内消息详情和脱敏收件预览。
- `PATCH /admin-api/v1/messages/{id}`：已落地；仅允许编辑草稿或计划消息并重新校验正文与收件范围。
- `POST /admin-api/v1/messages/{id}/publish`：已落地；事务发布并幂等生成收件人。
- `POST /admin-api/v1/messages/{id}/cancel`：已落地；取消允许状态内的消息并写不可变审计。
- `GET /admin-api/v1/messages/{id}/recipients`：已落地；返回收件、送达和已读统计及有界脱敏预览。
- `GET /admin-api/v1/files/{id}`：文件详情。
- `GET /admin-api/v1/files/{id}/usages`：文件业务引用。
- `GET /admin-api/v1/notification-deliveries/{id}`：已落地；返回脱敏目标提示、状态与安全失败摘要。
- `GET /admin-api/v1/api-clients/{id}`：已落地；按当前租户和 `sys.api_client.read` 权限读取单个机器身份，只返回 Secret 元数据。
- `GET /admin-api/v1/block-rules`：已落地；按当前租户查询 `iam.block_rules`。
- `POST /admin-api/v1/block-rules`：已落地；创建访问控制规则并写不可变审计。
- `PATCH /admin-api/v1/block-rules/{id}`：已落地；修改或启停访问控制规则。
- `DELETE /admin-api/v1/block-rules/{id}`：已落地；撤销规则并记录影响。
- `GET /admin-api/v1/ops/health`：已落地；返回面向 UI 的依赖健康摘要，不暴露 Secret。
- `GET /admin-api/v1/ops/runtime-summary`：已落地；返回版本、构建、Worker、队列和依赖延迟摘要。

## P3

- `GET /admin-api/v1/me/oauth-accounts`：已落地；返回本人第三方绑定列表。
- `POST /admin-api/v1/me/oauth/{provider}/start`：已落地；使用单次 state 与 S256 PKCE 开始绑定。
- `POST /admin-api/v1/me/oauth/{provider}/callback`：已落地；校验并单次消费 state/code，完成账号绑定。
- `DELETE /admin-api/v1/me/oauth/{provider}`：已落地；按本人权限解绑第三方账号并写审计。
- `GET /admin-api/v1/me/mfa`：已落地；读取本人 MFA 状态，不返回 TOTP Secret。
- `POST /admin-api/v1/me/mfa/totp/enroll`：已落地；创建短期 TOTP enrollment，Secret 只显示一次。
- `POST /admin-api/v1/me/mfa/totp/verify`：已落地；验证 enrollment 并启用 TOTP。
- `DELETE /admin-api/v1/me/mfa/totp`：已落地；Step-up 后禁用 TOTP。
- `POST /admin-api/v1/me/mfa/recovery-codes/rotate`：已落地；轮换恢复码，明文只返回一次。

## 新权限码

- `sys.login_provider_config.read/create/update/delete/rotate_secret/preflight`：第三方登录全局配置生命周期；启用/停用复用 update。
- `app.login_provider_binding.read/update`：读取或原子更新 App 范围第三方登录绑定。

- `notify.message.read`：已落地；查看站内消息
- `notify.message.create`：已落地；创建站内消息
- `notify.message.update`：已落地；更新站内消息
- `notify.message.publish`：已落地；发布站内消息
- `notify.message.cancel`：已落地；取消站内消息
- `notify.recipient.read`：已落地；查看消息收件情况
- `iam.block_rule.read`：已落地；查看访问控制规则
- `iam.block_rule.create`：已落地；创建访问控制规则
- `iam.block_rule.update`：已落地；更新访问控制规则
- `iam.block_rule.delete`：已落地；撤销访问控制规则
- `ops.health.read`：已落地；查看服务状态
