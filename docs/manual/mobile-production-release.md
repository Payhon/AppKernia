# AppKernia 多端正式版 App 编译与发布

本文说明如何在 macOS 或 Windows 上构建 Android、iOS、HarmonyOS 正式签名产物。正式版与自定义调试基座是两条独立流水线：正式版必须使用组织自有发布签名、release 配置和商店验收，不得使用公共调试证书、模拟器 App 或 unsigned HAP。

## 1. 发布前检查

发布负责人必须先确认：

1. `apps/ak-mobile/manifest.json` 中 `versionName`、`versionCode`、AppID 和三端原生标识正确；
2. API Base URL 指向 HTTPS 生产环境，且不包含开发 Mock/调试后门；
3. `zh-CN`、`en-US` 语言包和随包隐私/用户协议快照是当前发布版本；
4. Android/iOS/HarmonyOS 发布证书、Profile 有效且属于 AppKernia；
5. 执行静态门禁：

```bash
apps/ak-mobile/scripts/check-project.sh
python3 blueprint/mobile/scripts/validate_blueprint_specs.py
python3 blueprint/scripts/validate_i18n_contract.py
```

## 2. 自动化命令

仓库根目录：

```bash
# 仅展示三端 release 命令；缺少签名值会以 <missing> 显示但不会打包
pnpm build:mobile:release:dry-run

# 在全部签名已配置后连续构建三端
pnpm build:mobile:release

# 单平台
pnpm build:mobile:release:android
pnpm build:mobile:release:ios
pnpm build:mobile:release:harmony:prepare
pnpm build:mobile:release:harmony

# 输出最新正式产物路径和 SHA-256
pnpm build:mobile:release:verify
```

严格预检同时检查 Android/iOS 签名输入和已生成 Harmony 工程的 Signing Config，因此完整三端发布应按以下顺序执行：

```bash
pnpm build:mobile:release:dry-run
pnpm build:mobile:release:harmony:prepare
# 在 DevEco Studio 中配置 release Signing Config
pnpm build:mobile:release:preflight
pnpm build:mobile:release
pnpm build:mobile:release:verify
```

`release:all` 不会自动创建、下载或替换证书，也不会上传应用商店。缺少任何签名材料时严格失败。

## 3. Android 正式版

必须使用自有 Android Keystore：

```text
AK_ANDROID_CERT_FILE       Keystore/JKS 文件
AK_ANDROID_CERT_ALIAS      Key Alias
AK_ANDROID_CERT_PASSWORD   Key 密码
AK_ANDROID_STORE_PASSWORD  Keystore 密码
AK_ANDROID_CHANNELS        可选；例如 google、huawei，多个用逗号分隔
```

macOS：

```bash
export AK_ANDROID_CERT_FILE=/secure/appkernia-release.jks
export AK_ANDROID_CERT_ALIAS=appkernia
read -s AK_ANDROID_CERT_PASSWORD && export AK_ANDROID_CERT_PASSWORD
read -s AK_ANDROID_STORE_PASSWORD && export AK_ANDROID_STORE_PASSWORD
export AK_ANDROID_CHANNELS=google
pnpm build:mobile:release:android
```

Windows PowerShell：

```powershell
$env:AK_ANDROID_CERT_FILE = 'D:\secure\appkernia-release.jks'
$env:AK_ANDROID_CERT_ALIAS = 'appkernia'
$env:AK_ANDROID_CERT_PASSWORD = Read-Host 'key password' -MaskInput
$env:AK_ANDROID_STORE_PASSWORD = Read-Host 'store password' -MaskInput
$env:AK_ANDROID_CHANNELS = 'google'
pnpm build:mobile:release:android
```

脚本固定 `androidpacktype=0`，不会使用自定义基座的公共调试证书。`AK_ANDROID_CHANNELS` 是否产出 APK/AAB 由当前 HBuilderX/DCloud 渠道规则决定；Google Play 交付还必须使用 bundletool 检查 AAB、64 位 ABI 和 16 KB page alignment。

上传前至少检查：

- 包名 `com.appkernia.mobile`；
- 版本号/版本码单调递增；
- 发布证书 SHA-256 与商店登记一致；
- 64 位 native ELF 的 LOAD alignment；
- 冷启动、升级安装、回滚策略和强制升级提示。

## 4. iOS 正式版

准备 Apple Distribution p12 和 App Store Provisioning Profile：

```text
AK_IOS_DIST_PROFILE        App Store Provisioning Profile
AK_IOS_DIST_CERT_FILE      Apple Distribution p12
AK_IOS_DIST_CERT_PASSWORD  p12 密码
AK_IOS_SUPPORTED_DEVICE    可选，默认 iPhone
```

macOS：

```bash
export AK_IOS_DIST_PROFILE=/secure/AppKernia_AppStore.mobileprovision
export AK_IOS_DIST_CERT_FILE=/secure/AppKernia_Distribution.p12
read -s AK_IOS_DIST_CERT_PASSWORD && export AK_IOS_DIST_CERT_PASSWORD
pnpm build:mobile:release:ios
```

Windows PowerShell 可发起 HBuilderX 云打包：

```powershell
$env:AK_IOS_DIST_PROFILE = 'D:\secure\AppKernia_AppStore.mobileprovision'
$env:AK_IOS_DIST_CERT_FILE = 'D:\secure\AppKernia_Distribution.p12'
$env:AK_IOS_DIST_CERT_PASSWORD = Read-Host 'p12 password' -MaskInput
pnpm build:mobile:release:ios
```

脚本固定 `channels=phone`、开启 dSYM，并使用临时 `0600` HBuilderX 配置文件传递签名信息。构建后的 IPA 仍需在 macOS 上使用 Xcode/Transporter 或 App Store Connect API 上传；Windows 只能完成云打包，不能替代 Apple 的最终上传和签名验收环境。

上传前至少检查：

- `CFBundleIdentifier=com.appkernia.mobile`；
- `CFBundleShortVersionString` 和 `CFBundleVersion`；
- Distribution 证书、Entitlements 和 App Store Profile；
- AppIcon、启动页、Privacy Manifest；
- TestFlight 安装、升级、Keychain 和网络策略。

## 5. HarmonyOS 正式版

HarmonyOS 使用两阶段 release 流程，避免 HBuilderX 重新生成工程时覆盖 DevEco Signing Config。

### 5.1 生成并清理工程

```bash
pnpm build:mobile:release:harmony:prepare
```

该命令会：

- 生成最新 UTS/UVue DevEco 工程；
- 覆盖 `com.appkernia.mobile` AppScope、AppKernia 名称/版本/分层图标；
- 删除 HBuilderX/DCloud 旧签名引用；
- 停在配置正式签名之前。

### 5.2 配置正式签名

用 DevEco Studio 打开：

```text
apps/ak-mobile/unpackage/dist/dev/app-harmony
```

在 Project Structure > Signing Configs 中配置 AppKernia release 证书、私钥和 Provision Profile，并确保 default product 引用该 Signing Config。签名材料只留在本机用户目录或 CI Secret Store，不提交生成工程、密码或私钥。

### 5.3 生成正式 APP

```bash
pnpm build:mobile:release:harmony
```

脚本先用 DevEco 官方签名工具验证 Profile 绑定 `com.appkernia.mobile`，然后运行：

```text
buildMode=release assembleApp
```

正式商店交付物是 signed `.app`，不是模拟器 unsigned HAP。自动签名需要在线 HarmonyOS 设备和有权限的华为开发者帐号；没有设备时应报告 blocked，不能把 unsigned HAP 当作正式包。

## 6. 签名信息安全

- 所有密码只通过环境变量/CI Secret 注入；
- Android/iOS 密码写入临时 `configure.json`，脚本退出时立即删除；
- 不在 `package.json`、Markdown、构建日志或 Git 中记录密码、Alias 对应私钥内容、证书私钥或 Profile 内容；
- Harmony DevEco Signing Config 位于被忽略的生成目录，不复制到仓库；
- 构建日志可以保存产物路径、版本、SHA-256 和脱敏证书指纹，不能保存秘密值。

## 7. 产物校验和发布边界

构建后：

```bash
pnpm build:mobile:release:verify
```

该命令要求同时找到：

- Android release APK/AAB；
- iOS release IPA；
- Harmony signed release APP；

并输出最新产物 SHA-256。随后仍需执行平台专项检查、安装测试、商店上传、审核提交和发布确认。以下状态必须分开报告：

```text
build succeeded
→ artifact identity/signature verified
→ uploaded
→ store processing passed
→ submitted for review
→ approved
→ released
```

任何前置阶段成功都不能冒充后续阶段完成。

## 8. 资讯 App 上架前专项清单

“应核 AppKernia”资讯版除通用签名检查外，还必须完成：

1. 生产 API 已执行 `000019_information_content`，公开 `/api/v1/public/content/*` 与 HTTPS `/s/{slug}` 可从公网访问；分享页封面允许微信/商店爬虫读取。
2. 每个启用的视频外链域名已写入 App 级 `content.video_external_hosts`；敏感词表、公开客服入口、举报、拉黑和账号注销路径已由运营/法务确认。
3. 在“系统 → 系统设置 → 分享配置”激活微信配置，并在应用管理完成绑定和预检；微信 AppID、Android 发布包名/应用签名、iOS Bundle ID/HTTPS Universal Link、Harmony Bundle Name 必须与微信开放平台和构建身份一致。后台不采集微信 AppSecret。
4. 在每次原生包构建前执行分享配置导出和漂移门禁；导出只更新微信节点和 iOS Associated Domains，后台保存本身不会改变已安装 App：

   ```bash
   cd server
   go run ./cmd/ak-cli app-share export \
     --app-id <public-app-uuid> \
     --output ../apps/ak-mobile \
     --android-package <release-package> \
     --android-signature <wechat-release-signature> \
     --ios-bundle-id <release-bundle-id> \
     --harmony-bundle-name <release-bundle-name>

   go run ./cmd/ak-cli app-share export \
     --app-id <public-app-uuid> \
     --output ../apps/ak-mobile \
     --android-package <release-package> \
     --android-signature <wechat-release-signature> \
     --ios-bundle-id <release-bundle-id> \
     --harmony-bundle-name <release-bundle-name> \
     --check
   ```

5. Android/iOS/HarmonyOS 都必须使用导出后的 Manifest 重新制作自定义基座/原生包，并在安装微信的发布签名真机分别验证已启用的好友、朋友圈、收藏场景；Provider 缺失、配置停用或 SDK 失败必须验证系统分享降级。
6. 使用真实内容完成 `zh-CN`/`en-US`、深色模式、动态字号、VoiceOver/TalkBack、安全区、离线/错误/刷新、视频切后台暂停和并发 401 单 Sheet 验收。
7. App Store 准备 UGC 审核说明与客服联系方式；国内安卓/Harmony 市场准备隐私清单、SDK 清单、软著/备案等渠道材料；Google Play 准备 Data safety 与账号删除入口说明。

上述清单完成前，可以报告“功能代码/编译通过”，不能报告“可直接上架”或“审核通过”。

### 8.1 App 第三方登录原生配置导出

第三方登录配置在后台完成预检并被 App 选择后，必须在制作原生包前导出。导出器读取 App 仍选中的未删除配置；只有 `active + ready` 且绑定开启的配置会在快照中标记为可登录。绑定关闭、配置停用或仅因 Secret 轮换回到草稿时仍保留公开构建字段，避免误删安全解绑所需能力或触发无意义重建。密钥、授权码、Token 和私钥不会写入移动端目录。

```bash
cd server
go run ./cmd/ak-cli app-login-provider export \
  --app-id <public-app-uuid> \
  --output ../apps/ak-mobile \
  --android-package <release-package> \
  --android-signature <wechat-release-signature> \
  --android-certificate-sha256 <google-release-sha256> \
  --ios-bundle-id <release-bundle-id> \
  --harmony-bundle-name <release-bundle-name>

go run ./cmd/ak-cli app-login-provider export \
  --app-id <public-app-uuid> \
  --output ../apps/ak-mobile \
  --android-package <release-package> \
  --android-signature <wechat-release-signature> \
  --android-certificate-sha256 <google-release-sha256> \
  --ios-bundle-id <release-bundle-id> \
  --harmony-bundle-name <release-bundle-name> \
  --check
```

同名环境变量 `AK_ANDROID_PACKAGE`、`AK_ANDROID_SIGNATURE`、`AK_ANDROID_CERTIFICATE_SHA256`、`AK_IOS_BUNDLE_ID`、`AK_HARMONY_BUNDLE_NAME` 可替代身份参数。导出器会更新 DCloud `uni-oauth`、iOS Entitlements、Android HTTPS App Link、HarmonyOS 受控 Overlay，以及 `config/login-providers.generated.json` 无密钥能力快照；`--check` 用于阻止构建字段或构建 Hash 漂移。Secret 轮换不改变构建 Hash，Client ID、包身份、签名指纹或 Link 变化必须重新导出和打包。

自定义调试基座请阅读 [AppKernia 多端自定义基座编译与打包](./mobile-custom-base-build.md)。
## 9. 多厂商 Push 发布门禁

Android Push 生产构建必须设置 `AK_REQUIRE_PUSH_CONFIG=1`，并在以下两个变体中选择一个：

```bash
AK_ANDROID_PUSH_VARIANT=google bash apps/ak-mobile/scripts/build-platform.sh android
AK_ANDROID_PUSH_VARIANT=china bash apps/ak-mobile/scripts/build-platform.sh android
```

Google 变体需要 `AK_FCM_CONFIG_FILE`；China 变体需要各厂商已批准的精确依赖坐标、公开客户端参数，以及华为/荣耀配置文件。生成过程不会读取服务端 Master Secret、Service Account 或 APNs 私钥。产物生成后必须执行：

```bash
python3 apps/ak-mobile/scripts/verify-push-variant.py google /absolute/path/to/app.aab
python3 apps/ak-mobile/scripts/verify-push-variant.py china /absolute/path/to/app.apk
```

构建通过只证明依赖边界和 UTS 编译。生产发布还必须完成 [Push SDK、许可证与隐私门禁清单](../compliance/push-sdk-inventory.md) 中的账号、签名、隐私和真机验证；对应 Admin provider 在此之前保持 disabled，全局 `AK_PUSH_ENABLED` 不得全量开启。

### 9.1 监控与告警

内部 `/internal/v1/metrics` 提供下列脱敏 Prometheus 指标；标签只包含 `app_id`、`provider`、`category`、`result`、`status` 或 `environment`，不得新增 Token、用户 ID、设备 ID：

- `appkernia_push_deliveries_total`、`appkernia_push_opened_total`：30 天窗口内按结果聚合；受理率、失败率、无效 Token 率和点击率由这两个 Gauge 计算，不使用 `rate()`。
- `appkernia_push_queue_delay_seconds_sum/count`：Delivery 创建到厂商受理的等待时间。
- `appkernia_push_queue_backlog`：pending/processing 积压。
- `appkernia_push_provider_fault`：因持续鉴权/配置错误自动 fault 的应用渠道。
- `appkernia_push_metrics_scrape_error`：指标读取失败。

部署方必须按应用和厂商建立至少以下规则：`provider_fault > 0` 立即告警；`metrics_scrape_error > 0` 立即告警；积压连续增长、5xx/transient 比例、invalid_token 比例和平均 queue delay 超过该厂商基线时告警。阈值必须在灰度期按真实流量确定，不能把未经测量的统一百分比写成生产默认。告警只引用上述低基数标签，并链接到投递 ID/厂商请求 ID 的受控日志检索，不记录原始 Token 或完整载荷。
