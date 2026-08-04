package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Query struct {
	TenantID uuid.UUID
	StartAt  time.Time
	EndAt    time.Time
}

type SummaryAccess struct {
	Users          bool
	Sessions       bool
	FailedJobs     bool
	SecurityEvents bool
	Messages       bool
}

type Metric struct {
	Key   string
	Value int64
}

type TrendAccess struct {
	Logins         bool
	Users          bool
	FailedJobs     bool
	SecurityEvents bool
}

type TrendPoint struct {
	Day   time.Time
	Value int64
}

type TrendSeries struct {
	Key    string
	Points []TrendPoint
}

type ActivityAccess struct {
	Operations     bool
	FailedJobs     bool
	SecurityEvents bool
}

type OperationActivity struct {
	ID           uuid.UUID
	ModuleCode   string
	ActionName   string
	ResourceType string
	Succeeded    bool
	ErrorCode    string
	OccurredAt   time.Time
}

type FailedJobActivity struct {
	ID           uuid.UUID
	ScheduleCode string
	ScheduleName string
	ErrorCode    string
	OccurredAt   time.Time
}

type SecurityActivity struct {
	ID         uuid.UUID
	EventType  string
	Severity   string
	Source     string
	OccurredAt time.Time
}

type Activity struct {
	Operations     []OperationActivity
	FailedJobs     []FailedJobActivity
	SecurityEvents []SecurityActivity
}

type Repository interface {
	Summary(context.Context, Query, SummaryAccess) ([]Metric, error)
	Trends(context.Context, Query, TrendAccess) ([]TrendSeries, error)
	Activity(context.Context, Query, ActivityAccess) (Activity, error)
}
