package domain

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmailAlreadyExists      = errors.New("email already exists")
	ErrIdentityNotFound        = errors.New("identity not found")
	ErrRefreshInvalid          = errors.New("refresh token is invalid")
	ErrRefreshReused           = errors.New("refresh token reuse detected")
	ErrSessionNotFound         = errors.New("session not found")
	ErrDeviceNotFound          = errors.New("device not found")
	ErrPasswordChanged         = errors.New("password changed concurrently")
	ErrResetTokenInvalid       = errors.New("password reset token is invalid")
	ErrRegistrationTenant      = errors.New("registration tenant is unavailable")
	ErrLoginCaptchaInvalid     = errors.New("login captcha is invalid")
	ErrLoginCaptchaNotRequired = errors.New("login captcha is not required")
	ErrLoginCaptchaCooldown    = errors.New("login captcha generation is cooling down")
)

type User struct {
	ID           uuid.UUID
	Email        string
	Mobile       string
	DisplayName  string
	Locale       string
	TimeZone     string
	Status       string
	AvatarFileID *uuid.UUID
}

type Tenant struct {
	ID     uuid.UUID
	Code   string
	Name   string
	Status string
}

type Credential struct {
	User         User
	PasswordHash string
}

type CreateIdentity struct {
	TenantCode   string
	TenantName   string
	Email        string
	DisplayName  string
	Locale       string
	PasswordHash string
}

func (input CreateIdentity) Normalize() CreateIdentity {
	input.TenantCode = strings.ToLower(strings.TrimSpace(input.TenantCode))
	input.TenantName = strings.TrimSpace(input.TenantName)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Locale = strings.TrimSpace(input.Locale)
	return input
}

type Repository interface {
	CreateIdentity(context.Context, CreateIdentity) (User, Tenant, error)
	RegisterAdmin(context.Context, RegisterAdmin) error
	FindCredentialByEmail(context.Context, string) (Credential, error)
	ResolveActiveMobileAppMembership(context.Context, uuid.UUID, uuid.UUID) (Tenant, error)
	ListUserTenants(context.Context, uuid.UUID) ([]Tenant, error)
	UpdateSelfProfile(context.Context, UpdateSelfProfile) (User, error)
	GetSelfPasswordState(context.Context, uuid.UUID) (SelfPasswordState, error)
	ChangeSelfPassword(context.Context, ChangeSelfPassword) error
	LoginCaptchaRequired(context.Context, []byte, time.Time) (bool, error)
	CheckLoginCaptchaGeneration(context.Context, []byte, time.Time) error
	RecordLoginFailure(context.Context, LoginFailure) (int32, error)
	ResetLoginFailures(context.Context, []byte) error
	CreateLoginCaptcha(context.Context, LoginCaptchaChallenge) (uuid.UUID, error)
	VerifyLoginCaptcha(context.Context, LoginCaptchaAttempt) error
	CreateInteractiveCaptcha(context.Context, LoginCaptchaChallenge) (uuid.UUID, error)
	VerifyInteractiveCaptcha(context.Context, LoginCaptchaAttempt) error
	PreparePasswordReset(context.Context, PreparePasswordReset) (*PasswordResetRecipient, error)
	GetPasswordResetState(context.Context, []byte) (PasswordResetState, error)
	ResetPassword(context.Context, ResetPassword) error
}

type RegisterAdmin struct {
	TenantCode   string
	Email        string
	DisplayName  string
	Locale       string
	PasswordHash string
	RequestID    string
	IPAddress    *netip.Addr
	UserAgent    string
}

type PreparePasswordReset struct {
	Email      string
	TargetHash []byte
	SecretHash []byte
	ExpiresAt  time.Time
	RequestID  string
	IPAddress  *netip.Addr
}

type PasswordResetRecipient struct {
	TenantID uuid.UUID
	Email    string
	Locale   string
}

type PasswordResetState struct {
	UserID         uuid.UUID
	CurrentHash    string
	CurrentVersion int32
	HistoryHashes  []string
}

type ResetPassword struct {
	TokenHash       []byte
	UserID          uuid.UUID
	ExpectedHash    string
	ExpectedVersion int32
	NewHash         string
	RequestID       string
	IPAddress       *netip.Addr
	UserAgent       string
}

type LoginFailure struct {
	UserID    *uuid.UUID
	AppID     *uuid.UUID
	Audience  string
	RequestID string
	IPAddress *netip.Addr
	UserAgent string
	ScopeHash []byte
	FailedAt  time.Time
	ExpiresAt time.Time
}

type LoginCaptchaChallenge struct {
	ID          uuid.UUID
	ScopeHash   []byte
	CaptchaType string
	ProofHash   []byte
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type LoginCaptchaAttempt struct {
	ID          uuid.UUID
	ScopeHash   []byte
	CaptchaType string
	ProofHash   []byte
	Valid       bool
	Now         time.Time
}

type SelfPasswordState struct {
	CurrentHash    string
	CurrentVersion int32
	HistoryHashes  []string
}

type ChangeSelfPassword struct {
	UserID          uuid.UUID
	TenantID        uuid.UUID
	SessionID       uuid.UUID
	ExpectedHash    string
	NewHash         string
	ExpectedVersion int32
	RequestID       string
	IPAddress       *netip.Addr
	UserAgent       string
}

type UpdateSelfProfile struct {
	UserID      uuid.UUID
	TenantID    uuid.UUID
	SessionID   uuid.UUID
	DisplayName *string
	Locale      *string
	TimeZone    *string
	RequestID   string
	IPAddress   *netip.Addr
	UserAgent   string
}

type Session struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	TenantID           uuid.UUID
	AppID              *uuid.UUID
	Audience           string
	AccessTokenVersion int32
	AbsoluteExpiresAt  time.Time
}

type SelfSession struct {
	ID                uuid.UUID
	Audience          string
	Status            string
	IPAddress         *netip.Addr
	UserAgent         string
	LastSeenAt        time.Time
	AbsoluteExpiresAt time.Time
	CreatedAt         time.Time
}

type SelfSessionScope struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
	Audience string
}

type RevokeSelfSession struct {
	SelfSessionScope
	SessionID        uuid.UUID
	ActorSessionID   uuid.UUID
	RequestID        string
	IPAddress        *netip.Addr
	UserAgent        string
	RevocationReason string
}

type CreateSession struct {
	UserID            uuid.UUID
	TenantID          uuid.UUID
	AppID             *uuid.UUID
	Audience          string
	AuthMethod        string
	RefreshTokenHash  []byte
	AbsoluteExpiresAt time.Time
	IdleExpiresAt     time.Time
	RefreshExpiresAt  time.Time
	IPAddress         *netip.Addr
	UserAgent         string
	DeviceKey         string
	RequestID         string
}

type SelfDevice struct {
	ID                 uuid.UUID
	Platform           string
	DeviceName         string
	Model              string
	OSVersion          string
	AppVersion         string
	LastIP             *netip.Addr
	LastSeenAt         *time.Time
	CreatedAt          time.Time
	LatestUserAgent    string
	ActiveSessionCount int64
	Current            bool
}

type SelfDeviceScope struct {
	UserID    uuid.UUID
	TenantID  uuid.UUID
	SessionID uuid.UUID
	Audience  string
}

type RemoveSelfDevice struct {
	SelfDeviceScope
	DeviceID  uuid.UUID
	RequestID string
	IPAddress *netip.Addr
	UserAgent string
}

type AuthContext struct {
	User             User
	Tenant           Tenant
	TimeZone         string
	Roles            []string
	Permissions      []string
	Menus            []Menu
	AvailableTenants []Tenant
}

type AuthenticatedContext struct {
	AuthContext
	SessionID   uuid.UUID
	AppID       *uuid.UUID
	APIClientID *uuid.UUID
}

type Menu struct {
	ID           uuid.UUID
	ParentID     *uuid.UUID
	Code         string
	I18nKey      string
	Title        string
	Type         string
	RoutePath    *string
	ComponentKey *string
	Icon         *string
	Affix        bool
	SortOrder    int32
	FeatureFlag  string
}

type SessionRepository interface {
	CreateSession(context.Context, CreateSession) (Session, error)
	RotateRefreshToken(context.Context, []byte, []byte, *netip.Addr) (Session, error)
	RevokeSession(context.Context, uuid.UUID, string) error
	ValidateSession(context.Context, Session) error
	GetAuthContext(context.Context, uuid.UUID, uuid.UUID) (AuthContext, error)
	ListSelfSessions(context.Context, SelfSessionScope) ([]SelfSession, error)
	RevokeSelfSession(context.Context, RevokeSelfSession) error
	ListSelfDevices(context.Context, SelfDeviceScope) ([]SelfDevice, error)
	RemoveSelfDevice(context.Context, RemoveSelfDevice) (bool, error)
}
