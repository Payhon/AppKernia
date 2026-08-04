# Decisions

- Reuse the Profile Security card language and place registered devices before active sessions, because removing a device affects one or more sessions.
- Show a textual `Current device` tag alongside platform, not a color-only indicator.
- Use the server device UUID as the React key and removal target.
- Show latest user agent, IP, last activity, first registration and active-session count; never expose the local device key, access tokens or refresh tokens.
- Require confirmation for every removal. Current-device copy explicitly states that the user will be signed out immediately.
- Device removal is a non-idempotent security write and is never automatically refreshed or replayed.
- Keep loading skeleton, guided empty state, recoverable load error, mutation success and mutation error visible in both languages.
