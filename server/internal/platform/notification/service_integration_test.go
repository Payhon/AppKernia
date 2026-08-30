//go:build integration

package notification

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/appkernia/appkernia/server/internal/modules/notificationadmin/jobdefs"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func TestSubmissionIsTransactionalIdempotentScopedAndCancellable(t *testing.T) {
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
	user, tenant, err := identity.CreateIdentity(ctx, iamdomain.CreateIdentity{
		TenantCode: "notification-service-" + suffix, TenantName: "Notification Service Integration",
		Email: "notification-service-" + suffix + "@example.test", DisplayName: "Notification Caller",
		Locale: "en-US", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration",
	})
	if err != nil {
		t.Fatal(err)
	}
	var appID uuid.UUID
	if err = pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default=true`, tenant.ID).Scan(&appID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO app.user_memberships(app_id,tenant_id,user_id,source,status)
		VALUES($1,$2,$3,'admin_created','active')`, appID, tenant.ID, user.ID); err != nil {
		t.Fatal(err)
	}

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{SkipUnknownJobCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	queue := jobqueue.NewRiverAdapter(pool, riverClient, jobdefs.Registry())
	service := NewPostgresService(pool, queue)
	now := time.Now().UTC().Truncate(time.Second)
	service.clock = func() time.Time { return now }
	scope := Scope{
		TenantID: tenant.ID, AppID: appID, ActorKind: "user", ActorID: user.ID,
		RequestID: "notification-service-integration", SourceIP: "127.0.0.1", UserAgent: "integration",
	}
	scheduledAt := now.Add(10 * time.Minute)
	command := SubmitCommand{
		IdempotencyKey: "notification-service-" + suffix,
		Source:         "billing.security", BusinessEventID: "invoice:" + suffix, Category: "service_security",
		Audience: Audience{Type: "users", UserIDs: []uuid.UUID{user.ID}},
		Content: Content{Type: "inline", Inline: &LocalizedContent{
			Title: map[string]string{"zh-CN": "安全提醒", "en-US": "Security notice"},
			Body:  map[string]string{"zh-CN": "您的安全设置已更新。", "en-US": "Your security settings were updated."},
		}},
		Push: true, ScheduledAt: &scheduledAt, TTLSeconds: 3600, CollapseKey: "account-security",
		RouteKey: "notification_detail", ResourceID: "invoice-" + suffix,
	}

	submission, err := service.Submit(ctx, scope, command)
	if err != nil {
		t.Fatal(err)
	}
	if submission.Status != "scheduled" || submission.MessageID == uuid.Nil || submission.RunID == uuid.Nil {
		t.Fatalf("unexpected submission: %#v", submission)
	}
	status, err := service.Status(ctx, scope, submission.MessageID)
	if err != nil || status.RunID != submission.RunID || status.RecipientCount != 1 || status.Status != "scheduled" {
		t.Fatalf("status=%#v err=%v", status, err)
	}
	var taskRunID uuid.UUID
	var riverJobID int64
	var taskStatus, taskKind, queueName string
	var maxAttempts int
	if err = pool.QueryRow(ctx, `SELECT id,river_job_id,status,task_kind,queue_name,max_attempts FROM jobs.task_runs
		WHERE tenant_id=$1 AND app_id=$2 AND correlation_id=$3`, tenant.ID, appID, submission.RunID).
		Scan(&taskRunID, &riverJobID, &taskStatus, &taskKind, &queueName, &maxAttempts); err != nil {
		t.Fatal(err)
	}
	if taskStatus != "scheduled" || taskKind != "appkernia-message-publish" || queueName != jobdefs.Queue || maxAttempts != jobdefs.PublishMaxAttempts {
		t.Fatalf("unexpected tracked task: status=%s kind=%s queue=%s attempts=%d", taskStatus, taskKind, queueName, maxAttempts)
	}
	state, err := queue.GetState(ctx, tenant.ID, taskRunID)
	if err != nil || state.RiverJobID != riverJobID || state.Status != "scheduled" {
		t.Fatalf("state=%#v err=%v", state, err)
	}

	duplicate, err := service.Submit(ctx, scope, command)
	if err != nil || duplicate != submission {
		t.Fatalf("duplicate=%#v err=%v", duplicate, err)
	}
	conflicting := command
	conflicting.Content.Inline = &LocalizedContent{
		Title: map[string]string{"zh-CN": "不同标题", "en-US": "Different title"},
		Body:  map[string]string{"zh-CN": "不同正文。", "en-US": "Different body."},
	}
	if _, err = service.Submit(ctx, scope, conflicting); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("idempotency conflict error=%v", err)
	}
	crossApp := scope
	crossApp.AppID = uuid.New()
	if _, err = service.Status(ctx, crossApp, submission.MessageID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-app status error=%v", err)
	}
	crossTenant := scope
	crossTenant.TenantID = uuid.New()
	if _, err = service.Status(ctx, crossTenant, submission.MessageID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant status error=%v", err)
	}

	rollbackCommand := command
	rollbackCommand.IdempotencyKey = "notification-rollback-" + suffix
	rollbackCommand.BusinessEventID = "rollback:" + suffix
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	rolledBack, err := service.SubmitTx(ctx, tx, scope, rollbackCommand)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Status(ctx, scope, rolledBack.MessageID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("rolled-back submission remained visible: %v", err)
	}

	if err = service.Cancel(ctx, scope, submission.MessageID); err != nil {
		t.Fatal(err)
	}
	status, err = service.Status(ctx, scope, submission.MessageID)
	if err != nil || status.Status != "cancelled" || status.CompletedAt == nil {
		t.Fatalf("cancelled status=%#v err=%v", status, err)
	}
	state, err = queue.GetState(ctx, tenant.ID, taskRunID)
	if err != nil || state.Status != "cancelled" {
		t.Fatalf("cancelled task=%#v err=%v", state, err)
	}
}
