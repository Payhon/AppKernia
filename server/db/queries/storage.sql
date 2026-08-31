-- name: CreateSelfAvatarUploadSession :one
INSERT INTO storage.upload_sessions (
    tenant_id, user_id, provider, bucket_name, object_key, original_name,
    media_type, expected_size, status, expires_at
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('provider'), sqlc.arg('bucket_name'),
    sqlc.arg('object_key'), sqlc.arg('original_name'), sqlc.arg('media_type'),
    sqlc.arg('expected_size'), 'initiated', sqlc.arg('expires_at')
)
RETURNING id, provider, bucket_name, object_key, expires_at;

-- name: GetSelfAvatarUploadSession :one
SELECT id, tenant_id, user_id, provider, bucket_name, object_key, original_name, media_type, expected_size, expires_at
FROM storage.upload_sessions
WHERE id = sqlc.arg('id')
  AND tenant_id = sqlc.arg('tenant_id')
  AND user_id = sqlc.arg('user_id')
  AND status = 'initiated'
  AND expires_at > now();

-- name: LockSelfAvatarUploadSession :one
SELECT id, tenant_id, user_id, provider, bucket_name, object_key, original_name, media_type, expected_size, expires_at
FROM storage.upload_sessions
WHERE id = sqlc.arg('id')
  AND tenant_id = sqlc.arg('tenant_id')
  AND user_id = sqlc.arg('user_id')
  AND status = 'initiated'
  AND expires_at > now()
FOR UPDATE;

-- name: GetSelfAvatarFileIDForUpdate :one
SELECT avatar_file_id
FROM iam.users
WHERE id = sqlc.arg('user_id') AND status = 'active' AND deleted_at IS NULL
FOR UPDATE;

-- name: InsertReadySelfAvatarFile :one
INSERT INTO storage.files (
    tenant_id, owner_user_id, provider, bucket_name, object_key, original_name,
    media_type, extension, size_bytes, sha256, visibility, status, scan_status,
    metadata
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('provider'), sqlc.arg('bucket_name'),
    sqlc.arg('object_key'), sqlc.arg('original_name'), sqlc.arg('media_type'),
    sqlc.arg('extension'), sqlc.arg('size_bytes'), sqlc.arg('sha256'), 'private',
    'ready', 'skipped', jsonb_build_object('purpose', 'avatar', 'adapter', sqlc.arg('provider')::varchar)
)
ON CONFLICT (tenant_id, sha256, size_bytes)
    WHERE sha256 IS NOT NULL AND status = 'ready' AND deleted_at IS NULL
DO UPDATE SET updated_at = storage.files.updated_at
RETURNING id, provider, bucket_name, object_key;

-- name: CompleteSelfAvatarUploadSession :exec
UPDATE storage.upload_sessions
SET file_id = sqlc.arg('file_id'), status = 'completed', completed_at = now()
WHERE id = sqlc.arg('id') AND tenant_id = sqlc.arg('tenant_id') AND user_id = sqlc.arg('user_id');

-- name: UpdateSelfAvatarFile :exec
UPDATE iam.users
SET avatar_file_id = sqlc.arg('file_id')
WHERE id = sqlc.arg('user_id') AND status = 'active' AND deleted_at IS NULL;

-- name: DeletePreviousSelfAvatarUsage :exec
DELETE FROM storage.file_usages
WHERE tenant_id = sqlc.arg('tenant_id')
  AND module_code = 'iam'
  AND entity_type = 'iam.user'
  AND entity_id = sqlc.arg('user_id')
  AND field_name = 'avatar_file_id';

-- name: InsertSelfAvatarUsage :exec
INSERT INTO storage.file_usages (
    file_id, tenant_id, module_code, entity_type, entity_id, field_name
)
VALUES (
    sqlc.arg('file_id'), sqlc.arg('tenant_id'), 'iam', 'iam.user',
    sqlc.arg('user_id'), 'avatar_file_id'
);

-- name: InsertSelfAvatarAudit :exec
INSERT INTO audit.operation_logs (
    tenant_id, user_id, session_id, request_id, module_code, action_name,
    resource_type, resource_id, http_method, request_path, response_status,
    client_ip, user_agent, before_data, after_data, succeeded
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('session_id'),
    sqlc.arg('request_id'), 'storage', 'iam.me.avatar.update', 'iam.user',
    sqlc.arg('resource_id'), sqlc.arg('http_method'),
    sqlc.arg('request_path'), 200,
    sqlc.narg('client_ip'), sqlc.narg('user_agent'),
    jsonb_build_object('avatar_file_id', sqlc.narg('before_file_id')::uuid),
    jsonb_build_object('avatar_file_id', sqlc.arg('after_file_id')::uuid), true
);

-- name: GetSelfAvatarObject :one
SELECT f.id, f.provider, f.bucket_name, f.object_key, f.media_type, f.size_bytes, f.sha256, f.updated_at
FROM iam.users AS u
JOIN storage.files AS f ON f.id = u.avatar_file_id
WHERE u.id = sqlc.arg('user_id')
  AND f.tenant_id = sqlc.arg('tenant_id')
  AND f.owner_user_id = sqlc.arg('user_id')
  AND u.status = 'active'
  AND u.deleted_at IS NULL
  AND f.status = 'ready'
  AND f.scan_status IN ('clean', 'skipped')
  AND f.deleted_at IS NULL;
