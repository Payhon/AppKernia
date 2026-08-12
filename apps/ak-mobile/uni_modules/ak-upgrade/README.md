# ak-upgrade

AppKernia 的 uni-app x 原生升级模块。它通过注入的项目后端 Provider 获取升级策略，不依赖 uniCloud、后台通知或其他 uni-module。

- Android：优先打开已配置应用市场，无法打开时可下载后端签名 APK，并调用系统安装器。
- iOS / HarmonyOS：只打开应用市场或 HTTPS 发布地址。
- `uni_app_x` 不实现 WGT 热更新；模块仅查询 `native_app` 策略。
- 自动检查网络失败放行启动；已展示的强制升级保持阻断并允许重试。

真实应用市场跳转、APK 安装和强制升级返回行为必须在真机验证，平台编译不能替代该证据。
