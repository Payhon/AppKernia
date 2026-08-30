# ADR-0021：多厂商离线消息推送

- 状态：Accepted
- 日期：2026-08-29
- 范围：Backend、Admin、Mobile、数据库、构建与发布

## 背景

现有通知中心已有消息、收件人、投递、Push 设备和 River Worker，但缺少设备注册 API、多厂商发送实现、发布扇出以及配置治理。AppKernia 同时面向 App Store、Google Play、中国 Android 厂商渠道和 HarmonyOS NEXT，不能依赖包含全部 SDK 的单一 Android 安装包。

## 决策

1. 服务端直接对接 APNs HTTP/2、FCM HTTP v1、华为 Android、荣耀、小米、OPPO、vivo、魅族和 HarmonyOS Push Kit；不引入第三方聚合推送。
2. Android 固定为 `android_google` 与 `android_china` 两个互斥构建变体；不支持的设备仅保留站内消息。
3. `uni_modules/ak-push` 是移动业务代码唯一可调用的 Push Port。法定同意和用户 Push 总开关生效前，不申请权限、不初始化 SDK、不获取或上传 Token。
4. 厂商配置按 tenant、App、环境和 provider 隔离。服务端凭据仅写入、加密保存并支持版本轮换；客户端构建只接收厂商明确要求的公开参数。
5. `service_security` 与 `news_operations` 是稳定协议分类。运营推送默认关闭，并要求 `notify.operations.publish` 独立权限。HarmonyOS 和 vivo 的厂商场景值按两种业务分类分别配置，运行时按消息分类选择。
6. `sent` 仅表示厂商受理。首期只统计受理、失败、Token 失效和应用打开，不声称设备已展示。
7. 发送结果统一为 `accepted`、`invalid_token`、`throttled`、`transient`、`permanent`、`auth_config_error` 和 `unknown_after_write`。已写出但结果未知的请求不自动重放。

## 安全与失败语义

- 全局 Kill Switch、应用/环境配置、用户订阅和 OS 权限共同决定是否启用。
- Token 密文保存并使用 HMAC Hash 去重；日志、审计、指标和 Admin API 均不返回 Token 或凭据明文。
- 点击载荷仅允许 schema version、delivery/message ID、受控 `route_key` 和不透明资源 ID，不允许任意 URL、组件名、脚本或静默后台执行。
- 限流、明确 5xx 和写出前连接错误使用 `Retry-After`、指数退避和抖动；无效 Token 立即失效，鉴权错误使渠道进入故障状态。
- 用户退出登录、设备换号、关闭 Push 或检测到系统权限撤销时停用绑定。站内消息不依赖 Push 成功。

## 交付门禁

代码编译、Mock Provider 和接口契约不能替代厂商验收。生产启用前仍必须逐渠道完成账号权益、凭据、包名/Bundle ID、签名指纹、隐私联网审查和前台/后台/终止/离线恢复真机矩阵；未通过的 provider 保持禁用。
