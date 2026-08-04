//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	jobs "github.com/appkernia/appkernia/server/internal/modules/jobadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func TestScheduleLifecycleExecutionIsTenantScopedIdempotentAndAudited(t *testing.T) {
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
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{SkipUnknownJobCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	suffix := uuid.NewString()
	identity := iamrepo.NewPostgres(pool)
	user, tenant, err := identity.CreateIdentity(ctx, iamdomain.CreateIdentity{
		TenantCode:   "jobs-" + suffix,
		TenantName:   "Jobs Integration",
		Email:        "jobs-" + suffix + "@example.test",
		DisplayName:  "Jobs Owner",
		Locale:       "en-US",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := db.New(pool).CreateSession(ctx, db.CreateSessionParams{
		UserID: user.ID, TenantID: &tenant.ID, Audience: "ak-admin",
		AbsoluteExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		IdleExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(30 * time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(pool, riverClient)
	principal := jobs.Principal{
		TenantID: tenant.ID, UserID: user.ID, SessionID: session.ID,
		RequestID: "jobs-integration", IPAddress: "127.0.0.1", UserAgent: "integration",
	}
	now := time.Now().UTC().Truncate(time.Second)
	input := jobs.ScheduleInput{
		Code: "health." + suffix, Name: "Health snapshot", HandlerKey: "system.health.snapshot",
		CronExpression: "0 * * * *", TimeZone: "UTC", Payload: json.RawMessage(`{}`), QueueName: "default",
		OverlapPolicy: "skip", MisfirePolicy: "fire_once", TimeoutSeconds: 60, MaxAttempts: 3,
	}
	schedule, err := repo.CreateSchedule(ctx, principal, input, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if schedule.Status != "active" || schedule.ID == uuid.Nil {
		t.Fatalf("unexpected schedule: %#v", schedule)
	}
	if _, err = repo.GetSchedule(ctx, uuid.New(), schedule.ID); !errors.Is(err, jobs.ErrNotFound) {
		t.Fatalf("cross-tenant schedule error=%v", err)
	}
	paused, err := repo.ChangeStatus(ctx, principal, schedule.ID, "paused", nil)
	if err != nil || paused.Status != "paused" || paused.NextRunAt != nil {
		t.Fatalf("paused=%#v err=%v", paused, err)
	}
	resumedAt := now.Add(2 * time.Hour)
	resumed, err := repo.ChangeStatus(ctx, principal, schedule.ID, "active", &resumedAt)
	if err != nil || resumed.Status != "active" || resumed.NextRunAt == nil {
		t.Fatalf("resumed=%#v err=%v", resumed, err)
	}
	first, err := repo.Execute(ctx, principal, schedule.ID, "integration-idempotency-"+suffix, now)
	if err != nil || first.RiverJobID == nil || first.Status != "queued" {
		t.Fatalf("first run=%#v err=%v", first, err)
	}
	duplicate, err := repo.Execute(ctx, principal, schedule.ID, "integration-idempotency-"+suffix, now.Add(time.Second))
	if err != nil || duplicate.ID != first.ID || duplicate.RiverJobID == nil || *duplicate.RiverJobID != *first.RiverJobID {
		t.Fatalf("duplicate run=%#v err=%v", duplicate, err)
	}
	if _, err = repo.Execute(ctx, principal, schedule.ID, "integration-overlap-"+suffix, now.Add(2*time.Second)); !errors.Is(err, jobs.ErrConflict) {
		t.Fatalf("overlap error=%v", err)
	}
	runs, err := repo.ListRuns(ctx, tenant.ID, schedule.ID, jobs.PageFilter{Page: 1, PageSize: 20})
	if err != nil || runs.Total != 1 || len(runs.Items) != 1 || runs.Items[0].ID != first.ID {
		t.Fatalf("runs=%#v err=%v", runs, err)
	}
	if _, err = repo.ListRuns(ctx, uuid.New(), schedule.ID, jobs.PageFilter{Page: 1, PageSize: 20}); !errors.Is(err, jobs.ErrNotFound) {
		t.Fatalf("cross-tenant run history error=%v", err)
	}
	var audits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$1 AND module_code='jobs' AND action_name IN ('jobs.schedule.create','jobs.schedule.pause','jobs.schedule.resume','jobs.schedule.execute')`, tenant.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 4 {
		t.Fatalf("expected four schedule audits, got %d", audits)
	}
	var riverJobs int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE id=$1`, *first.RiverJobID).Scan(&riverJobs); err != nil {
		t.Fatal(err)
	}
	if riverJobs != 1 {
		t.Fatalf("expected one River job, got %d", riverJobs)
	}
}
