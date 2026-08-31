package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	profile "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/application"
	profiledomain "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
)

type Handler struct {
	service *profile.Service
	catalog *i18n.Catalog
}

func NewHandler(service *profile.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func bearer(request *ghttp.Request) string {
	return bearerHeader(request.Header.Get("Authorization"))
}
func bearerHeader(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return ""
	}
	return parts[1]
}
func (handler *Handler) failure(request *ghttp.Request, status int, code, messageKey string) {
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: messageKey, Message: handler.catalog.Translate(httpx.Locale(request), messageKey, nil)}, RequestID: httpx.RequestID(request)})
}
func (handler *Handler) authFailure(request *ghttp.Request) {
	handler.failure(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
}
func (handler *Handler) Preferences(request *ghttp.Request) {
	data, err := handler.service.Preferences(request.Context(), bearer(request))
	if err != nil {
		handler.authFailure(request)
		return
	}
	handler.success(request, data)
}
func (handler *Handler) UpdatePreferences(request *ghttp.Request) {
	var body struct {
		Locale                  *string         `json:"locale"`
		Appearance              *string         `json:"appearance"`
		NotificationPreferences map[string]bool `json:"notification_preferences"`
	}
	if err := decode(request, &body); err != nil {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	data, err := handler.service.UpdatePreferences(request.Context(), bearer(request), httpx.RequestID(request), body.Locale, body.Appearance, body.NotificationPreferences)
	if errors.Is(err, profile.ErrInvalidPreferences) {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		handler.authFailure(request)
		return
	}
	handler.success(request, data)
}
func (handler *Handler) NotificationPreferences(request *ghttp.Request) {
	data, err := handler.service.Preferences(request.Context(), bearer(request))
	if err != nil {
		handler.authFailure(request)
		return
	}
	handler.success(request, map[string]map[string]bool{"notification_preferences": data.NotificationPreferences})
}
func (handler *Handler) UpdateNotificationPreferences(request *ghttp.Request) {
	var body notificationPreferencesRequest
	if err := decode(request, &body); err != nil {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	data, err := handler.service.UpdatePreferences(request.Context(), bearer(request), httpx.RequestID(request), nil, nil, body.NotificationPreferences)
	if errors.Is(err, profile.ErrInvalidPreferences) {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		handler.authFailure(request)
		return
	}
	handler.success(request, map[string]map[string]bool{"notification_preferences": data.NotificationPreferences})
}

type notificationPreferencesRequest struct {
	NotificationPreferences map[string]bool `json:"notification_preferences"`
}

func (handler *Handler) UnreadCount(request *ghttp.Request) {
	count, err := handler.service.UnreadCount(request.Context(), bearer(request))
	if err != nil {
		handler.authFailure(request)
		return
	}
	handler.success(request, map[string]int64{"count": count})
}
func (handler *Handler) LoginEvents(request *ghttp.Request) {
	data, err := handler.service.LoginEvents(request.Context(), bearer(request))
	if err != nil {
		handler.authFailure(request)
		return
	}
	handler.success(request, data)
}
func (handler *Handler) SecurityEvents(request *ghttp.Request) {
	data, err := handler.service.SecurityEvents(request.Context(), bearer(request))
	if err != nil {
		handler.authFailure(request)
		return
	}
	handler.success(request, data)
}
func (handler *Handler) Notifications(request *ghttp.Request) {
	limit := 20
	if raw := request.GetQuery("limit").Int(); raw > 0 {
		limit = raw
	}
	data, err := handler.service.Notifications(request.Context(), bearer(request), request.GetQuery("cursor").String(), limit)
	if errors.Is(err, profile.ErrInvalidPreferences) {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		handler.authFailure(request)
		return
	}
	handler.success(request, data)
}
func (handler *Handler) Notification(request *ghttp.Request) {
	data, err := handler.service.Notification(request.Context(), bearer(request), request.Get("id").String())
	if errors.Is(err, profile.ErrInvalidPreferences) {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if errors.Is(err, profiledomain.ErrNotificationNotFound) {
		handler.failure(request, http.StatusNotFound, "NOTIFY.RECIPIENT.NOT_FOUND", "errors.common.not_found")
		return
	}
	if err != nil {
		handler.authFailure(request)
		return
	}
	handler.success(request, data)
}
func (handler *Handler) MarkNotificationRead(request *ghttp.Request) {
	err := handler.service.MarkNotificationRead(request.Context(), bearer(request), httpx.RequestID(request), request.Get("id").String())
	if errors.Is(err, profile.ErrInvalidPreferences) {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		handler.failure(request, http.StatusNotFound, "NOTIFY.RECIPIENT.NOT_FOUND", "errors.common.not_found")
		return
	}
	handler.success(request, map[string]bool{"read": true})
}

func (handler *Handler) MarkAllNotificationsRead(request *ghttp.Request) {
	updated, err := handler.service.MarkAllNotificationsRead(request.Context(), bearer(request), httpx.RequestID(request))
	if err != nil {
		handler.authFailure(request)
		return
	}
	handler.success(request, map[string]int64{"updated_count": updated})
}

func (handler *Handler) AppVersion(request *ghttp.Request) {
	appID, ok := requestAppID(request)
	if !ok {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	packageType := strings.TrimSpace(request.GetQuery("package_type").String())
	if packageType == "" {
		packageType = "native_app"
	}
	release, err := handler.service.PublicPackageRelease(request.Context(), appID, request.GetQuery("platform").String(), packageType)
	if status, code, key := publicReleaseError(err); err != nil {
		handler.failure(request, status, code, key)
		return
	}
	release.UpgradeURL = handler.service.SignedPackageURL(release)
	request.Response.Header().Set("Vary", "Accept-Language, X-AppID")
	request.Response.Header().Set("Cache-Control", "no-store")
	handler.success(request, publicReleaseResponse(release, httpx.Locale(request)))
}
func (handler *Handler) AppVersionDownload(request *ghttp.Request) {
	appID, ok := requestAppID(request)
	releaseID, releaseErr := uuid.Parse(request.Get("release_id").String())
	fileID, fileErr := uuid.Parse(request.Get("file_id").String())
	expires, expiresErr := strconv.ParseInt(request.GetQuery("expires").String(), 10, 64)
	if !ok || releaseErr != nil || fileErr != nil || expiresErr != nil {
		handler.failure(request, http.StatusNotFound, "SYS.MOBILE_RELEASE.NOT_FOUND", "errors.common.not_found")
		return
	}
	file, reader, err := handler.service.OpenPackageDownload(request.Context(), appID, releaseID, fileID, expires, request.GetQuery("signature").String())
	if err != nil {
		handler.failure(request, http.StatusNotFound, "SYS.MOBILE_RELEASE.NOT_FOUND", "errors.common.not_found")
		return
	}
	defer func() { _ = reader.Close() }()
	request.Response.Header().Set("Content-Type", file.MediaType)
	request.Response.Header().Set("Content-Length", fmt.Sprintf("%d", file.SizeBytes))
	request.Response.Header().Set("Content-Disposition", `attachment; filename="app-package"`)
	request.Response.Header().Set("Cache-Control", "private, no-store")
	request.Response.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(request.Response.BufferWriter, reader)
}
func (handler *Handler) AdminReleases(request *ghttp.Request) {
	appID, ok := requestAppID(request)
	if !ok {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if strings.TrimSpace(request.GetRouter("app_id").String()) == "" {
		data, err := handler.service.AdminReleases(request.Context(), bearer(request), appID)
		if err != nil {
			handler.adminError(request, err)
			return
		}
		handler.success(request, data)
		return
	}
	filter, ok := releaseFilter(request)
	if !ok {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	data, err := handler.service.AdminReleasePage(request.Context(), bearer(request), appID, filter)
	if err != nil {
		handler.adminError(request, err)
		return
	}
	handler.success(request, data)
}
func (handler *Handler) AdminRelease(request *ghttp.Request) {
	appID, ok := requestAppID(request)
	id, err := uuid.Parse(request.Get("id").String())
	if !ok || err != nil {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if request.Method == http.MethodDelete {
		if err = handler.service.DeleteReleases(request.Context(), bearer(request), httpx.RequestID(request), appID, []uuid.UUID{id}); err != nil {
			handler.adminError(request, err)
			return
		}
		handler.success(request, map[string]bool{"deleted": true})
		return
	}
	data, err := handler.service.AdminRelease(request.Context(), bearer(request), appID, id)
	if err != nil {
		handler.adminError(request, err)
		return
	}
	handler.success(request, data)
}
func (handler *Handler) AdminCreateRelease(request *ghttp.Request) {
	var release releaseRequest
	if err := decode(request, &release); err != nil {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	appID, ok := requestAppID(request)
	if !ok {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	out, err := handler.service.CreateRelease(request.Context(), bearer(request), httpx.RequestID(request), appID, release.domain(uuid.Nil, 0))
	if err != nil {
		handler.adminError(request, err)
		return
	}
	handler.successStatus(request, http.StatusCreated, out)
}
func (handler *Handler) AdminUpdateRelease(request *ghttp.Request) {
	id, err := uuid.Parse(request.Get("id").String())
	if err != nil {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var release releaseRequest
	if err = decode(request, &release); err != nil {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	appID, ok := requestAppID(request)
	if !ok {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	out, err := handler.service.UpdateRelease(request.Context(), bearer(request), httpx.RequestID(request), appID, release.domain(id, release.LockVersion))
	if err != nil {
		handler.adminError(request, err)
		return
	}
	handler.success(request, out)
}
func (handler *Handler) AdminPublishRelease(request *ghttp.Request) {
	handler.releasePublicationAction(request, true)
}
func (handler *Handler) AdminUnpublishRelease(request *ghttp.Request) {
	handler.releasePublicationAction(request, false)
}
func (handler *Handler) releasePublicationAction(request *ghttp.Request, publish bool) {
	appID, ok := requestAppID(request)
	id, err := uuid.Parse(request.Get("id").String())
	var body struct {
		LockVersion int32 `json:"lock_version"`
	}
	if !ok || err != nil || decode(request, &body) != nil {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var out profiledomain.Release
	if publish {
		out, err = handler.service.PublishRelease(request.Context(), bearer(request), httpx.RequestID(request), appID, id, body.LockVersion)
	} else {
		out, err = handler.service.UnpublishRelease(request.Context(), bearer(request), httpx.RequestID(request), appID, id, body.LockVersion)
	}
	if err != nil {
		handler.adminError(request, err)
		return
	}
	handler.success(request, out)
}
func (handler *Handler) AdminBatchDeleteReleases(request *ghttp.Request) {
	appID, ok := requestAppID(request)
	var body struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if !ok || decode(request, &body) != nil {
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err := handler.service.DeleteReleases(request.Context(), bearer(request), httpx.RequestID(request), appID, body.IDs); err != nil {
		handler.adminError(request, err)
		return
	}
	handler.success(request, map[string]int{"deleted_count": len(body.IDs)})
}

func releaseFilter(request *ghttp.Request) (profiledomain.ReleaseFilter, bool) {
	filter := profiledomain.ReleaseFilter{Query: strings.TrimSpace(request.GetQuery("q").String()), PackageType: strings.TrimSpace(request.GetQuery("package_type").String()), Platform: strings.TrimSpace(request.GetQuery("platform").String()), PublishStatus: strings.TrimSpace(request.GetQuery("publish_status").String()), Page: 1, PageSize: 20}
	if raw := request.GetQuery("page").String(); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return filter, false
		}
		filter.Page = int32(value)
	}
	if raw := request.GetQuery("page_size").String(); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return filter, false
		}
		filter.PageSize = int32(value)
	}
	return filter, true
}
func requestAppID(request *ghttp.Request) (uuid.UUID, bool) {
	raw := strings.TrimSpace(request.GetRouter("app_id").String())
	if raw == "" {
		raw = strings.TrimSpace(request.Header.Get("X-AppID"))
	}
	id, err := uuid.Parse(raw)
	return id, err == nil && id != uuid.Nil
}

type mobileReleaseResponse struct {
	Platform       string               `json:"platform"`
	PackageType    string               `json:"package_type"`
	DeliveryMode   string               `json:"delivery_mode"`
	CurrentVersion string               `json:"current_version"`
	MinimumVersion string               `json:"minimum_version"`
	UpgradeURL     *string              `json:"upgrade_url"`
	StoreList      []mobileReleaseStore `json:"store_list"`
	Title          string               `json:"title"`
	ReleaseNotes   string               `json:"release_notes"`
	IsSilently     bool                 `json:"is_silently"`
	IsMandatory    bool                 `json:"is_mandatory"`
	PublishedAt    *time.Time           `json:"published_at,omitempty"`
}

type mobileReleaseStore struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Scheme   string    `json:"scheme"`
	Priority int32     `json:"priority"`
}

func publicReleaseResponse(release profiledomain.Release, locale i18n.Locale) mobileReleaseResponse {
	localeCode := string(locale)
	title, notes := release.Titles[localeCode], release.Contents[localeCode]
	if notes == "" {
		notes = release.ReleaseNotes[localeCode]
	}
	if title == "" {
		title = release.Titles["zh-CN"]
	}
	if notes == "" {
		notes = release.Contents["zh-CN"]
	}
	if notes == "" {
		notes = release.ReleaseNotes["zh-CN"]
	}
	if release.PackageType == "" {
		release.PackageType = "native_app"
	}
	deliveryMode := "external_link"
	if release.PackageFileID != nil {
		deliveryMode = "internal_package"
	}
	stores := make([]mobileReleaseStore, 0, len(release.StoreList))
	for _, store := range release.StoreList {
		stores = append(stores, mobileReleaseStore{ID: store.ID, Name: store.Name, Scheme: store.Scheme, Priority: store.Priority})
	}
	return mobileReleaseResponse{Platform: release.Platform, PackageType: release.PackageType, DeliveryMode: deliveryMode, CurrentVersion: release.CurrentVersion,
		MinimumVersion: release.MinimumVersion, UpgradeURL: release.UpgradeURL, StoreList: stores, Title: title, ReleaseNotes: notes,
		IsSilently: release.IsSilently, IsMandatory: release.IsMandatory, PublishedAt: release.LastPublishedAt}
}
func publicReleaseError(err error) (int, string, string) {
	if err == nil {
		return 0, "", ""
	}
	if errors.Is(err, profile.ErrInvalidRelease) {
		return http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed"
	}
	if errors.Is(err, profiledomain.ErrReleasePackageTypeUnsupported) {
		return http.StatusUnprocessableEntity, "SYS.MOBILE_RELEASE.UNSUPPORTED_PACKAGE_TYPE", "errors.mobile_release.unsupported_package_type"
	}
	if errors.Is(err, profiledomain.ErrReleaseDeliveryModeUnsupported) {
		return http.StatusUnprocessableEntity, "SYS.MOBILE_RELEASE.UNSUPPORTED_DELIVERY_MODE", "errors.mobile_release.unsupported_delivery_mode"
	}
	return http.StatusServiceUnavailable, "APP.RELEASE.UNAVAILABLE", "errors.common.unknown"
}

func (handler *Handler) adminError(request *ghttp.Request, err error) {
	switch {
	case errors.Is(err, iamapp.ErrInvalidAccessToken):
		handler.authFailure(request)
	case errors.Is(err, profile.ErrForbidden):
		handler.failure(request, http.StatusForbidden, "AUTH.PERMISSION.FORBIDDEN", "errors.common.forbidden")
	case errors.Is(err, profile.ErrInvalidRelease):
		handler.failure(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
	case errors.Is(err, profiledomain.ErrReleaseConflict):
		handler.failure(request, http.StatusConflict, "SYS.MOBILE_RELEASE.CONFLICT", "errors.common.conflict")
	case errors.Is(err, profiledomain.ErrReleaseFrozen):
		handler.failure(request, http.StatusConflict, "SYS.MOBILE_RELEASE.FROZEN", "errors.common.conflict")
	case errors.Is(err, profiledomain.ErrReleaseDeleteForbidden):
		handler.failure(request, http.StatusConflict, "SYS.MOBILE_RELEASE.DELETE_FORBIDDEN", "errors.common.conflict")
	case errors.Is(err, profiledomain.ErrReleaseVersionNotIncreasing):
		handler.failure(request, http.StatusConflict, "SYS.MOBILE_RELEASE.VERSION_NOT_INCREASING", "errors.common.conflict")
	case errors.Is(err, profiledomain.ErrReleaseFileInvalid):
		handler.failure(request, http.StatusUnprocessableEntity, "SYS.MOBILE_RELEASE.FILE_INVALID", "errors.validation.failed")
	case errors.Is(err, profiledomain.ErrReleasePackageTypeUnsupported):
		handler.failure(request, http.StatusUnprocessableEntity, "SYS.MOBILE_RELEASE.UNSUPPORTED_PACKAGE_TYPE", "errors.mobile_release.unsupported_package_type")
	case errors.Is(err, profiledomain.ErrReleaseDeliveryModeUnsupported):
		handler.failure(request, http.StatusUnprocessableEntity, "SYS.MOBILE_RELEASE.UNSUPPORTED_DELIVERY_MODE", "errors.mobile_release.unsupported_delivery_mode")
	case errors.Is(err, profiledomain.ErrReleaseNotFound):
		handler.failure(request, http.StatusNotFound, "SYS.MOBILE_RELEASE.NOT_FOUND", "errors.common.not_found")
	default:
		handler.failure(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
	}
}

type releaseRequest struct {
	PackageType          string            `json:"package_type"`
	Platforms            []string          `json:"platforms"`
	Version              string            `json:"version"`
	MinimumNativeVersion *string           `json:"minimum_native_version"`
	Titles               map[string]string `json:"titles"`
	Contents             map[string]string `json:"contents"`
	PackageFileID        *uuid.UUID        `json:"package_file_id"`
	ExternalURL          *string           `json:"external_url"`
	StoreListingIDs      []uuid.UUID       `json:"store_listing_ids"`
	CreateEnv            string            `json:"create_env"`
	IsSilently           bool              `json:"is_silently"`
	IsMandatory          bool              `json:"is_mandatory"`
	PublishNow           bool              `json:"publish_now"`
	Platform             string            `json:"platform"`
	CurrentVersion       string            `json:"current_version"`
	MinimumVersion       string            `json:"minimum_version"`
	UpgradeURL           *string           `json:"upgrade_url"`
	ReleaseNotes         map[string]string `json:"release_notes"`
	Active               bool              `json:"active"`
	LockVersion          int32             `json:"lock_version"`
}

func (x releaseRequest) domain(id uuid.UUID, lock int32) profiledomain.Release {
	return profiledomain.Release{ID: id, PackageType: x.PackageType, Platforms: x.Platforms, Version: x.Version,
		MinimumNativeVersion: x.MinimumNativeVersion, Titles: x.Titles, Contents: x.Contents, PackageFileID: x.PackageFileID,
		ExternalURL: x.ExternalURL, StoreListingIDs: x.StoreListingIDs, CreateEnv: x.CreateEnv,
		IsSilently: x.IsSilently, IsMandatory: x.IsMandatory, Platform: x.Platform, CurrentVersion: x.CurrentVersion,
		MinimumVersion: x.MinimumVersion, UpgradeURL: x.UpgradeURL, ReleaseNotes: x.ReleaseNotes,
		Active: x.Active || x.PublishNow, LockVersion: lock}
}
func (handler *Handler) success(request *ghttp.Request, data any) {
	handler.successStatus(request, http.StatusOK, data)
}
func (handler *Handler) successStatus(request *ghttp.Request, status int, data any) {
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(request)})
}
func decode(request *ghttp.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(request.Body, 32<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values")
	}
	return nil
}
