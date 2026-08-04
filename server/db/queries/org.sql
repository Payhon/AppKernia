-- name: ListOrgUnits :many
SELECT u.id, u.parent_id, u.code::text AS code, u.name, u.unit_type, u.phone,
       coalesce(u.email::text, '') AS email, u.sort_order, u.status, u.updated_at,
       (SELECT count(*) FROM org.user_units uu WHERE uu.tenant_id = u.tenant_id AND uu.unit_id = u.id) AS direct_member_count,
       (SELECT count(*) FROM org.units child WHERE child.tenant_id = u.tenant_id AND child.parent_id = u.id AND child.deleted_at IS NULL) AS child_count
FROM org.units u
WHERE u.tenant_id = sqlc.arg('tenant_id') AND u.deleted_at IS NULL
ORDER BY u.sort_order, u.name, u.id;

-- name: GetOrgUnitForUpdate :one
SELECT id, parent_id, code::text AS code, name, unit_type, phone, coalesce(email::text, '') AS email,
       sort_order, status, updated_at
FROM org.units
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id') AND deleted_at IS NULL
FOR UPDATE;

-- name: CreateOrgUnit :one
INSERT INTO org.units (tenant_id, parent_id, code, name, unit_type, phone, email, sort_order, status)
VALUES (sqlc.arg('tenant_id'), sqlc.narg('parent_id'), sqlc.arg('code'), sqlc.arg('name'),
        sqlc.arg('unit_type'), sqlc.narg('phone'), sqlc.narg('email'), sqlc.arg('sort_order'), sqlc.arg('status'))
RETURNING id, parent_id, code::text AS code, name, unit_type, phone, coalesce(email::text, '') AS email,
          sort_order, status, updated_at;

-- name: UpdateOrgUnit :one
UPDATE org.units
SET code = sqlc.arg('code'), name = sqlc.arg('name'), unit_type = sqlc.arg('unit_type'),
    phone = sqlc.narg('phone'), email = sqlc.narg('email'), sort_order = sqlc.arg('sort_order'), status = sqlc.arg('status')
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, parent_id, code::text AS code, name, unit_type, phone, coalesce(email::text, '') AS email,
          sort_order, status, updated_at;

-- name: OrgUnitParentExists :one
SELECT EXISTS(
    SELECT 1 FROM org.units
    WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('parent_id') AND deleted_at IS NULL
);

-- name: MoveOrgUnit :one
UPDATE org.units
SET parent_id = sqlc.narg('parent_id'), sort_order = sqlc.arg('sort_order')
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, parent_id, code::text AS code, name, unit_type, phone, coalesce(email::text, '') AS email,
          sort_order, status, updated_at;

-- name: GetOrgUnitOccupancy :one
SELECT
  (SELECT count(*) FROM org.units child WHERE child.tenant_id = sqlc.arg('tenant_id') AND child.parent_id = sqlc.arg('id') AND child.deleted_at IS NULL) AS child_count,
  (SELECT count(*) FROM org.user_units member WHERE member.tenant_id = sqlc.arg('tenant_id') AND member.unit_id = sqlc.arg('id')) AS member_count;

-- name: SoftDeleteOrgUnit :execrows
UPDATE org.units SET deleted_at = now()
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: ListOrgPositions :many
SELECT p.id, p.code::text AS code, p.name, p.description, p.sort_order, p.status, p.updated_at,
       (SELECT count(*) FROM org.user_positions up WHERE up.tenant_id = p.tenant_id AND up.position_id = p.id) AS member_count
FROM org.positions p
WHERE p.tenant_id = sqlc.arg('tenant_id') AND p.deleted_at IS NULL
  AND (sqlc.arg('query')::text = '' OR p.code ILIKE '%' || sqlc.arg('query')::text || '%' OR p.name ILIKE '%' || sqlc.arg('query')::text || '%')
  AND (sqlc.arg('status')::text = '' OR p.status = sqlc.arg('status')::text)
  AND (sqlc.narg('unit_id')::uuid IS NULL OR EXISTS (
      SELECT 1 FROM org.user_positions up
      WHERE up.tenant_id = p.tenant_id AND up.position_id = p.id AND up.unit_id = sqlc.narg('unit_id')::uuid
  ))
ORDER BY p.sort_order, p.name, p.id;

-- name: GetOrgPositionForUpdate :one
SELECT id, code::text AS code, name, description, sort_order, status, updated_at
FROM org.positions
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id') AND deleted_at IS NULL
FOR UPDATE;

-- name: CreateOrgPosition :one
INSERT INTO org.positions (tenant_id, code, name, description, sort_order, status)
VALUES (sqlc.arg('tenant_id'), sqlc.arg('code'), sqlc.arg('name'), sqlc.narg('description'), sqlc.arg('sort_order'), sqlc.arg('status'))
RETURNING id, code::text AS code, name, description, sort_order, status, updated_at;

-- name: UpdateOrgPosition :one
UPDATE org.positions
SET code = sqlc.arg('code'), name = sqlc.arg('name'), description = sqlc.narg('description'),
    sort_order = sqlc.arg('sort_order'), status = sqlc.arg('status')
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, code::text AS code, name, description, sort_order, status, updated_at;

-- name: CountOrgPositionMembers :one
SELECT count(*) FROM org.user_positions
WHERE tenant_id = sqlc.arg('tenant_id') AND position_id = sqlc.arg('id');

-- name: SoftDeleteOrgPosition :execrows
UPDATE org.positions SET deleted_at = now()
WHERE tenant_id = sqlc.arg('tenant_id') AND id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: InsertOrgOperationAudit :exec
INSERT INTO audit.operation_logs (
  tenant_id, user_id, session_id, request_id, module_code, action_name, resource_type,
  resource_id, http_method, request_path, response_status, client_ip, user_agent,
  before_data, after_data, succeeded
)
VALUES (
  sqlc.arg('tenant_id'), sqlc.arg('user_id'), sqlc.arg('session_id'), sqlc.arg('request_id'),
  'org', sqlc.arg('action_name'), sqlc.arg('resource_type'), sqlc.arg('resource_id'),
  sqlc.arg('http_method'), sqlc.arg('request_path'), sqlc.arg('response_status'),
  sqlc.narg('client_ip'), sqlc.narg('user_agent'), sqlc.arg('before_data')::jsonb,
  sqlc.arg('after_data')::jsonb, true
);
