# AKADM-250 Review Checklist

- [x] Secret is returned once, acknowledged, and never persisted or captured in a screenshot/artifact.
- [x] API Client status, expiry, CIDR and permission contracts are enforced server-side.
- [x] Machine credentials cannot authenticate to `ak-admin`.
- [x] Webhook URL validation rejects SSRF targets and DNS rebinding cases.
- [x] HMAC timestamp/event ID contract and idempotent test delivery are visible.
- [x] Delivery response/error content is bounded and safely rendered.
- [x] All strings have complete `zh-CN` and `en-US` keys.
- [x] Loading, empty, error, conflict and permission states are present.
- [x] Keyboard, axe and 375/768/1024/1440 screenshots are verified.
