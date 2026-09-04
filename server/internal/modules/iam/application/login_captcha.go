package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	InteractiveCaptchaTypeSettingKey = "interactive_captcha.type"

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
	ID       uuid.UUID                `json:"id"`
	Token    string                   `json:"token"`
	Response platformcaptcha.Response `json:"response"`
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

// InteractiveCaptchaScope is the authenticated server-side binding for a
// reusable CAPTCHA. Callers must pass normalized targets and trusted session
// values; none of these fields is exposed in the proof token.
type InteractiveCaptchaScope struct {
	Audience  string
	AppID     uuid.UUID
	Scene     string
	Target    string
	UserID    uuid.UUID
	SessionID uuid.UUID
	Purpose   string
	Resource  string
	Client    ClientMetadata
}

func (service *AuthService) CreateLoginCaptcha(ctx context.Context, email, audience string, client ClientMetadata) (LoginCaptcha, error) {
	normalizedEmail, validEmail := normalizeEmail(email)
	if !validEmail {
		return LoginCaptcha{}, ErrProfileValidation
	}
	if audience != "ak-admin" {
		return LoginCaptcha{}, ErrAudienceMismatch
	}
	scopeHash := loginScopeHash(service.loginProtectionKey, normalizedEmail, audience, client.IPAddress)
	now := service.clock().UTC()
	if err := service.identities.CheckLoginCaptchaGeneration(ctx, scopeHash, now); err != nil {
		return LoginCaptcha{}, mapLoginCaptchaGenerationError(err)
	}
	return service.createInteractiveCaptcha(ctx, scopeHash, now, true)
}

func (service *AuthService) CreateInteractiveCaptcha(ctx context.Context, scope InteractiveCaptchaScope) (LoginCaptcha, error) {
	scopeHash, err := interactiveCaptchaScopeHash(service.loginProtectionKey, scope)
	if err != nil {
		return LoginCaptcha{}, err
	}
	return service.createInteractiveCaptcha(ctx, scopeHash, service.clock().UTC(), false)
}

func (service *AuthService) createInteractiveCaptcha(ctx context.Context, scopeHash []byte, now time.Time, loginGate bool) (LoginCaptcha, error) {
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
	challenge := domain.LoginCaptchaChallenge{
		ID: id, ScopeHash: scopeHash, CaptchaType: string(kind), ProofHash: platformcaptcha.TokenHash(token),
		CreatedAt: now, ExpiresAt: expiresAt,
	}
	var storedID uuid.UUID
	if loginGate {
		storedID, err = service.identities.CreateLoginCaptcha(ctx, challenge)
	} else {
		storedID, err = service.identities.CreateInteractiveCaptcha(ctx, challenge)
	}
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

func (service *AuthService) VerifyInteractiveCaptcha(ctx context.Context, value *LoginCaptchaInput, scope InteractiveCaptchaScope) error {
	scopeHash, err := interactiveCaptchaScopeHash(service.loginProtectionKey, scope)
	if err != nil {
		return ErrCaptchaInvalid
	}
	return service.verifyInteractiveCaptcha(ctx, value, scopeHash, service.clock().UTC(), true)
}

func (service *AuthService) verifyLoginCaptcha(ctx context.Context, value *LoginCaptchaInput, scopeHash []byte, now time.Time) error {
	return service.verifyInteractiveCaptcha(ctx, value, scopeHash, now, false)
}

func (service *AuthService) verifyInteractiveCaptcha(ctx context.Context, value *LoginCaptchaInput, scopeHash []byte, now time.Time, shared bool) error {
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
	attempt := domain.LoginCaptchaAttempt{
		ID: value.ID, ScopeHash: scopeHash, CaptchaType: string(proof.Solution.Type),
		ProofHash: platformcaptcha.TokenHash(value.Token), Valid: validationErr == nil && valid, Now: now,
	}
	if shared {
		err = service.identities.VerifyInteractiveCaptcha(ctx, attempt)
	} else {
		err = service.identities.VerifyLoginCaptcha(ctx, attempt)
	}
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

func interactiveCaptchaScopeHash(key []byte, scope InteractiveCaptchaScope) ([]byte, error) {
	audience, scene, target := strings.TrimSpace(scope.Audience), strings.TrimSpace(scope.Scene), strings.TrimSpace(scope.Target)
	deviceKey := strings.TrimSpace(scope.Client.DeviceKey)
	if audience != "ak-mobile" || scope.AppID == uuid.Nil || scene == "" || target == "" || scope.Client.IPAddress == nil || len(deviceKey) < 16 || len(deviceKey) > 512 {
		return nil, ErrProfileValidation
	}
	ip := scope.Client.IPAddress.Unmap().String()
	deviceDigest := sha256.Sum256([]byte(deviceKey))
	payload, err := json.Marshal(struct {
		Audience, AppID, Scene, Target, IP, DeviceHash, UserID, SessionID, Purpose, Resource string
	}{
		audience, scope.AppID.String(), scene, target, ip, base64.RawURLEncoding.EncodeToString(deviceDigest[:]),
		scope.UserID.String(), scope.SessionID.String(), strings.TrimSpace(scope.Purpose), strings.TrimSpace(scope.Resource),
	})
	if err != nil {
		return nil, fmt.Errorf("encode interactive captcha scope: %w", err)
	}
	hasher := hmac.New(sha256.New, key)
	_, _ = hasher.Write(payload)
	return hasher.Sum(nil), nil
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
