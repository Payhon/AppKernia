# 页面开发规格

## 1. 全局页面状态

每个异步页面都必须显式实现：

```text
initial → loading → content | empty | error | offline | forbidden
                         ↘ refreshing / mutating
```

表单必须处理：键盘遮挡、重复提交、服务端字段错误、离开未保存提示、弱网取消和成功后返回。

## 2. 页面矩阵

| Route Key | pages.json path | Access | Phase | Feature Flag | Permissions | API |
|---|---|---:|---:|---|---|---|
| `bootstrap` | `pages/bootstrap/index` | public | P0 | — | — | POST /api/v1/auth/token/refresh<br>GET /api/v1/auth/context<br>GET /api/v1/me/preferences |
| `privacy.consent` | `pages/privacy/consent` | public | P0 | — | — | GET /api/v1/public/legal/{document_type} |
| `onboarding` | `pages/onboarding/index` | public | P1 | onboarding_enabled | — | — |
| `auth.login.password` | `pages/auth/login/index` | guest | P0 | — | — | POST /api/v1/auth/login/password<br>GET /api/v1/auth/oauth/providers |
| `auth.login.otp` | `pages/auth/login-otp/index` | guest | P0 | otp_login_enabled | — | POST /api/v1/auth/mobile/send-code<br>POST /api/v1/auth/login/otp |
| `auth.register` | `pages/auth/register/index` | guest | P0 | registration_enabled | — | POST /api/v1/auth/register |
| `auth.forgot-password` | `pages/auth/forgot-password/index` | guest | P0 | — | — | POST /api/v1/auth/password/forgot |
| `auth.reset-password` | `pages/auth/reset-password/index` | guest | P0 | — | — | POST /api/v1/auth/password/reset |
| `auth.verify-contact` | `pages/auth/verify-contact/index` | mixed | P0 | — | — | POST /api/v1/auth/email/send-code<br>POST /api/v1/auth/email/verify<br>POST /api/v1/auth/mobile/send-code<br>POST /api/v1/auth/mobile/verify |
| `auth.mfa-challenge` | `pages/auth/mfa-challenge/index` | challenge | P1 | mfa_enabled | — | POST /api/v1/auth/mfa/verify |
| `auth.oauth-callback` | `pages/auth/oauth-callback/index` | mixed | P2 | oauth_login_enabled | — | POST /api/v1/auth/oauth/{provider}/callback |
| `home` | `pages/home/index` | authenticated | P0 | — | — | GET /api/v1/auth/context<br>GET /api/v1/me/notifications/unread-count |
| `notifications.list` | `pages/notifications/index` | authenticated | P1 | — | notify.message.read_self | GET /api/v1/me/notifications<br>POST /api/v1/me/notifications/read-all |
| `notifications.detail` | `pages/notifications/detail` | authenticated | P1 | — | notify.message.read_self<br>notify.message.mark_read_self | GET /api/v1/me/notifications/{message_id}<br>PATCH /api/v1/me/notifications/{message_id}/read |
| `profile.index` | `pages/profile/index` | authenticated | P0 | — | iam.user.read_self | GET /api/v1/me |
| `profile.basic` | `pages/profile/basic/index` | authenticated | P1 | — | iam.user.read_self | GET /api/v1/me |
| `profile.edit` | `pages/profile/edit/index` | authenticated | P1 | — | iam.user.update_self | GET /api/v1/me<br>PATCH /api/v1/me<br>POST /api/v1/me/avatar/upload-session |
| `profile.security` | `pages/profile/security/index` | authenticated | P1 | — | iam.user.read_self | GET /api/v1/me/login-events<br>GET /api/v1/me/security-events |
| `profile.change-password` | `pages/profile/change-password/index` | authenticated | P1 | — | iam.user.update_self | POST /api/v1/auth/password/change |
| `profile.sessions` | `pages/profile/sessions/index` | authenticated | P1 | — | iam.session.read_self<br>iam.session.revoke_self | GET /api/v1/me/sessions<br>DELETE /api/v1/me/sessions/{session_id}<br>POST /api/v1/auth/logout-all |
| `profile.devices` | `pages/profile/devices/index` | authenticated | P1 | — | iam.device.read_self<br>iam.device.revoke_self | GET /api/v1/me/devices<br>DELETE /api/v1/me/devices/{device_id} |
| `profile.mfa` | `pages/profile/mfa/index` | authenticated | P1 | mfa_enabled | iam.mfa.manage_self | POST /api/v1/auth/step-up<br>POST /api/v1/me/mfa/totp/setup<br>POST /api/v1/me/mfa/totp/confirm<br>DELETE /api/v1/me/mfa/totp<br>POST /api/v1/me/mfa/recovery-codes/rotate |
| `profile.connections` | `pages/profile/connections/index` | authenticated | P2 | oauth_login_enabled | iam.oauth.manage_self | GET /api/v1/me/oauth-accounts<br>DELETE /api/v1/me/oauth-accounts/{account_id}<br>POST /api/v1/auth/oauth/{provider}/authorize |
| `settings.index` | `pages/settings/index` | authenticated | P1 | — | iam.preference.manage_self | GET /api/v1/me/preferences |
| `settings.language` | `pages/settings/language/index` | authenticated | P1 | — | iam.preference.manage_self | GET /api/v1/me/preferences<br>PATCH /api/v1/me/preferences |
| `settings.theme` | `pages/settings/theme/index` | authenticated | P2 | dark_mode | iam.preference.manage_self | GET /api/v1/me/preferences<br>PATCH /api/v1/me/preferences |
| `settings.notifications` | `pages/settings/notifications/index` | authenticated | P2 | push_notifications | notify.preference.manage_self | GET /api/v1/me/notification-preferences<br>PATCH /api/v1/me/notification-preferences<br>POST /api/v1/me/push-devices<br>DELETE /api/v1/me/push-devices/{device_id} |
| `tenant.switch` | `pages/tenant/switch/index` | authenticated | P2 | multi_tenant | iam.tenant.switch_self | GET /api/v1/auth/context<br>POST /api/v1/auth/switch-tenant |
| `legal.privacy` | `pages/legal/privacy/index` | public | P0 | — | — | GET /api/v1/public/legal/{document_type} |
| `legal.terms` | `pages/legal/terms/index` | public | P0 | — | — | GET /api/v1/public/legal/{document_type} |
| `about` | `pages/about/index` | public | P1 | — | — | GET /api/v1/public/app-version |
| `account.deletion` | `pages/account/deletion/index` | authenticated | P2 | account_deletion | iam.user.request_deletion_self | POST /api/v1/auth/step-up<br>POST /api/v1/me/account-deletion<br>GET /api/v1/me/account-deletion<br>DELETE /api/v1/me/account-deletion |
| `error.forbidden` | `pages/error/forbidden/index` | mixed | P0 | — | — | — |
| `error.offline` | `pages/error/offline/index` | mixed | P0 | — | — | — |
| `dev.components` | `pages/dev/components/index` | development | P0 | dev_component_gallery | — | — |

## 3. 认证页面关键约束

- 登录错误使用统一文案，不能暴露账号是否存在。
- OTP 倒计时以服务端冷却信息为准，App 重启后不能绕过。
- 密码输入默认隐藏，不记录剪贴板、日志或埋点。
- Refresh 失败后只执行一次统一 Session 清理。
- MFA Challenge Token 与正式 Session Token 分离。

## 4. 个人资料

- 普通 PATCH 不能直接修改邮箱/手机号验证状态。
- 头像必须先创建上传会话，成功后仅提交后端认可的 file id。
- 使用 optimistic concurrency；冲突时提示重新加载，而不是覆盖他人/其他设备修改。

## 5. 安全中心

- 会话和设备操作显示当前设备标记、最后活动时间和脱敏位置。
- 撤销当前会话后立即清空本地 Secret 并 reLaunch 登录页。
- MFA Secret 和恢复码只展示一次，严禁进入截图基线、日志和测试 Fixture。
- 解绑最后一种可用登录方式前必须阻止或要求先设置另一种凭据。

## 6. 消息

- 使用 cursor/infinite list，不使用桌面分页器。
- 进入详情后由服务端确认已读，失败时不伪造最终状态。
- Push/WebSocket 只刷新未读数和 Query，不作为消息内容事实源。

## 7. 账号注销

必须显示：影响范围、冷静期、生效时间、可撤销条件、数据保留规则和 Step-up。注销成功后撤销 Session、Push Token 并清理本地用户数据。

## 8. 双语页面验收

- 所有 Route、Tab、导航标题、表单标签、按钮、校验、Toast、空状态和错误状态必须同时存在 `zh-CN`、`en-US` key。
- `settings.language` 展示“简体中文 / English”，切换后无需重启，立即更新当前页面标题与 TabBar，并在登录状态写入服务端偏好。
- 登录、注册、个人中心、消息、语言设置、法律文档和错误页必须在 Android/iOS/Harmony 上做英文长文本和截断检查。
- 法律 API 请求携带 locale，用户同意记录必须保存 document version、locale 和 content hash。
- 服务端 error code 在两种语言下保持不变；客户端优先本地翻译，后端 message 仅作回退。
