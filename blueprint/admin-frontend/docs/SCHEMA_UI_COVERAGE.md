# PostgreSQL Schema 与后台 UI 覆盖说明

本文件对照 AK 后端核心 Schema，明确每张表是否需要后台页面。原则是：**功能必须可追踪，但基础设施表不应为了“每表一个 CRUD”而暴露不安全的管理入口。**

## 覆盖结论

- 核心表总数：95
- 页面直接覆盖：85
- 脱敏聚合覆盖：3
- 仅后端或非 Admin 自助事实源：7

`spec/schema-ui-coverage.json` 是机器可读事实源，并由 `scripts/validate_blueprint_specs.py` 强制校验全量覆盖。

## 无直接 CRUD 页面的表

| 表 | 分类 | 前端处理 |
|---|---|---|
| `iam.login_captcha_challenges` | `indirect_aggregate` | 登录页仅消费一次性图形挑战，不读取答案 Hash、盐或挑战记录。 关联：`auth.login`。 |
| `iam.login_failure_states` | `indirect_aggregate` | 登录页只按稳定错误码展示渐进式验证码，不暴露失败状态原始记录。 关联：`auth.login`。 |
| `jobs.inbox_events` | `backend_only` | 消息消费幂等与去重基础设施；仅通过运行指标/告警间接观测，不提供逐行管理页面。 关联：无直接页面。 |
| `notify.push_devices` | `indirect_aggregate` | 仅展示推送设备数量、平台、最近活跃与投递状态；Push Token 不进入通用列表或前端日志，也不提供直接 CRUD。 关联：`system.users.accounts`、`system.notifications.deliveries`、`system.notifications.push-channels`。 |
| `sys.idempotency_keys` | `backend_only` | HTTP 幂等基础设施；记录由服务端生命周期自动清理，不允许后台人工编辑。 关联：无直接页面。 |
| `content.article_bookmarks` | `backend_only` | 移动端书签由用户自助维护，不在 Admin 内容管理页直接编辑。 关联：无直接页面。 |
| `iam.user_preferences` | `backend_only` | 移动端用户偏好由当前用户接口维护，不在 Admin 页面直接编辑。 关联：无直接页面。 |
| `app.legal_consents` | `backend_only` | 同意记录是 App 用户受保护的事实源，Admin 不提供编辑或删除入口。 关联：无直接页面。 |
| `iam.app_oauth_accounts` | `backend_only` | App 级外部身份只由 Mobile 本人登录与绑定流程维护；Admin 不提供 Subject、Union ID 或资料快照 CRUD。 关联：无直接 Admin 页面。 |
| `iam.oauth_authorization_flows` | `backend_only` | State、Nonce、PKCE 与一次性票据为短期安全状态，只由服务端消费与清理。 关联：无直接页面。 |

## 设计约束

- `notify.push_devices` 绝不向浏览器返回原始 Push Token；只返回脱敏设备元数据和统计。
- `jobs.inbox_events` 不提供重放/删除按钮；事件补偿应通过受控运维命令或专用 Runbook。
- `sys.idempotency_keys` 不提供人工修改；清理策略由后端任务和 TTL 管理。
- `iam.app_oauth_accounts` 和 `app.user_login_identifiers` 按 App 隔离；Admin 不暴露外部 Subject、完整邮箱或手机号作为跨 App 合并依据。
- `iam.oauth_authorization_flows` 的 State、Nonce、PKCE 和票据不得进入 Admin 查询、日志或导出。
- 后续新增 Migration 表时，CI 必须要求同步更新本覆盖清单，否则校验失败。
