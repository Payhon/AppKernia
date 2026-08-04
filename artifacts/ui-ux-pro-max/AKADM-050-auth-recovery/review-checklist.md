# AKADM-050 Review Checklist

- [x] Repository-local `ui-ux-pro-max` commands ran and real output is recorded.
- [x] Backend feature flags and anonymous-only routes fail closed.
- [x] Registration agreement is explicit and not preselected.
- [x] Forgot-password response is identical for known and unknown accounts.
- [x] Recovery token is one-time, hashed at rest, short-lived and not logged/persisted by Admin.
- [x] Reset revokes existing sessions and refresh families transactionally.
- [x] All visible labels and errors use matching `zh-CN` / `en-US` keys.
- [x] Password-manager autocomplete, keyboard flow, contrast and live announcements pass.
- [x] PostgreSQL integration and Docker API probes pass.
- [x] Both-locale axe and responsive screenshots pass.
