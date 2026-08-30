---
title: Security model
description: Risks AppKernia rejects by default and the boundaries every integration must preserve.
---

# Security model

- Never commit secrets, tokens, passwords, OTPs, MFA secrets, private keys, or real third-party credentials.
- Never disable TLS verification; production Mobile APIs do not accept HTTP.
- The server stores refresh-token hashes only; Mobile uses system secure storage and Admin uses a Secure HttpOnly cookie.
- The server generates random object keys and re-checks tenant and reference authorization on download.
- V1 does not fetch arbitrary remote URLs, reducing SSRF exposure.
- Schedules invoke compile-time registered handlers, never Shell, SQL, or source code stored in the database.
- Modules form a compile-time catalog; unknown binary upload or execution is not supported.
- Audit and logs redact fields and never record Authorization, cookies, or presigned URLs.
- Scan content is never uploaded, persisted, or logged. Only absolute HTTPS URLs matching the runtime allowlist enter the built-in bridge-free WebView; an out-of-scope redirect closes it. See the [Scanner capability](../mobile-components/scanner).

Use GitHub Private Vulnerability Reporting or a Security Advisory. Never publish exploit details or real data in a public issue. See the [security policy](../community/security).
