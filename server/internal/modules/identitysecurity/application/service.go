package application

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	identity "github.com/appkernia/appkernia/server/internal/modules/identitysecurity/domain"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Sealer interface {
	Seal([]byte, string) ([]byte, int32, error)
	Open([]byte, string) ([]byte, error)
}

type Config struct {
	MFAEnabled    bool
	OAuthEnabled  bool
	OAuthAdapter  string
	AdminBaseURL  string
	EnrollmentTTL time.Duration
	OAuthStateTTL time.Duration
}

type Service struct {
	auth   Authenticator
	repo   identity.Repository
	sealer Sealer
	config Config
	now    func() time.Time
}

func NewService(auth Authenticator, repo identity.Repository, sealer Sealer, config Config) *Service {
	if config.EnrollmentTTL <= 0 {
		config.EnrollmentTTL = 10 * time.Minute
	}
	if config.OAuthStateTTL <= 0 {
		config.OAuthStateTTL = 5 * time.Minute
	}
	return &Service{auth: auth, repo: repo, sealer: sealer, config: config, now: time.Now}
}

func (service *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	authenticated, err := service.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return authenticated, err
	}
	if !slices.Contains(authenticated.Permissions, permission) {
		return authenticated, identity.ErrForbidden
	}
	return authenticated, nil
}

func principal(authenticated iamdomain.AuthenticatedContext, input identity.Principal) identity.Principal {
	input.UserID = authenticated.User.ID
	input.TenantID = authenticated.Tenant.ID
	input.SessionID = authenticated.SessionID
	return input
}

func mfaAAD(userID string) string { return "identity-security:totp:" + userID }
func oauthAAD(userID, provider string) string {
	return "identity-security:oauth:" + userID + ":" + provider
}

func (service *Service) MFAStatus(ctx context.Context, token string) (identity.MFAStatus, error) {
	authenticated, err := service.authorize(ctx, token, "iam.mfa.manage_self")
	if err != nil {
		return identity.MFAStatus{}, err
	}
	if !service.config.MFAEnabled {
		return identity.MFAStatus{}, identity.ErrFeatureDisabled
	}
	return service.repo.MFAStatus(ctx, authenticated.User.ID)
}

func (service *Service) EnrollTOTP(ctx context.Context, token string, input identity.Principal) (identity.TOTPEnrollment, error) {
	authenticated, err := service.authorize(ctx, token, "iam.mfa.manage_self")
	if err != nil {
		return identity.TOTPEnrollment{}, err
	}
	if !service.config.MFAEnabled {
		return identity.TOTPEnrollment{}, identity.ErrFeatureDisabled
	}
	secret, err := totpSecret()
	if err != nil {
		return identity.TOTPEnrollment{}, err
	}
	ciphertext, _, err := service.sealer.Seal([]byte(secret), mfaAAD(authenticated.User.ID.String()))
	if err != nil {
		return identity.TOTPEnrollment{}, err
	}
	if err = service.repo.ReplacePendingTOTP(ctx, principal(authenticated, input), ciphertext); err != nil {
		return identity.TOTPEnrollment{}, err
	}
	return identity.TOTPEnrollment{Secret: secret, OTPAuthURI: totpURI(secret, authenticated.User.Email), ExpiresAt: service.now().UTC().Add(service.config.EnrollmentTTL)}, nil
}

func (service *Service) VerifyTOTP(ctx context.Context, token string, input identity.Principal, verification identity.VerifyTOTPInput) (identity.RecoveryCodes, error) {
	authenticated, err := service.authorize(ctx, token, "iam.mfa.manage_self")
	if err != nil {
		return identity.RecoveryCodes{}, err
	}
	if !service.config.MFAEnabled {
		return identity.RecoveryCodes{}, identity.ErrFeatureDisabled
	}
	factor, err := service.repo.PendingTOTP(ctx, authenticated.User.ID)
	if err != nil {
		return identity.RecoveryCodes{}, err
	}
	plaintext, err := service.sealer.Open(factor.Ciphertext, mfaAAD(authenticated.User.ID.String()))
	if err != nil || !verifyTOTP(string(plaintext), strings.TrimSpace(verification.Code), service.now().UTC()) {
		return identity.RecoveryCodes{}, identity.ErrInvalid
	}
	codes, hashes, err := recoveryCodes()
	if err != nil {
		return identity.RecoveryCodes{}, err
	}
	if err = service.repo.ActivateTOTP(ctx, principal(authenticated, input), factor.ID, hashes); err != nil {
		return identity.RecoveryCodes{}, err
	}
	return identity.RecoveryCodes{Codes: codes}, nil
}

func (service *Service) validateStepUp(ctx context.Context, userID uuid.UUID, proof identity.StepUpInput) error {
	proof.Method = strings.TrimSpace(proof.Method)
	proof.Proof = strings.TrimSpace(proof.Proof)
	switch proof.Method {
	case "password":
		hash, err := service.repo.PasswordHash(ctx, userID)
		if err != nil || !iamapp.VerifyPassword(hash, proof.Proof) {
			return identity.ErrStepUpRequired
		}
	case "totp":
		factor, err := service.repo.ActiveTOTP(ctx, userID)
		if err != nil {
			return identity.ErrStepUpRequired
		}
		plaintext, err := service.sealer.Open(factor.Ciphertext, mfaAAD(userID.String()))
		if err != nil || !verifyTOTP(string(plaintext), proof.Proof, service.now().UTC()) {
			return identity.ErrStepUpRequired
		}
	default:
		return identity.ErrStepUpRequired
	}
	return nil
}

func (service *Service) DisableTOTP(ctx context.Context, token string, input identity.Principal, proof identity.StepUpInput) error {
	authenticated, err := service.authorize(ctx, token, "iam.mfa.manage_self")
	if err != nil {
		return err
	}
	if !service.config.MFAEnabled {
		return identity.ErrFeatureDisabled
	}
	if err = service.validateStepUp(ctx, authenticated.User.ID, proof); err != nil {
		return err
	}
	return service.repo.DisableTOTP(ctx, principal(authenticated, input))
}

func (service *Service) RotateRecoveryCodes(ctx context.Context, token string, input identity.Principal, proof identity.StepUpInput) (identity.RecoveryCodes, error) {
	authenticated, err := service.authorize(ctx, token, "iam.mfa.manage_self")
	if err != nil {
		return identity.RecoveryCodes{}, err
	}
	if !service.config.MFAEnabled {
		return identity.RecoveryCodes{}, identity.ErrFeatureDisabled
	}
	if err = service.validateStepUp(ctx, authenticated.User.ID, proof); err != nil {
		return identity.RecoveryCodes{}, err
	}
	codes, hashes, err := recoveryCodes()
	if err != nil {
		return identity.RecoveryCodes{}, err
	}
	if err = service.repo.RotateRecoveryCodes(ctx, principal(authenticated, input), hashes); err != nil {
		return identity.RecoveryCodes{}, err
	}
	return identity.RecoveryCodes{Codes: codes}, nil
}

func (service *Service) OAuthAccounts(ctx context.Context, token string) ([]identity.OAuthAccount, error) {
	authenticated, err := service.authorize(ctx, token, "iam.oauth.manage_self")
	if err != nil {
		return nil, err
	}
	if !service.config.OAuthEnabled {
		return nil, identity.ErrFeatureDisabled
	}
	return service.repo.ListOAuth(ctx, authenticated.User.ID)
}

func (service *Service) StartOAuth(ctx context.Context, token string, input identity.Principal, provider string) (identity.OAuthStart, error) {
	authenticated, err := service.authorize(ctx, token, "iam.oauth.manage_self")
	if err != nil {
		return identity.OAuthStart{}, err
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if !service.config.OAuthEnabled {
		return identity.OAuthStart{}, identity.ErrFeatureDisabled
	}
	if service.config.OAuthAdapter != "local-mock" || provider != "local" {
		return identity.OAuthStart{}, identity.ErrProviderDisabled
	}
	state, err := opaqueToken(32)
	if err != nil {
		return identity.OAuthStart{}, err
	}
	verifier, err := opaqueToken(48)
	if err != nil {
		return identity.OAuthStart{}, err
	}
	code, err := opaqueToken(32)
	if err != nil {
		return identity.OAuthStart{}, err
	}
	encryptedVerifier, _, err := service.sealer.Seal([]byte(verifier), oauthAAD(authenticated.User.ID.String(), provider))
	if err != nil {
		return identity.OAuthStart{}, err
	}
	expiresAt := service.now().UTC().Add(service.config.OAuthStateTTL)
	challenge := identity.OAuthChallenge{Provider: provider, StateHash: sha256Bytes(state), CodeHash: sha256Bytes(code), PKCEVerifierEncrypted: encryptedVerifier, PKCEChallenge: pkceChallenge(verifier), ExpiresAt: expiresAt}
	if err = service.repo.SaveOAuthChallenge(ctx, principal(authenticated, input), challenge); err != nil {
		return identity.OAuthStart{}, err
	}
	callback := strings.TrimRight(service.config.AdminBaseURL, "/") + "/auth/callback/" + provider + "?code=" + code + "&state=" + state
	return identity.OAuthStart{AuthorizationURL: callback, ExpiresAt: expiresAt}, nil
}

func (service *Service) CompleteOAuth(ctx context.Context, token string, input identity.Principal, provider string, completion identity.OAuthCompleteInput) (identity.OAuthAccount, error) {
	authenticated, err := service.authorize(ctx, token, "iam.oauth.manage_self")
	if err != nil {
		return identity.OAuthAccount{}, err
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if !service.config.OAuthEnabled || service.config.OAuthAdapter != "local-mock" || provider != "local" {
		return identity.OAuthAccount{}, identity.ErrProviderDisabled
	}
	if len(completion.Code) < 32 || len(completion.State) < 32 {
		return identity.OAuthAccount{}, identity.ErrOAuthState
	}
	subjectDigest := sha256Bytes(completion.Code)
	subject := fmt.Sprintf("local:%x", subjectDigest)
	accountHint := "local-" + fmt.Sprintf("%x", subjectDigest[:4])
	challenge := identity.OAuthChallenge{Provider: provider, StateHash: sha256Bytes(completion.State), CodeHash: subjectDigest, ExpiresAt: service.now().UTC()}
	return service.repo.CompleteOAuth(ctx, principal(authenticated, input), challenge, identity.OAuthIdentity{Provider: provider, Subject: subject, AccountHint: accountHint})
}

func (service *Service) DeleteOAuth(ctx context.Context, token string, input identity.Principal, provider string) error {
	authenticated, err := service.authorize(ctx, token, "iam.oauth.manage_self")
	if err != nil {
		return err
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if !service.config.OAuthEnabled || provider == "" {
		return identity.ErrFeatureDisabled
	}
	return service.repo.DeleteOAuth(ctx, principal(authenticated, input), provider)
}
