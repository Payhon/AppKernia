package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	platformcaptcha "github.com/appkernia/appkernia/server/internal/platform/captcha"
	"github.com/google/uuid"
)

const (
	AdminLoginCaptchaTypeSettingKey = "admin.login_captcha.type"

	loginCaptchaThreshold = 3
	loginFailureWindow    = 30 * time.Minute
	loginCaptchaLifetime  = 5 * time.Minute
	loginCaptchaClockSkew = 30 * time.Second
)

// WithLoginCaptchaTypeProvider injects the global setting lookup without
// coupling IAM to the system-settings module.
func WithLoginCaptchaTypeProvider(provider func(context.Context) (platformcaptcha.Type, error)) AuthOption {
	return func(service *AuthService) {
		service.loginCaptchaTypeProvider = provider
	}
}

type LoginCaptchaInput struct {
	ID       uuid.UUID
	Token    string
	Response platformcaptcha.Response
}

type LoginCaptcha struct {
	ID             uuid.UUID
	Token          string
	Type           platformcaptcha.Type
	ExpiresInSec   int64
	Image          platformcaptcha.Image
	PromptImage    *platformcaptcha.Image
	RequiredPoints int
	TileImage      *platformcaptcha.Image
	InitialPoint   *platformcaptcha.Point
	ThumbImage     *platformcaptcha.Image
}

func (service *AuthService) CreateLoginCaptcha(ctx context.Context, email, audience string, client ClientMetadata) (LoginCaptcha, error) {
	normalizedEmail, validEmail := normalizeEmail(email)
	if !validEmail {
		return LoginCaptcha{}, ErrProfileValidation
	}
	if audience != "ak-admin" {
		return LoginCaptcha{}, ErrAudienceMismatch
	}
	now := service.clock().UTC()
	scopeHash := loginScopeHash(service.loginProtectionKey, normalizedEmail, audience, client.IPAddress)
	if err := service.identities.CheckLoginCaptchaGeneration(ctx, scopeHash, now); err != nil {
		return LoginCaptcha{}, mapLoginCaptchaGenerationError(err)
	}
	kind := platformcaptcha.TypeSlide
	if service.loginCaptchaTypeProvider != nil {
		var err error
		kind, err = service.loginCaptchaTypeProvider(ctx)
		if err != nil {
			return LoginCaptcha{}, fmt.Errorf("resolve login captcha type: %w", err)
		}
	}
	if !kind.Valid() {
		return LoginCaptcha{}, fmt.Errorf("resolve login captcha type: %w", platformcaptcha.ErrInvalidType)
	}
	public, solution, err := service.loginCaptcha.Generate(kind)
	if err != nil {
		return LoginCaptcha{}, fmt.Errorf("generate login captcha: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return LoginCaptcha{}, fmt.Errorf("create login captcha id: %w", err)
	}
	expiresAt := now.Add(loginCaptchaLifetime)
	proof, err := platformcaptcha.NewProof(id.String(), scopeHash, now, expiresAt, solution)
	if err != nil {
		return LoginCaptcha{}, fmt.Errorf("create login captcha proof: %w", err)
	}
	token, err := service.loginCaptchaCodec.Seal(proof)
	if err != nil {
		return LoginCaptcha{}, fmt.Errorf("seal login captcha proof: %w", err)
	}
	storedID, err := service.identities.CreateLoginCaptcha(ctx, domain.LoginCaptchaChallenge{
		ID: id, ScopeHash: scopeHash, CaptchaType: string(kind), ProofHash: platformcaptcha.TokenHash(token),
		CreatedAt: now, ExpiresAt: expiresAt,
	})
	if err != nil {
		return LoginCaptcha{}, mapLoginCaptchaGenerationError(err)
	}
	if storedID != id {
		return LoginCaptcha{}, fmt.Errorf("store login captcha: challenge id mismatch")
	}
	return LoginCaptcha{
		ID: id, Token: token, Type: public.Type, ExpiresInSec: int64(expiresAt.Sub(now) / time.Second),
		Image: public.Image, PromptImage: public.PromptImage, RequiredPoints: public.RequiredPoints,
		TileImage: public.TileImage, InitialPoint: public.InitialPoint, ThumbImage: public.ThumbImage,
	}, nil
}

func (service *AuthService) verifyLoginCaptcha(ctx context.Context, value *LoginCaptchaInput, scopeHash []byte, now time.Time) error {
	if value == nil {
		return ErrCaptchaRequired
	}
	if value.ID == uuid.Nil || value.Token == "" || strings.TrimSpace(value.Token) != value.Token {
		return ErrCaptchaInvalid
	}
	proof, err := service.loginCaptchaCodec.Open(value.Token)
	if err != nil || proof.ChallengeID != value.ID.String() || !hmac.Equal(proof.ScopeHash, scopeHash) ||
		proof.IssuedAt > now.Add(loginCaptchaClockSkew).Unix() || proof.ExpiresAt <= now.Unix() {
		return ErrCaptchaInvalid
	}
	valid, validationErr := platformcaptcha.Validate(proof.Solution, value.Response)
	err = service.identities.VerifyLoginCaptcha(ctx, domain.LoginCaptchaAttempt{
		ID: value.ID, ScopeHash: scopeHash, CaptchaType: string(proof.Solution.Type),
		ProofHash: platformcaptcha.TokenHash(value.Token), Valid: validationErr == nil && valid, Now: now,
	})
	if err != nil {
		if errors.Is(err, domain.ErrLoginCaptchaInvalid) {
			return ErrCaptchaInvalid
		}
		return fmt.Errorf("verify login captcha: %w", err)
	}
	if validationErr != nil || !valid {
		return ErrCaptchaInvalid
	}
	return nil
}

func mapLoginCaptchaGenerationError(err error) error {
	switch {
	case errors.Is(err, domain.ErrLoginCaptchaNotRequired):
		return ErrCaptchaNotRequired
	case errors.Is(err, domain.ErrLoginCaptchaCooldown):
		return ErrCaptchaCooldown
	default:
		return fmt.Errorf("store login captcha: %w", err)
	}
}

func loginScopeHash(key []byte, email, audience string, ipAddress *netip.Addr) []byte {
	ip := "unknown"
	if ipAddress != nil {
		ip = ipAddress.Unmap().String()
	}
	value := strings.ToLower(strings.TrimSpace(email)) + "\n" + strings.TrimSpace(audience) + "\n" + ip
	hasher := hmac.New(sha256.New, key)
	_, _ = hasher.Write([]byte(value))
	return hasher.Sum(nil)
}
