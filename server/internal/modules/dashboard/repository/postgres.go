package repository

import (
	"context"
	"fmt"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/dashboard/domain"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (repository *Postgres) Summary(ctx context.Context, query domain.Query, access domain.SummaryAccess) ([]domain.Metric, error) {
	queries := db.New(repository.pool)
	startAt, endAt := timestamp(query.StartAt), timestamp(query.EndAt)
	result := make([]domain.Metric, 0, 6)
	if access.Users {
		row, err := queries.DashboardUserSummary(ctx, db.DashboardUserSummaryParams{StartAt: startAt, TenantID: query.TenantID})
		if err != nil {
			return nil, fmt.Errorf("dashboard user summary: %w", err)
		}
		result = append(result, domain.Metric{Key: "users.total", Value: row.TotalUsers}, domain.Metric{Key: "users.new", Value: row.NewUsers})
	}
	if access.Sessions {
		value, err := queries.DashboardActiveSessionCount(ctx, db.DashboardActiveSessionCountParams{TenantID: &query.TenantID, EndAt: endAt})
		if err != nil {
			return nil, fmt.Errorf("dashboard active sessions: %w", err)
		}
		result = append(result, domain.Metric{Key: "sessions.active", Value: value})
	}
	if access.FailedJobs {
		value, err := queries.DashboardFailedJobCount(ctx, db.DashboardFailedJobCountParams{TenantID: &query.TenantID, StartAt: startAt, EndAt: endAt})
		if err != nil {
			return nil, fmt.Errorf("dashboard failed jobs: %w", err)
		}
		result = append(result, domain.Metric{Key: "jobs.failed", Value: value})
	}
	if access.SecurityEvents {
		value, err := queries.DashboardOpenSecurityEventCount(ctx, &query.TenantID)
		if err != nil {
			return nil, fmt.Errorf("dashboard open security events: %w", err)
		}
		result = append(result, domain.Metric{Key: "security.open", Value: value})
	}
	if access.Messages {
		value, err := queries.DashboardPublishedMessageCount(ctx, db.DashboardPublishedMessageCountParams{TenantID: query.TenantID, StartAt: startAt, EndAt: endAt})
		if err != nil {
			return nil, fmt.Errorf("dashboard published messages: %w", err)
		}
		result = append(result, domain.Metric{Key: "messages.published", Value: value})
	}
	return result, nil
}

func (repository *Postgres) Trends(ctx context.Context, query domain.Query, access domain.TrendAccess) ([]domain.TrendSeries, error) {
	queries := db.New(repository.pool)
	startAt, endAt := timestamp(query.StartAt), timestamp(query.EndAt)
	result := make([]domain.TrendSeries, 0, 5)
	if access.Logins {
		rows, err := queries.DashboardLoginTrend(ctx, db.DashboardLoginTrendParams{TenantID: &query.TenantID, StartAt: startAt, EndAt: endAt})
		if err != nil {
			return nil, fmt.Errorf("dashboard login trend: %w", err)
		}
		success, failure := make([]domain.TrendPoint, 0, len(rows)), make([]domain.TrendPoint, 0, len(rows))
		for _, row := range rows {
			success = append(success, domain.TrendPoint{Day: row.Day.Time.UTC(), Value: row.SuccessCount})
			failure = append(failure, domain.TrendPoint{Day: row.Day.Time.UTC(), Value: row.FailureCount})
		}
		result = append(result, domain.TrendSeries{Key: "logins.success", Points: success}, domain.TrendSeries{Key: "logins.failure", Points: failure})
	}
	if access.Users {
		rows, err := queries.DashboardUserTrend(ctx, db.DashboardUserTrendParams{TenantID: query.TenantID, StartAt: startAt, EndAt: endAt})
		if err != nil {
			return nil, fmt.Errorf("dashboard user trend: %w", err)
		}
		result = append(result, trendSeries("users.new", rowsToPoints(rows)))
	}
	if access.FailedJobs {
		rows, err := queries.DashboardFailedJobTrend(ctx, db.DashboardFailedJobTrendParams{TenantID: &query.TenantID, StartAt: startAt, EndAt: endAt})
		if err != nil {
			return nil, fmt.Errorf("dashboard failed job trend: %w", err)
		}
		points := make([]domain.TrendPoint, 0, len(rows))
		for _, row := range rows {
			points = append(points, domain.TrendPoint{Day: row.Day.Time.UTC(), Value: row.Value})
		}
		result = append(result, trendSeries("jobs.failed", points))
	}
	if access.SecurityEvents {
		rows, err := queries.DashboardSecurityTrend(ctx, db.DashboardSecurityTrendParams{TenantID: &query.TenantID, StartAt: startAt, EndAt: endAt})
		if err != nil {
			return nil, fmt.Errorf("dashboard security trend: %w", err)
		}
		points := make([]domain.TrendPoint, 0, len(rows))
		for _, row := range rows {
			points = append(points, domain.TrendPoint{Day: row.Day.Time.UTC(), Value: row.Value})
		}
		result = append(result, trendSeries("security.events", points))
	}
	return result, nil
}

func (repository *Postgres) Activity(ctx context.Context, query domain.Query, access domain.ActivityAccess) (domain.Activity, error) {
	queries := db.New(repository.pool)
	startAt, endAt := timestamp(query.StartAt), timestamp(query.EndAt)
	result := domain.Activity{Operations: []domain.OperationActivity{}, FailedJobs: []domain.FailedJobActivity{}, SecurityEvents: []domain.SecurityActivity{}}
	if access.Operations {
		rows, err := queries.DashboardRecentOperations(ctx, db.DashboardRecentOperationsParams{TenantID: &query.TenantID, StartAt: startAt, EndAt: endAt})
		if err != nil {
			return domain.Activity{}, fmt.Errorf("dashboard recent operations: %w", err)
		}
		for _, row := range rows {
			result.Operations = append(result.Operations, domain.OperationActivity{
				ID: row.ID, ModuleCode: row.ModuleCode, ActionName: row.ActionName,
				ResourceType: valueOrEmpty(row.ResourceType), Succeeded: row.Succeeded,
				ErrorCode: valueOrEmpty(row.ErrorCode), OccurredAt: row.OccurredAt.Time.UTC(),
			})
		}
	}
	if access.FailedJobs {
		rows, err := queries.DashboardRecentFailedJobs(ctx, db.DashboardRecentFailedJobsParams{TenantID: &query.TenantID, StartAt: startAt, EndAt: endAt})
		if err != nil {
			return domain.Activity{}, fmt.Errorf("dashboard recent failed jobs: %w", err)
		}
		for _, row := range rows {
			result.FailedJobs = append(result.FailedJobs, domain.FailedJobActivity{
				ID: row.ID, ScheduleCode: row.ScheduleCode, ScheduleName: row.ScheduleName,
				ErrorCode: valueOrEmpty(row.ErrorCode), OccurredAt: row.ScheduledAt.Time.UTC(),
			})
		}
	}
	if access.SecurityEvents {
		rows, err := queries.DashboardRecentSecurityEvents(ctx, db.DashboardRecentSecurityEventsParams{TenantID: &query.TenantID, StartAt: startAt, EndAt: endAt})
		if err != nil {
			return domain.Activity{}, fmt.Errorf("dashboard recent security events: %w", err)
		}
		for _, row := range rows {
			result.SecurityEvents = append(result.SecurityEvents, domain.SecurityActivity{
				ID: row.ID, EventType: row.EventType, Severity: row.Severity,
				Source: row.Source, OccurredAt: row.OccurredAt.Time.UTC(),
			})
		}
	}
	return result, nil
}

func rowsToPoints(rows []db.DashboardUserTrendRow) []domain.TrendPoint {
	result := make([]domain.TrendPoint, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.TrendPoint{Day: row.Day.Time.UTC(), Value: row.Value})
	}
	return result
}

func trendSeries(key string, points []domain.TrendPoint) domain.TrendSeries {
	return domain.TrendSeries{Key: key, Points: points}
}

func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
