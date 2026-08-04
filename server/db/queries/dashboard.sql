-- name: DashboardUserSummary :one
SELECT count(*) AS total_users,
       count(*) FILTER (WHERE u.created_at >= sqlc.arg('start_at')) AS new_users
FROM iam.tenant_members tm
JOIN iam.users u ON u.id = tm.user_id AND u.deleted_at IS NULL
WHERE tm.tenant_id = sqlc.arg('tenant_id')
  AND tm.status = 'active';

-- name: DashboardActiveSessionCount :one
SELECT count(*)
FROM iam.sessions
WHERE tenant_id = sqlc.arg('tenant_id')
  AND status = 'active'
  AND revoked_at IS NULL
  AND absolute_expires_at > sqlc.arg('end_at');

-- name: DashboardFailedJobCount :one
SELECT count(*)
FROM jobs.schedule_runs run
JOIN jobs.schedules schedule ON schedule.id = run.schedule_id
WHERE schedule.tenant_id = sqlc.arg('tenant_id')
  AND run.status = 'failed'
  AND run.scheduled_at >= sqlc.arg('start_at')
  AND run.scheduled_at <= sqlc.arg('end_at');

-- name: DashboardOpenSecurityEventCount :one
SELECT count(*)
FROM audit.security_events
WHERE tenant_id = sqlc.arg('tenant_id')
  AND resolved_at IS NULL;

-- name: DashboardPublishedMessageCount :one
SELECT count(*)
FROM notify.messages
WHERE tenant_id = sqlc.arg('tenant_id')
  AND status = 'published'
  AND deleted_at IS NULL
  AND published_at >= sqlc.arg('start_at')
  AND published_at <= sqlc.arg('end_at');

-- name: DashboardLoginTrend :many
WITH days AS (
    SELECT generate_series(
        date_trunc('day', sqlc.arg('start_at')::timestamptz),
        date_trunc('day', sqlc.arg('end_at')::timestamptz),
        interval '1 day'
    ) AS day
)
SELECT days.day::timestamptz AS day,
       count(event.id) FILTER (WHERE event.result = 'success') AS success_count,
       count(event.id) FILTER (WHERE event.result IN ('failure', 'blocked')) AS failure_count
FROM days
LEFT JOIN audit.login_events event
  ON event.tenant_id = sqlc.arg('tenant_id')
 AND event.audience = 'ak-admin'
 AND event.occurred_at >= days.day
 AND event.occurred_at < days.day + interval '1 day'
GROUP BY days.day
ORDER BY days.day;

-- name: DashboardUserTrend :many
WITH days AS (
    SELECT generate_series(
        date_trunc('day', sqlc.arg('start_at')::timestamptz),
        date_trunc('day', sqlc.arg('end_at')::timestamptz),
        interval '1 day'
    ) AS day
)
SELECT days.day::timestamptz AS day, count(DISTINCT u.id) AS value
FROM days
LEFT JOIN iam.tenant_members tm ON tm.tenant_id = sqlc.arg('tenant_id') AND tm.status = 'active'
LEFT JOIN iam.users u
  ON u.id = tm.user_id
 AND u.deleted_at IS NULL
 AND u.created_at >= days.day
 AND u.created_at < days.day + interval '1 day'
GROUP BY days.day
ORDER BY days.day;

-- name: DashboardFailedJobTrend :many
WITH days AS (
    SELECT generate_series(
        date_trunc('day', sqlc.arg('start_at')::timestamptz),
        date_trunc('day', sqlc.arg('end_at')::timestamptz),
        interval '1 day'
    ) AS day
)
SELECT days.day::timestamptz AS day, count(run.id) AS value
FROM days
LEFT JOIN jobs.schedules schedule ON schedule.tenant_id = sqlc.arg('tenant_id')
LEFT JOIN jobs.schedule_runs run
  ON run.schedule_id = schedule.id
 AND run.status = 'failed'
 AND run.scheduled_at >= days.day
 AND run.scheduled_at < days.day + interval '1 day'
GROUP BY days.day
ORDER BY days.day;

-- name: DashboardSecurityTrend :many
WITH days AS (
    SELECT generate_series(
        date_trunc('day', sqlc.arg('start_at')::timestamptz),
        date_trunc('day', sqlc.arg('end_at')::timestamptz),
        interval '1 day'
    ) AS day
)
SELECT days.day::timestamptz AS day, count(event.id) AS value
FROM days
LEFT JOIN audit.security_events event
  ON event.tenant_id = sqlc.arg('tenant_id')
 AND event.occurred_at >= days.day
 AND event.occurred_at < days.day + interval '1 day'
GROUP BY days.day
ORDER BY days.day;

-- name: DashboardRecentOperations :many
SELECT id, module_code, action_name, resource_type, succeeded, error_code, occurred_at
FROM audit.operation_logs
WHERE tenant_id = sqlc.arg('tenant_id')
  AND occurred_at >= sqlc.arg('start_at')
  AND occurred_at <= sqlc.arg('end_at')
ORDER BY occurred_at DESC, id DESC
LIMIT 10;

-- name: DashboardRecentFailedJobs :many
SELECT run.id, schedule.code AS schedule_code, schedule.name AS schedule_name,
       run.error_code, run.scheduled_at
FROM jobs.schedule_runs run
JOIN jobs.schedules schedule ON schedule.id = run.schedule_id
WHERE schedule.tenant_id = sqlc.arg('tenant_id')
  AND run.status = 'failed'
  AND run.scheduled_at >= sqlc.arg('start_at')
  AND run.scheduled_at <= sqlc.arg('end_at')
ORDER BY run.scheduled_at DESC, run.id DESC
LIMIT 10;

-- name: DashboardRecentSecurityEvents :many
SELECT id, event_type, severity, source, occurred_at
FROM audit.security_events
WHERE tenant_id = sqlc.arg('tenant_id')
  AND occurred_at >= sqlc.arg('start_at')
  AND occurred_at <= sqlc.arg('end_at')
ORDER BY occurred_at DESC, id DESC
LIMIT 10;
