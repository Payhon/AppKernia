# System Notifications Page Override

This page override extends `design-system/appkernia-admin/MASTER.md`.

- Four sibling routes share URL-backed status, type, channel, locale, and query filters.
- Message create/edit uses a labelled form plus a safe preview panel; at mobile width the preview follows the form.
- Audience selection is server-resolved. Display the exact count and only a bounded, non-sensitive user preview.
- Publish is a high-impact action: require a confirmation modal naming the message and exact recipient count.
- HTML preview must be sanitized with the same allowlist contract enforced by the API; external scripts, event handlers, iframes, forms, and unsafe URLs are removed.
- Delivery target is always a hint. The encrypted value and hash never enter the Admin response.
- Failed deliveries use an error status, a safe error-code/summary detail, and a confirmed retry button. Disable retry during the request.
- On narrow screens use stacked filters and horizontally scrollable tables; keep action buttons reachable and focus-visible.
