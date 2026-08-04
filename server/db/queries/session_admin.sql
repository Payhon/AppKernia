-- name: SessionAdminList :many
SELECT s.id, s.user_id, COALESCE(u.email, '') AS email, u.display_name, s.audience,
       COALESCE(d.platform, CASE s.audience WHEN 'ak-mobile' THEN 'unknown' WHEN 'ak-admin' THEN 'web' ELSE 'unknown' END) AS platform,
       COALESCE(NULLIF(d.device_name, ''), NULLIF(d.model, ''), '')::text AS device_hint,
       COALESCE(host(s.ip_address), '')::text AS ip_address,
       (CASE WHEN s.status = 'active' AND s.absolute_expires_at <= now() THEN 'expired' ELSE s.status END)::text AS effective_status,
       s.id = sqlc.arg('current_session_id') AS current,
       s.last_seen_at, s.absolute_expires_at, s.revoked_at
FROM iam.sessions s
JOIN iam.users u ON u.id = s.user_id
LEFT JOIN iam.devices d ON d.id = s.device_id
WHERE s.tenant_id = sqlc.arg('tenant_id')
  AND s.last_seen_at >= sqlc.arg('from_at') AND s.last_seen_at <= sqlc.arg('to_at')
  AND (sqlc.arg('query')::text = '' OR u.display_name ILIKE '%' || sqlc.arg('query') || '%' OR u.email ILIKE '%' || sqlc.arg('query') || '%')
  AND (sqlc.arg('audience')::text = '' OR s.audience = sqlc.arg('audience'))
  AND (sqlc.arg('platform')::text = '' OR COALESCE(d.platform, CASE s.audience WHEN 'ak-admin' THEN 'web' ELSE 'unknown' END) = sqlc.arg('platform'))
  AND (sqlc.arg('ip')::text = '' OR COALESCE(host(s.ip_address), '') ILIKE '%' || sqlc.arg('ip') || '%')
  AND (sqlc.arg('status')::text = '' OR CASE WHEN s.status = 'active' AND s.absolute_expires_at <= now() THEN 'expired' ELSE s.status END = sqlc.arg('status'))
ORDER BY s.last_seen_at DESC, s.id DESC
LIMIT sqlc.arg('page_size') OFFSET sqlc.arg('page_offset');

-- name: SessionAdminCount :one
SELECT count(*)
FROM iam.sessions s
JOIN iam.users u ON u.id = s.user_id
LEFT JOIN iam.devices d ON d.id = s.device_id
WHERE s.tenant_id = sqlc.arg('tenant_id')
  AND s.last_seen_at >= sqlc.arg('from_at') AND s.last_seen_at <= sqlc.arg('to_at')
  AND (sqlc.arg('query')::text = '' OR u.display_name ILIKE '%' || sqlc.arg('query') || '%' OR u.email ILIKE '%' || sqlc.arg('query') || '%')
  AND (sqlc.arg('audience')::text = '' OR s.audience = sqlc.arg('audience'))
  AND (sqlc.arg('platform')::text = '' OR COALESCE(d.platform, CASE s.audience WHEN 'ak-admin' THEN 'web' ELSE 'unknown' END) = sqlc.arg('platform'))
  AND (sqlc.arg('ip')::text = '' OR COALESCE(host(s.ip_address), '') ILIKE '%' || sqlc.arg('ip') || '%')
  AND (sqlc.arg('status')::text = '' OR CASE WHEN s.status = 'active' AND s.absolute_expires_at <= now() THEN 'expired' ELSE s.status END = sqlc.arg('status'));

-- name: SessionAdminLockForRevoke :one
SELECT id, id = sqlc.arg('current_session_id') AS current
FROM iam.sessions
WHERE id = sqlc.arg('id') AND tenant_id = sqlc.arg('tenant_id') AND status = 'active' AND revoked_at IS NULL AND absolute_expires_at > now()
FOR UPDATE;

-- name: SessionAdminRevoke :execrows
UPDATE iam.sessions SET status='revoked', revoked_at=now(), revoke_reason='admin_force_logout', access_token_version=access_token_version+1
WHERE id=sqlc.arg('id') AND tenant_id=sqlc.arg('tenant_id') AND status='active' AND revoked_at IS NULL;

-- name: SessionAdminRevokeRefreshTokens :exec
UPDATE iam.refresh_tokens SET revoked_at=COALESCE(revoked_at, now()) WHERE session_id=sqlc.arg('session_id') AND revoked_at IS NULL;

-- name: SessionAdminInsertAudit :exec
INSERT INTO audit.operation_logs (
  tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,request_summary,after_data,succeeded
) VALUES (
  sqlc.arg('tenant_id'),sqlc.arg('actor_id'),sqlc.arg('actor_session_id'),sqlc.arg('request_id'),'iam','iam.session.revoke','iam.session.revoke','session',sqlc.arg('resource_id'),'DELETE','/admin-api/v1/online-sessions/{id}',200,sqlc.arg('client_ip'),sqlc.arg('user_agent'),'{}'::jsonb,jsonb_build_object('revoked',true,'current',sqlc.arg('current')::boolean),true
);
