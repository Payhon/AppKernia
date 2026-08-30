# ak-scanner

AppKernia 的相机扫码 UTS 能力端口。模块固定使用 `onlyFromCamera: true`，首期只请求 `qrCode` 与 `barCode`，业务页面不得直接调用 `uni.scanCode`。

扫码内容只回传给调用模块，不上传、不持久化、不写日志。事件协调、处理器优先级、可信 WebView 与结果展示由 `src/features/scanner` 负责。
