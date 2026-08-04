-- name: AuditListOperations :many
SELECT id, user_id, session_id, request_id, COALESCE(trace_id, '') AS trace_id,
       module_code, action_name, COALESCE(permission_code, '') AS permission_code,
       COALESCE(resource_type, '') AS resource_type, COALESCE(resource_id, '') AS resource_id,
       COALESCE(http_method, '') AS http_method, COALESCE(request_path, '') AS request_path,
       response_status, COALESCE(client_ip::text, '') AS client_ip,
       COALESCE(request_summary, '{}'::jsonb) AS request_summary,
       COALESCE(before_data, '{}'::jsonb) AS before_data,
       COALESCE(after_data, '{}'::jsonb) AS after_data,
       duration_ms, succeeded, COALESCE(error_code, '') AS error_code, occurred_at
FROM audit.operation_logs
WHERE tenant_id = sqlc.arg('tenant_id')
  AND occurred_at >= sqlc.arg('from_at')
  AND occurred_at <= sqlc.arg('to_at')
  AND (sqlc.arg('query')::text = '' OR request_id ILIKE '%' || sqlc.arg('query') || '%'
       OR action_name ILIKE '%' || sqlc.arg('query') || '%'
       OR COALESCE(resource_id, '') ILIKE '%' || sqlc.arg('query') || '%')
  AND (sqlc.arg('module_code')::text = '' OR module_code = sqlc.arg('module_code'))
  AND (sqlc.arg('result')::text = '' OR (sqlc.arg('result') = 'success' AND succeeded)
       OR (sqlc.arg('result') = 'failure' AND NOT succeeded))
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg('page_size') OFFSET sqlc.arg('page_offset');

-- name: AuditCountOperations :one
SELECT count(*)
FROM audit.operation_logs
WHERE tenant_id = sqlc.arg('tenant_id')
  AND occurred_at >= sqlc.arg('from_at')
  AND occurred_at <= sqlc.arg('to_at')
  AND (sqlc.arg('query')::text = '' OR request_id ILIKE '%' || sqlc.arg('query') || '%'
       OR action_name ILIKE '%' || sqlc.arg('query') || '%'
       OR COALESCE(resource_id, '') ILIKE '%' || sqlc.arg('query') || '%')
  AND (sqlc.arg('module_code')::text = '' OR module_code = sqlc.arg('module_code'))
  AND (sqlc.arg('result')::text = '' OR (sqlc.arg('result') = 'success' AND succeeded)
       OR (sqlc.arg('result') = 'failure' AND NOT succeeded));

-- name: AuditListLogins :many
SELECT id, user_id, session_id, COALESCE(request_id, '') AS request_id,
       COALESCE(login_identifier_hint, '') AS login_identifier_hint,
       auth_method, audience, result, COALESCE(failure_reason, '') AS failure_reason,
       COALESCE(client_ip::text, '') AS client_ip, occurred_at
FROM audit.login_events
WHERE tenant_id = sqlc.arg('tenant_id')
  AND occurred_at >= sqlc.arg('from_at')
  AND occurred_at <= sqlc.arg('to_at')
  AND (sqlc.arg('result')::text = '' OR result = sqlc.arg('result'))
  AND (sqlc.arg('audience')::text = '' OR audience = sqlc.arg('audience'))
  AND (sqlc.arg('auth_method')::text = '' OR auth_method = sqlc.arg('auth_method'))
  AND (sqlc.arg('query')::text = '' OR COALESCE(request_id, '') ILIKE '%' || sqlc.arg('query') || '%'
       OR COALESCE(login_identifier_hint, '') ILIKE '%' || sqlc.arg('query') || '%')
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg('page_size') OFFSET sqlc.arg('page_offset');

-- name: AuditCountLogins :one
SELECT count(*)
FROM audit.login_events
WHERE tenant_id = sqlc.arg('tenant_id')
  AND occurred_at >= sqlc.arg('from_at')
  AND occurred_at <= sqlc.arg('to_at')
  AND (sqlc.arg('result')::text = '' OR result = sqlc.arg('result'))
  AND (sqlc.arg('audience')::text = '' OR audience = sqlc.arg('audience'))
  AND (sqlc.arg('auth_method')::text = '' OR auth_method = sqlc.arg('auth_method'))
  AND (sqlc.arg('query')::text = '' OR COALESCE(request_id, '') ILIKE '%' || sqlc.arg('query') || '%'
       OR COALESCE(login_identifier_hint, '') ILIKE '%' || sqlc.arg('query') || '%');

-- name: AuditListSecurityEvents :many
SELECT id, user_id, session_id, event_type, severity, source,
       COALESCE(client_ip::text, '') AS client_ip, resolved_at, resolved_by, occurred_at
FROM audit.security_events
WHERE tenant_id = sqlc.arg('tenant_id')
  AND occurred_at >= sqlc.arg('from_at')
  AND occurred_at <= sqlc.arg('to_at')
  AND (sqlc.arg('query')::text = '' OR event_type ILIKE '%' || sqlc.arg('query') || '%'
       OR source ILIKE '%' || sqlc.arg('query') || '%')
  AND (sqlc.arg('severity')::text = '' OR severity = sqlc.arg('severity'))
  AND (sqlc.arg('source')::text = '' OR source = sqlc.arg('source'))
  AND (sqlc.arg('status')::text = '' OR (sqlc.arg('status') = 'open' AND resolved_at IS NULL)
       OR (sqlc.arg('status') = 'resolved' AND resolved_at IS NOT NULL))
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg('page_size') OFFSET sqlc.arg('page_offset');

-- name: AuditCountSecurityEvents :one
SELECT count(*)
FROM audit.security_events
WHERE tenant_id = sqlc.arg('tenant_id')
  AND occurred_at >= sqlc.arg('from_at')
  AND occurred_at <= sqlc.arg('to_at')
  AND (sqlc.arg('query')::text = '' OR event_type ILIKE '%' || sqlc.arg('query') || '%'
       OR source ILIKE '%' || sqlc.arg('query') || '%')
  AND (sqlc.arg('severity')::text = '' OR severity = sqlc.arg('severity'))
  AND (sqlc.arg('source')::text = '' OR source = sqlc.arg('source'))
  AND (sqlc.arg('status')::text = '' OR (sqlc.arg('status') = 'open' AND resolved_at IS NULL)
       OR (sqlc.arg('status') = 'resolved' AND resolved_at IS NOT NULL));

-- name: AuditGetSecurityEvent :one
SELECT id, user_id, session_id, event_type, severity, source,
       COALESCE(client_ip::text, '') AS client_ip, details, resolved_at, resolved_by, occurred_at
FROM audit.security_events
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id');
