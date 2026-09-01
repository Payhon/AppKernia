-- name: ListFeedbacks :many
SELECT id,user_id,description,platform,app_version,status,lock_version,created_at,updated_at
FROM app.feedbacks
WHERE tenant_id=sqlc.arg('tenant_id') AND app_id=sqlc.arg('app_id')
 AND (sqlc.arg('admin')::boolean OR user_id=sqlc.arg('user_id'))
 AND (sqlc.arg('status')::text='' OR status=sqlc.arg('status'))
 AND (sqlc.arg('query')::text='' OR description ILIKE '%'||sqlc.arg('query')::text||'%')
 AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at>=sqlc.narg('created_from'))
 AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at<=sqlc.narg('created_to'))
ORDER BY created_at DESC,id DESC LIMIT sqlc.arg('page_size') OFFSET sqlc.arg('page_offset')::bigint;

-- name: CountFeedbacks :one
SELECT count(*) FROM app.feedbacks
WHERE tenant_id=sqlc.arg('tenant_id') AND app_id=sqlc.arg('app_id')
 AND (sqlc.arg('admin')::boolean OR user_id=sqlc.arg('user_id'))
 AND (sqlc.arg('status')::text='' OR status=sqlc.arg('status'))
 AND (sqlc.arg('query')::text='' OR description ILIKE '%'||sqlc.arg('query')::text||'%')
 AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at>=sqlc.narg('created_from'))
 AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at<=sqlc.narg('created_to'));
