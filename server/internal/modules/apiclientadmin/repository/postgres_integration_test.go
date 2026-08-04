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
	clients "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/domain"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAPIClientLifecycleIsTenantScopedHashedAndAudited(t *testing.T) {
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
	identity := iamrepo.NewPostgres(pool)
	user, tenant, err := identity.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "clients-" + suffix, TenantName: "Client Integration", Email: "clients-" + suffix + "@example.test", DisplayName: "Client Owner", Locale: "en-US", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := db.New(pool).CreateSession(ctx, db.CreateSessionParams{UserID: user.ID, TenantID: &tenant.ID, Audience: "ak-admin", AbsoluteExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}, IdleExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(30 * time.Minute), Valid: true}})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(pool)
	p := clients.Principal{TenantID: tenant.ID, UserID: user.ID, SessionID: session.ID, RequestID: "api-client-integration", UserAgent: "integration"}
	created, err := repo.Create(ctx, p, "ak_"+suffix, clients.Input{Name: "Integration", AllowedCIDRs: []string{"203.0.113.0/24"}, Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	read, err := repo.Get(ctx, tenant.ID, created.ID)
	if err != nil || read.ID != created.ID || read.ClientID != created.ClientID {
		t.Fatalf("read=%#v err=%v", read, err)
	}
	if _, err = repo.Get(ctx, uuid.New(), created.ID); !errors.Is(err, clients.ErrNotFound) {
		t.Fatalf("cross-tenant get error=%v", err)
	}
	plaintext := "aks_integration_" + suffix
	digest := sha256.Sum256([]byte(plaintext))
	secret, err := repo.CreateSecret(ctx, p, created.ID, "aks_integration", digest[:], nil)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := repo.ReplacePermissions(ctx, p, created.ID, []string{"sys.webhook.read"})
	if err != nil || len(updated.Permissions) != 1 {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
	authed, err := repo.Authenticate(ctx, clients.Credential{ClientID: created.ClientID, SecretHash: digest[:], IPAddress: "203.0.113.10"})
	if err != nil || authed.ID != created.ID {
		t.Fatalf("authed=%#v err=%v", authed, err)
	}
	if _, err = repo.Authenticate(ctx, clients.Credential{ClientID: created.ClientID, SecretHash: digest[:], IPAddress: "10.0.0.1"}); !errors.Is(err, clients.ErrCredential) {
		t.Fatalf("CIDR error=%v", err)
	}
	other, err := repo.List(ctx, uuid.New(), clients.Filter{Page: 1, PageSize: 20})
	if err != nil || other.Total != 0 {
		t.Fatalf("cross tenant page=%#v err=%v", other, err)
	}
	if err = repo.RevokeSecret(ctx, p, created.ID, secret.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Authenticate(ctx, clients.Credential{ClientID: created.ClientID, SecretHash: digest[:], IPAddress: "203.0.113.10"}); !errors.Is(err, clients.ErrCredential) {
		t.Fatalf("revoked error=%v", err)
	}
	var audits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND action_name IN ('sys.api_client.create','sys.api_client.rotate_secret','sys.api_client.assign_permission','sys.api_client.revoke_secret')`, tenant.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 4 {
		t.Fatalf("audits=%d", audits)
	}
}
