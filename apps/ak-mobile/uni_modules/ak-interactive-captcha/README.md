# ak-interactive-captcha

AppKernia 的 uni-app x 原生交互验证码组件。组件不发网络请求，只渲染服务端 Challenge 并通过事件输出答案。

```uvue
<ak-interactive-captcha :open="captchaOpen" :challenge="captchaChallenge" :loading="captchaLoading"
  :error-text="captchaError" @confirm="confirmCaptcha" @refresh="refreshCaptcha" @close="closeCaptcha" />
```

- Props：`open`、`challenge`、`loading`、`errorText`。
- Events：`confirm(response)`、`refresh`、`close`。
- `click` 输出有序 `points`；`slide | drag` 输出原图坐标 `point`；`rotate` 输出 `0..360` 的 `angle`。
- Challenge 变化、过期或 Modal 关闭会清空/禁用旧答案。调用方应在 `confirm` 后把 `{id, token, response}` 交给短信发送接口；每次发送或重发都重新获取 Challenge。

四种 Challenge 的公开形状如下，`base64` 内容省略：

```json
{"type":"click","image":{"width":300,"height":220},"prompt_image":{"width":180,"height":40},"required_points":3}
{"type":"slide","image":{"width":300,"height":220},"tile_image":{"width":60,"height":60},"initial_point":{"x":10,"y":80}}
{"type":"drag","image":{"width":300,"height":220},"tile_image":{"width":60,"height":60},"initial_point":{"x":10,"y":80}}
{"type":"rotate","image":{"width":220,"height":220},"thumb_image":{"width":80,"height":80}}
```

对应 `confirm(response)` 分别输出：

```json
{"type":"click","points":[{"x":40,"y":60},{"x":120,"y":90},{"x":220,"y":150}]}
{"type":"slide","point":{"x":138,"y":80}}
{"type":"drag","point":{"x":138,"y":80}}
{"type":"rotate","angle":217}
```

交互与坐标换算参考 go-captcha-uni 1.0.7（MIT），未复制其 JavaScript 组件、字体、图标或 CSS；本组件使用强类型 UTS/UVue 独立实现。
