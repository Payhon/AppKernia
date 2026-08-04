# Online Sessions Page Override

This page inherits `../MASTER.md` and overrides only the following rules.

- Use a compact URL-driven filter row and server-paginated table. User identity, audience/platform, safe device/IP hints, last activity, expiry and status are the primary scan fields.
- Render current session with a text badge and persistent explanatory warning; never depend on row color alone.
- Force logout is a danger action shown only with `iam.session.revoke`. The confirmation names the safe user/device hints and impact. Current-session confirmation adds an explicit sign-out warning.
- After revoking a non-current session, invalidate the list and announce success. After revoking the current session, clear in-memory authentication and navigate to login.
- Use only safe, server-returned hints. Do not expose token hashes, raw token data, full user-agent strings or hidden device metadata.
- At 375px, hide low-priority columns while keeping identity, platform, status/time and the row action keyboard reachable. Horizontal scrolling stays inside the card.
