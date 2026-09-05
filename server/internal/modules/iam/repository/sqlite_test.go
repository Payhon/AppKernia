package repository_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepository "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	platformsqlite "github.com/appkernia/appkernia/server/internal/platform/sqlite"
	"github.com/google/uuid"
	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestSQLiteBootstrapAndAuthenticationPersistence(t *testing.T) {
	ctx := context.Background()
	database, err := platformsqlite.Open(ctx, filepath.Join(t.TempDir(), "data", "appkernia.db"))
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	repository := iamrepository.NewSQLite(database)

	bootstrap := iamrepository.BootstrapAdminInput{
		TenantCode: "platform", TenantName: "Platform", Email: "admin@example.com",
		DisplayName: "Administrator", Locale: "zh-CN", Password: "correct horse battery staple",
	}
	result, err := repository.BootstrapAdmin(ctx, bootstrap)
	if err != nil {
		t.Fatalf("bootstrap administrator: %v", err)
	}
	if result.GrantedPermissions != 7 || result.GrantedMenus != 1 {
		t.Fatalf("bootstrap grants = permissions:%d menus:%d", result.GrantedPermissions, result.GrantedMenus)
	}
	again, err := repository.BootstrapAdmin(ctx, bootstrap)
	if err != nil {
		t.Fatalf("repeat bootstrap administrator: %v", err)
	}
	if again.User.ID != result.User.ID || again.Tenant.ID != result.Tenant.ID || again.GrantedPermissions != 0 || again.GrantedMenus != 0 {
		t.Fatalf("bootstrap is not idempotent: first=%#v second=%#v", result, again)
	}

	credential, err := repository.FindCredentialByEmail(ctx, " ADMIN@example.com ")
	if err != nil || !application.VerifyPassword(credential.PasswordHash, bootstrap.Password) {
		t.Fatalf("find bootstrap credential: err=%v valid=%t", err, application.VerifyPassword(credential.PasswordHash, bootstrap.Password))
	}
	authContext, err := repository.GetAuthContext(ctx, result.User.ID, result.Tenant.ID)
	if err != nil {
		t.Fatalf("get auth context: %v", err)
	}
	if len(authContext.Roles) != 1 || authContext.Roles[0] != "super-admin" || len(authContext.Permissions) != 7 || len(authContext.Menus) != 1 || authContext.Menus[0].Code != "dashboard" {
		t.Fatalf("unexpected auth context: %#v", authContext)
	}

	now := time.Now().UTC()
	firstHash := sha256.Sum256([]byte("first refresh token"))
	first, err := repository.CreateSession(ctx, domain.CreateSession{
		UserID: result.User.ID, TenantID: result.Tenant.ID, Audience: "ak-admin", AuthMethod: "password",
		RefreshTokenHash: firstHash[:], AbsoluteExpiresAt: now.Add(24 * time.Hour), IdleExpiresAt: now.Add(time.Hour),
		RefreshExpiresAt: now.Add(24 * time.Hour), UserAgent: "sqlite-test", DeviceKey: "browser-one",
	})
	if err != nil {
		t.Fatalf("create first session: %v", err)
	}
	if err = repository.ValidateSession(ctx, first); err != nil {
		t.Fatalf("validate first session: %v", err)
	}
	newDisplayName := "SQLite Administrator"
	updated, err := repository.UpdateSelfProfile(ctx, domain.UpdateSelfProfile{
		UserID: result.User.ID, TenantID: result.Tenant.ID, SessionID: first.ID, DisplayName: &newDisplayName,
	})
	if err != nil || updated.DisplayName != newDisplayName {
		t.Fatalf("update self profile: user=%#v err=%v", updated, err)
	}

	secondHash := sha256.Sum256([]byte("second refresh token"))
	second, err := repository.CreateSession(ctx, domain.CreateSession{
		UserID: result.User.ID, TenantID: result.Tenant.ID, Audience: "ak-admin", AuthMethod: "password",
		RefreshTokenHash: secondHash[:], AbsoluteExpiresAt: now.Add(24 * time.Hour), IdleExpiresAt: now.Add(time.Hour),
		RefreshExpiresAt: now.Add(24 * time.Hour), UserAgent: "sqlite-test-two", DeviceKey: "browser-two",
	})
	if err != nil {
		t.Fatalf("create second session: %v", err)
	}
	sessions, err := repository.ListSelfSessions(ctx, domain.SelfSessionScope{UserID: result.User.ID, TenantID: result.Tenant.ID, Audience: "ak-admin"})
	if err != nil || len(sessions) != 2 {
		t.Fatalf("list self sessions: count=%d err=%v", len(sessions), err)
	}
	devices, err := repository.ListSelfDevices(ctx, domain.SelfDeviceScope{
		UserID: result.User.ID, TenantID: result.Tenant.ID, SessionID: first.ID, Audience: "ak-admin",
	})
	if err != nil || len(devices) != 2 || !devices[0].Current {
		t.Fatalf("list self devices: devices=%#v err=%v", devices, err)
	}
	thirdHash := sha256.Sum256([]byte("third refresh token"))
	rotated, err := repository.RotateRefreshToken(ctx, firstHash[:], thirdHash[:], nil)
	if err != nil || rotated.ID != first.ID {
		t.Fatalf("rotate refresh token: session=%#v err=%v", rotated, err)
	}
	if _, err = repository.RotateRefreshToken(ctx, firstHash[:], sha256Bytes("fourth refresh token"), nil); !errors.Is(err, domain.ErrRefreshReused) {
		t.Fatalf("reuse old refresh token error = %v", err)
	}
	if err = repository.ValidateSession(ctx, first); !errors.Is(err, domain.ErrRefreshInvalid) {
		t.Fatalf("reused session validation error = %v", err)
	}

	passwordState, err := repository.GetSelfPasswordState(ctx, result.User.ID)
	if err != nil {
		t.Fatalf("get password state: %v", err)
	}
	newPasswordHash, err := application.HashPassword("an entirely new secure password")
	if err != nil {
		t.Fatalf("hash new password: %v", err)
	}
	if err = repository.ChangeSelfPassword(ctx, domain.ChangeSelfPassword{
		UserID: result.User.ID, TenantID: result.Tenant.ID, SessionID: second.ID,
		ExpectedHash: passwordState.CurrentHash, ExpectedVersion: passwordState.CurrentVersion, NewHash: newPasswordHash,
	}); err != nil {
		t.Fatalf("change self password: %v", err)
	}
	if err = repository.ValidateSession(ctx, second); err != nil {
		t.Fatalf("current session should survive password change: %v", err)
	}
	passwordState, err = repository.GetSelfPasswordState(ctx, result.User.ID)
	if err != nil || passwordState.CurrentVersion != 2 || len(passwordState.HistoryHashes) != 1 {
		t.Fatalf("updated password state = %#v err=%v", passwordState, err)
	}
	devices, err = repository.ListSelfDevices(ctx, domain.SelfDeviceScope{
		UserID: result.User.ID, TenantID: result.Tenant.ID, SessionID: second.ID, Audience: "ak-admin",
	})
	if err != nil {
		t.Fatalf("list devices before removal: %v", err)
	}
	var currentDeviceID uuid.UUID
	for _, device := range devices {
		if device.Current {
			currentDeviceID = device.ID
		}
	}
	removedCurrent, err := repository.RemoveSelfDevice(ctx, domain.RemoveSelfDevice{
		SelfDeviceScope: domain.SelfDeviceScope{UserID: result.User.ID, TenantID: result.Tenant.ID, SessionID: second.ID, Audience: "ak-admin"},
		DeviceID:        currentDeviceID,
	})
	if err != nil || !removedCurrent {
		t.Fatalf("remove current device: current=%t err=%v", removedCurrent, err)
	}
	if err = repository.ValidateSession(ctx, second); !errors.Is(err, domain.ErrRefreshInvalid) {
		t.Fatalf("removed-device session validation error = %v", err)
	}

	scopeHash := sha256.Sum256([]byte("login scope"))
	for index := 0; index < 3; index++ {
		_, err = repository.RecordLoginFailure(ctx, domain.LoginFailure{
			UserID: &result.User.ID, Audience: "ak-admin", ScopeHash: scopeHash[:],
			FailedAt: now.Add(time.Duration(index) * time.Second), ExpiresAt: now.Add(15 * time.Minute),
		})
		if err != nil {
			t.Fatalf("record login failure %d: %v", index+1, err)
		}
	}
	required, err := repository.LoginCaptchaRequired(ctx, scopeHash[:], now.Add(4*time.Second))
	if err != nil || !required {
		t.Fatalf("login captcha required = %t err=%v", required, err)
	}
	proofHash := sha256.Sum256([]byte("captcha proof"))
	challengeID := uuid.New()
	if _, err = repository.CreateLoginCaptcha(ctx, domain.LoginCaptchaChallenge{
		ID: challengeID, ScopeHash: scopeHash[:], CaptchaType: "slide", ProofHash: proofHash[:],
		CreatedAt: now.Add(4 * time.Second), ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("create login captcha: %v", err)
	}
	if err = repository.VerifyLoginCaptcha(ctx, domain.LoginCaptchaAttempt{
		ID: challengeID, ScopeHash: scopeHash[:], CaptchaType: "slide", ProofHash: proofHash[:], Valid: true, Now: now.Add(5 * time.Second),
	}); err != nil {
		t.Fatalf("verify login captcha: %v", err)
	}

	resetSessionHash := sha256.Sum256([]byte("reset session refresh"))
	resetSession, err := repository.CreateSession(ctx, domain.CreateSession{
		UserID: result.User.ID, TenantID: result.Tenant.ID, Audience: "ak-admin", RefreshTokenHash: resetSessionHash[:],
		AbsoluteExpiresAt: now.Add(24 * time.Hour), IdleExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create password-reset session: %v", err)
	}
	targetHash, secretHash := sha256.Sum256([]byte("reset target")), sha256.Sum256([]byte("reset secret"))
	recipient, err := repository.PreparePasswordReset(ctx, domain.PreparePasswordReset{
		Email: bootstrap.Email, TargetHash: targetHash[:], SecretHash: secretHash[:], ExpiresAt: now.Add(time.Hour),
	})
	if err != nil || recipient == nil || recipient.TenantID != result.Tenant.ID {
		t.Fatalf("prepare password reset: recipient=%#v err=%v", recipient, err)
	}
	resetState, err := repository.GetPasswordResetState(ctx, secretHash[:])
	if err != nil || resetState.CurrentVersion != 2 {
		t.Fatalf("get password reset state: state=%#v err=%v", resetState, err)
	}
	resetPasswordHash, err := application.HashPassword("one final replacement password")
	if err != nil {
		t.Fatalf("hash reset password: %v", err)
	}
	if err = repository.ResetPassword(ctx, domain.ResetPassword{
		TokenHash: secretHash[:], UserID: result.User.ID, ExpectedHash: resetState.CurrentHash,
		ExpectedVersion: resetState.CurrentVersion, NewHash: resetPasswordHash,
	}); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if err = repository.ValidateSession(ctx, resetSession); !errors.Is(err, domain.ErrRefreshInvalid) {
		t.Fatalf("password-reset session validation error = %v", err)
	}
}

func sha256Bytes(value string) []byte {
	hash := sha256.Sum256([]byte(value))
	return hash[:]
}
