---
title: Notification operations
description: Observe notification delivery and safely handle failures through overview, runs, queue tasks, and failure center views.
---

# Notification operations

The Admin route `/system/notifications/operations` provides an app-scoped view of notification execution. It exposes only tenant/App-scoped, sanitized business projections—not River Args, device tokens, full payloads, credentials, stacks, or raw provider responses.

## Four URL-restorable tabs

| Tab            | Purpose                                                                                               |
| -------------- | ----------------------------------------------------------------------------------------------------- |
| Overview       | Volume, provider acceptance, failures, invalid tokens, skips, open rate, queue depth, and P95 latency |
| Message runs   | Scheduling, publishing, audience freeze, fanout, and per-device delivery stages                       |
| Queue tasks    | Task kind, state, attempts, next run, duration, related resource, and safe error                      |
| Failure center | Terminal failures with single and controlled batch retry                                              |

Tabs, filters, and pagination are encoded in the URL. The page polls every 15 seconds only while it is visible and unfinished work exists; polling pauses when the browser tab is hidden, while manual refresh remains available.

## Run states

A message run can be `scheduled`, `queued`, `running`, `completed`, `completed_with_failures`, `failed`, `cancelled`, or `expired`.

A task can be `scheduled`, `queued`, `running`, `retry_wait`, `succeeded`, `failed`, or `cancelled`. River remains the live scheduling source of truth, while the workbench reads tenant/App-scoped business projections. A reconciliation job repairs projection drift caused by abrupt worker exits.

## Safe retry rules

1. Retry only terminal tasks for which the server returns `retryable=true`.
2. `transient` and `throttled` tasks can be retried after automatic attempts are exhausted.
3. For `auth_config_error`, repair the [push channel](./push-channels) and pass preflight first.
4. `unknown_after_write` may already have reached the provider. It permits only a single-item retry with explicit duplicate-risk acknowledgement.
5. A batch contains at most 100 items; the server returns an accept/reject decision for each item.
6. A retry creates a new task linked to the original. It never overwrites the original run or attempt history.

Running tasks, future scheduled tasks, cancelled messages, and expired messages cannot be retried manually. The workbench does not edit task arguments, force-stop a running task, or directly update River records.

## Recommended investigation order

1. In Overview, check whether queue depth, oldest wait, P95 queue latency, or fault counts keep increasing.
2. In Message runs, find whether the pipeline stopped at publish, fanout, or device delivery, then compare recipient, evaluated, delivery, and skip counts.
3. In Queue tasks, inspect task kind, attempt count, next retry, normalized error code, and Trace ID.
4. Repair a provider authentication failure before replaying delivery work.
5. Use the Trace ID in OpenTelemetry, Loki, or Tempo for a full trace. Admin retains only a sanitized summary.

## Permissions

| Permission                  | Purpose                                              |
| --------------------------- | ---------------------------------------------------- |
| `notify.observability.read` | Read overview, runs, tasks, and safe error summaries |
| `notify.task.retry`         | Retry notification tasks                             |
| `notify.delivery.read`      | Read delivery records                                |
| `notify.delivery.retry`     | Compatibility permission for delivery retry          |
| `notify.operations.publish` | Publish `news_operations` notifications              |

Button visibility is not authorization. SQL queries always filter both tenant and app. Retry operations audit the operator, original task, new task, and decision.

## Retention and monitoring

Task, attempt, and message-run details are retained for 90 days; daily aggregates remain for 13 months. Cleanup removes only terminal data that has completed aggregation, never `scheduled`, `queued`, `running`, or `retry_wait` records.

Alerts should cover sustained queue depth and oldest-wait growth, permanent failures, retry spikes, provider latency, invalid-token rate, repeated authentication failures, and pipeline duration. Metric labels must not contain user IDs, tokens, Trace IDs, or message bodies.

Read [Notification and push architecture](../concepts/notification-architecture) for the publish/fanout/delivery flow and [Notification API](../api/mobile-notifications) for service integration.
