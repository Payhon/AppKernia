-- name: GetLoginProviderConfigPublic :one
SELECT id,tenant_id,name,description,provider_code,external_client_id,
       config_schema_version,public_config,secret_field_names,
       secret_ciphertext IS NOT NULL AS has_secret,
       credential_fingerprint,status,last_preflight_at,last_preflight_status,
       last_preflight_issues,lock_version,created_at,updated_at
FROM sys.login_provider_configs
WHERE tenant_id=sqlc.arg('tenant_id') AND id=sqlc.arg('id') AND deleted_at IS NULL;

-- name: ListRuntimeLoginProviders :many
SELECT b.tenant_id,b.app_id,b.login_provider_config_id,b.provider_code,b.enabled,b.sort_order,
       c.external_client_id,c.config_schema_version,c.public_config,c.secret_ciphertext,
       c.secret_key_version,c.secret_field_names,c.status,c.last_preflight_status
FROM app.application_login_provider_bindings b
JOIN sys.login_provider_configs c
  ON c.tenant_id=b.tenant_id AND c.id=b.login_provider_config_id AND c.provider_code=b.provider_code
WHERE b.app_id=sqlc.arg('app_id') AND c.deleted_at IS NULL
ORDER BY b.sort_order,b.provider_code;

-- name: GetAppLoginIdentifierTarget :one
SELECT i.id,i.tenant_id,i.app_id,i.user_id,i.identifier_type,i.normalized_value::text AS normalized_value,
       i.display_hint,i.verified_at,i.status,u.locale
FROM app.user_login_identifiers i
JOIN app.user_memberships m
  ON m.tenant_id=i.tenant_id AND m.app_id=i.app_id AND m.user_id=i.user_id
JOIN iam.users u ON u.id=i.user_id
WHERE i.id=sqlc.arg('id') AND i.app_id=sqlc.arg('app_id') AND i.user_id=sqlc.arg('user_id')
  AND i.status='active' AND i.verified_at IS NOT NULL AND m.status='active'
  AND u.status='active' AND u.deleted_at IS NULL;

-- name: ListAppProfileIdentifiers :many
SELECT identifier_type,normalized_value::text AS normalized_value
FROM app.user_login_identifiers
WHERE app_id=sqlc.arg('app_id') AND user_id=sqlc.arg('user_id')
  AND status='active' AND verified_at IS NOT NULL
  AND identifier_type IN ('email','mobile')
ORDER BY identifier_type;
