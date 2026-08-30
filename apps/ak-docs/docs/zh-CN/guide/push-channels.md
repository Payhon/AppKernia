---
title: 推送渠道配置
description: 在 Admin 中申请、配置、预检并安全启用 APNs、FCM、国内 Android 厂商和 HarmonyOS Push Kit。
---

# 推送渠道配置

Admin 的“通知中心 → 推送渠道”按应用和环境管理离线推送。页面不会要求管理员直接输入设备 Token，也不会回显服务器凭据；测试推送只能选择当前 App 已注册的安全设备摘要。

## 支持范围

| 目标构建         | 唯一客户端通道                                             | 服务端渠道                |
| ---------------- | ---------------------------------------------------------- | ------------------------- |
| iOS              | Apple 系统通知能力                                         | APNs HTTP/2 + ES256 Token |
| `android_google` | FCM                                                        | FCM HTTP v1 + OAuth 2.0   |
| `android_china`  | 华为、荣耀、小米、OPPO、vivo、魅族中当前设备可用的唯一通道 | 各厂商官方 REST API       |
| HarmonyOS NEXT   | HarmonyOS Push Kit                                         | Push Kit 服务端 API       |

Google 和 China Android 变体互斥。Google 包不能包含国内厂商 SDK，China 包不能依赖 Firebase/GMS；不支持任何已编译通道的设备只使用站内消息。

## 页面操作顺序

1. 在页面顶部选择 App 和 `development`、`staging` 或 `production` 环境。
2. 点击标题右侧的问号图标，按 APNs、FCM、华为 Android、荣耀、小米、OPPO、vivo、魅族、Harmony 的 Tab 查看申请步骤、字段来源、风险提示和官方资料。
3. 创建草稿并填写强类型公开字段。秘密字段通过独立的只写入口保存。
4. 执行预检，修复包名、Bundle ID、环境、签名身份或凭据解析错误。
5. 预检状态为 ready 后激活渠道；需要更换凭据时使用“轮换”，不要把旧秘密复制到普通配置字段。
6. 从已注册设备列表选择一台匹配 Provider 的设备发送测试通知，再查看投递结果。

## 各渠道需要准备的字段

| 渠道         | 公开或身份字段                                                 | 只写秘密             |
| ------------ | -------------------------------------------------------------- | -------------------- |
| APNs         | Team ID、Key ID、Bundle ID、Sandbox/Production 环境            | `.p8` 私钥           |
| FCM          | Firebase Project ID、Android Package Name                      | Service Account JSON |
| 华为 Android | Client ID、Package Name、通知 Channel ID                       | Client Secret        |
| 荣耀         | App ID、Client ID、Package Name、通知 Channel ID               | Client Secret        |
| 小米         | App ID、App Key、Package Name、Region、通知 Channel ID         | App Secret           |
| OPPO         | App Key、Package Name、通知 Channel ID                         | Master Secret        |
| vivo         | App ID、App Key、Package Name、服务消息分类、运营消息分类      | App Secret           |
| 魅族         | App ID、App Key、Package Name                                  | App Secret           |
| Harmony      | Project ID、Client ID、Bundle Name、服务消息分类、运营消息分类 | Service Account JSON |

问号指引中的链接只指向 Apple、Firebase、华为、荣耀、小米、OPPO、vivo 和魅族官方域名。厂商控制台可能调整入口；正式申请时仍应以页面内当前官方链接和对应控制台显示为准。

## 状态和安全语义

- `draft`：配置可以编辑，但不能用于发送。
- 预检通过：服务器确认必填字段、密钥格式和已知身份关系可解析；它不代表厂商账号已经审批完成。
- `active`：在全局 Push 开关、用户订阅、设备和消息有效期也满足时可以发送。
- `faulted`：连续鉴权或配置错误使渠道进入故障状态，修复并重新预检后才能恢复。
- `disabled`：停止创建该渠道的新厂商投递；站内消息不受影响。

凭据按 tenant、App、环境和 Provider 隔离，并带密钥版本与指纹。日志、截图、Fixture、指标和 Admin 响应都不应包含私钥、Service Account、Client Secret、Master Secret 或设备 Token。

## Android 构建门禁

使用 `AK_ANDROID_PUSH_VARIANT=google|china` 选择构建变体。生产构建还必须注入与 App 身份匹配的公开客户端配置，并通过最终 APK/AAB 依赖与 marker 检查。生成配置是被忽略的临时产物，服务器 Secret 永远不能进入安装包。

所有 SDK 都必须在用户完成隐私同意并主动开启 Push 后才初始化。Firebase 自动初始化需要关闭；不能证明延迟初始化和首次联网行为合规的厂商渠道应保持禁用。

## 上线前检查

- 厂商账号、推送权益、消息分类和频控已经审批。
- Package Name、Bundle ID、Bundle Name、签名指纹和发布环境与最终安装包一致。
- SDK 版本、校验和、许可证和隐私清单已经固定并评审。
- 对应渠道已完成前台、后台、被终止、离线恢复、Token 更新、重装、账号切换和双语通知点击真机矩阵。

渠道配置完成后，使用[消息运营工作台](./notification-operations)观察任务与失败，并阅读[消息推送架构](../concepts/notification-architecture)理解异步边界。
