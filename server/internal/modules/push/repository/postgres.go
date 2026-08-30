package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	push "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool  *pgxpool.Pool
	queue jobqueue.Enqueuer
}

func NewPostgres(pool *pgxpool.Pool, queue jobqueue.Enqueuer) *Postgres {
	return &Postgres{pool: pool, queue: queue}
}

type scanner interface{ Scan(...any) error }

const configColumns = `id,app_id,environment,provider,config_schema_version,public_config,secret_field_names,
(secret_ciphertext IS NOT NULL),coalesce(credential_fingerprint,''),status,last_preflight_at,
coalesce(last_preflight_status,''),last_preflight_issues,lock_version,created_at,updated_at`

func scanConfig(row scanner) (push.ProviderConfig, error) {
	var out push.ProviderConfig
	var issues []byte
	err := row.Scan(&out.ID, &out.AppID, &out.Environment, &out.Provider, &out.ConfigSchemaVersion, &out.PublicConfig,
		&out.SecretFieldNames, &out.HasSecret, &out.CredentialFingerprint, &out.Status, &out.LastPreflightAt,
		&out.LastPreflightStatus, &issues, &out.LockVersion, &out.CreatedAt, &out.UpdatedAt)
	if err == nil {
		_ = json.Unmarshal(issues, &out.LastPreflightIssues)
		if out.LastPreflightIssues == nil {
			out.LastPreflightIssues = []string{}
		}
		if out.SecretFieldNames == nil {
			out.SecretFieldNames = []string{}
		}
	}
	return out, err
}

func scanDevice(row scanner) (push.Device, error) {
	var out push.Device
	err := row.Scan(&out.ID, &out.Provider, &out.Platform, &out.BuildVariant, &out.Locale, &out.SDKVersion,
		&out.AppVersion, &out.Status, &out.RegisteredAt, &out.TokenUpdatedAt, &out.InvalidatedAt)
	return out, err
}

func (r *Postgres) SessionDevice(ctx context.Context, sessionID, userID, appID uuid.UUID) (uuid.UUID, error) {
	var deviceID *uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT s.device_id FROM iam.sessions s
WHERE s.id=$1 AND s.user_id=$2 AND s.app_id=$3 AND s.audience='ak-mobile'
  AND s.status='active' AND s.revoked_at IS NULL`, sessionID, userID, appID).Scan(&deviceID)
	if errors.Is(err, pgx.ErrNoRows) || deviceID == nil {
		return uuid.Nil, push.ErrUnavailable
	}
	return *deviceID, err
}

func (r *Postgres) HasCurrentLegalConsent(ctx context.Context, appID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(DISTINCT p.page_type)
FROM content.pages p
JOIN content.page_revisions rev ON rev.id=p.current_revision_id AND rev.app_id=p.app_id
JOIN app.legal_consents c ON c.app_id=p.app_id AND c.user_id=$2 AND c.page_type=p.page_type
  AND c.revision_id=rev.id AND c.content_hash=rev.content_hash
WHERE p.app_id=$1 AND p.status='published' AND p.page_type IN ('privacy-policy','terms-of-service')`, appID, userID).Scan(&count)
	return count == 2, err
}

func (r *Postgres) CurrentDevice(ctx context.Context, p push.Principal) (*push.Device, error) {
	out, err := scanDevice(r.pool.QueryRow(ctx, `SELECT id,provider,platform,build_variant,locale,sdk_version,app_version,status,registered_at,token_updated_at,invalidated_at
FROM notify.push_devices WHERE tenant_id=$1 AND app_id=$2 AND user_id=$3 AND device_id=$4 AND status='active'
ORDER BY updated_at DESC LIMIT 1`, p.TenantID, p.AppID, p.UserID, p.DeviceID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &out, err
}

func (r *Postgres) UpsertDevice(ctx context.Context, p push.Principal, in push.DeviceInput, hash, ciphertext []byte, keyVersion int32) (push.Device, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return push.Device{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `UPDATE notify.push_devices SET status='disabled',invalidated_at=now(),invalid_reason='replaced_by_registration'
WHERE app_id=$1 AND device_id=$2 AND status='active' AND NOT (provider=$3 AND token_hash=$4)`, p.AppID, p.DeviceID, in.Provider, hash); err != nil {
		return push.Device{}, err
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO notify.push_devices(
tenant_id,app_id,user_id,device_id,provider,token_hash,token_ciphertext,key_version,status,platform,build_variant,locale,sdk_version,app_version,registered_at,token_updated_at,invalidated_at,invalid_reason)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,'active',$9,$10,$11,$12,$13,now(),now(),NULL,NULL)
ON CONFLICT(app_id,provider,token_hash) DO UPDATE SET
tenant_id=excluded.tenant_id,user_id=excluded.user_id,device_id=excluded.device_id,token_ciphertext=excluded.token_ciphertext,
key_version=excluded.key_version,status='active',platform=excluded.platform,build_variant=excluded.build_variant,locale=excluded.locale,
sdk_version=excluded.sdk_version,app_version=excluded.app_version,registered_at=now(),token_updated_at=now(),invalidated_at=NULL,invalid_reason=NULL
RETURNING id`, p.TenantID, p.AppID, p.UserID, p.DeviceID, in.Provider, hash, ciphertext, keyVersion, in.Platform, in.BuildVariant, in.Locale, in.SDKVersion, in.AppVersion).Scan(&id)
	if err != nil {
		return push.Device{}, classify(err)
	}
	out, err := scanDevice(tx.QueryRow(ctx, `SELECT id,provider,platform,build_variant,locale,sdk_version,app_version,status,registered_at,token_updated_at,invalidated_at FROM notify.push_devices WHERE id=$1`, id))
	if err != nil {
		return push.Device{}, err
	}
	if err = audit(ctx, tx, p, "notify.push_device.register", "notify.preference.manage_self", "push_device", id.String(), "POST", "/api/v1/me/push-devices", nil, map[string]any{"provider": out.Provider, "platform": out.Platform, "build_variant": out.BuildVariant, "status": out.Status}); err != nil {
		return push.Device{}, err
	}
	return out, tx.Commit(ctx)
}

func (r *Postgres) DisableDevice(ctx context.Context, p push.Principal, id uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE notify.push_devices SET status='disabled',invalidated_at=now(),invalid_reason='user_disabled'
WHERE id=$1 AND tenant_id=$2 AND app_id=$3 AND user_id=$4 AND device_id=$5 AND status<>'disabled'`, id, p.TenantID, p.AppID, p.UserID, p.DeviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return push.ErrNotFound
	}
	if err = audit(ctx, tx, p, "notify.push_device.disable", "notify.preference.manage_self", "push_device", id.String(), "DELETE", "/api/v1/me/push-devices/"+id.String(), nil, map[string]any{"status": "disabled"}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Postgres) MarkOpened(ctx context.Context, p push.Principal, deliveryID uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var messageRunID *uuid.UUID
	err = tx.QueryRow(ctx, `UPDATE notify.deliveries d SET opened_at=coalesce(opened_at,now()),updated_at=now()
WHERE d.id=$1 AND d.tenant_id=$2 AND d.app_id=$3 AND d.user_id=$4 AND d.channel='push'
	  AND EXISTS (SELECT 1 FROM notify.push_devices pd WHERE pd.id=d.push_device_id AND pd.device_id=$5)
	RETURNING d.message_run_id`, deliveryID, p.TenantID, p.AppID, p.UserID, p.DeviceID).Scan(&messageRunID)
	if errors.Is(err, pgx.ErrNoRows) {
		return push.ErrNotFound
	}
	if err != nil {
		return err
	}
	if messageRunID != nil {
		if _, err = tx.Exec(ctx, `UPDATE notify.message_runs r SET opened_count=(SELECT count(*) FROM notify.deliveries d WHERE d.message_run_id=r.id AND d.opened_at IS NOT NULL) WHERE r.id=$1`, *messageRunID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Postgres) ListConfigs(ctx context.Context, tenantID, appID uuid.UUID, environment string) ([]push.ProviderConfig, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+configColumns+` FROM notify.push_provider_configs
WHERE tenant_id=$1 AND app_id=$2 AND environment=$3 ORDER BY provider`, tenantID, appID, environment)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []push.ProviderConfig{}
	for rows.Next() {
		item, scanErr := scanConfig(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Postgres) GetConfig(ctx context.Context, tenantID, appID, id uuid.UUID) (push.ProviderConfig, error) {
	out, err := scanConfig(r.pool.QueryRow(ctx, `SELECT `+configColumns+` FROM notify.push_provider_configs WHERE tenant_id=$1 AND app_id=$2 AND id=$3`, tenantID, appID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, push.ErrNotFound
	}
	return out, err
}

func getConfigTx(ctx context.Context, tx pgx.Tx, tenantID, appID, id uuid.UUID, lock bool) (push.ProviderConfig, error) {
	sql := `SELECT ` + configColumns + ` FROM notify.push_provider_configs WHERE tenant_id=$1 AND app_id=$2 AND id=$3`
	if lock {
		sql += ` FOR UPDATE`
	}
	out, err := scanConfig(tx.QueryRow(ctx, sql, tenantID, appID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, push.ErrNotFound
	}
	return out, err
}

func (r *Postgres) UpsertConfig(ctx context.Context, p push.Principal, in push.ProviderConfigInput) (push.ProviderConfig, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return push.ProviderConfig{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	var existingLock int32
	err = tx.QueryRow(ctx, `SELECT id,lock_version FROM notify.push_provider_configs WHERE tenant_id=$1 AND app_id=$2 AND environment=$3 AND provider=$4 FOR UPDATE`, p.TenantID, p.AppID, in.Environment, in.Provider).Scan(&id, &existingLock)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if in.LockVersion != 0 {
			return push.ProviderConfig{}, push.ErrConflict
		}
		err = tx.QueryRow(ctx, `INSERT INTO notify.push_provider_configs(tenant_id,app_id,environment,provider,config_schema_version,public_config,created_by,updated_by)
VALUES($1,$2,$3,$4,$5,$6,$7,$7) RETURNING id`, p.TenantID, p.AppID, in.Environment, in.Provider, in.ConfigSchemaVersion, in.PublicConfig, p.UserID).Scan(&id)
	case err != nil:
		return push.ProviderConfig{}, err
	default:
		if in.LockVersion != existingLock {
			return push.ProviderConfig{}, push.ErrConflict
		}
		tag, updateErr := tx.Exec(ctx, `UPDATE notify.push_provider_configs SET config_schema_version=$1,public_config=$2,status='draft',last_preflight_at=NULL,last_preflight_status=NULL,last_preflight_issues='[]',lock_version=lock_version+1,updated_by=$3 WHERE id=$4 AND lock_version=$5`, in.ConfigSchemaVersion, in.PublicConfig, p.UserID, id, in.LockVersion)
		if updateErr != nil {
			return push.ProviderConfig{}, classify(updateErr)
		}
		if tag.RowsAffected() != 1 {
			return push.ProviderConfig{}, push.ErrConflict
		}
	}
	if err != nil {
		return push.ProviderConfig{}, classify(err)
	}
	out, err := getConfigTx(ctx, tx, p.TenantID, p.AppID, id, false)
	if err != nil {
		return push.ProviderConfig{}, err
	}
	if err = audit(ctx, tx, p, "notify.push_provider.save", "notify.push_provider.manage", "push_provider_config", id.String(), "PUT", "/admin-api/v1/apps/"+p.AppID.String()+"/push-provider-configs/"+in.Provider, nil, safeConfig(out)); err != nil {
		return push.ProviderConfig{}, err
	}
	return out, tx.Commit(ctx)
}

func (r *Postgres) RotateSecret(ctx context.Context, p push.Principal, id uuid.UUID, lockVersion int32, ciphertext []byte, keyVersion int32, names []string, fingerprint string) (push.ProviderConfig, error) {
	return r.updateConfig(ctx, p, id, lockVersion, "notify.push_provider.rotate_secret", "notify.push_provider.rotate_secret", func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE notify.push_provider_configs SET secret_ciphertext=$1,secret_key_version=$2,secret_field_names=$3,credential_fingerprint=$4,status='draft',last_preflight_at=NULL,last_preflight_status=NULL,last_preflight_issues='[]',lock_version=lock_version+1,updated_by=$5 WHERE id=$6 AND tenant_id=$7 AND app_id=$8 AND lock_version=$9`, ciphertext, keyVersion, names, fingerprint, p.UserID, id, p.TenantID, p.AppID, lockVersion)
		return err
	})
}

func (r *Postgres) SetStatus(ctx context.Context, p push.Principal, id uuid.UUID, lockVersion int32, status string) (push.ProviderConfig, error) {
	return r.updateConfig(ctx, p, id, lockVersion, "notify.push_provider."+status, "notify.push_provider.manage", func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE notify.push_provider_configs SET status=$1,lock_version=lock_version+1,updated_by=$2 WHERE id=$3 AND tenant_id=$4 AND app_id=$5 AND lock_version=$6`, status, p.UserID, id, p.TenantID, p.AppID, lockVersion)
		return err
	})
}

func (r *Postgres) RecordPreflight(ctx context.Context, p push.Principal, id uuid.UUID, lockVersion int32, result push.Preflight) (push.ProviderConfig, error) {
	status := "failed"
	if result.Ready {
		status = "ready"
	}
	issues, _ := json.Marshal(result.Issues)
	return r.updateConfig(ctx, p, id, lockVersion, "notify.push.preflight", "notify.push.preflight", func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE notify.push_provider_configs SET last_preflight_at=$1,last_preflight_status=$2,last_preflight_issues=$3,lock_version=lock_version+1,updated_by=$4 WHERE id=$5 AND tenant_id=$6 AND app_id=$7 AND lock_version=$8`, result.CheckedAt, status, issues, p.UserID, id, p.TenantID, p.AppID, lockVersion)
		return err
	})
}

func (r *Postgres) updateConfig(ctx context.Context, p push.Principal, id uuid.UUID, lockVersion int32, action, permission string, mutate func(pgx.Tx) error) (push.ProviderConfig, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return push.ProviderConfig{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := getConfigTx(ctx, tx, p.TenantID, p.AppID, id, true)
	if err != nil {
		return push.ProviderConfig{}, err
	}
	if before.LockVersion != lockVersion {
		return push.ProviderConfig{}, push.ErrConflict
	}
	if err = mutate(tx); err != nil {
		return push.ProviderConfig{}, classify(err)
	}
	after, err := getConfigTx(ctx, tx, p.TenantID, p.AppID, id, false)
	if err != nil {
		return push.ProviderConfig{}, err
	}
	if after.LockVersion != lockVersion+1 {
		return push.ProviderConfig{}, push.ErrConflict
	}
	if err = audit(ctx, tx, p, action, permission, "push_provider_config", id.String(), "POST", "/admin-api/v1/apps/"+p.AppID.String()+"/push-provider-configs/"+id.String(), safeConfig(before), safeConfig(after)); err != nil {
		return push.ProviderConfig{}, err
	}
	return after, tx.Commit(ctx)
}

func (r *Postgres) RuntimeCapability(ctx context.Context, appID uuid.UUID, environment string) (push.RuntimeCapability, error) {
	rows, err := r.pool.Query(ctx, `SELECT provider FROM notify.push_provider_configs WHERE app_id=$1 AND environment=$2 AND status='active' AND last_preflight_status='ready' ORDER BY provider`, appID, environment)
	if err != nil {
		return push.RuntimeCapability{}, err
	}
	defer rows.Close()
	providers := []string{}
	variants := map[string]bool{}
	for rows.Next() {
		var provider string
		if err = rows.Scan(&provider); err != nil {
			return push.RuntimeCapability{}, err
		}
		providers = append(providers, provider)
		switch provider {
		case push.ProviderAPNS:
			variants["ios"] = true
		case push.ProviderFCM:
			variants["android_google"] = true
		case push.ProviderHarmony:
			variants["harmony"] = true
		default:
			variants["android_china"] = true
		}
	}
	buildVariants := []string{}
	for _, candidate := range []string{"ios", "android_google", "android_china", "harmony"} {
		if variants[candidate] {
			buildVariants = append(buildVariants, candidate)
		}
	}
	return push.RuntimeCapability{Enabled: len(providers) > 0, Environment: environment, Providers: providers, BuildVariants: buildVariants}, rows.Err()
}

func (r *Postgres) TestDevice(ctx context.Context, tenantID, appID, id uuid.UUID) (push.Device, error) {
	out, err := scanDevice(r.pool.QueryRow(ctx, `SELECT id,provider,platform,build_variant,locale,sdk_version,app_version,status,registered_at,token_updated_at,invalidated_at FROM notify.push_devices WHERE id=$1 AND tenant_id=$2 AND app_id=$3 AND status='active'`, id, tenantID, appID))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, push.ErrNotFound
	}
	return out, err
}

func (r *Postgres) ListTestDevices(ctx context.Context, tenantID, appID uuid.UUID, provider string) ([]push.Device, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,provider,platform,build_variant,locale,sdk_version,app_version,status,registered_at,token_updated_at,invalidated_at
FROM notify.push_devices WHERE tenant_id=$1 AND app_id=$2 AND provider=$3 AND status='active'
ORDER BY token_updated_at DESC,id DESC LIMIT 100`, tenantID, appID, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []push.Device{}
	for rows.Next() {
		item, scanErr := scanDevice(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Postgres) DeliverySummary(ctx context.Context, tenantID, appID uuid.UUID) ([]push.DeliverySummaryItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT provider,
coalesce(nullif(metadata->>'push_category',''),'service_security') AS category,
coalesce(provider_result,status) AS result,count(*),count(*) FILTER (WHERE opened_at IS NOT NULL)
FROM notify.deliveries
WHERE tenant_id=$1 AND app_id=$2 AND channel='push' AND created_at >= now()-interval '30 days'
GROUP BY provider,category,result ORDER BY provider,category,result`, tenantID, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []push.DeliverySummaryItem{}
	for rows.Next() {
		var item push.DeliverySummaryItem
		if err = rows.Scan(&item.Provider, &item.Category, &item.Result, &item.Count, &item.OpenedCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Postgres) QueueTestDelivery(ctx context.Context, p push.Principal, id, configID, deviceID uuid.UUID, payload []byte, payloadKeyVersion int32) error {
	if r.queue == nil {
		return push.ErrUnavailable
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var inserted uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO notify.deliveries(
id,tenant_id,app_id,user_id,channel,target_ciphertext,target_hash,target_hint,target_key_version,provider,
rendered_subject,rendered_body,dedupe_key,payload_ciphertext,payload_key_version,push_device_id,status,max_attempts,metadata)
SELECT $1,d.tenant_id,d.app_id,d.user_id,'push',d.token_ciphertext,d.token_hash,'device:'||right(d.id::text,8),d.key_version,d.provider,
'','', $2,$3,$4,d.id,'pending',5,jsonb_build_object('push_config_id',$5::text,'test',true)
FROM notify.push_devices d WHERE d.id=$6 AND d.tenant_id=$7 AND d.app_id=$8 AND d.status='active'
RETURNING id`, id, "push-test:"+id.String(), payload, payloadKeyVersion, configID, deviceID, p.TenantID, p.AppID).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return push.ErrNotFound
	}
	if err != nil {
		return classify(err)
	}
	if inserted != id {
		return push.ErrConflict
	}
	appID := p.AppID
	run, enqueueErr := r.queue.EnqueueTx(ctx, tx, jobqueue.Spec{
		Scope: jobqueue.Scope{TenantID: p.TenantID, AppID: &appID, ModuleCode: "notify", ResourceType: "notification_delivery", ResourceID: &id},
		Args:  notify.DeliveryJobArgs{DeliveryID: id}, Queue: "notifications", MaxAttempts: 5, UniqueByArgs: true,
	})
	if enqueueErr != nil {
		return enqueueErr
	}
	if _, err = tx.Exec(ctx, `UPDATE notify.deliveries SET task_run_id=$2 WHERE id=$1`, id, run.ID); err != nil {
		return err
	}
	if err = audit(ctx, tx, p, "notify.push.test", "notify.push.test", "notification_delivery", id.String(), "POST", "/admin-api/v1/apps/"+p.AppID.String()+"/push-provider-configs/"+configID.String()+"/test", nil, map[string]any{"delivery_id": id, "push_device_id": deviceID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func safeConfig(item push.ProviderConfig) map[string]any {
	return map[string]any{"id": item.ID, "app_id": item.AppID, "environment": item.Environment, "provider": item.Provider, "status": item.Status, "secret_field_names": item.SecretFieldNames, "credential_fingerprint": item.CredentialFingerprint, "last_preflight_status": item.LastPreflightStatus, "lock_version": item.LockVersion}
}

func audit(ctx context.Context, tx pgx.Tx, p push.Principal, action, permission, resource, resourceID, method, path string, before, after any) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	var ip *netip.Addr
	if parsed, err := netip.ParseAddr(strings.TrimSpace(p.IPAddress)); err == nil {
		ip = &parsed
	}
	_, err := tx.Exec(ctx, `INSERT INTO audit.operation_logs(
tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,
http_method,request_path,response_status,client_ip,user_agent,before_data,after_data,succeeded)
VALUES($1,$2,nullif($3,'00000000-0000-0000-0000-000000000000'::uuid),nullif($4,''),'notify',$5,$6,$7,$8,$9,$10,200,$11,nullif($12,''),$13,$14,true)`,
		p.TenantID, p.UserID, p.SessionID, p.RequestID, action, permission, resource, resourceID, method, path, ip, p.UserAgent, beforeJSON, afterJSON)
	return err
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "40001":
			return push.ErrConflict
		case "23503", "23514", "22P02":
			return push.ErrInvalid
		}
	}
	return fmt.Errorf("push repository: %w", err)
}

var _ = time.Second
