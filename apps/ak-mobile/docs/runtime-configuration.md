# Runtime API configuration

Development defaults to `http://127.0.0.1:8080/api/v1` for the local simulator.
Android physical devices cannot reach the host loopback address: set `akRuntime.apiBaseUrl` to a host-reachable development address before installing. Production configuration remains HTTPS-only.

`akRuntime.appId` is a packaged, public UUID used as `X-AppID` on every Mobile API and protected asset request. The development build uses the seeded local App `00000000-0000-4000-8000-000000000001`; a production build must replace it with the UUID created in App 管理. It is an application scope selector, never a credential.
