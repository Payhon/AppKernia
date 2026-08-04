//go:build integration

package repository

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	settingsrepo "github.com/appkernia/appkernia/server/internal/modules/systemsettings/repository"
	webhooks "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWebhookLifecycleEncryptsSecretScopesTenantAndDeduplicatesEvent(t *testing.T) {
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
	user, tenant, err := identity.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "hooks-" + suffix, TenantName: "Webhook Integration", Email: "hooks-" + suffix + "@example.test", DisplayName: "Webhook Owner", Locale: "zh-CN", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := db.New(pool).CreateSession(ctx, db.CreateSessionParams{UserID: user.ID, TenantID: &tenant.ID, Audience: "ak-admin", AbsoluteExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}, IdleExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(30 * time.Minute), Valid: true}})
	if err != nil {
		t.Fatal(err)
	}
	sealer, err := settingsrepo.NewAESGCMSealer(bytes.Repeat([]byte{0x51}, 32), 4)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("whsec_integration_secret")
	cipher, version, err := sealer.Seal(plain, tenant.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(pool)
	p := webhooks.Principal{TenantID: tenant.ID, UserID: user.ID, SessionID: session.ID, RequestID: "webhook-integration", UserAgent: "integration"}
	endpoint, err := repo.Create(ctx, p, webhooks.Input{Name: "Integration", EndpointURL: "https://hooks.example.test/events", EventTypes: []string{"order.created"}, MaxAttempts: 8, TimeoutSeconds: 10, Status: "active"}, cipher, version)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetStored(ctx, tenant.ID, endpoint.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stored.SecretCiphertext, plain) || stored.SecretKeyVersion != 4 {
		t.Fatal("plaintext or key version contract violated")
	}
	opened, err := sealer.Open(stored.SecretCiphertext, tenant.ID.String())
	if err != nil || !bytes.Equal(opened, plain) {
		t.Fatalf("open=%q err=%v", opened, err)
	}
	if _, err = repo.GetStored(ctx, uuid.New(), endpoint.ID); !errors.Is(err, webhooks.ErrNotFound) {
		t.Fatalf("cross tenant err=%v", err)
	}
	eventID := uuid.NewSHA1(endpoint.ID, []byte("same-idempotency-key"))
	first, created, err := repo.CreateTestDelivery(ctx, p, endpoint.ID, "same-idempotency-key", eventID, "order.created", map[string]any{"number": 1})
	if err != nil || !created {
		t.Fatalf("first=%#v created=%v err=%v", first, created, err)
	}
	duplicate, created, err := repo.CreateTestDelivery(ctx, p, endpoint.ID, "same-idempotency-key", eventID, "order.created", map[string]any{"number": 2})
	if err != nil || created || duplicate.ID != first.ID {
		t.Fatalf("duplicate=%#v created=%v err=%v", duplicate, created, err)
	}
	completed, err := repo.CompleteDelivery(ctx, p, endpoint.ID, first.ID, webhooks.DeliveryResult{StatusCode: 204}, nil)
	if err != nil || completed.Status != "succeeded" || completed.AttemptCount != 1 {
		t.Fatalf("completed=%#v err=%v", completed, err)
	}
	page, err := repo.Deliveries(ctx, tenant.ID, endpoint.ID, 1, 20)
	if err != nil || page.Total != 1 {
		t.Fatalf("page=%#v err=%v", page, err)
	}
	if _, err = repo.Deliveries(ctx, uuid.New(), endpoint.ID, 1, 20); !errors.Is(err, webhooks.ErrNotFound) {
		t.Fatalf("cross tenant deliveries err=%v", err)
	}
	var audits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND action_name IN ('sys.webhook.create','sys.webhook.test')`, tenant.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 2 {
		t.Fatalf("audits=%d", audits)
	}
}
