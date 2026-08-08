---
title: Responses, errors, and idempotency
description: Common response shapes, locale negotiation, pagination, and safe retry behavior.
---

# Responses, errors, and idempotency

Success:

```json
{
  "code": "OK",
  "message": "success",
  "data": {},
  "request_id": "01900000-0000-7000-8000-000000000001"
}
```

Error:

```json
{
  "error": {
    "code": "IAM.AUTH.INVALID_CREDENTIALS",
    "message_key": "errors.iam.auth.invalid_credentials",
    "message": "Invalid email or password",
    "details": {}
  },
  "request_id": "01900000-0000-7000-8000-000000000001"
}
```

Clients branch on stable `error.code` or `message_key`, never on `message`.

| Status | Meaning                                 | Client behavior                     |
| ------ | --------------------------------------- | ----------------------------------- |
| `400`  | Malformed request                       | Fix the request                     |
| `401`  | Invalid session or expired access token | One single-flight refresh           |
| `403`  | Authenticated but unauthorized          | Do not refresh                      |
| `404`  | Missing or invisible resource           | Do not infer cross-tenant existence |
| `409`  | Version or state conflict               | Reload and resolve explicitly       |
| `422`  | Field validation                        | Map details to local fields         |
| `429`  | Rate limited                            | Honor `Retry-After`                 |

Send `Accept-Language: zh-CN` or `en-US`; read `Content-Language`. Codes and raw values do not change with language.

Admin lists commonly use `page` and `page_size`. Mobile notifications and articles use opaque cursors; clients must not parse a cursor.

GET and HEAD may use limited backoff. POST, PATCH, and DELETE are not retried by default. Replay a write only when its contract explicitly supports `Idempotency-Key`.
