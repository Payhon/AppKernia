package application

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	identity "github.com/appkernia/appkernia/server/internal/modules/identitysecurity/domain"
	"github.com/google/uuid"
)

type testAuth struct {
	value iamdomain.AuthenticatedContext
}

func (auth testAuth) Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error) {
	return auth.value, nil
}

type testSealer struct{}

func (testSealer) Seal(value []byte, _ string) ([]byte, int32, error) {
	return append([]byte(nil), value...), 1, nil
}
func (testSealer) Open(value []byte, _ string) ([]byte, error) {
	return append([]byte(nil), value...), nil
}

type testRepo struct {
	pending       identity.TOTPFactor
	active        identity.TOTPFactor
	passwordHash  string
	recoveryCount int64
	challenge     identity.OAuthChallenge
	challengeUsed bool
}

func (repo *testRepo) MFAStatus(context.Context, uuid.UUID) (identity.MFAStatus, error) {
	return identity.MFAStatus{TOTPEnabled: repo.active.ID != uuid.Nil, RecoveryCodesRemaining: repo.recoveryCount}, nil
}
func (repo *testRepo) ReplacePendingTOTP(_ context.Context, _ identity.Principal, ciphertext []byte) error {
	repo.pending = identity.TOTPFactor{ID: uuid.New(), Ciphertext: ciphertext, Status: "pending"}
	return nil
}
func (repo *testRepo) PendingTOTP(context.Context, uuid.UUID) (identity.TOTPFactor, error) {
	if repo.pending.ID == uuid.Nil {
		return identity.TOTPFactor{}, identity.ErrNotFound
	}
	return repo.pending, nil
}
func (repo *testRepo) ActiveTOTP(context.Context, uuid.UUID) (identity.TOTPFactor, error) {
	if repo.active.ID == uuid.Nil {
		return identity.TOTPFactor{}, identity.ErrNotFound
	}
	return repo.active, nil
}
func (repo *testRepo) ActivateTOTP(_ context.Context, _ identity.Principal, factorID uuid.UUID, hashes [][]byte) error {
	if factorID != repo.pending.ID {
		return identity.ErrConflict
	}
	repo.active = repo.pending
	repo.active.Status = "active"
	repo.pending = identity.TOTPFactor{}
	repo.recoveryCount = int64(len(hashes))
	return nil
}
func (repo *testRepo) DisableTOTP(context.Context, identity.Principal) error {
	repo.active = identity.TOTPFactor{}
	repo.recoveryCount = 0
	return nil
}
func (repo *testRepo) RotateRecoveryCodes(_ context.Context, _ identity.Principal, hashes [][]byte) error {
	repo.recoveryCount = int64(len(hashes))
	return nil
}
func (repo *testRepo) PasswordHash(context.Context, uuid.UUID) (string, error) {
	return repo.passwordHash, nil
}
func (repo *testRepo) ListOAuth(context.Context, uuid.UUID) ([]identity.OAuthAccount, error) {
	return []identity.OAuthAccount{}, nil
}
func (repo *testRepo) SaveOAuthChallenge(_ context.Context, _ identity.Principal, challenge identity.OAuthChallenge) error {
	repo.challenge = challenge
	repo.challengeUsed = false
	return nil
}
func (repo *testRepo) CompleteOAuth(_ context.Context, _ identity.Principal, challenge identity.OAuthChallenge, oauth identity.OAuthIdentity) (identity.OAuthAccount, error) {
	if repo.challengeUsed || !equalBytes(repo.challenge.StateHash, challenge.StateHash) || !equalBytes(repo.challenge.CodeHash, challenge.CodeHash) {
		return identity.OAuthAccount{}, identity.ErrOAuthState
	}
	repo.challengeUsed = true
	return identity.OAuthAccount{ID: uuid.New(), Provider: oauth.Provider, AccountHint: oauth.AccountHint, Status: "active", BoundAt: time.Now()}, nil
}
func (repo *testRepo) DeleteOAuth(context.Context, identity.Principal, string) error { return nil }

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func TestTOTPEnrollmentRecoveryCodesAndStepUp(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	passwordHash, err := iamapp.HashPassword("identity security password 2026!")
	if err != nil {
		t.Fatal(err)
	}
	userID, tenantID, sessionID := uuid.New(), uuid.New(), uuid.New()
	repo := &testRepo{passwordHash: passwordHash}
	service := NewService(testAuth{iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{User: iamdomain.User{ID: userID, Email: "owner@example.test"}, Tenant: iamdomain.Tenant{ID: tenantID}, Permissions: []string{"iam.mfa.manage_self", "iam.oauth.manage_self"}}, SessionID: sessionID}}, repo, testSealer{}, Config{MFAEnabled: true, OAuthEnabled: true, OAuthAdapter: "local-mock", AdminOrigin: "https://admin.example.test"})
	service.now = func() time.Time { return now }
	enrollment, err := service.EnrollTOTP(context.Background(), "token", identity.Principal{RequestID: "enroll"})
	if err != nil || enrollment.Secret == "" || enrollment.OTPAuthURI == "" {
		t.Fatalf("enrollment=%#v err=%v", enrollment, err)
	}
	code, err := totpAt(enrollment.Secret, now)
	if err != nil {
		t.Fatal(err)
	}
	codes, err := service.VerifyTOTP(context.Background(), "token", identity.Principal{RequestID: "verify"}, identity.VerifyTOTPInput{Code: code})
	if err != nil || len(codes.Codes) != 10 || repo.recoveryCount != 10 {
		t.Fatalf("codes=%#v count=%d err=%v", codes, repo.recoveryCount, err)
	}
	if _, err = service.RotateRecoveryCodes(context.Background(), "token", identity.Principal{RequestID: "rotate"}, identity.StepUpInput{Method: "password", Proof: "wrong password"}); !errors.Is(err, identity.ErrStepUpRequired) {
		t.Fatalf("wrong proof error=%v", err)
	}
	rotated, err := service.RotateRecoveryCodes(context.Background(), "token", identity.Principal{RequestID: "rotate"}, identity.StepUpInput{Method: "password", Proof: "identity security password 2026!"})
	if err != nil || len(rotated.Codes) != 10 {
		t.Fatalf("rotated=%#v err=%v", rotated, err)
	}
}

func TestOAuthStartUsesS256StateAndSingleUseCallback(t *testing.T) {
	userID := uuid.New()
	repo := &testRepo{}
	service := NewService(testAuth{iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{User: iamdomain.User{ID: userID, Email: "owner@example.test"}, Tenant: iamdomain.Tenant{ID: uuid.New()}, Permissions: []string{"iam.oauth.manage_self"}}, SessionID: uuid.New()}}, repo, testSealer{}, Config{OAuthEnabled: true, OAuthAdapter: "local-mock", AdminOrigin: "https://admin.example.test"})
	started, err := service.StartOAuth(context.Background(), "token", identity.Principal{RequestID: "start"}, "local")
	if err != nil || started.AuthorizationURL == "" {
		t.Fatalf("started=%#v err=%v", started, err)
	}
	verifier := string(repo.challenge.PKCEVerifierEncrypted)
	if repo.challenge.PKCEChallenge != pkceChallenge(verifier) || len(repo.challenge.StateHash) != 32 || len(repo.challenge.CodeHash) != 32 {
		t.Fatalf("challenge=%#v", repo.challenge)
	}
	parsed := started.AuthorizationURL
	code := valueFromURL(t, parsed, "code")
	state := valueFromURL(t, parsed, "state")
	completion := identity.OAuthCompleteInput{Code: code, State: state}
	if _, err = service.CompleteOAuth(context.Background(), "token", identity.Principal{RequestID: "complete"}, "local", completion); err != nil {
		t.Fatal(err)
	}
	if _, err = service.CompleteOAuth(context.Background(), "token", identity.Principal{RequestID: "replay"}, "local", completion); !errors.Is(err, identity.ErrOAuthState) {
		t.Fatalf("replay error=%v", err)
	}
}

func TestFeatureFlagsFailClosedForEveryIdentitySecurityFlow(t *testing.T) {
	authenticated := iamdomain.AuthenticatedContext{
		AuthContext: iamdomain.AuthContext{
			User:        iamdomain.User{ID: uuid.New(), Email: "owner@example.test"},
			Tenant:      iamdomain.Tenant{ID: uuid.New()},
			Permissions: []string{"iam.mfa.manage_self", "iam.oauth.manage_self"},
		},
		SessionID: uuid.New(),
	}
	service := NewService(testAuth{authenticated}, &testRepo{}, testSealer{}, Config{})
	ctx := context.Background()
	principal := identity.Principal{RequestID: "feature-disabled"}
	if _, err := service.VerifyTOTP(ctx, "token", principal, identity.VerifyTOTPInput{Code: "123456"}); !errors.Is(err, identity.ErrFeatureDisabled) {
		t.Fatalf("VerifyTOTP error=%v", err)
	}
	if err := service.DisableTOTP(ctx, "token", principal, identity.StepUpInput{Method: "password", Proof: "proof"}); !errors.Is(err, identity.ErrFeatureDisabled) {
		t.Fatalf("DisableTOTP error=%v", err)
	}
	if _, err := service.RotateRecoveryCodes(ctx, "token", principal, identity.StepUpInput{Method: "password", Proof: "proof"}); !errors.Is(err, identity.ErrFeatureDisabled) {
		t.Fatalf("RotateRecoveryCodes error=%v", err)
	}
	if _, err := service.StartOAuth(ctx, "token", principal, "local"); !errors.Is(err, identity.ErrFeatureDisabled) {
		t.Fatalf("StartOAuth error=%v", err)
	}
}

func valueFromURL(t *testing.T, raw, key string) string {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Query().Get(key)
}
