package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden = errors.New("block rule operation forbidden")
	ErrInvalid   = errors.New("block rule input invalid")
	ErrNotFound  = errors.New("block rule not found")
)

type Principal struct {
	TenantID, UserID, SessionID     uuid.UUID
	RequestID, IPAddress, UserAgent string
}
type Filter struct {
	SubjectType, SubjectHint, Scope, Status, Expiry string
	Page, PageSize                                  int32
}
type CreateInput struct {
	SubjectType  string     `json:"subject_type"`
	SubjectValue string     `json:"subject_value"`
	Action       string     `json:"action"`
	Reason       string     `json:"reason"`
	StartsAt     *time.Time `json:"starts_at,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Status       string     `json:"status"`
}
type UpdateInput struct {
	Action    string     `json:"action"`
	Reason    string     `json:"reason"`
	StartsAt  time.Time  `json:"starts_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Status    string     `json:"status"`
}
type Rule struct {
	ID          uuid.UUID  `json:"id"`
	SubjectType string     `json:"subject_type"`
	SubjectHint string     `json:"subject_hint"`
	Scope       string     `json:"scope"`
	Action      string     `json:"action"`
	Reason      string     `json:"reason"`
	StartsAt    time.Time  `json:"starts_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
type Page struct {
	Items    []Rule `json:"items"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
	Total    int64  `json:"total"`
}
type RevokeResult struct {
	ID      uuid.UUID `json:"id"`
	Revoked bool      `json:"revoked"`
}

type Repository interface {
	List(context.Context, uuid.UUID, Filter) (Page, error)
	Create(context.Context, Principal, CreateInput) (Rule, error)
	Update(context.Context, Principal, uuid.UUID, UpdateInput) (Rule, error)
	Revoke(context.Context, Principal, uuid.UUID) (RevokeResult, error)
}
