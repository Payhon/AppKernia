package domain

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden     = errors.New("session administration permission denied")
	ErrInvalid       = errors.New("session administration input invalid")
	ErrSessionAbsent = errors.New("session is not active or does not exist")
)

type Filter struct {
	Query, Audience, Platform, Status, IP string
	FromAt, ToAt                          time.Time
	Page, PageSize                        int32
}

type Session struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	UserHint    string     `json:"user_hint"`
	DisplayName string     `json:"display_name"`
	Audience    string     `json:"audience"`
	Platform    string     `json:"platform"`
	DeviceHint  string     `json:"device_hint"`
	IPHint      string     `json:"ip_hint"`
	Status      string     `json:"status"`
	Current     bool       `json:"current"`
	LastSeenAt  time.Time  `json:"last_seen_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
}

type Page struct {
	Items    []Session `json:"items"`
	Total    int64     `json:"total"`
	Page     int32     `json:"page"`
	PageSize int32     `json:"page_size"`
}

type Principal struct {
	TenantID, UserID, SessionID uuid.UUID
	RequestID                   string
	IPAddress                   *netip.Addr
	UserAgent                   string
}

type RevokeResult struct {
	ID      uuid.UUID `json:"id"`
	Revoked bool      `json:"revoked"`
	Current bool      `json:"current"`
}

type Repository interface {
	List(context.Context, uuid.UUID, uuid.UUID, Filter) (Page, error)
	Revoke(context.Context, Principal, uuid.UUID) (RevokeResult, error)
}
