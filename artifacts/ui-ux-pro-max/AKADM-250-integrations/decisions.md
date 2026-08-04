# AKADM-250 Design Decisions

- Reuse AppKernia's existing system typeface and navy/blue Admin tokens; do not load the suggested Google fonts or introduce orange marketing CTAs.
- API Client and Webhook remain separate static routes because their permissions and security consequences differ.
- A newly generated client secret is shown in a blocking one-time dialog. Closing requires an explicit saved acknowledgement; the value is never placed in URL, storage, logs, screenshots, or later API responses.
- CIDRs and permissions are edited in a constrained form. Machine credentials are always `ak-api` audience and cannot become Admin sessions.
- Webhook endpoint validation is server authoritative. UI explains that loopback, private/link-local/multicast and credential-bearing URLs are rejected.
- Delivery history displays bounded response/error summaries, event ID, attempts and status; it never renders raw HTML or secrets.
- Test delivery is confirmed and uses an idempotency key. Development uses an explicit local Mock Adapter when external delivery is unavailable.
