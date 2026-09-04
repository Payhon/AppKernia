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

- `POST /auth/password/forgot` returns generic `202` to resist enumeration; the mobile path first requires the interactive CAPTCHA below.
- `POST /auth/password/reset` uses email, OTP, and new password, then revokes App Mobile sessions.
- `POST /auth/password/change` requires a bearer token and revokes other sessions.

## Interactive CAPTCHA before SMS delivery

Every SMS send or resend is a two-step operation: create a single-use interactive challenge, then submit the user's answer together with its `id` and opaque `token` to the original SMS endpoint. Email OTP does not use this flow.

### 1. Create a challenge

`POST /auth/sms-captcha`

```bash
curl -X POST 'http://localhost:8080/api/v1/auth/sms-captcha' \
  -H 'Content-Type: application/json' \
  -H 'Accept-Language: en-US' \
  -H 'X-AppID: YOUR_APP_UUID' \
  -H 'X-AK-Device-Key: YOUR_INSTALLATION_UUID' \
  -d '{"scene":"login","mobile":"+15551234567"}'
```

| `scene`             | Authentication | Target fields                              |
| ------------------- | -------------- | ------------------------------------------ |
| `login`             | Anonymous      | `mobile`                                   |
| `registration`      | Anonymous      | `mobile`                                   |
| `password_reset`    | Anonymous      | `mobile`                                   |
| `identifier_verify` | Mobile session | `mobile`                                   |
| `step_up`           | Mobile session | `identifier_id`, `purpose`, and `resource` |

`step_up` never accepts a client-supplied phone number. The server resolves the real mobile identifier from the current user, App, and `identifier_id`. Authenticated requests also include `Authorization: Bearer ACCESS_TOKEN`.

The successful response discriminator is `data.type: click | slide | drag | rotate`. Every type includes `captcha_id`, `captcha_token`, `expires_in_seconds`, and a main image with original dimensions:

| Type     | Type-specific fields              | Client answer                        |
| -------- | --------------------------------- | ------------------------------------ |
| `click`  | `prompt_image`, `required_points` | Ordered `points[]`                   |
| `slide`  | `tile_image`, `initial_point`     | Original-image `point`               |
| `drag`   | `tile_image`, `initial_point`     | Tile top-left original-image `point` |
| `rotate` | `thumb_image`                     | Integer `angle` from `0..360`        |

Challenges expire after 300 seconds and responses carry `Cache-Control: no-store`. See [`ak-interactive-captcha`](../mobile-components/interactive-captcha) for rendering and coordinate conversion.

### 2. Submit the proof to the SMS endpoint

SMS login example:

```json
{
  "mobile": "+15551234567",
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

Send that body to `/auth/mobile/send-code`. The other scenes use `/auth/registration/send-code`, `/auth/password/forgot`, `/me/login-identifiers/{type}/challenge` with `type=mobile`, or `/auth/step-up/verification-code`. The server recomputes the App, scene, normalized mobile, IP, installation, and session scope from the actual SMS request. A proof cannot cross scenes, devices, or requests.

| HTTP | Stable code               | Client action                                       |
| ---- | ------------------------- | --------------------------------------------------- |
| 428  | `IAM.CAPTCHA.REQUIRED`    | Fetch a new challenge; no OTP is created            |
| 422  | `IAM.CAPTCHA.INVALID`     | Invalid, expired, consumed, or cross-scope; refresh |
| 429  | `IAM.CAPTCHA.COOLDOWN`    | Wait for `Retry-After: 2`, then refresh             |
| 503  | `IAM.CAPTCHA.UNAVAILABLE` | Keep the error state and do not call the SMS sender |

The server creates an OTP challenge and queues SMS delivery only after the proof is verified and consumed once. Never log, parse, or cache `captcha_token`; changing the configured type does not invalidate an issued challenge.
