package repository

import (
	"context"
	"errors"
	"time"

	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Postgres) ResolveApp(ctx context.Context, appID uuid.UUID) (uuid.UUID, string, error) {
	var tenantID uuid.UUID
	var locale string
	err := r.pool.QueryRow(ctx, `SELECT tenant_id,default_locale FROM app.applications
WHERE id=$1 AND status='active' AND deleted_at IS NULL`, appID).Scan(&tenantID, &locale)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", login.ErrNotFound
	}
	return tenantID, locale, err
}

const runtimeSelect = `SELECT a.tenant_id,a.id,c.id,a.default_locale,b.provider_code,c.external_client_id,c.public_config,
c.secret_ciphertext,c.secret_key_version,c.secret_field_names,a.registration_enabled,b.enabled,b.sort_order
FROM app.application_login_provider_bindings b
JOIN app.applications a ON a.tenant_id=b.tenant_id AND a.id=b.app_id
JOIN sys.login_provider_configs c ON c.tenant_id=b.tenant_id AND c.id=b.login_provider_config_id AND c.provider_code=b.provider_code`

func scanRuntime(row scanner) (login.RuntimeProvider, error) {
	var out login.RuntimeProvider
	err := row.Scan(
		&out.TenantID, &out.AppID, &out.ConfigID, &out.DefaultLocale, &out.ProviderCode, &out.ExternalClientID, &out.PublicConfig,
		&out.SecretCiphertext, &out.SecretKeyVersion, &out.SecretFieldNames, &out.RegistrationEnabled, &out.Enabled, &out.SortOrder,
	)
	if out.SecretFieldNames == nil {
		out.SecretFieldNames = []string{}
	}
	return out, err
}

func (r *Postgres) RuntimeProviders(ctx context.Context, appID uuid.UUID) ([]login.RuntimeProvider, error) {
	rows, err := r.pool.Query(ctx, runtimeSelect+` WHERE a.id=$1 AND a.status='active' AND a.deleted_at IS NULL
AND b.enabled AND c.status='active' AND c.last_preflight_status='ready' AND c.deleted_at IS NULL
ORDER BY b.sort_order,b.provider_code`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []login.RuntimeProvider{}
	for rows.Next() {
		item, scanErr := scanRuntime(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Postgres) RuntimeProvider(ctx context.Context, appID uuid.UUID, providerCode string) (login.RuntimeProvider, error) {
	out, err := scanRuntime(r.pool.QueryRow(ctx, runtimeSelect+` WHERE a.id=$1 AND b.provider_code=$2
AND a.status='active' AND a.deleted_at IS NULL AND b.enabled
AND c.status='active' AND c.last_preflight_status='ready' AND c.deleted_at IS NULL`, appID, providerCode))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, login.ErrProviderUnavailable
	}
	return out, err
}

func (r *Postgres) RuntimeProviderForReauth(ctx context.Context, appID uuid.UUID, providerCode string, userID, accountID uuid.UUID) (login.RuntimeProvider, error) {
	out, err := scanRuntime(r.pool.QueryRow(ctx, runtimeSelect+` JOIN iam.app_oauth_accounts o
ON o.app_id=b.app_id AND o.provider_code=b.provider_code AND o.external_client_id=c.external_client_id
WHERE a.id=$1 AND b.provider_code=$2 AND o.user_id=$3 AND o.id=$4 AND o.status='active'
  AND a.status='active' AND a.deleted_at IS NULL AND c.deleted_at IS NULL`, appID, providerCode, userID, accountID))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, login.ErrProviderUnavailable
	}
	return out, err
}

const flowColumns = `f.id,f.tenant_id,f.app_id,f.login_provider_config_id,f.provider_code,f.mode,
	f.platform,f.build_variant,f.user_id,f.session_id,f.reauth_purpose,f.target_oauth_account_id,f.state_hash,f.nonce_hash,f.pkce_verifier_ciphertext,
f.pkce_key_version,f.device_key_hash,f.verified_identity_ciphertext,f.verified_identity_key_version,
f.completion_ticket_hash,f.completion_ticket_expires_at,f.expires_at,f.provider_verified_at,f.consumed_at,f.failure_count`

const flowSelect = `SELECT ` + flowColumns + ` FROM iam.oauth_authorization_flows f`

func scanFlow(row scanner) (login.Flow, error) {
	var out login.Flow
	err := row.Scan(
		&out.ID, &out.TenantID, &out.AppID, &out.ConfigID, &out.ProviderCode, &out.Mode,
		&out.Platform, &out.BuildVariant, &out.UserID, &out.SessionID, &out.ReauthPurpose, &out.TargetOAuthAccountID, &out.StateHash, &out.NonceHash,
		&out.PKCECiphertext, &out.PKCEKeyVersion, &out.DeviceKeyHash, &out.VerifiedIdentityCiphertext,
		&out.VerifiedIdentityKeyVersion, &out.CompletionTicketHash, &out.CompletionTicketExpiresAt,
		&out.ExpiresAt, &out.ProviderVerifiedAt, &out.ConsumedAt, &out.FailureCount,
	)
	return out, err
}

func (r *Postgres) CreateFlow(ctx context.Context, input login.FlowCreate) (login.Flow, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `INSERT INTO iam.oauth_authorization_flows(
tenant_id,app_id,login_provider_config_id,provider_code,mode,platform,build_variant,user_id,session_id,reauth_purpose,target_oauth_account_id,
state_hash,nonce_hash,pkce_verifier_ciphertext,pkce_key_version,device_key_hash,expires_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,nullif($10,''),$11,$12,$13,$14,$15,$16,$17) RETURNING id`,
		input.TenantID, input.AppID, input.ConfigID, input.ProviderCode, input.Mode, input.Platform, input.BuildVariant,
		input.UserID, input.SessionID, input.ReauthPurpose, input.TargetOAuthAccountID, input.StateHash, nullableBytes(input.NonceHash), nullableBytes(input.PKCECiphertext),
		input.PKCEKeyVersion, input.DeviceKeyHash, input.ExpiresAt).Scan(&id)
	if err != nil {
		return login.Flow{}, classify(err)
	}
	return r.GetFlow(ctx, id, input.AppID, input.ProviderCode, input.DeviceKeyHash)
}

func nullableBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func (r *Postgres) GetFlow(ctx context.Context, id, appID uuid.UUID, providerCode string, deviceHash []byte) (login.Flow, error) {
	out, err := scanFlow(r.pool.QueryRow(ctx, flowSelect+` WHERE f.id=$1 AND f.app_id=$2 AND f.provider_code=$3 AND f.device_key_hash=$4`, id, appID, providerCode, deviceHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, login.ErrFlowInvalid
	}
	return out, err
}

func (r *Postgres) GetBrowserFlow(ctx context.Context, providerCode string, stateHash []byte) (login.Flow, error) {
	out, err := scanFlow(r.pool.QueryRow(ctx, flowSelect+` WHERE f.provider_code=$1 AND f.state_hash=$2
AND f.consumed_at IS NULL AND f.provider_verified_at IS NULL AND f.expires_at>now()`, providerCode, stateHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, login.ErrFlowInvalid
	}
	return out, err
}

func (r *Postgres) ClaimNativeFlow(ctx context.Context, id, appID uuid.UUID, providerCode string, deviceHash, stateHash []byte) (login.Flow, error) {
	out, err := scanFlow(r.pool.QueryRow(ctx, `WITH claimed AS (
UPDATE iam.oauth_authorization_flows SET consumed_at=now()
WHERE id=$1 AND app_id=$2 AND provider_code=$3 AND device_key_hash=$4 AND state_hash=$5
  AND consumed_at IS NULL AND provider_verified_at IS NULL AND expires_at>now()
RETURNING *) SELECT `+flowColumns+` FROM claimed f`, id, appID, providerCode, deviceHash, stateHash))
	if err == nil {
		return out, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	return r.classifyUnclaimableFlow(ctx, id, appID, providerCode, deviceHash)
}

func (r *Postgres) classifyUnclaimableFlow(ctx context.Context, id, appID uuid.UUID, providerCode string, deviceHash []byte) (login.Flow, error) {
	out, err := r.GetFlow(ctx, id, appID, providerCode, deviceHash)
	if err != nil {
		return out, err
	}
	if out.ConsumedAt != nil {
		return out, login.ErrFlowConsumed
	}
	if !out.ExpiresAt.After(time.Now()) {
		return out, login.ErrFlowExpired
	}
	return out, login.ErrFlowInvalid
}

func (r *Postgres) MarkBrowserVerified(ctx context.Context, id uuid.UUID, ticketHash, ciphertext []byte, keyVersion int32) error {
	tag, err := r.pool.Exec(ctx, `UPDATE iam.oauth_authorization_flows
SET verified_identity_ciphertext=$1,verified_identity_key_version=$2,provider_verified_at=now(),
    completion_ticket_hash=$3,completion_ticket_expires_at=LEAST(expires_at,now()+interval '60 seconds')
WHERE id=$4 AND consumed_at IS NULL AND provider_verified_at IS NULL AND expires_at>now()+interval '1 second'`,
		ciphertext, keyVersion, ticketHash, id)
	if err != nil {
		return classify(err)
	}
	if tag.RowsAffected() != 1 {
		return login.ErrFlowConsumed
	}
	return nil
}

func (r *Postgres) ClaimTicketFlow(ctx context.Context, id, appID uuid.UUID, providerCode string, deviceHash, ticketHash []byte) (login.Flow, error) {
	out, err := scanFlow(r.pool.QueryRow(ctx, `WITH claimed AS (
UPDATE iam.oauth_authorization_flows SET consumed_at=now()
WHERE id=$1 AND app_id=$2 AND provider_code=$3 AND device_key_hash=$4 AND completion_ticket_hash=$5
  AND provider_verified_at IS NOT NULL AND completion_ticket_expires_at>now() AND consumed_at IS NULL
RETURNING *) SELECT `+flowColumns+` FROM claimed f`, id, appID, providerCode, deviceHash, ticketHash))
	if err == nil {
		return out, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	return r.classifyUnclaimableFlow(ctx, id, appID, providerCode, deviceHash)
}
