package repository

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

func TestValidLoginCaptchaType(t *testing.T) {
	for _, value := range []string{"click", "slide", "drag", "rotate"} {
		if !validLoginCaptchaType(value) {
			t.Fatalf("supported type %q was rejected", value)
		}
	}
	for _, value := range []string{"", "CLICK", "text", "slide "} {
		if validLoginCaptchaType(value) {
			t.Fatalf("unsupported type %q was accepted", value)
		}
	}
}

func TestLoginCaptchaRepositoryRejectsInvalidInputsBeforeDatabaseAccess(t *testing.T) {
	repository := &Postgres{}
	if _, err := repository.CreateLoginCaptcha(context.Background(), domain.LoginCaptchaChallenge{
		ID: uuid.New(), ScopeHash: make([]byte, 32), CaptchaType: "text", ProofHash: make([]byte, 32),
		CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
	}); err == nil {
		t.Fatal("unsupported challenge type must fail")
	}
	if err := repository.VerifyLoginCaptcha(context.Background(), domain.LoginCaptchaAttempt{
		ID: uuid.New(), ScopeHash: []byte("short"), ProofHash: make([]byte, 32), Now: time.Now(),
	}); !errors.Is(err, domain.ErrLoginCaptchaInvalid) {
		t.Fatalf("invalid proof input must fail closed, got %v", err)
	}
}

func TestLoginCaptchaProofMatchesHashAndDatabaseType(t *testing.T) {
	hash := bytes.Repeat([]byte{0x41}, 32)
	input := domain.LoginCaptchaAttempt{CaptchaType: "slide", ProofHash: append([]byte(nil), hash...)}
	if !loginCaptchaProofMatches("slide", hash, input) {
		t.Fatal("matching proof hash and type must pass")
	}
	if loginCaptchaProofMatches("drag", hash, input) {
		t.Fatal("database and authenticated proof types must agree")
	}
	changed := append([]byte(nil), hash...)
	changed[0] ^= 0xff
	if loginCaptchaProofMatches("slide", changed, input) {
		t.Fatal("proof hash mismatch must fail")
	}
}
