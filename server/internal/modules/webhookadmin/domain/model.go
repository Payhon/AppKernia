package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden = errors.New("webhook operation forbidden")
	ErrInvalid   = errors.New("webhook operation invalid")
	ErrNotFound  = errors.New("webhook not found")
	ErrConflict  = errors.New("webhook conflict")
	ErrDelivery  = errors.New("webhook delivery failed")
)

type Principal struct {
	TenantID, UserID, SessionID     uuid.UUID
	RequestID, IPAddress, UserAgent string
}
type Filter struct {
	Query, Status, EventType string
	Page, PageSize           int32
}
type Input struct {
	Name           string   `json:"name"`
	EndpointURL    string   `json:"endpoint_url"`
	EventTypes     []string `json:"event_types"`
	MaxAttempts    int32    `json:"max_attempts"`
	TimeoutSeconds int32    `json:"timeout_seconds"`
	Status         string   `json:"status"`
}
type Endpoint struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	EndpointURL    string    `json:"endpoint_url"`
	EventTypes     []string  `json:"event_types"`
	MaxAttempts    int32     `json:"max_attempts"`
	TimeoutSeconds int32     `json:"timeout_seconds"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
type EndpointPage struct {
	Items    []Endpoint `json:"items"`
	Page     int32      `json:"page"`
	PageSize int32      `json:"page_size"`
	Total    int64      `json:"total"`
}
type CreatedEndpoint struct {
	Endpoint      Endpoint `json:"endpoint"`
	SigningSecret string   `json:"signing_secret"`
}
type Delivery struct {
	ID             uuid.UUID      `json:"id"`
	EndpointID     uuid.UUID      `json:"endpoint_id"`
	EventID        uuid.UUID      `json:"event_id"`
	EventType      string         `json:"event_type"`
	Payload        map[string]any `json:"payload"`
	Status         string         `json:"status"`
	AttemptCount   int32          `json:"attempt_count"`
	NextAttemptAt  *time.Time     `json:"next_attempt_at,omitempty"`
	ResponseStatus *int32         `json:"response_status,omitempty"`
	ResponseBody   string         `json:"response_body,omitempty"`
	LastError      string         `json:"last_error,omitempty"`
	DeliveredAt    *time.Time     `json:"delivered_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}
type DeliveryPage struct {
	Items    []Delivery `json:"items"`
	Page     int32      `json:"page"`
	PageSize int32      `json:"page_size"`
	Total    int64      `json:"total"`
}
type TestInput struct {
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
}
type StoredEndpoint struct {
	Endpoint
	TenantID         uuid.UUID
	SecretCiphertext []byte
	SecretKeyVersion int32
}
type DeliveryResult struct {
	StatusCode int
	Body       string
}

type Repository interface {
	List(context.Context, uuid.UUID, Filter) (EndpointPage, error)
	Create(context.Context, Principal, Input, []byte, int32) (Endpoint, error)
	Update(context.Context, Principal, uuid.UUID, Input) (Endpoint, error)
	GetStored(context.Context, uuid.UUID, uuid.UUID) (StoredEndpoint, error)
	CreateTestDelivery(context.Context, Principal, uuid.UUID, string, uuid.UUID, string, map[string]any) (Delivery, bool, error)
	CompleteDelivery(context.Context, Principal, uuid.UUID, uuid.UUID, DeliveryResult, error) (Delivery, error)
	Deliveries(context.Context, uuid.UUID, uuid.UUID, int32, int32) (DeliveryPage, error)
}
type Sealer interface {
	Seal([]byte, string) ([]byte, int32, error)
	Open([]byte, string) ([]byte, error)
}
type Adapter interface {
	Deliver(context.Context, string, map[string]string, []byte, time.Duration) (DeliveryResult, error)
}
