package domain

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden     = errors.New("user administration permission denied")
	ErrInvalid       = errors.New("user administration input invalid")
	ErrNotFound      = errors.New("tenant user not found")
	ErrEmailConflict = errors.New("user email conflict")
	ErrRoleInvalid   = errors.New("role does not belong to tenant")
	ErrOrgInvalid    = errors.New("organization assignment does not belong to tenant")
	ErrLastAdmin     = errors.New("last active tenant administrator cannot be disabled")
	ErrSessionAbsent = errors.New("user session not found")
)

type Principal struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	SessionID uuid.UUID
	RequestID string
	IPAddress *netip.Addr
	UserAgent string
}

type Reference struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

type User struct {
	ID                 uuid.UUID   `json:"id"`
	Email              string      `json:"email"`
	Username           string      `json:"username"`
	DisplayName        string      `json:"display_name"`
	Locale             string      `json:"locale"`
	TimeZone           string      `json:"time_zone"`
	Status             string      `json:"status"`
	GlobalStatus       string      `json:"global_status"`
	MemberStatus       string      `json:"member_status"`
	IsSystem           bool        `json:"is_system"`
	Roles              []Reference `json:"roles"`
	Units              []Reference `json:"units"`
	Positions          []Reference `json:"positions"`
	ActiveSessionCount int64       `json:"active_session_count"`
	LastLoginAt        *time.Time  `json:"last_login_at"`
	LastActiveAt       *time.Time  `json:"last_active_at"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

type Filters struct {
	Query       string
	Status      string
	UnitID      *uuid.UUID
	PositionID  *uuid.UUID
	RoleID      *uuid.UUID
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Page        int32
	PageSize    int32
	Sort        string
}

type Page struct {
	Items    []User `json:"items"`
	Total    int64  `json:"total"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
}

type CreateInput struct {
	Email             string
	DisplayName       string
	Locale            string
	TimeZone          string
	TemporaryPassword string
}

type UpdateInput struct {
	DisplayName string
	Locale      string
	TimeZone    string
}

type AssignmentInput struct {
	UnitIDs           []uuid.UUID
	PrimaryUnitID     *uuid.UUID
	PositionIDs       []uuid.UUID
	PrimaryPositionID *uuid.UUID
}

type Session struct {
	ID         uuid.UUID  `json:"id"`
	Audience   string     `json:"audience"`
	Status     string     `json:"status"`
	IPAddress  string     `json:"ip_address"`
	UserAgent  string     `json:"user_agent"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	Current    bool       `json:"current"`
}

type ImportRow struct {
	Row               int
	Email             string
	DisplayName       string
	Locale            string
	TimeZone          string
	TemporaryPassword string
}

type ImportError struct {
	Row  int    `json:"row"`
	Code string `json:"code"`
}

type ImportResult struct {
	Created int           `json:"created"`
	Failed  int           `json:"failed"`
	Errors  []ImportError `json:"errors"`
}

type Repository interface {
	ListRoleOptions(context.Context, uuid.UUID) ([]Reference, error)
	ListUsers(context.Context, uuid.UUID, Filters) (Page, error)
	GetUser(context.Context, uuid.UUID, uuid.UUID) (User, error)
	CreateUser(context.Context, Principal, CreateInput, string) (User, error)
	UpdateUser(context.Context, Principal, uuid.UUID, UpdateInput) (User, error)
	SetMemberStatus(context.Context, Principal, uuid.UUID, string) (User, error)
	UnlockUser(context.Context, Principal, uuid.UUID) error
	ResetPassword(context.Context, Principal, uuid.UUID, string) (int64, error)
	ReplaceRoles(context.Context, Principal, uuid.UUID, []uuid.UUID) (User, error)
	ReplaceAssignments(context.Context, Principal, uuid.UUID, AssignmentInput) (User, error)
	ListSessions(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) ([]Session, error)
	RevokeSession(context.Context, Principal, uuid.UUID, uuid.UUID) error
}
