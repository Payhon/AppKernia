package domain

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden       = errors.New("audit administration permission denied")
	ErrInvalid         = errors.New("audit administration input invalid")
	ErrNotFound        = errors.New("audit resource not found")
	ErrAlreadyResolved = errors.New("security event already resolved")
)

type Principal struct {
	TenantID, UserID, SessionID uuid.UUID
	RequestID                   string
	IPAddress                   *netip.Addr
	UserAgent                   string
}

type PageFilter struct {
	Query          string
	FromAt, ToAt   time.Time
	Page, PageSize int32
}

type OperationFilter struct {
	PageFilter
	ModuleCode, Result string
}

type LoginFilter struct {
	PageFilter
	Result, Audience, AuthMethod string
}

type SecurityFilter struct {
	PageFilter
	Severity, Source, Status string
}

type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int32 `json:"page"`
	PageSize int32 `json:"page_size"`
}

type Operation struct {
	ID             uuid.UUID      `json:"id"`
	UserID         *uuid.UUID     `json:"user_id"`
	SessionID      *uuid.UUID     `json:"session_id"`
	RequestID      string         `json:"request_id"`
	TraceID        string         `json:"trace_id"`
	ModuleCode     string         `json:"module_code"`
	ActionName     string         `json:"action_name"`
	PermissionCode string         `json:"permission_code"`
	ResourceType   string         `json:"resource_type"`
	ResourceID     string         `json:"resource_id"`
	HTTPMethod     string         `json:"http_method"`
	RequestPath    string         `json:"request_path"`
	ResponseStatus *int32         `json:"response_status"`
	ClientIP       string         `json:"client_ip"`
	RequestSummary map[string]any `json:"request_summary"`
	BeforeData     map[string]any `json:"before_data"`
	AfterData      map[string]any `json:"after_data"`
	DurationMS     *int32         `json:"duration_ms"`
	Succeeded      bool           `json:"succeeded"`
	ErrorCode      string         `json:"error_code"`
	OccurredAt     time.Time      `json:"occurred_at"`
}

type Login struct {
	ID                  uuid.UUID  `json:"id"`
	UserID              *uuid.UUID `json:"user_id"`
	SessionID           *uuid.UUID `json:"session_id"`
	RequestID           string     `json:"request_id"`
	LoginIdentifierHint string     `json:"login_identifier_hint"`
	AuthMethod          string     `json:"auth_method"`
	Audience            string     `json:"audience"`
	Result              string     `json:"result"`
	FailureReason       string     `json:"failure_reason"`
	ClientIP            string     `json:"client_ip"`
	OccurredAt          time.Time  `json:"occurred_at"`
}

type SecurityEvent struct {
	ID         uuid.UUID      `json:"id"`
	UserID     *uuid.UUID     `json:"user_id"`
	SessionID  *uuid.UUID     `json:"session_id"`
	EventType  string         `json:"event_type"`
	Severity   string         `json:"severity"`
	Source     string         `json:"source"`
	ClientIP   string         `json:"client_ip"`
	Details    map[string]any `json:"details"`
	ResolvedAt *time.Time     `json:"resolved_at"`
	ResolvedBy *uuid.UUID     `json:"resolved_by"`
	OccurredAt time.Time      `json:"occurred_at"`
}

type Repository interface {
	ListOperations(context.Context, uuid.UUID, OperationFilter) (Page[Operation], error)
	ListLogins(context.Context, uuid.UUID, LoginFilter) (Page[Login], error)
	ListSecurityEvents(context.Context, uuid.UUID, SecurityFilter) (Page[SecurityEvent], error)
	GetSecurityEvent(context.Context, uuid.UUID, uuid.UUID) (SecurityEvent, error)
	ResolveSecurityEvent(context.Context, Principal, uuid.UUID) (SecurityEvent, error)
}
