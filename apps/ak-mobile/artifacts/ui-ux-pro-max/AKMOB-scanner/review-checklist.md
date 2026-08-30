# Review checklist

- [x] 首页扫码按钮位于消息右侧，触控区至少 44 × 44px，并有双语读屏名称。
- [x] 访客可用、single-flight、防重复和页面卸载订阅释放已实现。
- [x] 二维码/条形码格式、取消/失败分流和强类型事件已实现。
- [x] 权限拒绝含重试、系统设置和取消；权限页只读状态不主动申请。
- [x] 普通结果、长文本、复制成功/失败和越界拦截反馈均使用 `ak-*`。
- [x] 一次性 Token、HTTPS/域名重校验和无消息桥边界已实现。
- [x] Android HBuilderX VDOM 编译完成。
- [x] iOS 与 HarmonyOS HBuilderX 编译完成（Harmony 为未签名 debug HAP）。
- [ ] Android、iOS、HarmonyOS 真机二维码/条形码、权限和 WebView 验收。
- [ ] 三端动态字号、读屏、高对比度和减少动效人工验收。

编译或静态检查不能替代真机验收，未取得证据前保持未勾选。
