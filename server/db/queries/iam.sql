-- name: CreateTenant :one
INSERT INTO iam.tenants (code, name, status, settings)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTenantByID :one
SELECT * FROM iam.tenants
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetActiveTenantByCode :one
SELECT * FROM iam.tenants
WHERE code = $1 AND status = 'active' AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO iam.users (username, email, mobile, display_name, locale, time_zone, status, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM iam.users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM iam.users
WHERE email = $1 AND deleted_at IS NULL;

-- name: UpdateSelfProfile :one
UPDATE iam.users
SET display_name = COALESCE(sqlc.narg('display_name')::varchar, display_name),
    locale = COALESCE(sqlc.narg('locale')::varchar, locale),
    time_zone = COALESCE(sqlc.narg('time_zone')::varchar, time_zone)
WHERE id = sqlc.arg('id') AND deleted_at IS NULL AND status = 'active'
RETURNING *;

-- name: InsertSelfProfileAudit :exec
INSERT INTO audit.operation_logs (
    tenant_id, user_id, session_id, request_id, module_code, action_name,
    resource_type, resource_id, http_method, request_path, response_status,
    client_ip, user_agent, before_data, after_data, succeeded
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('session_id'), sqlc.arg('request_id'),
    'iam', 'iam.me.update', 'iam.user', sqlc.arg('resource_id'), 'PATCH',
    '/admin-api/v1/me', 200, sqlc.narg('client_ip'), sqlc.narg('user_agent'),
    sqlc.arg('before_data'), sqlc.arg('after_data'), true
);

-- name: CreateUserCredential :one
INSERT INTO iam.user_credentials (user_id, password_hash, password_algorithm)
VALUES ($1, $2, 'argon2id')
RETURNING *;

-- name: GetDefaultMemberRole :one
SELECT id
FROM iam.roles
WHERE tenant_id = $1
  AND code = 'member'
  AND status = 'active'
  AND deleted_at IS NULL
ORDER BY is_default DESC, id
LIMIT 1;

-- name: InsertAdminRegistrationAudit :exec
INSERT INTO audit.operation_logs (
    tenant_id, user_id, request_id, module_code, action_name, resource_type,
    resource_id, http_method, request_path, response_status, client_ip,
    user_agent, before_data, after_data, succeeded
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.narg('request_id'),
    'iam', 'iam.auth.register', 'iam.user', sqlc.arg('resource_id'), 'POST',
    '/admin-api/v1/auth/register', 202, sqlc.narg('client_ip'), sqlc.narg('user_agent'),
    '{}'::jsonb, jsonb_build_object('registered', true, 'role', 'member'), true
);

-- name: GetCredentialByEmail :one
SELECT u.id AS user_id,
       u.email,
       u.display_name,
       u.locale,
       u.status,
       c.password_hash,
       c.password_version,
       c.failed_attempts,
       c.locked_until
FROM iam.users u
JOIN iam.user_credentials c ON c.user_id = u.id
WHERE u.email = $1 AND u.deleted_at IS NULL;

-- name: GetSelfPasswordState :one
SELECT password_hash, password_version
FROM iam.user_credentials
WHERE user_id = $1;

-- name: ListRecentPasswordHashes :many
SELECT password_hash
FROM iam.password_history
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 5;

-- name: InsertPasswordResetChallenge :one
INSERT INTO iam.verification_challenges (
    user_id, challenge_type, target_hash, secret_hash, expires_at, created_ip,
    metadata
)
SELECT sqlc.arg('user_id'), 'password_reset', sqlc.arg('target_hash'),
       sqlc.arg('secret_hash'), sqlc.arg('expires_at'), sqlc.narg('created_ip'),
       jsonb_build_object('channel', 'email')
WHERE NOT EXISTS (
    SELECT 1
    FROM iam.verification_challenges
    WHERE target_hash = sqlc.arg('target_hash')
      AND challenge_type = 'password_reset'
      AND consumed_at IS NULL
      AND expires_at > now()
      AND created_at > now() - interval '60 seconds'
)
RETURNING id;

-- name: GetPasswordResetState :one
SELECT vc.user_id,
       c.password_hash,
       c.password_version
FROM iam.verification_challenges AS vc
JOIN iam.user_credentials AS c ON c.user_id = vc.user_id
JOIN iam.users AS u ON u.id = vc.user_id
WHERE vc.secret_hash = $1
  AND vc.challenge_type = 'password_reset'
  AND vc.consumed_at IS NULL
  AND vc.expires_at > now()
  AND vc.attempts < vc.max_attempts
  AND u.status = 'active'
  AND u.deleted_at IS NULL
ORDER BY vc.created_at DESC
LIMIT 1;

-- name: LockPasswordResetChallenge :one
SELECT id, user_id
FROM iam.verification_challenges
WHERE secret_hash = sqlc.arg('secret_hash')
  AND challenge_type = 'password_reset'
  AND consumed_at IS NULL
  AND expires_at > now()
  AND attempts < max_attempts
FOR UPDATE;

-- name: UpdatePasswordAfterReset :one
UPDATE iam.user_credentials
SET password_hash = sqlc.arg('new_hash'),
    password_algorithm = 'argon2id',
    password_version = password_version + 1,
    password_changed_at = now(),
    force_password_change = false,
    failed_attempts = 0,
    locked_until = NULL
WHERE user_id = sqlc.arg('user_id')
  AND password_hash = sqlc.arg('expected_hash')
  AND password_version = sqlc.arg('expected_version')
RETURNING password_version;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE iam.refresh_tokens
SET revoked_at = COALESCE(revoked_at, now())
WHERE revoked_at IS NULL
  AND session_id IN (SELECT id FROM iam.sessions WHERE user_id = $1);

-- name: RevokeAllUserSessions :execrows
UPDATE iam.sessions
SET status = 'revoked',
    revoked_at = COALESCE(revoked_at, now()),
    revoke_reason = 'password_reset',
    access_token_version = access_token_version + 1
WHERE user_id = $1
  AND status = 'active'
  AND revoked_at IS NULL;

-- name: ConsumePasswordResetChallenges :exec
UPDATE iam.verification_challenges
SET consumed_at = COALESCE(consumed_at, now())
WHERE user_id = $1
  AND challenge_type = 'password_reset'
  AND consumed_at IS NULL;

-- name: InsertPasswordResetAudit :exec
INSERT INTO audit.operation_logs (
    tenant_id, user_id, request_id, module_code, action_name, resource_type,
    resource_id, http_method, request_path, response_status, client_ip,
    user_agent, before_data, after_data, succeeded
)
VALUES (
    (
      SELECT tm.tenant_id FROM iam.tenant_members AS tm
      WHERE tm.user_id = sqlc.arg('user_id') AND tm.status = 'active'
      ORDER BY tm.created_at, tm.tenant_id LIMIT 1
    ),
    sqlc.arg('user_id'), sqlc.narg('request_id'), 'iam', 'iam.auth.password.reset',
    'iam.user_credential', sqlc.arg('resource_id'), 'POST',
    '/admin-api/v1/auth/password/reset', 200, sqlc.narg('client_ip'),
    sqlc.narg('user_agent'),
    jsonb_build_object('password_version', sqlc.arg('before_version')::integer),
    jsonb_build_object(
      'password_version', sqlc.arg('after_version')::integer,
      'sessions_revoked', sqlc.arg('revoked_session_count')::bigint
    ), true
);

-- name: UpdateSelfPasswordConditional :one
UPDATE iam.user_credentials
SET password_hash = sqlc.arg('new_hash'),
    password_algorithm = 'argon2id',
    password_version = password_version + 1,
    password_changed_at = now(),
    force_password_change = false,
    failed_attempts = 0,
    locked_until = NULL
WHERE user_id = sqlc.arg('user_id')
  AND password_hash = sqlc.arg('expected_hash')
  AND password_version = sqlc.arg('expected_version')
RETURNING password_version;

-- name: InsertPasswordHistory :exec
INSERT INTO iam.password_history (user_id, password_hash, password_algorithm)
VALUES ($1, $2, 'argon2id');

-- name: RevokeOtherSessionRefreshTokens :exec
UPDATE iam.refresh_tokens
SET revoked_at = COALESCE(revoked_at, now())
WHERE revoked_at IS NULL
  AND session_id IN (
    SELECT s.id FROM iam.sessions AS s
    WHERE s.user_id = sqlc.arg('user_id') AND s.id <> sqlc.arg('current_session_id')
  );

-- name: RevokeOtherSessions :execrows
UPDATE iam.sessions
SET status = 'revoked', revoked_at = COALESCE(revoked_at, now()),
    revoke_reason = 'password_changed', access_token_version = access_token_version + 1
WHERE user_id = sqlc.arg('user_id')
  AND id <> sqlc.arg('current_session_id')
  AND status = 'active'
  AND revoked_at IS NULL;

-- name: InsertSelfPasswordChangeAudit :exec
INSERT INTO audit.operation_logs (
    tenant_id, user_id, session_id, request_id, module_code, action_name,
    resource_type, resource_id, http_method, request_path, response_status,
    client_ip, user_agent, before_data, after_data, succeeded
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('session_id'), sqlc.arg('request_id'),
    'iam', 'iam.me.password.change', 'iam.user_credential', sqlc.arg('resource_id'), 'POST',
    '/admin-api/v1/me/password/change', 200, sqlc.narg('client_ip'), sqlc.narg('user_agent'),
    jsonb_build_object('password_version', sqlc.arg('before_version')::integer),
    jsonb_build_object(
      'password_version', sqlc.arg('after_version')::integer,
      'other_sessions_revoked', sqlc.arg('revoked_session_count')::bigint
    ),
    true
);

-- name: CreateTenantMember :one
INSERT INTO iam.tenant_members (tenant_id, user_id, member_number, display_name, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListUserTenants :many
SELECT t.id, t.code, t.name, tm.status, tm.joined_at
FROM iam.tenant_members tm
JOIN iam.tenants t ON t.id = tm.tenant_id AND t.deleted_at IS NULL
WHERE tm.user_id = $1 AND tm.status = 'active'
ORDER BY t.name, t.id;

-- name: CreateRole :one
INSERT INTO iam.roles (tenant_id, code, name, description, role_type, data_scope, is_default, is_system, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: AssignUserRole :exec
INSERT INTO iam.user_roles (tenant_id, user_id, role_id, granted_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, user_id, role_id) DO NOTHING;

-- name: ListEffectiveRoleCodes :many
SELECT r.code::text
FROM iam.user_roles ur
JOIN iam.roles r
  ON r.tenant_id = ur.tenant_id
 AND r.id = ur.role_id
 AND r.deleted_at IS NULL
WHERE ur.tenant_id = $1
  AND ur.user_id = $2
  AND r.status = 'active'
  AND ur.valid_from <= now()
  AND (ur.valid_until IS NULL OR ur.valid_until > now())
ORDER BY r.code;

-- name: ListEffectivePermissionCodes :many
SELECT DISTINCT p.code::text
FROM iam.user_roles ur
JOIN iam.roles r
  ON r.tenant_id = ur.tenant_id
 AND r.id = ur.role_id
 AND r.deleted_at IS NULL
JOIN iam.role_permissions rp
  ON rp.tenant_id = ur.tenant_id
 AND rp.role_id = ur.role_id
JOIN iam.permissions p
  ON p.id = rp.permission_id
 AND p.status = 'active'
WHERE ur.tenant_id = $1
  AND ur.user_id = $2
  AND r.status = 'active'
  AND ur.valid_from <= now()
  AND (ur.valid_until IS NULL OR ur.valid_until > now())
ORDER BY p.code::text;
