package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden  = errors.New("api client operation forbidden")
	ErrInvalid    = errors.New("api client operation invalid")
	ErrNotFound   = errors.New("api client not found")
	ErrConflict   = errors.New("api client conflict")
	ErrCredential = errors.New("api client credential invalid")
)

type Principal struct {
	TenantID, UserID, SessionID     uuid.UUID
	RequestID, IPAddress, UserAgent string
}
type Filter struct {
	Query, Status  string
	Page, PageSize int32
}
type Input struct {
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	AllowedCIDRs []string   `json:"allowed_cidrs"`
	Status       string     `json:"status"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	BoundUserID  *uuid.UUID `json:"bound_user_id"`
}
type Secret struct {
	ID         uuid.UUID  `json:"id"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}
type BoundUser struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
}
type Client struct {
	ID           uuid.UUID   `json:"id"`
	TenantID     uuid.UUID   `json:"-"`
	ClientID     string      `json:"client_id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	AllowedCIDRs []string    `json:"allowed_cidrs"`
	Status       string      `json:"status"`
	ExpiresAt    *time.Time  `json:"expires_at,omitempty"`
	BoundUserID  *uuid.UUID  `json:"bound_user_id"`
	BoundUser    *BoundUser  `json:"bound_user"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Secrets      []Secret    `json:"secrets"`
	Permissions  []string    `json:"permissions"`
	AppIDs       []uuid.UUID `json:"app_ids"`
}
type Page struct {
	Items    []Client `json:"items"`
	Page     int32    `json:"page"`
	PageSize int32    `json:"page_size"`
	Total    int64    `json:"total"`
}
type CreatedSecret struct {
	Secret    Secret `json:"metadata"`
	Plaintext string `json:"secret"`
}
type Credential struct {
	ClientID   string
	SecretHash []byte
	IPAddress  string
}

type TokenMetadata struct {
	RequestID, IPAddress, UserAgent string
}

type TokenExchangeAudit struct {
	TenantID       *uuid.UUID
	ClientID       string
	IdentifierHash []byte
	RequestID      string
	IPAddress      string
	UserAgent      string
	Result         string
	FailureReason  string
}

type MachinePrincipal struct {
	TenantID    uuid.UUID
	ClientID    uuid.UUID
	AppID       uuid.UUID
	Permissions []string
}

type AgentAudit struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	ClientID  uuid.UUID
	RequestID string
	Operation string
	Method    string
	Path      string
	IPAddress string
	UserAgent string
}

type Repository interface {
	List(context.Context, uuid.UUID, Filter) (Page, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (Client, error)
	Create(context.Context, Principal, string, Input) (Client, error)
	Update(context.Context, Principal, uuid.UUID, Input) (Client, error)
	CreateSecret(context.Context, Principal, uuid.UUID, string, []byte, *time.Time) (Secret, error)
	RevokeSecret(context.Context, Principal, uuid.UUID, uuid.UUID) error
	ReplacePermissions(context.Context, Principal, uuid.UUID, []string) (Client, error)
	ReplaceApps(context.Context, Principal, uuid.UUID, []uuid.UUID) (Client, error)
	Authenticate(context.Context, Credential) (Client, error)
	AuditTokenExchange(context.Context, TokenExchangeAudit) error
	AuditAgentAuthentication(context.Context, AgentAudit) error
}
