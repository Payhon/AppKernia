# System Schedules Page Override

Inherits `../MASTER.md`. The following rules override it for `system.integrations.schedules`.

## Information hierarchy

- Header: page title, concise safety description and permission-gated create action.
- Search panel: name/handler query, status, IANA time zone, reset and search.
- Table: name/code, registered handler, Cron, time zone, next run, status and permission-gated actions.
- Edit Drawer: identity, registered handler, Cron/time-zone preview, JSON payload and bounded execution policies.
- Run-history Drawer: trigger, status, scheduled/started/finished times, attempt and safe error code/summary.

## Interaction and safety

- `handler_key` is never a free-text command field; options originate from the backend compiled registry.
- Cron/time-zone validation happens before save and produces five future instants. Offset changes remain visible for DST review.
- Pause/resume and manual execute use confirmation dialogs; manual execution reports the new run identifier and queued state.
- JSON payload is parsed as data only. The UI does not evaluate scripts, templates, SQL or commands.
- Non-idempotent mutations are not automatically replayed after authentication refresh.

## Responsive and accessibility

- At 375px the search panel and Drawer fields use one column; table overflow is bounded and keyboard focusable.
- At 768px and above policy fields use a two-column grid where space allows.
- Every input has a persistent label/id pair; validation uses field help plus `role=alert`.
- Loading and mutations expose status; visible focus and reduced motion inherit the Master.
