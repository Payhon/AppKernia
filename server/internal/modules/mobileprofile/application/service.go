package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	profile "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	storage "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
)

var ErrInvalidPreferences = errors.New("invalid mobile preferences")
var ErrInvalidRelease = errors.New("invalid mobile release")
var ErrForbidden = errors.New("mobile release permission denied")

type Service struct {
	auth       *application.AuthService
	repository profile.Repository
	releases   profile.ReleaseRepository
	objects    storage.ObjectStore
	signingKey []byte
	clock      func() time.Time
}

type Option func(*Service)

func WithPackageDownloads(objects storage.ObjectStore, signingKey []byte) Option {
	return func(service *Service) {
		service.objects = objects
		service.signingKey = append([]byte(nil), signingKey...)
	}
}

func NewService(auth *application.AuthService, repository profile.Repository, releases profile.ReleaseRepository, options ...Option) *Service {
	service := &Service{auth: auth, repository: repository, releases: releases, clock: time.Now}
	for _, option := range options {
		option(service)
	}
	return service
}

func (service *Service) PublicRelease(ctx context.Context, appID uuid.UUID, platform string) (profile.Release, error) {
	return service.PublicPackageRelease(ctx, appID, platform, "native_app")
}
func (service *Service) PublicPackageRelease(ctx context.Context, appID uuid.UUID, platform, packageType string) (profile.Release, error) {
	if appID == uuid.Nil || !validPlatform(platform) || (packageType != "native_app" && packageType != "wgt") {
		return profile.Release{}, ErrInvalidRelease
	}
	appType, err := service.releases.ApplicationType(ctx, appID)
	if err != nil {
		return profile.Release{}, err
	}
	if err = validReleaseCapabilities(appType, profile.Release{PackageType: packageType, Platforms: []string{platform}}); err != nil {
		return profile.Release{}, err
	}
	release, err := service.releases.ActivePackageRelease(ctx, appID, packageType, platform)
	if err != nil {
		return profile.Release{}, err
	}
	if err = validReleaseCapabilities(appType, release); err != nil {
		return profile.Release{}, err
	}
	return release, nil
}
func (service *Service) AdminReleases(ctx context.Context, token string, appID uuid.UUID) ([]profile.Release, error) {
	actor, err := service.admin(ctx, token, "mobile.release.read")
	if err != nil {
		return nil, err
	}
	if appID == uuid.Nil {
		return nil, ErrInvalidRelease
	}
	page, err := service.releases.ListReleasePage(ctx, actor.Tenant.ID, appID, profile.ReleaseFilter{Page: 1, PageSize: 100})
	return page.Items, err
}
func (service *Service) AdminReleasePage(ctx context.Context, token string, appID uuid.UUID, filter profile.ReleaseFilter) (profile.ReleasePage, error) {
	actor, err := service.admin(ctx, token, "mobile.release.read")
	if err != nil {
		return profile.ReleasePage{}, err
	}
	filter.Query = strings.TrimSpace(filter.Query)
	filter.PackageType = strings.TrimSpace(filter.PackageType)
	filter.Platform = strings.TrimSpace(filter.Platform)
	filter.PublishStatus = strings.TrimSpace(filter.PublishStatus)
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	if appID == uuid.Nil || len([]rune(filter.Query)) > 160 || filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 ||
		(filter.PackageType != "" && filter.PackageType != "native_app" && filter.PackageType != "wgt") ||
		(filter.Platform != "" && !validPlatform(filter.Platform)) ||
		(filter.PublishStatus != "" && !slices.Contains([]string{"draft", "online", "partial", "offline"}, filter.PublishStatus)) {
		return profile.ReleasePage{}, ErrInvalidRelease
	}
	return service.releases.ListReleasePage(ctx, actor.Tenant.ID, appID, filter)
}
func (service *Service) AdminRelease(ctx context.Context, token string, appID, id uuid.UUID) (profile.Release, error) {
	actor, err := service.admin(ctx, token, "mobile.release.read")
	if err != nil {
		return profile.Release{}, err
	}
	if appID == uuid.Nil || id == uuid.Nil {
		return profile.Release{}, ErrInvalidRelease
	}
	return service.releases.GetRelease(ctx, actor.Tenant.ID, appID, id)
}
func (service *Service) CreateRelease(ctx context.Context, token, requestID string, appID uuid.UUID, release profile.Release) (profile.Release, error) {
	actor, err := service.admin(ctx, token, "mobile.release.create")
	if err != nil {
		return profile.Release{}, err
	}
	if appID == uuid.Nil {
		return profile.Release{}, ErrInvalidRelease
	}
	release = normalizeRelease(release)
	if err = validRelease(release, false); err != nil {
		return profile.Release{}, err
	}
	if err = service.validateReleaseCapabilities(ctx, appID, release); err != nil {
		return profile.Release{}, err
	}
	if release.Active {
		if err = service.validatePackageArchive(ctx, actor.Tenant.ID, release); err != nil {
			return profile.Release{}, err
		}
	}
	created, err := service.releases.CreateDraft(ctx, actor.Tenant.ID, appID, release, actor.User.ID, requestID)
	if err != nil || !release.Active {
		return created, err
	}
	published, publishErr := service.releases.Publish(ctx, actor.Tenant.ID, appID, created.ID, created.LockVersion, actor.User.ID, requestID)
	if publishErr == nil {
		return published, nil
	}
	// publish_now is one user command. Compensate the just-created unpublished
	// draft so a version conflict or pending manifest does not leave a hidden
	// duplicate that a retry would multiply.
	if cleanupErr := service.releases.Delete(ctx, actor.Tenant.ID, appID, []uuid.UUID{created.ID}, actor.User.ID, requestID); cleanupErr != nil {
		return profile.Release{}, fmt.Errorf("publish newly created release: %w (draft cleanup: %v)", publishErr, cleanupErr)
	}
	return profile.Release{}, publishErr
}
func (service *Service) UpdateRelease(ctx context.Context, token, requestID string, appID uuid.UUID, release profile.Release) (profile.Release, error) {
	actor, err := service.admin(ctx, token, "mobile.release.update")
	if err != nil {
		return profile.Release{}, err
	}
	if appID == uuid.Nil {
		return profile.Release{}, ErrInvalidRelease
	}
	release = normalizeRelease(release)
	if err = validRelease(release, true); err != nil {
		return profile.Release{}, err
	}
	if err = service.validateReleaseCapabilities(ctx, appID, release); err != nil {
		return profile.Release{}, err
	}
	return service.releases.UpdateDraft(ctx, actor.Tenant.ID, appID, release, actor.User.ID, requestID)
}
func (service *Service) PublishRelease(ctx context.Context, token, requestID string, appID, id uuid.UUID, lockVersion int32) (profile.Release, error) {
	actor, err := service.admin(ctx, token, "mobile.release.publish")
	if err != nil {
		return profile.Release{}, err
	}
	if appID == uuid.Nil || id == uuid.Nil || lockVersion < 1 {
		return profile.Release{}, ErrInvalidRelease
	}
	current, err := service.releases.GetRelease(ctx, actor.Tenant.ID, appID, id)
	if err != nil {
		return profile.Release{}, err
	}
	if err = service.validateReleaseCapabilities(ctx, appID, current); err != nil {
		return profile.Release{}, err
	}
	if err = service.validatePackageArchive(ctx, actor.Tenant.ID, current); err != nil {
		return profile.Release{}, err
	}
	return service.releases.Publish(ctx, actor.Tenant.ID, appID, id, lockVersion, actor.User.ID, requestID)
}

func (service *Service) validateReleaseCapabilities(ctx context.Context, appID uuid.UUID, release profile.Release) error {
	appType, err := service.releases.ApplicationType(ctx, appID)
	if err != nil {
		return err
	}
	return validReleaseCapabilities(appType, release)
}

func validReleaseCapabilities(appType string, release profile.Release) error {
	if appType != "uni_app" && appType != "uni_app_x" {
		return profile.ErrReleaseNotFound
	}
	if appType == "uni_app_x" && release.PackageType == "wgt" {
		return profile.ErrReleasePackageTypeUnsupported
	}
	if release.PackageType == "native_app" && release.PackageFileID != nil &&
		(len(release.Platforms) != 1 || release.Platforms[0] != "android") {
		return profile.ErrReleaseDeliveryModeUnsupported
	}
	return nil
}

type packageFileRepository interface {
	PackageFile(context.Context, uuid.UUID, uuid.UUID) (profile.PackageFile, error)
}

func (service *Service) validatePackageArchive(ctx context.Context, tenantID uuid.UUID, release profile.Release) error {
	if release.PackageFileID == nil {
		return nil
	}
	files, ok := service.releases.(packageFileRepository)
	if !ok || service.objects == nil || len(service.signingKey) == 0 {
		return profile.ErrReleaseFileInvalid
	}
	file, err := files.PackageFile(ctx, tenantID, *release.PackageFileID)
	if err != nil {
		return err
	}
	reader, err := service.objects.Open(ctx, storage.ObjectRef{TenantID: file.TenantID, Provider: file.Provider, Bucket: file.Bucket, Key: file.ObjectKey})
	if err != nil {
		return profile.ErrReleaseFileInvalid
	}
	defer func() { _ = reader.Close() }()
	header := make([]byte, 4)
	if _, err = io.ReadFull(reader, header); err != nil {
		return profile.ErrReleaseFileInvalid
	}
	// APK, IPA, HAP and WGT are ZIP-family archives. Accept the normal,
	// empty-archive and spanning signatures; extension/MIME checks remain in
	// the repository and select the expected package family.
	if !isZipArchiveHeader(header) {
		return profile.ErrReleaseFileInvalid
	}
	return nil
}

func isZipArchiveHeader(header []byte) bool {
	return len(header) >= 4 && header[0] == 'P' && header[1] == 'K' &&
		((header[2] == 3 && header[3] == 4) || (header[2] == 5 && header[3] == 6) || (header[2] == 7 && header[3] == 8))
}
func (service *Service) UnpublishRelease(ctx context.Context, token, requestID string, appID, id uuid.UUID, lockVersion int32) (profile.Release, error) {
	actor, err := service.admin(ctx, token, "mobile.release.publish")
	if err != nil {
		return profile.Release{}, err
	}
	if appID == uuid.Nil || id == uuid.Nil || lockVersion < 1 {
		return profile.Release{}, ErrInvalidRelease
	}
	return service.releases.Unpublish(ctx, actor.Tenant.ID, appID, id, lockVersion, actor.User.ID, requestID)
}
func (service *Service) DeleteReleases(ctx context.Context, token, requestID string, appID uuid.UUID, ids []uuid.UUID) error {
	actor, err := service.admin(ctx, token, "mobile.release.delete")
	if err != nil {
		return err
	}
	if appID == uuid.Nil || len(ids) < 1 || len(ids) > 100 {
		return ErrInvalidRelease
	}
	return service.releases.Delete(ctx, actor.Tenant.ID, appID, ids, actor.User.ID, requestID)
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
	x = normalizeRelease(x)
	current, currentOK := parseSemver(x.Version)
	minimumOK := true
	var minimum semver
	if x.MinimumNativeVersion != nil {
		minimum, minimumOK = parseSemver(*x.MinimumNativeVersion)
	}
	if (update && (x.ID == uuid.Nil || x.LockVersion < 1)) ||
		!currentOK || !minimumOK || (x.MinimumNativeVersion != nil && compareSemver(minimum, current) > 0) ||
		len(x.Platforms) < 1 || len(x.Platforms) > 3 || len(x.StoreListingIDs) > 100 {
		return ErrInvalidRelease
	}
	if x.Active && (strings.TrimSpace(x.Titles["zh-CN"]) == "" || strings.TrimSpace(x.Titles["en-US"]) == "" ||
		strings.TrimSpace(x.Contents["zh-CN"]) == "" || strings.TrimSpace(x.Contents["en-US"]) == "") {
		return ErrInvalidRelease
	}
	seen := map[string]bool{}
	for _, platform := range x.Platforms {
		if !validPlatform(platform) || seen[platform] {
			return ErrInvalidRelease
		}
		seen[platform] = true
	}
	if (x.PackageType == "native_app" && len(x.Platforms) != 1) || (x.PackageType == "wgt" && x.MinimumNativeVersion == nil) || (x.PackageType != "native_app" && x.PackageType != "wgt") {
		return ErrInvalidRelease
	}
	if x.PackageFileID != nil && x.ExternalURL != nil {
		return ErrInvalidRelease
	}
	if x.Active && (x.PackageFileID == nil && x.ExternalURL == nil) {
		return ErrInvalidRelease
	}
	if x.ExternalURL != nil && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(*x.ExternalURL)), "https://") {
		return ErrInvalidRelease
	}
	return nil
}

func normalizeRelease(input profile.Release) profile.Release {
	if input.PackageType == "" {
		input.PackageType = "native_app"
	}
	if len(input.Platforms) == 0 && input.Platform != "" {
		input.Platforms = []string{input.Platform}
	}
	if input.Version == "" {
		input.Version = input.CurrentVersion
	}
	if input.MinimumNativeVersion == nil && input.MinimumVersion != "" {
		value := input.MinimumVersion
		input.MinimumNativeVersion = &value
	}
	if input.ExternalURL == nil {
		input.ExternalURL = input.UpgradeURL
	}
	if len(input.Contents) == 0 {
		input.Contents = input.ReleaseNotes
	}
	if len(input.Titles) == 0 {
		input.Titles = map[string]string{"zh-CN": input.Version, "en-US": input.Version}
	}
	if input.CreateEnv == "" {
		input.CreateEnv = "upgrade_center"
	}
	input.Platforms = uniqueStrings(input.Platforms)
	return input
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func validPlatform(value string) bool {
	return value == "android" || value == "ios" || value == "harmony"
}

func (service *Service) SignedPackageURL(release profile.Release) *string {
	if release.PackageFileID == nil || len(service.signingKey) == 0 {
		return release.ExternalURL
	}
	expires := service.clock().UTC().Add(5 * time.Minute).Unix()
	signature := signPackageDownload(service.signingKey, release.AppID, release.ID, *release.PackageFileID, expires)
	url := fmt.Sprintf("/api/v1/public/app-version/download/%s/%s?expires=%d&signature=%s", release.ID, release.PackageFileID.String(), expires, signature)
	return &url
}

func (service *Service) OpenPackageDownload(ctx context.Context, appID, releaseID, fileID uuid.UUID, expires int64, signature string) (profile.PackageFile, io.ReadCloser, error) {
	if service.objects == nil || len(service.signingKey) == 0 || appID == uuid.Nil || releaseID == uuid.Nil || fileID == uuid.Nil {
		return profile.PackageFile{}, nil, ErrInvalidRelease
	}
	if !validPackageDownloadSignature(service.signingKey, appID, releaseID, fileID, expires, service.clock().UTC().Unix(), signature) {
		return profile.PackageFile{}, nil, ErrInvalidRelease
	}
	file, err := service.releases.PublishedPackageFile(ctx, appID, releaseID, fileID)
	if err != nil {
		return profile.PackageFile{}, nil, err
	}
	reader, err := service.objects.Open(ctx, storage.ObjectRef{TenantID: file.TenantID, Provider: file.Provider, Bucket: file.Bucket, Key: file.ObjectKey})
	if err != nil {
		return profile.PackageFile{}, nil, err
	}
	return file, reader, nil
}

func packageDownloadPayload(appID, releaseID, fileID uuid.UUID, expires int64) string {
	return appID.String() + "|" + releaseID.String() + "|" + fileID.String() + "|" + strconv.FormatInt(expires, 10)
}

func signPackageDownload(key []byte, appID, releaseID, fileID uuid.UUID, expires int64) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(packageDownloadPayload(appID, releaseID, fileID, expires)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func validPackageDownloadSignature(key []byte, appID, releaseID, fileID uuid.UUID, expires, now int64, signature string) bool {
	if len(key) == 0 || expires < now || expires > now+600 {
		return false
	}
	expected, err := base64.RawURLEncoding.DecodeString(signPackageDownload(key, appID, releaseID, fileID, expires))
	if err != nil {
		return false
	}
	actual, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(signature))
	return err == nil && hmac.Equal(expected, actual)
}

type semver [3]uint64

func parseSemver(raw string) (semver, bool) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	var out semver
	for i, part := range parts {
		if part == "" || len(part) > 1 && part[0] == '0' {
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
	principal, err := service.auth.Authenticate(ctx, token, "ak-mobile")
	if err != nil {
		return domain.AuthenticatedContext{}, err
	}
	if principal.AppID == nil || *principal.AppID == uuid.Nil {
		return domain.AuthenticatedContext{}, ErrForbidden
	}
	return principal, nil
}
func (service *Service) Preferences(ctx context.Context, token string) (profile.Preferences, error) {
	principal, err := service.authenticate(ctx, token)
	if err != nil {
		return profile.Preferences{}, err
	}
	return service.repository.GetPreferences(ctx, *principal.AppID, principal.User.ID)
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
		if key != "in_app" && key != "push" && key != "push_service" && key != "push_operations" && key != "email" {
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
		UserID: principal.User.ID, AppID: *principal.AppID, TenantID: principal.Tenant.ID, SessionID: principal.SessionID, RequestID: requestID,
		Locale: locale, Appearance: appearance, NotificationPreferences: notifications,
	})
}
func (service *Service) UnreadCount(ctx context.Context, token string) (int64, error) {
	principal, err := service.authenticate(ctx, token)
	if err != nil {
		return 0, err
	}
	return service.repository.UnreadCount(ctx, principal.User.ID, principal.Tenant.ID, *principal.AppID)
}
func (service *Service) LoginEvents(ctx context.Context, token string) ([]profile.LoginEvent, error) {
	principal, err := service.authenticate(ctx, token)
	if err != nil {
		return nil, err
	}
	return service.repository.LoginEvents(ctx, principal.User.ID, *principal.AppID)
}
func (service *Service) SecurityEvents(ctx context.Context, token string) ([]profile.SecurityEvent, error) {
	principal, err := service.authenticate(ctx, token)
	if err != nil {
		return nil, err
	}
	return service.repository.SecurityEvents(ctx, principal.User.ID, *principal.AppID)
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
	return service.repository.Notifications(ctx, p.User.ID, p.Tenant.ID, *p.AppID, cursor, limit)
}
func (service *Service) Notification(ctx context.Context, token, id string) (profile.Notification, error) {
	messageID, err := uuid.Parse(id)
	if err != nil {
		return profile.Notification{}, ErrInvalidPreferences
	}
	p, err := service.authenticate(ctx, token)
	if err != nil {
		return profile.Notification{}, err
	}
	return service.repository.Notification(ctx, p.User.ID, p.Tenant.ID, *p.AppID, messageID)
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
	return service.repository.MarkNotificationRead(ctx, p.User.ID, p.Tenant.ID, *p.AppID, p.SessionID, messageID, requestID)
}
