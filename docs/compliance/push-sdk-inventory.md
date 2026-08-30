# Push SDK、许可证与隐私门禁清单

本清单记录构建边界，不保存账号文件、证书、Token 或 Secret。Android 实际依赖摘要由 `apps/ak-mobile/scripts/configure-push-variant.py` 生成到被忽略的 `.push-build/android-push-summary.json`。

| 渠道 | 客户端集成 | 版本固定方式 | 自动初始化/联网门禁 | 上线状态 |
|---|---|---|---|---|
| APNs | iOS 系统 UserNotifications | 随 iOS SDK | 法定同意和用户开启后手动申请与注册 | 待签名设备、Sandbox/Production 验收 |
| FCM | Firebase Messaging | `push-sdk-lock.json` 固定 BOM 与 Gradle 插件 | Messaging/Analytics auto-init 显式关闭 | 待 Google 配置、依赖树与 GMS 真机验收 |
| 华为 Android | 官方 Push Kit | 生产构建传入已批准的精确 Maven 坐标 | `push_kit_auto_init_enabled=false`，手动取 Token | 待账号、AAR/Maven 版本、许可证及真机验收 |
| 荣耀 | 官方 Push SDK | lock 文件固定 SDK 与插件版本 | 仅 China 变体，用户开启后取 Token | 待账号、隐私联网与真机验收 |
| 小米 | 官方 Mi Push | 生产构建传入已批准的精确依赖 | 仅 China 变体，用户开启后注册 | 待账号、版本/许可证及真机验收 |
| OPPO | 官方 Push SDK | 生产构建传入已批准的精确依赖 | 仅 China 变体；客户端 AppSecret 与服务端 MasterSecret 分离 | 待账号、版本/许可证及真机验收 |
| vivo | 官方 Push SDK | 生产构建传入已批准的精确依赖 | 仅 China 变体，用户开启后初始化 | 待账号、消息分类权益及真机验收 |
| 魅族 | 官方 Push SDK | 生产构建传入已批准的精确依赖 | 仅 China 变体；注册后读取官方 PushId | 待账号、版本/许可证及真机验收 |
| HarmonyOS NEXT | 系统 `@kit.PushKit` | 随 HarmonyOS SDK | 法定同意和用户开启后调用 `getToken` | 待签名、场景分类权益及真机验收 |

## 发布要求

- `android_google` 产物不得包含任一国内厂商类；`android_china` 不得包含 Firebase/GMS。发布流水线必须对最终 APK/AAB 调用 `verify-push-variant.py`。
- 所有外部 SDK 必须记录来源 URL、精确版本、文件 SHA-256、许可证文本、隐私政策、首次联网行为和数据字段。任一项缺失不得把该渠道标记为可生产。
- 配置文件和生成 Manifest 是临时产物并已加入 `.gitignore`；服务端私钥、Service Account、Client Secret 和 Master Secret 永不进入安装包。
