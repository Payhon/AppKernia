# AppKernia 多端自定义基座编译与打包

本文说明如何在 macOS 或 Windows 上，为 `apps/ak-mobile` 构建 Android、iOS、HarmonyOS 自定义调试产物。Android/iOS 强制使用 HBuilderX custom playground；HarmonyOS 没有 Android/iOS 式“基座”，对应交付物是由 HBuilderX 生成源码、再由 DevEco 工具链打包的 AppKernia HAP。

三端必须同时满足：

- 原生标识为 `com.appkernia.mobile`；
- App 名称为 `AppKernia`；
- 图标来自 `apps/ak-admin/public/brand/appkernia-mark.png`；
- 不使用 `io.dcloud.uniappx` 或 DCloud/HBuilderX 默认图标；
- Android/iOS 运行命令明确使用 `--playground custom`。

## 1. 环境要求

通用环境：

- Node.js 24 LTS、pnpm 11；
- Python 3 和 Pillow；
- 已登录 HBuilderX，版本使用项目当前验证的稳定 Patch；
- 首次云打包时可正常访问 DCloud 打包服务。

平台要求：

| 平台 | macOS | Windows | 额外要求 |
| --- | --- | --- | --- |
| Android 自定义基座 | 支持 | 支持 | Android SDK/设备只在运行验收时需要 |
| iOS 模拟器基座 | 支持 | 不支持 | macOS、Xcode、iOS Simulator |
| iOS 真机基座 | 支持 | 支持云打包 | Apple Development p12、开发 Provisioning Profile |
| HarmonyOS HAP | 支持 | 支持 | DevEco Studio、HarmonyOS SDK、OHPM、Hvigor |

脚本会自动探测常见安装位置，也可显式配置：

```text
HBUILDERX_CLI       HBuilderX cli/cli.exe 的绝对路径
DEVECO_STUDIO_HOME  DevEco Studio Contents/安装根目录
PYTHON_BIN          Python 3 可执行文件
```

macOS 示例：

```bash
export HBUILDERX_CLI=/Applications/HBuilderX.app/Contents/MacOS/cli
export DEVECO_STUDIO_HOME=/Applications/DevEco-Studio.app/Contents
```

Windows PowerShell 示例：

```powershell
$env:HBUILDERX_CLI = 'C:\Tools\HBuilderX\cli.exe'
$env:DEVECO_STUDIO_HOME = 'C:\Program Files\Huawei\DevEco Studio'
$env:PYTHON_BIN = 'C:\Python312\python.exe'
```

## 2. 根目录自动化命令

在仓库根目录执行：

```bash
# 仅检查工具链和资产契约
pnpm build:mobile:base:preflight

# 展示完整三端命令，不发起云打包或原生编译
pnpm build:mobile:base:dry-run

# 自动构建当前系统支持的三端自定义产物
pnpm build:mobile:base

# 单平台
pnpm build:mobile:base:android
pnpm build:mobile:base:ios:simulator
pnpm build:mobile:base:ios:device
pnpm build:mobile:base:harmony

# 检查 APK、iOS App/IPA、HAP 的原生身份和实际图标内容
pnpm build:mobile:base:verify
```

`build:mobile:base` 的 iOS 默认目标：macOS 使用 simulator，Windows 使用 device。可以覆盖：

```bash
AK_CUSTOM_IOS_TARGET=device pnpm build:mobile:base
```

PowerShell：

```powershell
$env:AK_CUSTOM_IOS_TARGET = 'device'
pnpm build:mobile:base
```

## 3. Android 自定义基座

执行：

```bash
pnpm build:mobile:base:android
```

脚本将：

1. 从品牌主图重新生成 Android 密度图标和 Android 12 启动图；
2. 校验 `manifest.json` 的 custom-base 契约；
3. 调用 HBuilderX cloud pack，固定 `iscustom=true`；
4. 使用公共调试证书生成调试 APK；
5. 保存脱敏日志到 `apps/ak-mobile/unpackage/mobile-package-logs/`。

公共调试证书只适用于调试基座，不得作为正式版签名。

运行时入口：

```bash
apps/ak-mobile/scripts/build-platform.sh android
```

该入口固定传入 `--playground custom`，不会静默回退默认基座。

## 4. iOS 自定义基座

### 4.1 模拟器

仅 macOS：

```bash
pnpm build:mobile:base:ios:simulator
```

产物是 x86_64/模拟器 App，只证明模拟器安装运行，不代表 iOS 真机签名通过。

### 4.2 真机

先把开发签名材料通过当前终端或 CI Secret 注入，不写入 Git：

```text
AK_IOS_DEV_PROFILE        Apple Development Provisioning Profile
AK_IOS_DEV_CERT_FILE      Apple Development p12
AK_IOS_DEV_CERT_PASSWORD  p12 密码
```

macOS：

```bash
export AK_IOS_DEV_PROFILE=/secure/AppKernia_Development.mobileprovision
export AK_IOS_DEV_CERT_FILE=/secure/AppKernia_Development.p12
read -s AK_IOS_DEV_CERT_PASSWORD && export AK_IOS_DEV_CERT_PASSWORD
pnpm build:mobile:base:ios:device
```

Windows PowerShell：

```powershell
$env:AK_IOS_DEV_PROFILE = 'D:\secure\AppKernia_Development.mobileprovision'
$env:AK_IOS_DEV_CERT_FILE = 'D:\secure\AppKernia_Development.p12'
$env:AK_IOS_DEV_CERT_PASSWORD = Read-Host 'p12 password' -MaskInput
pnpm build:mobile:base:ios:device
```

脚本把签名值写入进程级临时 `configure.json`，权限设为 `0600`（Windows 使用当前用户临时目录 ACL），HBuilderX 读取后立即删除。密码不会出现在 `package.json`、命令行参数或构建报告中。

iOS 运行入口同样固定 custom playground：

```bash
apps/ak-mobile/scripts/build-platform.sh ios
```

## 5. HarmonyOS 自定义 HAP

模拟器/无签名调试 HAP：

```bash
pnpm build:mobile:base:harmony
```

脚本顺序：

1. HBuilderX 编译 UTS/UVue 并生成 DevEco 工程；
2. `prepare-harmony-native.py --unsigned` 覆盖 AppScope、AppKernia 包名、版本和图标，并清除 DCloud 旧签名引用；
3. 无代理运行 OHPM；
4. 运行 `buildMode=debug assembleHap`；
5. 生成 `entry-default-unsigned.hap`，可安装到官方模拟器。

真机调试签名采用两阶段流程：

1. 先运行上述 unsigned 命令生成原生工程；
2. 用 DevEco Studio 打开 `apps/ak-mobile/unpackage/dist/dev/app-harmony`；
3. 在 Project Structure 中为 `com.appkernia.mobile` 配置自动或手工调试签名；
4. 连接在线 HarmonyOS 设备；
5. 执行：

```bash
pnpm build:mobile:base:harmony:signed
```

该命令不会重新调用 HBuilderX 生成工程，因此不会覆盖刚配置的 Signing Config。脚本会验证 Provision Profile 属于 `com.appkernia.mobile`，再生成 signed debug HAP。

## 6. 产物与日志

常见产物：

```text
apps/ak-mobile/unpackage/debug/android_debug.apk
apps/ak-mobile/unpackage/debug/*simulator*.app
apps/ak-mobile/unpackage/dist/dev/app-harmony/entry/build/**/outputs/**/*.hap
```

日志：

```text
apps/ak-mobile/unpackage/mobile-package-logs/
```

`unpackage/` 已忽略，不得把 APK、IPA/HAP、证书、Profile 或临时配置提交到 Git。

## 7. 验收边界

- `dry-run`：只证明命令编排和路径选择，不证明编译成功。
- 云打包成功：只证明产物生成，不证明安装运行。
- 模拟器启动：不证明物理设备签名、安全存储、通知或硬件能力。
- 真机自定义基座：仍需记录设备、OS/API、产物 SHA-256、安装、冷启动和关键日志。
- `verify`：校验原生身份和图标，不代替商店发布、隐私合规或完整业务 E2E。

正式版 App 请继续阅读 [AppKernia 多端正式版 App 编译与发布](./mobile-production-release.md)。
