package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden       = errors.New("job schedule operation forbidden")
	ErrInvalid         = errors.New("job schedule operation invalid")
	ErrNotFound        = errors.New("job schedule not found")
	ErrConflict        = errors.New("job schedule conflict")
	ErrHandlerUnknown  = errors.New("job handler is not registered")
	ErrTransition      = errors.New("job schedule transition is not allowed")
	ErrExecutionDenied = errors.New("job schedule execution is not allowed")
)

const ScheduleRunJobKind = "appkernia-schedule-run"

type ScheduleRunArgs struct {
	RunID uuid.UUID `json:"run_id"`
}

func (ScheduleRunArgs) Kind() string { return ScheduleRunJobKind }

type Principal struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	SessionID uuid.UUID
	RequestID string
	IPAddress string
	UserAgent string
}

type PageFilter struct {
	Query    string
	Status   string
	TimeZone string
	Page     int32
	PageSize int32
}

type HandlerDefinition struct {
	Key             string          `json:"key"`
	NameKey         string          `json:"name_key"`
	DescriptionKey  string          `json:"description_key"`
	PayloadSchema   json.RawMessage `json:"payload_schema"`
	DefaultQueue    string          `json:"default_queue"`
	AllowedPayloads []string        `json:"-"`
}

func CompiledHandlers() []HandlerDefinition {
	return []HandlerDefinition{
		{
			Key:            "system.health.snapshot",
			NameKey:        "schedules.handlers.system_health_snapshot.name",
			DescriptionKey: "schedules.handlers.system_health_snapshot.description",
			PayloadSchema:  json.RawMessage(`{"type":"object","additionalProperties":false}`),
			DefaultQueue:   "default",
		},
	}
}

func FindCompiledHandler(key string) (HandlerDefinition, bool) {
	for _, item := range CompiledHandlers() {
		if item.Key == key {
			return item, true
		}
	}
	return HandlerDefinition{}, false
}

type Schedule struct {
	ID             uuid.UUID       `json:"id"`
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	HandlerKey     string          `json:"handler_key"`
	CronExpression string          `json:"cron_expression"`
	TimeZone       string          `json:"time_zone"`
	Payload        json.RawMessage `json:"payload"`
	QueueName      string          `json:"queue_name"`
	OverlapPolicy  string          `json:"overlap_policy"`
	MisfirePolicy  string          `json:"misfire_policy"`
	TimeoutSeconds int32           `json:"timeout_seconds"`
	MaxAttempts    int32           `json:"max_attempts"`
	Status         string          `json:"status"`
	LastEnqueuedAt *time.Time      `json:"last_enqueued_at,omitempty"`
	NextRunAt      *time.Time      `json:"next_run_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type ScheduleInput struct {
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	HandlerKey     string          `json:"handler_key"`
	CronExpression string          `json:"cron_expression"`
	TimeZone       string          `json:"time_zone"`
	Payload        json.RawMessage `json:"payload"`
	QueueName      string          `json:"queue_name"`
	OverlapPolicy  string          `json:"overlap_policy"`
	MisfirePolicy  string          `json:"misfire_policy"`
	TimeoutSeconds int32           `json:"timeout_seconds"`
	MaxAttempts    int32           `json:"max_attempts"`
}

type SchedulePage struct {
	Items    []Schedule `json:"items"`
	Page     int32      `json:"page"`
	PageSize int32      `json:"page_size"`
	Total    int64      `json:"total"`
}

type CronPreview struct {
	CronExpression string      `json:"cron_expression"`
	TimeZone       string      `json:"time_zone"`
	NextRuns       []time.Time `json:"next_runs"`
}

type Run struct {
	ID           uuid.UUID       `json:"id"`
	ScheduleID   uuid.UUID       `json:"schedule_id"`
	RiverJobID   *int64          `json:"river_job_id,omitempty"`
	TriggerType  string          `json:"trigger_type"`
	Status       string          `json:"status"`
	Attempt      int32           `json:"attempt"`
	ScheduledAt  time.Time       `json:"scheduled_at"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	FinishedAt   *time.Time      `json:"finished_at,omitempty"`
	WorkerID     string          `json:"worker_id,omitempty"`
	Output       json.RawMessage `json:"output,omitempty"`
	ErrorCode    string          `json:"error_code,omitempty"`
	ErrorSummary string          `json:"error_summary,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type RunPage struct {
	Items    []Run `json:"items"`
	Page     int32 `json:"page"`
	PageSize int32 `json:"page_size"`
	Total    int64 `json:"total"`
}

type Repository interface {
	ListSchedules(context.Context, uuid.UUID, PageFilter) (SchedulePage, error)
	GetSchedule(context.Context, uuid.UUID, uuid.UUID) (Schedule, error)
	CreateSchedule(context.Context, Principal, ScheduleInput, time.Time) (Schedule, error)
	UpdateSchedule(context.Context, Principal, uuid.UUID, ScheduleInput, time.Time) (Schedule, error)
	ChangeStatus(context.Context, Principal, uuid.UUID, string, *time.Time) (Schedule, error)
	Execute(context.Context, Principal, uuid.UUID, string, time.Time) (Run, error)
	ListRuns(context.Context, uuid.UUID, uuid.UUID, PageFilter) (RunPage, error)
}
