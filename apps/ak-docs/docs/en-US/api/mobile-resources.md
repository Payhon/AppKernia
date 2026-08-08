---
title: Mobile user and public resources
description: Profile, sessions, notifications, content, regions, and public configuration APIs.
---

# Mobile user and public resources

Public routes include `/public/config`, `/public/legal/{document_type}`, `/public/pages/{slug}`, `/public/app-version`, `/public/dictionaries/{code}`, and `/regions`.

Authenticated self-service routes:

| Method          | Path                  | Purpose                                            |
| --------------- | --------------------- | -------------------------------------------------- |
| `GET` / `PATCH` | `/me`                 | Read or update display name, locale, and time zone |
| `GET` / `PATCH` | `/me/preferences`     | Locale, appearance, notification preferences       |
| `GET`           | `/me/sessions`        | Own Mobile sessions                                |
| `DELETE`        | `/me/sessions/{id}`   | Revoke an own session                              |
| `GET`           | `/me/devices`         | Own devices                                        |
| `DELETE`        | `/me/devices/{id}`    | Remove device and related sessions                 |
| `GET`           | `/me/login-events`    | Redacted login history                             |
| `GET`           | `/me/security-events` | Security events related to the user                |

```bash
curl -X PATCH 'http://localhost:8080/api/v1/me' \
  -H 'Authorization: Bearer YOUR_ACCESS_TOKEN' \
  -H 'Content-Type: application/json' \
  -H 'Accept-Language: en-US' \
  -H 'X-AppID: YOUR_APP_UUID' \
  -d '{"display_name":"AK Developer","locale":"en-US","time_zone":"America/Los_Angeles"}'
```

Notifications use an opaque cursor. The server confirms read state. Articles and categories return published content in the current App/tenant scope. Bookmarks use `PUT` and `DELETE /me/article-bookmarks/{article_id}`.
