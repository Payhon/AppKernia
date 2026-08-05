-- name: UpsertCorePermission :one
INSERT INTO iam.permissions (
    code, name, module_code, resource_name, action_name, permission_kind, status
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    module_code = EXCLUDED.module_code,
    resource_name = EXCLUDED.resource_name,
    action_name = EXCLUDED.action_name,
    permission_kind = EXCLUDED.permission_kind,
    status = EXCLUDED.status,
    updated_at = now()
RETURNING id;

-- name: UpsertSystemAdminRole :one
INSERT INTO iam.roles (
    tenant_id, code, name, description, role_type, data_scope,
    is_default, is_system, status
)
VALUES ($1, 'super-admin', 'Super Administrator', 'Core bootstrap administrator role',
        'system', 'tenant', false, true, 'active')
ON CONFLICT (tenant_id, code) WHERE deleted_at IS NULL DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    status = 'active',
    updated_at = now()
RETURNING id;

-- name: GrantAllActivePermissionsToRole :execrows
INSERT INTO iam.role_permissions (tenant_id, role_id, permission_id, granted_by)
SELECT $1, $2, permission.id, $3
FROM iam.permissions permission
WHERE permission.status = 'active'
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

-- name: SyncSystemAdminPermissions :execrows
INSERT INTO iam.role_permissions (tenant_id, role_id, permission_id)
SELECT role.tenant_id, role.id, permission.id
FROM iam.roles role
CROSS JOIN iam.permissions permission
WHERE role.code = 'super-admin'
  AND role.is_system
  AND role.status = 'active'
  AND role.deleted_at IS NULL
  AND permission.status = 'active'
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

-- name: SyncDefaultTenantRoles :execrows
INSERT INTO iam.roles (
    tenant_id, code, name, description, role_type, data_scope,
    is_default, is_system, status
)
SELECT tenant.id, definition.code, definition.name, definition.description,
       'system', definition.data_scope, definition.is_default, true, 'active'
FROM iam.tenants AS tenant
CROSS JOIN (
    VALUES
      ('tenant-admin'::public.citext, 'Tenant Administrator'::varchar, 'Built-in tenant administrator role'::varchar, 'tenant'::varchar, false),
      ('member'::public.citext, 'Member'::varchar, 'Built-in tenant member role'::varchar, 'self'::varchar, true)
) AS definition(code, name, description, data_scope, is_default)
WHERE tenant.status = 'active' AND tenant.deleted_at IS NULL
ON CONFLICT (tenant_id, code) WHERE deleted_at IS NULL DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    data_scope = EXCLUDED.data_scope,
    is_default = EXCLUDED.is_default,
    is_system = true,
    status = 'active',
    updated_at = now();

-- name: ListActiveTenantIDsForConfigSeed :many
SELECT id
FROM iam.tenants
WHERE status = 'active' AND deleted_at IS NULL
ORDER BY id;

-- name: UpsertTenantCoreConfig :one
INSERT INTO sys.config_items (
    tenant_id, module_code, config_group, config_key, display_name, value_type,
    value_json, default_value_json, is_secret, is_public, validation_schema,
    description, sort_order, status
)
VALUES (
    sqlc.arg('tenant_id'), sqlc.arg('module_code'), sqlc.arg('config_group'),
    sqlc.arg('config_key'), sqlc.arg('display_name'), sqlc.arg('value_type'),
    sqlc.narg('value_json'), sqlc.narg('default_value_json'), sqlc.arg('is_secret'),
    sqlc.arg('is_public'), sqlc.arg('validation_schema'), sqlc.arg('description'),
    sqlc.arg('sort_order'), sqlc.arg('status')
)
ON CONFLICT (tenant_id, module_code, config_group, config_key)
    WHERE tenant_id IS NOT NULL
DO UPDATE SET
    display_name = EXCLUDED.display_name,
    default_value_json = CASE WHEN sys.config_items.is_secret THEN NULL ELSE EXCLUDED.default_value_json END,
    is_public = CASE WHEN sys.config_items.is_secret THEN false ELSE EXCLUDED.is_public END,
    validation_schema = EXCLUDED.validation_schema,
    description = EXCLUDED.description,
    sort_order = EXCLUDED.sort_order,
    updated_at = now()
RETURNING id;

-- name: UpsertCoreRegion :exec
INSERT INTO sys.regions (
    code, parent_code, level, name, full_name, postal_code,
    longitude, latitude, status, metadata
)
VALUES (
    sqlc.arg('code'), sqlc.narg('parent_code'), sqlc.arg('level'),
    sqlc.arg('name'), sqlc.narg('full_name'), sqlc.narg('postal_code'),
    sqlc.narg('longitude'), sqlc.narg('latitude'), sqlc.arg('status'),
    sqlc.arg('metadata')
)
ON CONFLICT (code) DO UPDATE SET
    parent_code = EXCLUDED.parent_code,
    level = EXCLUDED.level,
    name = EXCLUDED.name,
    full_name = EXCLUDED.full_name,
    postal_code = COALESCE(EXCLUDED.postal_code, sys.regions.postal_code),
    longitude = COALESCE(EXCLUDED.longitude, sys.regions.longitude),
    latitude = COALESCE(EXCLUDED.latitude, sys.regions.latitude),
    status = EXCLUDED.status,
    metadata = sys.regions.metadata || EXCLUDED.metadata,
    updated_at = now();
