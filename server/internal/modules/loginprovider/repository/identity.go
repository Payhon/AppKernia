package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type accountRecord struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	ProviderCode        string
	ProviderUsername    *string
	Status              string
	BoundAt             time.Time
	LastAuthenticatedAt *time.Time
	LoginEnabled        bool
}

const accountSelect = `SELECT o.id,o.user_id,o.provider_code,o.provider_username,o.status,o.bound_at,o.last_authenticated_at,
EXISTS(SELECT 1 FROM app.application_login_provider_bindings b
JOIN sys.login_provider_configs c ON c.tenant_id=b.tenant_id AND c.id=b.login_provider_config_id
WHERE b.app_id=o.app_id AND b.provider_code=o.provider_code AND b.enabled
  AND c.status='active' AND c.last_preflight_status='ready' AND c.deleted_at IS NULL)
FROM iam.app_oauth_accounts o`

func scanAccount(row scanner) (accountRecord, error) {
	var out accountRecord
	err := row.Scan(&out.ID, &out.UserID, &out.ProviderCode, &out.ProviderUsername, &out.Status, &out.BoundAt, &out.LastAuthenticatedAt, &out.LoginEnabled)
	return out, err
}

func resolveRuntimeTx(ctx context.Context, tx pgx.Tx, expected login.RuntimeProvider, resolution login.IdentityResolution) (login.RuntimeProvider, error) {
	condition := ` AND b.enabled AND c.status='active' AND c.last_preflight_status='ready'`
	args := []any{expected.AppID, expected.ConfigID, expected.ProviderCode}
	if resolution.Mode == "reauth" && resolution.UserID != nil && resolution.TargetOAuthAccountID != nil {
		condition = ` AND EXISTS(SELECT 1 FROM iam.app_oauth_accounts o WHERE o.id=$4 AND o.app_id=a.id
AND o.user_id=$5 AND o.provider_code=b.provider_code AND o.external_client_id=c.external_client_id AND o.status='active')`
		args = append(args, *resolution.TargetOAuthAccountID, *resolution.UserID)
	}
	out, err := scanRuntime(tx.QueryRow(ctx, runtimeSelect+` WHERE a.id=$1 AND c.id=$2 AND b.provider_code=$3
AND a.status='active' AND a.deleted_at IS NULL AND c.deleted_at IS NULL`+condition, args...))
	if errors.Is(err, pgx.ErrNoRows) || err == nil && out.ExternalClientID != expected.ExternalClientID {
		return out, login.ErrProviderUnavailable
	}
	return out, err
}

func findIdentityAccount(ctx context.Context, tx pgx.Tx, appID uuid.UUID, providerCode string, identity login.VerifiedIdentity) (accountRecord, error) {
	out, err := scanAccount(tx.QueryRow(ctx, accountSelect+` WHERE o.app_id=$1 AND o.provider_code=$2 AND o.issuer=$3
AND o.external_client_id=$4 AND o.subject=$5 FOR UPDATE OF o`, appID, providerCode, identity.Issuer, identity.ExternalClientID, identity.Subject))
	if errors.Is(err, pgx.ErrNoRows) {
		return out, login.ErrNotFound
	}
	return out, err
}

func insertAccount(ctx context.Context, tx pgx.Tx, runtime login.RuntimeProvider, userID uuid.UUID, identity login.VerifiedIdentity) (accountRecord, error) {
	profile := identity.Profile
	if len(profile) == 0 {
		profile = json.RawMessage(`{}`)
	}
	var id uuid.UUID
	err := tx.QueryRow(ctx, `INSERT INTO iam.app_oauth_accounts(
tenant_id,app_id,user_id,provider_code,issuer,external_client_id,subject,union_subject,provider_username,provider_profile,last_authenticated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,nullif($8,''),nullif($9,''),$10,now()) RETURNING id`, runtime.TenantID, runtime.AppID, userID,
		runtime.ProviderCode, identity.Issuer, identity.ExternalClientID, identity.Subject, identity.UnionSubject, identity.ProviderUsername, profile).Scan(&id)
	if err != nil {
		return accountRecord{}, classifyIdentity(err)
	}
	return scanAccount(tx.QueryRow(ctx, accountSelect+` WHERE o.id=$1`, id))
}

func createAtomicLoginSession(ctx context.Context, tx pgx.Tx, resolved login.ResolvedIdentity, session login.AtomicLoginSession) error {
	if session.ID == uuid.Nil || len(session.RefreshTokenHash) != 32 || session.AccessTokenVersion != 1 ||
		!session.AbsoluteExpiresAt.After(time.Now().UTC()) || !session.IdleExpiresAt.After(time.Now().UTC()) ||
		!session.RefreshExpiresAt.After(time.Now().UTC()) {
		return login.ErrInvalid
	}
	var deviceID *uuid.UUID
	if session.DeviceKey != "" {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO iam.devices(user_id,device_key,platform,last_ip,last_seen_at)
VALUES($1,$2,'unknown',nullif($3,'')::inet,now())
ON CONFLICT(user_id,device_key) DO UPDATE SET last_ip=EXCLUDED.last_ip,last_seen_at=now(),updated_at=now()
RETURNING id`, resolved.UserID, session.DeviceKey, session.IPAddress).Scan(&id)
		if err != nil {
			return err
		}
		deviceID = &id
	}
	if _, err := tx.Exec(ctx, `INSERT INTO iam.sessions(
id,user_id,tenant_id,app_id,device_id,audience,status,access_token_version,
absolute_expires_at,idle_expires_at,ip_address,user_agent)
VALUES($1,$2,$3,$4,$5,'ak-mobile','active',$6,$7,$8,nullif($9,'')::inet,nullif($10,''))`,
		session.ID, resolved.UserID, resolved.TenantID, resolved.AppID, deviceID, session.AccessTokenVersion,
		session.AbsoluteExpiresAt, session.IdleExpiresAt, session.IPAddress, session.UserAgent); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO iam.refresh_tokens(session_id,token_hash,expires_at,created_ip)
VALUES($1,$2,$3,nullif($4,'')::inet)`, session.ID, session.RefreshTokenHash, session.RefreshExpiresAt, session.IPAddress); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO audit.login_events(
tenant_id,user_id,session_id,app_id,request_id,auth_method,audience,result,client_ip,user_agent,device_info)
VALUES($1,$2,$3,$4,nullif($5,''),'oauth','ak-mobile','success',nullif($6,'')::inet,nullif($7,''),
jsonb_build_object('platform','unknown','registered',$8::boolean))`, resolved.TenantID, resolved.UserID,
		session.ID, resolved.AppID, session.RequestID, session.IPAddress, session.UserAgent, session.DeviceKey != "")
	return err
}

func (r *Postgres) ResolveIdentity(ctx context.Context, resolution login.IdentityResolution, sessionFactory login.LoginSessionFactory) (login.ResolvedIdentity, error) {
	attempts := 1
	if resolution.Mode == "login" {
		attempts = 2
	}
	for attempt := 0; attempt < attempts; attempt++ {
		resolved, err := r.resolveIdentityAttempt(ctx, resolution, sessionFactory)
		if err == nil {
			return resolved, nil
		}
		if attempt+1 == attempts || !retryableIdentityResolution(err) {
			return login.ResolvedIdentity{}, classifyIdentity(err)
		}
	}
	return login.ResolvedIdentity{}, login.ErrIdentityConflict
}

func (r *Postgres) resolveIdentityAttempt(ctx context.Context, resolution login.IdentityResolution, sessionFactory login.LoginSessionFactory) (login.ResolvedIdentity, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return login.ResolvedIdentity{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	runtime, err := resolveRuntimeTx(ctx, tx, resolution.Runtime, resolution)
	if err != nil {
		return login.ResolvedIdentity{}, err
	}
	identity := resolution.Identity
	if strings.TrimSpace(identity.Issuer) == "" || identity.ExternalClientID != runtime.ExternalClientID || strings.TrimSpace(identity.Subject) == "" {
		return login.ResolvedIdentity{}, login.ErrCallbackInvalid
	}
	account, accountErr := findIdentityAccount(ctx, tx, runtime.AppID, runtime.ProviderCode, identity)
	resolved := login.ResolvedIdentity{TenantID: runtime.TenantID, AppID: runtime.AppID}
	switch resolution.Mode {
	case "login":
		if accountErr == nil {
			if account.Status != "active" {
				return login.ResolvedIdentity{}, login.ErrProviderUnavailable
			}
			var activeMembership bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.user_memberships m
WHERE m.app_id=$1 AND m.tenant_id=$2 AND m.user_id=$3 AND m.status='active')`, runtime.AppID, runtime.TenantID, account.UserID).Scan(&activeMembership); err != nil {
				return login.ResolvedIdentity{}, err
			}
			if !activeMembership {
				return login.ResolvedIdentity{}, login.ErrProviderUnavailable
			}
			if _, err = tx.Exec(ctx, `UPDATE iam.app_oauth_accounts SET last_authenticated_at=now() WHERE id=$1`, account.ID); err != nil {
				return login.ResolvedIdentity{}, err
			}
			resolved.UserID = account.UserID
			resolved.Account = mapOAuthAccount(account, runtime.ProviderCode)
			break
		}
		if !errors.Is(accountErr, login.ErrNotFound) {
			return login.ResolvedIdentity{}, accountErr
		}
		if !runtime.RegistrationEnabled {
			return login.ResolvedIdentity{}, login.ErrProviderUnavailable
		}
		if identity.VerifiedEmail != "" {
			var exists bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.user_login_identifiers
WHERE app_id=$1 AND identifier_type='email' AND normalized_value=$2 AND status='active')`, runtime.AppID, strings.ToLower(strings.TrimSpace(identity.VerifiedEmail))).Scan(&exists); err != nil {
				return login.ResolvedIdentity{}, err
			}
			if exists {
				return login.ResolvedIdentity{}, login.ErrLinkRequired
			}
		}
		displayName := strings.Join(strings.Fields(identity.DisplayName), " ")
		if displayName == "" {
			displayName = strings.Join(strings.Fields(identity.ProviderUsername), " ")
		}
		if displayName == "" {
			displayName = runtime.ProviderCode
		}
		if len([]rune(displayName)) > 120 {
			displayName = string([]rune(displayName)[:120])
		}
		locale := runtime.DefaultLocale
		if locale != "en-US" {
			locale = "zh-CN"
		}
		var userID uuid.UUID
		if err = tx.QueryRow(ctx, `INSERT INTO iam.users(display_name,status,locale) VALUES($1,'active',$2) RETURNING id`, displayName, locale).Scan(&userID); err != nil {
			return login.ResolvedIdentity{}, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, runtime.TenantID, userID); err != nil {
			return login.ResolvedIdentity{}, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO app.user_memberships(app_id,tenant_id,user_id,source,status,verified_at)
VALUES($1,$2,$3,'federated_registration','active',now())`, runtime.AppID, runtime.TenantID, userID); err != nil {
			return login.ResolvedIdentity{}, err
		}
		account, err = insertAccount(ctx, tx, runtime, userID, identity)
		if err != nil {
			return login.ResolvedIdentity{}, err
		}
		resolved.UserID, resolved.Account, resolved.Created = userID, mapOAuthAccount(account, runtime.ProviderCode), true
	case "bind":
		if resolution.UserID == nil || resolution.SessionID == nil {
			return login.ResolvedIdentity{}, login.ErrFlowInvalid
		}
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text || ':' || $2::text,0))`, runtime.AppID, *resolution.UserID); err != nil {
			return login.ResolvedIdentity{}, err
		}
		var membership bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.user_memberships
WHERE app_id=$1 AND tenant_id=$2 AND user_id=$3 AND status='active')`, runtime.AppID, runtime.TenantID, *resolution.UserID).Scan(&membership); err != nil || !membership {
			return login.ResolvedIdentity{}, login.ErrForbidden
		}
		if accountErr == nil {
			if account.UserID != *resolution.UserID {
				return login.ResolvedIdentity{}, login.ErrIdentityConflict
			}
			resolved.UserID, resolved.Account = account.UserID, mapOAuthAccount(account, runtime.ProviderCode)
			break
		}
		if !errors.Is(accountErr, login.ErrNotFound) {
			return login.ResolvedIdentity{}, accountErr
		}
		var other bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.app_oauth_accounts
WHERE app_id=$1 AND user_id=$2 AND provider_code=$3)`, runtime.AppID, *resolution.UserID, runtime.ProviderCode).Scan(&other); err != nil {
			return login.ResolvedIdentity{}, err
		}
		if other {
			return login.ResolvedIdentity{}, login.ErrIdentityConflict
		}
		account, err = insertAccount(ctx, tx, runtime, *resolution.UserID, identity)
		if err != nil {
			return login.ResolvedIdentity{}, err
		}
		resolved.UserID, resolved.Account = *resolution.UserID, mapOAuthAccount(account, runtime.ProviderCode)
	case "reauth":
		if resolution.UserID == nil || resolution.SessionID == nil || resolution.TargetOAuthAccountID == nil ||
			accountErr != nil || account.ID != *resolution.TargetOAuthAccountID || account.UserID != *resolution.UserID || account.Status != "active" {
			return login.ResolvedIdentity{}, login.ErrStepUpInvalid
		}
		if _, err = tx.Exec(ctx, `UPDATE iam.app_oauth_accounts SET last_authenticated_at=now() WHERE id=$1`, account.ID); err != nil {
			return login.ResolvedIdentity{}, err
		}
		resolved.UserID, resolved.Account = account.UserID, mapOAuthAccount(account, runtime.ProviderCode)
	default:
		return login.ResolvedIdentity{}, login.ErrFlowInvalid
	}
	securitySessionID := resolution.SessionID
	if resolution.Mode == "login" {
		if sessionFactory == nil {
			return login.ResolvedIdentity{}, login.ErrProviderUnavailable
		}
		session, sessionErr := sessionFactory(ctx, resolved.UserID, resolved.TenantID, resolved.AppID)
		if sessionErr != nil {
			return login.ResolvedIdentity{}, sessionErr
		}
		if sessionErr = createAtomicLoginSession(ctx, tx, resolved, session); sessionErr != nil {
			return login.ResolvedIdentity{}, sessionErr
		}
		securitySessionID = &session.ID
	}
	if resolution.Mode != "login" || resolved.Created {
		details, _ := json.Marshal(map[string]any{"app_id": runtime.AppID, "provider_code": runtime.ProviderCode, "mode": resolution.Mode, "created": resolved.Created})
		_, err = tx.Exec(ctx, `INSERT INTO audit.security_events(tenant_id,user_id,session_id,app_id,event_type,severity,source,details)
VALUES($1,$2,$3,$4,$5,'info','oauth',$6)`, runtime.TenantID, resolved.UserID, securitySessionID, runtime.AppID,
			"iam.oauth."+resolution.Mode, details)
		if err != nil {
			return login.ResolvedIdentity{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return login.ResolvedIdentity{}, classifyIdentity(err)
	}
	return resolved, nil
}

func retryableIdentityResolution(err error) bool {
	if errors.Is(err, login.ErrIdentityConflict) || errors.Is(err, login.ErrConflict) {
		return true
	}
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && (postgresError.Code == "23505" || postgresError.Code == "40001")
}

func mapOAuthAccount(record accountRecord, providerCode string) login.OAuthAccount {
	username := ""
	if record.ProviderUsername != nil {
		username = providerHint(*record.ProviderUsername)
	}
	return login.OAuthAccount{
		ID: record.ID, ProviderCode: providerCode, DisplayNameKey: "login_provider." + providerCode,
		ProviderUsernameHint: username, Status: record.Status, LoginEnabled: record.LoginEnabled,
		LoginCapable: record.Status == "active" && record.LoginEnabled, CanBind: false, CanChange: false,
		BoundAt: record.BoundAt, LastAuthenticatedAt: record.LastAuthenticatedAt,
	}
}

func providerHint(value string) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) <= 3 {
		return "***"
	}
	runes := []rune(value)
	return string(runes[:1]) + "***" + string(runes[len(runes)-1:])
}

func (r *Postgres) ListOAuthAccounts(ctx context.Context, appID, userID uuid.UUID) ([]login.OAuthAccount, error) {
	rows, err := r.pool.Query(ctx, accountSelect+` WHERE o.app_id=$1 AND o.user_id=$2 ORDER BY o.bound_at,o.id`, appID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []accountRecord{}
	for rows.Next() {
		item, scanErr := scanAccount(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	usable, err := r.usableLoginMethodCount(ctx, r.pool, appID, userID, uuid.Nil, "")
	if err != nil {
		return nil, err
	}
	items := make([]login.OAuthAccount, 0, len(records))
	for _, record := range records {
		item := mapOAuthAccount(record, record.ProviderCode)
		remaining := usable
		if record.Status == "active" && record.LoginEnabled {
			remaining--
		}
		item.CanUnbind = remaining > 0
		if !item.CanUnbind {
			item.BlockReason = "last_login_method"
		}
		items = append(items, item)
	}
	return items, nil
}

type querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *Postgres) usableLoginMethodCount(ctx context.Context, queryer querier, appID, userID, excludedAccount uuid.UUID, excludedIdentifier string) (int, error) {
	var password bool
	err := queryer.QueryRow(ctx, `SELECT EXISTS(
SELECT 1 FROM iam.user_credentials c
JOIN app.user_login_identifiers i ON i.user_id=c.user_id AND i.app_id=$1
WHERE c.user_id=$2 AND i.identifier_type='email' AND i.status='active' AND i.verified_at IS NOT NULL
  AND length(i.normalized_value::text)<=254
  AND i.normalized_value::text ~ '^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$')`, appID, userID).Scan(&password)
	if err != nil {
		return 0, err
	}
	var identifiers int
	err = queryer.QueryRow(ctx, `SELECT count(*) FROM app.user_login_identifiers
WHERE app_id=$1 AND user_id=$2 AND status='active' AND verified_at IS NOT NULL
AND CASE
  WHEN identifier_type='mobile' THEN normalized_value::text ~ '^\+[1-9][0-9]{7,14}$'
  WHEN identifier_type='email' THEN length(normalized_value::text)<=254
    AND normalized_value::text ~ '^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$'
  ELSE false
END
AND (nullif($3,'') IS NULL OR identifier_type<>$3)`, appID, userID, excludedIdentifier).Scan(&identifiers)
	if err != nil {
		return 0, err
	}
	var oauth int
	err = queryer.QueryRow(ctx, `SELECT count(*) FROM iam.app_oauth_accounts o
WHERE o.app_id=$1 AND o.user_id=$2 AND o.status='active' AND o.id<>$3
AND EXISTS(SELECT 1 FROM app.application_login_provider_bindings b
JOIN sys.login_provider_configs c ON c.tenant_id=b.tenant_id AND c.id=b.login_provider_config_id
WHERE b.app_id=o.app_id AND b.provider_code=o.provider_code AND b.enabled
  AND c.status='active' AND c.last_preflight_status='ready' AND c.deleted_at IS NULL)`, appID, userID, excludedAccount).Scan(&oauth)
	if err != nil {
		return 0, err
	}
	total := identifiers + oauth
	if password && excludedIdentifier != "email" {
		total++
	}
	return total, nil
}

func (r *Postgres) DeleteOAuthAccount(ctx context.Context, principal login.Principal, appID, accountID uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	record, err := scanAccount(tx.QueryRow(ctx, accountSelect+` WHERE o.id=$1 AND o.app_id=$2 AND o.user_id=$3 FOR UPDATE OF o`, accountID, appID, principal.UserID))
	if errors.Is(err, pgx.ErrNoRows) {
		return login.ErrNotFound
	}
	if err != nil {
		return err
	}
	remaining, err := r.usableLoginMethodCount(ctx, tx, appID, principal.UserID, accountID, "")
	if err != nil {
		return err
	}
	if remaining < 1 {
		return login.ErrLastLoginMethod
	}
	if _, err = tx.Exec(ctx, `DELETE FROM iam.app_oauth_accounts WHERE id=$1 AND app_id=$2 AND user_id=$3`, accountID, appID, principal.UserID); err != nil {
		return err
	}
	details, _ := json.Marshal(map[string]any{"app_id": appID, "provider_code": record.ProviderCode, "account_id": accountID})
	if _, err = tx.Exec(ctx, `INSERT INTO audit.security_events(tenant_id,user_id,session_id,app_id,event_type,severity,source,client_ip,details)
VALUES($1,$2,$3,$4,'iam.oauth.unbind','medium','oauth',nullif($5,'')::inet,$6)`, principal.TenantID, principal.UserID, principal.SessionID, appID, principal.IPAddress, details); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func classifyIdentity(err error) error {
	classified := classify(err)
	if errors.Is(classified, login.ErrConflict) {
		return login.ErrIdentityConflict
	}
	return classified
}
