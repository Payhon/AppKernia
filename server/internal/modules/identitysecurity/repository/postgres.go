package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	identity "github.com/appkernia/appkernia/server/internal/modules/identitysecurity/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (repo *Postgres) MFAStatus(ctx context.Context, userID uuid.UUID) (identity.MFAStatus, error) {
	var status identity.MFAStatus
	err := repo.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM iam.mfa_factors WHERE user_id=$1 AND factor_type='totp' AND status='active'),
		       (SELECT verified_at FROM iam.mfa_factors WHERE user_id=$1 AND factor_type='totp' AND status='active' ORDER BY verified_at DESC LIMIT 1),
		       (SELECT count(*) FROM iam.mfa_recovery_codes WHERE user_id=$1 AND used_at IS NULL)`, userID).
		Scan(&status.TOTPEnabled, &status.TOTPVerifiedAt, &status.RecoveryCodesRemaining)
	return status, err
}

func (repo *Postgres) ReplacePendingTOTP(ctx context.Context, principal identity.Principal, ciphertext []byte) error {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var active bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.mfa_factors WHERE user_id=$1 AND factor_type='totp' AND status='active')`, principal.UserID).Scan(&active); err != nil {
		return err
	}
	if active {
		return identity.ErrConflict
	}
	if _, err = tx.Exec(ctx, `DELETE FROM iam.mfa_factors WHERE user_id=$1 AND factor_type='totp' AND status='pending'`, principal.UserID); err != nil {
		return err
	}
	var id uuid.UUID
	if err = tx.QueryRow(ctx, `INSERT INTO iam.mfa_factors(user_id,factor_type,display_name,secret_encrypted,status) VALUES($1,'totp','Authenticator app',$2,'pending') RETURNING id`, principal.UserID, ciphertext).Scan(&id); err != nil {
		return err
	}
	if err = writeAudit(ctx, tx, principal, "iam.mfa.totp.enroll", "mfa_factor", id.String(), "POST", "/admin-api/v1/me/mfa/totp/enroll", map[string]any{"factor_type": "totp", "status": "pending"}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repo *Postgres) PendingTOTP(ctx context.Context, userID uuid.UUID) (identity.TOTPFactor, error) {
	var factor identity.TOTPFactor
	err := repo.pool.QueryRow(ctx, `SELECT id,secret_encrypted,status FROM iam.mfa_factors WHERE user_id=$1 AND factor_type='totp' AND status='pending' AND created_at>now()-interval '10 minutes' ORDER BY created_at DESC LIMIT 1`, userID).Scan(&factor.ID, &factor.Ciphertext, &factor.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		err = identity.ErrNotFound
	}
	return factor, err
}

func (repo *Postgres) ActiveTOTP(ctx context.Context, userID uuid.UUID) (identity.TOTPFactor, error) {
	var factor identity.TOTPFactor
	err := repo.pool.QueryRow(ctx, `SELECT id,secret_encrypted,status FROM iam.mfa_factors WHERE user_id=$1 AND factor_type='totp' AND status='active' ORDER BY verified_at DESC LIMIT 1`, userID).Scan(&factor.ID, &factor.Ciphertext, &factor.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		err = identity.ErrNotFound
	}
	return factor, err
}

func (repo *Postgres) ActivateTOTP(ctx context.Context, principal identity.Principal, factorID uuid.UUID, hashes [][]byte) error {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE iam.mfa_factors SET status='active',verified_at=now() WHERE id=$1 AND user_id=$2 AND factor_type='totp' AND status='pending' AND created_at>now()-interval '10 minutes'`, factorID, principal.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return identity.ErrConflict
	}
	if _, err = tx.Exec(ctx, `DELETE FROM iam.mfa_recovery_codes WHERE user_id=$1`, principal.UserID); err != nil {
		return err
	}
	for _, hash := range hashes {
		if _, err = tx.Exec(ctx, `INSERT INTO iam.mfa_recovery_codes(user_id,code_hash) VALUES($1,$2)`, principal.UserID, hash); err != nil {
			return err
		}
	}
	if err = writeAudit(ctx, tx, principal, "iam.mfa.totp.enable", "mfa_factor", factorID.String(), "POST", "/admin-api/v1/me/mfa/totp/verify", map[string]any{"factor_type": "totp", "status": "active", "recovery_code_count": len(hashes)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repo *Postgres) DisableTOTP(ctx context.Context, principal identity.Principal) error {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var factorID uuid.UUID
	if err = tx.QueryRow(ctx, `UPDATE iam.mfa_factors SET status='disabled' WHERE user_id=$1 AND factor_type='totp' AND status='active' RETURNING id`, principal.UserID).Scan(&factorID); errors.Is(err, pgx.ErrNoRows) {
		return identity.ErrNotFound
	} else if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM iam.mfa_recovery_codes WHERE user_id=$1`, principal.UserID); err != nil {
		return err
	}
	if err = writeAudit(ctx, tx, principal, "iam.mfa.totp.disable", "mfa_factor", factorID.String(), "DELETE", "/admin-api/v1/me/mfa/totp", map[string]any{"factor_type": "totp", "status": "disabled"}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repo *Postgres) RotateRecoveryCodes(ctx context.Context, principal identity.Principal, hashes [][]byte) error {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var active bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.mfa_factors WHERE user_id=$1 AND factor_type='totp' AND status='active')`, principal.UserID).Scan(&active); err != nil {
		return err
	}
	if !active {
		return identity.ErrConflict
	}
	if _, err = tx.Exec(ctx, `DELETE FROM iam.mfa_recovery_codes WHERE user_id=$1`, principal.UserID); err != nil {
		return err
	}
	for _, hash := range hashes {
		if _, err = tx.Exec(ctx, `INSERT INTO iam.mfa_recovery_codes(user_id,code_hash) VALUES($1,$2)`, principal.UserID, hash); err != nil {
			return err
		}
	}
	if err = writeAudit(ctx, tx, principal, "iam.mfa.recovery.rotate", "mfa_recovery_codes", principal.UserID.String(), "POST", "/admin-api/v1/me/mfa/recovery-codes/rotate", map[string]any{"rotated": true, "recovery_code_count": len(hashes)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repo *Postgres) PasswordHash(ctx context.Context, userID uuid.UUID) (string, error) {
	var hash string
	err := repo.pool.QueryRow(ctx, `SELECT password_hash FROM iam.user_credentials WHERE user_id=$1`, userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		err = identity.ErrNotFound
	}
	return hash, err
}

func (repo *Postgres) ListOAuth(ctx context.Context, userID uuid.UUID) ([]identity.OAuthAccount, error) {
	rows, err := repo.pool.Query(ctx, `SELECT id,provider,COALESCE(provider_username,''),status,created_at FROM iam.oauth_accounts WHERE user_id=$1 ORDER BY provider,id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := []identity.OAuthAccount{}
	for rows.Next() {
		var account identity.OAuthAccount
		if err = rows.Scan(&account.ID, &account.Provider, &account.AccountHint, &account.Status, &account.BoundAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (repo *Postgres) SaveOAuthChallenge(ctx context.Context, principal identity.Principal, challenge identity.OAuthChallenge) error {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `UPDATE iam.oauth_binding_challenges SET consumed_at=now() WHERE user_id=$1 AND provider=$2 AND consumed_at IS NULL`, principal.UserID, challenge.Provider); err != nil {
		return err
	}
	var id uuid.UUID
	if err = tx.QueryRow(ctx, `INSERT INTO iam.oauth_binding_challenges(user_id,tenant_id,provider,state_hash,authorization_code_hash,pkce_verifier_encrypted,pkce_challenge,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, principal.UserID, principal.TenantID, challenge.Provider, challenge.StateHash, challenge.CodeHash, challenge.PKCEVerifierEncrypted, challenge.PKCEChallenge, challenge.ExpiresAt).Scan(&id); err != nil {
		return err
	}
	if err = writeAudit(ctx, tx, principal, "iam.oauth.bind.start", "oauth_binding_challenge", id.String(), "POST", "/admin-api/v1/me/oauth/"+challenge.Provider+"/start", map[string]any{"provider": challenge.Provider, "expires_at": challenge.ExpiresAt, "pkce_method": "S256"}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repo *Postgres) CompleteOAuth(ctx context.Context, principal identity.Principal, challenge identity.OAuthChallenge, oauth identity.OAuthIdentity) (identity.OAuthAccount, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return identity.OAuthAccount{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var challengeID uuid.UUID
	err = tx.QueryRow(ctx, `UPDATE iam.oauth_binding_challenges SET consumed_at=now() WHERE user_id=$1 AND tenant_id=$2 AND provider=$3 AND state_hash=$4 AND authorization_code_hash=$5 AND consumed_at IS NULL AND expires_at>$6 RETURNING id`, principal.UserID, principal.TenantID, oauth.Provider, challenge.StateHash, challenge.CodeHash, challenge.ExpiresAt).Scan(&challengeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.OAuthAccount{}, identity.ErrOAuthState
	}
	if err != nil {
		return identity.OAuthAccount{}, err
	}
	var account identity.OAuthAccount
	err = tx.QueryRow(ctx, `INSERT INTO iam.oauth_accounts(user_id,provider,subject,provider_username,provider_profile,status) VALUES($1,$2,$3,$4,'{}','active') ON CONFLICT (user_id,provider) DO UPDATE SET subject=EXCLUDED.subject,provider_username=EXCLUDED.provider_username,status='active' RETURNING id,provider,COALESCE(provider_username,''),status,created_at`, principal.UserID, oauth.Provider, oauth.Subject, oauth.AccountHint).Scan(&account.ID, &account.Provider, &account.AccountHint, &account.Status, &account.BoundAt)
	if err != nil {
		return identity.OAuthAccount{}, err
	}
	if err = writeAudit(ctx, tx, principal, "iam.oauth.bind.complete", "oauth_account", account.ID.String(), "POST", "/admin-api/v1/me/oauth/"+oauth.Provider+"/callback", map[string]any{"provider": oauth.Provider, "account_hint": oauth.AccountHint, "status": "active"}); err != nil {
		return identity.OAuthAccount{}, err
	}
	return account, tx.Commit(ctx)
}

func (repo *Postgres) DeleteOAuth(ctx context.Context, principal identity.Principal, provider string) error {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var alternate bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.user_credentials WHERE user_id=$1 AND password_hash IS NOT NULL) OR EXISTS(SELECT 1 FROM iam.mfa_factors WHERE user_id=$1 AND status='active')`, principal.UserID).Scan(&alternate); err != nil {
		return err
	}
	if !alternate {
		return identity.ErrConflict
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `DELETE FROM iam.oauth_accounts WHERE user_id=$1 AND provider=$2 RETURNING id`, principal.UserID, provider).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.ErrNotFound
	}
	if err != nil {
		return err
	}
	if err = writeAudit(ctx, tx, principal, "iam.oauth.unbind", "oauth_account", id.String(), "DELETE", "/admin-api/v1/me/oauth/"+provider, map[string]any{"provider": provider, "unbound": true}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func writeAudit(ctx context.Context, tx pgx.Tx, principal identity.Principal, action, resourceType, resourceID, method, path string, after any) error {
	raw, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,after_data,succeeded) VALUES($1,$2,NULLIF($3,'00000000-0000-0000-0000-000000000000'::uuid),$4,'iam',$5,$5,$6,$7,$8,$9,200,NULLIF($10,'')::inet,NULLIF($11,''),$12,true)`, principal.TenantID, principal.UserID, principal.SessionID, principal.RequestID, action, resourceType, resourceID, method, path, strings.TrimSpace(principal.IPAddress), strings.TrimSpace(principal.UserAgent), raw)
	return err
}
