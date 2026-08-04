//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	identity "github.com/appkernia/appkernia/server/internal/modules/identitysecurity/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMFAAndOAuthLifecycleIsSelfScopedSingleUseAndAudited(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	suffix := uuid.NewString()
	passwordHash, err := iamapp.HashPassword("identity integration password 2026!")
	if err != nil {
		t.Fatal(err)
	}
	identityRepo := iamrepo.NewPostgres(pool)
	user, tenant, err := identityRepo.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "identity-" + suffix, TenantName: "Identity Integration", Email: "identity-" + suffix + "@example.test", DisplayName: "Identity Owner", Locale: "en-US", PasswordHash: passwordHash})
	if err != nil {
		t.Fatal(err)
	}
	session, err := db.New(pool).CreateSession(ctx, db.CreateSessionParams{UserID: user.ID, TenantID: &tenant.ID, Audience: "ak-admin", AbsoluteExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}, IdleExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(30 * time.Minute), Valid: true}})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(pool)
	principal := identity.Principal{TenantID: tenant.ID, UserID: user.ID, SessionID: session.ID, RequestID: "identity-integration", IPAddress: "127.0.0.1", UserAgent: "integration"}
	ciphertext := []byte("encrypted-totp-material")
	if err = repo.ReplacePendingTOTP(ctx, principal, ciphertext); err != nil {
		t.Fatal(err)
	}
	pending, err := repo.PendingTOTP(ctx, user.ID)
	if err != nil || string(pending.Ciphertext) != string(ciphertext) {
		t.Fatalf("pending=%#v err=%v", pending, err)
	}
	hashes := make([][]byte, 10)
	for index := range hashes {
		digest := sha256.Sum256([]byte(suffix + string(rune('a'+index))))
		hashes[index] = digest[:]
	}
	if err = repo.ActivateTOTP(ctx, principal, pending.ID, hashes); err != nil {
		t.Fatal(err)
	}
	status, err := repo.MFAStatus(ctx, user.ID)
	if err != nil || !status.TOTPEnabled || status.RecoveryCodesRemaining != 10 {
		t.Fatalf("status=%#v err=%v", status, err)
	}
	if stored, err := repo.PasswordHash(ctx, user.ID); err != nil || stored != passwordHash {
		t.Fatalf("password hash lookup mismatch err=%v", err)
	}
	stateHash := sha256.Sum256([]byte("state-" + suffix))
	codeHash := sha256.Sum256([]byte("code-" + suffix))
	challenge := identity.OAuthChallenge{Provider: "local", StateHash: stateHash[:], CodeHash: codeHash[:], PKCEVerifierEncrypted: []byte("encrypted-verifier"), PKCEChallenge: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ", ExpiresAt: time.Now().Add(5 * time.Minute)}
	if err = repo.SaveOAuthChallenge(ctx, principal, challenge); err != nil {
		t.Fatal(err)
	}
	account, err := repo.CompleteOAuth(ctx, principal, identity.OAuthChallenge{Provider: "local", StateHash: stateHash[:], CodeHash: codeHash[:], ExpiresAt: time.Now()}, identity.OAuthIdentity{Provider: "local", Subject: "local:" + suffix, AccountHint: "local-****"})
	if err != nil || account.Provider != "local" || account.AccountHint != "local-****" {
		t.Fatalf("account=%#v err=%v", account, err)
	}
	if _, err = repo.CompleteOAuth(ctx, principal, identity.OAuthChallenge{Provider: "local", StateHash: stateHash[:], CodeHash: codeHash[:], ExpiresAt: time.Now()}, identity.OAuthIdentity{Provider: "local", Subject: "local:" + suffix, AccountHint: "local-****"}); !errors.Is(err, identity.ErrOAuthState) {
		t.Fatalf("replay error=%v", err)
	}
	accounts, err := repo.ListOAuth(ctx, user.ID)
	if err != nil || len(accounts) != 1 || accounts[0].AccountHint != "local-****" {
		t.Fatalf("accounts=%#v err=%v", accounts, err)
	}
	if err = repo.DeleteOAuth(ctx, principal, "local"); err != nil {
		t.Fatal(err)
	}
	if err = repo.DisableTOTP(ctx, principal); err != nil {
		t.Fatal(err)
	}
	status, err = repo.MFAStatus(ctx, user.ID)
	if err != nil || status.TOTPEnabled || status.RecoveryCodesRemaining != 0 {
		t.Fatalf("disabled status=%#v err=%v", status, err)
	}
	var audits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND action_name IN ('iam.mfa.totp.enroll','iam.mfa.totp.enable','iam.mfa.totp.disable','iam.oauth.bind.start','iam.oauth.bind.complete','iam.oauth.unbind')`, tenant.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 6 {
		t.Fatalf("audits=%d", audits)
	}
}
