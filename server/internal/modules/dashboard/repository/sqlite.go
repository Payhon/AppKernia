package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/dashboard/domain"
	"github.com/google/uuid"
)

const sqliteDashboardTimeLayout = "2006-01-02T15:04:05.000000000Z"

type SQLite struct {
	db *sql.DB
}

var _ domain.Repository = (*SQLite)(nil)

func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db}
}

func (repository *SQLite) Summary(ctx context.Context, query domain.Query, access domain.SummaryAccess) ([]domain.Metric, error) {
	startAt, endAt := sqliteDashboardTime(query.StartAt), sqliteDashboardTime(query.EndAt)
	result := make([]domain.Metric, 0, 6)
	if access.Users {
		var total, recent int64
		err := repository.db.QueryRowContext(ctx, `
			SELECT count(*), coalesce(sum(CASE WHEN u.created_at >= ? THEN 1 ELSE 0 END), 0)
			FROM iam_tenant_members tm
			JOIN iam_users u ON u.id = tm.user_id AND u.deleted_at IS NULL
			WHERE tm.tenant_id = ? AND tm.status = 'active'`, startAt, query.TenantID.String()).Scan(&total, &recent)
		if err != nil {
			return nil, fmt.Errorf("dashboard user summary: %w", err)
		}
		result = append(result, domain.Metric{Key: "users.total", Value: total}, domain.Metric{Key: "users.new", Value: recent})
	}
	if access.Sessions {
		value, err := repository.count(ctx, `
			SELECT count(*) FROM iam_sessions
			WHERE tenant_id = ? AND status = 'active' AND revoked_at IS NULL AND absolute_expires_at > ?`, query.TenantID.String(), endAt)
		if err != nil {
			return nil, fmt.Errorf("dashboard active sessions: %w", err)
		}
		result = append(result, domain.Metric{Key: "sessions.active", Value: value})
	}
	if access.FailedJobs {
		value, err := repository.count(ctx, `
			SELECT count(*) FROM jobs_schedule_runs run
			JOIN jobs_schedules schedule ON schedule.id = run.schedule_id
			WHERE schedule.tenant_id = ? AND run.status = 'failed' AND run.scheduled_at >= ? AND run.scheduled_at <= ?`, query.TenantID.String(), startAt, endAt)
		if err != nil {
			return nil, fmt.Errorf("dashboard failed jobs: %w", err)
		}
		result = append(result, domain.Metric{Key: "jobs.failed", Value: value})
	}
	if access.SecurityEvents {
		value, err := repository.count(ctx, `SELECT count(*) FROM audit_security_events WHERE tenant_id = ? AND resolved_at IS NULL`, query.TenantID.String())
		if err != nil {
			return nil, fmt.Errorf("dashboard open security events: %w", err)
		}
		result = append(result, domain.Metric{Key: "security.open", Value: value})
	}
	if access.Messages {
		value, err := repository.count(ctx, `
			SELECT count(*) FROM notify_messages
			WHERE tenant_id = ? AND status = 'published' AND published_at >= ? AND published_at <= ?`, query.TenantID.String(), startAt, endAt)
		if err != nil {
			return nil, fmt.Errorf("dashboard published messages: %w", err)
		}
		result = append(result, domain.Metric{Key: "messages.published", Value: value})
	}
	return result, nil
}

func (repository *SQLite) Trends(ctx context.Context, query domain.Query, access domain.TrendAccess) ([]domain.TrendSeries, error) {
	days := sqliteDashboardDays(query.StartAt, query.EndAt)
	result := make([]domain.TrendSeries, 0, 5)
	if len(days) == 0 {
		if access.Logins {
			result = append(result, domain.TrendSeries{Key: "logins.success", Points: []domain.TrendPoint{}}, domain.TrendSeries{Key: "logins.failure", Points: []domain.TrendPoint{}})
		}
		if access.Users {
			result = append(result, domain.TrendSeries{Key: "users.new", Points: []domain.TrendPoint{}})
		}
		if access.FailedJobs {
			result = append(result, domain.TrendSeries{Key: "jobs.failed", Points: []domain.TrendPoint{}})
		}
		if access.SecurityEvents {
			result = append(result, domain.TrendSeries{Key: "security.events", Points: []domain.TrendPoint{}})
		}
		return result, nil
	}
	startAt, endAt := sqliteDashboardTime(days[0]), sqliteDashboardTime(days[len(days)-1].AddDate(0, 0, 1))
	if access.Logins {
		success, failure, err := repository.loginTrend(ctx, query.TenantID, startAt, endAt, days)
		if err != nil {
			return nil, fmt.Errorf("dashboard login trend: %w", err)
		}
		result = append(result, domain.TrendSeries{Key: "logins.success", Points: success}, domain.TrendSeries{Key: "logins.failure", Points: failure})
	}
	if access.Users {
		points, err := repository.trend(ctx, `
			SELECT substr(u.created_at, 1, 10), count(DISTINCT u.id)
			FROM iam_tenant_members tm
			JOIN iam_users u ON u.id = tm.user_id AND u.deleted_at IS NULL
			WHERE tm.tenant_id = ? AND tm.status = 'active' AND u.created_at >= ? AND u.created_at < ?
			GROUP BY substr(u.created_at, 1, 10)`, days, query.TenantID.String(), startAt, endAt)
		if err != nil {
			return nil, fmt.Errorf("dashboard user trend: %w", err)
		}
		result = append(result, domain.TrendSeries{Key: "users.new", Points: points})
	}
	if access.FailedJobs {
		points, err := repository.trend(ctx, `
			SELECT substr(run.scheduled_at, 1, 10), count(*)
			FROM jobs_schedule_runs run
			JOIN jobs_schedules schedule ON schedule.id = run.schedule_id
			WHERE schedule.tenant_id = ? AND run.status = 'failed' AND run.scheduled_at >= ? AND run.scheduled_at < ?
			GROUP BY substr(run.scheduled_at, 1, 10)`, days, query.TenantID.String(), startAt, endAt)
		if err != nil {
			return nil, fmt.Errorf("dashboard failed job trend: %w", err)
		}
		result = append(result, domain.TrendSeries{Key: "jobs.failed", Points: points})
	}
	if access.SecurityEvents {
		points, err := repository.trend(ctx, `
			SELECT substr(occurred_at, 1, 10), count(*)
			FROM audit_security_events
			WHERE tenant_id = ? AND occurred_at >= ? AND occurred_at < ?
			GROUP BY substr(occurred_at, 1, 10)`, days, query.TenantID.String(), startAt, endAt)
		if err != nil {
			return nil, fmt.Errorf("dashboard security trend: %w", err)
		}
		result = append(result, domain.TrendSeries{Key: "security.events", Points: points})
	}
	return result, nil
}

func (repository *SQLite) Activity(ctx context.Context, query domain.Query, access domain.ActivityAccess) (domain.Activity, error) {
	startAt, endAt := sqliteDashboardTime(query.StartAt), sqliteDashboardTime(query.EndAt)
	result := domain.Activity{Operations: []domain.OperationActivity{}, FailedJobs: []domain.FailedJobActivity{}, SecurityEvents: []domain.SecurityActivity{}}
	if access.Operations {
		rows, err := repository.db.QueryContext(ctx, `
			SELECT id, module_code, action_name, resource_type, succeeded, error_code, occurred_at
			FROM audit_operation_logs
			WHERE tenant_id = ? AND occurred_at >= ? AND occurred_at <= ?
			ORDER BY occurred_at DESC, id DESC LIMIT 10`, query.TenantID.String(), startAt, endAt)
		if err != nil {
			return domain.Activity{}, fmt.Errorf("dashboard recent operations: %w", err)
		}
		for rows.Next() {
			var id, occurredAt string
			var moduleCode, actionName string
			var resourceType, errorCode sql.NullString
			var succeeded bool
			if err = rows.Scan(&id, &moduleCode, &actionName, &resourceType, &succeeded, &errorCode, &occurredAt); err != nil {
				rows.Close()
				return domain.Activity{}, fmt.Errorf("dashboard recent operations: %w", err)
			}
			parsedID, parsedAt, parseErr := sqliteDashboardValues(id, occurredAt)
			if parseErr != nil {
				rows.Close()
				return domain.Activity{}, fmt.Errorf("dashboard recent operations: %w", parseErr)
			}
			result.Operations = append(result.Operations, domain.OperationActivity{ID: parsedID, ModuleCode: moduleCode, ActionName: actionName, ResourceType: resourceType.String, Succeeded: succeeded, ErrorCode: errorCode.String, OccurredAt: parsedAt})
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return domain.Activity{}, fmt.Errorf("dashboard recent operations: %w", err)
		}
		rows.Close()
	}
	if access.FailedJobs {
		rows, err := repository.db.QueryContext(ctx, `
			SELECT run.id, schedule.code, schedule.name, run.error_code, run.scheduled_at
			FROM jobs_schedule_runs run
			JOIN jobs_schedules schedule ON schedule.id = run.schedule_id
			WHERE schedule.tenant_id = ? AND run.status = 'failed' AND run.scheduled_at >= ? AND run.scheduled_at <= ?
			ORDER BY run.scheduled_at DESC, run.id DESC LIMIT 10`, query.TenantID.String(), startAt, endAt)
		if err != nil {
			return domain.Activity{}, fmt.Errorf("dashboard recent failed jobs: %w", err)
		}
		for rows.Next() {
			var id, scheduleCode, scheduleName, occurredAt string
			var errorCode sql.NullString
			if err = rows.Scan(&id, &scheduleCode, &scheduleName, &errorCode, &occurredAt); err != nil {
				rows.Close()
				return domain.Activity{}, fmt.Errorf("dashboard recent failed jobs: %w", err)
			}
			parsedID, parsedAt, parseErr := sqliteDashboardValues(id, occurredAt)
			if parseErr != nil {
				rows.Close()
				return domain.Activity{}, fmt.Errorf("dashboard recent failed jobs: %w", parseErr)
			}
			result.FailedJobs = append(result.FailedJobs, domain.FailedJobActivity{ID: parsedID, ScheduleCode: scheduleCode, ScheduleName: scheduleName, ErrorCode: errorCode.String, OccurredAt: parsedAt})
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return domain.Activity{}, fmt.Errorf("dashboard recent failed jobs: %w", err)
		}
		rows.Close()
	}
	if access.SecurityEvents {
		rows, err := repository.db.QueryContext(ctx, `
			SELECT id, event_type, severity, source, occurred_at
			FROM audit_security_events
			WHERE tenant_id = ? AND occurred_at >= ? AND occurred_at <= ?
			ORDER BY occurred_at DESC, id DESC LIMIT 10`, query.TenantID.String(), startAt, endAt)
		if err != nil {
			return domain.Activity{}, fmt.Errorf("dashboard recent security events: %w", err)
		}
		for rows.Next() {
			var id, eventType, severity, source, occurredAt string
			if err = rows.Scan(&id, &eventType, &severity, &source, &occurredAt); err != nil {
				rows.Close()
				return domain.Activity{}, fmt.Errorf("dashboard recent security events: %w", err)
			}
			parsedID, parsedAt, parseErr := sqliteDashboardValues(id, occurredAt)
			if parseErr != nil {
				rows.Close()
				return domain.Activity{}, fmt.Errorf("dashboard recent security events: %w", parseErr)
			}
			result.SecurityEvents = append(result.SecurityEvents, domain.SecurityActivity{ID: parsedID, EventType: eventType, Severity: severity, Source: source, OccurredAt: parsedAt})
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return domain.Activity{}, fmt.Errorf("dashboard recent security events: %w", err)
		}
		rows.Close()
	}
	return result, nil
}

func (repository *SQLite) count(ctx context.Context, query string, args ...any) (int64, error) {
	var value int64
	err := repository.db.QueryRowContext(ctx, query, args...).Scan(&value)
	return value, err
}

func (repository *SQLite) loginTrend(ctx context.Context, tenantID uuid.UUID, startAt, endAt string, days []time.Time) ([]domain.TrendPoint, []domain.TrendPoint, error) {
	rows, err := repository.db.QueryContext(ctx, `
		SELECT substr(occurred_at, 1, 10),
		       sum(CASE WHEN succeeded = 1 THEN 1 ELSE 0 END),
		       sum(CASE WHEN succeeded = 0 THEN 1 ELSE 0 END)
		FROM audit_login_events
		WHERE tenant_id = ? AND audience = 'ak-admin' AND occurred_at >= ? AND occurred_at < ?
		GROUP BY substr(occurred_at, 1, 10)`, tenantID.String(), startAt, endAt)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	values := make(map[string][2]int64, len(days))
	for rows.Next() {
		var day string
		var success, failure int64
		if err = rows.Scan(&day, &success, &failure); err != nil {
			return nil, nil, err
		}
		values[day] = [2]int64{success, failure}
	}
	if err = rows.Err(); err != nil {
		return nil, nil, err
	}
	success, failure := make([]domain.TrendPoint, 0, len(days)), make([]domain.TrendPoint, 0, len(days))
	for _, day := range days {
		value := values[day.Format(time.DateOnly)]
		success = append(success, domain.TrendPoint{Day: day, Value: value[0]})
		failure = append(failure, domain.TrendPoint{Day: day, Value: value[1]})
	}
	return success, failure, nil
}

func (repository *SQLite) trend(ctx context.Context, query string, days []time.Time, args ...any) ([]domain.TrendPoint, error) {
	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make(map[string]int64, len(days))
	for rows.Next() {
		var day string
		var value int64
		if err = rows.Scan(&day, &value); err != nil {
			return nil, err
		}
		values[day] = value
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	points := make([]domain.TrendPoint, 0, len(days))
	for _, day := range days {
		points = append(points, domain.TrendPoint{Day: day, Value: values[day.Format(time.DateOnly)]})
	}
	return points, nil
}

func sqliteDashboardDays(startAt, endAt time.Time) []time.Time {
	start := startAt.UTC()
	end := endAt.UTC()
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	if start.After(end) {
		return []time.Time{}
	}
	days := make([]time.Time, 0, int(end.Sub(start)/(24*time.Hour))+1)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		days = append(days, day)
	}
	return days
}

func sqliteDashboardTime(value time.Time) string {
	return value.UTC().Format(sqliteDashboardTimeLayout)
}

func sqliteDashboardValues(rawID, rawTime string) (uuid.UUID, time.Time, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, time.Time{}, fmt.Errorf("parse id: %w", err)
	}
	value, err := time.Parse(time.RFC3339Nano, rawTime)
	if err != nil {
		return uuid.Nil, time.Time{}, fmt.Errorf("parse time: %w", err)
	}
	return id, value.UTC(), nil
}
