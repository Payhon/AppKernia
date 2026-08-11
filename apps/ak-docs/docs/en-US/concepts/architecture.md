---
title: Architecture
description: AppKernia's three-surface modular architecture.
---

# Architecture

AppKernia is a three-surface system where one modular-monolith server drives two client classes. Admin and Mobile share durable business facts and an OpenAPI contract while keeping separate route prefixes, token audiences, interaction models, and local-storage policies.

<div className="ak-diagram" role="group" aria-label="AppKernia overall three-surface architecture">

```mermaid
flowchart TB
  accTitle: AppKernia overall three-surface architecture
  accDescr: Mobile enters through the App API and Admin through the Admin API; both reach one modular server backed by PostgreSQL, object storage, queues, external-event adapters, and observability.
  subgraph Clients["Client product surfaces"]
    direction LR
    User["App user"] --> Mobile["AK Mobile<br/>uni-app x / UTS / UVue"]
    Operator["Operator / developer"] --> Admin["AK Admin<br/>React / Vite / TanStack"]
  end
  Mobile --> AppBoundary["/api/v1<br/>ak-mobile audience"]
  Admin --> AdminBoundary["/admin-api/v1<br/>ak-admin audience"]
  AppBoundary --> API["ak-api<br/>GoFrame HTTP boundary"]
  AdminBoundary --> API
  API --> Modules["Modular application and domain"]
  subgraph Foundations["Durable and asynchronous infrastructure"]
    direction LR
    PG[("PostgreSQL 18")]
    Storage["S3-compatible<br/>local adapter"]
    River["River internal jobs"]
    Outbox["Transactional outbox"]
  end
  Modules --> PG
  Modules --> Storage
  Modules --> River
  Modules --> Outbox
  River --> Worker["ak-worker"]
  Outbox --> Integrations["Webhook / external-message adapters"]
  API --> OTel["OpenTelemetry"]
  Worker --> OTel
```

</div>

<p className="ak-diagram-summary">Two client classes enter distinct security boundaries, but the same server modules, transactions, and data constraints own business facts. Worker, outbox, and OpenTelemetry complete the asynchronous and operational path.</p>

## Admin / Web architecture

<div className="ak-diagram" role="group" aria-label="AK Admin Web architecture">

```mermaid
flowchart TB
  accTitle: AK Admin Web architecture
  accDescr: A React App Shell organizes static routes, dynamic menus, session state, queries, forms, and UI; a generated client calls only the Admin API and the server remains the final authorization boundary.
  Browser["Browser"] --> Shell["React App Shell"]
  subgraph SPA["AK Admin SPA"]
    direction TB
    Shell --> Router["TanStack Router<br/>static route registry"]
    Shell --> Session["Auth session manager<br/>access token in memory"]
    Shell --> Query["TanStack Query<br/>server state"]
    Shell --> Store["Zustand<br/>theme / locale / shell state"]
    Router --> Features["Feature pages and route guards"]
    Features --> Forms["RHF + Zod"]
    Features --> UI["Ant Design + AK UI"]
    Query --> Client["Generated OpenAPI client"]
  end
  Client -->|"/admin-api/v1"| AdminAPI["Go Admin API"]
  AdminAPI --> Authz["Server authorization and SQL data scope"]
  AdminAPI --> Context["GET /auth/context<br/>menus / permissions / tenants"]
  Context --> Router
```

</div>

<p className="ak-diagram-summary">Admin routes exist statically at build time and server menus reference only allowed registry keys. TanStack Query owns server data while Zustand owns shell state; buttons and route guards improve UX but never replace Go API authorization.</p>

## Server architecture

<div className="ak-diagram" role="group" aria-label="AK Server modular-monolith architecture">

```mermaid
flowchart TB
  accTitle: AK Server modular-monolith architecture
  accDescr: API, Worker, and CLI enter shared application and domain modules; transport depends on application, application depends on domain and ports, and infrastructure adapters implement ports for databases, storage, and external systems.
  subgraph Entry["Processes and entry points"]
    API["ak-api<br/>HTTP / WebSocket"]
    Worker["ak-worker<br/>River / outbox"]
    CLI["ak-cli<br/>migrate / seed / generate"]
  end
  API --> Transport["Transport<br/>parse / validate / respond"]
  Transport --> Application["Application<br/>use case / transaction / idempotency"]
  Worker --> Application
  CLI --> Application
  Application --> Domain["Domain<br/>entities / rules / state machines"]
  Application --> Ports["Application and domain ports"]
  Infra["Infrastructure adapters<br/>pgx / sqlc / S3 / providers"] -.->|"implements"| Ports
  subgraph Backends["Infrastructure targets"]
    direction LR
    PG[("PostgreSQL 18")]
    Objects["S3-compatible storage"]
    External["Notification / OAuth / webhook"]
  end
  Infra --> PG
  Infra --> Objects
  Infra --> External
  subgraph Effects["Trusted side effects"]
    direction LR
    Audit["Audit and security events"]
    Jobs["River / outbox enqueue"]
  end
  Application --> Audit
  Application --> Jobs
```

</div>

<p className="ak-diagram-summary">The modular monolith preserves one transactional deployment boundary while Application, Domain, and ports isolate business logic from infrastructure. Auditable PostgreSQL/sqlc code retains critical tenant filters, locks, and constraints.</p>

## Mobile architecture

<div className="ak-diagram" role="group" aria-label="AK Mobile cross-platform layered architecture">

```mermaid
flowchart TB
  accTitle: AK Mobile cross-platform layered architecture
  accDescr: UVue pages use AK UI and call Presentation and Application layers; use cases reach network, secure storage, push, and device capabilities through ports, with Android, iOS, and HarmonyOS differences isolated in adapters.
  Pages["pages/*.uvue<br/>feature page"] --> Presentation["Presentation<br/>view state / navigation"]
  Presentation --> Application["Application use cases<br/>sign-in / refresh / profile / tenant"]
  Application --> Repositories["Repository / query cache<br/>tenant + subject scoped"]
  Application --> Ports["Core ports<br/>network / secure storage / push / OAuth / device"]
  Repositories --> Client["Generated UTS API client"]
  Client --> HTTP["AkHttpClient → /api/v1"]
  Ports --> Adapters["Platform adapters"]
  Adapters --> Android["Android"]
  Adapters --> IOS["iOS"]
  Adapters --> Harmony["HarmonyOS NEXT"]
  Pages --> AKUI["AK UI · ak-*"]
  AKUI --> UView["uView Ultra / uni native / adapter"]
```

</div>

<p className="ak-diagram-summary">Feature pages compose features and `ak-*` components; they do not build API URLs, read or write tokens, or call platform SDKs directly. Platform claims still require independent build, installation, and device evidence.</p>

## Responsibilities of the three surfaces

### AK Mobile

AK Mobile uses uni-app x, UTS/UVue, and VDOM. Feature pages reach platform and network capabilities through `ak-*` components, application use cases, and ports.

### AK Admin

AK Admin is a React SPA that only calls `/admin-api/v1`. Routes are statically compiled, menus are filtered by the server, and the Go API remains the final authorization boundary.

System remains a top-level data menu, but the Shell presents it through the bottom sidebar gear while the primary menu scrolls independently. The adjacent documentation icon opens a public, independently built OpenAPI page that neither reads Admin session credentials nor changes route and permission semantics. See [online OpenAPI reference and System menu](../api/online-reference).

### AK Server

AK Server uses GoFrame at the HTTP boundary and pgx/v5 plus sqlc for PostgreSQL. API, worker, and CLI share one codebase. River handles internal jobs and a transactional outbox handles external events.

`server/openapi/openapi.yaml` is the final API source of truth. A contract change updates routes, use cases, database/permissions/audit when relevant, generated clients, and tests together.

| Change                   | Check together                                                                   |
| ------------------------ | -------------------------------------------------------------------------------- |
| API field or response    | OpenAPI, Go implementation, both generated clients, contract tests               |
| Permission or data scope | Permission seed, backend middleware/application, SQL filter, denied-path tests   |
| User-visible message     | Stable error code, `zh-CN`/`en-US` catalogs, `Content-Language`, and UI          |
| Asynchronous side effect | Transaction boundary, River/outbox, idempotency, retry, audit, and observability |
