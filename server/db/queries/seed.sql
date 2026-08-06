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

-- name: UpsertCoreModule :exec
INSERT INTO sys.modules (
    code, name, name_key, version, description, description_key,
    capabilities, status
)
VALUES (
    sqlc.arg('code'), sqlc.arg('name'), sqlc.arg('name_key'),
    sqlc.arg('version'), sqlc.arg('description'),
    sqlc.arg('description_key'), sqlc.arg('capabilities'),
    sqlc.arg('status')
)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    name_key = EXCLUDED.name_key,
    version = EXCLUDED.version,
    description = EXCLUDED.description,
    description_key = EXCLUDED.description_key,
    capabilities = EXCLUDED.capabilities,
    status = EXCLUDED.status,
    updated_at = now();

-- name: DeleteModulesOutsideCoreCatalog :execrows
DELETE FROM sys.modules
WHERE code::text <> ALL(sqlc.arg('codes')::text[]);

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
    status = EXCLUDED.status,
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
    parent_code = CASE WHEN sys.regions.is_manually_managed THEN sys.regions.parent_code ELSE EXCLUDED.parent_code END,
    level = CASE WHEN sys.regions.is_manually_managed THEN sys.regions.level ELSE EXCLUDED.level END,
    name = CASE WHEN sys.regions.is_manually_managed THEN sys.regions.name ELSE EXCLUDED.name END,
    full_name = CASE WHEN sys.regions.is_manually_managed THEN sys.regions.full_name ELSE EXCLUDED.full_name END,
    postal_code = CASE WHEN sys.regions.is_manually_managed THEN sys.regions.postal_code ELSE COALESCE(EXCLUDED.postal_code, sys.regions.postal_code) END,
    longitude = CASE WHEN sys.regions.is_manually_managed THEN sys.regions.longitude ELSE COALESCE(EXCLUDED.longitude, sys.regions.longitude) END,
    latitude = CASE WHEN sys.regions.is_manually_managed THEN sys.regions.latitude ELSE COALESCE(EXCLUDED.latitude, sys.regions.latitude) END,
    status = CASE WHEN sys.regions.is_manually_managed THEN sys.regions.status ELSE EXCLUDED.status END,
    metadata = CASE WHEN sys.regions.is_manually_managed THEN sys.regions.metadata ELSE sys.regions.metadata || EXCLUDED.metadata END,
    updated_at = CASE WHEN sys.regions.is_manually_managed THEN sys.regions.updated_at ELSE now() END;

-- name: UpsertCoreDictionaryType :one
INSERT INTO sys.dict_types (
    code, name, name_key, description, description_key,
    is_system, visibility, extension_policy, status
)
VALUES (
    sqlc.arg('code'), sqlc.arg('name'), sqlc.arg('name_key'),
    sqlc.arg('description'), sqlc.arg('description_key'), true,
    sqlc.arg('visibility'), sqlc.arg('extension_policy'), sqlc.arg('status')
)
ON CONFLICT (code) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    name_key = EXCLUDED.name_key,
    description = EXCLUDED.description,
    description_key = EXCLUDED.description_key,
    is_system = true,
    visibility = EXCLUDED.visibility,
    extension_policy = EXCLUDED.extension_policy,
    status = EXCLUDED.status,
    updated_at = now()
RETURNING id;

-- name: UpsertCoreDictionaryItem :exec
INSERT INTO sys.dict_items (
    dict_type_id, tenant_id, item_value, label, locale, sort_order,
    is_default, extra, status
)
VALUES (
    sqlc.arg('dict_type_id'), NULL, sqlc.arg('item_value'), sqlc.arg('label'),
    sqlc.arg('locale'), sqlc.arg('sort_order'), sqlc.arg('is_default'),
    sqlc.arg('extra'), sqlc.arg('status')
)
ON CONFLICT (dict_type_id, item_value, COALESCE(locale, ''))
    WHERE tenant_id IS NULL
DO UPDATE SET
    label = EXCLUDED.label,
    sort_order = EXCLUDED.sort_order,
    is_default = EXCLUDED.is_default,
    extra = EXCLUDED.extra,
    status = EXCLUDED.status,
    updated_at = now();

-- name: UpsertCoreNotificationTemplate :exec
INSERT INTO notify.templates (
    code, name, channel, locale, subject_template, body_template,
    body_format, variables_schema, status
)
VALUES (
    sqlc.arg('code'), sqlc.arg('name'), sqlc.arg('channel'), sqlc.arg('locale'),
    sqlc.narg('subject_template'), sqlc.arg('body_template'), sqlc.arg('body_format'),
    sqlc.arg('variables_schema'), sqlc.arg('status')
)
ON CONFLICT (code, channel, COALESCE(locale, ''))
    WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    subject_template = EXCLUDED.subject_template,
    body_template = EXCLUDED.body_template,
    body_format = EXCLUDED.body_format,
    variables_schema = EXCLUDED.variables_schema,
    status = EXCLUDED.status,
    updated_at = now();
