//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	dashboarddomain "github.com/appkernia/appkernia/server/internal/modules/dashboard/domain"
	dashboardrepo "github.com/appkernia/appkernia/server/internal/modules/dashboard/repository"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDashboardRepositoryReturnsTenantScopedRealAggregates(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer pool.Close()

	suffix := uuid.NewString()
	user, tenant, err := iamrepo.NewPostgres(pool).CreateIdentity(ctx, domain.CreateIdentity{
		TenantCode: "dashboard-" + suffix, TenantName: "Dashboard Tenant",
		Email: "dashboard-" + suffix + "@example.test", DisplayName: "Dashboard User",
		Locale: "zh-CN", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$test$test",
	})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	now := time.Now().UTC()
	var scheduleID uuid.UUID
	if err = pool.QueryRow(ctx, `
		INSERT INTO jobs.schedules (tenant_id, code, name, handler_key, cron_expression)
		VALUES ($1, $2, 'Dashboard Test Schedule', 'dashboard.test', '0 * * * *') RETURNING id
	`, tenant.ID, "dashboard-"+suffix).Scan(&scheduleID); err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	statements := []struct {
		name string
		sql  string
		args []any
	}{
		{"session", `INSERT INTO iam.sessions (user_id, tenant_id, audience, absolute_expires_at) VALUES ($1,$2,'ak-admin',$3)`, []any{user.ID, tenant.ID, now.Add(time.Hour)}},
		{"failed job", `INSERT INTO jobs.schedule_runs (schedule_id,status,scheduled_at,error_code) VALUES ($1,'failed',$2,'DASHBOARD.TEST')`, []any{scheduleID, now}},
		{"security event", `INSERT INTO audit.security_events (tenant_id,user_id,event_type,severity,source) VALUES ($1,$2,'dashboard.test','high','integration')`, []any{tenant.ID, user.ID}},
		{"message", `INSERT INTO notify.messages (tenant_id,message_type,title,body,status,published_at) VALUES ($1,'notice','Dashboard test','Dashboard test body','published',$2)`, []any{tenant.ID, now}},
		{"operation", `INSERT INTO audit.operation_logs (tenant_id,user_id,request_id,module_code,action_name,succeeded) VALUES ($1,$2,$3,'dashboard','dashboard.test',true)`, []any{tenant.ID, user.ID, "dashboard-" + suffix}},
		{"login event", `INSERT INTO audit.login_events (tenant_id,user_id,auth_method,audience,result) VALUES ($1,$2,'password','ak-admin','success')`, []any{tenant.ID, user.ID}},
	}
	for _, statement := range statements {
		if _, err = pool.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatalf("insert %s fixture: %v", statement.name, err)
		}
	}

	repository := dashboardrepo.NewPostgres(pool)
	query := dashboarddomain.Query{TenantID: tenant.ID, StartAt: now.Add(-24 * time.Hour), EndAt: now.Add(time.Minute)}
	metrics, err := repository.Summary(ctx, query, dashboarddomain.SummaryAccess{Users: true, Sessions: true, FailedJobs: true, SecurityEvents: true, Messages: true})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	metricValues := make(map[string]int64, len(metrics))
	for _, metric := range metrics {
		metricValues[metric.Key] = metric.Value
	}
	for _, key := range []string{"users.total", "users.new", "sessions.active", "jobs.failed", "security.open", "messages.published"} {
		if metricValues[key] != 1 {
			t.Fatalf("expected %s=1, got metrics=%v", key, metricValues)
		}
	}
	series, err := repository.Trends(ctx, query, dashboarddomain.TrendAccess{Logins: true, Users: true, FailedJobs: true, SecurityEvents: true})
	if err != nil {
		t.Fatalf("trends: %v", err)
	}
	if len(series) != 5 {
		t.Fatalf("expected five permitted series, got %d", len(series))
	}
	expectedTotals := map[string]int64{
		"logins.success": 1, "logins.failure": 0, "users.new": 1, "jobs.failed": 1, "security.events": 1,
	}
	for _, item := range series {
		var total int64
		for _, point := range item.Points {
			total += point.Value
		}
		if total != expectedTotals[item.Key] {
			t.Fatalf("unexpected total in series %s, got %d (%s)", item.Key, total, fmt.Sprint(item.Points))
		}
	}
	activity, err := repository.Activity(ctx, query, dashboarddomain.ActivityAccess{Operations: true, FailedJobs: true, SecurityEvents: true})
	if err != nil {
		t.Fatalf("activity: %v", err)
	}
	if len(activity.Operations) != 1 || len(activity.FailedJobs) != 1 || len(activity.SecurityEvents) != 1 {
		t.Fatalf("unexpected activity counts: operations=%d jobs=%d security=%d", len(activity.Operations), len(activity.FailedJobs), len(activity.SecurityEvents))
	}
	if activity.FailedJobs[0].ErrorCode != "DASHBOARD.TEST" || activity.SecurityEvents[0].Severity != "high" {
		t.Fatalf("unexpected redacted activity: %#v %#v", activity.FailedJobs[0], activity.SecurityEvents[0])
	}
}
