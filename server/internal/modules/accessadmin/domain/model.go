package domain

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden    = errors.New("access administration permission denied")
	ErrInvalid      = errors.New("access administration input invalid")
	ErrNotFound     = errors.New("access administration resource not found")
	ErrConflict     = errors.New("access administration resource conflict")
	ErrSystemRole   = errors.New("system role is immutable")
	ErrRoleOccupied = errors.New("role has assigned members")
	ErrMenuOccupied = errors.New("menu has children or role assignments")
	ErrMenuCycle    = errors.New("menu hierarchy cycle")
	ErrMenuDepth    = errors.New("menu hierarchy exceeds maximum depth")
	ErrComponentKey = errors.New("menu component key is not registered")
)

type Principal struct {
	TenantID, UserID, SessionID uuid.UUID
	RequestID                   string
	IPAddress                   *netip.Addr
	UserAgent                   string
}

type Filters struct {
	Query, Status, RoleType string
	Page, PageSize          int32
}

type Role struct {
	ID            uuid.UUID   `json:"id"`
	ParentID      *uuid.UUID  `json:"parent_id"`
	Code          string      `json:"code"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	RoleType      string      `json:"role_type"`
	DataScope     string      `json:"data_scope"`
	SortOrder     int32       `json:"sort_order"`
	IsDefault     bool        `json:"is_default"`
	IsSystem      bool        `json:"is_system"`
	Status        string      `json:"status"`
	MemberCount   int64       `json:"member_count"`
	PermissionIDs []uuid.UUID `json:"permission_ids"`
	MenuIDs       []uuid.UUID `json:"menu_ids"`
	ScopeUnitIDs  []uuid.UUID `json:"scope_unit_ids"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type RolePage struct {
	Items    []Role `json:"items"`
	Total    int64  `json:"total"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
}

type RoleInput struct {
	ParentID    *uuid.UUID
	Code        string
	Name        string
	Description string
	SortOrder   int32
	Status      string
}

type Permission struct {
	ID             uuid.UUID `json:"id"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	ModuleCode     string    `json:"module_code"`
	ResourceName   string    `json:"resource_name"`
	ActionName     string    `json:"action_name"`
	PermissionKind string    `json:"permission_kind"`
	HTTPMethods    []string  `json:"http_methods"`
	RoutePattern   string    `json:"route_pattern"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
}

type PermissionFilters struct{ Query, ModuleCode, ResourceName, ActionName, PermissionKind, Status string }

type Menu struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       *uuid.UUID `json:"tenant_id"`
	ParentID       *uuid.UUID `json:"parent_id"`
	PermissionID   *uuid.UUID `json:"permission_id"`
	PermissionCode string     `json:"permission_code"`
	Code           string     `json:"code"`
	Title          string     `json:"title"`
	I18nKey        string     `json:"i18n_key"`
	Type           string     `json:"type"`
	Path           string     `json:"path"`
	ComponentKey   string     `json:"component_key"`
	Icon           string     `json:"icon"`
	ExternalURL    string     `json:"external_url"`
	OpenMode       string     `json:"open_mode"`
	Hidden         bool       `json:"hidden"`
	Affix          bool       `json:"affix"`
	SortOrder      int32      `json:"sort_order"`
	Status         string     `json:"status"`
	IsCore         bool       `json:"is_core"`
	RoleCount      int64      `json:"role_count"`
	Children       []Menu     `json:"children"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type MenuInput struct {
	ParentID     *uuid.UUID
	PermissionID *uuid.UUID
	Code         string
	Title        string
	I18nKey      string
	Type         string
	Path         string
	ComponentKey string
	Icon         string
	ExternalURL  string
	OpenMode     string
	Hidden       bool
	Affix        bool
	SortOrder    int32
	Status       string
}

type MenuMove struct {
	ParentID  *uuid.UUID
	SortOrder int32
}

type Repository interface {
	ListRoles(context.Context, uuid.UUID, Filters) (RolePage, error)
	CreateRole(context.Context, Principal, RoleInput) (Role, error)
	UpdateRole(context.Context, Principal, uuid.UUID, RoleInput) (Role, error)
	DeleteRole(context.Context, Principal, uuid.UUID) error
	ReplaceRolePermissions(context.Context, Principal, uuid.UUID, []uuid.UUID) (Role, error)
	ReplaceRoleMenus(context.Context, Principal, uuid.UUID, []uuid.UUID) (Role, error)
	ReplaceRoleDataScope(context.Context, Principal, uuid.UUID, string, []uuid.UUID) (Role, error)
	ListPermissions(context.Context, PermissionFilters) ([]Permission, error)
	ListMenus(context.Context, uuid.UUID) ([]Menu, error)
	CreateMenu(context.Context, Principal, MenuInput) (Menu, error)
	UpdateMenu(context.Context, Principal, uuid.UUID, MenuInput) (Menu, error)
	MoveMenu(context.Context, Principal, uuid.UUID, MenuMove) (Menu, error)
	DeleteMenu(context.Context, Principal, uuid.UUID) error
}
