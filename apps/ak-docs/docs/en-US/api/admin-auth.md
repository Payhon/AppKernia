---
title: Admin authentication API
description: Admin login, captcha, refresh, context, and self-service security.
---

# Admin authentication API

Admin only calls `/admin-api/v1`. The access token stays in memory and the refresh token uses a Secure, HttpOnly, SameSite cookie.

`POST /auth/login` accepts a valid email and a 12–256 character password. After the server-side failure threshold, it returns `IAM.AUTH.CAPTCHA_REQUIRED`; the client then calls `POST /auth/login/captcha`.

The captcha response contains a short-lived base64 `image/png`, `captcha_id`, and expiry. The answer is never returned as text or SVG. A challenge is bound to identity, audience, and source, and one login attempt consumes it.

| Method | Path                  | Purpose                                                           |
| ------ | --------------------- | ----------------------------------------------------------------- |
| `POST` | `/auth/token/refresh` | Rotate the Admin refresh cookie; follow OpenAPI CSRF requirements |
| `GET`  | `/auth/context`       | User, active tenant, roles, permissions, menus, feature flags     |
| `POST` | `/auth/switch-tenant` | Switch active tenant and refresh context                          |
| `POST` | `/auth/logout`        | Revoke the current Admin session                                  |

`/me` provides profile, avatar, session, device, password, TOTP, recovery-code, and OAuth connection operations. MFA secrets and recovery codes appear once and must never enter logs, fixtures, or screenshots.
