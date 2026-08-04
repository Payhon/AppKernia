-- name: CreateAdminFileUploadSession :one
INSERT INTO storage.upload_sessions (
    tenant_id, user_id, provider, bucket_name, object_key, original_name,
    media_type, expected_size, part_size, status, expires_at
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), 'local', 'appkernia-local',
    sqlc.arg('object_key'), sqlc.arg('original_name'), sqlc.arg('media_type'),
    sqlc.arg('expected_size'), sqlc.arg('part_size'), 'initiated', sqlc.arg('expires_at')
)
RETURNING id, original_name, media_type, expected_size, part_size, status, object_key, expires_at;

-- name: GetAdminFileUploadSession :one
SELECT id, original_name, media_type, expected_size, part_size, status, object_key, expires_at
FROM storage.upload_sessions
WHERE id = sqlc.arg('id') AND tenant_id = sqlc.arg('tenant_id')
  AND status IN ('initiated', 'uploading') AND expires_at > now();

-- name: ListAdminFileUploadParts :many
SELECT part_number, size_bytes, etag, checksum_sha256
FROM storage.upload_parts
WHERE upload_session_id = sqlc.arg('upload_session_id')
ORDER BY part_number;

-- name: UpsertAdminFileUploadPart :exec
INSERT INTO storage.upload_parts (upload_session_id, part_number, etag, size_bytes, checksum_sha256)
SELECT id, sqlc.arg('part_number'), sqlc.arg('etag'), sqlc.arg('size_bytes'), sqlc.arg('checksum_sha256')
FROM storage.upload_sessions
WHERE id = sqlc.arg('upload_session_id') AND tenant_id = sqlc.arg('tenant_id')
  AND status IN ('initiated', 'uploading') AND expires_at > now()
ON CONFLICT (upload_session_id, part_number) DO UPDATE
SET etag = EXCLUDED.etag, size_bytes = EXCLUDED.size_bytes,
    checksum_sha256 = EXCLUDED.checksum_sha256, uploaded_at = now();

-- name: MarkAdminUploadUploading :exec
UPDATE storage.upload_sessions SET status = 'uploading', updated_at = now()
WHERE id = sqlc.arg('id') AND tenant_id = sqlc.arg('tenant_id') AND status = 'initiated';

-- name: LockAdminFileUploadSession :one
SELECT id, user_id, original_name, media_type, expected_size, part_size, status, object_key, expires_at
FROM storage.upload_sessions
WHERE id = sqlc.arg('id') AND tenant_id = sqlc.arg('tenant_id')
  AND status IN ('initiated', 'uploading') AND expires_at > now()
FOR UPDATE;

-- name: AbortAdminFileUploadSession :execrows
UPDATE storage.upload_sessions SET status = 'aborted', updated_at = now()
WHERE id = sqlc.arg('id') AND tenant_id = sqlc.arg('tenant_id')
  AND status IN ('initiated', 'uploading');

-- name: InsertReadyAdminFile :one
INSERT INTO storage.files (
    tenant_id, owner_user_id, provider, bucket_name, object_key, original_name,
    media_type, extension, size_bytes, sha256, visibility, status, scan_status, metadata
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('owner_user_id'), 'local', 'appkernia-local',
    sqlc.arg('object_key'), sqlc.arg('original_name'), sqlc.arg('media_type'),
    sqlc.arg('extension'), sqlc.arg('size_bytes'), sqlc.arg('sha256'), 'private',
    'ready', sqlc.arg('scan_status'), jsonb_build_object('adapter', 'development-local')
)
ON CONFLICT (tenant_id, sha256, size_bytes)
    WHERE sha256 IS NOT NULL AND status = 'ready' AND deleted_at IS NULL
DO UPDATE SET updated_at = storage.files.updated_at
RETURNING id, owner_user_id, original_name, media_type, extension, size_bytes, status,
          scan_status, created_at, updated_at, object_key, sha256;

-- name: CompleteAdminFileUploadSession :execrows
UPDATE storage.upload_sessions
SET file_id = sqlc.arg('file_id'), status = 'completed', completed_at = now(), updated_at = now()
WHERE id = sqlc.arg('id') AND tenant_id = sqlc.arg('tenant_id')
  AND status IN ('initiated', 'uploading');

-- name: ListAdminFiles :many
SELECT f.id, f.owner_user_id, f.original_name, f.media_type, f.extension, f.size_bytes,
       f.status, f.scan_status, f.created_at, f.updated_at,
       (SELECT count(*) FROM storage.file_usages u WHERE u.tenant_id = f.tenant_id AND u.file_id = f.id) AS usage_count
FROM storage.files f
WHERE f.tenant_id = sqlc.arg('tenant_id') AND f.deleted_at IS NULL
  AND (sqlc.arg('query')::text = '' OR f.original_name ILIKE '%' || sqlc.arg('query') || '%')
  AND (sqlc.arg('status')::text = '' OR f.status = sqlc.arg('status'))
  AND (sqlc.arg('scan_status')::text = '' OR f.scan_status = sqlc.arg('scan_status'))
  AND (sqlc.arg('media_type')::text = '' OR f.media_type ILIKE sqlc.arg('media_type') || '%')
ORDER BY f.created_at DESC, f.id DESC
LIMIT sqlc.arg('page_size') OFFSET sqlc.arg('page_offset');

-- name: CountAdminFiles :one
SELECT count(*)
FROM storage.files f
WHERE f.tenant_id = sqlc.arg('tenant_id') AND f.deleted_at IS NULL
  AND (sqlc.arg('query')::text = '' OR f.original_name ILIKE '%' || sqlc.arg('query') || '%')
  AND (sqlc.arg('status')::text = '' OR f.status = sqlc.arg('status'))
  AND (sqlc.arg('scan_status')::text = '' OR f.scan_status = sqlc.arg('scan_status'))
  AND (sqlc.arg('media_type')::text = '' OR f.media_type ILIKE sqlc.arg('media_type') || '%');

-- name: GetAdminFile :one
SELECT f.id, f.owner_user_id, f.original_name, f.media_type, f.extension, f.size_bytes,
       f.status, f.scan_status, f.created_at, f.updated_at, f.object_key, f.sha256,
       (SELECT count(*) FROM storage.file_usages u WHERE u.tenant_id = f.tenant_id AND u.file_id = f.id) AS usage_count
FROM storage.files f
WHERE f.id = sqlc.arg('id') AND f.tenant_id = sqlc.arg('tenant_id') AND f.deleted_at IS NULL;

-- name: ListAdminFileUsages :many
SELECT id, module_code, entity_type, entity_id, field_name, created_at
FROM storage.file_usages
WHERE tenant_id = sqlc.arg('tenant_id') AND file_id = sqlc.arg('file_id')
ORDER BY created_at DESC, id DESC;

-- name: LockAdminFileForDelete :one
SELECT f.id, f.owner_user_id, f.original_name, f.media_type, f.extension, f.size_bytes,
       f.status, f.scan_status, f.created_at, f.updated_at, f.object_key, f.sha256,
       (SELECT count(*) FROM storage.file_usages u WHERE u.tenant_id = f.tenant_id AND u.file_id = f.id) AS usage_count
FROM storage.files f
WHERE f.id = sqlc.arg('id') AND f.tenant_id = sqlc.arg('tenant_id') AND f.deleted_at IS NULL
FOR UPDATE;

-- name: SoftDeleteAdminFile :execrows
UPDATE storage.files SET status = 'deleted', deleted_at = now(), updated_at = now()
WHERE id = sqlc.arg('id') AND tenant_id = sqlc.arg('tenant_id') AND deleted_at IS NULL;

-- name: InsertAdminStorageAudit :exec
INSERT INTO audit.operation_logs (
    tenant_id, user_id, session_id, request_id, module_code, action_name,
    resource_type, resource_id, http_method, request_path, response_status,
    client_ip, user_agent, after_data, succeeded
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('session_id'), sqlc.arg('request_id'),
    'storage', sqlc.arg('action_name'), sqlc.arg('resource_type'), sqlc.arg('resource_id'),
    sqlc.arg('http_method'), sqlc.arg('request_path'), sqlc.arg('response_status'),
    sqlc.narg('client_ip'), sqlc.narg('user_agent'), sqlc.arg('after_data'), true
);
