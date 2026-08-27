---
title: 移动端自定义基座与正式版打包
description: 使用跨平台脚本构建 AppKernia Android、iOS、HarmonyOS 自定义基座和正式发布包。
---

# 移动端自定义基座与正式版打包

AppKernia 提供统一的 Node.js 编排器，可从 macOS 或 Windows 调用 HBuilderX 和 DevEco 工具链。Android/iOS 自定义基座强制使用 custom playground；HarmonyOS 使用带 AppKernia 原生身份的本地 HAP，不采用 DCloud 默认包名或图标。

## 自定义基座

```bash
pnpm build:mobile:base:preflight
pnpm build:mobile:base:dry-run
pnpm build:mobile:base
pnpm build:mobile:base:verify
```

单平台入口：

```bash
pnpm build:mobile:base:android
pnpm build:mobile:base:ios:simulator
pnpm build:mobile:base:ios:device
pnpm build:mobile:base:harmony
pnpm build:mobile:base:harmony:signed
```

macOS 的 `all` 默认构建 iOS simulator 基座；Windows 默认构建 iOS device 基座并要求 Apple Development p12/Profile。可用 `AK_CUSTOM_IOS_TARGET=simulator|device` 覆盖。

## 正式版 App

```bash
pnpm build:mobile:release:dry-run
pnpm build:mobile:release:harmony:prepare
# 在 DevEco Studio 中配置 release Signing Config
pnpm build:mobile:release:preflight
pnpm build:mobile:release
pnpm build:mobile:release:verify
```

单平台入口：`build:mobile:release:android`、`build:mobile:release:ios`、`build:mobile:release:harmony:prepare` 和 `build:mobile:release:harmony`。

Android 使用自有 Keystore，iOS 使用 Apple Distribution p12 与 App Store Profile。脚本通过临时受限配置文件把签名值交给 HBuilderX，不把密码写入命令行或仓库。

Harmony 正式版分两步：

1. `release:harmony:prepare` 生成最新原生工程、覆盖 AppKernia AppScope，并清除旧签名引用；
2. 在 DevEco Studio 中为 `com.appkernia.mobile` 配置 release Signing Config，再运行 `release:harmony` 生成 signed APP。

`release:preflight` 会同时检查 Android/iOS 签名环境变量和 Harmony Signing Config；缺少任一项都会失败。

## 工具路径

脚本会探测常见安装位置，也可配置：

```text
HBUILDERX_CLI
DEVECO_STUDIO_HOME
PYTHON_BIN
```

Windows PowerShell 示例：

```powershell
$env:HBUILDERX_CLI = 'C:\Tools\HBuilderX\cli.exe'
$env:DEVECO_STUDIO_HOME = 'C:\Program Files\Huawei\DevEco Studio'
pnpm build:mobile:base:dry-run
```

## 签名环境变量

Android release：

```text
AK_ANDROID_CERT_FILE
AK_ANDROID_CERT_ALIAS
AK_ANDROID_CERT_PASSWORD
AK_ANDROID_STORE_PASSWORD
AK_ANDROID_CHANNELS（可选）
```

iOS development/release：

```text
AK_IOS_DEV_PROFILE / AK_IOS_DEV_CERT_FILE / AK_IOS_DEV_CERT_PASSWORD
AK_IOS_DIST_PROFILE / AK_IOS_DIST_CERT_FILE / AK_IOS_DIST_CERT_PASSWORD
```

不要把这些值写入 `.env` 后提交。CI 中应使用 Secret Store；本机使用当前终端的临时环境变量。

## 验收边界

- `dry-run` 只验证编排，不执行打包；
- build 成功不等于安装或真机通过；
- signed 产物不等于已经上传或通过商店审核；
- iOS simulator 不代表 iOS 真机；
- Harmony unsigned HAP 只用于官方模拟器，不是正式商店 APP。

仓库内完整操作手册：

- `docs/manual/mobile-custom-base-build.md`
- `docs/manual/mobile-production-release.md`

继续阅读[移动端开发](./mobile-development.md)和[项目结构](./project-structure.md)。
