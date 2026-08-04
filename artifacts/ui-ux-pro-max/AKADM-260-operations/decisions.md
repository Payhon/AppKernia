# AKADM-260 Design Decisions

- Reuse the established navy/blue Admin tokens, system typography, cards and semantic status colors; do not introduce neon, Google Fonts, marketing CTAs or scroll-snap.
- Access rules use server-generated subject hints only. Raw IP, CIDR, account, device or identifier values are accepted only in the create form and are never returned by list/update responses.
- Creation and revocation show the affected subject type, scope, action, start/expiry and an explicit acknowledgement before submission.
- Mobile access-rule tables collapse to subject/action controls; secondary metadata moves into the primary cell.
- Service status uses text plus color for every state. `not_configured` and `unknown` are distinct from `degraded`/`unavailable` and must not be shown as passed.
- Diagnostic copy contains only API/runtime versions, status, latency buckets/counts and timestamps; it excludes environment values, paths, connection strings and secrets.
- Refresh is user-triggered and query-backed; no aggressive polling or animation is introduced.
