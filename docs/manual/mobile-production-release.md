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

自定义调试基座请阅读 [AppKernia 多端自定义基座编译与打包](./mobile-custom-base-build.md)。
