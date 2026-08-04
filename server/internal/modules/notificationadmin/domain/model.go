package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden       = errors.New("notification operation forbidden")
	ErrInvalid         = errors.New("notification operation invalid")
	ErrNotFound        = errors.New("notification resource not found")
	ErrConflict        = errors.New("notification lifecycle conflict")
	ErrRecipientEmpty  = errors.New("notification recipient set is empty")
	ErrRetryNotAllowed = errors.New("notification delivery cannot be retried")
)

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
	Type     string
	Channel  string
	Locale   string
	Page     int32
	PageSize int32
}

type Message struct {
	ID              uuid.UUID   `json:"id"`
	MessageType     string      `json:"message_type"`
	Title           string      `json:"title"`
	Body            string      `json:"body"`
	BodyFormat      string      `json:"body_format"`
	Status          string      `json:"status"`
	AudienceScope   string      `json:"audience_scope"`
	AudienceUserIDs []uuid.UUID `json:"audience_user_ids"`
	ScheduledAt     *time.Time  `json:"scheduled_at,omitempty"`
	PublishedAt     *time.Time  `json:"published_at,omitempty"`
	ExpiresAt       *time.Time  `json:"expires_at,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type MessageInput struct {
	MessageType     string      `json:"message_type"`
	Title           string      `json:"title"`
	Body            string      `json:"body"`
	BodyFormat      string      `json:"body_format"`
	AudienceScope   string      `json:"audience_scope"`
	AudienceUserIDs []uuid.UUID `json:"audience_user_ids"`
	ScheduledAt     *time.Time  `json:"scheduled_at,omitempty"`
	ExpiresAt       *time.Time  `json:"expires_at,omitempty"`
}

type MessagePage struct {
	Items    []Message `json:"items"`
	Page     int32     `json:"page"`
	PageSize int32     `json:"page_size"`
	Total    int64     `json:"total"`
}

type Recipient struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	EmailHint   string    `json:"email_hint"`
}

type RecipientPreview struct {
	Count int64       `json:"count"`
	Items []Recipient `json:"items"`
}

type RecipientStats struct {
	Total     int64 `json:"total"`
	Pending   int64 `json:"pending"`
	Delivered int64 `json:"delivered"`
	Failed    int64 `json:"failed"`
	Read      int64 `json:"read"`
}

type Template struct {
	ID              uuid.UUID       `json:"id"`
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	Channel         string          `json:"channel"`
	Locale          *string         `json:"locale,omitempty"`
	SubjectTemplate *string         `json:"subject_template,omitempty"`
	BodyTemplate    string          `json:"body_template"`
	VariablesSchema json.RawMessage `json:"variables_schema"`
	Status          string          `json:"status"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type TemplateInput struct {
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	Channel         string          `json:"channel"`
	Locale          *string         `json:"locale,omitempty"`
	SubjectTemplate *string         `json:"subject_template,omitempty"`
	BodyTemplate    string          `json:"body_template"`
	VariablesSchema json.RawMessage `json:"variables_schema"`
	Status          string          `json:"status"`
}

type TemplatePage struct {
	Items    []Template `json:"items"`
	Page     int32      `json:"page"`
	PageSize int32      `json:"page_size"`
	Total    int64      `json:"total"`
}

type Delivery struct {
	ID            uuid.UUID  `json:"id"`
	MessageID     *uuid.UUID `json:"message_id,omitempty"`
	UserID        *uuid.UUID `json:"user_id,omitempty"`
	TemplateID    *uuid.UUID `json:"template_id,omitempty"`
	Channel       string     `json:"channel"`
	TargetHint    string     `json:"target_hint"`
	Provider      string     `json:"provider"`
	Status        string     `json:"status"`
	AttemptCount  int32      `json:"attempt_count"`
	MaxAttempts   int32      `json:"max_attempts"`
	ScheduledAt   time.Time  `json:"scheduled_at"`
	NextAttemptAt *time.Time `json:"next_attempt_at,omitempty"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	ErrorCode     string     `json:"error_code,omitempty"`
	ErrorSummary  string     `json:"error_summary,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type DeliveryPage struct {
	Items    []Delivery `json:"items"`
	Page     int32      `json:"page"`
	PageSize int32      `json:"page_size"`
	Total    int64      `json:"total"`
}

type Repository interface {
	ListMessages(context.Context, uuid.UUID, bool, PageFilter) (MessagePage, error)
	GetMessage(context.Context, uuid.UUID, uuid.UUID, bool) (Message, error)
	CreateMessage(context.Context, Principal, bool, MessageInput) (Message, error)
	UpdateMessage(context.Context, Principal, uuid.UUID, bool, MessageInput) (Message, error)
	PreviewRecipients(context.Context, uuid.UUID, Message) (RecipientPreview, error)
	PublishMessage(context.Context, Principal, uuid.UUID, bool) (Message, RecipientPreview, error)
	CancelMessage(context.Context, Principal, uuid.UUID, bool) (Message, error)
	RecipientStats(context.Context, uuid.UUID, uuid.UUID, bool) (RecipientStats, error)
	ListTemplates(context.Context, uuid.UUID, PageFilter) (TemplatePage, error)
	CreateTemplate(context.Context, Principal, TemplateInput) (Template, error)
	UpdateTemplate(context.Context, Principal, uuid.UUID, TemplateInput) (Template, error)
	ListDeliveries(context.Context, uuid.UUID, PageFilter) (DeliveryPage, error)
	GetDelivery(context.Context, uuid.UUID, uuid.UUID) (Delivery, error)
	RetryDelivery(context.Context, Principal, uuid.UUID) (Delivery, error)
}
