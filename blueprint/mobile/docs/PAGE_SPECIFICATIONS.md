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
| `auth.login.password` | `pages/auth/login/index` | guest | P0 | — | — | POST /api/v1/auth/login/password<br>POST /api/v1/auth/email/send-code<br>POST /api/v1/auth/sms-captcha<br>POST /api/v1/auth/mobile/send-code<br>POST /api/v1/auth/login/otp<br>GET /api/v1/auth/oauth/providers<br>POST /api/v1/auth/oauth/{provider}/authorize<br>POST /api/v1/auth/oauth/{provider}/callback |
| `auth.login.otp` | `pages/auth/login-otp/index` | guest | P0 | otp_login_enabled | — | POST /api/v1/auth/sms-captcha<br>POST /api/v1/auth/mobile/send-code<br>POST /api/v1/auth/login/otp |
| `auth.register` | `pages/auth/register/index` | guest | P0 | registration_enabled | — | POST /api/v1/auth/sms-captcha<br>POST /api/v1/auth/registration/send-code<br>POST /api/v1/auth/register |
| `auth.forgot-password` | `pages/auth/forgot-password/index` | guest | P0 | — | — | POST /api/v1/auth/sms-captcha<br>POST /api/v1/auth/password/forgot |
| `auth.reset-password` | `pages/auth/reset-password/index` | guest | P0 | — | — | POST /api/v1/auth/password/reset |
| `auth.verify-contact` | `pages/auth/verify-contact/index` | mixed | P0 | — | — | POST /api/v1/auth/email/send-code<br>POST /api/v1/auth/email/verify<br>POST /api/v1/auth/mobile/send-code<br>POST /api/v1/auth/mobile/verify |
| `auth.mfa-challenge` | `pages/auth/mfa-challenge/index` | challenge | P1 | mfa_enabled | — | POST /api/v1/auth/mfa/verify |
| `auth.oauth-callback` | `pages/auth/oauth-callback/index` | public | P2 | — | — | POST /api/v1/auth/oauth/{provider}/callback |
| `home` | `pages/home/index` | public | P0 | — | — | GET /api/v1/public/config<br>GET /api/v1/public/content/home<br>GET /api/v1/me/notifications/unread-count（登录后） |
| `scanner.webview` | `pages/scanner/webview/index` | public | P1 | — | — | — |
| `notifications.list` | `pages/notifications/index` | authenticated | P1 | — | notify.message.read_self | GET /api/v1/me/notifications<br>POST /api/v1/me/notifications/read-all |
| `notifications.detail` | `pages/notifications/detail` | authenticated | P1 | — | notify.message.read_self<br>notify.message.mark_read_self | GET /api/v1/me/notifications/{message_id}<br>PATCH /api/v1/me/notifications/{message_id}/read |
| `profile.index` | `pages/profile/index` | authenticated | P0 | — | iam.user.read_self | GET /api/v1/me |
| `profile.basic` | `pages/profile/basic/index` | authenticated | P1 | — | iam.user.read_self | GET /api/v1/me |
| `profile.edit` | `pages/profile/edit/index` | authenticated | P1 | — | iam.user.update_self, storage.file.upload_self | GET /api/v1/me<br>PATCH /api/v1/me<br>POST /api/v1/me/avatar/upload-session<br>POST /api/v1/me/avatar/upload-sessions/{id}/content<br>GET /api/v1/me/avatar/content |
| `profile.security` | `pages/profile/security/index` | authenticated | P1 | — | iam.user.read_self | GET /api/v1/me/login-events<br>GET /api/v1/me/security-events |
| `profile.change-password` | `pages/profile/change-password/index` | authenticated | P1 | — | iam.user.update_self | POST /api/v1/auth/password/change |
| `profile.sessions` | `pages/profile/sessions/index` | authenticated | P1 | — | iam.session.read_self<br>iam.session.revoke_self | GET /api/v1/me/sessions<br>DELETE /api/v1/me/sessions/{session_id}<br>POST /api/v1/auth/logout-all |
| `profile.devices` | `pages/profile/devices/index` | authenticated | P1 | — | iam.device.read_self<br>iam.device.revoke_self | GET /api/v1/me/devices<br>DELETE /api/v1/me/devices/{device_id} |
| `profile.mfa` | `pages/profile/mfa/index` | authenticated | P1 | mfa_enabled | iam.mfa.manage_self | POST /api/v1/auth/step-up<br>POST /api/v1/me/mfa/totp/setup<br>POST /api/v1/me/mfa/totp/confirm<br>DELETE /api/v1/me/mfa/totp<br>POST /api/v1/me/mfa/recovery-codes/rotate |
| `profile.connections` | `pages/profile/connections/index` | authenticated | P2 | — | — | GET /api/v1/auth/oauth/providers<br>POST /api/v1/auth/oauth/{provider}/authorize<br>POST /api/v1/auth/oauth/{provider}/callback<br>GET /api/v1/me/login-methods<br>POST /api/v1/auth/sms-captcha<br>POST /api/v1/me/login-identifiers/{identifier_type}/challenge<br>PUT /api/v1/me/login-identifiers/{identifier_type}<br>DELETE /api/v1/me/login-identifiers/{identifier_type}<br>POST /api/v1/auth/step-up/verification-code<br>POST /api/v1/auth/step-up<br>GET /api/v1/me/oauth-accounts<br>DELETE /api/v1/me/oauth-accounts/{account_id} |
| `settings.index` | `pages/settings/index` | authenticated | P1 | — | iam.preference.manage_self | GET /api/v1/me/preferences |
| `settings.language` | `pages/settings/language/index` | authenticated | P1 | — | iam.preference.manage_self | GET /api/v1/me/preferences<br>PATCH /api/v1/me/preferences |
| `settings.theme` | `pages/settings/theme/index` | authenticated | P2 | dark_mode | iam.preference.manage_self | GET /api/v1/me/preferences<br>PATCH /api/v1/me/preferences |
| `settings.notifications` | `pages/settings/notifications/index` | authenticated | P2 | push_notifications | notify.preference.manage_self | GET /api/v1/me/notification-preferences<br>PATCH /api/v1/me/notification-preferences<br>POST /api/v1/me/push-devices<br>DELETE /api/v1/me/push-devices/{device_id} |
| `tenant.switch` | `pages/tenant/switch/index` | authenticated | P2 | multi_tenant | iam.tenant.switch_self | GET /api/v1/auth/context<br>POST /api/v1/auth/switch-tenant |
| `legal.privacy` | `pages/legal/privacy/index` | public | P0 | — | — | GET /api/v1/public/legal/{document_type} |
| `legal.terms` | `pages/legal/terms/index` | public | P0 | — | — | GET /api/v1/public/legal/{document_type} |
| `about` | `pages/about/index` | public | P1 | — | — | GET /api/v1/public/app-version<br>GET /api/v1/public/pages/{slug} |
| `upgrade.dialog` | `uni_modules/ak-upgrade/pages/upgrade-dialog` | public | P1 | — | — | GET /api/v1/public/app-version<br>GET /api/v1/public/app-version/download/{release_id}/{file_id} |
| `account.deletion` | `pages/profile/account-deletion/index` | authenticated | P2 | account_deletion | iam.user.delete_self | POST /api/v1/me/account-deletion/verification-code<br>POST /api/v1/me/account-deletion/confirm |
| `error.forbidden` | `pages/error/forbidden/index` | mixed | P0 | — | — | — |
| `error.offline` | `pages/error/offline/index` | mixed | P0 | — | — | — |
| `dev.components` | `pages/dev/components/index` | development | P0 | dev_component_gallery | — | — |

### 首页扫码与受控 WebView

- 首页标题栏消息按钮右侧使用 44 × 44 `ak-icon-button` 触发扫码，游客可用；扫码 single-flight，业务页面不得直接调用 `uni.scanCode`。
- `ak-scanner` 固定只从相机读取二维码和条形码。协调器依次执行代码注册的业务处理器、可信网页处理器和结果展示兜底；取消不显示错误。
- 仅成功刷新 `/api/v1/public/config` 且绝对 HTTPS 地址命中规范化白名单时，才通过一次性内存 token 进入静态 WebView 页。初始加载和每次跳转都复验，越界后关闭并显示原始扫码结果。
- 普通文本、条码、未命中域名、配置缺失或刷新失败使用 `ak-bottom-sheet` 展示；只有用户点击复制后才写剪贴板。扫码内容不上传、不持久化、不写日志。

## 3. 认证页面关键约束

- 登录错误使用统一文案，不能暴露账号是否存在。
- OTP 倒计时以服务端冷却信息为准，App 重启后不能绕过。
- 所有短信发送与重发先完成一次新的交互式验证码；邮箱 OTP 不进入该 Modal。证明通过前不启动 OTP 倒计时。
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

必须明确显示当前 App 删除范围、其他 App/管理端不受影响、不可恢复警告、清除内容与依法匿名留存规则。使用当前账号已验证邮箱的 6 位验证码，验证码完整且已勾选确认后才允许二次确认；成功后不得再调用已失效会话接口，只执行本地 Push 注销、清除安全 Session、认证上下文和敏感缓存并 reLaunch 登录页。

## 8. 双语页面验收

- 所有 Route、Tab、导航标题、表单标签、按钮、校验、Toast、空状态和错误状态必须同时存在 `zh-CN`、`en-US` key。
- `settings.language` 展示“简体中文 / English”，切换后无需重启，立即更新当前页面标题与 TabBar，并在登录状态写入服务端偏好。
- 登录、注册、个人中心、消息、语言设置、法律文档和错误页必须在 Android/iOS/Harmony 上做英文长文本和截断检查。
- 法律 API 请求携带 locale，用户同意记录必须保存 document version、locale 和 content hash。
- 服务端 error code 在两种语言下保持不变；客户端优先本地翻译，后端 message 仅作回退。

## 帮助与关于、问题反馈

- `help`：沿用登录规则；FAQ / 联系支持打开已发布单页，关于页独立加载正文和版本策略；底部从安装包读取当前版本，断网不隐藏。未发布的 CMS 内容显示“内容暂未配置”，不回填静态客服信息。
- `feedback.create`：问题描述 1–2000 字，联系方式选填、最多 200 字，最多 3 张 ≤5 MiB 截图（后台更严格策略优先）；相册由用户操作触发。`ak-text-area` 包装原生 textarea，输入区随键盘调整；上传进度、失败重试、移除/取消。仅使用内存草稿，提交失败保持内容，相同请求使用稳定幂等键。
- `feedback`：当前用户当前 App 的分页列表；返回页面重新加载，空态、离线态、错误重试和下一页失败均可区分。
- `feedback.detail`：原始内容、截图、处理状态、提交时间及追加式后台回复；截图通过带鉴权的临时下载读取，卸载时取消请求并删除临时文件。
- 状态 `pending/processing/resolved` 与平台值属于固定协议枚举。所有标签使用 AkI18n，后台文本以已发布语言为准，受限块/Markdown 渲染不执行 HTML、脚本或远程组件。
- 验收区分模拟器、源码编译、未签名包与真机；设计规范见 `apps/ak-mobile/design-system/pages/help-feedback.md`，实际证据以交付报告为准。

### 资讯公开分享地址

详情响应 `share_url` 由可信 Server origin 生成并带语言参数。分享优先使用该值；对旧 Server 未返回字段时回退原 `/s/{slug}?app_id=` 生成规则。页面协议阅读不产生同意记录；原生同意流程保持不变。H5 下载平台检测不进入 Mobile 或 Server UA 逻辑。
