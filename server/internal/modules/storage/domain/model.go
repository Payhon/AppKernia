package domain

import (
	"context"
	"errors"
	"io"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

const MaxAvatarBytes int64 = 5 * 1024 * 1024

const MaxFileBytes int64 = 100 * 1024 * 1024

var (
	ErrFeatureDisabled = errors.New("avatar upload is disabled")
	ErrUploadInvalid   = errors.New("avatar upload is invalid")
	ErrUploadNotFound  = errors.New("avatar upload session not found")
	ErrAvatarNotFound  = errors.New("avatar not found")
	ErrObjectNotFound  = errors.New("object not found")
	ErrStorageConfig   = errors.New("object storage configuration is unavailable")
)

type Principal struct {
	UserID    uuid.UUID
	TenantID  uuid.UUID
	SessionID uuid.UUID
}

type ClientMetadata struct {
	RequestID   string
	IPAddress   *netip.Addr
	UserAgent   string
	HTTPMethod  string
	RequestPath string
}

type CreateAvatarUpload struct {
	Principal
	OriginalName string
	MediaType    string
	ExpectedSize int64
	ObjectKey    string
	Provider     string
	Bucket       string
	ExpiresAt    time.Time
}

type AvatarUploadSession struct {
	ID           uuid.UUID
	ObjectKey    string
	Provider     string
	Bucket       string
	OriginalName string
	MediaType    string
	ExpectedSize int64
	ExpiresAt    time.Time
}

type CompleteAvatarUpload struct {
	Principal
	ClientMetadata
	UploadSessionID uuid.UUID
	ObjectKey       string
	Provider        string
	Bucket          string
	OriginalName    string
	MediaType       string
	Extension       string
	SizeBytes       int64
	SHA256          []byte
}

type AvatarObject struct {
	FileID    uuid.UUID
	ObjectKey string
	Provider  string
	Bucket    string
	MediaType string
	SizeBytes int64
	SHA256    []byte
	UpdatedAt time.Time
}

type AvatarCompletion struct {
	FileID    uuid.UUID
	ObjectKey string
	Provider  string
	Bucket    string
}

type UploadPolicy struct {
	Provider          string   `json:"provider"`
	Bucket            string   `json:"-"`
	PathPrefix        string   `json:"-"`
	MaxImageBytes     int64    `json:"max_image_bytes"`
	MaxFileBytes      int64    `json:"max_file_bytes"`
	ImageMediaTypes   []string `json:"image_media_types"`
	FileMediaTypes    []string `json:"file_media_types"`
	ConfigurationSafe bool     `json:"configuration_safe"`
}

type ObjectRef struct {
	TenantID uuid.UUID
	Provider string
	Bucket   string
	Key      string
}

type Repository interface {
	CreateAvatarUpload(context.Context, CreateAvatarUpload) (AvatarUploadSession, error)
	GetAvatarUpload(context.Context, Principal, uuid.UUID) (AvatarUploadSession, error)
	CompleteAvatarUpload(context.Context, CompleteAvatarUpload) (AvatarCompletion, error)
	GetAvatarObject(context.Context, Principal) (AvatarObject, error)
}

type ObjectStore interface {
	ResolvePolicy(context.Context, uuid.UUID) (UploadPolicy, error)
	Put(context.Context, ObjectRef, []byte) error
	Open(context.Context, ObjectRef) (io.ReadCloser, error)
	Delete(context.Context, ObjectRef) error
}
