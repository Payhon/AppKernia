package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OTPDispatcher interface {
	Queue(context.Context, pgx.Tx, login.OTPChallenge) error
}

type Postgres struct {
	pool *pgxpool.Pool
	otp  OTPDispatcher
}

func NewPostgres(pool *pgxpool.Pool, otp OTPDispatcher) *Postgres {
	return &Postgres{pool: pool, otp: otp}
}

type scanner interface{ Scan(...any) error }

const configSelect = `SELECT c.id,c.tenant_id,c.name,coalesce(c.description,''),c.provider_code,c.external_client_id,
c.config_schema_version,c.public_config,c.secret_field_names,(c.secret_ciphertext IS NOT NULL),
coalesce(c.credential_fingerprint,''),c.status,c.last_preflight_at,c.last_preflight_status,c.last_preflight_issues,
(SELECT count(*) FROM app.application_login_provider_bindings b WHERE b.tenant_id=c.tenant_id AND b.login_provider_config_id=c.id),
c.lock_version,c.created_at,c.updated_at
FROM sys.login_provider_configs c`

func scanConfig(row scanner) (login.Config, error) {
	var out login.Config
	var issues []byte
	err := row.Scan(
		&out.ID, &out.TenantID, &out.Name, &out.Description, &out.ProviderCode, &out.ExternalClientID,
		&out.ConfigSchemaVersion, &out.PublicConfig, &out.SecretFieldNames, &out.HasSecret,
		&out.CredentialFingerprint, &out.Status, &out.LastPreflightAt, &out.LastPreflightStatus, &issues,
		&out.BindingCount, &out.LockVersion, &out.CreatedAt, &out.UpdatedAt,
	)
	if err == nil && json.Unmarshal(issues, &out.LastPreflightIssues) != nil {
		err = login.ErrInvalid
	}
	if out.LastPreflightIssues == nil {
		out.LastPreflightIssues = []string{}
	}
	if out.SecretFieldNames == nil {
		out.SecretFieldNames = []string{}
	}
	return out, err
}

func (r *Postgres) ListConfigs(ctx context.Context, tenantID uuid.UUID, filter login.ListFilter) (login.ConfigPage, error) {
	where, args := ` WHERE c.tenant_id=$1 AND c.deleted_at IS NULL`, []any{tenantID}
	add := func(fragment string, value any) {
		args = append(args, value)
		where += fmt.Sprintf(fragment, len(args))
	}
	if filter.Query != "" {
		value := "%" + filter.Query + "%"
		args = append(args, value)
		position := len(args)
		where += fmt.Sprintf(` AND (c.name ILIKE $%d OR c.description ILIKE $%d OR c.external_client_id ILIKE $%d)`, position, position, position)
	}
	if filter.ProviderCode != "" {
		add(` AND c.provider_code=$%d`, filter.ProviderCode)
	}
	if filter.Status != "" {
		add(` AND c.status=$%d`, filter.Status)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM sys.login_provider_configs c`+where, args...).Scan(&total); err != nil {
		return login.ConfigPage{}, err
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := r.pool.Query(ctx, configSelect+where+fmt.Sprintf(` ORDER BY c.updated_at DESC,c.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return login.ConfigPage{}, err
	}
	defer rows.Close()
	items := []login.Config{}
	for rows.Next() {
		item, scanErr := scanConfig(rows)
		if scanErr != nil {
			return login.ConfigPage{}, scanErr
		}
		items = append(items, item)
	}
	return login.ConfigPage{Items: items, Page: filter.Page, PageSize: filter.PageSize, Total: total}, rows.Err()
}

func (r *Postgres) GetConfig(ctx context.Context, tenantID, id uuid.UUID) (login.Config, error) {
	out, err := scanConfig(r.pool.QueryRow(ctx, configSelect+` WHERE c.tenant_id=$1 AND c.id=$2 AND c.deleted_at IS NULL`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, login.ErrNotFound
	}
	return out, err
}

func (r *Postgres) GetConfigSecret(ctx context.Context, tenantID, id uuid.UUID) (login.Config, error) {
	out, err := r.GetConfig(ctx, tenantID, id)
	if err != nil {
		return out, err
	}
	err = r.pool.QueryRow(ctx, `SELECT secret_ciphertext,secret_key_version
FROM sys.login_provider_configs WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, id).Scan(&out.SecretCiphertext, &out.SecretKeyVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, login.ErrNotFound
	}
	return out, err
}

func getConfigTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, lock bool) (login.Config, error) {
	query := configSelect + ` WHERE c.tenant_id=$1 AND c.id=$2 AND c.deleted_at IS NULL`
	if lock {
		query += ` FOR UPDATE OF c`
	}
	out, err := scanConfig(tx.QueryRow(ctx, query, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, login.ErrNotFound
	}
	return out, err
}

func (r *Postgres) CreateConfig(ctx context.Context, principal login.Principal, input login.ConfigInput) (login.Config, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return login.Config{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO sys.login_provider_configs(
tenant_id,name,description,provider_code,external_client_id,config_schema_version,public_config,created_by,updated_by)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8) RETURNING id`, principal.TenantID, input.Name, input.Description,
		input.ProviderCode, input.ExternalClientID, input.ConfigSchemaVersion, input.PublicConfig, principal.UserID).Scan(&id)
	if err != nil {
		return login.Config{}, classify(err)
	}
	out, err := getConfigTx(ctx, tx, principal.TenantID, id, false)
	if err != nil {
		return login.Config{}, err
	}
	if err = audit(ctx, tx, principal, "sys.login_provider_config.create", "sys.login_provider_config.create", "login_provider_config", id.String(), "POST", "/admin-api/v1/login-provider-configs", nil, auditConfig(out)); err != nil {
		return login.Config{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return login.Config{}, err
	}
	return out, nil
}

func (r *Postgres) UpdateConfig(ctx context.Context, principal login.Principal, id uuid.UUID, input login.ConfigInput) (login.Config, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return login.Config{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfigTx(ctx, tx, principal.TenantID, id, true)
	if err != nil {
		return login.Config{}, err
	}
	if before.LockVersion != input.LockVersion || before.ProviderCode != input.ProviderCode {
		return login.Config{}, login.ErrConflict
	}
	if before.ExternalClientID != input.ExternalClientID {
		var inUse bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(
SELECT 1 FROM app.application_login_provider_bindings b
JOIN iam.app_oauth_accounts a ON a.app_id=b.app_id AND a.provider_code=b.provider_code
WHERE b.tenant_id=$1 AND b.login_provider_config_id=$2
  AND a.external_client_id=$3)`, principal.TenantID, id, before.ExternalClientID).Scan(&inUse); err != nil {
			return login.Config{}, err
		}
		if inUse {
			return login.Config{}, login.ErrInUse
		}
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.login_provider_configs
SET name=$1,description=$2,external_client_id=$3,config_schema_version=$4,public_config=$5,
    status='draft',last_preflight_at=NULL,last_preflight_status=NULL,last_preflight_issues='[]'::jsonb,
    lock_version=lock_version+1,updated_by=$6
WHERE tenant_id=$7 AND id=$8 AND deleted_at IS NULL AND lock_version=$9`, input.Name, input.Description,
		input.ExternalClientID, input.ConfigSchemaVersion, input.PublicConfig, principal.UserID, principal.TenantID, id, input.LockVersion)
	if err != nil {
		return login.Config{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return login.Config{}, login.ErrConflict
	}
	after, err := getConfigTx(ctx, tx, principal.TenantID, id, false)
	if err != nil {
		return login.Config{}, err
	}
	if err = audit(ctx, tx, principal, "sys.login_provider_config.update", "sys.login_provider_config.update", "login_provider_config", id.String(), "PATCH", "/admin-api/v1/login-provider-configs/"+id.String(), auditConfig(before), auditConfig(after)); err != nil {
		return login.Config{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return login.Config{}, err
	}
	return after, nil
}

func (r *Postgres) RotateSecret(ctx context.Context, principal login.Principal, id uuid.UUID, lockVersion int32, ciphertext []byte, keyVersion int32, names []string, fingerprint string) (login.Config, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return login.Config{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfigTx(ctx, tx, principal.TenantID, id, true)
	if err != nil {
		return login.Config{}, err
	}
	if before.LockVersion != lockVersion {
		return login.Config{}, login.ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.login_provider_configs
SET secret_ciphertext=$1,secret_key_version=$2,secret_field_names=$3,credential_fingerprint=$4,
    status='draft',last_preflight_at=NULL,last_preflight_status=NULL,last_preflight_issues='[]'::jsonb,
    lock_version=lock_version+1,updated_by=$5
WHERE tenant_id=$6 AND id=$7 AND deleted_at IS NULL AND lock_version=$8`, ciphertext, keyVersion, names, fingerprint,
		principal.UserID, principal.TenantID, id, lockVersion)
	if err != nil {
		return login.Config{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return login.Config{}, login.ErrConflict
	}
	after, err := getConfigTx(ctx, tx, principal.TenantID, id, false)
	if err != nil {
		return login.Config{}, err
	}
	if err = audit(ctx, tx, principal, "sys.login_provider_config.rotate_secret", "sys.login_provider_config.rotate_secret", "login_provider_config", id.String(), "POST", "/admin-api/v1/login-provider-configs/"+id.String()+"/rotate-secret", map[string]any{"secret_field_names": before.SecretFieldNames}, map[string]any{"secret_field_names": after.SecretFieldNames}); err != nil {
		return login.Config{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return login.Config{}, err
	}
	return after, nil
}

func (r *Postgres) SetPreflight(ctx context.Context, principal login.Principal, id uuid.UUID, lockVersion int32, ready bool, issues []string) (login.Config, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return login.Config{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfigTx(ctx, tx, principal.TenantID, id, true)
	if err != nil {
		return login.Config{}, err
	}
	if before.LockVersion != lockVersion {
		return login.Config{}, login.ErrConflict
	}
	status := "failed"
	if ready {
		status = "ready"
	}
	issuesJSON, _ := json.Marshal(issues)
	tag, err := tx.Exec(ctx, `UPDATE sys.login_provider_configs
SET last_preflight_at=now(),last_preflight_status=$1,last_preflight_issues=$2,
    status=CASE WHEN $1='failed' THEN 'draft' ELSE status END,
    lock_version=lock_version+1,updated_by=$3
WHERE tenant_id=$4 AND id=$5 AND deleted_at IS NULL AND lock_version=$6`, status, issuesJSON, principal.UserID,
		principal.TenantID, id, lockVersion)
	if err != nil {
		return login.Config{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return login.Config{}, login.ErrConflict
	}
	after, err := getConfigTx(ctx, tx, principal.TenantID, id, false)
	if err != nil {
		return login.Config{}, err
	}
	if err = audit(ctx, tx, principal, "sys.login_provider_config.preflight", "sys.login_provider_config.preflight", "login_provider_config", id.String(), "POST", "/admin-api/v1/login-provider-configs/"+id.String()+"/preflight", auditConfig(before), auditConfig(after)); err != nil {
		return login.Config{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return login.Config{}, err
	}
	return after, nil
}

func (r *Postgres) SetStatus(ctx context.Context, principal login.Principal, id uuid.UUID, lockVersion int32, status string) (login.Config, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return login.Config{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfigTx(ctx, tx, principal.TenantID, id, true)
	if err != nil {
		return login.Config{}, err
	}
	if before.LockVersion != lockVersion || status == "active" && (before.LastPreflightStatus == nil || *before.LastPreflightStatus != "ready") {
		return login.Config{}, login.ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.login_provider_configs SET status=$1,lock_version=lock_version+1,updated_by=$2
WHERE tenant_id=$3 AND id=$4 AND deleted_at IS NULL AND lock_version=$5`, status, principal.UserID, principal.TenantID, id, lockVersion)
	if err != nil {
		return login.Config{}, classify(err)
	}
	if tag.RowsAffected() != 1 {
		return login.Config{}, login.ErrConflict
	}
	after, err := getConfigTx(ctx, tx, principal.TenantID, id, false)
	if err != nil {
		return login.Config{}, err
	}
	action := "sys.login_provider_config.activate"
	if status == "disabled" {
		action = "sys.login_provider_config.disable"
	}
	if err = audit(ctx, tx, principal, action, "sys.login_provider_config.update", "login_provider_config", id.String(), "POST", "/admin-api/v1/login-provider-configs/"+id.String()+"/"+strings.TrimPrefix(action, "sys.login_provider_config."), auditConfig(before), auditConfig(after)); err != nil {
		return login.Config{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return login.Config{}, err
	}
	return after, nil
}

func (r *Postgres) DeleteConfig(ctx context.Context, principal login.Principal, id uuid.UUID, lockVersion int32) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfigTx(ctx, tx, principal.TenantID, id, true)
	if err != nil {
		return err
	}
	if before.LockVersion != lockVersion {
		return login.ErrConflict
	}
	if before.BindingCount > 0 {
		return login.ErrInUse
	}
	tag, err := tx.Exec(ctx, `UPDATE sys.login_provider_configs
SET deleted_at=now(),status='disabled',secret_ciphertext=NULL,secret_key_version=NULL,
    secret_field_names='{}'::text[],credential_fingerprint=NULL,
    lock_version=lock_version+1,updated_by=$1
WHERE tenant_id=$2 AND id=$3 AND lock_version=$4 AND deleted_at IS NULL`, principal.UserID, principal.TenantID, id, lockVersion)
	if err != nil {
		return classify(err)
	}
	if tag.RowsAffected() != 1 {
		return login.ErrConflict
	}
	if err = audit(ctx, tx, principal, "sys.login_provider_config.delete", "sys.login_provider_config.delete", "login_provider_config", id.String(), "DELETE", "/admin-api/v1/login-provider-configs/"+id.String(), auditConfig(before), map[string]bool{"deleted": true}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const bindingSelect = `SELECT b.id,b.app_id,b.provider_code,b.login_provider_config_id,c.name,c.status,c.last_preflight_status,
b.enabled,b.sort_order,b.lock_version,b.updated_at
FROM app.application_login_provider_bindings b
JOIN sys.login_provider_configs c ON c.tenant_id=b.tenant_id AND c.id=b.login_provider_config_id`

func scanBinding(row scanner) (login.Binding, error) {
	var out login.Binding
	var id, configID uuid.UUID
	var name, status string
	var updated time.Time
	err := row.Scan(&id, &out.AppID, &out.ProviderCode, &configID, &name, &status, &out.PreflightStatus, &out.Enabled, &out.SortOrder, &out.LockVersion, &updated)
	if err == nil {
		out.ID, out.LoginProviderConfigID, out.ConfigName, out.ConfigStatus, out.UpdatedAt = &id, &configID, &name, &status, &updated
	}
	return out, err
}

func listBindingsQuerier(ctx context.Context, queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, tenantID, appID uuid.UUID) ([]login.Binding, error) {
	rows, err := queryer.Query(ctx, bindingSelect+` WHERE b.tenant_id=$1 AND b.app_id=$2 ORDER BY b.sort_order,b.provider_code`, tenantID, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []login.Binding{}
	for rows.Next() {
		item, scanErr := scanBinding(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Postgres) ListBindings(ctx context.Context, tenantID, appID uuid.UUID) ([]login.Binding, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL)`, tenantID, appID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, login.ErrNotFound
	}
	return listBindingsQuerier(ctx, r.pool, tenantID, appID)
}

func (r *Postgres) ReplaceBindings(ctx context.Context, principal login.Principal, appID uuid.UUID, inputs []login.BindingInput) ([]login.Binding, error) {
	if len(inputs) != len(login.ProviderCodes) {
		return nil, login.ErrInvalid
	}
	registeredProviders := make(map[string]struct{}, len(login.ProviderCodes))
	for _, code := range login.ProviderCodes {
		registeredProviders[code] = struct{}{}
	}
	seenProviders := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if _, registered := registeredProviders[input.ProviderCode]; !registered {
			return nil, login.ErrInvalid
		}
		if _, duplicate := seenProviders[input.ProviderCode]; duplicate {
			return nil, login.ErrInvalid
		}
		seenProviders[input.ProviderCode] = struct{}{}
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL)`, principal.TenantID, appID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, login.ErrNotFound
	}
	before, err := listBindingsQuerier(ctx, tx, principal.TenantID, appID)
	if err != nil {
		return nil, err
	}
	current := make(map[string]login.Binding, len(before))
	for _, item := range before {
		current[item.ProviderCode] = item
	}
	for _, input := range inputs {
		stored, found := current[input.ProviderCode]
		if input.LoginProviderConfigID == nil {
			if input.Enabled || !found && input.LockVersion != 0 || found && stored.LockVersion != input.LockVersion {
				return nil, login.ErrConflict
			}
			if found {
				var identitiesExist bool
				if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.app_oauth_accounts
WHERE app_id=$1 AND provider_code=$2)`, appID, input.ProviderCode).Scan(&identitiesExist); err != nil {
					return nil, err
				}
				if identitiesExist {
					return nil, login.ErrInUse
				}
				if _, err = tx.Exec(ctx, `DELETE FROM app.application_login_provider_bindings
WHERE tenant_id=$1 AND app_id=$2 AND provider_code=$3 AND lock_version=$4`, principal.TenantID, appID, input.ProviderCode, input.LockVersion); err != nil {
					return nil, classify(err)
				}
			}
			continue
		}
		var externalClientID, configStatus string
		var preflightStatus *string
		if err = tx.QueryRow(ctx, `SELECT external_client_id,status,last_preflight_status FROM sys.login_provider_configs
WHERE tenant_id=$1 AND id=$2 AND provider_code=$3 AND deleted_at IS NULL
FOR SHARE`, principal.TenantID, *input.LoginProviderConfigID, input.ProviderCode).Scan(&externalClientID, &configStatus, &preflightStatus); errors.Is(err, pgx.ErrNoRows) {
			return nil, login.ErrInvalid
		}
		if err != nil {
			return nil, err
		}
		sameConfig := found && stored.LoginProviderConfigID != nil && *stored.LoginProviderConfigID == *input.LoginProviderConfigID
		if (input.Enabled || !sameConfig) && (configStatus != "active" || preflightStatus == nil || *preflightStatus != "ready") {
			return nil, login.ErrInvalid
		}
		var mismatchedIdentity bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.app_oauth_accounts
WHERE app_id=$1 AND provider_code=$2 AND external_client_id<>$3)`, appID, input.ProviderCode, externalClientID).Scan(&mismatchedIdentity); err != nil {
			return nil, err
		}
		if mismatchedIdentity {
			return nil, login.ErrInUse
		}
		if !found {
			if input.LockVersion != 0 {
				return nil, login.ErrConflict
			}
			_, err = tx.Exec(ctx, `INSERT INTO app.application_login_provider_bindings(
tenant_id,app_id,provider_code,login_provider_config_id,enabled,sort_order,created_by,updated_by)
VALUES($1,$2,$3,$4,$5,$6,$7,$7)`, principal.TenantID, appID, input.ProviderCode, *input.LoginProviderConfigID, input.Enabled, input.SortOrder, principal.UserID)
		} else {
			if stored.LockVersion != input.LockVersion {
				return nil, login.ErrConflict
			}
			var tag pgconn.CommandTag
			tag, err = tx.Exec(ctx, `UPDATE app.application_login_provider_bindings
SET login_provider_config_id=$1,enabled=$2,sort_order=$3,lock_version=lock_version+1,updated_by=$4
WHERE tenant_id=$5 AND app_id=$6 AND provider_code=$7 AND lock_version=$8`, *input.LoginProviderConfigID,
				input.Enabled, input.SortOrder, principal.UserID, principal.TenantID, appID, input.ProviderCode, input.LockVersion)
			if err == nil && tag.RowsAffected() != 1 {
				err = login.ErrConflict
			}
		}
		if err != nil {
			return nil, classify(err)
		}
	}
	after, err := listBindingsQuerier(ctx, tx, principal.TenantID, appID)
	if err != nil {
		return nil, err
	}
	if err = audit(ctx, tx, principal, "app.login_provider_binding.update", "app.login_provider_binding.update", "application_login_provider_bindings", appID.String(), "PUT", "/admin-api/v1/apps/"+appID.String()+"/login-provider-bindings", auditBindings(before), auditBindings(after)); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, classify(err)
	}
	return after, nil
}

func auditConfig(item login.Config) map[string]any {
	return map[string]any{
		"id": item.ID, "name": item.Name, "provider_code": item.ProviderCode,
		"external_client_id_hint": hint(item.ExternalClientID), "status": item.Status,
		"secret_field_names": item.SecretFieldNames, "preflight_status": item.LastPreflightStatus,
		"lock_version": item.LockVersion,
	}
}

func auditBindings(items []login.Binding) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"provider_code": item.ProviderCode, "login_provider_config_id": item.LoginProviderConfigID,
			"enabled": item.Enabled, "sort_order": item.SortOrder, "lock_version": item.LockVersion,
		})
	}
	return out
}

func hint(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func audit(ctx context.Context, tx pgx.Tx, principal login.Principal, action, permission, resource, resourceID, method, path string, before, after any) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	_, err := tx.Exec(ctx, `INSERT INTO audit.operation_logs(
tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,
http_method,request_path,response_status,client_ip,user_agent,before_data,after_data,succeeded)
VALUES($1,$2,nullif($3,'00000000-0000-0000-0000-000000000000'::uuid),nullif($4,''),'login_provider',$5,$6,$7,$8,$9,$10,200,nullif($11,'')::inet,nullif($12,''),$13,$14,true)`,
		principal.TenantID, principal.UserID, principal.SessionID, principal.RequestID, action, permission, resource, resourceID,
		method, path, principal.IPAddress, principal.UserAgent, beforeJSON, afterJSON)
	return err
}

func classify(err error) error {
	if err == nil || errors.Is(err, login.ErrConflict) {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "40001":
			return login.ErrConflict
		case "23503", "23514", "22P02", "22001":
			return login.ErrInvalid
		}
	}
	return err
}
