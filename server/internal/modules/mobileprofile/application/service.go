package application

import (
	"context"
	"encoding/base64"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	profile "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	"github.com/google/uuid"
)

var ErrInvalidPreferences = errors.New("invalid mobile preferences")
var ErrInvalidRelease = errors.New("invalid mobile release")
var ErrForbidden = errors.New("mobile release permission denied")

type Service struct {
	auth       *application.AuthService
	repository profile.Repository
	releases   profile.ReleaseRepository
}

func NewService(auth *application.AuthService, repository profile.Repository, releases profile.ReleaseRepository) *Service {
	return &Service{auth: auth, repository: repository, releases: releases}
}

func (service *Service) PublicRelease(ctx context.Context, platform string) (profile.Release, error) {
	if platform != "android" && platform != "ios" && platform != "harmony" {
		return profile.Release{}, ErrInvalidRelease
	}
	return service.releases.ActiveRelease(ctx, platform)
}
func (service *Service) AdminReleases(ctx context.Context, token string) ([]profile.Release, error) {
	if _, err := service.admin(ctx, token, "mobile.release.read"); err != nil {
		return nil, err
	}
	return service.releases.ListReleases(ctx)
}
func (service *Service) CreateRelease(ctx context.Context, token, requestID string, release profile.Release) (profile.Release, error) {
	actor, err := service.admin(ctx, token, "mobile.release.create")
	if err != nil {
		return profile.Release{}, err
	}
	if err = validRelease(release, false); err != nil {
		return profile.Release{}, err
	}
	return service.releases.CreateRelease(ctx, release, actor.User.ID, requestID)
}
func (service *Service) UpdateRelease(ctx context.Context, token, requestID string, release profile.Release) (profile.Release, error) {
	actor, err := service.admin(ctx, token, "mobile.release.update")
	if err != nil {
		return profile.Release{}, err
	}
	if err = validRelease(release, true); err != nil {
		return profile.Release{}, err
	}
	return service.releases.UpdateRelease(ctx, release, actor.User.ID, requestID)
}
func (service *Service) admin(ctx context.Context, token, permission string) (domain.AuthenticatedContext, error) {
	p, err := service.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return domain.AuthenticatedContext{}, err
	}
	if !slices.Contains(p.Permissions, permission) {
		return domain.AuthenticatedContext{}, ErrForbidden
	}
	return p, nil
}
func validRelease(x profile.Release, update bool) error {
	current, currentOK := parseSemver(x.CurrentVersion)
	minimum, minimumOK := parseSemver(x.MinimumVersion)
	if (update && (x.ID == uuid.Nil || x.LockVersion < 1)) ||
		(x.Platform != "android" && x.Platform != "ios" && x.Platform != "harmony") ||
		!currentOK || !minimumOK || compareSemver(minimum, current) > 0 ||
		strings.TrimSpace(x.ReleaseNotes["zh-CN"]) == "" || strings.TrimSpace(x.ReleaseNotes["en-US"]) == "" {
		return ErrInvalidRelease
	}
	if x.Active && (x.UpgradeURL == nil || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(*x.UpgradeURL)), "https://")) {
		return ErrInvalidRelease
	}
	if x.UpgradeURL != nil && strings.TrimSpace(*x.UpgradeURL) != "" && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(*x.UpgradeURL)), "https://") {
		return ErrInvalidRelease
	}
	return nil
}

type semver [3]uint64

func parseSemver(raw string) (semver, bool) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	var out semver
	for i, part := range parts {
		if part == "" {
			return semver{}, false
		}
		value, err := strconv.ParseUint(part, 10, 63)
		if err != nil {
			return semver{}, false
		}
		out[i] = value
	}
	return out, true
}
func compareSemver(left, right semver) int {
	for i := 0; i < 3; i++ {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	return 0
}

func (service *Service) authenticate(ctx context.Context, token string) (domain.AuthenticatedContext, error) {
	return service.auth.Authenticate(ctx, token, "ak-mobile")
}
func (service *Service) Preferences(ctx context.Context, token string) (profile.Preferences, error) {
	principal, err := service.authenticate(ctx, token)
	if err != nil {
		return profile.Preferences{}, err
	}
	return service.repository.GetPreferences(ctx, principal.User.ID)
}
func (service *Service) UpdatePreferences(ctx context.Context, token, requestID string, locale, appearance *string, notifications map[string]bool) (profile.Preferences, error) {
	if locale == nil && appearance == nil && notifications == nil {
		return profile.Preferences{}, ErrInvalidPreferences
	}
	if locale != nil && *locale != "zh-CN" && *locale != "en-US" {
		return profile.Preferences{}, ErrInvalidPreferences
	}
	if appearance != nil && *appearance != "system" && *appearance != "light" && *appearance != "dark" {
		return profile.Preferences{}, ErrInvalidPreferences
	}
	for key := range notifications {
		if key != "in_app" && key != "push" && key != "email" {
			return profile.Preferences{}, ErrInvalidPreferences
		}
	}
	principal, err := service.authenticate(ctx, token)
	if err != nil {
		return profile.Preferences{}, err
	}
	if locale != nil {
		normalized := strings.TrimSpace(*locale)
		locale = &normalized
	}
	return service.repository.UpdatePreferences(ctx, profile.PreferenceUpdate{
		UserID: principal.User.ID, TenantID: principal.Tenant.ID, SessionID: principal.SessionID, RequestID: requestID,
		Locale: locale, Appearance: appearance, NotificationPreferences: notifications,
	})
}
func (service *Service) UnreadCount(ctx context.Context, token string) (int64, error) {
	principal, err := service.authenticate(ctx, token)
	if err != nil {
		return 0, err
	}
	return service.repository.UnreadCount(ctx, principal.User.ID, principal.Tenant.ID)
}
func (service *Service) LoginEvents(ctx context.Context, token string) ([]profile.LoginEvent, error) {
	principal, err := service.authenticate(ctx, token)
	if err != nil {
		return nil, err
	}
	return service.repository.LoginEvents(ctx, principal.User.ID)
}
func (service *Service) SecurityEvents(ctx context.Context, token string) ([]profile.SecurityEvent, error) {
	principal, err := service.authenticate(ctx, token)
	if err != nil {
		return nil, err
	}
	return service.repository.SecurityEvents(ctx, principal.User.ID)
}
func (service *Service) Notifications(ctx context.Context, token, cursor string, limit int) (profile.NotificationPage, error) {
	if limit < 1 || limit > 100 {
		return profile.NotificationPage{}, ErrInvalidPreferences
	}
	if cursor != "" {
		if _, err := base64.RawURLEncoding.DecodeString(cursor); err != nil {
			return profile.NotificationPage{}, ErrInvalidPreferences
		}
	}
	p, err := service.authenticate(ctx, token)
	if err != nil {
		return profile.NotificationPage{}, err
	}
	return service.repository.Notifications(ctx, p.User.ID, p.Tenant.ID, cursor, limit)
}
func (service *Service) MarkNotificationRead(ctx context.Context, token, requestID, id string) error {
	messageID, err := uuid.Parse(id)
	if err != nil {
		return ErrInvalidPreferences
	}
	p, err := service.authenticate(ctx, token)
	if err != nil {
		return err
	}
	return service.repository.MarkNotificationRead(ctx, p.User.ID, p.Tenant.ID, p.SessionID, messageID, requestID)
}
