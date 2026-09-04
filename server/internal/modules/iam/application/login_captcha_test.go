package application

import (
	"bytes"
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	platformcaptcha "github.com/appkernia/appkernia/server/internal/platform/captcha"
	"github.com/google/uuid"
)

type captchaRepositoryStub struct {
	domain.Repository
	checkErr      error
	checkCalls    int
	created       domain.LoginCaptchaChallenge
	required      bool
	requiredCalls int
	attempt       domain.LoginCaptchaAttempt
	verifyCalls   int
	failureCount  int32
}

func (repository *captchaRepositoryStub) CheckLoginCaptchaGeneration(context.Context, []byte, time.Time) error {
	repository.checkCalls++
	return repository.checkErr
}

func (repository *captchaRepositoryStub) CreateLoginCaptcha(_ context.Context, input domain.LoginCaptchaChallenge) (uuid.UUID, error) {
	repository.created = input
	return input.ID, nil
}

func (repository *captchaRepositoryStub) LoginCaptchaRequired(context.Context, []byte, time.Time) (bool, error) {
	repository.requiredCalls++
	return repository.required, nil
}

func (repository *captchaRepositoryStub) VerifyLoginCaptcha(_ context.Context, input domain.LoginCaptchaAttempt) error {
	repository.verifyCalls++
	repository.attempt = input
	if !input.Valid {
		return domain.ErrLoginCaptchaInvalid
	}
	return nil
}

func (repository *captchaRepositoryStub) FindCredentialByEmail(context.Context, string) (domain.Credential, error) {
	return domain.Credential{}, domain.ErrIdentityNotFound
}

func (repository *captchaRepositoryStub) RecordLoginFailure(context.Context, domain.LoginFailure) (int32, error) {
	return repository.failureCount, nil
}

func TestCreateLoginCaptchaRequiresAdminThresholdBeforeGeneration(t *testing.T) {
	repository := &captchaRepositoryStub{checkErr: domain.ErrLoginCaptchaNotRequired}
	service := newCaptchaTestService(t, repository, platformcaptcha.TypeSlide)
	if _, err := service.CreateLoginCaptcha(context.Background(), "admin@example.test", "ak-admin", ClientMetadata{}); !errors.Is(err, ErrCaptchaNotRequired) {
		t.Fatalf("below-threshold generation must fail, got %v", err)
	}
	if repository.checkCalls != 1 || repository.created.ID != uuid.Nil {
		t.Fatalf("challenge generation must stop at precheck: calls=%d created=%v", repository.checkCalls, repository.created.ID)
	}
	repository.checkErr = nil
	if _, err := service.CreateLoginCaptcha(context.Background(), "admin@example.test", "ak-mobile", ClientMetadata{}); !errors.Is(err, ErrAudienceMismatch) {
		t.Fatalf("mobile audience must be rejected, got %v", err)
	}
	if repository.checkCalls != 1 {
		t.Fatal("non-admin request must not read login protection state")
	}
}

func TestCreateLoginCaptchaMapsCooldownAndStoresOnlyProofHash(t *testing.T) {
	repository := &captchaRepositoryStub{checkErr: domain.ErrLoginCaptchaCooldown}
	service := newCaptchaTestService(t, repository, platformcaptcha.TypeRotate)
	if _, err := service.CreateLoginCaptcha(context.Background(), "admin@example.test", "ak-admin", ClientMetadata{}); !errors.Is(err, ErrCaptchaCooldown) {
		t.Fatalf("cooldown must map to application error, got %v", err)
	}
	repository.checkErr = nil
	challenge, err := service.CreateLoginCaptcha(context.Background(), "admin@example.test", "ak-admin", ClientMetadata{})
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	if challenge.Type != platformcaptcha.TypeRotate || challenge.ThumbImage == nil || challenge.Token == "" || challenge.ExpiresInSec != 300 {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}
	if repository.created.CaptchaType != "rotate" || len(repository.created.ProofHash) != 32 || bytes.Contains(repository.created.ProofHash, []byte(challenge.Token)) {
		t.Fatalf("repository must receive only a proof hash: %+v", repository.created)
	}
}

func TestVerifyLoginCaptchaAuthenticatesProofAndPersistsOutcome(t *testing.T) {
	repository := &captchaRepositoryStub{}
	service := newCaptchaTestService(t, repository, platformcaptcha.TypeSlide)
	now := time.Unix(1_800_000_000, 0).UTC()
	service.clock = func() time.Time { return now }
	ip := netip.MustParseAddr("127.0.0.1")
	scopeHash := loginScopeHash(service.loginProtectionKey, "admin@example.test", "ak-admin", &ip)
	id := uuid.Must(uuid.NewV7())
	target := platformcaptcha.Point{X: 80, Y: 40}
	proof, err := platformcaptcha.NewProof(id.String(), scopeHash, now, now.Add(5*time.Minute), platformcaptcha.Solution{
		Type: platformcaptcha.TypeSlide, CanvasWidth: 320, CanvasHeight: 180, Point: &target,
	})
	if err != nil {
		t.Fatal(err)
	}
	token, err := service.loginCaptchaCodec.Seal(proof)
	if err != nil {
		t.Fatal(err)
	}
	responsePoint := target
	if err = service.verifyLoginCaptcha(context.Background(), &LoginCaptchaInput{
		ID: id, Token: token, Response: platformcaptcha.Response{Type: platformcaptcha.TypeSlide, Point: &responsePoint},
	}, scopeHash, now); err != nil {
		t.Fatalf("valid proof must pass: %v", err)
	}
	if !repository.attempt.Valid || repository.attempt.CaptchaType != "slide" || !bytes.Equal(repository.attempt.ProofHash, platformcaptcha.TokenHash(token)) {
		t.Fatalf("unexpected persisted attempt: %+v", repository.attempt)
	}
	wrongPoint := platformcaptcha.Point{X: 200, Y: 40}
	if err = service.verifyLoginCaptcha(context.Background(), &LoginCaptchaInput{
		ID: id, Token: token, Response: platformcaptcha.Response{Type: platformcaptcha.TypeSlide, Point: &wrongPoint},
	}, scopeHash, now); !errors.Is(err, ErrCaptchaInvalid) {
		t.Fatalf("wrong response must fail, got %v", err)
	}
	if repository.attempt.Valid {
		t.Fatal("wrong response must be recorded as an invalid attempt")
	}
}

func TestVerifyLoginCaptchaRejectsUnboundOrExpiredProofBeforeDatabase(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	scopeHash := bytes.Repeat([]byte{0x21}, 32)
	id := uuid.Must(uuid.NewV7())
	target := platformcaptcha.Point{X: 80, Y: 40}
	proofSolution := platformcaptcha.Solution{Type: platformcaptcha.TypeSlide, CanvasWidth: 320, CanvasHeight: 180, Point: &target}

	newInput := func(t *testing.T) (*captchaRepositoryStub, *AuthService, LoginCaptchaInput) {
		t.Helper()
		repository := &captchaRepositoryStub{}
		service := newCaptchaTestService(t, repository, platformcaptcha.TypeSlide)
		proof, err := platformcaptcha.NewProof(id.String(), scopeHash, now, now.Add(5*time.Minute), proofSolution)
		if err != nil {
			t.Fatal(err)
		}
		token, err := service.loginCaptchaCodec.Seal(proof)
		if err != nil {
			t.Fatal(err)
		}
		point := target
		return repository, service, LoginCaptchaInput{
			ID: id, Token: token, Response: platformcaptcha.Response{Type: platformcaptcha.TypeSlide, Point: &point},
		}
	}

	t.Run("challenge id", func(t *testing.T) {
		repository, service, input := newInput(t)
		input.ID = uuid.New()
		if err := service.verifyLoginCaptcha(context.Background(), &input, scopeHash, now); !errors.Is(err, ErrCaptchaInvalid) || repository.verifyCalls != 0 {
			t.Fatalf("id mismatch must fail before persistence: err=%v calls=%d", err, repository.verifyCalls)
		}
	})
	t.Run("scope", func(t *testing.T) {
		repository, service, input := newInput(t)
		otherScope := append([]byte(nil), scopeHash...)
		otherScope[0] ^= 0xff
		if err := service.verifyLoginCaptcha(context.Background(), &input, otherScope, now); !errors.Is(err, ErrCaptchaInvalid) || repository.verifyCalls != 0 {
			t.Fatalf("scope mismatch must fail before persistence: err=%v calls=%d", err, repository.verifyCalls)
		}
	})
	t.Run("expiry", func(t *testing.T) {
		repository, service, input := newInput(t)
		if err := service.verifyLoginCaptcha(context.Background(), &input, scopeHash, now.Add(5*time.Minute)); !errors.Is(err, ErrCaptchaInvalid) || repository.verifyCalls != 0 {
			t.Fatalf("expired proof must fail before persistence: err=%v calls=%d", err, repository.verifyCalls)
		}
	})
	t.Run("authentication tag", func(t *testing.T) {
		repository, service, input := newInput(t)
		tampered := []byte(input.Token)
		if tampered[0] == 'A' {
			tampered[0] = 'B'
		} else {
			tampered[0] = 'A'
		}
		input.Token = string(tampered)
		if err := service.verifyLoginCaptcha(context.Background(), &input, scopeHash, now); !errors.Is(err, ErrCaptchaInvalid) || repository.verifyCalls != 0 {
			t.Fatalf("tampered proof must fail before persistence: err=%v calls=%d", err, repository.verifyCalls)
		}
	})
}

func TestLoginProtectionIsAdminOnly(t *testing.T) {
	repository := &captchaRepositoryStub{required: true, failureCount: loginCaptchaThreshold}
	service := newCaptchaTestService(t, repository, platformcaptcha.TypeSlide)
	_, err := service.Login(context.Background(), LoginInput{
		Email: "mobile@example.test", Password: "wrong", Audience: "ak-mobile",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("mobile login must keep generic credential error, got %v", err)
	}
	if repository.requiredCalls != 0 {
		t.Fatal("mobile login must not enter CAPTCHA protection")
	}
}

func TestLoginScopeHashNormalizesEmailAndIPAddress(t *testing.T) {
	key := []byte("test-login-protection-key")
	ipv4Mapped := netip.MustParseAddr("::ffff:127.0.0.1")
	ipv4 := netip.MustParseAddr("127.0.0.1")
	first := loginScopeHash(key, " Admin@Example.Test ", "ak-admin", &ipv4Mapped)
	second := loginScopeHash(key, "admin@example.test", "ak-admin", &ipv4)
	if !bytes.Equal(first, second) {
		t.Fatal("equivalent identifiers must have the same protection scope")
	}
	apiScope := loginScopeHash(key, "admin@example.test", "ak-api", &ipv4)
	if bytes.Equal(first, apiScope) {
		t.Fatal("token audiences must have isolated protection scopes")
	}
	otherEmailScope := loginScopeHash(key, "other@example.test", "ak-admin", &ipv4)
	if bytes.Equal(first, otherEmailScope) {
		t.Fatal("email addresses must have isolated protection scopes")
	}
	otherIP := netip.MustParseAddr("127.0.0.2")
	otherIPScope := loginScopeHash(key, "admin@example.test", "ak-admin", &otherIP)
	if bytes.Equal(first, otherIPScope) {
		t.Fatal("source IP addresses must have isolated protection scopes")
	}
}

func newCaptchaTestService(t *testing.T, repository domain.Repository, kind platformcaptcha.Type) *AuthService {
	t.Helper()
	service, err := NewAuthService(repository, nil, nil,
		WithLoginProtectionKey([]byte("captcha-test-key")),
		WithLoginCaptchaTypeProvider(func(context.Context) (platformcaptcha.Type, error) { return kind, nil }),
	)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	return service
}
