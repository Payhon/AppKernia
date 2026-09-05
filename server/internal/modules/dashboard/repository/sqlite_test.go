package repository_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/dashboard/domain"
	"github.com/appkernia/appkernia/server/internal/modules/dashboard/repository"
	"github.com/google/uuid"
	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestSQLiteDashboard(t *testing.T) {
	db := openSQLiteDashboard(t)
	tenantID := uuid.New()
	startAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	endAt := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	query := domain.Query{TenantID: tenantID, StartAt: startAt, EndAt: endAt}
	repo := repository.NewSQLite(db)

	metrics, err := repo.Summary(context.Background(), query, domain.SummaryAccess{Users: true, Sessions: true, FailedJobs: true, SecurityEvents: true, Messages: true})
	if err != nil {
		t.Fatalf("empty summary: %v", err)
	}
	wantMetrics := []domain.Metric{{Key: "users.total"}, {Key: "users.new"}, {Key: "sessions.active"}, {Key: "jobs.failed"}, {Key: "security.open"}, {Key: "messages.published"}}
	if !reflect.DeepEqual(metrics, wantMetrics) {
		t.Fatalf("empty summary = %#v, want %#v", metrics, wantMetrics)
	}

	series, err := repo.Trends(context.Background(), query, domain.TrendAccess{Logins: true, Users: true, FailedJobs: true, SecurityEvents: true})
	if err != nil {
		t.Fatalf("empty trends: %v", err)
	}
	if len(series) != 5 {
		t.Fatalf("empty trend series count = %d, want 5", len(series))
	}
	for _, item := range series {
		if len(item.Points) != 3 {
			t.Fatalf("%s points = %d, want 3", item.Key, len(item.Points))
		}
		for _, point := range item.Points {
			if point.Value != 0 {
				t.Fatalf("%s value = %d, want 0", item.Key, point.Value)
			}
		}
	}

	activity, err := repo.Activity(context.Background(), query, domain.ActivityAccess{Operations: true, FailedJobs: true, SecurityEvents: true})
	if err != nil {
		t.Fatalf("empty activity: %v", err)
	}
	if activity.Operations == nil || activity.FailedJobs == nil || activity.SecurityEvents == nil {
		t.Fatal("empty activity slices must be non-nil")
	}
	if len(activity.Operations)+len(activity.FailedJobs)+len(activity.SecurityEvents) != 0 {
		t.Fatalf("empty activity = %#v", activity)
	}

	seedSQLiteDashboard(t, db, tenantID, startAt, endAt)
	metrics, err = repo.Summary(context.Background(), query, domain.SummaryAccess{Users: true, Sessions: true, FailedJobs: true, SecurityEvents: true, Messages: true})
	if err != nil {
		t.Fatalf("populated summary: %v", err)
	}
	wantMetrics = []domain.Metric{{Key: "users.total", Value: 2}, {Key: "users.new", Value: 1}, {Key: "sessions.active", Value: 1}, {Key: "jobs.failed", Value: 1}, {Key: "security.open", Value: 1}, {Key: "messages.published", Value: 1}}
	if !reflect.DeepEqual(metrics, wantMetrics) {
		t.Fatalf("populated summary = %#v, want %#v", metrics, wantMetrics)
	}

	series, err = repo.Trends(context.Background(), query, domain.TrendAccess{Logins: true, Users: true, FailedJobs: true, SecurityEvents: true})
	if err != nil {
		t.Fatalf("populated trends: %v", err)
	}
	trendValues := map[string][]int64{}
	for _, item := range series {
		for _, point := range item.Points {
			trendValues[item.Key] = append(trendValues[item.Key], point.Value)
		}
	}
	wantTrends := map[string][]int64{
		"logins.success":  {1, 0, 0},
		"logins.failure":  {0, 2, 0},
		"users.new":       {0, 1, 0},
		"jobs.failed":     {0, 1, 0},
		"security.events": {0, 1, 1},
	}
	if !reflect.DeepEqual(trendValues, wantTrends) {
		t.Fatalf("trends = %#v, want %#v", trendValues, wantTrends)
	}

	activity, err = repo.Activity(context.Background(), query, domain.ActivityAccess{Operations: true, FailedJobs: true, SecurityEvents: true})
	if err != nil {
		t.Fatalf("populated activity: %v", err)
	}
	if len(activity.Operations) != 2 || len(activity.FailedJobs) != 1 || len(activity.SecurityEvents) != 2 {
		t.Fatalf("activity lengths = operations:%d jobs:%d security:%d", len(activity.Operations), len(activity.FailedJobs), len(activity.SecurityEvents))
	}
	if activity.Operations[0].ActionName != "update" || !activity.Operations[0].Succeeded || activity.Operations[1].ErrorCode != "DASHBOARD.TEST" {
		t.Fatalf("operations = %#v", activity.Operations)
	}
	if activity.FailedJobs[0].ScheduleCode != "daily" || activity.FailedJobs[0].ErrorCode != "JOB.TEST" {
		t.Fatalf("failed jobs = %#v", activity.FailedJobs)
	}
	if activity.SecurityEvents[0].EventType != "session.reused" || activity.SecurityEvents[1].EventType != "login.failed" {
		t.Fatalf("security events = %#v", activity.SecurityEvents)
	}
}

func openSQLiteDashboard(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE iam_users (id TEXT PRIMARY KEY, created_at TEXT NOT NULL, deleted_at TEXT)`,
		`CREATE TABLE iam_tenant_members (tenant_id TEXT NOT NULL, user_id TEXT NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE iam_sessions (tenant_id TEXT NOT NULL, status TEXT NOT NULL, revoked_at TEXT, absolute_expires_at TEXT NOT NULL)`,
		`CREATE TABLE audit_login_events (tenant_id TEXT, audience TEXT NOT NULL, succeeded INTEGER NOT NULL, occurred_at TEXT NOT NULL)`,
		`CREATE TABLE jobs_schedules (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, code TEXT NOT NULL, name TEXT NOT NULL)`,
		`CREATE TABLE jobs_schedule_runs (id TEXT PRIMARY KEY, schedule_id TEXT NOT NULL, status TEXT NOT NULL, scheduled_at TEXT NOT NULL, error_code TEXT)`,
		`CREATE TABLE audit_security_events (id TEXT PRIMARY KEY, tenant_id TEXT, event_type TEXT NOT NULL, severity TEXT NOT NULL, source TEXT NOT NULL, resolved_at TEXT, occurred_at TEXT NOT NULL)`,
		`CREATE TABLE notify_messages (tenant_id TEXT NOT NULL, status TEXT NOT NULL, published_at TEXT)`,
		`CREATE TABLE audit_operation_logs (id TEXT PRIMARY KEY, tenant_id TEXT, module_code TEXT NOT NULL, action_name TEXT NOT NULL, resource_type TEXT, succeeded INTEGER NOT NULL, error_code TEXT, occurred_at TEXT NOT NULL)`,
	} {
		execSQLiteDashboard(t, db, statement)
	}
	return db
}

func seedSQLiteDashboard(t *testing.T, db *sql.DB, tenantID uuid.UUID, startAt, endAt time.Time) {
	t.Helper()
	otherTenantID := uuid.New()
	userBefore, userNew, userDeleted, otherUser := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, row := range []struct {
		id        uuid.UUID
		createdAt time.Time
		deletedAt any
		tenantID  uuid.UUID
		status    string
	}{
		{userBefore, startAt.AddDate(0, -1, 0), nil, tenantID, "active"},
		{userNew, startAt.Add(18 * time.Hour), nil, tenantID, "active"},
		{userDeleted, startAt.Add(20 * time.Hour), dashboardStamp(endAt), tenantID, "active"},
		{otherUser, startAt.Add(20 * time.Hour), nil, otherTenantID, "active"},
	} {
		execSQLiteDashboard(t, db, `INSERT INTO iam_users(id,created_at,deleted_at) VALUES(?,?,?)`, row.id.String(), dashboardStamp(row.createdAt), row.deletedAt)
		execSQLiteDashboard(t, db, `INSERT INTO iam_tenant_members(tenant_id,user_id,status) VALUES(?,?,?)`, row.tenantID.String(), row.id.String(), row.status)
	}
	execSQLiteDashboard(t, db, `INSERT INTO iam_sessions(tenant_id,status,revoked_at,absolute_expires_at) VALUES(?,?,?,?)`, tenantID.String(), "active", nil, dashboardStamp(endAt.Add(time.Hour)))
	execSQLiteDashboard(t, db, `INSERT INTO iam_sessions(tenant_id,status,revoked_at,absolute_expires_at) VALUES(?,?,?,?)`, tenantID.String(), "active", dashboardStamp(startAt), dashboardStamp(endAt.Add(time.Hour)))
	execSQLiteDashboard(t, db, `INSERT INTO iam_sessions(tenant_id,status,revoked_at,absolute_expires_at) VALUES(?,?,?,?)`, otherTenantID.String(), "active", nil, dashboardStamp(endAt.Add(time.Hour)))

	scheduleID, otherScheduleID := uuid.New(), uuid.New()
	execSQLiteDashboard(t, db, `INSERT INTO jobs_schedules(id,tenant_id,code,name) VALUES(?,?,?,?)`, scheduleID.String(), tenantID.String(), "daily", "Daily job")
	execSQLiteDashboard(t, db, `INSERT INTO jobs_schedules(id,tenant_id,code,name) VALUES(?,?,?,?)`, otherScheduleID.String(), otherTenantID.String(), "other", "Other job")
	execSQLiteDashboard(t, db, `INSERT INTO jobs_schedule_runs(id,schedule_id,status,scheduled_at,error_code) VALUES(?,?,?,?,?)`, uuid.NewString(), scheduleID.String(), "failed", dashboardStamp(startAt.Add(24*time.Hour)), "JOB.TEST")
	execSQLiteDashboard(t, db, `INSERT INTO jobs_schedule_runs(id,schedule_id,status,scheduled_at,error_code) VALUES(?,?,?,?,?)`, uuid.NewString(), scheduleID.String(), "succeeded", dashboardStamp(startAt.Add(24*time.Hour)), nil)
	execSQLiteDashboard(t, db, `INSERT INTO jobs_schedule_runs(id,schedule_id,status,scheduled_at,error_code) VALUES(?,?,?,?,?)`, uuid.NewString(), otherScheduleID.String(), "failed", dashboardStamp(startAt.Add(24*time.Hour)), "OTHER")

	execSQLiteDashboard(t, db, `INSERT INTO notify_messages(tenant_id,status,published_at) VALUES(?,?,?)`, tenantID.String(), "published", dashboardStamp(startAt.Add(24*time.Hour)))
	execSQLiteDashboard(t, db, `INSERT INTO notify_messages(tenant_id,status,published_at) VALUES(?,?,?)`, tenantID.String(), "draft", dashboardStamp(startAt.Add(24*time.Hour)))
	execSQLiteDashboard(t, db, `INSERT INTO notify_messages(tenant_id,status,published_at) VALUES(?,?,?)`, otherTenantID.String(), "published", dashboardStamp(startAt.Add(24*time.Hour)))

	dayOneEarly := time.Date(startAt.Year(), startAt.Month(), startAt.Day(), 1, 0, 0, 0, time.UTC)
	for _, row := range []struct {
		tenant    uuid.UUID
		audience  string
		succeeded int
		at        time.Time
	}{
		{tenantID, "ak-admin", 1, dayOneEarly},
		{tenantID, "ak-admin", 0, startAt.Add(24 * time.Hour)},
		{tenantID, "ak-admin", 0, startAt.Add(25 * time.Hour)},
		{tenantID, "ak-mobile", 0, startAt.Add(24 * time.Hour)},
		{otherTenantID, "ak-admin", 0, startAt.Add(24 * time.Hour)},
	} {
		execSQLiteDashboard(t, db, `INSERT INTO audit_login_events(tenant_id,audience,succeeded,occurred_at) VALUES(?,?,?,?)`, row.tenant.String(), row.audience, row.succeeded, dashboardStamp(row.at))
	}

	securityOne, securityTwo := uuid.New(), uuid.New()
	execSQLiteDashboard(t, db, `INSERT INTO audit_security_events(id,tenant_id,event_type,severity,source,resolved_at,occurred_at) VALUES(?,?,?,?,?,?,?)`, securityOne.String(), tenantID.String(), "login.failed", "warning", "iam", nil, dashboardStamp(startAt.Add(24*time.Hour)))
	execSQLiteDashboard(t, db, `INSERT INTO audit_security_events(id,tenant_id,event_type,severity,source,resolved_at,occurred_at) VALUES(?,?,?,?,?,?,?)`, securityTwo.String(), tenantID.String(), "session.reused", "critical", "iam", dashboardStamp(endAt), dashboardStamp(endAt))
	execSQLiteDashboard(t, db, `INSERT INTO audit_security_events(id,tenant_id,event_type,severity,source,resolved_at,occurred_at) VALUES(?,?,?,?,?,?,?)`, uuid.NewString(), otherTenantID.String(), "other", "warning", "iam", nil, dashboardStamp(startAt.Add(24*time.Hour)))

	execSQLiteDashboard(t, db, `INSERT INTO audit_operation_logs(id,tenant_id,module_code,action_name,resource_type,succeeded,error_code,occurred_at) VALUES(?,?,?,?,?,?,?,?)`, uuid.NewString(), tenantID.String(), "iam", "create", nil, 0, "DASHBOARD.TEST", dashboardStamp(startAt.Add(time.Hour)))
	execSQLiteDashboard(t, db, `INSERT INTO audit_operation_logs(id,tenant_id,module_code,action_name,resource_type,succeeded,error_code,occurred_at) VALUES(?,?,?,?,?,?,?,?)`, uuid.NewString(), tenantID.String(), "iam", "update", "user", 1, nil, dashboardStamp(endAt))
	execSQLiteDashboard(t, db, `INSERT INTO audit_operation_logs(id,tenant_id,module_code,action_name,resource_type,succeeded,error_code,occurred_at) VALUES(?,?,?,?,?,?,?,?)`, uuid.NewString(), tenantID.String(), "iam", "outside", nil, 1, nil, dashboardStamp(startAt.Add(-time.Second)))
}

func execSQLiteDashboard(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec sqlite statement: %v", err)
	}
}

func dashboardStamp(value time.Time) string {
	return value.UTC().Format("2006-01-02T15:04:05.000000000Z")
}
