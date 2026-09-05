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
	clientsapp "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/application"
	clients "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
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
	created, err := repo.Create(ctx, p, "ak_"+suffix, clients.Input{Name: "Integration", AllowedCIDRs: []string{"203.0.113.0/24"}, Status: "active", BoundUserID: &user.ID})
	if err != nil {
		t.Fatal(err)
	}
	if created.BoundUserID == nil || *created.BoundUserID != user.ID || created.BoundUser == nil || created.BoundUser.ID != user.ID || created.BoundUser.DisplayName != user.DisplayName {
		t.Fatalf("bound user projection=%#v", created)
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
	issuer, err := iamapp.NewDevelopmentTokenIssuer()
	if err != nil {
		t.Fatal(err)
	}
	tokenService := clientsapp.NewService(nil, repo, issuer)
	if token, _, tokenErr := tokenService.Token(ctx, created.ClientID, plaintext, clients.TokenMetadata{
		RequestID: "api-client-token-success", IPAddress: "203.0.113.10", UserAgent: "akone/integration",
	}); tokenErr != nil || token == "" {
		t.Fatalf("token=%q err=%v", token, tokenErr)
	}
	if _, _, tokenErr := tokenService.Token(ctx, created.ClientID, "aks_wrong_012345678901234567890123456789", clients.TokenMetadata{
		RequestID: "api-client-token-failure", IPAddress: "203.0.113.10", UserAgent: "akone/integration",
	}); !errors.Is(tokenErr, clients.ErrCredential) {
		t.Fatalf("invalid token exchange error=%v", tokenErr)
	}
	clientIdentifierHash := sha256.Sum256([]byte(created.ClientID))
	var successfulExchanges, failedExchanges, secretHashLeaks int
	if err = pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE request_id='api-client-token-success' AND result='success' AND failure_reason IS NULL),
		count(*) FILTER (WHERE request_id='api-client-token-failure' AND result='failure' AND failure_reason='invalid_credentials'),
		count(*) FILTER (WHERE login_identifier_hash=$3)
	FROM audit.login_events
	WHERE tenant_id=$1 AND auth_method='api_secret' AND audience='ak-api' AND user_id IS NULL
	AND request_id=ANY($2::text[]) AND login_identifier_hash IN ($3,$4)`, tenant.ID,
		[]string{"api-client-token-success", "api-client-token-failure"}, digest[:], clientIdentifierHash[:]).Scan(&successfulExchanges, &failedExchanges, &secretHashLeaks); err != nil {
		t.Fatal(err)
	}
	if successfulExchanges != 1 || failedExchanges != 1 || secretHashLeaks != 0 {
		t.Fatalf("token exchange audits success=%d failure=%d secret_hash_leaks=%d", successfulExchanges, failedExchanges, secretHashLeaks)
	}
	if _, err = repo.Authenticate(ctx, clients.Credential{ClientID: created.ClientID, SecretHash: digest[:], IPAddress: "10.0.0.1"}); !errors.Is(err, clients.ErrCredential) {
		t.Fatalf("CIDR error=%v", err)
	}
	if err = repo.AuditAgentAuthentication(ctx, clients.AgentAudit{TenantID: tenant.ID, UserID: user.ID, ClientID: created.ID, RequestID: "agent-read-integration", Operation: "getAdminDashboardSummary", Method: "GET", Path: "/admin-api/v1/dashboard/summary", IPAddress: "203.0.113.10", UserAgent: "akone/integration"}); err != nil {
		t.Fatal(err)
	}
	var delegatedAudits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE api_client_id=$1 AND user_id=$2 AND request_id='agent-read-integration' AND action_name='agent.delegation.authenticate' AND resource_id='getAdminDashboardSummary' AND response_status IS NULL AND succeeded`, created.ID, user.ID).Scan(&delegatedAudits); err != nil || delegatedAudits != 1 {
		t.Fatalf("delegated audits=%d err=%v", delegatedAudits, err)
	}
	otherUser, _, err := identity.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "clients-other-" + suffix, TenantName: "Other Client Integration", Email: "clients-other-" + suffix + "@example.test", DisplayName: "Other Owner", Locale: "en-US", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Update(ctx, p, created.ID, clients.Input{Name: "Integration", AllowedCIDRs: []string{"203.0.113.0/24"}, Status: "active", BoundUserID: &otherUser.ID}); !errors.Is(err, clients.ErrInvalid) {
		t.Fatalf("cross-tenant bound user error=%v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE sys.api_clients SET bound_user_id=$1 WHERE id=$2`, otherUser.ID, created.ID); err == nil {
		t.Fatal("database accepted a cross-tenant bound user")
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
