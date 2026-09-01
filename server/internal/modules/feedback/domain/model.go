package domain

import (
	"context"
	"errors"
	storage "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	"time"
)

var (
	ErrInvalid   = errors.New("invalid feedback")
	ErrForbidden = errors.New("feedback forbidden")
	ErrNotFound  = errors.New("feedback not found")
	ErrConflict  = errors.New("feedback conflict")
	ErrStorage   = errors.New("feedback storage unavailable")
)

const MaxImageBytes int64 = 5 * 1024 * 1024

// Status is a protocol state, never a configurable business dictionary.
func ValidStatus(s string) bool { return s == "pending" || s == "processing" || s == "resolved" }

type Scope struct {
	TenantID, AppID, UserID, SessionID uuid.UUID
	Admin                              bool
	RequestID                          string
}
type Input struct {
	Description string      `json:"description"`
	Contact     string      `json:"contact"`
	Platform    string      `json:"platform"`
	AppVersion  string      `json:"app_version"`
	FileIDs     []uuid.UUID `json:"file_ids"`
}
type ReplyInput struct {
	Body        string `json:"body"`
	Status      string `json:"status"`
	LockVersion int32  `json:"lock_version"`
}
type StatusInput struct {
	Status      string `json:"status"`
	LockVersion int32  `json:"lock_version"`
}
type Filter struct {
	Query, Status  string
	From, To       *time.Time
	Page, PageSize int32
}
type Feedback struct {
	ID          uuid.UUID    `json:"id"`
	UserID      uuid.UUID    `json:"user_id"`
	Description string       `json:"description"`
	Contact     string       `json:"contact"`
	Platform    string       `json:"platform"`
	AppVersion  string       `json:"app_version"`
	Status      string       `json:"status"`
	LockVersion int32        `json:"lock_version"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Attachments []Attachment `json:"attachments"`
	Replies     []Reply      `json:"replies"`
	Events      []Event      `json:"events"`
}
type Attachment struct {
	FileID    uuid.UUID `json:"file_id"`
	MediaType string    `json:"media_type"`
	SizeBytes int64     `json:"size_bytes"`
}
type Reply struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
type Event struct {
	ID        uuid.UUID `json:"id"`
	From      string    `json:"from_status"`
	To        string    `json:"to_status"`
	CreatedAt time.Time `json:"created_at"`
}
type Page struct {
	Items    []Feedback `json:"items"`
	Total    int64      `json:"total"`
	Page     int32      `json:"page"`
	PageSize int32      `json:"page_size"`
}
type UploadInput struct {
	OriginalName string `json:"original_name"`
	MediaType    string `json:"media_type"`
	SizeBytes    int64  `json:"size_bytes"`
}
type Upload struct {
	ID              uuid.UUID         `json:"id"`
	UploadURL       string            `json:"upload_url"`
	ExpiresAt       time.Time         `json:"expires_at"`
	FileID          *uuid.UUID        `json:"file_id"`
	Status          string            `json:"status"`
	Object          storage.ObjectRef `json:"-"`
	Name, MediaType string            `json:"-"`
	Size            int64             `json:"-"`
}
type File struct {
	Attachment
	Object storage.ObjectRef
	SHA256 []byte
}
type Repository interface {
	Transact(context.Context, func(Repository) error) error
	CheckScope(context.Context, Scope) error
	List(context.Context, Scope, Filter) (Page, error)
	Get(context.Context, Scope, uuid.UUID, bool) (Feedback, error)
	FindRequest(context.Context, Scope, uuid.UUID, []byte) (uuid.UUID, error)
	Create(context.Context, Scope, Input, uuid.UUID, []byte) (uuid.UUID, error)
	Attach(context.Context, Scope, uuid.UUID, []uuid.UUID) error
	Change(context.Context, Scope, uuid.UUID, string, int32) error
	FindReply(context.Context, Scope, uuid.UUID, uuid.UUID, []byte) (bool, error)
	Reply(context.Context, Scope, uuid.UUID, ReplyInput, uuid.UUID, []byte) error
	Audit(context.Context, Scope, uuid.UUID, string) error
	CreateUpload(context.Context, Scope, Upload) (Upload, error)
	GetUpload(context.Context, Scope, uuid.UUID, bool) (Upload, error)
	CompleteUpload(context.Context, Scope, Upload, []byte) (uuid.UUID, error)
	File(context.Context, Scope, uuid.UUID, uuid.UUID) (File, error)
	CancelUpload(context.Context, Scope, uuid.UUID) error
}
