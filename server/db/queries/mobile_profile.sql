-- name: GetMobilePreferences :one
SELECT u.locale,
       COALESCE(p.appearance, 'system') AS appearance,
       COALESCE(p.notification_preferences, '{"in_app":true,"push":false,"email":true}'::jsonb) AS notification_preferences
FROM iam.users u
LEFT JOIN iam.user_preferences p ON p.user_id = u.id
WHERE u.id = $1 AND u.deleted_at IS NULL;

-- name: UpdateMobileUserLocale :execrows
UPDATE iam.users SET locale = sqlc.arg(locale)
WHERE id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: UpsertMobilePreferences :exec
INSERT INTO iam.user_preferences (user_id, appearance, notification_preferences)
VALUES (sqlc.arg(user_id), sqlc.arg(appearance), sqlc.arg(notification_preferences))
ON CONFLICT (user_id) DO UPDATE
SET appearance = EXCLUDED.appearance,
    notification_preferences = EXCLUDED.notification_preferences;

-- name: CountMobileUnreadNotifications :one
SELECT count(*)
FROM notify.recipients r
JOIN notify.messages m ON m.tenant_id = r.tenant_id AND m.id = r.message_id
WHERE r.tenant_id = sqlc.arg(tenant_id)
  AND r.user_id = sqlc.arg(user_id)
  AND r.delivery_status = 'delivered'
  AND r.read_at IS NULL
  AND r.archived_at IS NULL
  AND m.status = 'published'
  AND m.deleted_at IS NULL
  AND (m.expires_at IS NULL OR m.expires_at > now());

-- name: ListMobileLoginEvents :many
SELECT id, auth_method, result, occurred_at, client_ip
FROM audit.login_events
WHERE user_id = $1 AND audience = 'ak-mobile'
ORDER BY occurred_at DESC, id DESC
LIMIT 100;

-- name: ListMobileSecurityEvents :many
SELECT id, event_type, severity, occurred_at
FROM audit.security_events
WHERE user_id = $1
ORDER BY occurred_at DESC, id DESC
LIMIT 100;

-- name: ListMobileNotifications :many
SELECT m.id, m.title, m.body, m.body_format, m.message_type, m.created_at, r.read_at
FROM notify.recipients r
JOIN notify.messages m ON m.tenant_id = r.tenant_id AND m.id = r.message_id
WHERE r.tenant_id = sqlc.arg(tenant_id)
  AND r.user_id = sqlc.arg(user_id)
  AND r.archived_at IS NULL
  AND r.delivery_status = 'delivered'
  AND m.status = 'published'
  AND m.deleted_at IS NULL
  AND (m.expires_at IS NULL OR m.expires_at > now())
  AND (m.created_at, m.id) < (sqlc.arg(cursor_created_at), sqlc.arg(cursor_id)::uuid)
ORDER BY m.created_at DESC, m.id DESC
LIMIT sqlc.arg(page_limit);

-- name: MarkMobileNotificationRead :execrows
UPDATE notify.recipients
SET read_at = COALESCE(read_at, now())
WHERE tenant_id = sqlc.arg(tenant_id)
  AND user_id = sqlc.arg(user_id)
  AND message_id = sqlc.arg(message_id)
  AND delivery_status = 'delivered'
  AND archived_at IS NULL;

-- name: GetActiveMobileRelease :one
SELECT id, platform, current_version, minimum_version, upgrade_url,
       release_notes, active, lock_version, updated_at
FROM sys.mobile_releases
WHERE platform = $1 AND active = true;

-- name: ListMobileReleases :many
SELECT id, platform, current_version, minimum_version, upgrade_url,
       release_notes, active, lock_version, updated_at
FROM sys.mobile_releases
ORDER BY platform, updated_at DESC, id DESC;

-- name: GetMobileReleaseByID :one
SELECT id, platform, current_version, minimum_version, upgrade_url,
       release_notes, active, lock_version, updated_at
FROM sys.mobile_releases
WHERE id = $1;

-- name: DeactivateMobileReleasesByPlatform :exec
UPDATE sys.mobile_releases SET active = false
WHERE platform = sqlc.arg(platform) AND active
  AND (sqlc.narg(except_id)::uuid IS NULL OR id <> sqlc.narg(except_id)::uuid);

-- name: CreateMobileRelease :one
INSERT INTO sys.mobile_releases (
  platform, current_version, minimum_version, upgrade_url, release_notes, active
) VALUES (
  sqlc.arg(platform), sqlc.arg(current_version), sqlc.arg(minimum_version),
  sqlc.narg(upgrade_url), sqlc.arg(release_notes), sqlc.arg(active)
)
RETURNING id, platform, current_version, minimum_version, upgrade_url,
          release_notes, active, lock_version, updated_at;

-- name: UpdateMobileRelease :one
UPDATE sys.mobile_releases
SET current_version = sqlc.arg(current_version),
    minimum_version = sqlc.arg(minimum_version),
    upgrade_url = sqlc.narg(upgrade_url),
    release_notes = sqlc.arg(release_notes),
    active = sqlc.arg(active),
    lock_version = lock_version + 1
WHERE id = sqlc.arg(id) AND lock_version = sqlc.arg(lock_version)
RETURNING id, platform, current_version, minimum_version, upgrade_url,
          release_notes, active, lock_version, updated_at;
