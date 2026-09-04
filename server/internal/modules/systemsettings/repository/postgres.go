package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	settings "github.com/appkernia/appkernia/server/internal/modules/systemsettings/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

// ListPublicConfigs returns only active, global, explicitly public, non-secret
// values. Tenant-scoped settings are never exposed through an anonymous API.
func (r *Postgres) ListPublicConfigs(ctx context.Context) (map[string]json.RawMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT concat_ws('.', module_code, config_group, config_key::text),
		       coalesce(value_json, default_value_json, 'null'::jsonb)
		FROM sys.config_items
		WHERE tenant_id IS NULL
		  AND is_public = true
		  AND is_secret = false
		  AND status = 'active'
		ORDER BY module_code, config_group, config_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make(map[string]json.RawMessage)
	for rows.Next() {
		var key string
		var value json.RawMessage
		if err = rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, rows.Err()
}

// GetGlobalString returns the effective value of one active, non-secret global
// string setting. Runtime consumers use this narrow method instead of loading
// the administrative configuration model or tenant overlays.
func (r *Postgres) GetGlobalString(ctx context.Context, moduleCode, group, key string) (string, error) {
	var value string
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(value_json, default_value_json) #>> '{}'
		FROM sys.config_items
		WHERE tenant_id IS NULL
		  AND module_code = $1
		  AND config_group = $2
		  AND config_key = $3
		  AND value_type = 'string'
		  AND is_secret = false
		  AND status = 'active'
		  AND jsonb_typeof(COALESCE(value_json, default_value_json)) = 'string'`, moduleCode, group, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", settings.ErrNotFound
	}
	return value, err
}

func (r *Postgres) ListRegions(ctx context.Context, f settings.RegionFilter) ([]settings.Region, error) {
	where, args := " WHERE r.deleted_at IS NULL", []any{}
	add := func(fragment string, value any) {
		args = append(args, value)
		where += fmt.Sprintf(fragment, len(args))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where += fmt.Sprintf(" AND (r.code ILIKE $%d OR r.name ILIKE $%d OR coalesce(r.full_name,'') ILIKE $%d)", n, n, n)
	} else if f.ParentCode != "" {
		add(" AND r.parent_code=$%d", f.ParentCode)
	} else {
		where += " AND r.parent_code IS NULL"
	}
	if f.Level != nil {
		add(" AND r.level=$%d", *f.Level)
	}
	if f.Status != "" {
		add(" AND r.status=$%d", f.Status)
	}
	args = append(args, f.Limit)
	rows, err := r.pool.Query(ctx, `SELECT r.code,r.parent_code,r.level,r.name,coalesce(r.full_name,''),coalesce(r.postal_code,''),r.longitude::float8,r.latitude::float8,r.status,EXISTS(SELECT 1 FROM sys.regions c WHERE c.parent_code=r.code AND c.deleted_at IS NULL),r.version,r.updated_at FROM sys.regions r`+where+fmt.Sprintf(" ORDER BY r.code LIMIT $%d", len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []settings.Region{}
	for rows.Next() {
		var x settings.Region
		if err = rows.Scan(&x.Code, &x.ParentCode, &x.Level, &x.Name, &x.FullName, &x.PostalCode, &x.Longitude, &x.Latitude, &x.Status, &x.HasChildren, &x.Version, &x.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

const regionSelect = `SELECT r.code,r.parent_code,r.level,r.name,coalesce(r.full_name,''),coalesce(r.postal_code,''),r.longitude::float8,r.latitude::float8,r.status,EXISTS(SELECT 1 FROM sys.regions c WHERE c.parent_code=r.code AND c.deleted_at IS NULL),r.version,r.updated_at FROM sys.regions r`

func scanRegion(row scanner) (settings.Region, error) {
	var x settings.Region
	err := row.Scan(&x.Code, &x.ParentCode, &x.Level, &x.Name, &x.FullName, &x.PostalCode, &x.Longitude, &x.Latitude, &x.Status, &x.HasChildren, &x.Version, &x.UpdatedAt)
	return x, err
}

func getRegion(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, code string, lock bool) (settings.Region, error) {
	sql := regionSelect + ` WHERE r.code=$1 AND r.deleted_at IS NULL`
	if lock {
		sql += ` FOR UPDATE OF r`
	}
	x, err := scanRegion(q.QueryRow(ctx, sql, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return x, settings.ErrNotFound
	}
	return x, err
}

func (r *Postgres) CreateRegion(ctx context.Context, p settings.Principal, in settings.RegionCreateInput) (settings.Region, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return settings.Region{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var code string
	err = tx.QueryRow(ctx, `INSERT INTO sys.regions(code,parent_code,level,name,full_name,postal_code,longitude,latitude,status,metadata,is_manually_managed) SELECT $1,parent.code,parent.level+1,$3,$4,nullif($5,''),$6,$7,$8,jsonb_build_object('management_origin','admin'),true FROM sys.regions parent WHERE parent.code=$2 AND parent.deleted_at IS NULL AND parent.level IN (0,1) RETURNING code`, in.Code, in.ParentCode, in.Name, in.FullName, in.PostalCode, in.Longitude, in.Latitude, in.Status).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return settings.Region{}, settings.ErrInvalid
	}
	if err != nil {
		return settings.Region{}, classify(err)
	}
	after, err := getRegion(ctx, tx, code, false)
	if err != nil {
		return settings.Region{}, err
	}
	if err = auditRegion(ctx, tx, p, "sys.region.create", code, "POST", "/admin-api/v1/regions", nil, after); err != nil {
		return settings.Region{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return settings.Region{}, err
	}
	return after, nil
}

func (r *Postgres) UpdateRegion(ctx context.Context, p settings.Principal, code string, in settings.RegionUpdateInput) (settings.Region, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return settings.Region{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getRegion(ctx, tx, code, true)
	if err != nil {
		return settings.Region{}, err
	}
	if before.Version != in.Version {
		return settings.Region{}, settings.ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.regions SET name=$1,full_name=$2,postal_code=nullif($3,''),longitude=$4,latitude=$5,status=$6,version=version+1,is_manually_managed=true,updated_at=now() WHERE code=$7 AND deleted_at IS NULL AND version=$8`, in.Name, in.FullName, in.PostalCode, in.Longitude, in.Latitude, in.Status, code, in.Version)
	if err != nil {
		return settings.Region{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return settings.Region{}, settings.ErrConflict
	}
	after, err := getRegion(ctx, tx, code, false)
	if err != nil {
		return settings.Region{}, err
	}
	if err = auditRegion(ctx, tx, p, "sys.region.update", code, "PATCH", "/admin-api/v1/regions/"+code, before, after); err != nil {
		return settings.Region{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return settings.Region{}, err
	}
	return after, nil
}

func (r *Postgres) DeleteRegion(ctx context.Context, p settings.Principal, code string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getRegion(ctx, tx, code, true)
	if err != nil {
		return err
	}
	var hasChildren bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sys.regions WHERE parent_code=$1 AND deleted_at IS NULL)`, code).Scan(&hasChildren); err != nil {
		return err
	}
	if hasChildren {
		return settings.ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.regions SET deleted_at=now(),version=version+1,is_manually_managed=true,updated_at=now() WHERE code=$1 AND deleted_at IS NULL`, code)
	if err != nil {
		return classify(err)
	}
	if tag.RowsAffected() != 1 {
		return settings.ErrNotFound
	}
	after := map[string]any{"code": code, "deleted": true, "version": before.Version + 1}
	if err = auditRegion(ctx, tx, p, "sys.region.delete", code, "DELETE", "/admin-api/v1/regions/"+code, before, after); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const configSelect = `SELECT c.id,c.tenant_id,c.module_code,c.config_group,c.config_key::text,c.display_name,c.value_type,c.value_json,c.default_value_json,c.is_secret,(c.secret_ciphertext IS NOT NULL),c.secret_key_version,c.is_public,c.validation_schema,coalesce(c.description,''),c.sort_order,c.status,c.version,(c.tenant_id IS NULL),c.created_at,c.updated_at FROM sys.config_items c`

type scanner interface{ Scan(...any) error }

func scanConfig(row scanner) (settings.ConfigItem, error) {
	var x settings.ConfigItem
	err := row.Scan(&x.ID, &x.TenantID, &x.ModuleCode, &x.ConfigGroup, &x.ConfigKey, &x.DisplayName, &x.ValueType, &x.Value, &x.DefaultValue, &x.IsSecret, &x.SecretConfigured, &x.SecretKeyVersion, &x.IsPublic, &x.ValidationSchema, &x.Description, &x.SortOrder, &x.Status, &x.Version, &x.IsLocked, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}

func configWhere(tenantID uuid.UUID, f settings.PageFilter) (string, []any) {
	where := ` WHERE (c.tenant_id IS NULL OR c.tenant_id=$1)`
	args := []any{tenantID}
	add := func(sql string, v any) { args = append(args, v); where += fmt.Sprintf(sql, len(args)) }
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where += fmt.Sprintf(` AND (c.config_key::text ILIKE $%d OR c.display_name ILIKE $%d OR c.description ILIKE $%d)`, n, n, n)
	}
	if f.ModuleCode != "" {
		add(` AND c.module_code=$%d`, f.ModuleCode)
	}
	if f.Group != "" {
		add(` AND c.config_group=$%d`, f.Group)
	}
	if f.ValueType != "" {
		add(` AND c.value_type=$%d`, f.ValueType)
	}
	if f.Status != "" {
		add(` AND c.status=$%d`, f.Status)
	}
	if f.IsPublic != nil {
		add(` AND c.is_public=$%d`, *f.IsPublic)
	}
	if f.IsSecret != nil {
		add(` AND c.is_secret=$%d`, *f.IsSecret)
	}
	return where, args
}
func (r *Postgres) ListConfigs(ctx context.Context, tenantID uuid.UUID, f settings.PageFilter) (settings.ConfigPage, error) {
	where, args := configWhere(tenantID, f)
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM sys.config_items c`+where, args...).Scan(&total); err != nil {
		return settings.ConfigPage{}, err
	}
	order := `c.sort_order,c.config_key`
	switch f.Sort {
	case "updated_desc":
		order = `c.updated_at DESC,c.id`
	case "key":
		order = `c.config_key,c.id`
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, configSelect+where+fmt.Sprintf(` ORDER BY %s LIMIT $%d OFFSET $%d`, order, len(args)-1, len(args)), args...)
	if err != nil {
		return settings.ConfigPage{}, err
	}
	defer rows.Close()
	items := []settings.ConfigItem{}
	for rows.Next() {
		x, e := scanConfig(rows)
		if e != nil {
			return settings.ConfigPage{}, e
		}
		items = append(items, x)
	}
	return settings.ConfigPage{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, rows.Err()
}
func getConfig(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, tenantID, id uuid.UUID, lock bool) (settings.ConfigItem, error) {
	sql := configSelect + ` WHERE (c.tenant_id IS NULL OR c.tenant_id=$1) AND c.id=$2`
	if lock {
		sql += ` FOR UPDATE`
	}
	x, err := scanConfig(q.QueryRow(ctx, sql, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return x, settings.ErrNotFound
	}
	return x, err
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return []byte(raw)
}
func (r *Postgres) CreateConfig(ctx context.Context, p settings.Principal, in settings.ConfigInput, sealed []byte, keyVersion int32) (settings.ConfigItem, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return settings.ConfigItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	var key any
	if in.IsSecret {
		key = keyVersion
	}
	err = tx.QueryRow(ctx, `INSERT INTO sys.config_items(tenant_id,module_code,config_group,config_key,display_name,value_type,value_json,default_value_json,is_secret,secret_ciphertext,secret_key_version,is_public,validation_schema,description,sort_order,status,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,nullif($14,''),$15,$16,$17,$17) RETURNING id`, p.TenantID, in.ModuleCode, in.ConfigGroup, in.ConfigKey, in.DisplayName, in.ValueType, nullableJSON(in.Value), nullableJSON(in.DefaultValue), in.IsSecret, nullableBytes(sealed), key, in.IsPublic, []byte(in.ValidationSchema), in.Description, in.SortOrder, in.Status, p.UserID).Scan(&id)
	if err != nil {
		return settings.ConfigItem{}, classify(err)
	}
	after, err := getConfig(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return settings.ConfigItem{}, err
	}
	if err = audit(ctx, tx, p, "sys.config.create", "sys.config", id, "POST", "/admin-api/v1/configs", nil, after); err != nil {
		return settings.ConfigItem{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return settings.ConfigItem{}, err
	}
	return after, nil
}
func nullableBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}
func (r *Postgres) UpdateConfig(ctx context.Context, p settings.Principal, id uuid.UUID, in settings.ConfigInput) (settings.ConfigItem, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return settings.ConfigItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfig(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return settings.ConfigItem{}, err
	}
	isGlobal := before.TenantID == nil
	if (isGlobal && !p.CanUpdateGlobalConfigs) || (!isGlobal && before.IsLocked) || before.IsSecret != in.IsSecret {
		return settings.ConfigItem{}, settings.ErrLocked
	}
	if catalogConfig(before) && (before.ModuleCode != in.ModuleCode || before.ConfigGroup != in.ConfigGroup || before.ConfigKey != in.ConfigKey || before.DisplayName != in.DisplayName || before.ValueType != in.ValueType || !sameJSON(before.DefaultValue, in.DefaultValue) || before.IsPublic != in.IsPublic || before.Description != in.Description || before.SortOrder != in.SortOrder || before.Status != in.Status || !sameJSON(before.ValidationSchema, in.ValidationSchema)) {
		return settings.ConfigItem{}, settings.ErrLocked
	}
	var targetTenant any = p.TenantID
	if isGlobal {
		targetTenant = nil
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.config_items SET module_code=$1,config_group=$2,config_key=$3,display_name=$4,value_type=$5,value_json=$6,default_value_json=$7,is_public=$8,validation_schema=$9,description=nullif($10,''),sort_order=$11,status=$12,version=version+1,updated_by=$13 WHERE tenant_id IS NOT DISTINCT FROM $14 AND id=$15 AND version=$16`, in.ModuleCode, in.ConfigGroup, in.ConfigKey, in.DisplayName, in.ValueType, nullableJSON(in.Value), nullableJSON(in.DefaultValue), in.IsPublic, []byte(in.ValidationSchema), in.Description, in.SortOrder, in.Status, p.UserID, targetTenant, id, in.Version)
	if err != nil {
		return settings.ConfigItem{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return settings.ConfigItem{}, settings.ErrConflict
	}
	after, err := getConfig(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return settings.ConfigItem{}, err
	}
	action := "sys.config.update"
	if isGlobal {
		action = "sys.platform_config.update"
	}
	if err = audit(ctx, tx, p, action, "sys.config", id, "PATCH", "/admin-api/v1/configs/"+id.String(), before, after); err != nil {
		return settings.ConfigItem{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return settings.ConfigItem{}, err
	}
	return after, nil
}

func catalogConfig(item settings.ConfigItem) bool {
	var schema map[string]any
	if json.Unmarshal(item.ValidationSchema, &schema) != nil {
		return false
	}
	value, _ := schema["x-appkernia-catalog"].(bool)
	return value
}

func sameJSON(left, right json.RawMessage) bool {
	var a, b any
	return json.Unmarshal(left, &a) == nil && json.Unmarshal(right, &b) == nil && reflect.DeepEqual(a, b)
}
func (r *Postgres) RotateSecret(ctx context.Context, p settings.Principal, id uuid.UUID, version int32, sealed []byte, keyVersion int32) (settings.ConfigItem, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return settings.ConfigItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfig(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return settings.ConfigItem{}, err
	}
	if !before.IsSecret || before.IsLocked {
		return settings.ConfigItem{}, settings.ErrLocked
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.config_items SET secret_ciphertext=$1,secret_key_version=$2,version=version+1,updated_by=$3 WHERE tenant_id=$4 AND id=$5 AND version=$6`, sealed, keyVersion, p.UserID, p.TenantID, id, version)
	if err != nil {
		return settings.ConfigItem{}, err
	}
	if tag.RowsAffected() != 1 {
		return settings.ConfigItem{}, settings.ErrConflict
	}
	after, err := getConfig(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return settings.ConfigItem{}, err
	}
	before.SecretConfigured, after.SecretConfigured = false, true
	before.SecretKeyVersion = nil
	if err = audit(ctx, tx, p, "sys.config.rotate_secret", "sys.config", id, "POST", "/admin-api/v1/configs/"+id.String()+"/rotate-secret", before, after); err != nil {
		return settings.ConfigItem{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return settings.ConfigItem{}, err
	}
	return after, nil
}

const typeSelect = `SELECT d.id,d.tenant_id,d.code::text,d.name,coalesce(d.name_key,''),coalesce(d.description,''),coalesce(d.description_key,''),d.is_system,d.status,(d.tenant_id IS NULL OR d.is_system),d.visibility,d.extension_policy,d.created_at,d.updated_at FROM sys.dict_types d`

func scanType(row scanner) (settings.DictType, error) {
	var x settings.DictType
	err := row.Scan(&x.ID, &x.TenantID, &x.Code, &x.Name, &x.NameKey, &x.Description, &x.DescriptionKey, &x.IsSystem, &x.Status, &x.IsLocked, &x.Visibility, &x.ExtensionPolicy, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}
func (r *Postgres) ListDictTypes(ctx context.Context, tenantID uuid.UUID, f settings.PageFilter) (settings.DictTypePage, error) {
	where := ` WHERE (d.tenant_id IS NULL OR d.tenant_id=$1)`
	args := []any{tenantID}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where += fmt.Sprintf(` AND (d.code::text ILIKE $%d OR d.name ILIKE $%d)`, n, n)
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(` AND d.status=$%d`, len(args))
	}
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM sys.dict_types d`+where, args...).Scan(&total); err != nil {
		return settings.DictTypePage{}, err
	}
	order := `d.name,d.id`
	switch f.Sort {
	case "updated_desc":
		order = `d.updated_at DESC,d.id`
	case "key":
		order = `d.code,d.id`
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, typeSelect+where+fmt.Sprintf(` ORDER BY %s LIMIT $%d OFFSET $%d`, order, len(args)-1, len(args)), args...)
	if err != nil {
		return settings.DictTypePage{}, err
	}
	defer rows.Close()
	items := []settings.DictType{}
	for rows.Next() {
		x, e := scanType(rows)
		if e != nil {
			return settings.DictTypePage{}, e
		}
		items = append(items, x)
	}
	return settings.DictTypePage{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, rows.Err()
}
func getType(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, tenantID, id uuid.UUID, lock bool) (settings.DictType, error) {
	sql := typeSelect + ` WHERE (d.tenant_id IS NULL OR d.tenant_id=$1) AND d.id=$2`
	if lock {
		sql += ` FOR UPDATE`
	}
	x, err := scanType(q.QueryRow(ctx, sql, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return x, settings.ErrNotFound
	}
	return x, err
}
func (r *Postgres) CreateDictType(ctx context.Context, p settings.Principal, in settings.DictTypeInput) (settings.DictType, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return settings.DictType{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	if err = tx.QueryRow(ctx, `INSERT INTO sys.dict_types(tenant_id,code,name,description,is_system,status) VALUES($1,$2,$3,nullif($4,''),false,$5) RETURNING id`, p.TenantID, in.Code, in.Name, in.Description, in.Status).Scan(&id); err != nil {
		return settings.DictType{}, classify(err)
	}
	after, err := getType(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return settings.DictType{}, err
	}
	if err = audit(ctx, tx, p, "sys.dictionary.create", "sys.dict_type", id, "POST", "/admin-api/v1/dict-types", nil, after); err != nil {
		return settings.DictType{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return settings.DictType{}, err
	}
	return after, nil
}
func (r *Postgres) UpdateDictType(ctx context.Context, p settings.Principal, id uuid.UUID, in settings.DictTypeInput) (settings.DictType, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return settings.DictType{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getType(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return settings.DictType{}, err
	}
	if before.IsLocked {
		return settings.DictType{}, settings.ErrLocked
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.dict_types SET code=$1,name=$2,description=nullif($3,''),status=$4 WHERE tenant_id=$5 AND id=$6`, in.Code, in.Name, in.Description, in.Status, p.TenantID, id)
	if err != nil {
		return settings.DictType{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return settings.DictType{}, settings.ErrNotFound
	}
	after, err := getType(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return settings.DictType{}, err
	}
	if err = audit(ctx, tx, p, "sys.dictionary.update", "sys.dict_type", id, "PATCH", "/admin-api/v1/dict-types/"+id.String(), before, after); err != nil {
		return settings.DictType{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return settings.DictType{}, err
	}
	return after, nil
}

const itemSelect = `SELECT i.id,i.dict_type_id,i.tenant_id,i.item_value,i.label,i.locale,coalesce(i.color,''),coalesce(i.css_class,''),i.sort_order,i.is_default,i.extra,i.status,(i.tenant_id IS NULL),i.created_at,i.updated_at FROM sys.dict_items i JOIN sys.dict_types d ON d.id=i.dict_type_id`

func scanItem(row scanner) (settings.DictItem, error) {
	var x settings.DictItem
	err := row.Scan(&x.ID, &x.DictTypeID, &x.TenantID, &x.ItemValue, &x.Label, &x.Locale, &x.Color, &x.CSSClass, &x.SortOrder, &x.IsDefault, &x.Extra, &x.Status, &x.IsLocked, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}
func (r *Postgres) ListDictItems(ctx context.Context, tenantID, typeID uuid.UUID, f settings.DictItemFilter) (settings.DictItemPage, error) {
	typ, err := getType(ctx, r.pool, tenantID, typeID, false)
	if err != nil {
		return settings.DictItemPage{}, err
	}
	where := ` WHERE i.dict_type_id=$1 AND (i.tenant_id IS NULL OR i.tenant_id=$2)`
	args := []any{typeID, tenantID}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where += fmt.Sprintf(` AND (i.item_value ILIKE $%d OR i.label ILIKE $%d)`, n, n)
	}
	if f.Locale != "" {
		if f.Locale == "neutral" {
			where += ` AND i.locale IS NULL`
		} else {
			args = append(args, f.Locale)
			where += fmt.Sprintf(` AND i.locale=$%d`, len(args))
		}
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(` AND i.status=$%d`, len(args))
	}
	var total int64
	if err = r.pool.QueryRow(ctx, `SELECT count(*) FROM sys.dict_items i`+where, args...).Scan(&total); err != nil {
		return settings.DictItemPage{}, err
	}
	order := `i.sort_order,i.id`
	switch f.Sort {
	case "label":
		order = `i.label,i.id`
	case "updated_desc":
		order = `i.updated_at DESC,i.id`
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, itemSelect+where+fmt.Sprintf(` ORDER BY %s LIMIT $%d OFFSET $%d`, order, len(args)-1, len(args)), args...)
	if err != nil {
		return settings.DictItemPage{}, err
	}
	defer rows.Close()
	items := []settings.DictItem{}
	for rows.Next() {
		x, e := scanItem(rows)
		if e != nil {
			return settings.DictItemPage{}, e
		}
		items = append(items, x)
	}
	return settings.DictItemPage{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize, Type: typ}, rows.Err()
}
func getItem(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, tenantID, id uuid.UUID, lock bool) (settings.DictItem, error) {
	sql := itemSelect + ` WHERE (d.tenant_id IS NULL OR d.tenant_id=$1) AND (i.tenant_id IS NULL OR i.tenant_id=$1) AND i.id=$2`
	if lock {
		sql += ` FOR UPDATE OF i,d`
	}
	x, err := scanItem(q.QueryRow(ctx, sql, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return x, settings.ErrNotFound
	}
	return x, err
}
func ensureWritableType(ctx context.Context, tx pgx.Tx, tenantID, typeID uuid.UUID) (settings.DictType, error) {
	typ, err := getType(ctx, tx, tenantID, typeID, true)
	if err != nil {
		return typ, err
	}
	if typ.TenantID == nil && typ.ExtensionPolicy == "fixed" {
		return typ, settings.ErrLocked
	}
	return typ, nil
}

func validateDictionaryExtension(ctx context.Context, tx pgx.Tx, typ settings.DictType, tenantID uuid.UUID, in settings.DictItemInput) error {
	if typ.TenantID != nil || typ.ExtensionPolicy == "open" {
		return nil
	}
	var registeredExtra json.RawMessage
	registeredErr := tx.QueryRow(ctx, `SELECT extra FROM sys.dict_items WHERE dict_type_id=$1 AND tenant_id IS NULL AND item_value=$2 ORDER BY locale NULLS FIRST LIMIT 1`, typ.ID, in.ItemValue).Scan(&registeredExtra)
	if registeredErr != nil && !errors.Is(registeredErr, pgx.ErrNoRows) {
		return registeredErr
	}
	registered := registeredErr == nil
	switch typ.ExtensionPolicy {
	case "registered":
		if !registered || !sameJSON(registeredExtra, in.Extra) {
			return settings.ErrInvalid
		}
	case "s3_compatible":
		if registered {
			if !sameJSON(registeredExtra, in.Extra) {
				return settings.ErrInvalid
			}
			return nil
		}
		var extra struct {
			Adapter string `json:"adapter"`
		}
		if json.Unmarshal(in.Extra, &extra) != nil || extra.Adapter != "s3_compatible" {
			return settings.ErrInvalid
		}
	default:
		return settings.ErrLocked
	}
	return nil
}
func (r *Postgres) CreateDictItem(ctx context.Context, p settings.Principal, typeID uuid.UUID, in settings.DictItemInput) (settings.DictItem, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return settings.DictItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	typ, err := ensureWritableType(ctx, tx, p.TenantID, typeID)
	if err != nil {
		return settings.DictItem{}, err
	}
	if err = validateDictionaryExtension(ctx, tx, typ, p.TenantID, in); err != nil {
		return settings.DictItem{}, err
	}
	var id uuid.UUID
	if err = tx.QueryRow(ctx, `INSERT INTO sys.dict_items(dict_type_id,tenant_id,item_value,label,locale,color,css_class,sort_order,is_default,extra,status) VALUES($1,$2,$3,$4,$5,nullif($6,''),nullif($7,''),$8,$9,$10,$11) RETURNING id`, typeID, p.TenantID, in.ItemValue, in.Label, in.Locale, in.Color, in.CSSClass, in.SortOrder, in.IsDefault, []byte(in.Extra), in.Status).Scan(&id); err != nil {
		return settings.DictItem{}, classify(err)
	}
	after, err := getItem(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return settings.DictItem{}, err
	}
	if err = audit(ctx, tx, p, "sys.dictionary.create", "sys.dict_item", id, "POST", "/admin-api/v1/dict-types/"+typeID.String()+"/items", nil, after); err != nil {
		return settings.DictItem{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return settings.DictItem{}, err
	}
	return after, nil
}
func (r *Postgres) UpdateDictItem(ctx context.Context, p settings.Principal, id uuid.UUID, in settings.DictItemInput) (settings.DictItem, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return settings.DictItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getItem(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return settings.DictItem{}, err
	}
	if before.IsLocked {
		return settings.DictItem{}, settings.ErrLocked
	}
	typ, err := getType(ctx, tx, p.TenantID, before.DictTypeID, false)
	if err != nil || validateDictionaryExtension(ctx, tx, typ, p.TenantID, in) != nil {
		return settings.DictItem{}, settings.ErrInvalid
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.dict_items SET item_value=$1,label=$2,locale=$3,color=nullif($4,''),css_class=nullif($5,''),sort_order=$6,is_default=$7,extra=$8,status=$9 WHERE id=$10`, in.ItemValue, in.Label, in.Locale, in.Color, in.CSSClass, in.SortOrder, in.IsDefault, []byte(in.Extra), in.Status, id)
	if err != nil {
		return settings.DictItem{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return settings.DictItem{}, settings.ErrNotFound
	}
	after, err := getItem(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return settings.DictItem{}, err
	}
	if err = audit(ctx, tx, p, "sys.dictionary.update", "sys.dict_item", id, "PATCH", "/admin-api/v1/dict-items/"+id.String(), before, after); err != nil {
		return settings.DictItem{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return settings.DictItem{}, err
	}
	return after, nil
}

func (r *Postgres) ResolveDictionary(ctx context.Context, tenantID *uuid.UUID, code, locale string, publicOnly bool) (settings.ResolvedDictionary, error) {
	args := []any{code}
	where := `d.code=$1 AND d.status='active'`
	if tenantID == nil {
		where += ` AND d.tenant_id IS NULL`
	} else {
		args = append(args, *tenantID)
		where += ` AND (d.tenant_id IS NULL OR d.tenant_id=$2)`
	}
	if publicOnly {
		where += ` AND d.visibility='public'`
	}
	query := `SELECT d.id,d.code::text,d.extension_policy FROM sys.dict_types d WHERE ` + where + ` ORDER BY d.tenant_id NULLS LAST LIMIT 1`
	var typeID uuid.UUID
	var out settings.ResolvedDictionary
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&typeID, &out.Code, &out.ExtensionPolicy); errors.Is(err, pgx.ErrNoRows) {
		return out, settings.ErrNotFound
	} else if err != nil {
		return out, err
	}
	out.Locale = locale
	itemArgs := []any{typeID, locale}
	tenantFilter := `i.tenant_id IS NULL`
	tenantRank := `0`
	if tenantID != nil {
		itemArgs = append(itemArgs, *tenantID)
		tenantFilter = `(i.tenant_id IS NULL OR i.tenant_id=$3)`
		tenantRank = `CASE WHEN i.tenant_id=$3 THEN 0 ELSE 1 END`
	}
	rows, err := r.pool.Query(ctx, `
WITH ranked AS (
    SELECT i.item_value,i.label,COALESCE(i.color,'') AS color,COALESCE(i.css_class,'') AS css_class,i.is_default,i.extra,i.sort_order,i.status,
           row_number() OVER (PARTITION BY i.item_value ORDER BY
             CASE WHEN i.locale=$2 THEN 0 WHEN i.locale IS NULL THEN 1 WHEN i.locale='zh-CN' THEN 2 ELSE 3 END,
             `+tenantRank+`,i.id) AS rank,
           min(CASE WHEN i.locale=$2 THEN 0 WHEN i.locale IS NULL THEN 1 WHEN i.locale='zh-CN' THEN 2 ELSE 3 END)
             OVER (PARTITION BY i.item_value) AS locale_rank
    FROM sys.dict_items i
    WHERE i.dict_type_id=$1 AND `+tenantFilter+`
)
SELECT item_value,label,color,css_class,is_default,extra FROM ranked
WHERE rank=1 AND locale_rank<3 AND status='active' ORDER BY sort_order,item_value`, itemArgs...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.Items = make([]settings.DictionaryOption, 0)
	for rows.Next() {
		var item settings.DictionaryOption
		if err = rows.Scan(&item.Value, &item.Label, &item.Color, &item.CSSClass, &item.IsDefault, &item.Extra); err != nil {
			return out, err
		}
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}
func (r *Postgres) DeleteDictItem(ctx context.Context, p settings.Principal, id uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getItem(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return err
	}
	if before.IsLocked {
		return settings.ErrLocked
	}
	tag, err := tx.Exec(ctx, `DELETE FROM sys.dict_items WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return settings.ErrNotFound
	}
	if err = audit(ctx, tx, p, "sys.dictionary.delete", "sys.dict_item", id, "DELETE", "/admin-api/v1/dict-items/"+id.String(), before, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func audit(ctx context.Context, tx pgx.Tx, p settings.Principal, action, resource string, id uuid.UUID, method, path string, before, after any) error {
	return auditResource(ctx, tx, p, action, resource, id.String(), method, path, before, after)
}

func auditRegion(ctx context.Context, tx pgx.Tx, p settings.Principal, action, code, method, path string, before, after any) error {
	return auditResource(ctx, tx, p, action, "sys.region", code, method, path, before, after)
}

func auditResource(ctx context.Context, tx pgx.Tx, p settings.Principal, action, resource, resourceID, method, path string, before, after any) error {
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	_, err := tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,before_data,after_data,succeeded) VALUES($1,$2,nullif($3,'00000000-0000-0000-0000-000000000000'::uuid),nullif($4,''),'sys',$5,$5,$6,$7,$8,$9,200,$10,nullif($11,''),$12,$13,true)`, p.TenantID, p.UserID, p.SessionID, p.RequestID, action, resource, resourceID, method, path, p.IPAddress, p.UserAgent, b, a)
	return err
}
func classify(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) && (p.Code == "23505" || p.Code == "23503" || p.Code == "23514") {
		return settings.ErrConflict
	}
	return fmt.Errorf("system settings write: %w", err)
}
