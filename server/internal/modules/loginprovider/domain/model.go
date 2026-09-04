// Package domain defines the stable, code-registered login-provider contract.
// Provider endpoints and executable adapters are never loaded from database
// values.
package domain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden           = errors.New("login provider operation forbidden")
	ErrInvalid             = errors.New("login provider input invalid")
	ErrNotFound            = errors.New("login provider resource not found")
	ErrConflict            = errors.New("login provider resource conflict")
	ErrInUse               = errors.New("login provider configuration is in use")
	ErrProviderUnavailable = errors.New("login provider is unavailable")
	ErrConfigStale         = errors.New("login provider build configuration is stale")
	ErrFlowInvalid         = errors.New("oauth authorization flow is invalid")
	ErrFlowExpired         = errors.New("oauth authorization flow expired")
	ErrFlowConsumed        = errors.New("oauth authorization flow already consumed")
	ErrAuthorizationDenied = errors.New("provider authorization denied")
	ErrIdentityConflict    = errors.New("external identity is already bound")
	ErrLinkRequired        = errors.New("existing account login is required before binding")
	ErrCallbackInvalid     = errors.New("provider callback is invalid")
	ErrLastLoginMethod     = errors.New("the last usable login method cannot be removed")
	ErrIdentifierConflict  = errors.New("identifier is already bound")
	ErrAccountExists       = errors.New("account already exists")
	ErrOTPInvalid          = errors.New("verification code is invalid")
	ErrDeliveryUnavailable = errors.New("identifier delivery channel is unavailable")
	ErrStepUpRequired      = errors.New("step-up authentication is required")
	ErrStepUpInvalid       = errors.New("step-up authentication is invalid")
)

const (
	ProviderWechat = "wechat"
	ProviderGitHub = "github"
	ProviderApple  = "apple"
	ProviderGoogle = "google"
)

var ProviderCodes = []string{ProviderWechat, ProviderGitHub, ProviderApple, ProviderGoogle}

type PlatformConfig struct {
	Enabled       bool   `json:"enabled"`
	PackageName   string `json:"package_name,omitempty"`
	AppSignature  string `json:"app_signature,omitempty"`
	BundleID      string `json:"bundle_id,omitempty"`
	UniversalLink string `json:"universal_link,omitempty"`
	BundleName    string `json:"bundle_name,omitempty"`
}

type WechatPublicConfig struct {
	Android PlatformConfig `json:"android"`
	IOS     PlatformConfig `json:"ios"`
	Harmony PlatformConfig `json:"harmony"`
}

type GitHubPublicConfig struct {
	AppReturnURI string `json:"app_return_uri"`
}

type ApplePublicConfig struct {
	TeamID string `json:"team_id"`
	KeyID  string `json:"key_id"`
}

type GooglePublicConfig struct {
	AndroidPackageName       string   `json:"android_package_name"`
	AndroidCertificateSHA256 []string `json:"android_certificate_sha256"`
}

func decodeStrict[T any](raw json.RawMessage) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if len(raw) == 0 {
		return value, ErrInvalid
	}
	if err := decoder.Decode(&value); err != nil {
		return value, ErrInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return value, ErrInvalid
	}
	return value, nil
}

// CanonicalPublicConfig decodes a provider v1 shape with unknown fields
// rejected, then emits the deterministic encoding/json representation shared
// by the API runtime and ak-cli build exporter.
func CanonicalPublicConfig(providerCode string, raw json.RawMessage) (json.RawMessage, error) {
	var value any
	switch providerCode {
	case ProviderWechat:
		decoded, err := decodeStrict[WechatPublicConfig](raw)
		if err != nil {
			return nil, err
		}
		value = decoded
	case ProviderGitHub:
		decoded, err := decodeStrict[GitHubPublicConfig](raw)
		if err != nil {
			return nil, err
		}
		value = decoded
	case ProviderApple:
		decoded, err := decodeStrict[ApplePublicConfig](raw)
		if err != nil {
			return nil, err
		}
		value = decoded
	case ProviderGoogle:
		decoded, err := decodeStrict[GooglePublicConfig](raw)
		if err != nil {
			return nil, err
		}
		value = decoded
	default:
		return nil, ErrInvalid
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical login provider config: %w", err)
	}
	return canonical, nil
}

// BuildConfigHash is the only supported build/runtime drift algorithm. It
// projects only values embedded in a native build; operational Apple signing
// metadata, secrets and mutable lifecycle fields are deliberately excluded.
func BuildConfigHash(providerCode, externalClientID string, rawPublicConfig json.RawMessage) (string, error) {
	var projection any
	switch providerCode {
	case ProviderWechat:
		value, err := decodeStrict[WechatPublicConfig](rawPublicConfig)
		if err != nil {
			return "", err
		}
		projection = struct {
			ExternalClientID string         `json:"external_client_id"`
			Android          PlatformConfig `json:"android"`
			IOS              PlatformConfig `json:"ios"`
			Harmony          PlatformConfig `json:"harmony"`
		}{externalClientID, value.Android, value.IOS, value.Harmony}
	case ProviderGitHub:
		value, err := decodeStrict[GitHubPublicConfig](rawPublicConfig)
		if err != nil {
			return "", err
		}
		projection = struct {
			ExternalClientID string `json:"external_client_id"`
			AppReturnURI     string `json:"app_return_uri"`
		}{externalClientID, value.AppReturnURI}
	case ProviderApple:
		if _, err := decodeStrict[ApplePublicConfig](rawPublicConfig); err != nil {
			return "", err
		}
		projection = struct {
			ExternalClientID string `json:"external_client_id"`
		}{externalClientID}
	case ProviderGoogle:
		value, err := decodeStrict[GooglePublicConfig](rawPublicConfig)
		if err != nil {
			return "", err
		}
		projection = struct {
			ExternalClientID         string   `json:"external_client_id"`
			AndroidPackageName       string   `json:"android_package_name"`
			AndroidCertificateSHA256 []string `json:"android_certificate_sha256"`
		}{externalClientID, value.AndroidPackageName, value.AndroidCertificateSHA256}
	default:
		return "", ErrInvalid
	}
	canonical, err := json.Marshal(projection)
	if err != nil {
		return "", ErrInvalid
	}
	digest := sha256.Sum256([]byte(providerCode + "\n" + string(canonical)))
	return hex.EncodeToString(digest[:]), nil
}

type FieldDescriptor struct {
	Name      string `json:"name"`
	Location  string `json:"location"`
	ValueType string `json:"value_type"`
	Required  bool   `json:"required"`
	Secret    bool   `json:"secret"`
	MaxLength int32  `json:"max_length"`
	HelpKey   string `json:"help_key"`
}

type ProviderDescriptor struct {
	ProviderCode        string            `json:"provider_code"`
	DisplayNameKey      string            `json:"display_name_key"`
	IconKey             string            `json:"icon_key"`
	AuthorizationKind   string            `json:"authorization_kind"`
	SupportedPlatforms  []string          `json:"supported_platforms"`
	BuildVariants       []string          `json:"build_variants"`
	ConfigSchemaVersion int32             `json:"config_schema_version"`
	RequiresSecret      bool              `json:"requires_secret"`
	Fields              []FieldDescriptor `json:"fields"`
	HelpURL             string            `json:"help_url"`
}

type Catalog struct {
	Items []ProviderDescriptor `json:"items"`
}

type Principal struct {
	TenantID, UserID, SessionID     uuid.UUID
	RequestID, IPAddress, UserAgent string
}

type Config struct {
	ID                    uuid.UUID       `json:"id"`
	TenantID              uuid.UUID       `json:"tenant_id,omitempty"`
	Name                  string          `json:"name"`
	Description           string          `json:"description"`
	ProviderCode          string          `json:"provider_code"`
	ExternalClientID      string          `json:"external_client_id"`
	ConfigSchemaVersion   int32           `json:"config_schema_version"`
	PublicConfig          json.RawMessage `json:"public_config"`
	SecretFieldNames      []string        `json:"secret_field_names"`
	HasSecret             bool            `json:"has_secret"`
	CredentialFingerprint string          `json:"credential_fingerprint,omitempty"`
	Status                string          `json:"status"`
	LastPreflightAt       *time.Time      `json:"last_preflight_at"`
	LastPreflightStatus   *string         `json:"last_preflight_status"`
	LastPreflightIssues   []string        `json:"last_preflight_issues"`
	BindingCount          int64           `json:"binding_count"`
	LockVersion           int32           `json:"lock_version"`
	CallbackURI           string          `json:"callback_uri,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	SecretCiphertext      []byte          `json:"-"`
	SecretKeyVersion      *int32          `json:"-"`
}

type ConfigInput struct {
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	ProviderCode        string          `json:"provider_code"`
	ExternalClientID    string          `json:"external_client_id"`
	ConfigSchemaVersion int32           `json:"config_schema_version"`
	PublicConfig        json.RawMessage `json:"public_config"`
	LockVersion         int32           `json:"lock_version,omitempty"`
}

type SecretInput struct {
	Values      map[string]string `json:"values"`
	LockVersion int32             `json:"lock_version"`
}

type ListFilter struct {
	Query, ProviderCode, Status string
	Page, PageSize              int32
}

type ConfigPage struct {
	Items    []Config `json:"items"`
	Page     int32    `json:"page"`
	PageSize int32    `json:"page_size"`
	Total    int64    `json:"total"`
}

type Binding struct {
	ID                    *uuid.UUID `json:"id"`
	AppID                 uuid.UUID  `json:"app_id"`
	ProviderCode          string     `json:"provider_code"`
	LoginProviderConfigID *uuid.UUID `json:"login_provider_config_id"`
	ConfigName            *string    `json:"config_name"`
	ConfigStatus          *string    `json:"config_status"`
	PreflightStatus       *string    `json:"preflight_status"`
	Enabled               bool       `json:"enabled"`
	SortOrder             int32      `json:"sort_order"`
	LockVersion           int32      `json:"lock_version"`
	UpdatedAt             *time.Time `json:"updated_at"`
}

type BindingInput struct {
	ProviderCode          string     `json:"provider_code"`
	LoginProviderConfigID *uuid.UUID `json:"login_provider_config_id"`
	Enabled               bool       `json:"enabled"`
	SortOrder             int32      `json:"sort_order"`
	LockVersion           int32      `json:"lock_version"`
}

type BulkBindingInput struct {
	Bindings []BindingInput `json:"bindings"`
}

type BindingList struct {
	Items []Binding `json:"items"`
}

type RuntimeProvider struct {
	TenantID            uuid.UUID
	AppID               uuid.UUID
	ConfigID            uuid.UUID
	DefaultLocale       string
	ProviderCode        string
	ExternalClientID    string
	PublicConfig        json.RawMessage
	SecretCiphertext    []byte
	SecretKeyVersion    *int32
	SecretFieldNames    []string
	RegistrationEnabled bool
	Enabled             bool
	SortOrder           int32
}

type MobileProvider struct {
	ProviderCode        string         `json:"provider_code"`
	DisplayNameKey      string         `json:"display_name_key"`
	IconKey             string         `json:"icon_key"`
	AuthorizationKind   string         `json:"authorization_kind"`
	SupportedPlatforms  []string       `json:"supported_platforms"`
	BuildVariants       []string       `json:"build_variants"`
	LoginEnabled        bool           `json:"login_enabled"`
	BindingEnabled      bool           `json:"binding_enabled"`
	SortOrder           int32          `json:"sort_order"`
	ConfigSchemaVersion int32          `json:"config_schema_version"`
	BuildConfigHash     string         `json:"build_config_hash"`
	BuildConfig         map[string]any `json:"build_config"`
}

type MobileProviderList struct {
	Items []MobileProvider `json:"items"`
}

type AuthorizeInput struct {
	Mode            string     `json:"mode"`
	Platform        string     `json:"platform"`
	BuildVariant    string     `json:"build_variant"`
	BuildConfigHash string     `json:"build_config_hash"`
	StepUpToken     string     `json:"step_up_token,omitempty"`
	ReauthPurpose   string     `json:"reauth_purpose,omitempty"`
	AccountID       *uuid.UUID `json:"account_id,omitempty"`
}

type AuthorizeResult struct {
	FlowID            uuid.UUID `json:"flow_id"`
	ProviderCode      string    `json:"provider_code"`
	Mode              string    `json:"mode"`
	AuthorizationKind string    `json:"authorization_kind"`
	AuthorizationURL  string    `json:"authorization_url,omitempty"`
	State             string    `json:"state"`
	Nonce             string    `json:"nonce,omitempty"`
	ExpiresAt         time.Time `json:"expires_at"`
}

type CallbackInput struct {
	FlowID            uuid.UUID `json:"flow_id"`
	State             string    `json:"state,omitempty"`
	AuthorizationCode string    `json:"authorization_code,omitempty"`
	IDToken           string    `json:"id_token,omitempty"`
	OneTimeTicket     string    `json:"one_time_ticket,omitempty"`
	DisplayName       string    `json:"display_name,omitempty"`
}

type Flow struct {
	ID, TenantID, AppID, ConfigID  uuid.UUID
	ProviderCode, Mode             string
	Platform, BuildVariant         string
	UserID, SessionID              *uuid.UUID
	ReauthPurpose                  string
	TargetOAuthAccountID           *uuid.UUID
	StateHash, NonceHash           []byte
	PKCECiphertext                 []byte
	PKCEKeyVersion                 *int32
	DeviceKeyHash                  []byte
	VerifiedIdentityCiphertext     []byte
	VerifiedIdentityKeyVersion     *int32
	CompletionTicketHash           []byte
	CompletionTicketExpiresAt      *time.Time
	ExpiresAt                      time.Time
	ProviderVerifiedAt, ConsumedAt *time.Time
	FailureCount                   int32
}

type FlowCreate struct {
	TenantID, AppID, ConfigID uuid.UUID
	ProviderCode, Mode        string
	Platform, BuildVariant    string
	UserID, SessionID         *uuid.UUID
	ReauthPurpose             string
	TargetOAuthAccountID      *uuid.UUID
	StateHash, NonceHash      []byte
	PKCECiphertext            []byte
	PKCEKeyVersion            *int32
	DeviceKeyHash             []byte
	ExpiresAt                 time.Time
}

type VerifiedIdentity struct {
	Issuer           string          `json:"issuer"`
	ExternalClientID string          `json:"external_client_id"`
	Subject          string          `json:"subject"`
	UnionSubject     string          `json:"union_subject,omitempty"`
	ProviderUsername string          `json:"provider_username,omitempty"`
	VerifiedEmail    string          `json:"verified_email,omitempty"`
	DisplayName      string          `json:"display_name,omitempty"`
	Profile          json.RawMessage `json:"profile,omitempty"`
	Nonce            string          `json:"-"`
}

type OAuthAccount struct {
	ID                   uuid.UUID  `json:"id"`
	ProviderCode         string     `json:"provider_code"`
	DisplayNameKey       string     `json:"display_name_key"`
	ProviderUsernameHint string     `json:"provider_username_hint"`
	Status               string     `json:"status"`
	LoginEnabled         bool       `json:"login_enabled"`
	LoginCapable         bool       `json:"login_capable"`
	CanBind              bool       `json:"can_bind"`
	CanChange            bool       `json:"can_change"`
	BoundAt              time.Time  `json:"bound_at"`
	LastAuthenticatedAt  *time.Time `json:"last_authenticated_at"`
	CanUnbind            bool       `json:"can_unbind"`
	BlockReason          string     `json:"block_reason,omitempty"`
}

type Identifier struct {
	ID             uuid.UUID  `json:"id"`
	IdentifierType string     `json:"identifier_type"`
	DisplayHint    string     `json:"display_hint"`
	Verified       bool       `json:"verified"`
	Status         string     `json:"status"`
	LoginCapable   bool       `json:"login_capable"`
	CanBind        bool       `json:"can_bind"`
	CanChange      bool       `json:"can_change"`
	CanUnbind      bool       `json:"can_unbind"`
	BlockReason    string     `json:"block_reason,omitempty"`
	VerifiedAt     *time.Time `json:"-"`
}

// IdentifierTarget is an internal-only projection used for step-up delivery.
// NormalizedValue must never be returned by a transport DTO or written to a
// generic log entry.
type IdentifierTarget struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	AppID           uuid.UUID
	UserID          uuid.UUID
	IdentifierType  string
	NormalizedValue string
	DisplayHint     string
	Locale          string
}

type PasswordMethod struct {
	Present      bool   `json:"present"`
	LoginCapable bool   `json:"login_capable"`
	CanBind      bool   `json:"can_bind"`
	CanChange    bool   `json:"can_change"`
	CanUnbind    bool   `json:"can_unbind"`
	BlockReason  string `json:"block_reason,omitempty"`
}

type LoginMethods struct {
	Password              PasswordMethod `json:"password"`
	Identifiers           []Identifier   `json:"identifiers"`
	OAuthAccounts         []OAuthAccount `json:"oauth_accounts"`
	RemainingLoginMethods int            `json:"remaining_login_methods"`
}

type SecretSealer interface {
	Seal([]byte, string) ([]byte, int32, error)
	Open([]byte, string) ([]byte, error)
}

type VerificationRequest struct {
	ProviderCode, ExternalClientID string
	Mode                           string
	PublicConfig                   json.RawMessage
	Secrets                        map[string]string
	AuthorizationCode, IDToken     string
	ExpectedSubject                string
	Nonce, RedirectURI             string
	PKCEVerifier                   string
}

type ProviderAdapter interface {
	AuthorizationURL(context.Context, ProviderDescriptor, RuntimeProvider, string, string, string) (string, error)
	Verify(context.Context, VerificationRequest) (VerifiedIdentity, error)
}

// AppleRevoker is deliberately separated from ProviderAdapter so tests and
// non-Apple adapters cannot accidentally claim revocation support. The
// authorization code and provider token remain ephemeral and are never
// persisted by the application service.
type AppleRevoker interface {
	RevokeApple(context.Context, VerificationRequest) error
}

type ResolvedIdentity struct {
	TenantID uuid.UUID
	AppID    uuid.UUID
	UserID   uuid.UUID
	Account  OAuthAccount
	Created  bool
}

// AtomicLoginSession contains only the persistence material needed to create
// an OAuth login session in the same serializable transaction as a newly
// registered external identity. Plain refresh/access tokens stay in the IAM
// application service and never cross the repository boundary.
type AtomicLoginSession struct {
	ID                 uuid.UUID
	RefreshTokenHash   []byte
	AbsoluteExpiresAt  time.Time
	IdleExpiresAt      time.Time
	RefreshExpiresAt   time.Time
	IPAddress          string
	UserAgent          string
	DeviceKey          string
	RequestID          string
	AccessTokenVersion int32
	AuthMethod         string
}

type LoginSessionFactory func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (AtomicLoginSession, error)

type IdentityResolution struct {
	Runtime              RuntimeProvider
	Identity             VerifiedIdentity
	Mode                 string
	UserID               *uuid.UUID
	SessionID            *uuid.UUID
	TargetOAuthAccountID *uuid.UUID
}

type OTPChallenge struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	AppID           uuid.UUID
	UserID          *uuid.UUID
	IdentifierType  string
	NormalizedValue string
	DisplayHint     string
	SecretHash      []byte
	Code            string
	Purpose         string
	Locale          string
	ExpiresAt       time.Time
	CreatedIP       string
	DeviceKeyHash   []byte
}

type OTPConsume struct {
	ID              uuid.UUID
	AppID           uuid.UUID
	IdentifierType  string
	NormalizedValue string
	TargetHash      []byte
	SecretHash      []byte
	Purpose         string
}

type AppLoginSettings struct {
	TenantID         uuid.UUID
	Registration     bool
	OTPEnabled       bool
	EmailOTPEnabled  bool
	MobileOTPEnabled bool
}

type OTPRegistration struct {
	AppID           uuid.UUID
	IdentifierType  string
	NormalizedValue string
	DisplayHint     string
	DisplayName     string
	Locale          string
	ChallengeID     uuid.UUID
	TargetHash      []byte
	SecretHash      []byte
}

type IdentifierMutation struct {
	Principal
	AppID           uuid.UUID
	IdentifierType  string
	NormalizedValue string
	DisplayHint     string
	ChallengeID     uuid.UUID
	TargetHash      []byte
	SecretHash      []byte
}

type Repository interface {
	ResolveApp(context.Context, uuid.UUID) (uuid.UUID, string, error)
	AppLoginSettings(context.Context, uuid.UUID) (AppLoginSettings, error)
	ListConfigs(context.Context, uuid.UUID, ListFilter) (ConfigPage, error)
	GetConfig(context.Context, uuid.UUID, uuid.UUID) (Config, error)
	GetConfigSecret(context.Context, uuid.UUID, uuid.UUID) (Config, error)
	CreateConfig(context.Context, Principal, ConfigInput) (Config, error)
	UpdateConfig(context.Context, Principal, uuid.UUID, ConfigInput) (Config, error)
	RotateSecret(context.Context, Principal, uuid.UUID, int32, []byte, int32, []string, string) (Config, error)
	SetPreflight(context.Context, Principal, uuid.UUID, int32, bool, []string) (Config, error)
	SetStatus(context.Context, Principal, uuid.UUID, int32, string) (Config, error)
	DeleteConfig(context.Context, Principal, uuid.UUID, int32) error
	ListBindings(context.Context, uuid.UUID, uuid.UUID) ([]Binding, error)
	ReplaceBindings(context.Context, Principal, uuid.UUID, []BindingInput) ([]Binding, error)
	RuntimeProviders(context.Context, uuid.UUID) ([]RuntimeProvider, error)
	RuntimeProvider(context.Context, uuid.UUID, string) (RuntimeProvider, error)
	RuntimeProviderForReauth(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (RuntimeProvider, error)
	CreateFlow(context.Context, FlowCreate) (Flow, error)
	GetFlow(context.Context, uuid.UUID, uuid.UUID, string, []byte) (Flow, error)
	GetBrowserFlow(context.Context, string, []byte) (Flow, error)
	ClaimNativeFlow(context.Context, uuid.UUID, uuid.UUID, string, []byte, []byte) (Flow, error)
	MarkBrowserVerified(context.Context, uuid.UUID, []byte, []byte, int32) error
	ClaimTicketFlow(context.Context, uuid.UUID, uuid.UUID, string, []byte, []byte) (Flow, error)
	ResolveIdentity(context.Context, IdentityResolution, LoginSessionFactory) (ResolvedIdentity, error)
	ListOAuthAccounts(context.Context, uuid.UUID, uuid.UUID) ([]OAuthAccount, error)
	DeleteOAuthAccount(context.Context, Principal, uuid.UUID, uuid.UUID) error
	LoginMethods(context.Context, uuid.UUID, uuid.UUID) (LoginMethods, error)
	IdentifierTarget(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (IdentifierTarget, error)
	FindOTPLoginUser(context.Context, uuid.UUID, string, string) (uuid.UUID, uuid.UUID, string, error)
	CreateOTPChallenge(context.Context, OTPChallenge) (uuid.UUID, error)
	ConsumeOTPChallenge(context.Context, OTPConsume) (uuid.UUID, error)
	RegisterWithOTP(context.Context, OTPRegistration, LoginSessionFactory) (uuid.UUID, error)
	ResetPasswordWithOTP(context.Context, OTPConsume, string) error
	SetPassword(context.Context, Principal, uuid.UUID, string) error
	UpsertIdentifier(context.Context, IdentifierMutation) (Identifier, error)
	DeleteIdentifier(context.Context, Principal, uuid.UUID, string) error
	CreateStepUpTicket(context.Context, StepUpTicket) error
	ConsumeStepUpTicket(context.Context, StepUpConsume) error
}

type StepUpTicket struct {
	ID        uuid.UUID
	Principal Principal
	AppID     uuid.UUID
	Purpose   string
	Resource  string
	Method    string
	TokenHash []byte
	ExpiresAt time.Time
}

type StepUpConsume struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	SessionID      uuid.UUID
	AppID          uuid.UUID
	Purpose        string
	Resource       string
	TokenHash      []byte
	RequiredMethod string
}
