---
title: 移动端开发
description: 使用 HBuilderX 运行与验证 uni-app x 移动工程。
---

# 移动端开发

AK Mobile 位于 `apps/ak-mobile`，使用 uni-app x、UTS/UVue、VDOM 与 AK UI。普通 Node/Vite 构建不能替代移动平台构建。

## 工具链

- HBuilderX 稳定版（项目当前以 5.15 稳定线或经验证补丁为基线）
- Android：Android 8 / API 26 及以上目标工具链
- iOS：Xcode、证书与 Provisioning Profile，最低 iOS 13
- HarmonyOS NEXT：HBuilderX + DevEco Studio，设备 API 14+

## 检查本机

```bash
apps/ak-mobile/scripts/detect-toolchain.sh
apps/ak-mobile/scripts/check-project.sh
```

## 打开与运行

1. 在 HBuilderX 打开 `apps/ak-mobile`。
2. 确认 `manifest.json` 和目标平台签名配置只使用本地凭据。
3. 将 API Base URL 指向本机或局域网可访问的 Go API。
4. 选择 Android、iOS 或 HarmonyOS 目标运行。

项目提供真实平台入口：

```bash
apps/ak-mobile/scripts/build-platform.sh android
apps/ak-mobile/scripts/build-platform.sh ios
apps/ak-mobile/scripts/build-platform.sh harmony
```

这些命令能否完成取决于本机 IDE、SDK、签名与设备。静态检查成功时，只能报告“静态门禁通过”；没有目标平台构建与设备记录时，不能报告该平台通过。

自定义基座、正式签名包、macOS/Windows 环境变量和根目录自动化入口见[移动端自定义基座与正式版打包](./mobile-packaging.md)。

## 组件使用

业务页面只能使用 `components/ak-ui` 暴露的 `ak-*`，不能直接绑定 `up-*`。从[移动组件总览](../mobile-components/)开始。
