---
title: Authentication and sessions
description: AppKernia tokens, rotation, sessions, and client storage boundaries.
---

# Authentication and sessions

| Token         | Purpose                     | Client storage                                             | Server storage                  |
| ------------- | --------------------------- | ---------------------------------------------------------- | ------------------------------- |
| Access token  | Short-lived API identity    | Memory only in Admin and Mobile                            | Not a long-lived session record |
| Refresh token | Rotating session credential | Admin Secure HttpOnly cookie; Mobile system secure storage | SHA-256 hash only               |

Access tokens use Ed25519/EdDSA, expire after about 15 minutes by default, and isolate `ak-mobile`, `ak-admin`, and `ak-api` audiences.

## How sign-in creates a session

<div className="ak-diagram" role="group" aria-label="AppKernia sign-in and session creation flow">

```mermaid
sequenceDiagram
  accTitle: AppKernia sign-in and session creation flow
  accDescr: A client signs in through the matching API surface, the server verifies the app, user, password, and audience, stores a session and refresh-token hash, then returns a short-lived access token and one-time refresh token.
  participant C as Mobile / Admin client
  participant A as Auth API
  participant D as PostgreSQL
  C->>A: credentials + app context + intended audience
  A->>D: load active user and membership
  D-->>A: password hash + tenant membership
  A->>A: verify password, status, audience, risk policy
  A->>D: create session + SHA-256 refresh hash
  D-->>A: committed session family
  A-->>C: short-lived access + one-time refresh
```

</div>

<p className="ak-diagram-summary">The sign-in entry point first establishes the Mobile or Admin audience. Only after verifying account, app, and tenant membership does the server create a session family; plaintext passwords and refresh tokens never enter the database or logs.</p>

## Refresh rotation

<div className="ak-diagram" role="group" aria-label="AppKernia refresh-token rotation and replay detection">

```mermaid
sequenceDiagram
  accTitle: AppKernia refresh-token rotation and replay detection
  accDescr: The server locks and verifies a refresh-token hash; an active token is consumed and replaced, while reuse of a consumed token revokes the entire session family and records a security event.
  participant C as Mobile / Admin client
  participant A as AK API
  participant D as PostgreSQL
  C->>A: old refresh token + expected audience
  A->>D: SELECT FOR UPDATE + verify token hash
  alt token is active and session is valid
    D-->>A: active token and session family
    A->>D: consume old + insert new hash atomically
    A-->>C: new access + one-time refresh token
  else token was consumed, revoked, or mismatched
    D-->>A: reuse / revoked / audience mismatch
    A->>D: revoke session family + security event
    A-->>C: stable unauthorized error, sign in again
  end
```

</div>

<p className="ak-diagram-summary">Every refresh atomically consumes the old token and creates a new one. Only one concurrent use can succeed; subsequent reuse revokes the full session chain.</p>

Only one concurrent use of an old token can succeed. Reuse of an already consumed token revokes the session family and creates a security event.

## Where a session ends

<div className="ak-diagram" role="group" aria-label="AppKernia session invalidation and client cleanup flow">

```mermaid
flowchart LR
  accTitle: AppKernia session invalidation and client cleanup flow
  accDescr: Sign-out, device revocation, password or MFA security events, refresh replay, and expiry invalidate a session; the client then clears credentials, protected cache, and user context.
  Active["Active session"] --> Logout["User signs out"]
  Active --> AdminRevoke["User or admin revokes device"]
  Active --> Risk["Password / MFA / risk event"]
  Active --> Replay["Refresh replay detected"]
  Active --> Expired["Absolute or idle expiry"]
  Logout --> Revoked["Session revoked"]
  AdminRevoke --> Revoked
  Risk --> Revoked
  Replay --> Revoked
  Expired --> Revoked
  Revoked --> Clear["Clear access, refresh, protected cache"]
  Clear --> SignIn["Return to sign-in or recovery"]
```

</div>

<p className="ak-diagram-summary">Session validity is a server fact. After a stable unauthorized result, clients clear local identity and tenant-scoped cache rather than relying on a hidden page or stale permission snapshot.</p>

- 401 refresh is single-flight and retries the original request at most once.
- 403 never triggers refresh.
- Writes without idempotency are not replayed automatically.
- Logout, tenant switch, and session invalidation clear protected caches.
- Admin refresh tokens use only a Secure HttpOnly cookie; Mobile refresh tokens use only system secure storage.
- Access tokens never enter localStorage, sessionStorage, IndexedDB, or ordinary `uni` storage.
