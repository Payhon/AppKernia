---
title: Server API
description: AppKernia API entry points, authentication boundaries, and OpenAPI source of truth.
---

# Server API

AppKernia describes its server contract with OpenAPI 3.1. These pages explain common routes, examples, and security boundaries; `server/openapi/openapi.yaml` in the current repository remains the final source for fields, enums, status codes, and schemas.

[Download the current OpenAPI YAML](/openapi.yaml)

| Surface      | Prefix          | Client                         | Identity boundary                          |
| ------------ | --------------- | ------------------------------ | ------------------------------------------ |
| App API      | `/api/v1`       | uni-app x and app users        | `ak-mobile` bearer token                   |
| Admin API    | `/admin-api/v1` | React Admin                    | `ak-admin` access token plus secure cookie |
| Internal API | `/internal/v1`  | Health and internal monitoring | Deployment network boundary                |

Admin and Mobile tokens are not interchangeable. `X-AppID` selects a public active app; tenant and user scope still come from the verified session.

```http
Accept: application/json
Accept-Language: en-US
X-Request-ID: 019…
X-AppID: 01900000-0000-7000-8000-000000000001
Authorization: Bearer YOUR_ACCESS_TOKEN
```

Start with [conventions](./conventions), [Mobile authentication](./mobile-auth), [Mobile resources](./mobile-resources), [Admin authentication](./admin-auth), or [Admin core resources](./admin-core).

<div class="ak-doc-callout"><strong>Version status</strong>The current API is 0.1.0 and the project has no stable release yet. Generate clients from OpenAPI and verify the schema hash instead of maintaining hand-written DTOs from this page.</div>
