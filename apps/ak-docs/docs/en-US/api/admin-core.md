---
title: Admin core resource API
description: Index of user, organization, authorization, content, files, notifications, jobs, configuration, and audit APIs.
---

# Admin core resource API

Use the [OpenAPI YAML](/openapi.yaml) for exact schemas, permission codes, and status codes.

| Resource     | Core routes                         | Purpose                                                               |
| ------------ | ----------------------------------- | --------------------------------------------------------------------- |
| Users        | `/users`, `/users/{id}`             | Query, create, update, enable/disable, unlock, reset, roles, sessions |
| Tenants      | `/tenants`, `/tenants/{id}/members` | Tenant profile and member lifecycle                                   |
| Organization | `/org/units`, `/org/positions`      | Tree moves, positions, assignments                                    |
| Roles        | `/roles/{id}/permissions`, `/menus` | Separate permissions, menus, and data scopes                          |
| Sessions     | `/online-sessions`                  | Query and revoke authorized sessions                                  |

List and detail operations apply tenant and data-scope filters in the server and SQL layer.

App-scoped resources include `/apps`, `/apps/{app_id}/users`, App content, notices/messages, and mobile releases. State transitions use explicit command routes; clients do not write persistent status directly.

| Platform area | Prefix                                                         | Security focus                                                     |
| ------------- | -------------------------------------------------------------- | ------------------------------------------------------------------ |
| Files         | `/files`                                                       | Random object keys, upload sessions, scan status, tenant isolation |
| Notifications | `/notification-templates`, `/notification-deliveries`          | Template variables, encrypted targets, retry classification        |
| Jobs          | `/job-handlers`, `/job-schedules`                              | Compile-time handlers only, no arbitrary code                      |
| Configuration | `/configs`                                                     | Encrypted secrets, no plaintext in list responses                  |
| Dictionaries  | `/dict-types`, `/dictionaries/{code}`                          | Backend-enforced extension policy                                  |
| API clients   | `/api-clients`                                                 | Secret shown once, hash stored                                     |
| Webhooks      | `/webhooks`                                                    | HMAC, replay defense, SSRF-safe URLs                               |
| Audit         | `/audit/operations`, `/audit/logins`, `/audit/security-events` | Field redaction and immutable evidence                             |
| Operations    | `/ops/health`, `/ops/runtime-summary`                          | Runtime status and compile-time module catalog                     |

`GET /internal/v1/health/live` checks the process. `GET /internal/v1/health/ready` returns `503` when a required dependency is unavailable.
