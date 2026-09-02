//go:build integration

package application

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type deletionAuthenticator struct{ principal iam.AuthenticatedContext }

type captureErasureQueue struct{ objectIDs []uuid.UUID }

func (q *captureErasureQueue) Enqueue(_ context.Context, _ pgx.Tx, _, _, _, objectID uuid.UUID) error {
	q.objectIDs = append(q.objectIDs, objectID)
	return nil
}

func (a deletionAuthenticator) Authenticate(_ context.Context, _ string, audience string) (iam.AuthenticatedContext, error) {
	if audience != "ak-mobile" {
		return iam.AuthenticatedContext{}, errors.New("unexpected audience")
	}
	return a.principal, nil
}

func TestAccountDeletionVerificationIsBoundAndAttemptLimited(t *testing.T) {
	pool := accountDeletionTestPool(t)
	ctx := context.Background()
	suffix := uuid.NewString()
	tenantID, appID := createDeletionTenantApp(t, pool, "verify-"+suffix)
	userID := createDeletionUser(t, pool, "verify-"+suffix+"@example.test", tenantID, appID)
	defer cleanupDeletionFixtures(pool, []uuid.UUID{tenantID})

	app := Application{ID: appID, TenantID: tenantID, DefaultLocale: "zh-CN"}
	notifier := &integrationOTPNotifier{}
	service := NewService(pool, deletionAuthenticator{principal: deletionPrincipal(userID, tenantID, appID)}, WithOTPNotifier(notifier), WithAccountDeletionEnabled(true))
	result, err := service.RequestAccountDeletionCode(ctx, "session-token", app)
	if err != nil || !result.Accepted || result.ExpiresInSeconds != 600 || result.RetryAfterSeconds != 60 || result.TargetHint == "" {
		t.Fatalf("request deletion code result=%#v err=%v", result, err)
	}
	if len(notifier.calls) != 1 || notifier.calls[0].Purpose != "account_delete" || notifier.calls[0].AppID != appID || notifier.calls[0].TenantID != tenantID || len(notifier.calls[0].Code) != 6 {
		t.Fatalf("deletion notification was not bound to current app and user: %#v", notifier.calls)
	}
	if cooldown, cooldownErr := service.RequestAccountDeletionCode(ctx, "session-token", app); !errors.Is(cooldownErr, ErrAccountDeletionCodeCooldown) || cooldown.RetryAfterSeconds < 1 || cooldown.TargetHint != result.TargetHint {
		t.Fatalf("cooldown result=%#v err=%v", cooldown, cooldownErr)
	}
	wrongCode := "000000"
	if notifier.calls[0].Code == wrongCode {
		wrongCode = "111111"
	}
	for attempt := 1; attempt <= 5; attempt++ {
		_, confirmErr := service.ConfirmAccountDeletion(ctx, "session-token", app, wrongCode, "", true)
		if attempt < 5 && !errors.Is(confirmErr, ErrAccountDeletionCodeInvalid) {
			t.Fatalf("attempt %d error=%v, want invalid code", attempt, confirmErr)
		}
		if attempt == 5 && !errors.Is(confirmErr, ErrAccountDeletionCodeExhausted) {
			t.Fatalf("attempt %d error=%v, want exhausted code", attempt, confirmErr)
		}
	}
	var attempts int
	var consumed bool
	if err = pool.QueryRow(ctx, `SELECT attempts,consumed_at IS NOT NULL FROM iam.verification_challenges WHERE user_id=$1 AND challenge_type='account_delete'`, userID).Scan(&attempts, &consumed); err != nil || attempts != 5 || !consumed {
		t.Fatalf("challenge attempts=%d consumed=%t err=%v", attempts, consumed, err)
	}
	var memberships int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM app.user_memberships WHERE app_id=$1 AND user_id=$2`, appID, userID).Scan(&memberships); err != nil || memberships != 1 {
		t.Fatalf("failed verification changed account membership count=%d err=%v", memberships, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE iam.verification_challenges SET created_at=now()-interval '2 minutes' WHERE user_id=$1 AND challenge_type='account_delete'`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RequestAccountDeletionCode(ctx, "session-token", app); err != nil {
		t.Fatal(err)
	}
	expiringCode := notifier.calls[len(notifier.calls)-1].Code
	if _, err = pool.Exec(ctx, `UPDATE iam.verification_challenges SET created_at=now()-interval '11 minutes',expires_at=now()-interval '1 minute'
		WHERE user_id=$1 AND challenge_type='account_delete' AND consumed_at IS NULL`, userID); err != nil {
		t.Fatal(err)
	}
	if _, confirmErr := service.ConfirmAccountDeletion(ctx, "session-token", app, expiringCode, "", true); !errors.Is(confirmErr, ErrAccountDeletionCodeExpired) {
		t.Fatalf("expired code error=%v, want expired", confirmErr)
	}
	assertCount(t, pool, `SELECT count(*) FROM app.user_memberships WHERE app_id=$1 AND user_id=$2`, 1, appID, userID)
}

func TestAppleBoundAccountDeletionFailsClosedWithoutStepUpConsumer(t *testing.T) {
	pool := accountDeletionTestPool(t)
	ctx := context.Background()
	suffix := uuid.NewString()
	tenantID, appID := createDeletionTenantApp(t, pool, "apple-fail-closed-"+suffix)
	defer cleanupDeletionFixtures(pool, []uuid.UUID{tenantID})
	userID := createDeletionUser(t, pool, "apple-"+suffix+"@example.test", tenantID, appID)
	if _, err := pool.Exec(ctx, `INSERT INTO iam.app_oauth_accounts(
tenant_id,app_id,user_id,provider_code,issuer,external_client_id,subject,status)
VALUES($1,$2,$3,'apple','https://appleid.apple.com','com.example.app',$4,'active')`, tenantID, appID, userID, "subject-"+suffix); err != nil {
		t.Fatal(err)
	}
	notifier := &integrationOTPNotifier{}
	service := NewService(pool, deletionAuthenticator{principal: deletionPrincipal(userID, tenantID, appID)}, WithOTPNotifier(notifier), WithAccountDeletionEnabled(true))
	app := Application{ID: appID, TenantID: tenantID, DefaultLocale: "zh-CN"}
	requirements, err := service.RequestAccountDeletionCode(ctx, "session", app)
	if err != nil || !requirements.ReauthRequired || requirements.ReauthProvider != "apple" {
		t.Fatalf("requirements=%#v err=%v", requirements, err)
	}
	if _, err = service.ConfirmAccountDeletion(ctx, "session", app, notifier.calls[0].Code, "forged", true); !errors.Is(err, ErrAccountDeletionUnavailable) {
		t.Fatalf("confirm error=%v, want fail-closed unavailable", err)
	}
	assertCount(t, pool, `SELECT count(*) FROM app.user_memberships WHERE app_id=$1 AND user_id=$2`, 1, appID, userID)
}

func TestAppleAccountDeletionRejectsBindingSwappedAfterReauth(t *testing.T) {
	pool := accountDeletionTestPool(t)
	ctx := context.Background()
	suffix := uuid.NewString()
	tenantID, appID := createDeletionTenantApp(t, pool, "apple-swap-"+suffix)
	defer cleanupDeletionFixtures(pool, []uuid.UUID{tenantID})
	userID := createDeletionUser(t, pool, "apple-swap-"+suffix+"@example.test", tenantID, appID)
	var originalID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO iam.app_oauth_accounts(
tenant_id,app_id,user_id,provider_code,issuer,external_client_id,subject,status)
VALUES($1,$2,$3,'apple','https://appleid.apple.com','com.example.app',$4,'active') RETURNING id`,
		tenantID, appID, userID, "original-"+suffix).Scan(&originalID); err != nil {
		t.Fatal(err)
	}
	notifier := &integrationOTPNotifier{}
	service := NewService(pool, deletionAuthenticator{principal: deletionPrincipal(userID, tenantID, appID)},
		WithOTPNotifier(notifier), WithAccountDeletionEnabled(true),
		WithAccountDeletionStepUp(func(context.Context, string, uuid.UUID, string) (uuid.UUID, error) {
			// Deterministically simulate another session replacing the binding in
			// the interval after the original provider reauth and before deletion.
			if _, err := pool.Exec(ctx, `DELETE FROM iam.app_oauth_accounts WHERE id=$1`, originalID); err != nil {
				return uuid.Nil, err
			}
			if _, err := pool.Exec(ctx, `INSERT INTO iam.app_oauth_accounts(
tenant_id,app_id,user_id,provider_code,issuer,external_client_id,subject,status)
VALUES($1,$2,$3,'apple','https://appleid.apple.com','com.example.app',$4,'active')`,
				tenantID, appID, userID, "replacement-"+suffix); err != nil {
				return uuid.Nil, err
			}
			return originalID, nil
		}),
	)
	app := Application{ID: appID, TenantID: tenantID, DefaultLocale: "zh-CN"}
	if _, err := service.RequestAccountDeletionCode(ctx, "session", app); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmAccountDeletion(ctx, "session", app, notifier.calls[0].Code, "fresh-provider-ticket", true); !errors.Is(err, ErrAccountDeletionUnavailable) {
		t.Fatalf("confirm error=%v, want fail-closed unavailable after Apple binding swap", err)
	}
	assertCount(t, pool, `SELECT count(*) FROM app.user_memberships WHERE app_id=$1 AND user_id=$2`, 1, appID, userID)
	assertCount(t, pool, `SELECT count(*) FROM iam.app_oauth_accounts WHERE app_id=$1 AND user_id=$2 AND provider_code='apple'`, 1, appID, userID)
}

func TestAccountDeletionRemovesOnlyCurrentAppThenDeletesUnsharedIdentity(t *testing.T) {
	pool := accountDeletionTestPool(t)
	ctx := context.Background()
	suffix := uuid.NewString()
	tenantA, appA := createDeletionTenantApp(t, pool, "scope-a-"+suffix)
	tenantB, appB := createDeletionTenantApp(t, pool, "scope-b-"+suffix)
	defer cleanupDeletionFixtures(pool, []uuid.UUID{tenantA, tenantB})
	email := "scope-" + suffix + "@example.test"
	userID := createDeletionUser(t, pool, email, tenantA, appA)
	if _, err := pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, tenantB, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app.user_memberships(app_id,tenant_id,user_id,source,status,verified_at) VALUES($1,$2,$3,'self_registration','active',now())`, appB, tenantB, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app.user_login_identifiers(tenant_id,app_id,user_id,identifier_type,normalized_value,display_hint,verified_at,status)
		VALUES($1,$2,$3,'email',$4,$5,now(),'active')`, tenantB, appB, userID, email, maskedEmail(email)); err != nil {
		t.Fatal(err)
	}
	sessionA := createDeletionSession(t, pool, userID, tenantA, appA)
	sessionB := createDeletionSession(t, pool, userID, tenantB, appB)
	if _, err := pool.Exec(ctx, `INSERT INTO iam.user_preferences(app_id,user_id,locale,appearance) VALUES($1,$2,'zh-CN','dark'),($3,$2,'en-US','light')`, appA, userID, appB); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,request_path,succeeded)
		VALUES($1,$2,$3,$4,'iam','mobile.preference.update','iam.preference.manage_self','/api/v1/me/preferences',true)`, tenantA, userID, sessionA, "scope-a-"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO audit.login_events(tenant_id,user_id,session_id,app_id,auth_method,audience,result,login_identifier_hash,login_identifier_hint,client_ip,user_agent,device_info)
		VALUES($1,$2,$3,$4,'password','ak-mobile','success',$5,$6,'192.0.2.1','integration-device','{"model":"private"}')`, tenantA, userID, sessionA, appA, sha256Bytes(email), email); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO audit.security_events(tenant_id,user_id,session_id,app_id,event_type,severity,source,client_ip,details)
		VALUES($1,$2,$3,$4,'test.private','low','integration','192.0.2.2','{"device":"private"}')`, tenantA, userID, sessionA, appA); err != nil {
		t.Fatal(err)
	}
	var exclusiveFileID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO storage.files(tenant_id,owner_user_id,provider,bucket_name,object_key,original_name,size_bytes,status,scan_status,app_id)
		VALUES($1,$2,'local','privacy-test',$3,'private-photo.jpg',4,'ready','clean',$4) RETURNING id`, tenantA, userID, "accounts/"+suffix+"/private-photo.jpg", appA).Scan(&exclusiveFileID); err != nil {
		t.Fatal(err)
	}
	otherUserID := createDeletionUser(t, pool, "reply-"+suffix+"@example.test", tenantA, appA)
	var articleID, parentCommentID, replyCommentID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO content.articles(tenant_id,app_id,slug,status) VALUES($1,$2,$3,'draft') RETURNING id`, tenantA, appA, "deletion-"+suffix).Scan(&articleID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO content.comments(tenant_id,app_id,article_id,author_id,status,body,body_fingerprint)
		VALUES($1,$2,$3,$4,'pending','parent comment',$5) RETURNING id`, tenantA, appA, articleID, userID, sha256Bytes("parent-"+suffix)).Scan(&parentCommentID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO content.comments(tenant_id,app_id,article_id,author_id,parent_id,root_id,status,body,body_fingerprint)
		VALUES($1,$2,$3,$4,$5,$5,'pending','retained reply',$6) RETURNING id`, tenantA, appA, articleID, otherUserID, parentCommentID, sha256Bytes("reply-"+suffix)).Scan(&replyCommentID); err != nil {
		t.Fatal(err)
	}

	notifier := &integrationOTPNotifier{}
	erasureQueue := &captureErasureQueue{}
	serviceA := NewService(pool, deletionAuthenticator{principal: deletionPrincipal(userID, tenantA, appA)}, WithOTPNotifier(notifier), WithObjectErasureQueue(erasureQueue), WithAccountDeletionEnabled(true))
	if _, err := serviceA.RequestAccountDeletionCode(ctx, "session-a", Application{ID: appA, TenantID: tenantA, DefaultLocale: "zh-CN"}); err != nil {
		t.Fatal(err)
	}
	codeA := notifier.calls[len(notifier.calls)-1].Code
	if result, err := serviceA.ConfirmAccountDeletion(ctx, "session-a", Application{ID: appA, TenantID: tenantA}, codeA, "", true); err != nil || !result.Deleted {
		t.Fatalf("delete current app result=%#v err=%v", result, err)
	}

	assertCount(t, pool, `SELECT count(*) FROM app.user_memberships WHERE app_id=$1 AND user_id=$2`, 0, appA, userID)
	assertCount(t, pool, `SELECT count(*) FROM iam.sessions WHERE id=$1`, 0, sessionA)
	assertCount(t, pool, `SELECT count(*) FROM iam.user_preferences WHERE app_id=$1 AND user_id=$2`, 0, appA, userID)
	assertCount(t, pool, `SELECT count(*) FROM app.user_memberships WHERE app_id=$1 AND user_id=$2`, 1, appB, userID)
	assertCount(t, pool, `SELECT count(*) FROM iam.sessions WHERE id=$1`, 1, sessionB)
	assertCount(t, pool, `SELECT count(*) FROM iam.users WHERE id=$1`, 1, userID)
	assertCount(t, pool, `SELECT count(*) FROM storage.files WHERE id=$1`, 0, exclusiveFileID)
	if len(erasureQueue.objectIDs) != 1 {
		t.Fatalf("exclusive object erasure jobs=%d, want 1", len(erasureQueue.objectIDs))
	}
	assertCount(t, pool, `SELECT count(*) FROM content.comments WHERE id=$1`, 0, parentCommentID)
	var replyParent, replyRoot *uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT parent_id,root_id FROM content.comments WHERE id=$1`, replyCommentID).Scan(&replyParent, &replyRoot); err != nil || replyParent != nil || replyRoot != nil {
		t.Fatalf("other user's reply was not retained and detached parent=%v root=%v err=%v", replyParent, replyRoot, err)
	}
	var actionCode string
	var globalDeleted bool
	var erasureStatus string
	if err := pool.QueryRow(ctx, `SELECT action_code,global_identity_deleted,status FROM audit.privacy_erasure_events WHERE app_id=$1 ORDER BY requested_at DESC LIMIT 1`, appA).Scan(&actionCode, &globalDeleted, &erasureStatus); err != nil || actionCode != "iam.user.delete_self" || globalDeleted || erasureStatus != "pending_objects" {
		t.Fatalf("current-app erasure action=%q global=%t status=%q err=%v", actionCode, globalDeleted, erasureStatus, err)
	}
	var operationUser *uuid.UUID
	var requestID string
	if err := pool.QueryRow(ctx, `SELECT user_id,request_id FROM audit.operation_logs WHERE app_id=$1 ORDER BY occurred_at DESC LIMIT 1`, appA).Scan(&operationUser, &requestID); err != nil || operationUser != nil || requestID != "erased" {
		t.Fatalf("operation audit was not anonymized user=%v request=%q err=%v", operationUser, requestID, err)
	}
	var loginUser *uuid.UUID
	var loginHint *string
	var loginHash []byte
	if err := pool.QueryRow(ctx, `SELECT user_id,login_identifier_hint,login_identifier_hash FROM audit.login_events WHERE app_id=$1 ORDER BY occurred_at DESC LIMIT 1`, appA).Scan(&loginUser, &loginHint, &loginHash); err != nil || loginUser != nil || loginHint != nil || loginHash != nil {
		t.Fatalf("login audit retained identifying data user=%v hint=%v hash=%x err=%v", loginUser, loginHint, loginHash, err)
	}

	serviceB := NewService(pool, deletionAuthenticator{principal: deletionPrincipal(userID, tenantB, appB)}, WithOTPNotifier(notifier), WithAccountDeletionEnabled(true))
	if _, err := serviceB.RequestAccountDeletionCode(ctx, "session-b", Application{ID: appB, TenantID: tenantB, DefaultLocale: "en-US"}); err != nil {
		t.Fatal(err)
	}
	codeB := notifier.calls[len(notifier.calls)-1].Code
	if _, err := serviceB.ConfirmAccountDeletion(ctx, "session-b", Application{ID: appB, TenantID: tenantB}, codeB, "", true); err != nil {
		t.Fatal(err)
	}
	assertCount(t, pool, `SELECT count(*) FROM iam.users WHERE id=$1`, 0, userID)
	assertCount(t, pool, `SELECT count(*) FROM iam.user_credentials WHERE user_id=$1`, 0, userID)
	assertCount(t, pool, `SELECT count(*) FROM iam.sessions WHERE user_id=$1`, 0, userID)
	if err := pool.QueryRow(ctx, `SELECT global_identity_deleted FROM audit.privacy_erasure_events WHERE app_id=$1 ORDER BY requested_at DESC LIMIT 1`, appB).Scan(&globalDeleted); err != nil || !globalDeleted {
		t.Fatalf("last mobile identity global deletion=%t err=%v", globalDeleted, err)
	}

	var newUserID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,locale,status,email_verified_at) VALUES($1,'New identity','zh-CN','active',now()) RETURNING id`, email).Scan(&newUserID); err != nil {
		t.Fatal(err)
	}
	if newUserID == userID {
		t.Fatal("re-registration reused the deleted global identity")
	}
	assertCount(t, pool, `SELECT count(*) FROM app.user_memberships WHERE app_id=$1 AND user_id=$2`, 0, appB, newUserID)
	if _, err := pool.Exec(ctx, `DELETE FROM iam.users WHERE id=$1`, newUserID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM content.articles WHERE id=$1`, articleID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM app.user_memberships WHERE user_id=$1`, otherUserID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM iam.tenant_members WHERE user_id=$1`, otherUserID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM iam.users WHERE id=$1`, otherUserID); err != nil {
		t.Fatal(err)
	}
}

func TestAccountDeletionRetainsIdentityUsedByAdmin(t *testing.T) {
	pool := accountDeletionTestPool(t)
	ctx := context.Background()
	suffix := uuid.NewString()
	tenantID, appID := createDeletionTenantApp(t, pool, "admin-retained-"+suffix)
	defer cleanupDeletionFixtures(pool, []uuid.UUID{tenantID})
	userID := createDeletionUser(t, pool, "admin-retained-"+suffix+"@example.test", tenantID, appID)
	mobileSessionID := createDeletionSession(t, pool, userID, tenantID, appID)
	var adminSessionID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO iam.sessions(user_id,tenant_id,audience,status,absolute_expires_at,app_id)
		VALUES($1,$2,'ak-admin','active',now()+interval '1 hour',$3) RETURNING id`, userID, tenantID, appID).Scan(&adminSessionID); err != nil {
		t.Fatal(err)
	}

	notifier := &integrationOTPNotifier{}
	service := NewService(pool, deletionAuthenticator{principal: deletionPrincipal(userID, tenantID, appID)}, WithOTPNotifier(notifier), WithAccountDeletionEnabled(true))
	app := Application{ID: appID, TenantID: tenantID, DefaultLocale: "zh-CN"}
	if _, err := service.RequestAccountDeletionCode(ctx, "mobile-session", app); err != nil {
		t.Fatal(err)
	}
	code := notifier.calls[len(notifier.calls)-1].Code
	if result, err := service.ConfirmAccountDeletion(ctx, "mobile-session", app, code, "", true); err != nil || !result.Deleted {
		t.Fatalf("delete app identity result=%#v err=%v", result, err)
	}

	assertCount(t, pool, `SELECT count(*) FROM app.user_memberships WHERE app_id=$1 AND user_id=$2`, 0, appID, userID)
	assertCount(t, pool, `SELECT count(*) FROM iam.sessions WHERE id=$1`, 0, mobileSessionID)
	assertCount(t, pool, `SELECT count(*) FROM iam.sessions WHERE id=$1`, 1, adminSessionID)
	assertCount(t, pool, `SELECT count(*) FROM iam.users WHERE id=$1`, 1, userID)
	assertCount(t, pool, `SELECT count(*) FROM iam.user_credentials WHERE user_id=$1`, 1, userID)
	var globalDeleted bool
	if err := pool.QueryRow(ctx, `SELECT global_identity_deleted FROM audit.privacy_erasure_events WHERE app_id=$1 ORDER BY requested_at DESC LIMIT 1`, appID).Scan(&globalDeleted); err != nil || globalDeleted {
		t.Fatalf("admin-shared identity global deletion=%t err=%v", globalDeleted, err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM iam.sessions WHERE id=$1`, adminSessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM iam.tenant_members WHERE tenant_id=$1 AND user_id=$2`, tenantID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM iam.users WHERE id=$1`, userID); err != nil {
		t.Fatal(err)
	}
}

func accountDeletionTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("AK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func createDeletionTenantApp(t *testing.T, pool *pgxpool.Pool, code string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	var tenantID, appID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,$2) RETURNING id`, code, code).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, tenantID).Scan(&appID); err != nil {
		t.Fatal(err)
	}
	return tenantID, appID
}

func createDeletionUser(t *testing.T, pool *pgxpool.Pool, email string, tenantID, appID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var userID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,locale,status,email_verified_at) VALUES($1,'Deletion User','zh-CN','active',now()) RETURNING id`, email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO iam.user_credentials(user_id,password_hash) VALUES($1,'integration-hash')`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, tenantID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app.user_memberships(app_id,tenant_id,user_id,source,status,verified_at) VALUES($1,$2,$3,'self_registration','active',now())`, appID, tenantID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app.user_login_identifiers(tenant_id,app_id,user_id,identifier_type,normalized_value,display_hint,verified_at,status)
		VALUES($1,$2,$3,'email',$4,$5,now(),'active')`, tenantID, appID, userID, strings.ToLower(email), maskedEmail(strings.ToLower(email))); err != nil {
		t.Fatal(err)
	}
	return userID
}

func createDeletionSession(t *testing.T, pool *pgxpool.Pool, userID, tenantID, appID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(context.Background(), `INSERT INTO iam.sessions(user_id,tenant_id,audience,status,absolute_expires_at,app_id) VALUES($1,$2,'ak-mobile','active',now()+interval '1 hour',$3) RETURNING id`, userID, tenantID, appID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func deletionPrincipal(userID, tenantID, appID uuid.UUID) iam.AuthenticatedContext {
	return iam.AuthenticatedContext{AuthContext: iam.AuthContext{User: iam.User{ID: userID}, Tenant: iam.Tenant{ID: tenantID}}, AppID: &appID}
}

func assertCount(t *testing.T, pool *pgxpool.Pool, query string, expected int, args ...any) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil || count != expected {
		t.Fatalf("query count=%d want=%d err=%v query=%s", count, expected, err, query)
	}
}

func cleanupDeletionFixtures(pool *pgxpool.Pool, tenantIDs []uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = pool.Exec(ctx, `DELETE FROM audit.privacy_erasure_events WHERE tenant_id=ANY($1::uuid[])`, tenantIDs)
	_, _ = pool.Exec(ctx, `DELETE FROM iam.tenants WHERE id=ANY($1::uuid[])`, tenantIDs)
}
