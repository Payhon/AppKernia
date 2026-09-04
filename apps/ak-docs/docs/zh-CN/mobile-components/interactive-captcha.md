---
title: ak-interactive-captcha
description: uni-app x 四类交互式验证码组件及短信门禁接入方法。
---

# `ak-interactive-captcha`

`ak-interactive-captcha` 位于 `apps/ak-mobile/uni_modules/ak-interactive-captcha`，支持 uni-app x Android、iOS 与 HarmonyOS Core。它只负责展示 Challenge 和输出答案，不发网络请求、不保存 Token，也不发送短信。

```vue
<ak-interactive-captcha
  :open="captchaOpen"
  :challenge="captchaChallenge"
  :loading="captchaLoading"
  :error-text="captchaError"
  @confirm="confirmCaptcha"
  @refresh="refreshCaptcha"
  @close="closeCaptcha"
/>
```

## Props 与事件

| Prop        | 类型                                      | 说明                     |
| ----------- | ----------------------------------------- | ------------------------ |
| `open`      | `boolean`                                 | 是否显示 Modal           |
| `challenge` | `AkInteractiveCaptchaChallenge` 或 `null` | 当前服务端 Challenge     |
| `loading`   | `boolean`                                 | 禁止确认、关闭和重复刷新 |
| `errorText` | `string`                                  | 调用方翻译后的可见错误   |

| 事件      | 参数                           | 调用方职责                      |
| --------- | ------------------------------ | ------------------------------- |
| `confirm` | `AkInteractiveCaptchaResponse` | 包装成 Proof 后调用对应短信接口 |
| `refresh` | 无                             | 请求一个新 Challenge            |
| `close`   | 无                             | 关闭并释放调用方状态            |

Challenge 切换、关闭、刷新或 300 秒到期时，组件会清空旧坐标和角度。背景点击不会关闭 Modal。

## 四种交互

| 类型     | 交互方式                   | `confirm` 输出                       |
| -------- | -------------------------- | ------------------------------------ |
| `click`  | 按提示依次点击主图，可撤销 | `{ type: 'click', points: [...] }`   |
| `slide`  | 拖动原生 slider            | `{ type: 'slide', point: { x, y } }` |
| `drag`   | 直接拖动拼图片             | `{ type: 'drag', point: { x, y } }`  |
| `rotate` | 拖动原生 slider 旋转缩略图 | `{ type: 'rotate', angle }`          |

点击和拖动坐标均按服务端主图的原始宽高输出，不要把屏幕显示坐标直接提交给后端。

## 接入 Repository

应用启动时已经用 `AkHttpClient` 配置 `captchaRuntime`。Feature 只需创建业务意图并持有返回的 Challenge：

```typescript
const repository = captchaRuntime.getRepository();
if (repository == null) return;

repository.create(
  { scene: 'login', mobile: '+8613800000000', identifierId: '', purpose: '', resource: '' },
  (challenge) => {
    this.captchaChallenge = challenge;
    this.captchaOpen = true;
  },
  (error) => {
    this.captchaError = akI18n.t(error.messageKey, null);
  },
);
```

收到 `confirm(response)` 后使用 `repository.submission(challenge, response)` 生成 `{ id, token, response }`，并立即调用原短信接口。发送成功后关闭 Modal，并以服务端 `retry_after_seconds` 启动 OTP 倒计时；失败时保留 Modal，Proof 过期、无效或耗尽则刷新。

每次发送和重发都必须重新创建 Challenge。邮箱 OTP 不显示该组件。完整 HTTP 契约见 [Mobile 认证 API](../api/mobile-auth#短信发送前的交互式验证码)。

## 复用边界

- 新模块复用现有 `captchaRuntime` 与模型，不在组件内新增请求或业务 Scene。
- `token` 只透传给后端，不解析、不缓存、不记录日志。
- 可见文案由 `AkI18n` 提供；组件内置安全区、44px 操作目标、错误播报且没有装饰动画。
- 当前承诺范围仅为 uni-app x Android/iOS/HarmonyOS Core，不包含传统 uni-app、H5 或小程序。
