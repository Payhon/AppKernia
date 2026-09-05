package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	platformcaptcha "github.com/appkernia/appkernia/server/internal/platform/captcha"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrAudienceMismatch       = errors.New("token audience mismatch")
	ErrAccessDenied           = errors.New("access denied")
	ErrProfileValidation      = errors.New("profile validation failed")
	ErrSessionValidation      = errors.New("session validation failed")
	ErrPasswordValidation     = errors.New("password validation failed")
	ErrCurrentPassword        = errors.New("current password is invalid")
	ErrPasswordReused         = errors.New("password was used recently")
	ErrDeviceValidation       = errors.New("device validation failed")
	ErrFeatureDisabled        = errors.New("feature is disabled")
	ErrRegistrationValidation = errors.New("registration validation failed")
	ErrResetTokenInvalid      = errors.New("password reset token is invalid")
	ErrTenantUnavailable      = errors.New("tenant is unavailable to this user")
	ErrCaptchaRequired        = errors.New("login captcha is required")
	ErrCaptchaInvalid         = errors.New("login captcha is invalid")
	ErrCaptchaNotRequired     = errors.New("login captcha is not required")
	ErrCaptchaCooldown        = errors.New("login captcha generation is cooling down")
)

type ClientMetadata struct {
	IPAddress *netip.Addr
	UserAgent string
	DeviceKey string
	RequestID string
}

type appProfileIdentifierRepository interface {
	AppProfileIdentifiers(context.Context, uuid.UUID, uuid.UUID) (string, string, error)
}

type UpdateSelfProfileInput struct {
	DisplayName *string
	Locale      *string
	TimeZone    *string
	RequestID   string
	Client      ClientMetadata
}

type SelfSession struct {
	domain.SelfSession
	Current bool
}

type RevokeSelfSessionInput struct {
	SessionID uuid.UUID
	RequestID string
	Client    ClientMetadata
}

type ChangeSelfPasswordInput struct {
	CurrentPassword string
	NewPassword     string
	RequestID       string
	Client          ClientMetadata
}

type SelfDevice struct {
	domain.SelfDevice
	Current bool
}

type RemoveSelfDeviceInput struct {
	DeviceID  uuid.UUID
	RequestID string
	Client    ClientMetadata
}

type LoginInput struct {
	Email    string
	Password string
	Audience string
	AppID    *uuid.UUID
	Client   ClientMetadata
	Captcha  *LoginCaptchaInput
}

type SwitchTenantInput struct {
	TenantID uuid.UUID
	Audience string
	Client   ClientMetadata
}

type SessionTokens struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	SessionID             uuid.UUID
	AppID                 *uuid.UUID
}

// PreparedMobileSession is a signed, opaque-token session envelope whose
// persistence is intentionally delegated to the caller's transaction. It is
// used by federated registration so user, membership, external identity,
// session, refresh hash and audit rows commit atomically.
type PreparedMobileSession struct {
	Tokens             SessionTokens
	RefreshTokenHash   []byte
	AbsoluteExpiresAt  time.Time
	IdleExpiresAt      time.Time
	RefreshExpiresAt   time.Time
	IPAddress          *netip.Addr
	UserAgent          string
	DeviceKey          string
	RequestID          string
	AccessTokenVersion int32
}

type appCredentialRepository interface {
	FindCredentialByAppIdentifier(context.Context, uuid.UUID, string, string) (domain.Credential, error)
}

type AuthService struct {
	identities               domain.Repository
	sessions                 domain.SessionRepository
	issuer                   *TokenIssuer
	clock                    func() time.Time
	dummyPasswordHash        string
	anonymous                AnonymousAuthConfig
	resetNotifier            PasswordResetNotifier
	loginProtectionKey       []byte
	loginCaptcha             *platformcaptcha.Service
	loginCaptchaCodec        *platformcaptcha.Codec
	loginCaptchaTypeProvider func(context.Context) (platformcaptcha.Type, error)
}

func appIDValue(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}

func NewAuthService(identities domain.Repository, sessions domain.SessionRepository, issuer *TokenIssuer, options ...AuthOption) (*AuthService, error) {
	dummyPasswordHash, err := HashPassword("appkernia-dummy-password-never-used")
	if err != nil {
		return nil, fmt.Errorf("create login timing defense: %w", err)
	}
	loginProtectionKey := make([]byte, 32)
	if _, err = rand.Read(loginProtectionKey); err != nil {
		return nil, fmt.Errorf("create login protection key: %w", err)
	}
	service := &AuthService{
		identities: identities, sessions: sessions, issuer: issuer,
		clock: time.Now, dummyPasswordHash: dummyPasswordHash,
		resetNotifier:      disabledPasswordResetNotifier{},
		loginProtectionKey: loginProtectionKey,
	}
	for _, option := range options {
		option(service)
	}
	service.loginCaptcha, err = platformcaptcha.NewService()
	if err != nil {
		return nil, fmt.Errorf("initialize login captcha: %w", err)
	}
	service.loginCaptchaCodec, err = platformcaptcha.NewCodec(service.loginProtectionKey)
	if err != nil {
		return nil, fmt.Errorf("initialize login captcha proof: %w", err)
	}
	return service, nil
}

func (service *AuthService) Login(ctx context.Context, input LoginInput) (SessionTokens, error) {
	deviceKey, err := normalizeDeviceKey(input.Client.DeviceKey)
	if err != nil {
		return SessionTokens{}, err
	}
	now := service.clock().UTC()
	scopeHash := loginScopeHash(service.loginProtectionKey, input.Email, input.Audience, input.Client.IPAddress)
	adminLogin := input.Audience == "ak-admin"
	if adminLogin {
		captchaRequired, captchaErr := service.identities.LoginCaptchaRequired(ctx, scopeHash, now)
		if captchaErr != nil {
			return SessionTokens{}, fmt.Errorf("read login protection state: %w", captchaErr)
		}
		if captchaRequired {
			if err = service.verifyLoginCaptcha(ctx, input.Captcha, scopeHash, now); err != nil {
				return SessionTokens{}, err
			}
		}
	}
	credential, findErr := service.identities.FindCredentialByEmail(ctx, input.Email)
	if input.Audience == "ak-mobile" && input.AppID != nil && *input.AppID != uuid.Nil {
		if scoped, ok := service.identities.(appCredentialRepository); ok {
			credential, findErr = scoped.FindCredentialByAppIdentifier(ctx, *input.AppID, "email", input.Email)
		}
	}
	passwordHash := service.dummyPasswordHash
	if findErr == nil {
		passwordHash = credential.PasswordHash
	}
	passwordValid := VerifyPassword(passwordHash, input.Password)
	if findErr != nil || credential.User.Status != "active" || !passwordValid {
		var userID *uuid.UUID
		if findErr == nil {
			value := credential.User.ID
			userID = &value
		}
		failureCount, recordErr := service.identities.RecordLoginFailure(ctx, domain.LoginFailure{
			UserID: userID, AppID: input.AppID, Audience: input.Audience, RequestID: input.Client.RequestID,
			IPAddress: input.Client.IPAddress, UserAgent: input.Client.UserAgent,
			ScopeHash: scopeHash, FailedAt: now, ExpiresAt: now.Add(loginFailureWindow),
		})
		if adminLogin && recordErr == nil && failureCount >= loginCaptchaThreshold {
			return SessionTokens{}, ErrCaptchaRequired
		}
		return SessionTokens{}, ErrInvalidCredentials
	}
	if !validAudience(input.Audience) {
		return SessionTokens{}, ErrAudienceMismatch
	}
	var tenant domain.Tenant
	if input.Audience == "ak-mobile" && input.AppID != nil && *input.AppID != uuid.Nil {
		tenant, err = service.identities.ResolveActiveMobileAppMembership(ctx, *input.AppID, credential.User.ID)
		if err != nil {
			return SessionTokens{}, ErrInvalidCredentials
		}
	} else {
		tenants, tenantErr := service.identities.ListUserTenants(ctx, credential.User.ID)
		if tenantErr != nil || len(tenants) == 0 {
			return SessionTokens{}, ErrInvalidCredentials
		}
		tenant = tenants[0]
	}
	plainRefresh, refreshHash, err := NewOpaqueToken()
	if err != nil {
		return SessionTokens{}, err
	}
	if err = service.identities.ResetLoginFailures(ctx, scopeHash); err != nil {
		return SessionTokens{}, fmt.Errorf("reset login protection state: %w", err)
	}
	refreshExpiresAt := now.Add(30 * 24 * time.Hour)
	session, err := service.sessions.CreateSession(ctx, domain.CreateSession{
		UserID: credential.User.ID, TenantID: tenant.ID, AppID: input.AppID, Audience: input.Audience,
		RefreshTokenHash: refreshHash, AbsoluteExpiresAt: refreshExpiresAt,
		IdleExpiresAt: now.Add(7 * 24 * time.Hour), RefreshExpiresAt: refreshExpiresAt,
		IPAddress: input.Client.IPAddress, UserAgent: input.Client.UserAgent, DeviceKey: deviceKey,
		RequestID: input.Client.RequestID,
	})
	if err != nil {
		return SessionTokens{}, fmt.Errorf("create login session: %w", err)
	}
	accessToken, accessExpiresAt, err := service.issuer.IssueForApp(
		session.UserID, session.TenantID, session.ID, session.Audience, session.AccessTokenVersion, appIDValue(session.AppID),
	)
	if err != nil {
		_ = service.sessions.RevokeSession(ctx, session.ID, "access_token_issue_failed")
		return SessionTokens{}, err
	}
	return SessionTokens{
		AccessToken: accessToken, AccessTokenExpiresAt: accessExpiresAt,
		RefreshToken: plainRefresh, RefreshTokenExpiresAt: refreshExpiresAt, SessionID: session.ID, AppID: session.AppID,
	}, nil
}

// IssueMobileSession is the single token-issuance path for verified OAuth and
// OTP identities. Provider tokens and OTP values never enter the IAM session.
func (service *AuthService) IssueMobileSession(ctx context.Context, userID, appID uuid.UUID, authMethod string, client ClientMetadata) (SessionTokens, error) {
	if userID == uuid.Nil || appID == uuid.Nil || (authMethod != "oauth" && authMethod != "email_otp" && authMethod != "sms_otp") {
		return SessionTokens{}, ErrInvalidCredentials
	}
	deviceKey, err := normalizeDeviceKey(client.DeviceKey)
	if err != nil {
		return SessionTokens{}, err
	}
	tenant, err := service.identities.ResolveActiveMobileAppMembership(ctx, appID, userID)
	if err != nil {
		return SessionTokens{}, ErrInvalidCredentials
	}
	plainRefresh, refreshHash, err := NewOpaqueToken()
	if err != nil {
		return SessionTokens{}, err
	}
	now := service.clock().UTC()
	refreshExpiresAt := now.Add(30 * 24 * time.Hour)
	session, err := service.sessions.CreateSession(ctx, domain.CreateSession{
		UserID: userID, TenantID: tenant.ID, AppID: &appID, Audience: "ak-mobile", AuthMethod: authMethod,
		RefreshTokenHash: refreshHash, AbsoluteExpiresAt: refreshExpiresAt, IdleExpiresAt: now.Add(7 * 24 * time.Hour),
		RefreshExpiresAt: refreshExpiresAt, IPAddress: client.IPAddress, UserAgent: client.UserAgent,
		DeviceKey: deviceKey, RequestID: client.RequestID,
	})
	if err != nil {
		return SessionTokens{}, fmt.Errorf("create verified mobile session: %w", err)
	}
	accessToken, accessExpiresAt, err := service.issuer.IssueForApp(userID, tenant.ID, session.ID, "ak-mobile", session.AccessTokenVersion, appID)
	if err != nil {
		_ = service.sessions.RevokeSession(ctx, session.ID, "access_token_issue_failed")
		return SessionTokens{}, err
	}
	return SessionTokens{AccessToken: accessToken, AccessTokenExpiresAt: accessExpiresAt, RefreshToken: plainRefresh,
		RefreshTokenExpiresAt: refreshExpiresAt, SessionID: session.ID, AppID: &appID}, nil
}

// PrepareAtomicMobileSession performs every non-database step required for a
// mobile OAuth or OTP session. The login-provider repository persists the returned
// hash and session metadata inside its identity transaction; any preparation
// error therefore occurs before that transaction can commit.
func (service *AuthService) PrepareAtomicMobileSession(userID, tenantID, appID uuid.UUID, authMethod string, client ClientMetadata) (PreparedMobileSession, error) {
	if userID == uuid.Nil || tenantID == uuid.Nil || appID == uuid.Nil || (authMethod != "oauth" && authMethod != "email_otp" && authMethod != "sms_otp") {
		return PreparedMobileSession{}, ErrInvalidCredentials
	}
	deviceKey, err := normalizeDeviceKey(client.DeviceKey)
	if err != nil {
		return PreparedMobileSession{}, err
	}
	plainRefresh, refreshHash, err := NewOpaqueToken()
	if err != nil {
		return PreparedMobileSession{}, err
	}
	sessionID, err := uuid.NewV7()
	if err != nil {
		return PreparedMobileSession{}, fmt.Errorf("generate mobile session id: %w", err)
	}
	now := service.clock().UTC()
	refreshExpiresAt := now.Add(30 * 24 * time.Hour)
	accessToken, accessExpiresAt, err := service.issuer.IssueForApp(userID, tenantID, sessionID, "ak-mobile", 1, appID)
	if err != nil {
		return PreparedMobileSession{}, err
	}
	appIDValue := appID
	return PreparedMobileSession{
		Tokens: SessionTokens{
			AccessToken: accessToken, AccessTokenExpiresAt: accessExpiresAt,
			RefreshToken: plainRefresh, RefreshTokenExpiresAt: refreshExpiresAt,
			SessionID: sessionID, AppID: &appIDValue,
		},
		RefreshTokenHash: refreshHash, AbsoluteExpiresAt: refreshExpiresAt,
		IdleExpiresAt: now.Add(7 * 24 * time.Hour), RefreshExpiresAt: refreshExpiresAt,
		IPAddress: client.IPAddress, UserAgent: client.UserAgent, DeviceKey: deviceKey,
		RequestID: client.RequestID, AccessTokenVersion: 1,
	}, nil
}

func (service *AuthService) VerifyUserPassword(ctx context.Context, userID uuid.UUID, password string) error {
	if userID == uuid.Nil || password == "" {
		return ErrCurrentPassword
	}
	state, err := service.identities.GetSelfPasswordState(ctx, userID)
	if err != nil || !VerifyPassword(state.CurrentHash, password) {
		return ErrCurrentPassword
	}
	return nil
}

func (service *AuthService) Refresh(ctx context.Context, refreshToken, audience string, client ClientMetadata) (SessionTokens, error) {
	if refreshToken == "" || !validAudience(audience) {
		return SessionTokens{}, domain.ErrRefreshInvalid
	}
	plainRefresh, newHash, err := NewOpaqueToken()
	if err != nil {
		return SessionTokens{}, err
	}
	session, err := service.sessions.RotateRefreshToken(ctx, HashOpaqueToken(refreshToken), newHash, client.IPAddress)
	if err != nil {
		return SessionTokens{}, err
	}
	if session.Audience != audience {
		_ = service.sessions.RevokeSession(ctx, session.ID, "audience_mismatch")
		return SessionTokens{}, ErrAudienceMismatch
	}
	accessToken, accessExpiresAt, err := service.issuer.IssueForApp(
		session.UserID, session.TenantID, session.ID, session.Audience, session.AccessTokenVersion, appIDValue(session.AppID),
	)
	if err != nil {
		return SessionTokens{}, err
	}
	return SessionTokens{
		AccessToken: accessToken, AccessTokenExpiresAt: accessExpiresAt,
		RefreshToken: plainRefresh, RefreshTokenExpiresAt: session.AbsoluteExpiresAt, SessionID: session.ID, AppID: session.AppID,
	}, nil
}

func (service *AuthService) Context(ctx context.Context, rawAccessToken, audience string) (domain.AuthContext, error) {
	authenticated, err := service.Authenticate(ctx, rawAccessToken, audience)
	if err != nil {
		return domain.AuthContext{}, err
	}
	return authenticated.AuthContext, nil
}

func (service *AuthService) SwitchTenant(ctx context.Context, rawAccessToken string, input SwitchTenantInput) (SessionTokens, error) {
	authenticated, err := service.Authenticate(ctx, rawAccessToken, input.Audience)
	if err != nil {
		return SessionTokens{}, err
	}
	if input.TenantID == uuid.Nil || input.TenantID == authenticated.Tenant.ID {
		return SessionTokens{}, ErrTenantUnavailable
	}
	allowed := false
	for _, tenant := range authenticated.AvailableTenants {
		if tenant.ID == input.TenantID && tenant.Status == "active" {
			allowed = true
			break
		}
	}
	if !allowed {
		return SessionTokens{}, ErrTenantUnavailable
	}
	deviceKey, err := normalizeDeviceKey(input.Client.DeviceKey)
	if err != nil {
		return SessionTokens{}, err
	}
	plainRefresh, refreshHash, err := NewOpaqueToken()
	if err != nil {
		return SessionTokens{}, err
	}
	now := service.clock().UTC()
	refreshExpiresAt := now.Add(30 * 24 * time.Hour)
	session, err := service.sessions.CreateSession(ctx, domain.CreateSession{
		UserID: authenticated.User.ID, TenantID: input.TenantID, Audience: input.Audience,
		AuthMethod:       "tenant_switch",
		RefreshTokenHash: refreshHash, AbsoluteExpiresAt: refreshExpiresAt,
		IdleExpiresAt: now.Add(7 * 24 * time.Hour), RefreshExpiresAt: refreshExpiresAt,
		IPAddress: input.Client.IPAddress, UserAgent: input.Client.UserAgent, DeviceKey: deviceKey,
		RequestID: input.Client.RequestID,
	})
	if err != nil {
		return SessionTokens{}, fmt.Errorf("create switched tenant session: %w", err)
	}
	accessToken, accessExpiresAt, err := service.issuer.Issue(session.UserID, session.TenantID, session.ID, session.Audience, session.AccessTokenVersion)
	if err != nil {
		_ = service.sessions.RevokeSession(ctx, session.ID, "access_token_issue_failed")
		return SessionTokens{}, err
	}
	if err = service.sessions.RevokeSession(ctx, authenticated.SessionID, "tenant_switch"); err != nil {
		_ = service.sessions.RevokeSession(ctx, session.ID, "tenant_switch_rollback")
		return SessionTokens{}, fmt.Errorf("revoke previous tenant session: %w", err)
	}
	return SessionTokens{AccessToken: accessToken, AccessTokenExpiresAt: accessExpiresAt, RefreshToken: plainRefresh, RefreshTokenExpiresAt: refreshExpiresAt, SessionID: session.ID}, nil
}

func (service *AuthService) Authenticate(ctx context.Context, rawAccessToken, audience string) (domain.AuthenticatedContext, error) {
	claims, err := service.issuer.Verify(strings.TrimSpace(rawAccessToken), audience)
	if err != nil {
		return domain.AuthenticatedContext{}, ErrInvalidAccessToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return domain.AuthenticatedContext{}, ErrInvalidAccessToken
	}
	if err = service.sessions.ValidateSession(ctx, domain.Session{
		ID: claims.SessionID, UserID: userID, TenantID: claims.TenantID,
		AppID:    appIDPointer(claims.AppID),
		Audience: audience, AccessTokenVersion: claims.TokenVersion,
	}); err != nil {
		return domain.AuthenticatedContext{}, ErrInvalidAccessToken
	}
	contextValue, err := service.sessions.GetAuthContext(ctx, userID, claims.TenantID)
	if err != nil {
		if errors.Is(err, domain.ErrIdentityNotFound) {
			return domain.AuthenticatedContext{}, ErrInvalidAccessToken
		}
		return domain.AuthenticatedContext{}, err
	}
	contextValue.AvailableTenants, err = service.identities.ListUserTenants(ctx, userID)
	if err != nil {
		return domain.AuthenticatedContext{}, err
	}
	if audience == "ak-mobile" && claims.AppID == uuid.Nil {
		return domain.AuthenticatedContext{}, ErrInvalidAccessToken
	}
	if audience == "ak-mobile" {
		if scoped, ok := service.identities.(appProfileIdentifierRepository); ok {
			contextValue.User.Email, contextValue.User.Mobile, err = scoped.AppProfileIdentifiers(ctx, claims.AppID, userID)
			if err != nil {
				return domain.AuthenticatedContext{}, err
			}
		}
	}
	return domain.AuthenticatedContext{AuthContext: contextValue, SessionID: claims.SessionID, AppID: appIDPointer(claims.AppID)}, nil
}

// ResolveDelegatedContext loads the current user, membership, role and
// permission state for an API Client delegation without creating a browser
// session or changing the token audience.
func (service *AuthService) ResolveDelegatedContext(ctx context.Context, userID, tenantID uuid.UUID) (domain.AuthenticatedContext, error) {
	contextValue, err := service.sessions.GetAuthContext(ctx, userID, tenantID)
	if errors.Is(err, domain.ErrIdentityNotFound) {
		return domain.AuthenticatedContext{}, ErrInvalidAccessToken
	}
	if err != nil {
		return domain.AuthenticatedContext{}, err
	}
	return domain.AuthenticatedContext{AuthContext: contextValue}, nil
}

func appIDPointer(value uuid.UUID) *uuid.UUID {
	if value == uuid.Nil {
		return nil
	}
	return &value
}

func (service *AuthService) UpdateSelfProfile(
	ctx context.Context,
	rawAccessToken string,
	audience string,
	input UpdateSelfProfileInput,
) (domain.User, error) {
	authenticated, err := service.Authenticate(ctx, rawAccessToken, audience)
	if err != nil {
		return domain.User{}, err
	}
	normalized, err := normalizeSelfProfile(input)
	if err != nil {
		return domain.User{}, err
	}
	user, err := service.identities.UpdateSelfProfile(ctx, domain.UpdateSelfProfile{
		UserID: authenticated.User.ID, TenantID: authenticated.Tenant.ID, SessionID: authenticated.SessionID,
		DisplayName: normalized.DisplayName, Locale: normalized.Locale, TimeZone: normalized.TimeZone,
		RequestID: normalized.RequestID, IPAddress: normalized.Client.IPAddress, UserAgent: normalized.Client.UserAgent,
	})
	if err != nil {
		return domain.User{}, err
	}
	if audience == "ak-mobile" && authenticated.AppID != nil {
		if scoped, ok := service.identities.(appProfileIdentifierRepository); ok {
			user.Email, user.Mobile, err = scoped.AppProfileIdentifiers(ctx, *authenticated.AppID, user.ID)
		}
	}
	return user, err
}

func (service *AuthService) ListSelfSessions(
	ctx context.Context,
	rawAccessToken string,
	audience string,
) ([]SelfSession, error) {
	authenticated, err := service.Authenticate(ctx, rawAccessToken, audience)
	if err != nil {
		return nil, err
	}
	rows, err := service.sessions.ListSelfSessions(ctx, domain.SelfSessionScope{
		UserID: authenticated.User.ID, TenantID: authenticated.Tenant.ID, Audience: audience,
	})
	if err != nil {
		return nil, err
	}
	result := make([]SelfSession, 0, len(rows))
	for _, row := range rows {
		result = append(result, SelfSession{SelfSession: row, Current: row.ID == authenticated.SessionID})
	}
	return result, nil
}

func (service *AuthService) RevokeSelfSession(
	ctx context.Context,
	rawAccessToken string,
	audience string,
	input RevokeSelfSessionInput,
) (bool, error) {
	if input.SessionID == uuid.Nil || strings.TrimSpace(input.RequestID) == "" {
		return false, ErrSessionValidation
	}
	authenticated, err := service.Authenticate(ctx, rawAccessToken, audience)
	if err != nil {
		return false, err
	}
	current := input.SessionID == authenticated.SessionID
	err = service.sessions.RevokeSelfSession(ctx, domain.RevokeSelfSession{
		SelfSessionScope: domain.SelfSessionScope{
			UserID: authenticated.User.ID, TenantID: authenticated.Tenant.ID, Audience: audience,
		},
		SessionID: input.SessionID, ActorSessionID: authenticated.SessionID,
		RequestID: input.RequestID, IPAddress: input.Client.IPAddress, UserAgent: input.Client.UserAgent,
		RevocationReason: "user_self_revoke",
	})
	return current, err
}

func (service *AuthService) ChangeSelfPassword(
	ctx context.Context,
	rawAccessToken string,
	audience string,
	input ChangeSelfPasswordInput,
) error {
	if strings.TrimSpace(input.RequestID) == "" || len(input.CurrentPassword) < 12 || len(input.CurrentPassword) > 256 ||
		len(input.NewPassword) < 12 || len(input.NewPassword) > 256 {
		return ErrPasswordValidation
	}
	authenticated, err := service.Authenticate(ctx, rawAccessToken, audience)
	if err != nil {
		return err
	}
	state, err := service.identities.GetSelfPasswordState(ctx, authenticated.User.ID)
	if err != nil {
		return err
	}
	if !VerifyPassword(state.CurrentHash, input.CurrentPassword) {
		return ErrCurrentPassword
	}
	for _, passwordHash := range append([]string{state.CurrentHash}, state.HistoryHashes...) {
		if VerifyPassword(passwordHash, input.NewPassword) {
			return ErrPasswordReused
		}
	}
	newHash, err := HashPassword(input.NewPassword)
	if err != nil {
		return ErrPasswordValidation
	}
	return service.identities.ChangeSelfPassword(ctx, domain.ChangeSelfPassword{
		UserID: authenticated.User.ID, TenantID: authenticated.Tenant.ID, SessionID: authenticated.SessionID,
		ExpectedHash: state.CurrentHash, NewHash: newHash, ExpectedVersion: state.CurrentVersion,
		RequestID: input.RequestID, IPAddress: input.Client.IPAddress, UserAgent: input.Client.UserAgent,
	})
}

func (service *AuthService) ListSelfDevices(
	ctx context.Context,
	rawAccessToken string,
	audience string,
) ([]SelfDevice, error) {
	authenticated, err := service.Authenticate(ctx, rawAccessToken, audience)
	if err != nil {
		return nil, err
	}
	rows, err := service.sessions.ListSelfDevices(ctx, domain.SelfDeviceScope{
		UserID: authenticated.User.ID, TenantID: authenticated.Tenant.ID,
		SessionID: authenticated.SessionID, Audience: audience,
	})
	if err != nil {
		return nil, err
	}
	result := make([]SelfDevice, 0, len(rows))
	for _, row := range rows {
		result = append(result, SelfDevice{SelfDevice: row, Current: row.Current})
	}
	return result, nil
}

func (service *AuthService) RemoveSelfDevice(
	ctx context.Context,
	rawAccessToken string,
	audience string,
	input RemoveSelfDeviceInput,
) (bool, error) {
	if input.DeviceID == uuid.Nil || strings.TrimSpace(input.RequestID) == "" {
		return false, ErrDeviceValidation
	}
	authenticated, err := service.Authenticate(ctx, rawAccessToken, audience)
	if err != nil {
		return false, err
	}
	return service.sessions.RemoveSelfDevice(ctx, domain.RemoveSelfDevice{
		SelfDeviceScope: domain.SelfDeviceScope{
			UserID: authenticated.User.ID, TenantID: authenticated.Tenant.ID,
			SessionID: authenticated.SessionID, Audience: audience,
		},
		DeviceID: input.DeviceID, RequestID: input.RequestID,
		IPAddress: input.Client.IPAddress, UserAgent: input.Client.UserAgent,
	})
}

func normalizeDeviceKey(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil {
		return "", ErrDeviceValidation
	}
	return parsed.String(), nil
}

func (service *AuthService) Logout(ctx context.Context, rawAccessToken, audience string) error {
	claims, err := service.issuer.Verify(strings.TrimSpace(rawAccessToken), audience)
	if err != nil {
		return err
	}
	return service.sessions.RevokeSession(ctx, claims.SessionID, "user_logout")
}

func validAudience(audience string) bool {
	return audience == "ak-admin" || audience == "ak-mobile" || audience == "ak-api"
}

func normalizeSelfProfile(input UpdateSelfProfileInput) (UpdateSelfProfileInput, error) {
	if input.DisplayName == nil && input.Locale == nil && input.TimeZone == nil {
		return UpdateSelfProfileInput{}, ErrProfileValidation
	}
	if input.DisplayName != nil {
		value := strings.TrimSpace(*input.DisplayName)
		if value == "" || len([]rune(value)) > 160 {
			return UpdateSelfProfileInput{}, ErrProfileValidation
		}
		input.DisplayName = &value
	}
	if input.Locale != nil {
		value := strings.TrimSpace(*input.Locale)
		if value != "zh-CN" && value != "en-US" {
			return UpdateSelfProfileInput{}, ErrProfileValidation
		}
		input.Locale = &value
	}
	if input.TimeZone != nil {
		value := strings.TrimSpace(*input.TimeZone)
		if value == "" || len(value) > 64 {
			return UpdateSelfProfileInput{}, ErrProfileValidation
		}
		if _, err := time.LoadLocation(value); err != nil {
			return UpdateSelfProfileInput{}, ErrProfileValidation
		}
		input.TimeZone = &value
	}
	if strings.TrimSpace(input.RequestID) == "" {
		return UpdateSelfProfileInput{}, ErrProfileValidation
	}
	return input, nil
}
