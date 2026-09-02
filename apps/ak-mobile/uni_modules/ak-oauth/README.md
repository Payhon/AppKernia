# ak-oauth

`ak-oauth` is the native authorization boundary for AppKernia. It never exchanges a provider credential for an App session locally; authorization codes, ID tokens and GitHub one-time tickets are returned to the typed Mobile repository and sent to AppKernia Backend over TLS.

Supported boundaries:

- Android/iOS WeChat: `uni.login` with `provider: "weixin"` and `onlyAuthorize: true`.
- iOS Apple: `AuthenticationServices` via the bundled Swift bridge. The bridge returns the authorization code and identity token for Backend verification. On the first authorization only, it also normalizes Apple `fullName` to at most 120 Unicode characters and sends it as optional display metadata; Backend never uses that value for subject matching or authorization.
- Android Google: AndroidX Credential Manager with `GetGoogleIdOption`, the exported server client ID and the Backend nonce. It tries authorized accounts first, retries with the full chooser only for `NoCredentialException`, and maps an explicit sheet cancellation to authorization denied. The capability is exposed only by an `android_google` build.
- Harmony WeChat: the DCloud uni-app x `uni.login` WeChat adapter with `onlyAuthorize: true`, backed by the fixed `@tencent/wechat_open_sdk` integration. The exported native overlay supplies the allowlisted `weixin` query scheme and `wxentity.action.open` action; a signed build and real WeChat credentials remain device gates.
- GitHub: system browser plus an HTTPS verified return link containing only `provider`, `flow_id` and `one_time_ticket`. OAuth `code` is handled by Backend's HTTPS browser callback and is never accepted by the App link parser.

No client secret, Apple private key, provider access token, refresh token or raw profile is stored by this module. The checked-in manifest templates intentionally contain no tenant host. `ak-cli app-login-provider export` writes the public build snapshot and platform allowlist; a signed native build and its domain-association files remain release gates.
