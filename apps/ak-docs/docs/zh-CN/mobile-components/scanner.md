---
title: 扫码能力
description: 通过 ak-scanner 扫描二维码和条形码，并用强类型事件、处理器和可信 WebView 安全消费结果。
---

# 扫码能力

`uni_modules/ak-scanner` 是 AppKernia 对系统扫码 API 的唯一能力端口。它固定从相机读取，首期支持二维码和一维条形码，不开放相册入口。业务页面不要直接调用 `uni.scanCode`。

扫码结果只留在当前 App 的内存中：不会上传服务端、不会持久化，也不会写入日志或审计记录。

## 分层与公开类型

原生端口负责启动系统扫码界面并统一平台返回值；`src/features/scanner` 中的协调器负责权限、事件、处理器优先级、可信网页和结果兜底。

```ts
type AkScanFormat = 'qr_code' | 'bar_code' | 'unknown';
type AkScanSource = 'camera';

class AkScanResult {
  scanId: string;
  rawValue: string;
  format: AkScanFormat;
  source: AkScanSource;
  scannedAt: number;
}

type AkScanResolution = 'consumed' | 'open_webview' | 'present_result';
type AkScanEventName = 'captured' | 'parsed' | 'resolved' | 'cancelled' | 'failed';
```

`captured` 表示平台已经返回原始值，`parsed` 表示已转换为统一类型，`resolved` 携带最终处理结果。用户主动关闭系统扫码界面只发出 `cancelled`，不会显示错误；权限或平台错误使用 `failed`。

## 调用和订阅

页面订阅协调器事件，在卸载时释放订阅。协调器是 single-flight：一次扫码完成前不会重复拉起系统界面。

```ts
import {
  scannerCoordinator,
  AkScanSubscription,
} from '@/src/features/scanner/application/scanner-coordinator.uts';
import { AkScanEvent } from '@/src/features/scanner/domain/models.uts';

let scanSubscription: AkScanSubscription | null = null;

export default {
  onLoad() {
    scanSubscription = scannerCoordinator.subscribe((event: AkScanEvent) => {
      if (event.name === 'resolved' && event.resolution === 'present_result') {
        // 使用 ak-bottom-sheet 展示 event.result
      }
    });
  },
  onUnload() {
    const current = scanSubscription;
    scanSubscription = null;
    if (current != null) current.dispose();
  },
  methods: {
    scan() {
      scannerCoordinator.scan();
    },
  },
};
```

## 注册业务处理器

处理器按 `priority` 从高到低执行，第一个 `canHandle` 命中的处理器决定结果。未来的 PC 登录、设备绑定等模块可以优先消费自己的协议，不需要修改首页或原生扫码端口。

```ts
import { AkScanResult } from '@/uni_modules/ak-scanner';
import { AkScanHandler } from '@/src/features/scanner/domain/models.uts';

const loginHandler: AkScanHandler = {
  id: 'pc-login',
  priority: 100,
  canHandle: (result: AkScanResult) => result.rawValue.startsWith('ak://login/'),
  handle: (_result: AkScanResult) => 'consumed',
};

const registration = scannerCoordinator.registerHandler(loginHandler);
// 模块卸载时调用 registration.dispose()
```

这只是扩展契约示例；当前版本没有实现 PC 登录或设备绑定。处理器必须由客户端代码编译注册，服务端配置不能下发可执行处理器。

## 权限和错误语义

- 只有用户点击“扫一扫”后才查询并申请相机权限。
- `onlyFromCamera: true`，不请求相册权限。
- 拒绝后结果浮层提供重试、打开系统设置和取消。
- `cancelled` 是正常用户路径；`permission_denied`、`scanner_unavailable`、`native_failure` 和 `busy` 是稳定失败码。
- 权限页只读取当前状态，打开页面本身不会弹出授权框。

参见[移动端权限中心](../guide/mobile-permissions)。

## 可信 WebView 边界

只有绝对 HTTPS URL、无凭据、无非 443 端口并命中运行时域名白名单时，协调器才会打开内置 WebView。配置缺失、刷新失败或解析异常都按关闭处理。

目标 URL 不会直接放入路由参数。协调器签发 60 秒有效的一次性内存 Token，静态 WebView 页面消费后立即删除；首次打开、`loading` 和 `load` 事件都会重新验证目标。跳转到未授权域名、HTTP 或其他外部 scheme 时页面立即关闭，并把原始扫码结果交还结果浮层。WebView 不开放扫码事件或 App 消息桥。

标准 WebView 守卫在加载事件发现越界后关闭页面，不承诺网络层零请求。若业务要求请求发送前严格阻断，需要另行实现三端原生导航代理。

## 结果兜底与复制

未命中白名单的 URL、普通文本、数字条码以及公开配置失败都使用 `ak-bottom-sheet` 展示格式和可换行原文。只有用户点击“复制结果”后才调用剪贴板 API，并分别反馈成功或失败。

## 平台矩阵

| 能力              | Android            | iOS                | HarmonyOS NEXT     |
| ----------------- | ------------------ | ------------------ | ------------------ |
| 二维码 / 条形码   | uni-app x 构建支持 | uni-app x 构建支持 | uni-app x 构建支持 |
| 相机权限适配器    | 支持               | 支持               | 支持               |
| 内置 WebView 守卫 | 支持               | 支持               | 支持               |
| 真机发布验收      | 每次发布前必做     | 每次发布前必做     | 每次发布前必做     |

iOS 模拟器没有可用相机，点击扫码会安全返回 `scanner_unavailable`，不会进入原生扫码界面。编译通过不等于真机扫码、权限恢复或 WebView 跳转验收；新增扫码能力或更新 HBuilderX 后必须重新制作包含 `uni-scanCode` 的自定义基座，不能复用功能加入前的旧基座。平台 API 以 [uni.scanCode](https://doc.dcloud.net.cn/uni-app-x/api/scan-code.html)、[web-view](https://doc.dcloud.net.cn/uni-app-x/component/web-view.html) 和[剪贴板 API](https://doc.dcloud.net.cn/uni-app-x/api/clipboard.html)为准。
