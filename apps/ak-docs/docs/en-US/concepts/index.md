---
title: Core concepts
description: Understand AppKernia architecture, authentication, authorization, tenancy, i18n, and security.
---

# Core concepts

AppKernia's value comes from shared boundaries, not only from a list of technologies.

## The three-surface contract chain

<div className="ak-diagram" role="group" aria-label="AppKernia three-surface contract chain">

```mermaid
flowchart TD
  accTitle: AppKernia three-surface contract chain
  accDescr: Database migrations and sqlc queries feed the Go server implementation, OpenAPI generates Admin and Mobile clients, and both page layers verify the result.
  DB["PostgreSQL migrations and sqlc"] --> Go["Go route / application / repository"]
  Go --> Spec["OpenAPI 3.1 source of truth"]
  Spec --> AdminClient["Generated Admin client"]
  Spec --> MobileClient["Generated Mobile client"]
  AdminClient --> AdminPage["React pages and integration tests"]
  MobileClient --> MobilePage["UTS / UVue pages and contract tests"]
```

</div>

<p className="ak-diagram-summary">An API change starts with durable data and server rules, moves through OpenAPI into two generated clients, then reaches Admin and Mobile pages and tests.</p>

An API change is never only a Handler edit. When fields, permissions, or data scope change, the server implementation, OpenAPI, database, permission seed, generated clients, and integration tests move together. A temporary hand-written DTO can compile while still creating runtime drift.

## Responsibility boundaries

| Layer        | Owns                                                     | Must not own                                       |
| ------------ | -------------------------------------------------------- | -------------------------------------------------- |
| Mobile       | User tasks, device state, offline/platform UX            | Trusting client-supplied tenant, role, or user     |
| Admin        | Operations workflows, forms, menus, data-scope UX        | Replacing server authorization with hidden buttons |
| API / Worker | Authorization, idempotency, audit, jobs, business rules  | Delegating security decisions to clients           |
| PostgreSQL   | Constraints, transactions, tenant filters, durable facts | Unvalidated dynamic execution                      |

## How one request moves

<div className="ak-diagram" role="group" aria-label="AppKernia protected request data flow">

```mermaid
sequenceDiagram
  accTitle: AppKernia protected request data flow
  accDescr: A client sends locale and identity context, the API verifies session and tenant, the application enforces rules, the repository filters in SQL, and the response carries stable codes and language.
  participant C as Mobile / Admin
  participant A as API middleware
  participant U as Application use case
  participant R as Repository / sqlc
  participant D as PostgreSQL
  C->>A: Accept-Language + audience credential
  A->>A: resolve locale, session, app, tenant
  A->>U: verified subject + permission + input
  U->>R: business operation + data scope
  R->>D: tenant-filtered SQL / transaction
  D-->>R: durable result
  R-->>U: domain result
  U-->>A: stable code + audit intent
  A-->>C: response + Content-Language + Request ID
```

</div>

<p className="ak-diagram-summary">The client supplies request context, but trusted user, tenant, permission, and data scope are resolved on the server; SQL enforces isolation and the client branches only on stable status.</p>

1. The client sends `Accept-Language`; Mobile also sends public `X-AppID`, and protected calls use credentials for the correct audience.
2. The API resolves locale, session, app, and tenant, then derives permission and data scope from verified server context.
3. The application layer enforces business rules; repository/sqlc code applies transactions and tenant filters in SQL.
4. Responses use stable business codes and `Content-Language`; clients branch on codes rather than localized copy.
5. Relevant writes produce audit or security events, while asynchronous work moves to the Worker.

- [Architecture](./architecture)
- [Authentication and sessions](./authentication)
- [Authorization and multi-tenancy](./permissions-tenancy)
- [Internationalization](./internationalization)
- [Security model](./security)
