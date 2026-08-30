# AK Mobile 所需后端增量

Backend Blueprint 已有 **26** 个 App API 基线；移动端完整页面建议增加 **28** 个端点。机器清单：

- `app-api-baseline.json`
- `app-api-delta.json`
- `app-permissions.delta.json`

## 必须优先补充（P0/P1）

- `GET /api/v1/auth/context`
- MFA Challenge / Step-up
- 用户偏好
- 未读计数和消息详情
- 自助登录/安全事件
- 法律文档和三平台版本检查
- `/api/v1/public/config` 返回可选的 `scanner.webview.enabled` 与规范化域名白名单；缺失时客户端关闭 WebView。

## P2 增量

- OAuth PKCE Provider。
- Push Device 与通知偏好。
- 多租户切换。
- 账号注销冷静期。

## 数据表映射

尽量复用 Backend Core Schema：

- Context/租户：`iam.users`、`iam.tenant_members`、角色权限关系。
- OAuth：`iam.oauth_accounts`。
- MFA：`iam.mfa_factors`、`iam.mfa_recovery_codes`。
- Push：`notify.push_devices`。
- 消息：`notify.messages`、`notify.recipients`。
- 安全记录：`audit.login_events`、`audit.security_events`。
- 偏好：优先增加受约束的 user preference 表或明确 JSONB Schema，不把任意配置塞入客户端。
- 扫码配置：`app.application_scanner_configs` 按 `(tenant_id,app_id)` 隔离，独立乐观锁；只保存规范化域名，不保存扫码内容。
- 注销：建议新增不可变状态流转表，不直接软删用户。

## 安全要求

- Refresh Rotation 与 Replay Detection 沿用 Backend Blueprint。
- Context 只返回当前 Session 可见的权限与租户。
- Step-up Token 短生命周期、单用途或按 action audience 限制。
- Push Token 服务端加密保存，日志只显示 Hash/后四位。
- App Version 只返回可信商店链接，不返回可执行脚本或任意下载地址。
- 扫码域名只允许 ASCII/Punycode 精确域名与受控通配符；拒绝协议、路径、凭据、非 443 端口、IP、localhost 和公共后缀通配符。服务端不得下发可执行处理器。

## 多语言后端增量

- Public Config/Auth Context 返回：`locale`、`default_locale=zh-CN`、`supported_locales=[zh-CN,en-US]`。
- 用户偏好 API 接受并返回规范 locale；拒绝未知值，平台别名由统一 LocaleResolver 规范化。
- 所有 App API 支持 `Accept-Language` 和 `Content-Language`；错误返回稳定 code 与可选 `message_key`。
- 法律文档、版本提示、字典、通知模板和站内消息生成流程按 locale 选择并回退 `zh-CN`。
- 用户同意记录保存实际文档 locale/version/hash；通知 Job 保存 template code、locale 和变量。
- 相关 OpenAPI、SQL、Seed、Backend Catalog、Mobile Catalog 和契约测试必须同一变更提交。
