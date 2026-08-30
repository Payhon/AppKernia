---
title: Notification and push API
description: Submit asynchronous notifications through the trusted M2M API and manage preferences, device bindings, in-app messages, and open metrics through Mobile APIs.
---

# Notification and push API

Every endpoint on this page is under `/api/v1`, but it spans two identity surfaces that are not interchangeable:

| Identity surface | Audience    | Purpose                                                      |
| ---------------- | ----------- | ------------------------------------------------------------ |
| Trusted service  | `ak-api`    | Submit, query, and cancel application notifications          |
| Current app user | `ak-mobile` | Preferences, in-app messages, push registration, open events |

The [current OpenAPI reference](./online-reference) remains the source of truth for fields, enums, and response schemas.

## Prepare a trusted service

1. Create an active API Client in Admin, with an expiration and optional CIDR policy.
2. Grant `notify.message.submit`; status queries also require `notify.message.status.read`.
3. An all-active-app-user audience additionally requires `notify.message.broadcast`.
4. A `news_operations` submission additionally requires `notify.operations.publish`.
5. Explicitly allow the target app for the API Client. An empty app allowlist denies all apps.
6. Exchange credentials at `/auth/client-token` for a short-lived `ak-api` JWT. A Machine Principal receives no refresh token.

## Submit a notification

Endpoint: `/apps/{app_id}/notifications`

```http
POST /api/v1/apps/01900000-0000-7000-8000-000000000001/notifications
Authorization: Bearer AK_API_ACCESS_TOKEN
Idempotency-Key: account-security-event-20260830-0001
Accept-Language: en-US
Content-Type: application/json

{
  "source": "account-security",
  "business_event_id": "20260830-0001",
  "category": "service_security",
  "audience": {
    "type": "users",
    "user_ids": ["01900000-0000-7000-8000-000000000099"]
  },
  "content": {
    "type": "inline",
    "inline": {
      "title": {
        "zh-CN": "安全提醒",
        "en-US": "Security alert"
      },
      "body": {
        "zh-CN": "账号安全状态已更新",
        "en-US": "Your account security state changed"
      }
    }
  },
  "push": true,
  "ttl_seconds": 3600,
  "route_key": "notification_detail",
  "resource_id": "opaque-business-id"
}
```

Success returns `202 Accepted` with `message_id`, `run_id`, current status, `status_url`, and creation time. This means the asynchronous pipeline accepted the submission—not that a provider or device received it.

Choose exactly one content union: a template or controlled bilingual inline content. The request does not accept raw tokens, arbitrary URLs, component names, scripts, or provider-specific payloads.

## Idempotency and cancellation

`Idempotency-Key` is 8–255 characters. The current service scopes it by tenant and caller identity (the API Client for M2M), so one API Client must generate globally unique keys across every app and source it can access. The same key and normalized body return the original submission; a different body under the same key returns `409 NOTIFY.IDEMPOTENCY.CONFLICT`.

- Read status: `/apps/{app_id}/notifications/{message_id}`
- Cancel: `/apps/{app_id}/notifications/{message_id}/cancel`

Status includes recipient, evaluated, device delivery, provider accepted, failed, invalid token, skipped, and opened counts. Cancellation can stop only a notification that has not entered publishing or fanout; provider-accepted notifications cannot be recalled.

## Current-user Mobile endpoints

These calls use an `ak-mobile` bearer token and the current app context:

| Method   | Path                                       | Purpose                                                        |
| -------- | ------------------------------------------ | -------------------------------------------------------------- |
| `GET`    | `/me/notification-preferences`             | Read push master and category subscriptions                    |
| `PATCH`  | `/me/notification-preferences`             | Update `push`, `push_service`, and `push_operations`           |
| `GET`    | `/me/notifications`                        | Page through delivered in-app notifications                    |
| `GET`    | `/me/notifications/{id}`                   | Read one notification owned by the current user and app        |
| `PATCH`  | `/me/notifications/{id}/read`              | Mark an in-app notification as read                            |
| `GET`    | `/me/push-devices/current`                 | Read current-installation status without returning a token     |
| `POST`   | `/me/push-devices`                         | Idempotently register or refresh an opaque provider token      |
| `DELETE` | `/me/push-devices/{push_device_id}`        | Deactivate the current user's binding in the current app       |
| `POST`   | `/me/push-deliveries/{delivery_id}/opened` | Idempotently record an open; does not mark in-app content read |

A registration includes provider, platform, build variant, token, normalized locale, SDK version, and app version. Tenant, app, user, and business device identity come from the verified session and app context; the response never returns the token.

## Internal Go integration

Inside the modular monolith, business modules should depend on `server/internal/platform/notification.Service`:

```go
type Service interface {
    Submit(ctx context.Context, scope Scope, cmd SubmitCommand) (Submission, error)
    SubmitTx(ctx context.Context, tx pgx.Tx, scope Scope, cmd SubmitCommand) (Submission, error)
    Status(ctx context.Context, scope Scope, id uuid.UUID) (SubmissionStatus, error)
    Cancel(ctx context.Context, scope Scope, id uuid.UUID) error
}
```

Use `SubmitTx` when a notification must commit atomically with a business fact. A trusted caller or authentication middleware constructs `Scope`; never copy tenant, app, or actor from an HTTP body. Business modules must not write `notify.*` tables or insert River jobs directly.

## Common errors

| HTTP | Stable code                   | Meaning                                                  |
| ---- | ----------------------------- | -------------------------------------------------------- |
| 401  | `AUTH.CLIENT.INVALID`         | Invalid credentials, JWT, client state, or expiration    |
| 403  | `COMMON.FORBIDDEN`            | CIDR, app allowlist, or permission check failed          |
| 409  | `NOTIFY.IDEMPOTENCY.CONFLICT` | The idempotency key is bound to a different request body |
| 422  | `NOTIFY.SUBMISSION.INVALID`   | Invalid audience, content union, schedule, TTL, or route |

Logs and audit events do not store access tokens, device tokens, or full message payloads. See [Notification operations](../guide/notification-operations) for runtime status and failure handling, and [Notification architecture](../concepts/notification-architecture) for asynchronous semantics.
