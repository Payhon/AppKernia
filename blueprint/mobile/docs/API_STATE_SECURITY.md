# API、状态与安全设计

## 1. AkHttpClient

职责：

- Base URL allowlist 与环境配置。
- `Authorization: Bearer` 注入。
- `X-Request-ID`、应用版本、平台和设备会话标识。
- UTS DTO 序列化/反序列化。
- RequestTask 取消、超时和页面销毁清理。
- 稳定错误模型。
- 401 single-flight refresh。
- 日志脱敏。

禁止把 uView 的请求工具作为核心网络契约，避免 UI 与 API 生命周期耦合。

## 2. Refresh 算法

Android 与 iOS 的 `uni.request` 当前返回 `RequestTask`，不能把浏览器 Promise 行为当成跨端前提。AK 使用显式回调/状态机与等待队列实现 single-flight，并在页面卸载时调用 `abort()`。


```text
request → 401
  → 当前已有 refresh flight：把请求加入等待队列
  → 否则从 SecureStorage 取 Refresh Token
  → POST /auth/token/refresh
  → 原子写入新 Refresh Token
  → 更新内存 Access Token
  → 原请求重试一次
```

如果旧 Refresh Token 重放、过期或撤销：清理整个 Session、Cache、权限和 Push 绑定状态，reLaunch 登录页。

## 3. 重试

- GET/HEAD：网络中断时可有限指数退避并支持取消。
- POST/PATCH/DELETE：默认不自动重试。
- 有服务端 Idempotency-Key 的明确操作才允许安全重试。
- 429 尊重服务端 Retry-After。

## 4. 安全存储

`ak-secure-storage` 提供统一接口：

```text
set(key, value, accessibility)
get(key)
delete(key)
clearNamespace()
```

平台实现：Android Keystore 加密材料、iOS Keychain、Harmony 安全存储。普通 `uni storage` 只保存语言、已同意法律版本、非敏感 UI 偏好。

## 5. OpenAPI 生成

Backend `app-v1` OpenAPI 生成：

- 明确 UTS 类型，避免长期使用无界 `UTSJSONObject`。
- Endpoint Method/Path/Auth/Request/Response/Error 描述。
- Schema Hash 供 CI 漂移检查。
- 生成代码单独目录且不可手改。

## 6. 隐私与日志

默认禁止记录：

```text
Authorization
Cookie
Refresh Token
Password
OTP
MFA Secret
Recovery Code
Push Token
完整手机号/邮箱
文件预签名 URL
OAuth code/verifier
```

## 7. 短信交互式验证码

- `POST /auth/sms-captcha` 创建 `login | registration | password_reset | identifier_verify | step_up` 挑战；后两类要求当前 Mobile Session。
- `ak-interactive-captcha` 只输出交互答案，不发起网络。Repository 将 OpenAPI Wire 映射为强类型组件模型，Runtime 统一供登录、注册、找回密码及账号连接页调用。
- 每次短信发送和重发都提交新的 `{ id, token, response }`。证明只可消费一次，并绑定 App、场景、规范化手机号、IP 和设备键；登录态场景额外绑定用户、Session、用途和资源。
- 缺少证明、证明无效或验证码刷新冷却分别按稳定错误码处理；只有验证成功后才创建 OTP Challenge。邮箱 OTP 保持原流程。

错误上报只发送稳定业务码、平台、版本、Request ID 和脱敏上下文。遥测 SDK 必须在隐私同意后初始化，并可通过配置完全关闭。
