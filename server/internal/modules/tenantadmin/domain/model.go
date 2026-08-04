package domain

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden = errors.New("tenant administration permission denied")
	ErrInvalid   = errors.New("tenant administration input invalid")
	ErrNotFound  = errors.New("tenant not found")
	ErrConflict  = errors.New("tenant code or member conflict")
	ErrLastAdmin = errors.New("last tenant administrator cannot leave")
)

type Principal struct {
	TenantID, UserID, SessionID uuid.UUID
	RequestID                   string
	IPAddress                   *netip.Addr
	UserAgent                   string
}
type Tenant struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	PlanCode    string    `json:"plan_code"`
	MemberCount int64     `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Filters struct {
	Query, Status, Sort string
	Page, PageSize      int32
}
type Page struct {
	Items          []Tenant `json:"items"`
	Total          int64    `json:"total"`
	Page, PageSize int32
}
type CreateInput struct{ Code, Name string }
type UpdateInput struct{ Name, Status string }
type Member struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status"`
	JoinedAt    time.Time `json:"joined_at"`
	RoleCodes   []string  `json:"role_codes"`
}
type AddMemberInput struct{ Email, DisplayName string }

type Repository interface {
	List(context.Context, uuid.UUID, Filters) (Page, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (Tenant, error)
	Create(context.Context, Principal, CreateInput) (Tenant, error)
	Update(context.Context, Principal, uuid.UUID, UpdateInput) (Tenant, error)
	Members(context.Context, uuid.UUID, uuid.UUID) ([]Member, error)
	AddMember(context.Context, Principal, uuid.UUID, AddMemberInput) (Member, error)
	SetMemberStatus(context.Context, Principal, uuid.UUID, uuid.UUID, string) (Member, error)
}
