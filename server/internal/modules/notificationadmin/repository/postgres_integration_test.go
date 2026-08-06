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
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func TestNotificationLifecycleTemplatesAndDeliveryRetryAreTenantScopedAndAudited(t *testing.T) {
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
	user, tenant, err := identity.CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "notify-" + suffix, TenantName: "Notification Integration", Email: "notify-" + suffix + "@example.test", DisplayName: "Notification Owner", Locale: "en-US", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if err != nil {
		t.Fatal(err)
	}
	var secondID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.users(email,display_name,locale,status) VALUES($1,'Recipient','zh-CN','active') RETURNING id`, "recipient-"+suffix+"@example.test").Scan(&secondID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO iam.tenant_members(tenant_id,user_id,status) VALUES($1,$2,'active')`, tenant.ID, secondID); err != nil {
		t.Fatal(err)
	}
	session, err := db.New(pool).CreateSession(ctx, db.CreateSessionParams{UserID: user.ID, TenantID: &tenant.ID, Audience: "ak-admin", AbsoluteExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}, IdleExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(30 * time.Minute), Valid: true}})
	if err != nil {
		t.Fatal(err)
	}
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{SkipUnknownJobCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	var _ *river.Client[pgx.Tx] = riverClient
	repo := NewPostgres(pool, riverClient)
	principal := notify.Principal{TenantID: tenant.ID, UserID: user.ID, SessionID: session.ID, RequestID: "notify-integration", IPAddress: "127.0.0.1", UserAgent: "integration"}
	message, err := repo.CreateMessage(ctx, principal, false, notify.MessageInput{MessageType: "system", Title: "Maintenance", Body: "A safe message", BodyFormat: "plain", AudienceScope: "selected", AudienceUserIDs: []uuid.UUID{secondID}})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := repo.PreviewRecipients(ctx, tenant.ID, message)
	if err != nil || preview.Count != 1 || len(preview.Items) != 1 || preview.Items[0].EmailHint == "" {
		t.Fatalf("preview=%#v err=%v", preview, err)
	}
	published, recipients, err := repo.PublishMessage(ctx, principal, message.ID, false)
	if err != nil || published.Status != "published" || recipients.Count != 1 {
		t.Fatalf("published=%#v recipients=%#v err=%v", published, recipients, err)
	}
	stats, err := repo.RecipientStats(ctx, tenant.ID, message.ID, false)
	if err != nil || stats.Total != 1 || stats.Pending != 1 {
		t.Fatalf("stats=%#v err=%v", stats, err)
	}
	if _, err = repo.GetMessage(ctx, uuid.New(), message.ID, false); !errors.Is(err, notify.ErrNotFound) {
		t.Fatalf("cross-tenant message error=%v", err)
	}
	locale := "en-US"
	subject := "Hello {{name}}"
	template, err := repo.CreateTemplate(ctx, principal, notify.TemplateInput{Code: "account.welcome", Name: "Welcome", Channel: "email", Locale: &locale, SubjectTemplate: &subject, BodyTemplate: "Welcome {{name}}", BodyFormat: "plain", VariablesSchema: []byte(`{"type":"object","properties":{"name":{"type":"string"}}}`), Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	templates, err := repo.ListTemplates(ctx, tenant.ID, notify.PageFilter{Page: 1, PageSize: 20, Locale: "en-US"})
	if err != nil || templates.Total < 1 || template.ID == uuid.Nil {
		t.Fatalf("templates=%#v err=%v", templates, err)
	}
	targetHash := sha256.Sum256([]byte("recipient@example.test"))
	var deliveryID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO notify.deliveries(tenant_id,message_id,user_id,template_id,channel,target_ciphertext,target_hash,target_hint,target_key_version,provider,status,attempt_count,max_attempts,last_error,retryable)
		VALUES($1,$2,$3,$4,'email',$5,$6,'r***@example.test',1,'local-mock','failed',1,3,$7,true) RETURNING id`, tenant.ID, message.ID, secondID, template.ID, []byte("encrypted"), targetHash[:], "temporary provider failure\nretryable").Scan(&deliveryID); err != nil {
		t.Fatal(err)
	}
	delivery, err := repo.GetDelivery(ctx, tenant.ID, deliveryID)
	if err != nil || delivery.ErrorCode != "PROVIDER_DELIVERY_FAILED" || delivery.ErrorSummary != "temporary provider failure retryable" || delivery.TargetHint != "r***@example.test" {
		t.Fatalf("delivery=%#v err=%v", delivery, err)
	}
	retried, err := repo.RetryDelivery(ctx, principal, deliveryID, false)
	if err != nil || retried.Status != "pending" || retried.ErrorSummary != "" {
		t.Fatalf("retried=%#v err=%v", retried, err)
	}
	if _, err = repo.RetryDelivery(ctx, principal, deliveryID, false); !errors.Is(err, notify.ErrRetryNotAllowed) {
		t.Fatalf("second retry error=%v", err)
	}
	var audits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND module_code='notify' AND action_name IN ('notify.message.create','notify.message.publish','notify.template.create','notify.delivery.retry')`, tenant.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 4 {
		t.Fatalf("expected four notification audits, got %d", audits)
	}
}
