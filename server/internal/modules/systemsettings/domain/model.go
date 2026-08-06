package domain

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden         = errors.New("system settings permission denied")
	ErrInvalid           = errors.New("system settings input invalid")
	ErrNotFound          = errors.New("system settings resource not found")
	ErrConflict          = errors.New("system settings resource conflict")
	ErrLocked            = errors.New("system settings resource is locked")
	ErrSecretUnavailable = errors.New("secret configuration adapter unavailable")
)

type Principal struct {
	TenantID, UserID, SessionID uuid.UUID
	RequestID                   string
	IPAddress                   *netip.Addr
	UserAgent                   string
}

type PageFilter struct {
	Query, ModuleCode, Group, ValueType, Status, Sort string
	IsPublic, IsSecret                                *bool
	Page, PageSize                                    int32
}

type ConfigItem struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         *uuid.UUID      `json:"tenant_id"`
	ModuleCode       string          `json:"module_code"`
	ConfigGroup      string          `json:"config_group"`
	ConfigKey        string          `json:"config_key"`
	DisplayName      string          `json:"display_name"`
	ValueType        string          `json:"value_type"`
	Value            json.RawMessage `json:"value"`
	DefaultValue     json.RawMessage `json:"default_value"`
	IsSecret         bool            `json:"is_secret"`
	SecretConfigured bool            `json:"secret_configured"`
	SecretKeyVersion *int32          `json:"secret_key_version"`
	IsPublic         bool            `json:"is_public"`
	ValidationSchema json.RawMessage `json:"validation_schema"`
	Description      string          `json:"description"`
	SortOrder        int32           `json:"sort_order"`
	Status           string          `json:"status"`
	Version          int32           `json:"version"`
	IsLocked         bool            `json:"is_locked"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type ConfigPage struct {
	Items    []ConfigItem `json:"items"`
	Total    int64        `json:"total"`
	Page     int32        `json:"page"`
	PageSize int32        `json:"page_size"`
}

type ConfigInput struct {
	ModuleCode, ConfigGroup, ConfigKey, DisplayName, ValueType string
	Value, DefaultValue, ValidationSchema                      json.RawMessage
	SecretValue                                                string
	IsSecret, IsPublic                                         bool
	Description, Status                                        string
	SortOrder, Version                                         int32
}

type DictType struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        *uuid.UUID `json:"tenant_id"`
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	NameKey         string     `json:"name_key,omitempty"`
	Description     string     `json:"description"`
	DescriptionKey  string     `json:"description_key,omitempty"`
	Status          string     `json:"status"`
	IsSystem        bool       `json:"is_system"`
	IsLocked        bool       `json:"is_locked"`
	Visibility      string     `json:"visibility"`
	ExtensionPolicy string     `json:"extension_policy"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type DictTypePage struct {
	Items    []DictType `json:"items"`
	Total    int64      `json:"total"`
	Page     int32      `json:"page"`
	PageSize int32      `json:"page_size"`
}
type DictTypeInput struct{ Code, Name, Description, Status string }

type DictItem struct {
	ID         uuid.UUID       `json:"id"`
	DictTypeID uuid.UUID       `json:"dict_type_id"`
	TenantID   *uuid.UUID      `json:"tenant_id,omitempty"`
	ItemValue  string          `json:"item_value"`
	Label      string          `json:"label"`
	Locale     *string         `json:"locale"`
	Color      string          `json:"color"`
	CSSClass   string          `json:"css_class"`
	SortOrder  int32           `json:"sort_order"`
	IsDefault  bool            `json:"is_default"`
	Extra      json.RawMessage `json:"extra"`
	Status     string          `json:"status"`
	IsLocked   bool            `json:"is_locked"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type DictionaryOption struct {
	Value     string          `json:"value"`
	Label     string          `json:"label"`
	Color     string          `json:"color,omitempty"`
	CSSClass  string          `json:"css_class,omitempty"`
	IsDefault bool            `json:"is_default"`
	Extra     json.RawMessage `json:"extra"`
}

type ResolvedDictionary struct {
	Code            string             `json:"code"`
	Locale          string             `json:"locale"`
	ExtensionPolicy string             `json:"extension_policy"`
	Items           []DictionaryOption `json:"items"`
}

type DictItemFilter struct {
	Query, Locale, Status, Sort string
	Page, PageSize              int32
}
type DictItemPage struct {
	Items    []DictItem `json:"items"`
	Total    int64      `json:"total"`
	Page     int32      `json:"page"`
	PageSize int32      `json:"page_size"`
	Type     DictType   `json:"type"`
}
type DictItemInput struct {
	ItemValue, Label string
	Locale           *string
	Color, CSSClass  string
	SortOrder        int32
	IsDefault        bool
	Extra            json.RawMessage
	Status           string
}

type RegionFilter struct {
	Query, ParentCode, Status string
	Level                     *int16
	Limit                     int32
}

type Region struct {
	Code        string    `json:"code"`
	ParentCode  *string   `json:"parent_code"`
	Level       int16     `json:"level"`
	Name        string    `json:"name"`
	FullName    string    `json:"full_name"`
	PostalCode  string    `json:"postal_code"`
	Longitude   *float64  `json:"longitude"`
	Latitude    *float64  `json:"latitude"`
	Status      string    `json:"status"`
	HasChildren bool      `json:"has_children"`
	Version     int32     `json:"version"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RegionCreateInput struct {
	Code, ParentCode, Name, FullName, PostalCode, Status string
	Longitude, Latitude                                  *float64
}

type RegionUpdateInput struct {
	Name, FullName, PostalCode, Status string
	Longitude, Latitude                *float64
	Version                            int32
}

type Repository interface {
	ListPublicConfigs(context.Context) (map[string]json.RawMessage, error)
	ListRegions(context.Context, RegionFilter) ([]Region, error)
	CreateRegion(context.Context, Principal, RegionCreateInput) (Region, error)
	UpdateRegion(context.Context, Principal, string, RegionUpdateInput) (Region, error)
	DeleteRegion(context.Context, Principal, string) error
	ListConfigs(context.Context, uuid.UUID, PageFilter) (ConfigPage, error)
	CreateConfig(context.Context, Principal, ConfigInput, []byte, int32) (ConfigItem, error)
	UpdateConfig(context.Context, Principal, uuid.UUID, ConfigInput) (ConfigItem, error)
	RotateSecret(context.Context, Principal, uuid.UUID, int32, []byte, int32) (ConfigItem, error)
	ListDictTypes(context.Context, uuid.UUID, PageFilter) (DictTypePage, error)
	CreateDictType(context.Context, Principal, DictTypeInput) (DictType, error)
	UpdateDictType(context.Context, Principal, uuid.UUID, DictTypeInput) (DictType, error)
	ListDictItems(context.Context, uuid.UUID, uuid.UUID, DictItemFilter) (DictItemPage, error)
	CreateDictItem(context.Context, Principal, uuid.UUID, DictItemInput) (DictItem, error)
	UpdateDictItem(context.Context, Principal, uuid.UUID, DictItemInput) (DictItem, error)
	DeleteDictItem(context.Context, Principal, uuid.UUID) error
	ResolveDictionary(context.Context, *uuid.UUID, string, string, bool) (ResolvedDictionary, error)
}

type SecretSealer interface {
	Seal(plaintext []byte, aad string) ([]byte, int32, error)
}
