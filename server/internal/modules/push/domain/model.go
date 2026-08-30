package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden   = errors.New("push operation forbidden")
	ErrInvalid     = errors.New("push input invalid")
	ErrNotFound    = errors.New("push resource not found")
	ErrConflict    = errors.New("push configuration conflict")
	ErrUnavailable = errors.New("push capability unavailable")
)

const (
	ProviderAPNS          = "apns"
	ProviderFCM           = "fcm"
	ProviderHuaweiAndroid = "huawei_android"
	ProviderHonor         = "honor"
	ProviderXiaomi        = "xiaomi"
	ProviderOPPO          = "oppo"
	ProviderVivo          = "vivo"
	ProviderMeizu         = "meizu"
	ProviderHarmony       = "harmony"

	CategoryServiceSecurity = "service_security"
	CategoryNewsOperations  = "news_operations"
)

var Providers = []string{
	ProviderAPNS,
	ProviderFCM,
	ProviderHuaweiAndroid,
	ProviderHonor,
	ProviderXiaomi,
	ProviderOPPO,
	ProviderVivo,
	ProviderMeizu,
	ProviderHarmony,
}

type Principal struct {
	TenantID, AppID, UserID, SessionID, DeviceID uuid.UUID
	RequestID, IPAddress, UserAgent              string
}

type DeviceInput struct {
	Provider     string `json:"provider"`
	Platform     string `json:"platform"`
	BuildVariant string `json:"build_variant"`
	Token        string `json:"token"`
	Locale       string `json:"locale"`
	SDKVersion   string `json:"sdk_version"`
	AppVersion   string `json:"app_version"`
}

type Device struct {
	ID             uuid.UUID  `json:"id"`
	Provider       string     `json:"provider"`
	Platform       string     `json:"platform"`
	BuildVariant   string     `json:"build_variant"`
	Locale         string     `json:"locale"`
	SDKVersion     string     `json:"sdk_version"`
	AppVersion     string     `json:"app_version"`
	Status         string     `json:"status"`
	RegisteredAt   time.Time  `json:"registered_at"`
	TokenUpdatedAt time.Time  `json:"token_updated_at"`
	InvalidatedAt  *time.Time `json:"invalidated_at,omitempty"`
}

type ProviderCatalogItem struct {
	Provider            string   `json:"provider"`
	Platforms           []string `json:"platforms"`
	BuildVariants       []string `json:"build_variants"`
	PublicFields        []string `json:"public_fields"`
	SecretFields        []string `json:"secret_fields"`
	SupportsPreflight   bool     `json:"supports_preflight"`
	SupportsTest        bool     `json:"supports_test"`
	ConfigSchemaVersion int32    `json:"config_schema_version"`
}

type ProviderConfig struct {
	ID                    uuid.UUID       `json:"id"`
	AppID                 uuid.UUID       `json:"app_id"`
	Environment           string          `json:"environment"`
	Provider              string          `json:"provider"`
	ConfigSchemaVersion   int32           `json:"config_schema_version"`
	PublicConfig          json.RawMessage `json:"public_config"`
	SecretFieldNames      []string        `json:"secret_field_names"`
	HasSecret             bool            `json:"has_secret"`
	CredentialFingerprint string          `json:"credential_fingerprint,omitempty"`
	Status                string          `json:"status"`
	LastPreflightAt       *time.Time      `json:"last_preflight_at,omitempty"`
	LastPreflightStatus   string          `json:"last_preflight_status,omitempty"`
	LastPreflightIssues   []string        `json:"last_preflight_issues"`
	LockVersion           int32           `json:"lock_version"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

type ProviderConfigInput struct {
	Environment         string          `json:"environment"`
	Provider            string          `json:"provider"`
	ConfigSchemaVersion int32           `json:"config_schema_version"`
	PublicConfig        json.RawMessage `json:"public_config"`
	LockVersion         int32           `json:"lock_version,omitempty"`
}

type SecretInput struct {
	Values      map[string]string `json:"values"`
	LockVersion int32             `json:"lock_version"`
}

type Preflight struct {
	Ready       bool      `json:"ready"`
	Provider    string    `json:"provider"`
	Environment string    `json:"environment"`
	Issues      []string  `json:"issues"`
	CheckedAt   time.Time `json:"checked_at"`
}

type TestInput struct {
	PushDeviceID uuid.UUID `json:"push_device_id"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
}

type TestDelivery struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

type DeliverySummaryItem struct {
	Provider    string `json:"provider"`
	Category    string `json:"category"`
	Result      string `json:"result"`
	Count       int64  `json:"count"`
	OpenedCount int64  `json:"opened_count"`
}

type RuntimeCapability struct {
	Enabled       bool     `json:"enabled"`
	Environment   string   `json:"environment"`
	Providers     []string `json:"providers"`
	BuildVariants []string `json:"build_variants"`
}

type Repository interface {
	SessionDevice(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (uuid.UUID, error)
	HasCurrentLegalConsent(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	CurrentDevice(context.Context, Principal) (*Device, error)
	UpsertDevice(context.Context, Principal, DeviceInput, []byte, []byte, int32) (Device, error)
	DisableDevice(context.Context, Principal, uuid.UUID) error
	MarkOpened(context.Context, Principal, uuid.UUID) error

	ListConfigs(context.Context, uuid.UUID, uuid.UUID, string) ([]ProviderConfig, error)
	GetConfig(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (ProviderConfig, error)
	UpsertConfig(context.Context, Principal, ProviderConfigInput) (ProviderConfig, error)
	RotateSecret(context.Context, Principal, uuid.UUID, int32, []byte, int32, []string, string) (ProviderConfig, error)
	SetStatus(context.Context, Principal, uuid.UUID, int32, string) (ProviderConfig, error)
	RecordPreflight(context.Context, Principal, uuid.UUID, int32, Preflight) (ProviderConfig, error)
	RuntimeCapability(context.Context, uuid.UUID, string) (RuntimeCapability, error)
	TestDevice(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Device, error)
	ListTestDevices(context.Context, uuid.UUID, uuid.UUID, string) ([]Device, error)
	DeliverySummary(context.Context, uuid.UUID, uuid.UUID) ([]DeliverySummaryItem, error)
	QueueTestDelivery(context.Context, Principal, uuid.UUID, uuid.UUID, uuid.UUID, []byte, int32) error
}

type SecretSealer interface {
	Seal([]byte, string) ([]byte, int32, error)
}

type SendPayload struct {
	SchemaVersion int               `json:"schema_version"`
	DeliveryID    uuid.UUID         `json:"delivery_id"`
	MessageID     uuid.UUID         `json:"message_id"`
	Title         string            `json:"title"`
	Body          string            `json:"body"`
	Category      string            `json:"category"`
	TTLSeconds    int               `json:"ttl_seconds"`
	CollapseKey   string            `json:"collapse_key,omitempty"`
	RouteKey      string            `json:"route_key,omitempty"`
	RouteParams   map[string]string `json:"route_params,omitempty"`
}

type SendResult struct {
	ProviderMessageID string
	Class             string
	RetryAfter        time.Duration
	ErrorCode         string
	SafeSummary       string
}

type Sender interface {
	Send(context.Context, uuid.UUID, uuid.UUID, string, string, SendPayload) SendResult
}

type Preflighter interface {
	Preflight(context.Context, uuid.UUID, uuid.UUID, string) []string
}
