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

- `POST /auth/password/forgot`：通用 `202`，防止账号枚举。
- `POST /auth/password/reset`：邮箱 + OTP + 新密码，成功后撤销当前 App Mobile Sessions。
- `POST /auth/password/change`：需要 Bearer Token，并撤销其他会话。
