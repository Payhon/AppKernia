//go:build integration

package application_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConcurrentRefreshRotatesOnceAndRevokesOnReuse(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer pool.Close()

	password := "refresh integration password"
	passwordHash, err := application.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	suffix := uuid.NewString()
	postgresRepository := repository.NewPostgres(pool)
	user, _, err := postgresRepository.CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: fmt.Sprintf("refresh-%s", suffix), TenantName: "Refresh Tenant",
		Email: fmt.Sprintf("refresh-%s@example.test", suffix), DisplayName: "Refresh User",
		Locale: "zh-CN", PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	issuer, err := application.NewDevelopmentTokenIssuer()
	if err != nil {
		t.Fatalf("create token issuer: %v", err)
	}
	service, err := application.NewAuthService(postgresRepository, postgresRepository, issuer)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	login, err := service.Login(ctx, application.LoginInput{
		Email: fmt.Sprintf("refresh-%s@example.test", suffix), Password: password, Audience: "ak-admin",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	start := make(chan struct{})
	type refreshResult struct {
		tokens application.SessionTokens
		err    error
	}
	results := make(chan refreshResult, 2)
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			rotatedTokens, refreshErr := service.Refresh(ctx, login.RefreshToken, "ak-admin", application.ClientMetadata{})
			results <- refreshResult{tokens: rotatedTokens, err: refreshErr}
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)

	successes, reuseFailures := 0, 0
	var rotatedAccessToken string
	for result := range results {
		switch {
		case result.err == nil:
			successes++
			rotatedAccessToken = result.tokens.AccessToken
		case errors.Is(result.err, domain.ErrRefreshReused):
			reuseFailures++
		default:
			t.Fatalf("unexpected refresh result: %v", result.err)
		}
	}
	if successes != 1 || reuseFailures != 1 {
		t.Fatalf("expected one success and one reuse failure, got success=%d reuse=%d", successes, reuseFailures)
	}
	var eventCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.security_events WHERE user_id = $1 AND event_type = 'iam.refresh_token.reuse'`, user.ID).Scan(&eventCount); err != nil {
		t.Fatalf("count security events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("expected one refresh reuse security event, got %d", eventCount)
	}
	if _, contextErr := service.Context(ctx, rotatedAccessToken, "ak-admin"); !errors.Is(contextErr, application.ErrInvalidAccessToken) {
		t.Fatalf("reused token family must invalidate rotated access token, got %v", contextErr)
	}
}

func TestSelfSessionManagementIsScopedRevokesTokenFamilyAndAudits(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer pool.Close()

	password := "session integration password"
	passwordHash, err := application.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	suffix := uuid.NewString()
	repo := repository.NewPostgres(pool)
	user, tenant, err := repo.CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: "sessions-" + suffix, TenantName: "Sessions Tenant",
		Email: "sessions-" + suffix + "@example.test", DisplayName: "Sessions User",
		Locale: "zh-CN", PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	foreignUser, _, err := repo.CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: "foreign-" + suffix, TenantName: "Foreign Tenant",
		Email: "foreign-" + suffix + "@example.test", DisplayName: "Foreign User",
		Locale: "en-US", PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatalf("create foreign identity: %v", err)
	}
	issuer, err := application.NewDevelopmentTokenIssuer()
	if err != nil {
		t.Fatalf("create token issuer: %v", err)
	}
	service, err := application.NewAuthService(repo, repo, issuer)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	email := "sessions-" + suffix + "@example.test"
	first, err := service.Login(ctx, application.LoginInput{Email: email, Password: password, Audience: "ak-admin"})
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	second, err := service.Login(ctx, application.LoginInput{Email: email, Password: password, Audience: "ak-admin"})
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	foreign, err := service.Login(ctx, application.LoginInput{
		Email: "foreign-" + suffix + "@example.test", Password: password, Audience: "ak-admin",
	})
	if err != nil {
		t.Fatalf("foreign login: %v", err)
	}

	sessions, err := service.ListSelfSessions(ctx, first.AccessToken, "ak-admin")
	if err != nil {
		t.Fatalf("list self sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected two self sessions, got %d", len(sessions))
	}
	currentCount := 0
	for _, session := range sessions {
		if session.Current {
			currentCount++
			if session.ID != first.SessionID {
				t.Fatalf("wrong session marked current: %s", session.ID)
			}
		}
	}
	if currentCount != 1 {
		t.Fatalf("expected exactly one current session, got %d", currentCount)
	}

	current, err := service.RevokeSelfSession(ctx, first.AccessToken, "ak-admin", application.RevokeSelfSessionInput{
		SessionID: second.SessionID, RequestID: "self-session-revoke-" + suffix,
	})
	if err != nil || current {
		t.Fatalf("revoke other self session: current=%t err=%v", current, err)
	}
	if _, contextErr := service.Context(ctx, second.AccessToken, "ak-admin"); !errors.Is(contextErr, application.ErrInvalidAccessToken) {
		t.Fatalf("revoked access token must be invalid, got %v", contextErr)
	}
	if _, refreshErr := service.Refresh(ctx, second.RefreshToken, "ak-admin", application.ClientMetadata{}); !errors.Is(refreshErr, domain.ErrRefreshInvalid) {
		t.Fatalf("revoked refresh family must be invalid, got %v", refreshErr)
	}
	if _, err = service.RevokeSelfSession(ctx, first.AccessToken, "ak-admin", application.RevokeSelfSessionInput{
		SessionID: foreign.SessionID, RequestID: "foreign-session-revoke-" + suffix,
	}); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("foreign session must remain outside self scope, got %v", err)
	}
	if _, contextErr := service.Context(ctx, foreign.AccessToken, "ak-admin"); contextErr != nil {
		t.Fatalf("foreign session must remain active: %v", contextErr)
	}

	var auditCount, revokedRefreshCount int
	if err = pool.QueryRow(ctx, `
		SELECT count(*) FROM audit.operation_logs
		WHERE user_id = $1 AND tenant_id = $2 AND request_id = $3
		  AND action_name = 'iam.me.session.revoke' AND resource_id = $4 AND succeeded = true
	`, user.ID, tenant.ID, "self-session-revoke-"+suffix, second.SessionID.String()).Scan(&auditCount); err != nil {
		t.Fatalf("count session revoke audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one session revoke audit record, got %d", auditCount)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM iam.refresh_tokens WHERE session_id = $1 AND revoked_at IS NOT NULL`, second.SessionID).Scan(&revokedRefreshCount); err != nil {
		t.Fatalf("count revoked refresh tokens: %v", err)
	}
	if revokedRefreshCount == 0 {
		t.Fatal("expected the revoked session refresh token family to be revoked")
	}
	if foreignUser.ID == user.ID {
		t.Fatal("foreign fixture must use a distinct user")
	}
}

func TestSelfDeviceManagementBindsSessionsScopesRemovalAndAudits(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer pool.Close()

	password := "device integration password"
	passwordHash, err := application.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	suffix := uuid.NewString()
	repo := repository.NewPostgres(pool)
	user, tenant, err := repo.CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: "devices-" + suffix, TenantName: "Devices Tenant",
		Email: "devices-" + suffix + "@example.test", DisplayName: "Devices User",
		Locale: "zh-CN", PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	_, _, err = repo.CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: "device-foreign-" + suffix, TenantName: "Foreign Device Tenant",
		Email: "device-foreign-" + suffix + "@example.test", DisplayName: "Foreign Device User",
		Locale: "en-US", PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatalf("create foreign identity: %v", err)
	}
	issuer, err := application.NewDevelopmentTokenIssuer()
	if err != nil {
		t.Fatalf("create token issuer: %v", err)
	}
	service, err := application.NewAuthService(repo, repo, issuer)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	email := "devices-" + suffix + "@example.test"
	if _, loginErr := service.Login(ctx, application.LoginInput{
		Email: email, Password: "invalid device password", Audience: "ak-admin",
		Client: application.ClientMetadata{RequestID: "known-login-failure-" + suffix},
	}); !errors.Is(loginErr, application.ErrInvalidCredentials) {
		t.Fatalf("known invalid login: %v", loginErr)
	}
	if _, loginErr := service.Login(ctx, application.LoginInput{
		Email: "missing-" + suffix + "@example.test", Password: "invalid device password", Audience: "ak-admin",
		Client: application.ClientMetadata{RequestID: "unknown-login-failure-" + suffix},
	}); !errors.Is(loginErr, application.ErrInvalidCredentials) {
		t.Fatalf("unknown invalid login: %v", loginErr)
	}
	currentDeviceKey := uuid.NewString()
	otherDeviceKey := uuid.NewString()
	current, err := service.Login(ctx, application.LoginInput{
		Email: email, Password: password, Audience: "ak-admin",
		Client: application.ClientMetadata{DeviceKey: currentDeviceKey, UserAgent: "Current Device Browser"},
	})
	if err != nil {
		t.Fatalf("current device login: %v", err)
	}
	other, err := service.Login(ctx, application.LoginInput{
		Email: email, Password: password, Audience: "ak-admin",
		Client: application.ClientMetadata{DeviceKey: otherDeviceKey, UserAgent: "Other Device Browser"},
	})
	if err != nil {
		t.Fatalf("other device login: %v", err)
	}
	otherAPI, err := service.Login(ctx, application.LoginInput{
		Email: email, Password: password, Audience: "ak-api",
		Client: application.ClientMetadata{DeviceKey: otherDeviceKey, UserAgent: "Other Device API Client"},
	})
	if err != nil {
		t.Fatalf("other device API login: %v", err)
	}
	foreign, err := service.Login(ctx, application.LoginInput{
		Email: "device-foreign-" + suffix + "@example.test", Password: password, Audience: "ak-admin",
		Client: application.ClientMetadata{DeviceKey: uuid.NewString(), UserAgent: "Foreign Device Browser"},
	})
	if err != nil {
		t.Fatalf("foreign device login: %v", err)
	}

	devices, err := service.ListSelfDevices(ctx, current.AccessToken, "ak-admin")
	if err != nil {
		t.Fatalf("list self devices: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected two self devices, got %d", len(devices))
	}
	var currentDeviceID, otherDeviceID uuid.UUID
	currentCount := 0
	for _, device := range devices {
		if device.Current {
			currentCount++
			currentDeviceID = device.ID
			if device.LatestUserAgent != "Current Device Browser" {
				t.Fatalf("unexpected current device agent: %q", device.LatestUserAgent)
			}
		} else {
			otherDeviceID = device.ID
			if device.ActiveSessionCount != 1 {
				t.Fatalf("admin list must count only the scoped audience, got %d", device.ActiveSessionCount)
			}
		}
	}
	if currentCount != 1 || currentDeviceID == uuid.Nil || otherDeviceID == uuid.Nil {
		t.Fatalf("expected one current and one other device: current=%d currentID=%s otherID=%s", currentCount, currentDeviceID, otherDeviceID)
	}
	var successfulLoginEvents, knownFailureEvents, unknownFailureEvents int
	if err = pool.QueryRow(ctx, `
		SELECT count(*) FROM audit.login_events
		WHERE tenant_id = $1 AND user_id = $2 AND result = 'success'
		  AND login_identifier_hash IS NULL AND login_identifier_hint IS NULL
		  AND device_info ? 'platform' AND device_info ? 'registered'
		  AND device_info::text NOT LIKE '%' || $3 || '%'
	`, tenant.ID, user.ID, otherDeviceKey).Scan(&successfulLoginEvents); err != nil {
		t.Fatalf("count successful login events: %v", err)
	}
	if successfulLoginEvents != 3 {
		t.Fatalf("expected three redacted successful login events, got %d", successfulLoginEvents)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.login_events WHERE request_id=$1 AND user_id=$2 AND tenant_id=$3 AND result='failure' AND failure_reason='invalid_credentials' AND login_identifier_hash IS NULL AND login_identifier_hint IS NULL`, "known-login-failure-"+suffix, user.ID, tenant.ID).Scan(&knownFailureEvents); err != nil {
		t.Fatalf("count known failed login event: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.login_events WHERE request_id=$1 AND user_id IS NULL AND result='failure' AND failure_reason='invalid_credentials' AND login_identifier_hash IS NULL AND login_identifier_hint IS NULL`, "unknown-login-failure-"+suffix).Scan(&unknownFailureEvents); err != nil {
		t.Fatalf("count unknown failed login event: %v", err)
	}
	if knownFailureEvents != 1 || unknownFailureEvents != 1 {
		t.Fatalf("expected one redacted known and unknown failure event, got known=%d unknown=%d", knownFailureEvents, unknownFailureEvents)
	}
	foreignDevices, err := service.ListSelfDevices(ctx, foreign.AccessToken, "ak-admin")
	if err != nil || len(foreignDevices) != 1 {
		t.Fatalf("list foreign devices: count=%d err=%v", len(foreignDevices), err)
	}
	if _, err = service.RemoveSelfDevice(ctx, current.AccessToken, "ak-admin", application.RemoveSelfDeviceInput{
		DeviceID: foreignDevices[0].ID, RequestID: "foreign-device-remove-" + suffix,
	}); !errors.Is(err, domain.ErrDeviceNotFound) {
		t.Fatalf("foreign device must remain outside self scope, got %v", err)
	}
	if _, contextErr := service.Context(ctx, foreign.AccessToken, "ak-admin"); contextErr != nil {
		t.Fatalf("foreign device session must remain active: %v", contextErr)
	}

	requestID := "device-remove-" + suffix
	removedCurrent, err := service.RemoveSelfDevice(ctx, current.AccessToken, "ak-admin", application.RemoveSelfDeviceInput{
		DeviceID: otherDeviceID, RequestID: requestID, Client: application.ClientMetadata{UserAgent: "Device Test"},
	})
	if err != nil || removedCurrent {
		t.Fatalf("remove other device: current=%t err=%v", removedCurrent, err)
	}
	if _, contextErr := service.Context(ctx, other.AccessToken, "ak-admin"); !errors.Is(contextErr, application.ErrInvalidAccessToken) {
		t.Fatalf("removed device admin access token must be invalid, got %v", contextErr)
	}
	if _, contextErr := service.Context(ctx, otherAPI.AccessToken, "ak-api"); !errors.Is(contextErr, application.ErrInvalidAccessToken) {
		t.Fatalf("removed device API access token must be invalid, got %v", contextErr)
	}
	if _, refreshErr := service.Refresh(ctx, other.RefreshToken, "ak-admin", application.ClientMetadata{}); !errors.Is(refreshErr, domain.ErrRefreshInvalid) {
		t.Fatalf("removed device refresh family must be invalid, got %v", refreshErr)
	}
	var deviceCount, auditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM iam.devices WHERE id = $1 AND user_id = $2`, otherDeviceID, user.ID).Scan(&deviceCount); err != nil {
		t.Fatalf("count removed device: %v", err)
	}
	if deviceCount != 0 {
		t.Fatalf("expected removed device row to be deleted, got %d", deviceCount)
	}
	if err = pool.QueryRow(ctx, `
		SELECT count(*) FROM audit.operation_logs
		WHERE user_id = $1 AND tenant_id = $2 AND request_id = $3
		  AND action_name = 'iam.me.device.remove' AND resource_id = $4 AND succeeded = true
		  AND (after_data->>'revoked_session_count')::integer = 2
	`, user.ID, tenant.ID, requestID, otherDeviceID.String()).Scan(&auditCount); err != nil {
		t.Fatalf("count device remove audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one device removal audit record, got %d", auditCount)
	}
	removedCurrent, err = service.RemoveSelfDevice(ctx, current.AccessToken, "ak-admin", application.RemoveSelfDeviceInput{
		DeviceID: currentDeviceID, RequestID: "current-device-remove-" + suffix,
	})
	if err != nil || !removedCurrent {
		t.Fatalf("remove current device: current=%t err=%v", removedCurrent, err)
	}
	if _, contextErr := service.Context(ctx, current.AccessToken, "ak-admin"); !errors.Is(contextErr, application.ErrInvalidAccessToken) {
		t.Fatalf("removed current device token must be invalid, got %v", contextErr)
	}
}

func TestSelfPasswordChangeRevokesOtherSessionsAndRejectsReuse(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer pool.Close()

	oldPassword := "old session password 2026"
	newPassword := "new session password 2026"
	passwordHash, err := application.HashPassword(oldPassword)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	suffix := uuid.NewString()
	repo := repository.NewPostgres(pool)
	user, tenant, err := repo.CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: "password-" + suffix, TenantName: "Password Tenant",
		Email: "password-" + suffix + "@example.test", DisplayName: "Password User",
		Locale: "zh-CN", PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	issuer, err := application.NewDevelopmentTokenIssuer()
	if err != nil {
		t.Fatalf("create token issuer: %v", err)
	}
	service, err := application.NewAuthService(repo, repo, issuer)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	email := "password-" + suffix + "@example.test"
	current, err := service.Login(ctx, application.LoginInput{Email: email, Password: oldPassword, Audience: "ak-admin"})
	if err != nil {
		t.Fatalf("current login: %v", err)
	}
	other, err := service.Login(ctx, application.LoginInput{Email: email, Password: oldPassword, Audience: "ak-admin"})
	if err != nil {
		t.Fatalf("other login: %v", err)
	}
	if err = service.ChangeSelfPassword(ctx, current.AccessToken, "ak-admin", application.ChangeSelfPasswordInput{
		CurrentPassword: "wrong current password", NewPassword: newPassword, RequestID: "wrong-" + suffix,
	}); !errors.Is(err, application.ErrCurrentPassword) {
		t.Fatalf("expected invalid current password error, got %v", err)
	}
	if err = service.ChangeSelfPassword(ctx, current.AccessToken, "ak-admin", application.ChangeSelfPasswordInput{
		CurrentPassword: oldPassword, NewPassword: oldPassword, RequestID: "same-" + suffix,
	}); !errors.Is(err, application.ErrPasswordReused) {
		t.Fatalf("expected current password reuse error, got %v", err)
	}
	requestID := "password-change-" + suffix
	if err = service.ChangeSelfPassword(ctx, current.AccessToken, "ak-admin", application.ChangeSelfPasswordInput{
		CurrentPassword: oldPassword, NewPassword: newPassword, RequestID: requestID,
	}); err != nil {
		t.Fatalf("change self password: %v", err)
	}
	if _, contextErr := service.Context(ctx, current.AccessToken, "ak-admin"); contextErr != nil {
		t.Fatalf("current session must remain active: %v", contextErr)
	}
	if _, contextErr := service.Context(ctx, other.AccessToken, "ak-admin"); !errors.Is(contextErr, application.ErrInvalidAccessToken) {
		t.Fatalf("other access token must be invalid, got %v", contextErr)
	}
	if _, refreshErr := service.Refresh(ctx, other.RefreshToken, "ak-admin", application.ClientMetadata{}); !errors.Is(refreshErr, domain.ErrRefreshInvalid) {
		t.Fatalf("other refresh family must be invalid, got %v", refreshErr)
	}
	if _, loginErr := service.Login(ctx, application.LoginInput{Email: email, Password: oldPassword, Audience: "ak-admin"}); !errors.Is(loginErr, application.ErrInvalidCredentials) {
		t.Fatalf("old password must stop authenticating, got %v", loginErr)
	}
	if _, loginErr := service.Login(ctx, application.LoginInput{Email: email, Password: newPassword, Audience: "ak-admin"}); loginErr != nil {
		t.Fatalf("new password must authenticate: %v", loginErr)
	}
	if err = service.ChangeSelfPassword(ctx, current.AccessToken, "ak-admin", application.ChangeSelfPasswordInput{
		CurrentPassword: newPassword, NewPassword: oldPassword, RequestID: "reuse-" + suffix,
	}); !errors.Is(err, application.ErrPasswordReused) {
		t.Fatalf("recent password reuse must be rejected, got %v", err)
	}

	var auditCount, historyCount int
	if err = pool.QueryRow(ctx, `
		SELECT count(*) FROM audit.operation_logs
		WHERE user_id = $1 AND tenant_id = $2 AND request_id = $3
		  AND action_name = 'iam.me.password.change' AND succeeded = true
		  AND before_data->>'password_version' = '1'
		  AND after_data->>'password_version' = '2'
		  AND (after_data->>'other_sessions_revoked')::integer >= 1
		  AND before_data::text NOT LIKE '%password_hash%'
		  AND after_data::text NOT LIKE '%password_hash%'
	`, user.ID, tenant.ID, requestID).Scan(&auditCount); err != nil {
		t.Fatalf("count password change audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one redacted password change audit, got %d", auditCount)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM iam.password_history WHERE user_id = $1`, user.ID).Scan(&historyCount); err != nil {
		t.Fatalf("count password history: %v", err)
	}
	if historyCount != 1 {
		t.Fatalf("expected one password history entry, got %d", historyCount)
	}
}

type capturePasswordResetNotifier struct {
	notifications []application.PasswordResetNotification
}

func (notifier *capturePasswordResetNotifier) SendPasswordReset(
	_ context.Context,
	notification application.PasswordResetNotification,
) error {
	notifier.notifications = append(notifier.notifications, notification)
	return nil
}

func TestAnonymousRegistrationAndPasswordRecoveryArePrivateOneTimeAndTransactional(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer pool.Close()

	suffix := uuid.NewString()
	repo := repository.NewPostgres(pool)
	oldPassword := "recovery old password 2026!"
	newPassword := "recovery new password 2026!"
	oldHash, err := application.HashPassword(oldPassword)
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}
	ownerEmail := "recovery-owner-" + suffix + "@example.test"
	owner, tenant, err := repo.CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: "recovery-" + suffix, TenantName: "Recovery Tenant",
		Email: ownerEmail, DisplayName: "Recovery Owner", Locale: "en-US", PasswordHash: oldHash,
	})
	if err != nil {
		t.Fatalf("create recovery identity: %v", err)
	}
	if _, err = pool.Exec(ctx, `
		INSERT INTO iam.roles (tenant_id, code, name, role_type, data_scope, is_default, status)
		VALUES ($1, 'member', 'Member', 'system', 'self', true, 'active')
	`, tenant.ID); err != nil {
		t.Fatalf("create registration member role: %v", err)
	}

	notifier := &capturePasswordResetNotifier{}
	issuer, err := application.NewDevelopmentTokenIssuer()
	if err != nil {
		t.Fatalf("create token issuer: %v", err)
	}
	service, err := application.NewAuthService(repo, repo, issuer, application.WithAnonymousAuth(
		application.AnonymousAuthConfig{
			AdminRegistrationEnabled: true, RegistrationTenantCode: tenant.Code,
			PasswordRecoveryEnabled: true,
		}, notifier,
	))
	if err != nil {
		t.Fatalf("create anonymous auth service: %v", err)
	}

	registeredEmail := "registered-" + suffix + "@example.test"
	registration := application.RegisterInput{
		Email: registeredEmail, DisplayName: "Registered Member",
		Password: "registered password 2026!", Locale: "zh-CN", AcceptTerms: true,
		RequestID: "register-" + suffix,
	}
	if err = service.Register(ctx, registration); err != nil {
		t.Fatalf("register admin member: %v", err)
	}
	if err = service.Register(ctx, registration); err != nil {
		t.Fatalf("duplicate registration must preserve the generic accepted result: %v", err)
	}
	var registeredCount, membershipCount, registrationAuditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM iam.users WHERE email = $1`, registeredEmail).Scan(&registeredCount); err != nil {
		t.Fatalf("count registered user: %v", err)
	}
	if err = pool.QueryRow(ctx, `
		SELECT count(*)
		FROM iam.tenant_members tm
		JOIN iam.user_roles ur ON ur.tenant_id = tm.tenant_id AND ur.user_id = tm.user_id
		JOIN iam.roles r ON r.id = ur.role_id
		JOIN iam.users u ON u.id = tm.user_id
		WHERE u.email = $1 AND tm.tenant_id = $2 AND r.code = 'member'
	`, registeredEmail, tenant.ID).Scan(&membershipCount); err != nil {
		t.Fatalf("count registered membership: %v", err)
	}
	if err = pool.QueryRow(ctx, `
		SELECT count(*) FROM audit.operation_logs
		WHERE tenant_id = $1 AND request_id = $2 AND action_name = 'iam.auth.register'
		  AND before_data::text NOT LIKE '%password%' AND after_data::text NOT LIKE '%password%'
	`, tenant.ID, registration.RequestID).Scan(&registrationAuditCount); err != nil {
		t.Fatalf("count registration audit: %v", err)
	}
	if registeredCount != 1 || membershipCount != 1 || registrationAuditCount != 1 {
		t.Fatalf("unexpected registration persistence: users=%d memberships=%d audits=%d", registeredCount, membershipCount, registrationAuditCount)
	}

	firstSession, err := service.Login(ctx, application.LoginInput{Email: ownerEmail, Password: oldPassword, Audience: "ak-admin"})
	if err != nil {
		t.Fatalf("create first pre-reset session: %v", err)
	}
	secondSession, err := service.Login(ctx, application.LoginInput{Email: ownerEmail, Password: oldPassword, Audience: "ak-api"})
	if err != nil {
		t.Fatalf("create second pre-reset session: %v", err)
	}
	unknownRetry, err := service.ForgotPassword(ctx, application.ForgotPasswordInput{Email: "unknown-" + suffix + "@example.test"})
	if err != nil {
		t.Fatalf("unknown forgot password: %v", err)
	}
	knownRetry, err := service.ForgotPassword(ctx, application.ForgotPasswordInput{Email: ownerEmail})
	if err != nil {
		t.Fatalf("known forgot password: %v", err)
	}
	if unknownRetry != knownRetry || knownRetry != 60 || len(notifier.notifications) != 1 {
		t.Fatalf("forgot response leaked identity or delivery count is wrong: unknown=%d known=%d deliveries=%d", unknownRetry, knownRetry, len(notifier.notifications))
	}
	notification := notifier.notifications[0]
	if notification.Email != ownerEmail || notification.Locale != "en-US" || notification.Token == "" {
		t.Fatalf("unexpected recovery notification: %#v", notification)
	}
	if _, err = service.ForgotPassword(ctx, application.ForgotPasswordInput{Email: ownerEmail}); err != nil {
		t.Fatalf("cooldown forgot password: %v", err)
	}
	if len(notifier.notifications) != 1 {
		t.Fatalf("cooldown must suppress a second delivery, got %d", len(notifier.notifications))
	}

	if err = service.ResetPassword(ctx, application.ResetPasswordInput{
		Token: notification.Token, NewPassword: oldPassword, RequestID: "reuse-reset-" + suffix,
	}); !errors.Is(err, application.ErrPasswordReused) {
		t.Fatalf("expected reset password reuse rejection, got %v", err)
	}
	resetRequestID := "password-reset-" + suffix
	if err = service.ResetPassword(ctx, application.ResetPasswordInput{
		Token: notification.Token, NewPassword: newPassword, RequestID: resetRequestID,
	}); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if err = service.ResetPassword(ctx, application.ResetPasswordInput{
		Token: notification.Token, NewPassword: "another recovery password 2026!",
	}); !errors.Is(err, application.ErrResetTokenInvalid) {
		t.Fatalf("consumed token must be rejected, got %v", err)
	}
	if _, contextErr := service.Context(ctx, firstSession.AccessToken, "ak-admin"); !errors.Is(contextErr, application.ErrInvalidAccessToken) {
		t.Fatalf("admin access token must be revoked, got %v", contextErr)
	}
	if _, contextErr := service.Context(ctx, secondSession.AccessToken, "ak-api"); !errors.Is(contextErr, application.ErrInvalidAccessToken) {
		t.Fatalf("api access token must be revoked, got %v", contextErr)
	}
	if _, refreshErr := service.Refresh(ctx, firstSession.RefreshToken, "ak-admin", application.ClientMetadata{}); !errors.Is(refreshErr, domain.ErrRefreshInvalid) {
		t.Fatalf("refresh family must be revoked, got %v", refreshErr)
	}
	if _, loginErr := service.Login(ctx, application.LoginInput{Email: ownerEmail, Password: oldPassword, Audience: "ak-admin"}); !errors.Is(loginErr, application.ErrInvalidCredentials) {
		t.Fatalf("old password must fail after reset, got %v", loginErr)
	}
	if _, loginErr := service.Login(ctx, application.LoginInput{Email: ownerEmail, Password: newPassword, Audience: "ak-admin"}); loginErr != nil {
		t.Fatalf("new password must authenticate after reset: %v", loginErr)
	}

	var consumedCount, resetAuditCount, rawTokenCount int
	if err = pool.QueryRow(ctx, `
		SELECT count(*) FROM iam.verification_challenges
		WHERE user_id = $1 AND challenge_type = 'password_reset'
		  AND consumed_at IS NOT NULL AND octet_length(secret_hash) = 32 AND octet_length(target_hash) = 32
	`, owner.ID).Scan(&consumedCount); err != nil {
		t.Fatalf("count consumed challenges: %v", err)
	}
	if err = pool.QueryRow(ctx, `
		SELECT count(*) FROM audit.operation_logs
		WHERE user_id = $1 AND tenant_id = $2 AND request_id = $3
		  AND action_name = 'iam.auth.password.reset' AND succeeded = true
		  AND (after_data->>'sessions_revoked')::integer = 2
		  AND before_data::text NOT LIKE '%password_hash%'
		  AND after_data::text NOT LIKE '%password_hash%'
	`, owner.ID, tenant.ID, resetRequestID).Scan(&resetAuditCount); err != nil {
		t.Fatalf("count password reset audit: %v", err)
	}
	if err = pool.QueryRow(ctx, `
		SELECT count(*) FROM iam.verification_challenges
		WHERE metadata::text LIKE '%' || $1 || '%'
	`, notification.Token).Scan(&rawTokenCount); err != nil {
		t.Fatalf("scan for raw reset token: %v", err)
	}
	if consumedCount != 1 || resetAuditCount != 1 || rawTokenCount != 0 {
		t.Fatalf("unexpected reset persistence: consumed=%d audits=%d raw_tokens=%d", consumedCount, resetAuditCount, rawTokenCount)
	}
}

func TestSwitchTenantIssuesNewScopedSessionAndRevokesPreviousSession(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer pool.Close()

	suffix := uuid.NewString()
	password := "switch tenant integration password 2026!"
	hash, err := application.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repo := repository.NewPostgres(pool)
	user, source, err := repo.CreateIdentity(ctx, domain.CreateIdentity{TenantCode: "switch-source-" + suffix, TenantName: "Switch Source", Email: "switch-" + suffix + "@example.test", DisplayName: "Switch User", Locale: "zh-CN", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	var targetID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name,status,settings) VALUES($1,'Switch Target','active','{}') RETURNING id`, "switch-target-"+suffix).Scan(&targetID); err != nil {
		t.Fatalf("create target tenant: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status,invited_by) VALUES($1,$2,'active',$2)`, targetID, user.ID); err != nil {
		t.Fatalf("create target membership: %v", err)
	}
	issuer, err := application.NewDevelopmentTokenIssuer()
	if err != nil {
		t.Fatalf("create issuer: %v", err)
	}
	service, err := application.NewAuthService(repo, repo, issuer)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	login, err := service.Login(ctx, application.LoginInput{Email: "switch-" + suffix + "@example.test", Password: password, Audience: "ak-admin"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	contextBefore, err := service.Context(ctx, login.AccessToken, "ak-admin")
	if err != nil || contextBefore.Tenant.ID != source.ID || len(contextBefore.AvailableTenants) != 2 {
		t.Fatalf("context before = %#v, error = %v", contextBefore, err)
	}

	switched, err := service.SwitchTenant(ctx, login.AccessToken, application.SwitchTenantInput{TenantID: targetID, Audience: "ak-admin", Client: application.ClientMetadata{RequestID: "switch-" + suffix}})
	if err != nil {
		t.Fatalf("switch tenant: %v", err)
	}
	contextAfter, err := service.Context(ctx, switched.AccessToken, "ak-admin")
	if err != nil || contextAfter.Tenant.ID != targetID {
		t.Fatalf("context after tenant = %s, error = %v", contextAfter.Tenant.ID, err)
	}
	if _, err = service.Context(ctx, login.AccessToken, "ak-admin"); !errors.Is(err, application.ErrInvalidAccessToken) {
		t.Fatalf("old access token error = %v", err)
	}
	if _, err = service.Refresh(ctx, login.RefreshToken, "ak-admin", application.ClientMetadata{}); !errors.Is(err, domain.ErrRefreshInvalid) {
		t.Fatalf("old refresh token error = %v", err)
	}
	if _, err = service.SwitchTenant(ctx, switched.AccessToken, application.SwitchTenantInput{TenantID: uuid.New(), Audience: "ak-admin"}); !errors.Is(err, application.ErrTenantUnavailable) {
		t.Fatalf("foreign tenant switch error = %v", err)
	}

	var activeOld, activeNew, switchEvents int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM iam.sessions WHERE user_id=$1 AND tenant_id=$2 AND status='active'`, user.ID, source.ID).Scan(&activeOld); err != nil {
		t.Fatalf("count old sessions: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM iam.sessions WHERE user_id=$1 AND tenant_id=$2 AND status='active'`, user.ID, targetID).Scan(&activeNew); err != nil {
		t.Fatalf("count new sessions: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.login_events WHERE user_id=$1 AND tenant_id=$2 AND session_id=$3 AND auth_method='tenant_switch' AND result='success'`, user.ID, targetID, switched.SessionID).Scan(&switchEvents); err != nil {
		t.Fatalf("count tenant switch events: %v", err)
	}
	if activeOld != 0 || activeNew != 1 {
		t.Fatalf("active sessions old=%d new=%d", activeOld, activeNew)
	}
	if switchEvents != 1 {
		t.Fatalf("tenant switch events=%d, want 1", switchEvents)
	}
}
