package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden        = errors.New("identity security operation forbidden")
	ErrInvalid          = errors.New("identity security input invalid")
	ErrNotFound         = errors.New("identity security resource not found")
	ErrConflict         = errors.New("identity security state conflict")
	ErrStepUpRequired   = errors.New("step-up proof is invalid")
	ErrFeatureDisabled  = errors.New("identity security feature disabled")
	ErrOAuthState       = errors.New("oauth state is invalid or expired")
	ErrProviderDisabled = errors.New("oauth provider is unavailable")
)

type Principal struct {
	TenantID, UserID, SessionID     uuid.UUID
	RequestID, IPAddress, UserAgent string
}

type MFAStatus struct {
	TOTPEnabled            bool       `json:"totp_enabled"`
	TOTPVerifiedAt         *time.Time `json:"totp_verified_at,omitempty"`
	RecoveryCodesRemaining int64      `json:"recovery_codes_remaining"`
}

type TOTPFactor struct {
	ID         uuid.UUID
	Ciphertext []byte
	Status     string
}

type TOTPEnrollment struct {
	Secret     string    `json:"secret"`
	OTPAuthURI string    `json:"otpauth_uri"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type VerifyTOTPInput struct {
	Code string `json:"code"`
}

type StepUpInput struct {
	Method string `json:"method"`
	Proof  string `json:"proof"`
}

type RecoveryCodes struct {
	Codes []string `json:"codes"`
}

type OAuthAccount struct {
	ID          uuid.UUID `json:"id"`
	Provider    string    `json:"provider"`
	AccountHint string    `json:"account_hint"`
	Status      string    `json:"status"`
	BoundAt     time.Time `json:"bound_at"`
}

type OAuthStart struct {
	AuthorizationURL string    `json:"authorization_url"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type OAuthCompleteInput struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type OAuthChallenge struct {
	Provider              string
	StateHash, CodeHash   []byte
	PKCEVerifierEncrypted []byte
	PKCEChallenge         string
	ExpiresAt             time.Time
}

type OAuthIdentity struct {
	Provider, Subject, AccountHint string
}

type Repository interface {
	MFAStatus(context.Context, uuid.UUID) (MFAStatus, error)
	ReplacePendingTOTP(context.Context, Principal, []byte) error
	PendingTOTP(context.Context, uuid.UUID) (TOTPFactor, error)
	ActiveTOTP(context.Context, uuid.UUID) (TOTPFactor, error)
	ActivateTOTP(context.Context, Principal, uuid.UUID, [][]byte) error
	DisableTOTP(context.Context, Principal) error
	RotateRecoveryCodes(context.Context, Principal, [][]byte) error
	PasswordHash(context.Context, uuid.UUID) (string, error)
	ListOAuth(context.Context, uuid.UUID) ([]OAuthAccount, error)
	SaveOAuthChallenge(context.Context, Principal, OAuthChallenge) error
	CompleteOAuth(context.Context, Principal, OAuthChallenge, OAuthIdentity) (OAuthAccount, error)
	DeleteOAuth(context.Context, Principal, string) error
}
