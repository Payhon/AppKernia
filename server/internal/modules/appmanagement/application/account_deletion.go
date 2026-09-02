package application

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrAccountDeletionEmailUnverified = errors.New("account deletion email is not verified")
	ErrAccountDeletionCodeCooldown    = errors.New("account deletion verification code is cooling down")
	ErrAccountDeletionCodeInvalid     = errors.New("account deletion verification code is invalid")
	ErrAccountDeletionCodeExpired     = errors.New("account deletion verification code is expired")
	ErrAccountDeletionCodeExhausted   = errors.New("account deletion verification attempts are exhausted")
	ErrAccountDeletionUnavailable     = errors.New("account deletion is unavailable")
	ErrAccountDeletionDisabled        = errors.New("account deletion is disabled")
	ErrAccountDeletionStepUpRequired  = errors.New("account deletion step-up is required")
)

type AccountDeletionCode struct {
	Accepted             bool       `json:"accepted"`
	TargetHint           string     `json:"target_hint"`
	ExpiresInSeconds     int        `json:"expires_in_seconds"`
	RetryAfterSeconds    int        `json:"retry_after_seconds"`
	VerificationRequired bool       `json:"verification_required"`
	ReauthRequired       bool       `json:"reauth_required"`
	ReauthProvider       string     `json:"reauth_provider,omitempty"`
	ReauthAccountID      *uuid.UUID `json:"reauth_account_id,omitempty"`
}

type AccountDeletionResult struct {
	Deleted bool `json:"deleted"`
}

type erasureObject struct {
	FileID   uuid.UUID
	Provider string
	Bucket   string
	Key      string
}

func (s *Service) RequestAccountDeletionCode(ctx context.Context, token string, app Application) (AccountDeletionCode, error) {
	if !s.accountDeletionEnabled {
		return AccountDeletionCode{}, ErrAccountDeletionDisabled
	}
	principal, err := s.AuthenticateMobileMembership(ctx, token, app)
	if err != nil {
		return AccountDeletionCode{}, ErrMembershipMissing
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return AccountDeletionCode{}, fmt.Errorf("begin account deletion verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text || ':' || $2::text,0))`, app.ID, principal.User.ID); err != nil {
		return AccountDeletionCode{}, fmt.Errorf("lock account deletion requirement scope: %w", err)
	}
	var membershipUserID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT m.user_id FROM app.user_memberships m JOIN iam.users u ON u.id=m.user_id
		WHERE m.app_id=$1 AND m.tenant_id=$2 AND m.user_id=$3 AND m.status='active' AND u.deleted_at IS NULL
		FOR UPDATE OF u,m`, app.ID, app.TenantID, principal.User.ID).Scan(&membershipUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountDeletionCode{}, ErrMembershipMissing
	}
	if err != nil {
		return AccountDeletionCode{}, fmt.Errorf("load account deletion identity: %w", err)
	}
	var email, hint string
	var verified bool
	err = tx.QueryRow(ctx, `SELECT normalized_value::text,display_hint,verified_at IS NOT NULL
		FROM app.user_login_identifiers WHERE app_id=$1 AND user_id=$2 AND identifier_type='email' AND status='active'
		FOR UPDATE`, app.ID, principal.User.ID).Scan(&email, &hint, &verified)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AccountDeletionCode{}, fmt.Errorf("load account deletion email identifier: %w", err)
	}
	result := AccountDeletionCode{Accepted: true, TargetHint: hint, VerificationRequired: verified && strings.TrimSpace(email) != ""}
	var appleAccountID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM iam.app_oauth_accounts
		WHERE app_id=$1 AND user_id=$2 AND provider_code='apple' AND status='active' ORDER BY id LIMIT 1`, app.ID, principal.User.ID).Scan(&appleAccountID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AccountDeletionCode{}, fmt.Errorf("load Apple account deletion binding: %w", err)
	}
	if appleAccountID != uuid.Nil {
		result.ReauthRequired, result.ReauthProvider, result.ReauthAccountID = true, "apple", &appleAccountID
	}
	if !result.VerificationRequired {
		if err = tx.Commit(ctx); err != nil {
			return AccountDeletionCode{}, fmt.Errorf("commit account deletion requirements: %w", err)
		}
		return result, nil
	}
	result.ExpiresInSeconds = int(emailOTPLifetime.Seconds())
	result.RetryAfterSeconds = int(emailOTPCooldown.Seconds())
	var latest time.Time
	err = tx.QueryRow(ctx, `SELECT created_at FROM iam.verification_challenges
		WHERE user_id=$1 AND challenge_type='account_delete' AND metadata->>'app_id'=$2
		ORDER BY created_at DESC LIMIT 1`, principal.User.ID, app.ID.String()).Scan(&latest)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AccountDeletionCode{}, fmt.Errorf("load account deletion cooldown: %w", err)
	}
	if !latest.IsZero() && time.Since(latest) < emailOTPCooldown {
		result.RetryAfterSeconds = int((emailOTPCooldown - time.Since(latest)).Seconds()) + 1
		return result, ErrAccountDeletionCodeCooldown
	}
	if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET consumed_at=COALESCE(consumed_at,now())
		WHERE user_id=$1 AND challenge_type='account_delete' AND metadata->>'app_id'=$2 AND consumed_at IS NULL`, principal.User.ID, app.ID.String()); err != nil {
		return AccountDeletionCode{}, fmt.Errorf("replace account deletion challenge: %w", err)
	}
	if err = s.createOTP(ctx, tx, app, principal.User.ID, email, "account_delete"); err != nil {
		return AccountDeletionCode{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AccountDeletionCode{}, fmt.Errorf("commit account deletion verification: %w", err)
	}
	return result, nil
}

func (s *Service) ConfirmAccountDeletion(ctx context.Context, token string, app Application, verificationCode, stepUpToken string, acknowledged bool) (AccountDeletionResult, error) {
	if !s.accountDeletionEnabled {
		return AccountDeletionResult{}, ErrAccountDeletionDisabled
	}
	if !acknowledged {
		return AccountDeletionResult{}, ErrAccountDeletionCodeInvalid
	}
	principal, err := s.AuthenticateMobileMembership(ctx, token, app)
	if err != nil {
		return AccountDeletionResult{}, ErrMembershipMissing
	}
	var verifiedAppleAccountID uuid.UUID
	if s.accountDeletionStepUp != nil {
		verifiedAppleAccountID, err = s.accountDeletionStepUp(ctx, token, app.ID, stepUpToken)
		if err != nil {
			return AccountDeletionResult{}, ErrAccountDeletionStepUpRequired
		}
	} else {
		var appleBound bool
		if err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM iam.app_oauth_accounts
WHERE app_id=$1 AND user_id=$2 AND provider_code='apple' AND status='active')`, app.ID, principal.User.ID).Scan(&appleBound); err != nil || appleBound {
			return AccountDeletionResult{}, ErrAccountDeletionUnavailable
		}
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return AccountDeletionResult{}, fmt.Errorf("begin account deletion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text || ':' || $2::text,0))`, app.ID, principal.User.ID); err != nil {
		return AccountDeletionResult{}, fmt.Errorf("lock account deletion identity scope: %w", err)
	}
	var membershipUserID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT m.user_id FROM app.user_memberships m JOIN iam.users u ON u.id=m.user_id
		WHERE m.app_id=$1 AND m.tenant_id=$2 AND m.user_id=$3 AND m.status='active' AND u.deleted_at IS NULL
		FOR UPDATE OF u,m`, app.ID, app.TenantID, principal.User.ID).Scan(&membershipUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountDeletionResult{}, ErrMembershipMissing
	}
	if err != nil {
		return AccountDeletionResult{}, fmt.Errorf("lock account deletion identity: %w", err)
	}
	var currentAppleAccountID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM iam.app_oauth_accounts
WHERE app_id=$1 AND user_id=$2 AND provider_code='apple' AND status='active'
ORDER BY id LIMIT 1 FOR UPDATE`, app.ID, principal.User.ID).Scan(&currentAppleAccountID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AccountDeletionResult{}, fmt.Errorf("lock Apple account deletion binding: %w", err)
	}
	if currentAppleAccountID != uuid.Nil && currentAppleAccountID != verifiedAppleAccountID {
		return AccountDeletionResult{}, ErrAccountDeletionUnavailable
	}
	var email string
	var verified bool
	err = tx.QueryRow(ctx, `SELECT normalized_value::text,verified_at IS NOT NULL FROM app.user_login_identifiers
		WHERE app_id=$1 AND user_id=$2 AND identifier_type='email' AND status='active' FOR UPDATE`, app.ID, principal.User.ID).Scan(&email, &verified)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AccountDeletionResult{}, fmt.Errorf("lock account deletion email identifier: %w", err)
	}
	if verified && strings.TrimSpace(email) != "" && !validOTP(verificationCode) {
		return AccountDeletionResult{}, ErrAccountDeletionCodeInvalid
	}
	var challengeID uuid.UUID
	var secret []byte
	var attempts, maxAttempts int
	var expiresAt time.Time
	if verified && strings.TrimSpace(email) != "" {
		err = tx.QueryRow(ctx, `SELECT id,secret_hash,attempts,max_attempts,expires_at
		FROM iam.verification_challenges
		WHERE user_id=$1 AND challenge_type='account_delete' AND target_hash=$2 AND metadata->>'app_id'=$3 AND consumed_at IS NULL
		ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, principal.User.ID, sha256Bytes(email), app.ID.String()).
			Scan(&challengeID, &secret, &attempts, &maxAttempts, &expiresAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return AccountDeletionResult{}, ErrAccountDeletionCodeInvalid
		}
		if err != nil {
			return AccountDeletionResult{}, fmt.Errorf("load account deletion challenge: %w", err)
		}
		if attempts >= maxAttempts {
			_ = tx.Commit(ctx)
			return AccountDeletionResult{}, ErrAccountDeletionCodeExhausted
		}
		if !time.Now().UTC().Before(expiresAt) {
			_, _ = tx.Exec(ctx, `UPDATE iam.verification_challenges SET consumed_at=now() WHERE id=$1`, challengeID)
			_ = tx.Commit(ctx)
			return AccountDeletionResult{}, ErrAccountDeletionCodeExpired
		}
		if !hmac.Equal(secret, sha256Bytes(verificationCode)) {
			attempts++
			_, updateErr := tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=$2,consumed_at=CASE WHEN $2>=max_attempts THEN now() ELSE NULL END WHERE id=$1`, challengeID, attempts)
			if updateErr != nil {
				return AccountDeletionResult{}, fmt.Errorf("record account deletion verification failure: %w", updateErr)
			}
			if err = tx.Commit(ctx); err != nil {
				return AccountDeletionResult{}, fmt.Errorf("commit account deletion verification failure: %w", err)
			}
			if attempts >= maxAttempts {
				return AccountDeletionResult{}, ErrAccountDeletionCodeExhausted
			}
			return AccountDeletionResult{}, ErrAccountDeletionCodeInvalid
		}
		if _, err = tx.Exec(ctx, `UPDATE iam.verification_challenges SET attempts=attempts+1,consumed_at=now() WHERE id=$1`, challengeID); err != nil {
			return AccountDeletionResult{}, fmt.Errorf("consume account deletion challenge: %w", err)
		}
	}
	if err = s.eraseCurrentAppAccount(ctx, tx, app, principal.User.ID, email); err != nil {
		return AccountDeletionResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AccountDeletionResult{}, fmt.Errorf("commit account deletion: %w", err)
	}
	return AccountDeletionResult{Deleted: true}, nil
}

func (s *Service) eraseCurrentAppAccount(ctx context.Context, tx pgx.Tx, app Application, userID uuid.UUID, email string) error {
	counts := map[string]int64{}
	exec := func(name, query string, args ...any) error {
		tag, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("erase %s: %w", name, err)
		}
		counts[name] += tag.RowsAffected()
		return nil
	}

	if err := exec("legal_consents", `UPDATE app.legal_consents SET user_id=NULL,ip_address=NULL,user_agent=NULL,anonymized_at=now() WHERE app_id=$1 AND user_id=$2`, app.ID, userID); err != nil {
		return err
	}
	if err := exec("operation_audit", `UPDATE audit.operation_logs SET user_id=NULL,session_id=NULL,request_id='erased',trace_id=NULL,resource_id=NULL,client_ip=NULL,user_agent=NULL,request_summary=NULL,before_data=NULL,after_data=NULL,error_message=NULL
		WHERE app_id=$1 AND tenant_id=$2 AND user_id=$3`, app.ID, app.TenantID, userID); err != nil {
		return err
	}
	if err := exec("login_audit", `UPDATE audit.login_events SET user_id=NULL,session_id=NULL,request_id=NULL,login_identifier_hash=NULL,login_identifier_hint=NULL,failure_reason=NULL,client_ip=NULL,user_agent=NULL,device_info='{}'::jsonb,geo_info='{}'::jsonb
		WHERE app_id=$1 AND user_id=$2`, app.ID, userID); err != nil {
		return err
	}
	if err := exec("security_audit", `UPDATE audit.security_events SET user_id=NULL,session_id=NULL,client_ip=NULL,details='{}'::jsonb,resolved_by=NULL
		WHERE app_id=$1 AND user_id=$2`, app.ID, userID); err != nil {
		return err
	}
	if err := exec("notification_deliveries", `DELETE FROM notify.deliveries WHERE app_id=$1 AND (user_id=$2 OR target_hash=$3)`, app.ID, userID, sha256Bytes(strings.ToLower(strings.TrimSpace(email)))); err != nil {
		return err
	}
	statements := []struct {
		name  string
		query string
		args  []any
	}{
		{"feedback_usages", `DELETE FROM storage.file_usages WHERE module_code='app' AND entity_type='app.feedback' AND entity_id IN (SELECT id FROM app.feedbacks WHERE app_id=$1 AND user_id=$2)`, []any{app.ID, userID}},
		{"feedbacks", `DELETE FROM app.feedbacks WHERE app_id=$1 AND user_id=$2`, []any{app.ID, userID}},
		{"notification_recipients", `DELETE FROM notify.recipients WHERE app_id=$1 AND user_id=$2`, []any{app.ID, userID}},
		{"push_devices", `DELETE FROM notify.push_devices WHERE app_id=$1 AND user_id=$2`, []any{app.ID, userID}},
		{"comment_reports", `DELETE FROM content.comment_reports WHERE app_id=$1 AND reporter_id=$2`, []any{app.ID, userID}},
		{"comments", `DELETE FROM content.comments WHERE app_id=$1 AND author_id=$2`, []any{app.ID, userID}},
		{"bookmarks", `DELETE FROM content.article_bookmarks WHERE app_id=$1 AND user_id=$2`, []any{app.ID, userID}},
		{"blocked_users", `DELETE FROM content.blocked_users WHERE app_id=$1 AND (blocker_id=$2 OR blocked_id=$2)`, []any{app.ID, userID}},
		{"preferences", `DELETE FROM iam.user_preferences WHERE app_id=$1 AND user_id=$2`, []any{app.ID, userID}},
		{"sessions", `DELETE FROM iam.sessions WHERE app_id=$1 AND user_id=$2 AND audience='ak-mobile'`, []any{app.ID, userID}},
		{"verification_challenges", `DELETE FROM iam.verification_challenges WHERE user_id=$1 AND metadata->>'app_id'=$2`, []any{userID, app.ID.String()}},
		{"oauth_accounts", `DELETE FROM iam.app_oauth_accounts WHERE app_id=$1 AND user_id=$2`, []any{app.ID, userID}},
		{"login_identifiers", `DELETE FROM app.user_login_identifiers WHERE app_id=$1 AND user_id=$2`, []any{app.ID, userID}},
		{"app_membership", `DELETE FROM app.user_memberships WHERE app_id=$1 AND tenant_id=$2 AND user_id=$3`, []any{app.ID, app.TenantID, userID}},
	}
	for _, statement := range statements {
		if err := exec(statement.name, statement.query, statement.args...); err != nil {
			return err
		}
	}

	globalDelete, err := canDeleteGlobalIdentity(ctx, tx, userID)
	if err != nil {
		return err
	}
	if globalDelete {
		if err = s.eraseRemainingGlobalData(ctx, tx, userID, email, counts); err != nil {
			return err
		}
	}
	objects, err := collectErasureObjects(ctx, tx, app, userID, globalDelete)
	if err != nil {
		return err
	}
	if err = scrubAndRemoveStorageMetadata(ctx, tx, app, userID, globalDelete, objects, counts); err != nil {
		return err
	}
	if globalDelete {
		if err = exec("tenant_memberships", `DELETE FROM iam.tenant_members WHERE user_id=$1`, userID); err != nil {
			return err
		}
		if err = exec("global_identity", `DELETE FROM iam.users WHERE id=$1`, userID); err != nil {
			return err
		}
	}

	countsJSON, err := json.Marshal(counts)
	if err != nil {
		return fmt.Errorf("encode account erasure counts: %w", err)
	}
	status := "completed"
	var completedAt any = time.Now().UTC()
	if len(objects) > 0 {
		status, completedAt = "pending_objects", nil
		if s.erasureQueue == nil {
			return ErrAccountDeletionUnavailable
		}
	}
	var eventID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO audit.privacy_erasure_events(tenant_id,app_id,status,global_identity_deleted,erased_counts,completed_at)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, app.TenantID, app.ID, status, globalDelete, countsJSON, completedAt).Scan(&eventID)
	if err != nil {
		return fmt.Errorf("record privacy erasure event: %w", err)
	}
	enqueued := 0
	for _, object := range objects {
		var objectID uuid.UUID
		err = tx.QueryRow(ctx, `INSERT INTO audit.privacy_erasure_objects(event_id,tenant_id,provider,bucket_name,object_key)
			VALUES($1,$2,$3,$4,$5) ON CONFLICT(provider,bucket_name,object_key) DO NOTHING RETURNING id`, eventID, app.TenantID, object.Provider, object.Bucket, object.Key).Scan(&objectID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return fmt.Errorf("record privacy erasure object: %w", err)
		}
		if err = s.erasureQueue.Enqueue(ctx, tx, app.TenantID, app.ID, eventID, objectID); err != nil {
			return err
		}
		enqueued++
	}
	if len(objects) > 0 && enqueued == 0 {
		if _, err = tx.Exec(ctx, `UPDATE audit.privacy_erasure_events SET status='completed',completed_at=now() WHERE id=$1`, eventID); err != nil {
			return fmt.Errorf("complete deduplicated privacy erasure event: %w", err)
		}
	}
	return nil
}

func canDeleteGlobalIdentity(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (bool, error) {
	var retained bool
	err := tx.QueryRow(ctx, `SELECT
		EXISTS(SELECT 1 FROM app.user_memberships WHERE user_id=$1)
		OR EXISTS(SELECT 1 FROM iam.sessions WHERE user_id=$1 AND audience<>'ak-mobile')
		OR EXISTS(SELECT 1 FROM iam.user_roles WHERE user_id=$1)
		OR EXISTS(SELECT 1 FROM org.user_units WHERE user_id=$1)
		OR EXISTS(SELECT 1 FROM org.user_positions WHERE user_id=$1)
		OR EXISTS(SELECT 1 FROM app.application_team_members WHERE user_id=$1)
		OR EXISTS(SELECT 1 FROM app.applications WHERE owner_user_id=$1 AND deleted_at IS NULL)
		OR EXISTS(SELECT 1 FROM notify.messages WHERE sender_user_id=$1)
		OR EXISTS(SELECT 1 FROM org.units WHERE leader_user_id=$1 AND deleted_at IS NULL)
		OR EXISTS(SELECT 1 FROM billing.payment_orders WHERE user_id=$1)
		OR EXISTS(SELECT 1 FROM billing.wallet_accounts WHERE user_id=$1)`, userID).Scan(&retained)
	if err != nil {
		return false, fmt.Errorf("evaluate shared account identity: %w", err)
	}
	return !retained, nil
}

func (s *Service) eraseRemainingGlobalData(ctx context.Context, tx pgx.Tx, userID uuid.UUID, email string, counts map[string]int64) error {
	exec := func(name, query string, args ...any) error {
		tag, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("erase global %s: %w", name, err)
		}
		counts[name] += tag.RowsAffected()
		return nil
	}
	statements := []struct {
		name  string
		query string
		args  []any
	}{
		{"legal_consents", `UPDATE app.legal_consents SET user_id=NULL,ip_address=NULL,user_agent=NULL,anonymized_at=now() WHERE user_id=$1`, []any{userID}},
		{"operation_audit", `UPDATE audit.operation_logs SET user_id=NULL,session_id=NULL,request_id='erased',trace_id=NULL,resource_id=NULL,client_ip=NULL,user_agent=NULL,request_summary=NULL,before_data=NULL,after_data=NULL,error_message=NULL WHERE user_id=$1`, []any{userID}},
		{"login_audit", `UPDATE audit.login_events SET user_id=NULL,session_id=NULL,request_id=NULL,login_identifier_hash=NULL,login_identifier_hint=NULL,failure_reason=NULL,client_ip=NULL,user_agent=NULL,device_info='{}'::jsonb,geo_info='{}'::jsonb WHERE user_id=$1`, []any{userID}},
		{"security_audit", `UPDATE audit.security_events SET user_id=NULL,session_id=NULL,client_ip=NULL,details='{}'::jsonb,resolved_by=NULL WHERE user_id=$1`, []any{userID}},
		{"notification_deliveries", `DELETE FROM notify.deliveries WHERE user_id=$1 OR target_hash=$2`, []any{userID, sha256Bytes(strings.ToLower(strings.TrimSpace(email)))}},
		{"notification_recipients", `DELETE FROM notify.recipients WHERE user_id=$1`, []any{userID}},
		{"push_devices", `DELETE FROM notify.push_devices WHERE user_id=$1`, []any{userID}},
		{"comment_reports", `DELETE FROM content.comment_reports WHERE reporter_id=$1`, []any{userID}},
		{"comments", `DELETE FROM content.comments WHERE author_id=$1`, []any{userID}},
		{"bookmarks", `DELETE FROM content.article_bookmarks WHERE user_id=$1`, []any{userID}},
		{"blocked_users", `DELETE FROM content.blocked_users WHERE blocker_id=$1 OR blocked_id=$1`, []any{userID}},
		{"preferences", `DELETE FROM iam.user_preferences WHERE user_id=$1`, []any{userID}},
		{"sessions", `DELETE FROM iam.sessions WHERE user_id=$1`, []any{userID}},
		{"verification_challenges", `DELETE FROM iam.verification_challenges WHERE user_id=$1`, []any{userID}},
	}
	for _, statement := range statements {
		if err := exec(statement.name, statement.query, statement.args...); err != nil {
			return err
		}
	}
	return nil
}

func collectErasureObjects(ctx context.Context, tx pgx.Tx, app Application, userID uuid.UUID, global bool) ([]erasureObject, error) {
	fileScope := `f.app_id=$2`
	sessionScope := `s.app_id=$2`
	usageScope := `NOT EXISTS(SELECT 1 FROM storage.file_usages u WHERE u.file_id=f.id)`
	fileArgs := []any{userID, app.ID}
	sessionArgs := []any{userID, app.ID}
	if global {
		fileScope, sessionScope = `TRUE`, `TRUE`
		usageScope = `NOT EXISTS(SELECT 1 FROM storage.file_usages u WHERE u.file_id=f.id AND NOT (u.entity_type='iam.user' AND u.entity_id=$1))`
		fileArgs, sessionArgs = []any{userID}, []any{userID}
	}
	rows, err := tx.Query(ctx, `SELECT f.id,f.provider,f.bucket_name,f.object_key FROM storage.files f
		WHERE f.owner_user_id=$1 AND `+fileScope+` AND `+usageScope+`
		FOR UPDATE`, fileArgs...)
	if err != nil {
		return nil, fmt.Errorf("collect account files: %w", err)
	}
	objects := make([]erasureObject, 0)
	seen := map[string]struct{}{}
	for rows.Next() {
		var object erasureObject
		if err = rows.Scan(&object.FileID, &object.Provider, &object.Bucket, &object.Key); err != nil {
			rows.Close()
			return nil, err
		}
		seen[object.Provider+"\x00"+object.Bucket+"\x00"+object.Key] = struct{}{}
		objects = append(objects, object)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	rows, err = tx.Query(ctx, `SELECT s.provider,s.bucket_name,s.object_key FROM storage.upload_sessions s
		WHERE s.user_id=$1 AND `+sessionScope+` AND s.file_id IS NULL FOR UPDATE`, sessionArgs...)
	if err != nil {
		return nil, fmt.Errorf("collect account upload objects: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var provider, bucket, key string
		if err = rows.Scan(&provider, &bucket, &key); err != nil {
			return nil, err
		}
		dedupe := provider + "\x00" + bucket + "\x00" + key
		if _, ok := seen[dedupe]; ok {
			continue
		}
		seen[dedupe] = struct{}{}
		objects = append(objects, erasureObject{Provider: provider, Bucket: bucket, Key: key})
	}
	return objects, rows.Err()
}

func scrubAndRemoveStorageMetadata(ctx context.Context, tx pgx.Tx, app Application, userID uuid.UUID, global bool, objects []erasureObject, counts map[string]int64) error {
	if global {
		if tag, err := tx.Exec(ctx, `UPDATE iam.users SET avatar_file_id=NULL WHERE id=$1`, userID); err != nil {
			return err
		} else {
			counts["avatar_links"] += tag.RowsAffected()
		}
		if tag, err := tx.Exec(ctx, `DELETE FROM storage.file_usages WHERE entity_type='iam.user' AND entity_id=$1`, userID); err != nil {
			return err
		} else {
			counts["file_usages"] += tag.RowsAffected()
		}
	}
	fileScope := `app_id=$2`
	sessionScope := `app_id=$2`
	fileArgs := []any{userID, app.ID}
	sessionArgs := []any{userID, app.ID}
	if global {
		fileScope, sessionScope = `TRUE`, `TRUE`
		fileArgs, sessionArgs = []any{userID}, []any{userID}
	}
	if tag, err := tx.Exec(ctx, `DELETE FROM storage.upload_sessions WHERE user_id=$1 AND `+sessionScope, sessionArgs...); err != nil {
		return fmt.Errorf("erase upload sessions: %w", err)
	} else {
		counts["upload_sessions"] += tag.RowsAffected()
	}
	fileIDs := make([]uuid.UUID, 0, len(objects))
	for _, object := range objects {
		if object.FileID != uuid.Nil {
			fileIDs = append(fileIDs, object.FileID)
		}
	}
	if len(fileIDs) > 0 {
		if tag, err := tx.Exec(ctx, `DELETE FROM storage.files WHERE id=ANY($1::uuid[])`, fileIDs); err != nil {
			return fmt.Errorf("erase file metadata: %w", err)
		} else {
			counts["files"] += tag.RowsAffected()
		}
	}
	if tag, err := tx.Exec(ctx, `UPDATE storage.files SET owner_user_id=NULL,original_name='erased-account-file',metadata='{}'::jsonb
		WHERE owner_user_id=$1 AND `+fileScope, fileArgs...); err != nil {
		return fmt.Errorf("anonymize retained files: %w", err)
	} else {
		counts["anonymized_files"] += tag.RowsAffected()
	}
	return nil
}
