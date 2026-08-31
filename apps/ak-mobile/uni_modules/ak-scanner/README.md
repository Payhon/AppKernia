# ak-scanner

AppKernia 的相机扫码 UTS 能力端口。模块固定使用 `onlyFromCamera: true`，首期只请求 `qrCode` 与 `barCode`，业务页面不得直接调用 `uni.scanCode`。

扫码内容只回传给调用模块，不上传、不持久化、不写日志。事件协调、处理器优先级、可信 WebView 与结果展示由 `src/features/scanner` 负责。

iOS 模拟器没有可用相机，适配器会在调用原生扫码 API 前返回 `scanner_unavailable`；二维码、条形码和权限流程必须使用真机验收。新增扫码能力或更新 HBuilderX 后，应重新制作包含 `uni-scanCode` 的自定义基座，不能继续复用功能加入前的旧基座。
