package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OperationsFilter struct {
	Environment string
	Category    string
	Channel     string
	Provider    string
	TaskKind    string
	Status      string
	Query       string
	From        time.Time
	To          time.Time
	Page        int32
	PageSize    int32
}

type OperationsSummary struct {
	Queued            int64   `json:"queued"`
	Running           int64   `json:"running"`
	RetryWaiting      int64   `json:"retry_waiting"`
	Accepted          int64   `json:"accepted"`
	Failed            int64   `json:"failed"`
	InvalidTokens     int64   `json:"invalid_tokens"`
	Skipped           int64   `json:"skipped"`
	Opened            int64   `json:"opened"`
	OpenRate          float64 `json:"open_rate"`
	OldestWaitingSecs int64   `json:"oldest_waiting_seconds"`
	P95QueueDelayMS   int64   `json:"p95_queue_delay_ms"`
	HasUnfinishedWork bool    `json:"has_unfinished_work"`
}

type TrendBucket struct {
	Bucket        time.Time `json:"bucket"`
	Accepted      int64     `json:"accepted"`
	Failed        int64     `json:"failed"`
	InvalidTokens int64     `json:"invalid_tokens"`
	Opened        int64     `json:"opened"`
	Skipped       int64     `json:"skipped"`
}

type MessageRun struct {
	ID                uuid.UUID  `json:"id"`
	MessageID         uuid.UUID  `json:"message_id"`
	MessageTitle      string     `json:"message_title"`
	Category          string     `json:"category"`
	TriggerType       string     `json:"trigger_type"`
	Status            string     `json:"status"`
	RecipientCount    int64      `json:"recipient_count"`
	EvaluatedCount    int64      `json:"evaluated_count"`
	DeliveryCount     int64      `json:"delivery_count"`
	AcceptedCount     int64      `json:"accepted_count"`
	FailedCount       int64      `json:"failed_count"`
	InvalidTokenCount int64      `json:"invalid_token_count"`
	OpenedCount       int64      `json:"opened_count"`
	SkippedCount      int64      `json:"skipped_count"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type MessageRunPage struct {
	Items    []MessageRun `json:"items"`
	Page     int32        `json:"page"`
	PageSize int32        `json:"page_size"`
	Total    int64        `json:"total"`
}

type TaskAttempt struct {
	AttemptNumber     int32      `json:"attempt_number"`
	Status            string     `json:"status"`
	ResultClass       string     `json:"result_class,omitempty"`
	ErrorCode         string     `json:"error_code,omitempty"`
	ErrorSummary      string     `json:"error_summary,omitempty"`
	ExternalRequestID string     `json:"external_request_id,omitempty"`
	TraceID           string     `json:"trace_id,omitempty"`
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	DurationMS        *int64     `json:"duration_ms,omitempty"`
	NextRetryAt       *time.Time `json:"next_retry_at,omitempty"`
}

type TaskRun struct {
	ID               uuid.UUID     `json:"id"`
	TaskKind         string        `json:"task_kind"`
	QueueName        string        `json:"queue_name"`
	ResourceType     string        `json:"resource_type"`
	ResourceID       *uuid.UUID    `json:"resource_id,omitempty"`
	CorrelationID    *uuid.UUID    `json:"correlation_id,omitempty"`
	Status           string        `json:"status"`
	ScheduledAt      time.Time     `json:"scheduled_at"`
	StartedAt        *time.Time    `json:"started_at,omitempty"`
	FinalizedAt      *time.Time    `json:"finalized_at,omitempty"`
	NextRetryAt      *time.Time    `json:"next_retry_at,omitempty"`
	AttemptCount     int32         `json:"attempt_count"`
	MaxAttempts      int32         `json:"max_attempts"`
	LastResultClass  string        `json:"last_result_class,omitempty"`
	LastErrorCode    string        `json:"last_error_code,omitempty"`
	LastErrorSummary string        `json:"last_error_summary,omitempty"`
	TraceID          string        `json:"trace_id,omitempty"`
	Retryable        bool          `json:"retryable"`
	RetryRisk        string        `json:"retry_risk"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	Attempts         []TaskAttempt `json:"attempts,omitempty"`
}

type TaskRunPage struct {
	Items    []TaskRun `json:"items"`
	Page     int32     `json:"page"`
	PageSize int32     `json:"page_size"`
	Total    int64     `json:"total"`
}

type Failure struct {
	TaskRun
	MessageID    *uuid.UUID `json:"message_id,omitempty"`
	MessageTitle string     `json:"message_title,omitempty"`
	Provider     string     `json:"provider,omitempty"`
	Channel      string     `json:"channel,omitempty"`
}

type FailurePage struct {
	Items    []Failure `json:"items"`
	Page     int32     `json:"page"`
	PageSize int32     `json:"page_size"`
	Total    int64     `json:"total"`
}

type RetryItem struct {
	TaskID uuid.UUID `json:"task_id"`
}

type RetryInput struct {
	Items                    []RetryItem `json:"items"`
	AcknowledgeDuplicateRisk bool        `json:"acknowledge_duplicate_risk"`
}

type RetryResult struct {
	TaskID    uuid.UUID  `json:"task_id"`
	Accepted  bool       `json:"accepted"`
	NewTaskID *uuid.UUID `json:"new_task_id,omitempty"`
	Reason    string     `json:"reason,omitempty"`
}

type OperationsRepository interface {
	OperationsSummary(context.Context, uuid.UUID, uuid.UUID, OperationsFilter) (OperationsSummary, error)
	OperationsTrends(context.Context, uuid.UUID, uuid.UUID, OperationsFilter) ([]TrendBucket, error)
	ListMessageRuns(context.Context, uuid.UUID, uuid.UUID, OperationsFilter) (MessageRunPage, error)
	GetMessageRun(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (MessageRun, error)
	ListTaskRuns(context.Context, uuid.UUID, uuid.UUID, OperationsFilter) (TaskRunPage, error)
	GetTaskRun(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (TaskRun, error)
	ListFailures(context.Context, uuid.UUID, uuid.UUID, OperationsFilter) (FailurePage, error)
	RetryTasks(context.Context, Principal, RetryInput) ([]RetryResult, error)
}
