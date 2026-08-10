-- name: GetMobilePreferences :one
SELECT COALESCE(p.locale, u.locale) AS locale,
       COALESCE(p.appearance, 'system') AS appearance,
       COALESCE(p.notification_preferences, '{"in_app":true,"push":false,"email":true}'::jsonb) AS notification_preferences
FROM iam.users u
LEFT JOIN iam.user_preferences p ON p.user_id = u.id AND p.app_id = sqlc.arg(app_id)
WHERE u.id = sqlc.arg(user_id) AND u.deleted_at IS NULL;

-- name: UpsertMobilePreferences :exec
INSERT INTO iam.user_preferences (app_id, user_id, locale, appearance, notification_preferences)
VALUES (sqlc.arg(app_id), sqlc.arg(user_id), sqlc.arg(locale), sqlc.arg(appearance), sqlc.arg(notification_preferences))
ON CONFLICT (app_id, user_id) DO UPDATE
SET locale = EXCLUDED.locale,
    appearance = EXCLUDED.appearance,
    notification_preferences = EXCLUDED.notification_preferences;

-- name: CountMobileUnreadNotifications :one
SELECT count(*)
FROM notify.recipients r
JOIN notify.messages m ON m.tenant_id = r.tenant_id AND m.id = r.message_id
WHERE r.tenant_id = sqlc.arg(tenant_id)
  AND r.app_id = sqlc.arg(app_id)
  AND r.user_id = sqlc.arg(user_id)
  AND r.delivery_status = 'delivered'
  AND r.read_at IS NULL
  AND r.archived_at IS NULL
  AND m.status = 'published'
  AND m.app_id = sqlc.arg(app_id)
  AND m.deleted_at IS NULL
  AND (m.expires_at IS NULL OR m.expires_at > now());

-- name: ListMobileLoginEvents :many
SELECT id, auth_method, result, occurred_at, client_ip
FROM audit.login_events
WHERE user_id = sqlc.arg(user_id) AND app_id = sqlc.arg(app_id) AND audience = 'ak-mobile'
ORDER BY occurred_at DESC, id DESC
LIMIT 100;

-- name: ListMobileSecurityEvents :many
SELECT id, event_type, severity, occurred_at
FROM audit.security_events
WHERE user_id = sqlc.arg(user_id) AND app_id = sqlc.arg(app_id)
ORDER BY occurred_at DESC, id DESC
LIMIT 100;

-- name: ListMobileNotifications :many
SELECT m.id, m.title, m.body, m.body_format, m.message_type, m.created_at, r.read_at
FROM notify.recipients r
JOIN notify.messages m ON m.tenant_id = r.tenant_id AND m.id = r.message_id
WHERE r.tenant_id = sqlc.arg(tenant_id)
  AND r.app_id = sqlc.arg(app_id)
  AND r.user_id = sqlc.arg(user_id)
  AND r.archived_at IS NULL
  AND r.delivery_status = 'delivered'
  AND m.status = 'published'
  AND m.app_id = sqlc.arg(app_id)
  AND m.deleted_at IS NULL
  AND (m.expires_at IS NULL OR m.expires_at > now())
  AND (m.created_at, m.id) < (sqlc.arg(cursor_created_at), sqlc.arg(cursor_id)::uuid)
ORDER BY m.created_at DESC, m.id DESC
LIMIT sqlc.arg(page_limit);

-- name: MarkMobileNotificationRead :execrows
UPDATE notify.recipients
SET read_at = COALESCE(read_at, now())
WHERE tenant_id = sqlc.arg(tenant_id)
  AND app_id = sqlc.arg(app_id)
  AND user_id = sqlc.arg(user_id)
  AND message_id = sqlc.arg(message_id)
  AND delivery_status = 'delivered'
  AND archived_at IS NULL;
