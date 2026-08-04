-- name: UpsertCoreMenu :one
INSERT INTO sys.menus (
    tenant_id, parent_id, permission_id, code, title, menu_type,
    route_path, component_key, icon, affix, sort_order, metadata, status
)
VALUES (
    NULL, $1, $2,
    $3, $4, $5, $6, $7, $8, $9, $10, $11, 'active'
)
ON CONFLICT (code) WHERE tenant_id IS NULL AND deleted_at IS NULL DO UPDATE SET
    parent_id = EXCLUDED.parent_id,
    permission_id = EXCLUDED.permission_id,
    title = EXCLUDED.title,
    menu_type = EXCLUDED.menu_type,
    route_path = EXCLUDED.route_path,
    component_key = EXCLUDED.component_key,
    icon = EXCLUDED.icon,
    affix = EXCLUDED.affix,
    sort_order = EXCLUDED.sort_order,
    metadata = EXCLUDED.metadata,
    status = 'active',
    updated_at = now()
RETURNING id, permission_id;

-- name: GetActivePermissionIDByCode :one
SELECT id FROM iam.permissions WHERE code::text = $1 AND status = 'active';

-- name: GrantAllCoreMenusToRole :execrows
INSERT INTO sys.role_menus (tenant_id, role_id, menu_id)
SELECT $1, $2, menu.id
FROM sys.menus menu
WHERE menu.tenant_id IS NULL AND menu.deleted_at IS NULL AND menu.status = 'active'
ON CONFLICT (tenant_id, role_id, menu_id) DO NOTHING;

-- name: SyncSystemAdminMenus :execrows
INSERT INTO sys.role_menus (tenant_id, role_id, menu_id)
SELECT role.tenant_id, role.id, menu.id
FROM iam.roles role
CROSS JOIN sys.menus menu
WHERE role.code = 'super-admin'
  AND role.is_system
  AND role.status = 'active'
  AND role.deleted_at IS NULL
  AND menu.tenant_id IS NULL
  AND menu.status = 'active'
  AND menu.deleted_at IS NULL
ON CONFLICT (tenant_id, role_id, menu_id) DO NOTHING;

-- name: ListEffectiveMenus :many
SELECT DISTINCT menu.id, menu.parent_id, menu.code::text AS code, menu.title,
       menu.menu_type, menu.route_path, menu.component_key, menu.icon,
       menu.affix, menu.sort_order, menu.metadata
FROM iam.user_roles user_role
JOIN iam.roles role
  ON role.tenant_id = user_role.tenant_id
 AND role.id = user_role.role_id
 AND role.status = 'active'
 AND role.deleted_at IS NULL
JOIN sys.role_menus role_menu
  ON role_menu.tenant_id = user_role.tenant_id
 AND role_menu.role_id = user_role.role_id
JOIN sys.menus menu
  ON menu.id = role_menu.menu_id
 AND menu.status = 'active'
 AND menu.deleted_at IS NULL
LEFT JOIN iam.role_permissions role_permission
  ON role_permission.tenant_id = user_role.tenant_id
 AND role_permission.role_id = user_role.role_id
 AND role_permission.permission_id = menu.permission_id
WHERE user_role.tenant_id = $1
  AND user_role.user_id = $2
  AND user_role.valid_from <= now()
  AND (user_role.valid_until IS NULL OR user_role.valid_until > now())
  AND (menu.permission_id IS NULL OR role_permission.permission_id IS NOT NULL)
ORDER BY menu.sort_order, menu.code::text;
