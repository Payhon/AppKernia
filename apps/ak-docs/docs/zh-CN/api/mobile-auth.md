---
title: Mobile 认证 API
description: 注册、密码登录、刷新、上下文、改密与退出的核心契约。
---

# Mobile 认证 API

所有路径位于 `/api/v1`，并要求有效 `X-AppID`。安装标识 `X-AK-Device-Key` 是随机 UUID，只用于关联设备记录，不是认证因子。

## 注册与验证

| 方法                                                    | 路径                              | 说明                          |
| ------------------------------------------------------- | --------------------------------- | ----------------------------- |
| <span class="ak-endpoint ak-endpoint--post">POST</span> | `/auth/register`                  | 注册并启动 App 配置的邮箱验证 |
| <span class="ak-endpoint ak-endpoint--post">POST</span> | `/auth/registration/verify-email` | 使用一次性 6 位 OTP 激活成员  |
| <span class="ak-endpoint ak-endpoint--post">POST</span> | `/auth/registration/resend-code`  | 按服务端冷却时间重发          |

注册返回通用 `202 Accepted`，不会暴露邮箱是否已存在。

## 密码登录

<span class="ak-endpoint ak-endpoint--post">POST</span> `/auth/login/password`

```bash
curl -X POST 'http://localhost:8080/api/v1/auth/login/password' \
  -H 'Content-Type: application/json' \
  -H 'Accept-Language: zh-CN' \
  -H 'X-AppID: YOUR_APP_UUID' \
  -H 'X-AK-Device-Key: YOUR_INSTALLATION_UUID' \
  -d '{"email":"developer@example.test","password":"YOUR_LOCAL_PASSWORD"}'
```

请求字段：

| 字段       | 类型   | 约束        |
| ---------- | ------ | ----------- |
| `email`    | string | 合法邮箱    |
| `password` | string | 12–256 字符 |

成功响应包含短期 `access_token`、一次性明文 `refresh_token`、各自过期时间、`session_id` 与 `app_id`。Refresh Token 只能写入系统安全存储。

## 刷新与上下文

| 方法                                                    | 路径                  | 说明                                           |
| ------------------------------------------------------- | --------------------- | ---------------------------------------------- |
| <span class="ak-endpoint ak-endpoint--post">POST</span> | `/auth/token/refresh` | 消费旧 Refresh Token 并返回一对新 Token        |
| <span class="ak-endpoint ak-endpoint--get">GET</span>   | `/auth/context`       | 当前用户、活动租户、角色、权限与 Feature Flags |
| <span class="ak-endpoint ak-endpoint--post">POST</span> | `/auth/logout`        | 撤销当前 Mobile Session                        |

刷新请求：

```json
{ "refresh_token": "OPAQUE_TOKEN_FROM_SECURE_STORAGE" }
```

新 Refresh Token 必须原子替换旧值。写入失败时不能继续使用已被服务端消费的旧 Token。

## 密码恢复与修改

- `POST /auth/password/forgot`：通用 `202`，防止账号枚举；手机方式须先完成下述交互式验证码。
- `POST /auth/password/reset`：邮箱 + OTP + 新密码，成功后撤销当前 App Mobile Sessions。
- `POST /auth/password/change`：需要 Bearer Token，并撤销其他会话。

## 短信发送前的交互式验证码

每次发送或重发短信都分两步执行：先创建一次性交互 Challenge，再将用户答案连同 `id` 和不透明 `token` 交给原短信接口。邮箱 OTP 不进入此流程。

### 1. 创建 Challenge

<span class="ak-endpoint ak-endpoint--post">POST</span> `/auth/sms-captcha`

匿名场景：

```bash
curl -X POST 'http://localhost:8080/api/v1/auth/sms-captcha' \
  -H 'Content-Type: application/json' \
  -H 'Accept-Language: zh-CN' \
  -H 'X-AppID: YOUR_APP_UUID' \
  -H 'X-AK-Device-Key: YOUR_INSTALLATION_UUID' \
  -d '{"scene":"login","mobile":"+8613800000000"}'
```

| `scene`             | 身份要求       | 目标字段                               |
| ------------------- | -------------- | -------------------------------------- |
| `login`             | 匿名           | `mobile`                               |
| `registration`      | 匿名           | `mobile`                               |
| `password_reset`    | 匿名           | `mobile`                               |
| `identifier_verify` | Mobile Session | `mobile`                               |
| `step_up`           | Mobile Session | `identifier_id`、`purpose`、`resource` |

`step_up` 不接受客户端手机号；服务端根据当前用户、App 和 `identifier_id` 解析真实手机标识。登录态请求同时携带 `Authorization: Bearer ACCESS_TOKEN`。

成功响应的 `data.type` 为 `click | slide | drag | rotate`。所有类型共享 `captcha_id`、`captcha_token`、`expires_in_seconds` 和原始尺寸主图；类型专属字段如下：

| 类型     | 专属字段                          | 客户端答案                   |
| -------- | --------------------------------- | ---------------------------- |
| `click`  | `prompt_image`、`required_points` | 有序 `points[]`              |
| `slide`  | `tile_image`、`initial_point`     | 原图坐标 `point`             |
| `drag`   | `tile_image`、`initial_point`     | 拼图片左上角原图坐标 `point` |
| `rotate` | `thumb_image`                     | `0..360` 的整数 `angle`      |

Challenge 有效期为 300 秒，响应带 `Cache-Control: no-store`。组件实现与坐标换算见 [`ak-interactive-captcha`](../mobile-components/interactive-captcha)。

### 2. 携带 Proof 发送短信

以短信登录为例：

```json
{
  "mobile": "+8613800000000",
  "captcha": {
    "id": "CAPTCHA_UUID",
    "token": "OPAQUE_TOKEN",
    "response": {
      "type": "slide",
      "point": { "x": 138, "y": 80 }
    }
  }
}
```

将该请求提交到 `/auth/mobile/send-code`。其他场景分别提交到 `/auth/registration/send-code`、`/auth/password/forgot`、`/me/login-identifiers/{type}/challenge`（`type=mobile`）或 `/auth/step-up/verification-code`。服务端会根据实际短信请求重新计算 App、场景、手机号、IP、安装标识及会话 Scope；Proof 不能跨场景、设备或请求复用。

| HTTP | 稳定错误码                | 处理方式                                      |
| ---- | ------------------------- | --------------------------------------------- |
| 428  | `IAM.CAPTCHA.REQUIRED`    | 获取新 Challenge，不创建 OTP                  |
| 422  | `IAM.CAPTCHA.INVALID`     | Proof 错误、过期、已消费或 Scope 不匹配，刷新 |
| 429  | `IAM.CAPTCHA.COOLDOWN`    | 按 `Retry-After: 2` 等待后刷新                |
| 503  | `IAM.CAPTCHA.UNAVAILABLE` | 保留错误态，不调用短信接口                    |

只有 Proof 验证并单次消费成功后才创建 OTP Challenge 并进入短信投递队列。不要记录、解析或缓存 `captcha_token`；配置类型切换不影响已经签发的 Challenge。
