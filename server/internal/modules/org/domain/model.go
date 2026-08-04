package domain

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden        = errors.New("organization permission denied")
	ErrUnitNotFound     = errors.New("organization unit not found")
	ErrUnitCodeConflict = errors.New("organization unit code conflict")
	ErrUnitCycle        = errors.New("organization unit cycle")
	ErrUnitOccupied     = errors.New("organization unit occupied")
	ErrPositionNotFound = errors.New("organization position not found")
	ErrPositionConflict = errors.New("organization position code conflict")
	ErrPositionOccupied = errors.New("organization position occupied")
	ErrInvalid          = errors.New("organization input invalid")
)

type Principal struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	SessionID uuid.UUID
	RequestID string
	IPAddress *netip.Addr
	UserAgent string
}

type Unit struct {
	ID                uuid.UUID
	ParentID          *uuid.UUID
	Code              string
	Name              string
	UnitType          string
	Phone             string
	Email             string
	SortOrder         int32
	Status            string
	DirectMemberCount int64
	ChildCount        int64
	UpdatedAt         time.Time
	Children          []Unit
}

type UnitInput struct {
	ParentID  *uuid.UUID
	Code      string
	Name      string
	UnitType  string
	Phone     string
	Email     string
	SortOrder int32
	Status    string
}

type UnitMove struct {
	ParentID  *uuid.UUID
	SortOrder int32
}

type Position struct {
	ID          uuid.UUID
	Code        string
	Name        string
	Description string
	SortOrder   int32
	Status      string
	MemberCount int64
	UpdatedAt   time.Time
}

type PositionInput struct {
	Code        string
	Name        string
	Description string
	SortOrder   int32
	Status      string
}

type UnitOccupancy struct {
	ChildCount  int64
	MemberCount int64
}

type Repository interface {
	ListUnits(context.Context, uuid.UUID) ([]Unit, error)
	CreateUnit(context.Context, Principal, UnitInput) (Unit, error)
	UpdateUnit(context.Context, Principal, uuid.UUID, UnitInput) (Unit, error)
	MoveUnit(context.Context, Principal, uuid.UUID, UnitMove) (Unit, error)
	DeleteUnit(context.Context, Principal, uuid.UUID) (UnitOccupancy, error)
	ListPositions(context.Context, uuid.UUID, string, string, *uuid.UUID) ([]Position, error)
	CreatePosition(context.Context, Principal, PositionInput) (Position, error)
	UpdatePosition(context.Context, Principal, uuid.UUID, PositionInput) (Position, error)
	DeletePosition(context.Context, Principal, uuid.UUID) (int64, error)
}
