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

var (
	ErrFeatureDisabled = errors.New("avatar upload is disabled")
	ErrUploadInvalid   = errors.New("avatar upload is invalid")
	ErrUploadNotFound  = errors.New("avatar upload session not found")
	ErrAvatarNotFound  = errors.New("avatar not found")
	ErrObjectNotFound  = errors.New("object not found")
)

type Principal struct {
	UserID    uuid.UUID
	TenantID  uuid.UUID
	SessionID uuid.UUID
}

type ClientMetadata struct {
	RequestID string
	IPAddress *netip.Addr
	UserAgent string
}

type CreateAvatarUpload struct {
	Principal
	OriginalName string
	MediaType    string
	ExpectedSize int64
	ObjectKey    string
	ExpiresAt    time.Time
}

type AvatarUploadSession struct {
	ID           uuid.UUID
	ObjectKey    string
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
	OriginalName    string
	MediaType       string
	Extension       string
	SizeBytes       int64
	SHA256          []byte
}

type AvatarObject struct {
	FileID    uuid.UUID
	ObjectKey string
	MediaType string
	SizeBytes int64
	SHA256    []byte
	UpdatedAt time.Time
}

type AvatarCompletion struct {
	FileID    uuid.UUID
	ObjectKey string
}

type Repository interface {
	CreateAvatarUpload(context.Context, CreateAvatarUpload) (AvatarUploadSession, error)
	GetAvatarUpload(context.Context, Principal, uuid.UUID) (AvatarUploadSession, error)
	CompleteAvatarUpload(context.Context, CompleteAvatarUpload) (AvatarCompletion, error)
	GetAvatarObject(context.Context, Principal) (AvatarObject, error)
}

type ObjectStore interface {
	Put(context.Context, string, []byte) error
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}
