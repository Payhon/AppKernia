package domain

import (
	"context"
	"errors"
	"time"

	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
)

const (
	MaxFileBytes int64 = storagedomain.MaxFileBytes
	PartSize     int64 = 5 * 1024 * 1024
)

var (
	ErrForbidden        = errors.New("file operation forbidden")
	ErrInvalid          = errors.New("file operation invalid")
	ErrNotFound         = errors.New("file not found")
	ErrUploadNotFound   = errors.New("upload session not found")
	ErrUploadIncomplete = errors.New("upload is incomplete")
	ErrScanBlocked      = errors.New("file scan gate blocked access")
	ErrFileInUse        = errors.New("file is in use")
	ErrFeatureDisabled  = errors.New("file storage is disabled")
	ErrStorageConfig    = storagedomain.ErrStorageConfig
)

type Principal struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	SessionID uuid.UUID
	RequestID string
	IPAddress string
	UserAgent string
}

type FileFilter struct {
	Query       string
	Status      string
	ScanStatus  string
	MediaType   string
	Provider    string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Page        int32
	PageSize    int32
}

type File struct {
	ID           uuid.UUID  `json:"id"`
	OriginalName string     `json:"original_name"`
	MediaType    string     `json:"media_type"`
	Extension    string     `json:"extension"`
	SizeBytes    int64      `json:"size_bytes"`
	Status       string     `json:"status"`
	ScanStatus   string     `json:"scan_status"`
	Provider     string     `json:"provider"`
	UsageCount   int64      `json:"usage_count"`
	OwnerUserID  *uuid.UUID `json:"owner_user_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ObjectKey    string     `json:"-"`
	Bucket       string     `json:"-"`
	SHA256       []byte     `json:"-"`
}

type FilePage struct {
	Items    []File `json:"items"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
	Total    int64  `json:"total"`
}

type Usage struct {
	ID         uuid.UUID `json:"id"`
	ModuleCode string    `json:"module_code"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	FieldName  string    `json:"field_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type UploadSession struct {
	ID            uuid.UUID `json:"id"`
	OriginalName  string    `json:"original_name"`
	MediaType     string    `json:"media_type"`
	ExpectedSize  int64     `json:"expected_size"`
	PartSize      int64     `json:"part_size"`
	Status        string    `json:"status"`
	Provider      string    `json:"provider"`
	ObjectKey     string    `json:"-"`
	Bucket        string    `json:"-"`
	ExpiresAt     time.Time `json:"expires_at"`
	UploadedParts []Part    `json:"uploaded_parts"`
}

type Part struct {
	PartNumber int32  `json:"part_number"`
	SizeBytes  int64  `json:"size_bytes"`
	ETag       string `json:"etag"`
	Checksum   []byte `json:"-"`
}

type CreateUpload struct {
	Principal
	OriginalName string
	MediaType    string
	ExpectedSize int64
	ObjectKey    string
	Provider     string
	Bucket       string
	ExpiresAt    time.Time
}

type CompleteUpload struct {
	Principal
	UploadID   uuid.UUID
	ObjectKey  string
	Provider   string
	Bucket     string
	MediaType  string
	Extension  string
	SizeBytes  int64
	SHA256     []byte
	ScanStatus string
}

type Repository interface {
	CreateUpload(context.Context, CreateUpload) (UploadSession, error)
	GetUpload(context.Context, uuid.UUID, uuid.UUID) (UploadSession, error)
	UpsertPart(context.Context, uuid.UUID, uuid.UUID, Part) error
	AbortUpload(context.Context, Principal, uuid.UUID) (UploadSession, error)
	CompleteUpload(context.Context, CompleteUpload) (File, error)
	ListFiles(context.Context, uuid.UUID, FileFilter) (FilePage, error)
	GetFile(context.Context, uuid.UUID, uuid.UUID) (File, error)
	ListUsages(context.Context, uuid.UUID, uuid.UUID) ([]Usage, error)
	DeleteFile(context.Context, Principal, uuid.UUID) (File, error)
}

type UploadPolicy = storagedomain.UploadPolicy
type ObjectRef = storagedomain.ObjectRef
type ObjectStore = storagedomain.ObjectStore

var _ ObjectStore = (storagedomain.ObjectStore)(nil)
