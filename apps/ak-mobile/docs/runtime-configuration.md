# Runtime API configuration

Development defaults to `http://127.0.0.1:8080/api/v1` for the local simulator.
Android physical devices cannot reach the host loopback address: set `akRuntime.apiBaseUrl` to a host-reachable development address before installing. Production configuration remains HTTPS-only.
