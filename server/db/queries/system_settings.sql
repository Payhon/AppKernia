-- name: GetTenantConfigForUpdate :one
SELECT * FROM sys.config_items WHERE tenant_id = $1 AND id = $2 FOR UPDATE;

-- name: ListVisibleConfigs :many
SELECT * FROM sys.config_items WHERE tenant_id IS NULL OR tenant_id = $1 ORDER BY sort_order, config_key;

-- name: CreateTenantConfig :one
INSERT INTO sys.config_items (tenant_id, module_code, config_group, config_key, display_name, value_type, value_json, default_value_json, is_secret, secret_ciphertext, secret_key_version, is_public, validation_schema, description, sort_order, status, created_by, updated_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$17) RETURNING *;

-- name: GetVisibleDictType :one
SELECT * FROM sys.dict_types WHERE (tenant_id IS NULL OR tenant_id = $1) AND id = $2;

-- name: ListVisibleDictTypes :many
SELECT * FROM sys.dict_types WHERE tenant_id IS NULL OR tenant_id = $1 ORDER BY name, id;

-- name: CreateTenantDictType :one
INSERT INTO sys.dict_types (tenant_id, code, name, description, is_system, status)
VALUES ($1,$2,$3,$4,false,$5) RETURNING *;

-- name: ListDictItems :many
SELECT * FROM sys.dict_items WHERE dict_type_id = $1 ORDER BY sort_order, id;

-- name: CreateDictItem :one
INSERT INTO sys.dict_items (dict_type_id,item_value,label,locale,color,css_class,sort_order,is_default,extra,status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING *;

-- name: GetDictItemForUpdate :one
SELECT i.* FROM sys.dict_items i JOIN sys.dict_types d ON d.id=i.dict_type_id
WHERE (d.tenant_id IS NULL OR d.tenant_id=$1) AND i.id=$2 FOR UPDATE OF i;

-- name: DeleteDictItem :execrows
DELETE FROM sys.dict_items WHERE id=$1;
