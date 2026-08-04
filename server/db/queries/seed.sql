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
