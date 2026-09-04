package repository

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const otpCooldown = time.Minute

func challengeType(purpose, identifierType string) (string, error) {
	switch purpose {
	case "login":
		return "login_otp", nil
	case "registration":
		return "registration_otp", nil
	case "password_reset":
		return "password_reset", nil
	case "bind":
		if identifierType == "email" {
			return "email_verify", nil
		}
		if identifierType == "mobile" {
			return "mobile_verify", nil
		}
	case "step_up":
		if identifierType == "email" {
			return "email_otp", nil
		}
		if identifierType == "mobile" {
			return "sms_otp", nil
		}
	}
	return "", login.ErrInvalid
}

func (r *Postgres) CreateOTPChallenge(ctx context.Context, challenge login.OTPChallenge) (uuid.UUID, error) {
	challengeKind, err := challengeType(challenge.Purpose, challenge.IdentifierType)
	if err != nil || r.otp == nil || challenge.ID == uuid.Nil || challenge.TenantID == uuid.Nil || challenge.AppID == uuid.Nil ||
		len(challenge.SecretHash) != 32 || challenge.NormalizedValue == "" || challenge.DisplayHint == "" || !challenge.ExpiresAt.After(time.Now().UTC()) {
		return uuid.Nil, login.ErrDeliveryUnavailable
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var active bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications
WHERE id=$1 AND tenant_id=$2 AND status='active' AND deleted_at IS NULL)`, challenge.AppID, challenge.TenantID).Scan(&active); err != nil || !active {
		if err != nil {
			return uuid.Nil, err
		}
		return uuid.Nil, login.ErrProviderUnavailable
	}
	targetHash := hashValue(challenge.NormalizedValue)
	var latestID uuid.UUID
	var latest time.Time
	if err = tx.QueryRow(ctx, `SELECT id,created_at FROM iam.verification_challenges
WHERE challenge_type=$1 AND target_hash=$2 AND metadata->>'app_id'=$3 AND metadata->>'purpose'=$4
ORDER BY created_at DESC,id DESC LIMIT 1`, challengeKind, targetHash, challenge.AppID.String(), challenge.Purpose).Scan(&latestID, &latest); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, err
		}
	}
	if latestID != uuid.Nil && latest.Add(otpCooldown).After(time.Now().UTC()) {
		return latestID, login.ErrConflict
	}
	var appCount, ipCount, deviceCount int
	deviceHashHex := hex.EncodeToString(challenge.DeviceKeyHash)
	err = tx.QueryRow(ctx, `SELECT
count(*) FILTER (WHERE metadata->>'app_id'=$1),
count(*) FILTER (WHERE created_ip=nullif($2,'')::inet),
count(*) FILTER (WHERE metadata->>'device_key_hash'=nullif($3,''))
FROM iam.verification_challenges WHERE created_at>now()-interval '10 minutes' AND challenge_type IN ('login_otp','registration_otp','password_reset','email_verify','mobile_verify','email_otp','sms_otp')`,
		challenge.AppID.String(), challenge.CreatedIP, deviceHashHex).Scan(&appCount, &ipCount, &deviceCount)
	if err != nil {
		return uuid.Nil, err
	}
	if appCount >= 100 || challenge.CreatedIP != "" && ipCount >= 10 || deviceHashHex != "" && deviceCount >= 10 {
		return uuid.Nil, login.ErrConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET consumed_at=COALESCE(consumed_at,now())
WHERE challenge_type=$1 AND target_hash=$2 AND metadata->>'app_id'=$3 AND metadata->>'purpose'=$4 AND consumed_at IS NULL`, challengeKind, targetHash, challenge.AppID.String(), challenge.Purpose); err != nil {
		return uuid.Nil, err
	}
	metadata, _ := json.Marshal(map[string]string{
		"app_id": challenge.AppID.String(), "tenant_id": challenge.TenantID.String(),
		"identifier_type": challenge.IdentifierType, "purpose": challenge.Purpose,
		"device_key_hash": deviceHashHex,
	})
	_, err = tx.Exec(ctx, `INSERT INTO iam.verification_challenges(
id,user_id,challenge_type,target_hash,target_hint,secret_hash,max_attempts,expires_at,created_ip,metadata)
VALUES($1,$2,$3,$4,$5,$6,5,$7,nullif($8,'')::inet,$9)`, challenge.ID, challenge.UserID, challengeKind, targetHash,
		challenge.DisplayHint, challenge.SecretHash, challenge.ExpiresAt, challenge.CreatedIP, metadata)
	if err != nil {
		return uuid.Nil, classify(err)
	}
	// Execute the exact same delivery readiness/encryption/queue path for known
	// and unknown login identifiers. A dummy's delivery work is always rolled
	// back to the savepoint; a real delivery failure is also rolled back while
	// the challenge itself remains committed. This keeps repeated IDs,
	// cooldowns, rate limits and HTTP shapes indistinguishable.
	if _, err = tx.Exec(ctx, `SAVEPOINT login_otp_delivery`); err != nil {
		return uuid.Nil, err
	}
	deliveryErr := r.otp.Queue(ctx, tx, challenge)
	if deliveryErr != nil || challenge.Purpose == "login" && challenge.UserID == nil {
		if _, err = tx.Exec(ctx, `ROLLBACK TO SAVEPOINT login_otp_delivery`); err != nil {
			return uuid.Nil, err
		}
	}
	if _, err = tx.Exec(ctx, `RELEASE SAVEPOINT login_otp_delivery`); err != nil {
		return uuid.Nil, err
	}
	if deliveryErr != nil {
		if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges
SET metadata=metadata || jsonb_build_object('delivery_status','failed') WHERE id=$1`, challenge.ID); err != nil {
			return uuid.Nil, err
		}
		details, _ := json.Marshal(map[string]string{
			"app_id": challenge.AppID.String(), "identifier_type": challenge.IdentifierType,
			"purpose": challenge.Purpose,
		})
		if _, err = tx.Exec(ctx, `INSERT INTO audit.security_events(
tenant_id,user_id,app_id,event_type,severity,source,client_ip,details)
VALUES($1,$2,$3,'iam.otp.delivery_failed','medium','auth',nullif($4,'')::inet,$5)`,
			challenge.TenantID, challenge.UserID, challenge.AppID, challenge.CreatedIP, details); err != nil {
			return uuid.Nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return challenge.ID, nil
}

func (r *Postgres) RegisterWithOTP(ctx context.Context, input login.OTPRegistration, sessionFactory login.LoginSessionFactory) (uuid.UUID, error) {
	challengeKind, err := challengeType("registration", input.IdentifierType)
	if err != nil || input.AppID == uuid.Nil || input.ChallengeID == uuid.Nil || input.NormalizedValue == "" ||
		len(input.TargetHash) != 32 || len(input.SecretHash) != 32 || !hmac.Equal(input.TargetHash, hashValue(input.NormalizedValue)) || sessionFactory == nil {
		return uuid.Nil, login.ErrOTPInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var tenantID uuid.UUID
	var registrationEnabled, otpEnabled, emailEnabled, mobileEnabled bool
	err = tx.QueryRow(ctx, `SELECT a.tenant_id,a.registration_enabled,
COALESCE(s.otp_login_enabled,false),COALESCE(s.email_otp_enabled,true),COALESCE(s.sms_otp_enabled,false)
FROM app.applications a LEFT JOIN app.application_login_settings s ON s.tenant_id=a.tenant_id AND s.app_id=a.id
WHERE a.id=$1 AND a.status='active' AND a.deleted_at IS NULL FOR SHARE OF a`, input.AppID).Scan(
		&tenantID, &registrationEnabled, &otpEnabled, &emailEnabled, &mobileEnabled)
	if err != nil || !registrationEnabled || !otpEnabled || input.IdentifierType == "email" && !emailEnabled || input.IdentifierType == "mobile" && !mobileEnabled {
		return uuid.Nil, login.ErrProviderUnavailable
	}
	var storedSecret []byte
	var attempts, maxAttempts int
	var expiresAt time.Time
	err = tx.QueryRow(ctx, `SELECT secret_hash,attempts,max_attempts,expires_at FROM iam.verification_challenges
WHERE id=$1 AND challenge_type=$2 AND target_hash=$3 AND metadata->>'app_id'=$4
AND metadata->>'identifier_type'=$5 AND metadata->>'purpose'='registration' AND consumed_at IS NULL FOR UPDATE`,
		input.ChallengeID, challengeKind, input.TargetHash, input.AppID.String(), input.IdentifierType).Scan(&storedSecret, &attempts, &maxAttempts, &expiresAt)
	if err != nil || attempts >= maxAttempts || !time.Now().UTC().Before(expiresAt) || !hmac.Equal(storedSecret, input.SecretHash) {
		if err == nil {
			_, _ = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,
consumed_at=CASE WHEN attempts+1>=max_attempts OR expires_at<=now() THEN now() ELSE consumed_at END WHERE id=$1`, input.ChallengeID)
			_ = tx.Commit(ctx)
		}
		return uuid.Nil, login.ErrOTPInvalid
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.user_login_identifiers
WHERE app_id=$1 AND identifier_type=$2 AND normalized_value=$3 AND status='active')`, input.AppID, input.IdentifierType, input.NormalizedValue).Scan(&exists); err != nil {
		return uuid.Nil, err
	}
	if exists {
		return uuid.Nil, login.ErrAccountExists
	}
	locale := input.Locale
	if locale != "en-US" {
		locale = "zh-CN"
	}
	var userID uuid.UUID
	if err = tx.QueryRow(ctx, `INSERT INTO iam.users(display_name,status,locale) VALUES($1,'active',$2) RETURNING id`, input.DisplayName, locale).Scan(&userID); err != nil {
		return uuid.Nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, tenantID, userID); err != nil {
		return uuid.Nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.user_memberships(app_id,tenant_id,user_id,source,status,verified_at)
VALUES($1,$2,$3,'self_registration','active',now())`, input.AppID, tenantID, userID); err != nil {
		return uuid.Nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.user_login_identifiers(
tenant_id,app_id,user_id,identifier_type,normalized_value,display_hint,verified_at,status)
VALUES($1,$2,$3,$4,$5,$6,now(),'active')`, tenantID, input.AppID, userID, input.IdentifierType, input.NormalizedValue, input.DisplayHint); err != nil {
		return uuid.Nil, classify(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,consumed_at=now() WHERE id=$1`, input.ChallengeID); err != nil {
		return uuid.Nil, err
	}
	session, err := sessionFactory(ctx, userID, tenantID, input.AppID)
	if err != nil {
		return uuid.Nil, err
	}
	if err = createAtomicLoginSession(ctx, tx, login.ResolvedIdentity{TenantID: tenantID, AppID: input.AppID, UserID: userID, Created: true}, session); err != nil {
		return uuid.Nil, err
	}
	details, _ := json.Marshal(map[string]any{"app_id": input.AppID, "identifier_type": input.IdentifierType})
	if _, err = tx.Exec(ctx, `INSERT INTO audit.security_events(tenant_id,user_id,session_id,app_id,event_type,severity,source,details)
VALUES($1,$2,$3,$4,'iam.otp.registration','info','auth',$5)`, tenantID, userID, session.ID, input.AppID, details); err != nil {
		return uuid.Nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, classify(err)
	}
	return userID, nil
}

func (r *Postgres) ResetPasswordWithOTP(ctx context.Context, input login.OTPConsume, passwordHash string) error {
	challengeKind, err := challengeType("password_reset", input.IdentifierType)
	if err != nil || input.ID == uuid.Nil || input.AppID == uuid.Nil || input.NormalizedValue == "" || passwordHash == "" ||
		len(input.TargetHash) != 32 || len(input.SecretHash) != 32 || !hmac.Equal(input.TargetHash, hashValue(input.NormalizedValue)) {
		return login.ErrOTPInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID uuid.UUID
	var storedSecret []byte
	var attempts, maxAttempts int
	var expiresAt time.Time
	err = tx.QueryRow(ctx, `SELECT c.user_id,c.secret_hash,c.attempts,c.max_attempts,c.expires_at
FROM iam.verification_challenges c
JOIN app.user_login_identifiers i ON i.user_id=c.user_id AND i.app_id=$2 AND i.identifier_type=$3 AND i.normalized_value=$4
JOIN app.user_memberships m ON m.app_id=i.app_id AND m.user_id=i.user_id AND m.status='active'
JOIN iam.user_credentials p ON p.user_id=i.user_id
WHERE c.id=$1 AND c.challenge_type=$5 AND c.target_hash=$6 AND c.metadata->>'app_id'=$7
AND c.metadata->>'identifier_type'=$3 AND c.metadata->>'purpose'='password_reset' AND c.consumed_at IS NULL
AND i.status='active' AND i.verified_at IS NOT NULL FOR UPDATE OF c,p`, input.ID, input.AppID, input.IdentifierType,
		input.NormalizedValue, challengeKind, input.TargetHash, input.AppID.String()).Scan(&userID, &storedSecret, &attempts, &maxAttempts, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return login.ErrOTPInvalid
	}
	if err != nil {
		return err
	}
	if attempts >= maxAttempts || !time.Now().UTC().Before(expiresAt) || !hmac.Equal(storedSecret, input.SecretHash) {
		if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,
consumed_at=CASE WHEN attempts+1>=max_attempts OR expires_at<=now() THEN now() ELSE consumed_at END WHERE id=$1`, input.ID); err != nil {
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		return login.ErrOTPInvalid
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.user_credentials SET password_hash=$2,password_version=password_version+1,
password_changed_at=now(),force_password_change=false,failed_attempts=0,locked_until=NULL,updated_at=now() WHERE user_id=$1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,consumed_at=now() WHERE id=$1`, input.ID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.sessions SET status='revoked',revoked_at=now(),revoke_reason='app_password_reset',updated_at=now()
WHERE user_id=$1 AND audience='ak-mobile' AND app_id=$2 AND revoked_at IS NULL`, userID, input.AppID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Postgres) ConsumeOTPChallenge(ctx context.Context, input login.OTPConsume) (uuid.UUID, error) {
	challengeKind, err := challengeType(input.Purpose, input.IdentifierType)
	if err != nil || input.ID == uuid.Nil || input.AppID == uuid.Nil || input.NormalizedValue == "" || len(input.TargetHash) != 32 || len(input.SecretHash) != 32 ||
		!hmac.Equal(input.TargetHash, hashValue(input.NormalizedValue)) {
		return uuid.Nil, login.ErrOTPInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var storedSecret []byte
	var attempts, maxAttempts int
	var expiresAt time.Time
	err = tx.QueryRow(ctx, `SELECT secret_hash,attempts,max_attempts,expires_at
FROM iam.verification_challenges WHERE id=$1 AND challenge_type=$2 AND target_hash=$3
  AND metadata->>'app_id'=$4 AND metadata->>'identifier_type'=$5 AND metadata->>'purpose'=$6
  AND consumed_at IS NULL FOR UPDATE`, input.ID, challengeKind, input.TargetHash, input.AppID.String(), input.IdentifierType, input.Purpose).
		Scan(&storedSecret, &attempts, &maxAttempts, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, login.ErrOTPInvalid
	}
	if err != nil {
		return uuid.Nil, err
	}
	valid := attempts < maxAttempts && time.Now().UTC().Before(expiresAt) && hmac.Equal(storedSecret, input.SecretHash)
	if !valid {
		if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,
consumed_at=CASE WHEN attempts+1>=max_attempts OR expires_at<=now() THEN now() ELSE consumed_at END WHERE id=$1`, input.ID); err != nil {
			return uuid.Nil, err
		}
		if err = tx.Commit(ctx); err != nil {
			return uuid.Nil, err
		}
		return uuid.Nil, login.ErrOTPInvalid
	}
	var userID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT i.user_id FROM app.user_login_identifiers i
JOIN app.user_memberships m ON m.app_id=i.app_id AND m.user_id=i.user_id AND m.tenant_id=i.tenant_id
JOIN iam.users u ON u.id=i.user_id
WHERE i.app_id=$1 AND i.identifier_type=$2 AND i.normalized_value=$3
  AND i.status='active' AND i.verified_at IS NOT NULL AND m.status='active' AND u.status='active' AND u.deleted_at IS NULL`,
		input.AppID, input.IdentifierType, input.NormalizedValue).Scan(&userID)
	// A challenge with no bound user fails closed without revealing registration.
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, login.ErrOTPInvalid
	}
	if err != nil {
		return uuid.Nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,consumed_at=now() WHERE id=$1`, input.ID); err != nil {
		return uuid.Nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (r *Postgres) FindOTPLoginUser(ctx context.Context, appID uuid.UUID, identifierType, normalizedValue string) (uuid.UUID, uuid.UUID, string, error) {
	var userID, tenantID uuid.UUID
	var locale string
	err := r.pool.QueryRow(ctx, `SELECT i.user_id,i.tenant_id,u.locale
FROM app.user_login_identifiers i
JOIN app.user_memberships m ON m.app_id=i.app_id AND m.user_id=i.user_id AND m.tenant_id=i.tenant_id
JOIN iam.users u ON u.id=i.user_id
WHERE i.app_id=$1 AND i.identifier_type=$2 AND i.normalized_value=$3 AND i.status='active' AND i.verified_at IS NOT NULL
  AND m.status='active' AND u.status='active' AND u.deleted_at IS NULL`, appID, identifierType, normalizedValue).Scan(&userID, &tenantID, &locale)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, "", login.ErrNotFound
	}
	return userID, tenantID, locale, err
}

func scanIdentifier(row scanner) (login.Identifier, error) {
	var item login.Identifier
	var formatValid bool
	err := row.Scan(&item.ID, &item.IdentifierType, &item.DisplayHint, &item.VerifiedAt, &item.Status, &formatValid)
	if err == nil {
		item.Verified = item.VerifiedAt != nil
		item.LoginCapable = item.Verified && item.Status == "active" && formatValid
		item.CanBind = false
		item.CanChange = true
	}
	return item, err
}

func (r *Postgres) LoginMethods(ctx context.Context, appID, userID uuid.UUID) (login.LoginMethods, error) {
	var password bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.user_credentials WHERE user_id=$1)`, userID).Scan(&password); err != nil {
		return login.LoginMethods{}, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id,identifier_type,display_hint,verified_at,status,
CASE
  WHEN identifier_type='mobile' THEN normalized_value::text ~ '^\+[1-9][0-9]{7,14}$'
  WHEN identifier_type='email' THEN length(normalized_value::text)<=254
    AND normalized_value::text ~ '^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$'
  ELSE false
END AS format_valid
FROM app.user_login_identifiers WHERE app_id=$1 AND user_id=$2 ORDER BY identifier_type`, appID, userID)
	if err != nil {
		return login.LoginMethods{}, err
	}
	defer rows.Close()
	byType := map[string]login.Identifier{}
	for rows.Next() {
		item, scanErr := scanIdentifier(rows)
		if scanErr != nil {
			return login.LoginMethods{}, scanErr
		}
		byType[item.IdentifierType] = item
	}
	if err = rows.Err(); err != nil {
		return login.LoginMethods{}, err
	}
	usable, err := r.usableLoginMethodCount(ctx, r.pool, appID, userID, uuid.Nil, "")
	if err != nil {
		return login.LoginMethods{}, err
	}
	identifiers := make([]login.Identifier, 0, 2)
	for _, identifierType := range []string{"email", "mobile"} {
		item, exists := byType[identifierType]
		if !exists {
			item = login.Identifier{IdentifierType: identifierType, Status: "unbound", CanBind: true}
		} else {
			remaining := usable
			if item.LoginCapable {
				remaining--
				if identifierType == "email" && password {
					remaining--
				}
			}
			item.CanUnbind = remaining > 0
			if !item.CanUnbind {
				item.BlockReason = "last_login_method"
			}
		}
		identifiers = append(identifiers, item)
	}
	accounts, err := r.ListOAuthAccounts(ctx, appID, userID)
	if err != nil {
		return login.LoginMethods{}, err
	}
	accountByProvider := make(map[string]login.OAuthAccount, len(accounts))
	for _, account := range accounts {
		// OAuth subject replacement is intentionally not offered in v1. Users must
		// explicitly unbind first so identity conflicts cannot be hidden as a change.
		account.CanChange = false
		accountByProvider[account.ProviderCode] = account
	}
	stepUpCapable := byType["email"].LoginCapable || byType["mobile"].LoginCapable
	oauthMethods := make([]login.OAuthAccount, 0, len(login.ProviderCodes))
	for _, providerCode := range login.ProviderCodes {
		item, exists := accountByProvider[providerCode]
		if !exists {
			var enabled bool
			if err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.application_login_provider_bindings b
JOIN sys.login_provider_configs c ON c.tenant_id=b.tenant_id AND c.id=b.login_provider_config_id
WHERE b.app_id=$1 AND b.provider_code=$2 AND b.enabled AND c.status='active' AND c.last_preflight_status='ready' AND c.deleted_at IS NULL)`, appID, providerCode).Scan(&enabled); err != nil {
				return login.LoginMethods{}, err
			}
			item = login.OAuthAccount{ProviderCode: providerCode, DisplayNameKey: "login_provider." + providerCode,
				Status: "unbound", LoginEnabled: enabled, CanBind: enabled && stepUpCapable}
			if !enabled {
				item.BlockReason = "provider_disabled"
			} else if !stepUpCapable {
				item.BlockReason = "step_up_method_required"
			}
		}
		oauthMethods = append(oauthMethods, item)
		delete(accountByProvider, providerCode)
	}
	// Preserve bound identities created by a newer or removed server build.
	// They stay visible and safely removable, but cannot start a new bind/change
	// flow until the provider is registered in this binary again.
	for _, account := range accounts {
		if item, exists := accountByProvider[account.ProviderCode]; exists {
			item.DisplayNameKey = "login_provider.unknown"
			item.CanBind, item.CanChange = false, false
			oauthMethods = append(oauthMethods, item)
			delete(accountByProvider, account.ProviderCode)
		}
	}
	passwordMethod := login.PasswordMethod{Present: password, LoginCapable: password && byType["email"].LoginCapable, CanBind: !password && stepUpCapable, CanChange: password}
	return login.LoginMethods{Password: passwordMethod, Identifiers: identifiers, OAuthAccounts: oauthMethods, RemainingLoginMethods: usable}, nil
}

func (r *Postgres) SetPassword(ctx context.Context, principal login.Principal, appID uuid.UUID, passwordHash string) error {
	if principal.TenantID == uuid.Nil || principal.UserID == uuid.Nil || appID == uuid.Nil || passwordHash == "" {
		return login.ErrInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var eligible bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.user_memberships m
JOIN app.user_login_identifiers i ON i.app_id=m.app_id AND i.user_id=m.user_id
WHERE m.app_id=$1 AND m.tenant_id=$2 AND m.user_id=$3 AND m.status='active'
AND i.status='active' AND i.verified_at IS NOT NULL)`, appID, principal.TenantID, principal.UserID).Scan(&eligible); err != nil {
		return err
	}
	if !eligible {
		return login.ErrForbidden
	}
	tag, err := tx.Exec(ctx, `INSERT INTO iam.user_credentials(user_id,password_hash)
SELECT $1,$2 WHERE NOT EXISTS(SELECT 1 FROM iam.user_credentials WHERE user_id=$1)`, principal.UserID, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return login.ErrConflict
	}
	details, _ := json.Marshal(map[string]any{"app_id": appID})
	if _, err = tx.Exec(ctx, `INSERT INTO audit.security_events(tenant_id,user_id,session_id,app_id,event_type,severity,source,client_ip,user_agent,details)
VALUES($1,$2,$3,$4,'iam.password.bound','info','auth',nullif($5,'')::inet,nullif($6,''),$7)`, principal.TenantID,
		principal.UserID, principal.SessionID, appID, principal.IPAddress, principal.UserAgent, details); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Postgres) IdentifierTarget(ctx context.Context, appID, userID, identifierID uuid.UUID) (login.IdentifierTarget, error) {
	if appID == uuid.Nil || userID == uuid.Nil || identifierID == uuid.Nil {
		return login.IdentifierTarget{}, login.ErrInvalid
	}
	row, err := db.New(r.pool).GetAppLoginIdentifierTarget(ctx, db.GetAppLoginIdentifierTargetParams{ID: identifierID, AppID: appID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return login.IdentifierTarget{}, login.ErrNotFound
	}
	if err != nil {
		return login.IdentifierTarget{}, err
	}
	return login.IdentifierTarget{ID: row.ID, TenantID: row.TenantID, AppID: row.AppID, UserID: row.UserID,
		IdentifierType: row.IdentifierType, NormalizedValue: row.NormalizedValue, DisplayHint: row.DisplayHint, Locale: row.Locale}, nil
}

func (r *Postgres) UpsertIdentifier(ctx context.Context, mutation login.IdentifierMutation) (login.Identifier, error) {
	challengeKind, err := challengeType("bind", mutation.IdentifierType)
	if err != nil || mutation.AppID == uuid.Nil || mutation.UserID == uuid.Nil || mutation.ChallengeID == uuid.Nil {
		return login.Identifier{}, login.ErrInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return login.Identifier{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var stored []byte
	var attempts, maxAttempts int
	var expiresAt time.Time
	err = tx.QueryRow(ctx, `SELECT secret_hash,attempts,max_attempts,expires_at FROM iam.verification_challenges
WHERE id=$1 AND user_id=$2 AND challenge_type=$3 AND target_hash=$4 AND metadata->>'app_id'=$5
  AND metadata->>'identifier_type'=$6 AND metadata->>'purpose'='bind' AND consumed_at IS NULL FOR UPDATE`,
		mutation.ChallengeID, mutation.UserID, challengeKind, mutation.TargetHash, mutation.AppID.String(), mutation.IdentifierType).
		Scan(&stored, &attempts, &maxAttempts, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && (attempts >= maxAttempts || !time.Now().UTC().Before(expiresAt) || !hmac.Equal(stored, mutation.SecretHash)) {
		if err == nil {
			_, _ = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,
consumed_at=CASE WHEN attempts+1>=max_attempts OR expires_at<=now() THEN now() ELSE consumed_at END WHERE id=$1`, mutation.ChallengeID)
			_ = tx.Commit(ctx)
		}
		return login.Identifier{}, login.ErrOTPInvalid
	}
	if err != nil {
		return login.Identifier{}, err
	}
	var tenantID uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT tenant_id FROM app.user_memberships
WHERE app_id=$1 AND user_id=$2 AND status='active' FOR UPDATE`, mutation.AppID, mutation.UserID).Scan(&tenantID); errors.Is(err, pgx.ErrNoRows) {
		return login.Identifier{}, login.ErrForbidden
	}
	if err != nil {
		return login.Identifier{}, err
	}
	var identifierID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO app.user_login_identifiers(
tenant_id,app_id,user_id,identifier_type,normalized_value,display_hint,verified_at,status)
VALUES($1,$2,$3,$4,$5,$6,now(),'active')
ON CONFLICT(app_id,user_id,identifier_type) DO UPDATE SET normalized_value=EXCLUDED.normalized_value,
display_hint=EXCLUDED.display_hint,verified_at=now(),status='active',lock_version=app.user_login_identifiers.lock_version+1
RETURNING id`, tenantID, mutation.AppID, mutation.UserID, mutation.IdentifierType, mutation.NormalizedValue, mutation.DisplayHint).Scan(&identifierID)
	if err != nil {
		if errors.Is(classify(err), login.ErrConflict) {
			return login.Identifier{}, login.ErrIdentifierConflict
		}
		return login.Identifier{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,consumed_at=now() WHERE id=$1`, mutation.ChallengeID); err != nil {
		return login.Identifier{}, err
	}
	details, _ := json.Marshal(map[string]any{"app_id": mutation.AppID, "identifier_type": mutation.IdentifierType})
	if _, err = tx.Exec(ctx, `INSERT INTO audit.security_events(tenant_id,user_id,session_id,app_id,event_type,severity,source,client_ip,details)
VALUES($1,$2,$3,$4,'iam.identifier.bind','medium','auth',nullif($5,'')::inet,$6)`, tenantID, mutation.UserID, mutation.SessionID, mutation.AppID, mutation.IPAddress, details); err != nil {
		return login.Identifier{}, err
	}
	item, err := scanIdentifier(tx.QueryRow(ctx, `SELECT id,identifier_type,display_hint,verified_at,status,true
FROM app.user_login_identifiers WHERE id=$1`, identifierID))
	if err != nil {
		return login.Identifier{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return login.Identifier{}, classify(err)
	}
	item.CanChange = true
	return item, nil
}

func (r *Postgres) DeleteIdentifier(ctx context.Context, principal login.Principal, appID uuid.UUID, identifierType string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM app.user_login_identifiers
WHERE app_id=$1 AND user_id=$2 AND identifier_type=$3 FOR UPDATE`, appID, principal.UserID, identifierType).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return login.ErrNotFound
	}
	if err != nil {
		return err
	}
	remaining, err := r.usableLoginMethodCount(ctx, tx, appID, principal.UserID, uuid.Nil, identifierType)
	if err != nil {
		return err
	}
	if remaining < 1 {
		return login.ErrLastLoginMethod
	}
	if _, err = tx.Exec(ctx, `DELETE FROM app.user_login_identifiers WHERE id=$1`, id); err != nil {
		return err
	}
	details, _ := json.Marshal(map[string]any{"app_id": appID, "identifier_type": identifierType})
	if _, err = tx.Exec(ctx, `INSERT INTO audit.security_events(tenant_id,user_id,session_id,app_id,event_type,severity,source,client_ip,details)
VALUES($1,$2,$3,$4,'iam.identifier.unbind','medium','auth',nullif($5,'')::inet,$6)`, principal.TenantID, principal.UserID, principal.SessionID, appID, principal.IPAddress, details); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func hashValue(value string) []byte {
	digest := sha256Sum(value)
	return digest[:]
}

func sha256Sum(value string) [32]byte {
	// Kept local so raw identifiers never enter generic logging helpers.
	return sha256.Sum256([]byte(value))
}

func (r *Postgres) CreateStepUpTicket(ctx context.Context, ticket login.StepUpTicket) error {
	if ticket.ID == uuid.Nil || ticket.Principal.TenantID == uuid.Nil || ticket.Principal.UserID == uuid.Nil ||
		ticket.Principal.SessionID == uuid.Nil || ticket.AppID == uuid.Nil || ticket.Purpose == "" || ticket.Resource == "" ||
		len(ticket.TokenHash) != 32 || !ticket.ExpiresAt.After(time.Now().UTC()) {
		return login.ErrStepUpInvalid
	}
	metadata, _ := json.Marshal(map[string]string{
		"app_id": ticket.AppID.String(), "tenant_id": ticket.Principal.TenantID.String(),
		"session_id": ticket.Principal.SessionID.String(), "purpose": ticket.Purpose,
		"resource": ticket.Resource, "method": ticket.Method,
	})
	_, err := r.pool.Exec(ctx, `INSERT INTO iam.verification_challenges(
id,user_id,challenge_type,target_hash,target_hint,secret_hash,max_attempts,expires_at,created_ip,metadata)
VALUES($1,$2,'step_up',$3,'step-up',$4,1,$5,nullif($6,'')::inet,$7)`, ticket.ID, ticket.Principal.UserID,
		hashValue(ticket.Resource), ticket.TokenHash, ticket.ExpiresAt, ticket.Principal.IPAddress, metadata)
	if err != nil {
		return classify(err)
	}
	return nil
}

func (r *Postgres) ConsumeStepUpTicket(ctx context.Context, input login.StepUpConsume) error {
	if input.ID == uuid.Nil || input.UserID == uuid.Nil || input.SessionID == uuid.Nil || input.AppID == uuid.Nil ||
		input.Purpose == "" || input.Resource == "" || len(input.TokenHash) != 32 {
		return login.ErrStepUpInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var stored []byte
	var method string
	var expires time.Time
	err = tx.QueryRow(ctx, `SELECT secret_hash,metadata->>'method',expires_at
FROM iam.verification_challenges WHERE id=$1 AND user_id=$2 AND challenge_type='step_up'
  AND target_hash=$3 AND metadata->>'app_id'=$4 AND metadata->>'session_id'=$5
  AND metadata->>'purpose'=$6 AND metadata->>'resource'=$7 AND consumed_at IS NULL FOR UPDATE`,
		input.ID, input.UserID, hashValue(input.Resource), input.AppID.String(), input.SessionID.String(), input.Purpose, input.Resource).
		Scan(&stored, &method, &expires)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && (!time.Now().UTC().Before(expires) || !hmac.Equal(stored, input.TokenHash) || input.RequiredMethod != "" && method != input.RequiredMethod) {
		return login.ErrStepUpInvalid
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=1,consumed_at=now() WHERE id=$1`, input.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
