# 三端兼容与工程环境

## 1. 工具基线

- HBuilderX：以 5.15 稳定线或后续经项目验证的 Patch 为基线。
- uView Ultra：4.5.18。
- Harmony：HBuilderX + DevEco Studio 兼容工具链，设备 API 14+。
- iOS：Xcode/证书/Provisioning Profile；项目最低 iOS 13。
- Android：项目最低 Android 8/API 26，并保持 16 KB page-size 兼容。

版本升级必须更新 `spec/platform-matrix.json`、锁定信息、构建日志和兼容报告。

## 2. 不允许的“伪验证”

以下都不代表三端通过：

- 只运行 Web 预览。
- 只运行 TypeScript/UTS 静态检查。
- 只在 Android 标准基座运行。
- 只看 uView 官方兼容表。
- 只运行 iOS 模拟器而没有 Release/真机 Smoke。
- 只编译 Harmony 而没有真机返回键、安全区和存储验证。

## 3. 核心兼容用例

每个平台至少验证：

- 冷启动、热启动、前后台切换。
- 隐私同意和法律文档。
- 密码/OTP 输入、键盘遮挡、自动填充。
- Refresh Rotation 与离线恢复。
- TabBar、返回键、深链、OAuth 回调。
- 上传头像、图片权限拒绝和恢复。
- 会话撤销、安全存储清理。
- 列表滚动、下拉刷新、消息 Badge。
- 动态字号、长中文/英文、RTL 不作为首发但不得崩溃。
- 安全区、刘海、圆角屏。

## 4. 条件编译

条件编译只能存在于平台 Adapter 或极薄的 UI 兼容层。Feature 目录不得大量出现 `#ifdef APP-ANDROID/IOS/HARMONY`。

## 5. Vapor

Vapor 是独立升级项目，不是默认性能开关。只有 `AKMOB-240` 全部通过且 ADR 批准后，才能更改 manifest 默认渲染模式。
