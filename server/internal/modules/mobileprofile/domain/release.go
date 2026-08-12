package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrReleaseNotFound = errors.New("mobile release not found")
var ErrReleaseConflict = errors.New("mobile release conflict")
var ErrReleaseFrozen = errors.New("published mobile release is frozen")
var ErrReleaseDeleteForbidden = errors.New("published mobile release cannot be deleted")
var ErrReleaseFileInvalid = errors.New("mobile release file is invalid")
var ErrReleaseVersionNotIncreasing = errors.New("mobile release version must increase")
var ErrReleasePackageTypeUnsupported = errors.New("mobile release package type is unsupported for this application")
var ErrReleaseDeliveryModeUnsupported = errors.New("mobile release delivery mode is unsupported for this platform")

type StoreListing struct {
	ID       uuid.UUID
	Name     string
	Scheme   string
	Priority int32
}

type Release struct {
	ID                   uuid.UUID         `json:"id"`
	TenantID             uuid.UUID         `json:"tenant_id"`
	AppID                uuid.UUID         `json:"app_id"`
	PackageType          string            `json:"package_type"`
	Platforms            []string          `json:"platforms"`
	Version              string            `json:"version"`
	MinimumNativeVersion *string           `json:"minimum_native_version,omitempty"`
	Titles               map[string]string `json:"titles"`
	Contents             map[string]string `json:"contents"`
	PackageFileID        *uuid.UUID        `json:"package_file_id,omitempty"`
	ExternalURL          *string           `json:"external_url,omitempty"`
	DownloadURL          *string           `json:"download_url,omitempty"`
	StoreListingIDs      []uuid.UUID       `json:"store_listing_ids"`
	StoreList            []StoreListing    `json:"-"`
	CreateEnv            string            `json:"create_env"`
	IsSilently           bool              `json:"is_silently"`
	IsMandatory          bool              `json:"is_mandatory"`
	PublishStatus        string            `json:"publish_status"`
	PublishedPlatforms   []string          `json:"published_platforms"`
	EverPublishedAt      *time.Time        `json:"ever_published_at,omitempty"`
	LastPublishedAt      *time.Time        `json:"last_published_at,omitempty"`
	UnpublishedAt        *time.Time        `json:"unpublished_at,omitempty"`
	LockVersion          int32             `json:"lock_version"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`

	// Compatibility projection for /admin-api/v1/mobile/releases and existing
	// Mobile clients. New callers should use the fields above.
	Platform       string            `json:"platform"`
	CurrentVersion string            `json:"current_version"`
	MinimumVersion string            `json:"minimum_version"`
	UpgradeURL     *string           `json:"upgrade_url"`
	ReleaseNotes   map[string]string `json:"release_notes"`
	Active         bool              `json:"active"`
}

type ReleaseFilter struct {
	Query         string
	PackageType   string
	Platform      string
	PublishStatus string
	Page          int32
	PageSize      int32
}

type ReleasePage struct {
	Items    []Release `json:"items"`
	Page     int32     `json:"page"`
	PageSize int32     `json:"page_size"`
	Total    int64     `json:"total"`
}

type PackageFile struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	OriginalName string
	MediaType    string
	SizeBytes    int64
	Provider     string
	Bucket       string
	ObjectKey    string
}

type ReleaseRepository interface {
	// Compatibility operations.
	ActiveRelease(context.Context, uuid.UUID, string) (Release, error)
	ListReleases(context.Context, uuid.UUID) ([]Release, error)
	CreateRelease(context.Context, uuid.UUID, Release, uuid.UUID, string) (Release, error)
	UpdateRelease(context.Context, uuid.UUID, Release, uuid.UUID, string) (Release, error)

	ActivePackageRelease(context.Context, uuid.UUID, string, string) (Release, error)
	ApplicationType(context.Context, uuid.UUID) (string, error)
	ListReleasePage(context.Context, uuid.UUID, uuid.UUID, ReleaseFilter) (ReleasePage, error)
	GetRelease(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Release, error)
	CreateDraft(context.Context, uuid.UUID, uuid.UUID, Release, uuid.UUID, string) (Release, error)
	UpdateDraft(context.Context, uuid.UUID, uuid.UUID, Release, uuid.UUID, string) (Release, error)
	Publish(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int32, uuid.UUID, string) (Release, error)
	Unpublish(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int32, uuid.UUID, string) (Release, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID, uuid.UUID, string) error
	PublishedPackageFile(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (PackageFile, error)
}
