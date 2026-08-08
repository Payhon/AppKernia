---
title: Mobile authentication API
description: Registration, password login, refresh, context, password changes, and logout.
---

# Mobile authentication API

All routes use `/api/v1` and a valid `X-AppID`. `X-AK-Device-Key` is a random installation UUID used to associate a device record; it is not an authentication factor.

| Method | Path                              | Purpose                                                |
| ------ | --------------------------------- | ------------------------------------------------------ |
| `POST` | `/auth/register`                  | Register and begin configured email verification       |
| `POST` | `/auth/registration/verify-email` | Activate membership with a one-time six-digit OTP      |
| `POST` | `/auth/registration/resend-code`  | Resend subject to server cooldown                      |
| `POST` | `/auth/login/password`            | Password login                                         |
| `POST` | `/auth/token/refresh`             | Consume the old refresh token and return a new pair    |
| `GET`  | `/auth/context`                   | User, active tenant, roles, permissions, feature flags |
| `POST` | `/auth/logout`                    | Revoke the current mobile session                      |

```bash
curl -X POST 'http://localhost:8080/api/v1/auth/login/password' \
  -H 'Content-Type: application/json' \
  -H 'Accept-Language: en-US' \
  -H 'X-AppID: YOUR_APP_UUID' \
  -H 'X-AK-Device-Key: YOUR_INSTALLATION_UUID' \
  -d '{"email":"developer@example.test","password":"YOUR_LOCAL_PASSWORD"}'
```

`email` must be valid and `password` is 12–256 characters. A successful response includes a short-lived access token, a one-time plaintext refresh token, expiry values, `session_id`, and `app_id`. Store the refresh token in platform secure storage only.

Refresh body:

```json
{ "refresh_token": "OPAQUE_TOKEN_FROM_SECURE_STORAGE" }
```

Replace the old refresh token atomically. If local persistence fails, the client must not continue with a token the server has consumed.

- `POST /auth/password/forgot` returns generic `202` to resist enumeration.
- `POST /auth/password/reset` uses email, OTP, and new password, then revokes App Mobile sessions.
- `POST /auth/password/change` requires a bearer token and revokes other sessions.
