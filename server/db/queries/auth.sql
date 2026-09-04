-- name: CreateSession :one
INSERT INTO iam.sessions (
    user_id, tenant_id, app_id, device_id, audience, status, access_token_version,
    absolute_expires_at, idle_expires_at, ip_address, user_agent
)
VALUES (
    sqlc.arg('user_id'), sqlc.arg('tenant_id'), sqlc.narg('app_id'), sqlc.narg('device_id'), sqlc.arg('audience'),
    'active', 1, sqlc.arg('absolute_expires_at'), sqlc.arg('idle_expires_at'),
    sqlc.narg('ip_address'), sqlc.narg('user_agent')
)
RETURNING *;

-- name: UpsertWebDevice :one
INSERT INTO iam.devices (user_id, device_key, platform, last_ip, last_seen_at)
VALUES (sqlc.arg('user_id'), sqlc.arg('device_key'), 'web', sqlc.narg('last_ip'), now())
ON CONFLICT (user_id, device_key) DO UPDATE
SET last_ip = EXCLUDED.last_ip,
    last_seen_at = now(),
    updated_at = now()
RETURNING id;

-- name: CreateRefreshToken :one
INSERT INTO iam.refresh_tokens (
    session_id, token_hash, parent_token_id, expires_at, created_ip
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: InsertSuccessfulLoginEvent :exec
INSERT INTO audit.login_events (
    tenant_id, user_id, session_id, app_id, request_id, auth_method, audience, result,
    client_ip, user_agent, device_info
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('session_id'), sqlc.narg('app_id'), sqlc.narg('request_id'),
    sqlc.arg('auth_method'), sqlc.arg('audience'), 'success', sqlc.narg('client_ip'), sqlc.narg('user_agent'),
    jsonb_build_object('platform', 'web', 'registered', sqlc.arg('device_registered')::boolean)
);

-- name: InsertFailedLoginEvent :exec
INSERT INTO audit.login_events (
    tenant_id, user_id, app_id, request_id, auth_method, audience, result, failure_reason, client_ip, user_agent
)
VALUES (
    (
        SELECT tm.tenant_id
        FROM iam.tenant_members tm
        WHERE tm.user_id = sqlc.narg('user_id') AND tm.status = 'active'
        ORDER BY tm.created_at, tm.tenant_id
        LIMIT 1
    ),
    sqlc.narg('user_id'), sqlc.narg('app_id'), sqlc.narg('request_id'), 'password', sqlc.arg('audience'),
    'failure', 'invalid_credentials', sqlc.narg('client_ip'), sqlc.narg('user_agent')
);

-- name: GetActiveLoginFailureCount :one
SELECT failure_count
FROM iam.login_failure_states
WHERE scope_hash = sqlc.arg('scope_hash')
  AND expires_at > sqlc.arg('now_at');

-- name: UpsertLoginFailureState :one
INSERT INTO iam.login_failure_states (scope_hash, failure_count, last_failed_at, expires_at)
VALUES (sqlc.arg('scope_hash'), 1, sqlc.arg('now_at'), sqlc.arg('expires_at'))
ON CONFLICT (scope_hash) DO UPDATE
SET failure_count = CASE
        WHEN iam.login_failure_states.expires_at > sqlc.arg('now_at')
            THEN LEAST(iam.login_failure_states.failure_count + 1, 1000000)
        ELSE 1
    END,
    last_failed_at = sqlc.arg('now_at'),
    expires_at = sqlc.arg('expires_at')
RETURNING failure_count;

-- name: DeleteLoginFailureState :exec
DELETE FROM iam.login_failure_states
WHERE scope_hash = sqlc.arg('scope_hash');

-- name: GetLoginCaptchaFailureStateForUpdate :one
SELECT failure_count, expires_at
FROM iam.login_failure_states
WHERE scope_hash = sqlc.arg('scope_hash')
FOR UPDATE;

-- name: LoginCaptchaCoolingDown :one
SELECT EXISTS (
    SELECT 1
    FROM iam.login_captcha_challenges
    WHERE scope_hash = sqlc.arg('scope_hash')
      AND created_at > sqlc.arg('now_at')::timestamptz - interval '2 seconds'
) AS cooling_down;

-- name: InvalidateActiveLoginCaptchas :exec
UPDATE iam.login_captcha_challenges
SET consumed_at = COALESCE(consumed_at, sqlc.arg('now_at'))
WHERE scope_hash = sqlc.arg('scope_hash')
  AND consumed_at IS NULL;

-- name: InsertLoginCaptchaChallenge :one
INSERT INTO iam.login_captcha_challenges (
    id, scope_hash, captcha_type, proof_hash, expires_at, created_at
)
VALUES (
    sqlc.arg('id'), sqlc.arg('scope_hash'), sqlc.arg('captcha_type'),
    sqlc.arg('proof_hash'), sqlc.arg('expires_at'), sqlc.arg('created_at')
)
RETURNING id;

-- name: GetLoginCaptchaForUpdate :one
SELECT captcha_type, proof_hash, attempt_count
FROM iam.login_captcha_challenges
WHERE id = sqlc.arg('id')
  AND scope_hash = sqlc.arg('scope_hash')
  AND consumed_at IS NULL
  AND expires_at > sqlc.arg('now_at')
  AND attempt_count < 5
FOR UPDATE;

-- name: CompleteLoginCaptchaAttempt :exec
UPDATE iam.login_captcha_challenges
SET attempt_count = attempt_count + 1,
    consumed_at = CASE
        WHEN sqlc.arg('consume')::boolean OR attempt_count + 1 >= 5 THEN sqlc.arg('now_at')
        ELSE consumed_at
    END
WHERE id = sqlc.arg('id')
  AND consumed_at IS NULL
  AND attempt_count < 5;

-- name: GetRefreshTokenForUpdate :one
SELECT rt.id AS refresh_token_id,
       rt.session_id,
       rt.expires_at AS refresh_expires_at,
       rt.consumed_at,
       rt.revoked_at AS refresh_revoked_at,
       s.user_id,
       s.tenant_id,
       s.app_id,
       s.audience,
       s.status AS session_status,
       s.access_token_version,
       s.absolute_expires_at,
       s.revoked_at AS session_revoked_at
FROM iam.refresh_tokens rt
JOIN iam.sessions s ON s.id = rt.session_id
WHERE rt.token_hash = $1
FOR UPDATE OF rt, s;

-- name: MarkRefreshTokenConsumed :exec
UPDATE iam.refresh_tokens
SET consumed_at = now(), replaced_by_token_id = $2
WHERE id = $1 AND consumed_at IS NULL AND revoked_at IS NULL;

-- name: MarkRefreshTokenReused :exec
UPDATE iam.refresh_tokens
SET reuse_detected_at = COALESCE(reuse_detected_at, now())
WHERE id = $1;

-- name: RevokeSession :exec
UPDATE iam.sessions
SET status = 'revoked', revoked_at = COALESCE(revoked_at, now()),
    revoke_reason = $2, access_token_version = access_token_version + 1
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeSessionRefreshTokens :exec
UPDATE iam.refresh_tokens
SET revoked_at = COALESCE(revoked_at, now())
WHERE session_id = $1 AND revoked_at IS NULL;

-- name: InsertRefreshReuseSecurityEvent :exec
INSERT INTO audit.security_events (
    tenant_id, user_id, session_id, app_id, event_type, severity, source, client_ip, details
)
VALUES ($1, $2, $3, $4, 'iam.refresh_token.reuse', 'high', 'auth', $5, '{"action":"session_revoked"}'::jsonb);

-- name: GetAuthContextUser :one
SELECT u.id, u.email, u.display_name, u.locale, u.time_zone, u.avatar_file_id,
       t.id AS tenant_id, t.code AS tenant_code, t.name AS tenant_name
FROM iam.users u
JOIN iam.tenant_members tm ON tm.user_id = u.id AND tm.tenant_id = $2 AND tm.status = 'active'
JOIN iam.tenants t ON t.id = tm.tenant_id AND t.deleted_at IS NULL AND t.status = 'active'
WHERE u.id = $1 AND u.deleted_at IS NULL AND u.status = 'active';

-- name: GetActiveSession :one
SELECT id
FROM iam.sessions
WHERE id = $1
  AND user_id = $2
  AND tenant_id = $3
  AND audience = $4
  AND access_token_version = $5
	AND app_id IS NOT DISTINCT FROM sqlc.narg('app_id')
  AND status = 'active'
  AND revoked_at IS NULL
  AND absolute_expires_at > now();

-- name: ListSelfSessions :many
SELECT id, audience, status, ip_address, user_agent, last_seen_at,
       absolute_expires_at, created_at
FROM iam.sessions
WHERE user_id = sqlc.arg('user_id')
  AND tenant_id = sqlc.arg('tenant_id')
  AND audience = sqlc.arg('audience')
  AND status = 'active'
  AND revoked_at IS NULL
  AND absolute_expires_at > now()
ORDER BY last_seen_at DESC, created_at DESC, id;

-- name: LockSelfSessionForRevoke :one
SELECT id, status, revoked_at
FROM iam.sessions
WHERE id = sqlc.arg('session_id')
  AND user_id = sqlc.arg('user_id')
  AND tenant_id = sqlc.arg('tenant_id')
  AND audience = sqlc.arg('audience')
  AND status = 'active'
  AND revoked_at IS NULL
FOR UPDATE;

-- name: InsertSelfSessionRevokeAudit :exec
INSERT INTO audit.operation_logs (
    tenant_id, user_id, session_id, request_id, module_code, action_name,
    resource_type, resource_id, http_method, request_path, response_status,
    client_ip, user_agent, before_data, after_data, succeeded
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('actor_session_id'), sqlc.arg('request_id'),
    'iam', 'iam.me.session.revoke', 'iam.session', sqlc.arg('resource_id'), 'DELETE',
    sqlc.arg('request_path'), 200, sqlc.narg('client_ip'), sqlc.narg('user_agent'),
    '{"status":"active"}'::jsonb, '{"status":"revoked"}'::jsonb, true
);

-- name: ListSelfDevices :many
SELECT d.id,
       d.platform,
       d.device_name,
       d.model,
       d.os_version,
       d.app_version,
       d.last_ip,
       d.last_seen_at,
       d.created_at,
       CAST(COALESCE((
           SELECT s.user_agent
           FROM iam.sessions s
           WHERE s.user_id = d.user_id
             AND s.device_id = d.id
             AND s.tenant_id = sqlc.arg('tenant_id')
             AND s.audience = sqlc.arg('audience')
           ORDER BY s.last_seen_at DESC, s.created_at DESC, s.id
           LIMIT 1
       ), '') AS text) AS latest_user_agent,
       (
           SELECT count(*)
           FROM iam.sessions s
           WHERE s.user_id = d.user_id
             AND s.device_id = d.id
             AND s.tenant_id = sqlc.arg('tenant_id')
             AND s.audience = sqlc.arg('audience')
             AND s.status = 'active'
             AND s.revoked_at IS NULL
             AND s.absolute_expires_at > now()
       ) AS active_session_count,
       EXISTS (
           SELECT 1
           FROM iam.sessions s
           WHERE s.id = sqlc.arg('current_session_id')
             AND s.user_id = d.user_id
             AND s.device_id = d.id
       ) AS current
FROM iam.devices d
WHERE d.user_id = sqlc.arg('user_id')
  AND EXISTS (
      SELECT 1
      FROM iam.sessions s
      WHERE s.user_id = d.user_id
        AND s.device_id = d.id
        AND s.tenant_id = sqlc.arg('tenant_id')
        AND s.audience = sqlc.arg('audience')
  )
ORDER BY current DESC, d.last_seen_at DESC NULLS LAST, d.created_at DESC, d.id;

-- name: LockSelfDeviceForRemove :one
SELECT d.id,
       EXISTS (
           SELECT 1
           FROM iam.sessions current_session
           WHERE current_session.id = sqlc.arg('current_session_id')
             AND current_session.user_id = d.user_id
             AND current_session.device_id = d.id
       ) AS current
FROM iam.devices d
WHERE d.id = sqlc.arg('device_id')
  AND d.user_id = sqlc.arg('user_id')
  AND EXISTS (
      SELECT 1
      FROM iam.sessions scoped_session
      WHERE scoped_session.user_id = d.user_id
        AND scoped_session.device_id = d.id
        AND scoped_session.tenant_id = sqlc.arg('tenant_id')
        AND scoped_session.audience = sqlc.arg('audience')
  )
FOR UPDATE OF d;

-- name: RevokeSelfDeviceRefreshTokens :exec
UPDATE iam.refresh_tokens token
SET revoked_at = COALESCE(token.revoked_at, now())
WHERE token.revoked_at IS NULL
  AND EXISTS (
      SELECT 1
      FROM iam.sessions session
      WHERE session.id = token.session_id
        AND session.user_id = sqlc.arg('user_id')
        AND session.device_id = sqlc.arg('device_id')
  );

-- name: RevokeSelfDeviceSessions :one
WITH revoked AS (
    UPDATE iam.sessions
    SET status = 'revoked',
        revoked_at = COALESCE(revoked_at, now()),
        revoke_reason = 'device_removed',
        access_token_version = access_token_version + 1
    WHERE user_id = sqlc.arg('user_id')
      AND device_id = sqlc.arg('device_id')
      AND status = 'active'
      AND revoked_at IS NULL
    RETURNING id
)
SELECT count(*) AS revoked_count FROM revoked;

-- name: DeleteSelfDevice :exec
DELETE FROM iam.devices
WHERE id = sqlc.arg('device_id')
  AND user_id = sqlc.arg('user_id');

-- name: InsertSelfDeviceRemoveAudit :exec
INSERT INTO audit.operation_logs (
    tenant_id, user_id, session_id, request_id, module_code, action_name,
    resource_type, resource_id, http_method, request_path, response_status,
    client_ip, user_agent, before_data, after_data, succeeded
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('actor_session_id'), sqlc.arg('request_id'),
    'iam', 'iam.me.device.remove', 'iam.device', sqlc.arg('resource_id'), 'DELETE',
    sqlc.arg('request_path'), 200, sqlc.narg('client_ip'), sqlc.narg('user_agent'),
    '{"registered":true}'::jsonb,
    jsonb_build_object('removed', true, 'revoked_session_count', sqlc.arg('revoked_session_count')::bigint),
    true
);
