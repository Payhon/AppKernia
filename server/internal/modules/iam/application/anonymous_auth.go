package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

const (
	passwordResetLifetime = 15 * time.Minute
	passwordResetCooldown = 60
)

type AnonymousAuthConfig struct {
	AdminRegistrationEnabled bool
	RegistrationTenantCode   string
	PasswordRecoveryEnabled  bool
}

type AuthOption func(*AuthService)

func WithLoginProtectionKey(key []byte) AuthOption {
	return func(service *AuthService) {
		digest := sha256.Sum256(append([]byte("appkernia-login-protection\x00"), key...))
		service.loginProtectionKey = digest[:]
	}
}

func WithAnonymousAuth(config AnonymousAuthConfig, notifier PasswordResetNotifier) AuthOption {
	return func(service *AuthService) {
		service.anonymous = config
		if notifier != nil {
			service.resetNotifier = notifier
		}
	}
}

type PasswordResetNotification struct {
	TenantID uuid.UUID
	Email    string
	Locale   string
	Token    string
}

type PasswordResetNotifier interface {
	SendPasswordReset(context.Context, PasswordResetNotification) error
}

type disabledPasswordResetNotifier struct{}

func (disabledPasswordResetNotifier) SendPasswordReset(context.Context, PasswordResetNotification) error {
	return ErrFeatureDisabled
}

type RegisterInput struct {
	Email       string
	DisplayName string
	Password    string
	Locale      string
	AcceptTerms bool
	RequestID   string
	Client      ClientMetadata
}

type ForgotPasswordInput struct {
	Email     string
	RequestID string
	Client    ClientMetadata
}

type ResetPasswordInput struct {
	Token       string
	NewPassword string
	RequestID   string
	Client      ClientMetadata
}

func (service *AuthService) Register(ctx context.Context, input RegisterInput) error {
	if !service.anonymous.AdminRegistrationEnabled {
		return ErrFeatureDisabled
	}
	email, valid := normalizeEmail(input.Email)
	displayName := strings.TrimSpace(input.DisplayName)
	locale := strings.TrimSpace(input.Locale)
	if !valid || len(displayName) < 2 || len(displayName) > 120 ||
		(locale != "zh-CN" && locale != "en-US") || !input.AcceptTerms {
		return ErrRegistrationValidation
	}
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return ErrPasswordValidation
	}
	err = service.identities.RegisterAdmin(ctx, domain.RegisterAdmin{
		TenantCode: strings.ToLower(strings.TrimSpace(service.anonymous.RegistrationTenantCode)),
		Email:      email, DisplayName: displayName, Locale: locale, PasswordHash: passwordHash,
		RequestID: input.RequestID, IPAddress: input.Client.IPAddress, UserAgent: input.Client.UserAgent,
	})
	if errors.Is(err, domain.ErrEmailAlreadyExists) {
		return nil
	}
	return err
}

func (service *AuthService) ForgotPassword(ctx context.Context, input ForgotPasswordInput) (int, error) {
	if !service.anonymous.PasswordRecoveryEnabled {
		return 0, ErrFeatureDisabled
	}
	email, valid := normalizeEmail(input.Email)
	if !valid {
		return 0, ErrRegistrationValidation
	}
	plainToken, secretHash, err := NewOpaqueToken()
	if err != nil {
		return 0, err
	}
	recipient, err := service.identities.PreparePasswordReset(ctx, domain.PreparePasswordReset{
		Email: email, TargetHash: HashOpaqueToken(email), SecretHash: secretHash,
		ExpiresAt: service.clock().UTC().Add(passwordResetLifetime), RequestID: input.RequestID,
		IPAddress: input.Client.IPAddress,
	})
	if err != nil {
		return 0, err
	}
	if recipient != nil {
		_ = service.resetNotifier.SendPasswordReset(ctx, PasswordResetNotification{
			TenantID: recipient.TenantID, Email: recipient.Email, Locale: recipient.Locale, Token: plainToken,
		})
	}
	return passwordResetCooldown, nil
}

func (service *AuthService) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	if !service.anonymous.PasswordRecoveryEnabled {
		return ErrFeatureDisabled
	}
	token := strings.TrimSpace(input.Token)
	if len(token) < 32 || len(token) > 512 {
		return ErrResetTokenInvalid
	}
	tokenHash := HashOpaqueToken(token)
	state, err := service.identities.GetPasswordResetState(ctx, tokenHash)
	if errors.Is(err, domain.ErrResetTokenInvalid) {
		return ErrResetTokenInvalid
	}
	if err != nil {
		return err
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
	err = service.identities.ResetPassword(ctx, domain.ResetPassword{
		TokenHash: tokenHash, UserID: state.UserID, ExpectedHash: state.CurrentHash,
		ExpectedVersion: state.CurrentVersion, NewHash: newHash, RequestID: input.RequestID,
		IPAddress: input.Client.IPAddress, UserAgent: input.Client.UserAgent,
	})
	if errors.Is(err, domain.ErrResetTokenInvalid) {
		return ErrResetTokenInvalid
	}
	return err
}

func normalizeEmail(raw string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if len(normalized) < 3 || len(normalized) > 254 {
		return "", false
	}
	parsed, err := mail.ParseAddress(normalized)
	return normalized, err == nil && parsed.Address == normalized
}
