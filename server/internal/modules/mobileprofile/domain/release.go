package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrReleaseNotFound = errors.New("mobile release not found")
var ErrReleaseConflict = errors.New("mobile release conflict")

type Release struct {
	ID             uuid.UUID         `json:"id"`
	AppID          uuid.UUID         `json:"app_id"`
	Platform       string            `json:"platform"`
	CurrentVersion string            `json:"current_version"`
	MinimumVersion string            `json:"minimum_version"`
	UpgradeURL     *string           `json:"upgrade_url"`
	ReleaseNotes   map[string]string `json:"release_notes"`
	Active         bool              `json:"active"`
	LockVersion    int32             `json:"lock_version"`
	UpdatedAt      time.Time         `json:"updated_at"`
}
type ReleaseRepository interface {
	ActiveRelease(context.Context, uuid.UUID, string) (Release, error)
	ListReleases(context.Context, uuid.UUID) ([]Release, error)
	CreateRelease(context.Context, uuid.UUID, Release, uuid.UUID, string) (Release, error)
	UpdateRelease(context.Context, uuid.UUID, Release, uuid.UUID, string) (Release, error)
}
