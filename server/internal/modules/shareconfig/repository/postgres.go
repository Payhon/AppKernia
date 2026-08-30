package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	share "github.com/appkernia/appkernia/server/internal/modules/shareconfig/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

type scanner interface{ Scan(...any) error }

const configSelect = `SELECT c.id,c.tenant_id,c.name,coalesce(c.description,''),c.provider_code,c.external_app_id,
c.config_schema_version,c.public_config,c.secret_field_names,(c.secret_ciphertext IS NOT NULL),c.status,
(SELECT count(*) FROM app.application_share_bindings b WHERE b.tenant_id=c.tenant_id AND b.share_config_id=c.id),
c.lock_version,c.created_at,c.updated_at FROM sys.share_configs c`

func scanConfig(row scanner) (share.Config, error) {
	var out share.Config
	err := row.Scan(&out.ID, &out.TenantID, &out.Name, &out.Description, &out.ProviderCode, &out.ExternalAppID,
		&out.ConfigSchemaVersion, &out.PublicConfig, &out.SecretFieldNames, &out.HasSecret, &out.Status,
		&out.BindingCount, &out.LockVersion, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *Postgres) List(ctx context.Context, tenantID uuid.UUID, f share.ListFilter) (share.ConfigPage, error) {
	where, args := ` WHERE c.tenant_id=$1 AND c.deleted_at IS NULL`, []any{tenantID}
	add := func(fragment string, value any) {
		args = append(args, value)
		where += fmt.Sprintf(fragment, len(args))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where += fmt.Sprintf(` AND (c.name ILIKE $%d OR c.description ILIKE $%d OR c.external_app_id ILIKE $%d)`, n, n, n)
	}
	if f.ProviderCode != "" {
		add(` AND c.provider_code=$%d`, f.ProviderCode)
	}
	if f.Status != "" {
		add(` AND c.status=$%d`, f.Status)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM sys.share_configs c`+where, args...).Scan(&total); err != nil {
		return share.ConfigPage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, configSelect+where+fmt.Sprintf(` ORDER BY c.updated_at DESC,c.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return share.ConfigPage{}, err
	}
	defer rows.Close()
	items := []share.Config{}
	for rows.Next() {
		item, scanErr := scanConfig(rows)
		if scanErr != nil {
			return share.ConfigPage{}, scanErr
		}
		items = append(items, item)
	}
	return share.ConfigPage{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, rows.Err()
}

func (r *Postgres) Get(ctx context.Context, tenantID, id uuid.UUID) (share.Config, error) {
	out, err := scanConfig(r.pool.QueryRow(ctx, configSelect+` WHERE c.tenant_id=$1 AND c.id=$2 AND c.deleted_at IS NULL`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, share.ErrNotFound
	}
	return out, err
}

func getConfigTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, lock bool) (share.Config, error) {
	sql := configSelect + ` WHERE c.tenant_id=$1 AND c.id=$2 AND c.deleted_at IS NULL`
	if lock {
		sql += ` FOR UPDATE OF c`
	}
	out, err := scanConfig(tx.QueryRow(ctx, sql, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, share.ErrNotFound
	}
	return out, err
}

func (r *Postgres) Create(ctx context.Context, p share.Principal, in share.ConfigInput) (share.Config, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return share.Config{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO sys.share_configs(tenant_id,name,description,provider_code,external_app_id,config_schema_version,public_config,status,created_by,updated_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,'draft',$8,$8) RETURNING id`, p.TenantID, in.Name, in.Description, in.ProviderCode, in.ExternalAppID, in.ConfigSchemaVersion, in.PublicConfig, p.UserID).Scan(&id)
	if err != nil {
		return share.Config{}, classify(err)
	}
	after, err := getConfigTx(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return share.Config{}, err
	}
	if err = audit(ctx, tx, p, "sys.share_config.create", "sys.share_config.create", "share_config", id.String(), "POST", "/admin-api/v1/share-configs", nil, auditConfig(after)); err != nil {
		return share.Config{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return share.Config{}, err
	}
	return after, nil
}

func (r *Postgres) Update(ctx context.Context, p share.Principal, id uuid.UUID, in share.ConfigInput) (share.Config, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return share.Config{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfigTx(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return share.Config{}, err
	}
	if before.LockVersion != in.LockVersion {
		return share.Config{}, share.ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.share_configs SET name=$1,description=$2,external_app_id=$3,config_schema_version=$4,public_config=$5,lock_version=lock_version+1,updated_by=$6 WHERE tenant_id=$7 AND id=$8 AND deleted_at IS NULL AND lock_version=$9`, in.Name, in.Description, in.ExternalAppID, in.ConfigSchemaVersion, in.PublicConfig, p.UserID, p.TenantID, id, in.LockVersion)
	if err != nil {
		return share.Config{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return share.Config{}, share.ErrConflict
	}
	after, err := getConfigTx(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return share.Config{}, err
	}
	if err = audit(ctx, tx, p, "sys.share_config.update", "sys.share_config.update", "share_config", id.String(), "PATCH", "/admin-api/v1/share-configs/"+id.String(), auditConfig(before), auditConfig(after)); err != nil {
		return share.Config{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return share.Config{}, err
	}
	return after, nil
}

func (r *Postgres) SetStatus(ctx context.Context, p share.Principal, id uuid.UUID, lockVersion int32, status string) (share.Config, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return share.Config{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfigTx(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return share.Config{}, err
	}
	if before.LockVersion != lockVersion {
		return share.Config{}, share.ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.share_configs SET status=$1,lock_version=lock_version+1,updated_by=$2 WHERE tenant_id=$3 AND id=$4 AND deleted_at IS NULL AND lock_version=$5`, status, p.UserID, p.TenantID, id, lockVersion)
	if err != nil {
		return share.Config{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return share.Config{}, share.ErrConflict
	}
	after, err := getConfigTx(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return share.Config{}, err
	}
	action := "sys.share_config.activate"
	pathAction := "activate"
	if status == "disabled" {
		action = "sys.share_config.disable"
		pathAction = "disable"
	}
	if err = audit(ctx, tx, p, action, "sys.share_config.update", "share_config", id.String(), "POST", "/admin-api/v1/share-configs/"+id.String()+"/"+pathAction, auditConfig(before), auditConfig(after)); err != nil {
		return share.Config{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return share.Config{}, err
	}
	return after, nil
}

func (r *Postgres) Delete(ctx context.Context, p share.Principal, id uuid.UUID, lockVersion int32) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfigTx(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return err
	}
	if before.LockVersion != lockVersion {
		return share.ErrConflict
	}
	if before.BindingCount > 0 {
		return share.ErrInUse
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.share_configs SET deleted_at=now(),lock_version=lock_version+1,updated_by=$1 WHERE tenant_id=$2 AND id=$3 AND lock_version=$4 AND deleted_at IS NULL`, p.UserID, p.TenantID, id, lockVersion)
	if err != nil {
		return classify(err)
	}
	if tag.RowsAffected() != 1 {
		return share.ErrConflict
	}
	if err = audit(ctx, tx, p, "sys.share_config.delete", "sys.share_config.delete", "share_config", id.String(), "DELETE", "/admin-api/v1/share-configs/"+id.String(), auditConfig(before), map[string]any{"deleted": true}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Postgres) RotateSecret(ctx context.Context, p share.Principal, id uuid.UUID, lockVersion int32, ciphertext []byte, keyVersion int32, names []string) (share.Config, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return share.Config{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfigTx(ctx, tx, p.TenantID, id, true)
	if err != nil {
		return share.Config{}, err
	}
	if before.LockVersion != lockVersion {
		return share.Config{}, share.ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.share_configs SET secret_ciphertext=$1,secret_key_version=$2,secret_field_names=$3,lock_version=lock_version+1,updated_by=$4 WHERE tenant_id=$5 AND id=$6 AND lock_version=$7 AND deleted_at IS NULL`, ciphertext, keyVersion, names, p.UserID, p.TenantID, id, lockVersion)
	if err != nil {
		return share.Config{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return share.Config{}, share.ErrConflict
	}
	after, err := getConfigTx(ctx, tx, p.TenantID, id, false)
	if err != nil {
		return share.Config{}, err
	}
	if err = audit(ctx, tx, p, "sys.share_config.rotate_secret", "sys.share_config.rotate_secret", "share_config", id.String(), "POST", "/admin-api/v1/share-configs/"+id.String()+"/rotate-secret", map[string]any{"configured_fields": before.SecretFieldNames}, map[string]any{"configured_fields": after.SecretFieldNames}); err != nil {
		return share.Config{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return share.Config{}, err
	}
	return after, nil
}

const bindingSelect = `SELECT b.id,b.app_id,b.provider_code,b.share_config_id,c.name,c.status,b.enabled,b.scenes,b.share_origin,b.fallback_mode,b.lock_version,b.updated_at FROM app.application_share_bindings b JOIN sys.share_configs c ON c.tenant_id=b.tenant_id AND c.id=b.share_config_id`

func (r *Postgres) AppExists(ctx context.Context, tenantID, appID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL)`, tenantID, appID).Scan(&exists)
	return exists, err
}

func scanBinding(row scanner) (share.Binding, error) {
	var out share.Binding
	err := row.Scan(&out.ID, &out.AppID, &out.ProviderCode, &out.ShareConfigID, &out.ShareConfigName, &out.ConfigStatus, &out.Enabled, &out.Scenes, &out.ShareOrigin, &out.FallbackMode, &out.LockVersion, &out.UpdatedAt)
	return out, err
}

func (r *Postgres) ListBindings(ctx context.Context, tenantID, appID uuid.UUID) ([]share.Binding, error) {
	rows, err := r.pool.Query(ctx, bindingSelect+` WHERE b.tenant_id=$1 AND b.app_id=$2 ORDER BY b.provider_code`, tenantID, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []share.Binding{}
	for rows.Next() {
		item, scanErr := scanBinding(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func getBindingTx(ctx context.Context, tx pgx.Tx, tenantID, appID uuid.UUID, provider string, lock bool) (share.Binding, error) {
	sql := bindingSelect + ` WHERE b.tenant_id=$1 AND b.app_id=$2 AND b.provider_code=$3`
	if lock {
		sql += ` FOR UPDATE OF b`
	}
	out, err := scanBinding(tx.QueryRow(ctx, sql, tenantID, appID, provider))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, share.ErrNotFound
	}
	return out, err
}

func (r *Postgres) UpsertBinding(ctx context.Context, p share.Principal, appID uuid.UUID, provider string, in share.BindingInput) (share.Binding, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return share.Binding{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var appExists, configValid bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL),EXISTS(SELECT 1 FROM sys.share_configs WHERE tenant_id=$1 AND id=$3 AND provider_code=$4 AND status='active' AND deleted_at IS NULL)`, p.TenantID, appID, in.ShareConfigID, provider).Scan(&appExists, &configValid); err != nil {
		return share.Binding{}, err
	}
	if !appExists {
		return share.Binding{}, share.ErrNotFound
	}
	if !configValid {
		return share.Binding{}, share.ErrInvalid
	}
	before, getErr := getBindingTx(ctx, tx, p.TenantID, appID, provider, true)
	if getErr != nil && !errors.Is(getErr, share.ErrNotFound) {
		return share.Binding{}, getErr
	}
	if errors.Is(getErr, share.ErrNotFound) {
		if in.LockVersion != 0 {
			return share.Binding{}, share.ErrConflict
		}
		_, err = tx.Exec(ctx, `INSERT INTO app.application_share_bindings(tenant_id,app_id,provider_code,share_config_id,enabled,scenes,share_origin,fallback_mode,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, p.TenantID, appID, provider, in.ShareConfigID, in.Enabled, in.Scenes, in.ShareOrigin, in.FallbackMode, p.UserID)
	} else {
		if before.LockVersion != in.LockVersion {
			return share.Binding{}, share.ErrConflict
		}
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE app.application_share_bindings SET share_config_id=$1,enabled=$2,scenes=$3,share_origin=$4,fallback_mode=$5,lock_version=lock_version+1,updated_by=$6 WHERE tenant_id=$7 AND app_id=$8 AND provider_code=$9 AND lock_version=$10`, in.ShareConfigID, in.Enabled, in.Scenes, in.ShareOrigin, in.FallbackMode, p.UserID, p.TenantID, appID, provider, in.LockVersion)
		if err == nil && tag.RowsAffected() != 1 {
			err = share.ErrConflict
		}
	}
	if err != nil {
		return share.Binding{}, classify(err)
	}
	after, err := getBindingTx(ctx, tx, p.TenantID, appID, provider, false)
	if err != nil {
		return share.Binding{}, err
	}
	var beforeAudit any
	if getErr == nil {
		beforeAudit = auditBinding(before)
	}
	if err = audit(ctx, tx, p, "app.share_binding.update", "app.share_binding.update", "application_share_binding", after.ID.String(), "PUT", "/admin-api/v1/apps/"+appID.String()+"/share-bindings/"+provider, beforeAudit, auditBinding(after)); err != nil {
		return share.Binding{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return share.Binding{}, err
	}
	return after, nil
}

func (r *Postgres) DeleteBinding(ctx context.Context, p share.Principal, appID uuid.UUID, provider string, lockVersion int32) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getBindingTx(ctx, tx, p.TenantID, appID, provider, true)
	if err != nil {
		return err
	}
	if before.LockVersion != lockVersion {
		return share.ErrConflict
	}
	if _, err = tx.Exec(ctx, `DELETE FROM app.application_share_bindings WHERE tenant_id=$1 AND app_id=$2 AND provider_code=$3 AND lock_version=$4`, p.TenantID, appID, provider, lockVersion); err != nil {
		return classify(err)
	}
	if err = audit(ctx, tx, p, "app.share_binding.delete", "app.share_binding.update", "application_share_binding", before.ID.String(), "DELETE", "/admin-api/v1/apps/"+appID.String()+"/share-bindings/"+provider, auditBinding(before), map[string]any{"deleted": true}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Postgres) Runtime(ctx context.Context, appID uuid.UUID) ([]share.RuntimeProvider, error) {
	rows, err := r.pool.Query(ctx, `SELECT b.provider_code,b.enabled,b.scenes,b.fallback_mode FROM app.application_share_bindings b JOIN app.applications a ON a.tenant_id=b.tenant_id AND a.id=b.app_id JOIN sys.share_configs c ON c.tenant_id=b.tenant_id AND c.id=b.share_config_id WHERE b.app_id=$1 AND a.status='active' AND a.deleted_at IS NULL AND c.status='active' AND c.deleted_at IS NULL ORDER BY b.provider_code`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []share.RuntimeProvider{}
	for rows.Next() {
		var item share.RuntimeProvider
		if err = rows.Scan(&item.ProviderCode, &item.Enabled, &item.Scenes, &item.FallbackMode); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func auditConfig(item share.Config) map[string]any {
	return map[string]any{"id": item.ID, "name": item.Name, "provider_code": item.ProviderCode, "status": item.Status, "configured_secret_fields": item.SecretFieldNames, "lock_version": item.LockVersion}
}

func auditBinding(item share.Binding) map[string]any {
	return map[string]any{"id": item.ID, "app_id": item.AppID, "provider_code": item.ProviderCode, "share_config_id": item.ShareConfigID, "enabled": item.Enabled, "scenes": item.Scenes, "share_origin": item.ShareOrigin, "fallback_mode": item.FallbackMode, "lock_version": item.LockVersion}
}

func audit(ctx context.Context, tx pgx.Tx, p share.Principal, action, permission, resource, resourceID, method, path string, before, after any) error {
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	_, err := tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,before_data,after_data,succeeded) VALUES($1,$2,nullif($3,'00000000-0000-0000-0000-000000000000'::uuid),nullif($4,''),'share',$5,$6,$7,$8,$9,$10,200,$11,nullif($12,''),$13,$14,true)`, p.TenantID, p.UserID, p.SessionID, p.RequestID, action, permission, resource, resourceID, method, path, p.IPAddress, p.UserAgent, b, a)
	return err
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, share.ErrConflict) {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "40001":
			return share.ErrConflict
		case "23503", "23514", "22P02":
			return share.ErrInvalid
		}
	}
	return err
}
