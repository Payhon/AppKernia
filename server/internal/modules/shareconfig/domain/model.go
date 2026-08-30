package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden = errors.New("share configuration operation forbidden")
	ErrInvalid   = errors.New("share configuration input invalid")
	ErrNotFound  = errors.New("share configuration not found")
	ErrConflict  = errors.New("share configuration conflict")
	ErrInUse     = errors.New("share configuration is in use")
)

const ProviderWechat = "wechat"

type Principal struct {
	TenantID, UserID, SessionID     uuid.UUID
	RequestID, IPAddress, UserAgent string
}

type PlatformIdentity struct {
	Enabled       bool   `json:"enabled"`
	PackageName   string `json:"package_name,omitempty"`
	Signature     string `json:"signature,omitempty"`
	BundleID      string `json:"bundle_id,omitempty"`
	UniversalLink string `json:"universal_link,omitempty"`
	BundleName    string `json:"bundle_name,omitempty"`
}

type WechatPublicConfig struct {
	Android PlatformIdentity `json:"android"`
	IOS     PlatformIdentity `json:"ios"`
	Harmony PlatformIdentity `json:"harmony"`
}

type Config struct {
	ID                  uuid.UUID       `json:"id"`
	TenantID            uuid.UUID       `json:"tenant_id"`
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	ProviderCode        string          `json:"provider_code"`
	ExternalAppID       string          `json:"external_app_id"`
	ConfigSchemaVersion int32           `json:"config_schema_version"`
	PublicConfig        json.RawMessage `json:"public_config"`
	SecretFieldNames    []string        `json:"secret_field_names"`
	HasSecret           bool            `json:"has_secret"`
	Status              string          `json:"status"`
	BindingCount        int64           `json:"binding_count"`
	LockVersion         int32           `json:"lock_version"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type ConfigInput struct {
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	ProviderCode        string          `json:"provider_code"`
	ExternalAppID       string          `json:"external_app_id"`
	ConfigSchemaVersion int32           `json:"config_schema_version"`
	PublicConfig        json.RawMessage `json:"public_config"`
	LockVersion         int32           `json:"lock_version,omitempty"`
}

type SecretInput struct {
	Values      map[string]string `json:"values"`
	LockVersion int32             `json:"lock_version"`
}

type ListFilter struct {
	Query, ProviderCode, Status string
	Page, PageSize              int32
}

type ConfigPage struct {
	Items    []Config `json:"items"`
	Page     int32    `json:"page"`
	PageSize int32    `json:"page_size"`
	Total    int64    `json:"total"`
}

type Binding struct {
	ID              uuid.UUID `json:"id"`
	AppID           uuid.UUID `json:"app_id"`
	ProviderCode    string    `json:"provider_code"`
	ShareConfigID   uuid.UUID `json:"share_config_id"`
	ShareConfigName string    `json:"share_config_name"`
	ConfigStatus    string    `json:"config_status"`
	Enabled         bool      `json:"enabled"`
	Scenes          []string  `json:"scenes"`
	ShareOrigin     string    `json:"share_origin"`
	FallbackMode    string    `json:"fallback_mode"`
	LockVersion     int32     `json:"lock_version"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type BindingInput struct {
	ShareConfigID uuid.UUID `json:"share_config_id"`
	Enabled       bool      `json:"enabled"`
	Scenes        []string  `json:"scenes"`
	ShareOrigin   string    `json:"share_origin"`
	FallbackMode  string    `json:"fallback_mode"`
	LockVersion   int32     `json:"lock_version,omitempty"`
}

type Preflight struct {
	Ready        bool     `json:"ready"`
	ProviderCode string   `json:"provider_code"`
	Platforms    []string `json:"platforms"`
	Scenes       []string `json:"scenes"`
	Issues       []string `json:"issues"`
}

type RuntimeProvider struct {
	ProviderCode string   `json:"provider_code"`
	Enabled      bool     `json:"enabled"`
	Scenes       []string `json:"scenes"`
	FallbackMode string   `json:"fallback_mode"`
}

type Repository interface {
	List(context.Context, uuid.UUID, ListFilter) (ConfigPage, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (Config, error)
	Create(context.Context, Principal, ConfigInput) (Config, error)
	Update(context.Context, Principal, uuid.UUID, ConfigInput) (Config, error)
	SetStatus(context.Context, Principal, uuid.UUID, int32, string) (Config, error)
	Delete(context.Context, Principal, uuid.UUID, int32) error
	RotateSecret(context.Context, Principal, uuid.UUID, int32, []byte, int32, []string) (Config, error)
	AppExists(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	ListBindings(context.Context, uuid.UUID, uuid.UUID) ([]Binding, error)
	UpsertBinding(context.Context, Principal, uuid.UUID, string, BindingInput) (Binding, error)
	DeleteBinding(context.Context, Principal, uuid.UUID, string, int32) error
	Runtime(context.Context, uuid.UUID) ([]RuntimeProvider, error)
}

type SecretSealer interface {
	Seal([]byte, string) ([]byte, int32, error)
}
