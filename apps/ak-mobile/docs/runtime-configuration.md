# Runtime API configuration

Development defaults to `http://127.0.0.1:8080/api/v1` for the local simulator.
Android physical devices cannot reach the host loopback address: set `akRuntime.apiBaseUrl` to a host-reachable development address before installing. Production configuration remains HTTPS-only.

`akRuntime.appId` is a packaged, public UUID used as `X-AppID` on every Mobile API and protected asset request. The development build uses the seeded local App `00000000-0000-4000-8000-000000000001`; a production build must replace it with the UUID created in App 管理. It is an application scope selector, never a credential.

## Packaged startup snapshot

The privacy gate intentionally runs before any network request. Its App icon, localized display name and subtitle are generated from the Admin-managed startup metadata at release time:

```bash
cd server
go run ./cmd/ak-cli app-startup export \
  --app-id 00000000-0000-4000-8000-000000000001 \
  --output ../apps/ak-mobile
```

The command writes `src/generated/startup-snapshot.uts` and the scanned icon under `static/app-startup/`. Run the drift check in release CI after generation:

```bash
cd server
go run ./cmd/ak-cli app-startup export \
  --app-id 00000000-0000-4000-8000-000000000001 \
  --output ../apps/ak-mobile \
  --check
```

Admin edits do not remotely change this first-install surface; a new App binary must include a fresh export. The post-consent onboarding flow is different: it reads only the current immutable published revision from public configuration and downloads the protected locale assets at runtime.
