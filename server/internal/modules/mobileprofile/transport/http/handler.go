package http

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"

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

func (handler *Handler) AppVersion(request *ghttp.Request) {
	release, err := handler.service.PublicRelease(request.Context(), request.GetQuery("platform").String())
	if status, code, key := publicReleaseError(err); err != nil {
		handler.failure(request, status, code, key)
		return
	}
	request.Response.Header().Set("Vary", "Accept-Language")
	handler.success(request, publicReleaseResponse(release, httpx.Locale(request)))
}
func (handler *Handler) AdminReleases(request *ghttp.Request) {
	data, err := handler.service.AdminReleases(request.Context(), bearer(request))
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
	out, err := handler.service.CreateRelease(request.Context(), bearer(request), httpx.RequestID(request), release.domain(uuid.Nil, 0))
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
	out, err := handler.service.UpdateRelease(request.Context(), bearer(request), httpx.RequestID(request), release.domain(id, release.LockVersion))
	if err != nil {
		handler.adminError(request, err)
		return
	}
	handler.success(request, out)
}

type mobileReleaseResponse struct {
	Platform       string  `json:"platform"`
	CurrentVersion string  `json:"current_version"`
	MinimumVersion string  `json:"minimum_version"`
	UpgradeURL     *string `json:"upgrade_url"`
	ReleaseNotes   string  `json:"release_notes"`
}

func publicReleaseResponse(release profiledomain.Release, locale i18n.Locale) mobileReleaseResponse {
	return mobileReleaseResponse{Platform: release.Platform, CurrentVersion: release.CurrentVersion, MinimumVersion: release.MinimumVersion, UpgradeURL: release.UpgradeURL, ReleaseNotes: release.ReleaseNotes[string(locale)]}
}
func publicReleaseError(err error) (int, string, string) {
	if err == nil {
		return 0, "", ""
	}
	if errors.Is(err, profile.ErrInvalidRelease) {
		return http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed"
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
	case errors.Is(err, profiledomain.ErrReleaseNotFound):
		handler.failure(request, http.StatusNotFound, "SYS.MOBILE_RELEASE.NOT_FOUND", "errors.common.not_found")
	default:
		handler.failure(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
	}
}

type releaseRequest struct {
	Platform       string            `json:"platform"`
	CurrentVersion string            `json:"current_version"`
	MinimumVersion string            `json:"minimum_version"`
	UpgradeURL     *string           `json:"upgrade_url"`
	ReleaseNotes   map[string]string `json:"release_notes"`
	Active         bool              `json:"active"`
	LockVersion    int32             `json:"lock_version"`
}

func (x releaseRequest) domain(id uuid.UUID, lock int32) profiledomain.Release {
	return profiledomain.Release{ID: id, Platform: x.Platform, CurrentVersion: x.CurrentVersion, MinimumVersion: x.MinimumVersion, UpgradeURL: x.UpgradeURL, ReleaseNotes: x.ReleaseNotes, Active: x.Active, LockVersion: lock}
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
