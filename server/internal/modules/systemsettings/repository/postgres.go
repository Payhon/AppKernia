package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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

func (r *Postgres) ListRegions(ctx context.Context, f settings.RegionFilter) ([]settings.Region, error) {
	where, args := " WHERE 1=1", []any{}
	add := func(fragment string, value any) {
		args = append(args, value)
		where += fmt.Sprintf(fragment, len(args))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where += fmt.Sprintf(" AND (code ILIKE $%d OR name ILIKE $%d OR coalesce(full_name,'') ILIKE $%d)", n, n, n)
	} else if f.ParentCode != "" {
		add(" AND parent_code=$%d", f.ParentCode)
	} else {
		where += " AND parent_code IS NULL"
	}
	if f.Level != nil {
		add(" AND level=$%d", *f.Level)
	}
	if f.Status != "" {
		add(" AND status=$%d", f.Status)
	}
	args = append(args, f.Limit)
	rows, err := r.pool.Query(ctx, `SELECT r.code,r.parent_code,r.level,r.name,coalesce(r.full_name,''),coalesce(r.postal_code,''),r.longitude::float8,r.latitude::float8,r.status,EXISTS(SELECT 1 FROM sys.regions c WHERE c.parent_code=r.code),r.updated_at FROM sys.regions r`+where+fmt.Sprintf(" ORDER BY r.code LIMIT $%d", len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []settings.Region{}
	for rows.Next() {
		var x settings.Region
		if err = rows.Scan(&x.Code, &x.ParentCode, &x.Level, &x.Name, &x.FullName, &x.PostalCode, &x.Longitude, &x.Latitude, &x.Status, &x.HasChildren, &x.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (r *Postgres) ListModules(ctx context.Context, f settings.ModuleFilter) ([]settings.Module, error) {
	where, args := " WHERE 1=1", []any{}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where += fmt.Sprintf(" AND (code::text ILIKE $%d OR name ILIKE $%d)", len(args), len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND status=$%d", len(args))
	}
	rows, err := r.pool.Query(ctx, `SELECT id,code::text,name,version,coalesce(description,''),capabilities,status,installed_at,updated_at FROM sys.modules`+where+` ORDER BY code`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []settings.Module{}
	for rows.Next() {
		var x settings.Module
		if err = rows.Scan(&x.ID, &x.Code, &x.Name, &x.Version, &x.Description, &x.Capabilities, &x.Status, &x.InstalledAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
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
	if before.IsLocked || before.IsSecret != in.IsSecret {
		return settings.ConfigItem{}, settings.ErrLocked
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.config_items SET module_code=$1,config_group=$2,config_key=$3,display_name=$4,value_type=$5,value_json=$6,default_value_json=$7,is_public=$8,validation_schema=$9,description=nullif($10,''),sort_order=$11,status=$12,version=version+1,updated_by=$13 WHERE tenant_id=$14 AND id=$15 AND version=$16`, in.ModuleCode, in.ConfigGroup, in.ConfigKey, in.DisplayName, in.ValueType, nullableJSON(in.Value), nullableJSON(in.DefaultValue), in.IsPublic, []byte(in.ValidationSchema), in.Description, in.SortOrder, in.Status, p.UserID, p.TenantID, id, in.Version)
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
	if err = audit(ctx, tx, p, "sys.config.update", "sys.config", id, "PATCH", "/admin-api/v1/configs/"+id.String(), before, after); err != nil {
		return settings.ConfigItem{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return settings.ConfigItem{}, err
	}
	return after, nil
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

const typeSelect = `SELECT d.id,d.tenant_id,d.code::text,d.name,coalesce(d.description,''),d.is_system,d.status,(d.tenant_id IS NULL OR d.is_system),d.created_at,d.updated_at FROM sys.dict_types d`

func scanType(row scanner) (settings.DictType, error) {
	var x settings.DictType
	err := row.Scan(&x.ID, &x.TenantID, &x.Code, &x.Name, &x.Description, &x.IsSystem, &x.Status, &x.IsLocked, &x.CreatedAt, &x.UpdatedAt)
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

const itemSelect = `SELECT i.id,i.dict_type_id,i.item_value,i.label,i.locale,coalesce(i.color,''),coalesce(i.css_class,''),i.sort_order,i.is_default,i.extra,i.status,(d.tenant_id IS NULL OR d.is_system),i.created_at,i.updated_at FROM sys.dict_items i JOIN sys.dict_types d ON d.id=i.dict_type_id`

func scanItem(row scanner) (settings.DictItem, error) {
	var x settings.DictItem
	err := row.Scan(&x.ID, &x.DictTypeID, &x.ItemValue, &x.Label, &x.Locale, &x.Color, &x.CSSClass, &x.SortOrder, &x.IsDefault, &x.Extra, &x.Status, &x.IsLocked, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}
func (r *Postgres) ListDictItems(ctx context.Context, tenantID, typeID uuid.UUID, f settings.DictItemFilter) (settings.DictItemPage, error) {
	typ, err := getType(ctx, r.pool, tenantID, typeID, false)
	if err != nil {
		return settings.DictItemPage{}, err
	}
	where := ` WHERE i.dict_type_id=$1`
	args := []any{typeID}
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
	sql := itemSelect + ` WHERE (d.tenant_id IS NULL OR d.tenant_id=$1) AND i.id=$2`
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
	if typ.IsLocked {
		return typ, settings.ErrLocked
	}
	return typ, nil
}
func (r *Postgres) CreateDictItem(ctx context.Context, p settings.Principal, typeID uuid.UUID, in settings.DictItemInput) (settings.DictItem, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return settings.DictItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = ensureWritableType(ctx, tx, p.TenantID, typeID); err != nil {
		return settings.DictItem{}, err
	}
	var id uuid.UUID
	if err = tx.QueryRow(ctx, `INSERT INTO sys.dict_items(dict_type_id,item_value,label,locale,color,css_class,sort_order,is_default,extra,status) VALUES($1,$2,$3,$4,nullif($5,''),nullif($6,''),$7,$8,$9,$10) RETURNING id`, typeID, in.ItemValue, in.Label, in.Locale, in.Color, in.CSSClass, in.SortOrder, in.IsDefault, []byte(in.Extra), in.Status).Scan(&id); err != nil {
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
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	_, err := tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,before_data,after_data,succeeded) VALUES($1,$2,nullif($3,'00000000-0000-0000-0000-000000000000'::uuid),nullif($4,''),'sys',$5,$5,$6,$7,$8,$9,200,$10,nullif($11,''),$12,$13,true)`, p.TenantID, p.UserID, p.SessionID, p.RequestID, action, resource, id.String(), method, path, p.IPAddress, p.UserAgent, b, a)
	return err
}
func classify(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) && (p.Code == "23505" || p.Code == "23503" || p.Code == "23514") {
		return settings.ErrConflict
	}
	return fmt.Errorf("system settings write: %w", err)
}
