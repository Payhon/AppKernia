---
title: Admin 认证 API
description: Admin 登录、Captcha、刷新、上下文与自助安全 API。
---

# Admin 认证 API

Admin 只调用 `/admin-api/v1`。Access Token 只保存在内存；Refresh Token 使用 Secure + HttpOnly + SameSite Cookie。

## 登录

<span class="ak-endpoint ak-endpoint--post">POST</span> `/auth/login`

```json
{
  "email": "admin@example.test",
  "password": "YOUR_LOCAL_PASSWORD"
}
```

密码为 12–256 字符。服务端连续失败达到阈值时返回稳定错误 `IAM.AUTH.CAPTCHA_REQUIRED`，随后客户端调用：

<span class="ak-endpoint ak-endpoint--post">POST</span> `/auth/login/captcha`

Captcha 返回短时 `image/png` Base64、`captcha_id` 与过期秒数。答案不会以文字或 SVG 形式返回；挑战与标识、Audience 和来源绑定，一次登录尝试即消费。

## 会话

| 方法   | 路径                  | 说明                                                |
| ------ | --------------------- | --------------------------------------------------- |
| `POST` | `/auth/token/refresh` | 轮换 Admin Refresh Cookie，CSRF 规则以 OpenAPI 为准 |
| `GET`  | `/auth/context`       | 用户、活动租户、角色、权限、菜单与 Feature Flags    |
| `POST` | `/auth/switch-tenant` | 切换活动租户并更新上下文                            |
| `POST` | `/auth/logout`        | 撤销当前 Admin Session                              |

## 当前管理员安全

`/me` 下提供资料、头像、会话、设备、改密、TOTP、恢复码与 OAuth 绑定接口。MFA Secret 和恢复码只显示一次，不能进入日志、Fixture 或截图。
