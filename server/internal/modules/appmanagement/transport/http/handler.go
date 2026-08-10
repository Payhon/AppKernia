package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	app "github.com/appkernia/appkernia/server/internal/modules/appmanagement/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

const appContextKey = "appkernia.mobile_app"

type Handler struct {
	service *app.Service
	catalog *i18n.Catalog
}
type pageEnvelope[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

func NewHandler(service *app.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

// RequireMobileApp runs for every /api/v1 route, including public endpoints.
// It deliberately resolves only a public UUID and records the server-derived
// tenant scope; the header is never treated as a tenant or user assertion.
func (h *Handler) RequireMobileApp(r *ghttp.Request) {
	id, err := uuid.Parse(strings.TrimSpace(r.Header.Get("X-AppID")))
	if err != nil {
		h.fail(r, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	a, err := h.service.Resolve(r.Context(), id)
	if errors.Is(err, app.ErrAppNotFound) {
		h.fail(r, http.StatusNotFound, "APP.NOT_FOUND", "errors.common.not_found")
		return
	}
	if errors.Is(err, app.ErrAppDisabled) {
		h.fail(r, http.StatusForbidden, "APP.DISABLED", "errors.common.forbidden")
		return
	}
	if err != nil {
		h.fail(r, http.StatusServiceUnavailable, "APP.UNAVAILABLE", "errors.common.unknown")
		return
	}
	r.SetCtxVar(appContextKey, a)
	r.Response.Header().Set("Vary", "Accept-Language, X-AppID")
	r.Middleware.Next()
}

// RequireMobileSessionApp makes the App header/token relationship fail closed
// before any authenticated mobile handler runs. Public/auth bootstrap routes
// are exempt because they have no bearer session yet; refresh is checked after
// rotation by the auth handler against the stored session App ID.
func (h *Handler) RequireMobileSessionApp(r *ghttp.Request) {
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/public/") || strings.HasPrefix(path, "/api/v1/auth/") || path == "/api/v1/regions" {
		r.Middleware.Next()
		return
	}
	a, ok := currentApp(r)
	if !ok {
		h.fail(r, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	principal, err := h.service.AuthenticateMobileMembership(r.Context(), bearer(r), a)
	if err != nil || principal.AppID == nil || *principal.AppID != a.ID {
		h.fail(r, http.StatusUnauthorized, "APP.SESSION.MISMATCH", "errors.common.unauthorized")
		return
	}
	r.Middleware.Next()
}

func (h *Handler) PublicConfig(r *ghttp.Request) {
	a, ok := currentApp(r)
	if !ok {
		h.fail(r, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	r.Response.Header().Set("Cache-Control", "public, max-age=60")
	r.Response.WriteJsonExit(httpx.Success[map[string]any]{Code: "OK", Message: "OK", RequestID: httpx.RequestID(r), Data: map[string]any{
		"app_id": a.ID.String(), "appid": a.AppID, "app_type": a.AppType, "name": a.Name, "default_locale": a.DefaultLocale,
		"registration_enabled": a.RegistrationEnabled, "registration_verification_mode": a.RegistrationVerification,
	}})
}

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Locale      string `json:"locale"`
	AcceptTerms bool   `json:"accept_terms"`
}
type otpRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}
type resetRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) Register(r *ghttp.Request) {
	a, ok := currentApp(r)
	if !ok {
		h.fail(r, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	var body registerRequest
	if !decode(r, &body) || !body.AcceptTerms {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	err := h.service.RegisterMobile(r.Context(), a, body.Email, body.DisplayName, body.Password, body.Locale)
	if errors.Is(err, app.ErrAppDisabled) {
		h.fail(r, http.StatusForbidden, "APP.REGISTRATION.DISABLED", "errors.common.forbidden")
		return
	}
	if errors.Is(err, app.ErrInvalidInput) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		h.fail(r, http.StatusServiceUnavailable, "APP.REGISTRATION.UNAVAILABLE", "errors.common.unknown")
		return
	}
	r.Response.WriteHeader(http.StatusAccepted)
	r.Response.WriteJsonExit(httpx.Success[map[string]any]{Code: "OK", Message: "OK", Data: map[string]any{"accepted": true, "verification_required": a.RegistrationVerification == "email_otp"}, RequestID: httpx.RequestID(r)})
}
func (h *Handler) VerifyRegistration(r *ghttp.Request) {
	a, ok := currentApp(r)
	if !ok {
		h.fail(r, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	var body otpRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	err := h.service.VerifyRegistrationEmail(r.Context(), a, body.Email, body.Code)
	if errors.Is(err, app.ErrInvalidInput) {
		h.fail(r, http.StatusUnprocessableEntity, "IAM.EMAIL.VERIFICATION_INVALID", "errors.validation.failed")
		return
	}
	if err != nil {
		h.fail(r, http.StatusServiceUnavailable, "IAM.EMAIL.VERIFICATION_UNAVAILABLE", "errors.common.unknown")
		return
	}
	r.Response.WriteJsonExit(httpx.Success[map[string]bool]{Code: "OK", Message: "OK", Data: map[string]bool{"verified": true}, RequestID: httpx.RequestID(r)})
}
func (h *Handler) ResendRegistration(r *ghttp.Request) {
	a, ok := currentApp(r)
	if !ok {
		h.fail(r, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	var body otpRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	wait, err := h.service.ResendRegistrationEmail(r.Context(), a, body.Email)
	if errors.Is(err, app.ErrInvalidInput) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		h.fail(r, http.StatusServiceUnavailable, "IAM.EMAIL.VERIFICATION_UNAVAILABLE", "errors.common.unknown")
		return
	}
	r.Response.Header().Set("Retry-After", fmt.Sprint(wait))
	r.Response.WriteHeader(http.StatusAccepted)
	r.Response.WriteJsonExit(httpx.Success[map[string]any]{Code: "OK", Message: "OK", Data: map[string]any{"accepted": true, "retry_after_seconds": wait}, RequestID: httpx.RequestID(r)})
}
func (h *Handler) ForgotPassword(r *ghttp.Request) {
	a, ok := currentApp(r)
	if !ok {
		h.fail(r, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	var body otpRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	wait, err := h.service.ForgotMobilePassword(r.Context(), a, body.Email)
	if errors.Is(err, app.ErrInvalidInput) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		h.fail(r, http.StatusServiceUnavailable, "IAM.PASSWORD.RECOVERY_UNAVAILABLE", "errors.common.unknown")
		return
	}
	r.Response.Header().Set("Retry-After", fmt.Sprint(wait))
	r.Response.WriteHeader(http.StatusAccepted)
	r.Response.WriteJsonExit(httpx.Success[map[string]any]{Code: "OK", Message: "OK", Data: map[string]any{"accepted": true, "retry_after_seconds": wait}, RequestID: httpx.RequestID(r)})
}
func (h *Handler) ResetPassword(r *ghttp.Request) {
	a, ok := currentApp(r)
	if !ok {
		h.fail(r, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	var body resetRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	err := h.service.ResetMobilePassword(r.Context(), a, body.Email, body.Code, body.NewPassword)
	if errors.Is(err, app.ErrInvalidInput) {
		h.fail(r, http.StatusUnprocessableEntity, "IAM.PASSWORD.RESET_TOKEN_INVALID", "errors.validation.failed")
		return
	}
	if err != nil {
		h.fail(r, http.StatusServiceUnavailable, "IAM.PASSWORD.RECOVERY_UNAVAILABLE", "errors.common.unknown")
		return
	}
	r.Response.WriteJsonExit(httpx.Success[map[string]bool]{Code: "OK", Message: "OK", Data: map[string]bool{"reset": true}, RequestID: httpx.RequestID(r)})
}

func (h *Handler) Legal(r *ghttp.Request) { h.page(r, r.GetRouter("document_type").String()) }
func (h *Handler) Page(r *ghttp.Request)  { h.page(r, r.GetRouter("slug").String()) }
func (h *Handler) page(r *ghttp.Request, slug string) {
	a, ok := currentApp(r)
	if !ok {
		h.fail(r, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	item, err := h.service.PublicPage(r.Context(), a.ID, slug, string(httpx.Locale(r)))
	if errors.Is(err, app.ErrAppNotFound) {
		h.fail(r, http.StatusNotFound, "CONTENT.PAGE.NOT_FOUND", "errors.common.not_found")
		return
	}
	if err != nil {
		h.fail(r, http.StatusServiceUnavailable, "CONTENT.PAGE.UNAVAILABLE", "errors.common.unknown")
		return
	}
	r.Response.Header().Set("Cache-Control", "public, max-age=60")
	r.Response.WriteJsonExit(httpx.Success[app.PublicPage]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}

type consentRequest struct {
	DocumentType string `json:"document_type"`
	RevisionID   string `json:"revision_id"`
	ContentHash  string `json:"content_hash"`
	Locale       string `json:"locale"`
}

func (h *Handler) LegalConsent(r *ghttp.Request) {
	a, ok := currentApp(r)
	if !ok {
		h.fail(r, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	var in consentRequest
	if !decode(r, &in) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	revision, err := uuid.Parse(in.RevisionID)
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	err = h.service.RecordLegalConsent(r.Context(), bearer(r), a, in.DocumentType, revision, in.ContentHash, in.Locale, r.GetClientIp(), r.UserAgent())
	if errors.Is(err, app.ErrMembershipMissing) {
		h.fail(r, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	if errors.Is(err, app.ErrInvalidInput) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		h.fail(r, http.StatusServiceUnavailable, "CONTENT.CONSENT.UNAVAILABLE", "errors.common.unknown")
		return
	}
	r.Response.WriteJsonExit(httpx.Success[map[string]bool]{Code: "OK", Message: "OK", Data: map[string]bool{"accepted": true}, RequestID: httpx.RequestID(r)})
}

type adminAppRequest struct {
	AppID                         string                        `json:"appid"`
	AppType                       string                        `json:"app_type"`
	Code                          string                        `json:"code"`
	Name                          string                        `json:"name"`
	Description                   string                        `json:"description"`
	Introduction                  string                        `json:"introduction"`
	Remark                        string                        `json:"remark"`
	DefaultLocale                 string                        `json:"default_locale"`
	RegistrationEnabled           bool                          `json:"registration_enabled"`
	RegistrationVerification      string                        `json:"registration_verification_mode"`
	RegistrationVerificationAlias string                        `json:"registration_verification"`
	OwnerType                     string                        `json:"owner_type"`
	OwnerID                       *uuid.UUID                    `json:"owner_id"`
	IconFileID                    *uuid.UUID                    `json:"icon_file_id"`
	Managers                      []uuid.UUID                   `json:"managers"`
	Members                       []uuid.UUID                   `json:"members"`
	ScreenshotFileIDs             []uuid.UUID                   `json:"screenshot_file_ids"`
	Channels                      []app.ApplicationChannel      `json:"channels"`
	StoreListings                 []app.ApplicationStoreListing `json:"store_listings"`
	LockVersion                   int32                         `json:"lock_version"`
}

func (x adminAppRequest) input() app.AdminAppInput {
	verification := x.RegistrationVerification
	if verification == "" {
		verification = x.RegistrationVerificationAlias
	}
	return app.AdminAppInput{AppID: x.AppID, AppType: x.AppType, Code: x.Code, Name: x.Name, Description: x.Description,
		Introduction: x.Introduction, Remark: x.Remark, DefaultLocale: x.DefaultLocale,
		RegistrationEnabled: x.RegistrationEnabled, RegistrationVerification: verification, OwnerType: x.OwnerType,
		OwnerID: x.OwnerID, IconFileID: x.IconFileID, Managers: x.Managers, Members: x.Members,
		ScreenshotFileIDs: x.ScreenshotFileIDs, Channels: x.Channels, StoreListings: x.StoreListings, LockVersion: x.LockVersion}
}
func (h *Handler) AdminApps(r *ghttp.Request) {
	if r.Method == http.MethodGet {
		h.listApps(r)
		return
	}
	var body adminAppRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	item, err := h.service.CreateAdminApp(r.Context(), bearer(r), body.input())
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteHeader(http.StatusCreated)
	r.Response.WriteJsonExit(httpx.Success[app.Application]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}
func (h *Handler) listApps(r *ghttp.Request) {
	filter, ok := adminListFilter(r, "active", "disabled")
	if !ok {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	result, err := h.service.ListAdminApps(r.Context(), bearer(r), filter)
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[pageEnvelope[app.Application]]{Code: "OK", Message: "OK", Data: pageEnvelope[app.Application]{Items: result.Items, Total: result.Total}, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminApp(r *ghttp.Request) {
	id, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if r.Method == http.MethodGet {
		item, err := h.service.GetAdminApp(r.Context(), bearer(r), id)
		if h.adminFailure(r, err) {
			return
		}
		r.Response.WriteJsonExit(httpx.Success[app.Application]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
		return
	}
	if r.Method == http.MethodDelete {
		if h.adminFailure(r, h.service.DeleteAdminApps(r.Context(), bearer(r), []uuid.UUID{id}, httpx.RequestID(r))) {
			return
		}
		r.Response.WriteJsonExit(httpx.Success[map[string]bool]{Code: "OK", Message: "OK", Data: map[string]bool{"deleted": true}, RequestID: httpx.RequestID(r)})
		return
	}
	var body adminAppRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	item, err := h.service.UpdateAdminApp(r.Context(), bearer(r), id, body.input(), nil)
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[app.Application]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}

func (h *Handler) AdminBatchDeleteApps(r *ghttp.Request) {
	var body struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if h.adminFailure(r, h.service.DeleteAdminApps(r.Context(), bearer(r), body.IDs, httpx.RequestID(r))) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[map[string]int]{Code: "OK", Message: "OK", Data: map[string]int{"deleted_count": len(body.IDs)}, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminAppStatus(r *ghttp.Request, status string) {
	id, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var body struct {
		LockVersion int32 `json:"lock_version"`
	}
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	item, err := h.service.SetAdminAppStatus(r.Context(), bearer(r), id, status, body.LockVersion)
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[app.Application]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminEnableApp(r *ghttp.Request)  { h.AdminAppStatus(r, "active") }
func (h *Handler) AdminDisableApp(r *ghttp.Request) { h.AdminAppStatus(r, "disabled") }

type pageRequest struct {
	Slug         string                         `json:"slug"`
	PageType     string                         `json:"page_type"`
	DocumentType string                         `json:"document_type"`
	LockVersion  int32                          `json:"lock_version"`
	Translations map[string]app.PageTranslation `json:"translations"`
	Publish      bool                           `json:"publish"`
}

func (x pageRequest) input(slug string) app.PageInput {
	if slug == "" {
		slug = x.Slug
	}
	pageType := x.PageType
	if pageType == "" {
		pageType = x.DocumentType
	}
	return app.PageInput{Slug: slug, PageType: pageType, LockVersion: x.LockVersion, Translations: x.Translations}
}

func (h *Handler) AdminPages(r *ghttp.Request) {
	appID, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if r.Method == http.MethodGet {
		filter, ok := adminListFilter(r, "draft", "published", "archived")
		if !ok {
			h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
			return
		}
		result, err := h.service.AdminPages(r.Context(), bearer(r), appID, filter)
		if h.adminFailure(r, err) {
			return
		}
		r.Response.WriteJsonExit(httpx.Success[pageEnvelope[app.AdminPage]]{Code: "OK", Message: "OK", Data: pageEnvelope[app.AdminPage]{Items: result.Items, Total: result.Total}, RequestID: httpx.RequestID(r)})
		return
	}
	var body pageRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	item, err := h.service.SaveAdminPage(r.Context(), bearer(r), appID, body.input(""), body.Publish)
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[app.AdminPage]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminPage(r *ghttp.Request) {
	appID, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	slug := r.Get("slug").String()
	if r.Method == http.MethodPatch {
		var body pageRequest
		if !decode(r, &body) {
			h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
			return
		}
		item, err := h.service.SaveAdminPage(r.Context(), bearer(r), appID, body.input(slug), false)
		if h.adminFailure(r, err) {
			return
		}
		r.Response.WriteJsonExit(httpx.Success[app.AdminPage]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
		return
	}
	lockVersion, err := strconv.Atoi(r.GetQuery("lock_version").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if h.adminFailure(r, h.service.DeleteAdminPage(r.Context(), bearer(r), appID, slug, int32(lockVersion))) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[map[string]bool]{Code: "OK", Message: "OK", Data: map[string]bool{"deleted": true}, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminPublishPage(r *ghttp.Request) {
	appID, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var body struct {
		LockVersion int32 `json:"lock_version"`
	}
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	item, err := h.service.PublishAdminPage(r.Context(), bearer(r), appID, r.Get("slug").String(), body.LockVersion)
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[app.AdminPage]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminUsers(r *ghttp.Request) {
	appID, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	filter, ok := adminListFilter(r, "pending_verification", "active", "disabled")
	if !ok {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	result, err := h.service.AdminUsers(r.Context(), bearer(r), appID, filter)
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[pageEnvelope[app.AdminAppUser]]{Code: "OK", Message: "OK", Data: pageEnvelope[app.AdminAppUser]{Items: result.Items, Total: result.Total}, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminUser(r *ghttp.Request) {
	appID, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	userID, err := uuid.Parse(r.Get("user_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	item, err := h.service.GetAdminUser(r.Context(), bearer(r), appID, userID)
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[app.AdminAppUser]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}

type adminUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Locale      string `json:"locale"`
	Password    string `json:"password"`
}
type adminUserUpdateRequest struct {
	DisplayName *string `json:"display_name"`
	LockVersion int32   `json:"lock_version"`
}
type resetUserPasswordRequest struct {
	NewPassword string `json:"new_password"`
	LockVersion int32  `json:"lock_version"`
}
type adminUserActionRequest struct {
	LockVersion int32 `json:"lock_version"`
}

func (h *Handler) AdminCreateUser(r *ghttp.Request) {
	appID, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var body adminUserRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	item, err := h.service.CreateAdminUser(r.Context(), bearer(r), appID, app.AdminAppUserInput{Email: body.Email, DisplayName: body.DisplayName, Locale: body.Locale, Password: body.Password})
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteHeader(http.StatusCreated)
	r.Response.WriteJsonExit(httpx.Success[app.AdminAppUser]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminUpdateUser(r *ghttp.Request) {
	appID, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	userID, err := uuid.Parse(r.Get("user_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var body adminUserUpdateRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	item, err := h.service.UpdateAdminUser(r.Context(), bearer(r), appID, userID, body.DisplayName, body.LockVersion)
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[app.AdminAppUser]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminUnlockUser(r *ghttp.Request)         { h.AdminUserCommand(r, "unlock") }
func (h *Handler) AdminRevokeUserSessions(r *ghttp.Request) { h.AdminUserCommand(r, "revoke") }
func (h *Handler) AdminResetUserPassword(r *ghttp.Request) {
	appID, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	userID, err := uuid.Parse(r.Get("user_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var body resetUserPasswordRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	item, err := h.service.ResetAdminUserPassword(r.Context(), bearer(r), appID, userID, body.NewPassword, body.LockVersion)
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[app.AdminAppUser]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminUserCommand(r *ghttp.Request, command string) {
	appID, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	userID, err := uuid.Parse(r.Get("user_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var body adminUserActionRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var item app.AdminAppUser
	if command == "unlock" {
		item, err = h.service.UnlockAdminUser(r.Context(), bearer(r), appID, userID, body.LockVersion)
	} else {
		item, err = h.service.RevokeAdminUserSessions(r.Context(), bearer(r), appID, userID, body.LockVersion)
	}
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[app.AdminAppUser]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminUserStatus(r *ghttp.Request, status string) {
	appID, err := uuid.Parse(r.Get("app_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	userID, err := uuid.Parse(r.Get("user_id").String())
	if err != nil {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var body adminUserActionRequest
	if !decode(r, &body) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	item, err := h.service.SetAdminUserStatus(r.Context(), bearer(r), appID, userID, status, body.LockVersion)
	if h.adminFailure(r, err) {
		return
	}
	r.Response.WriteJsonExit(httpx.Success[app.AdminAppUser]{Code: "OK", Message: "OK", Data: item, RequestID: httpx.RequestID(r)})
}
func (h *Handler) AdminEnableUser(r *ghttp.Request)  { h.AdminUserStatus(r, "active") }
func (h *Handler) AdminDisableUser(r *ghttp.Request) { h.AdminUserStatus(r, "disabled") }

func adminListFilter(r *ghttp.Request, statuses ...string) (app.AdminListFilter, bool) {
	filter := app.AdminListFilter{Query: strings.TrimSpace(r.GetQuery("q").String()), Status: strings.TrimSpace(r.GetQuery("status").String()), AppType: strings.TrimSpace(r.GetQuery("app_type").String()), Page: 1, PageSize: 20}
	if len([]rune(filter.Query)) > 160 {
		return filter, false
	}
	if raw := r.GetQuery("page").String(); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return filter, false
		}
		filter.Page = int32(value)
	}
	if raw := r.GetQuery("page_size").String(); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return filter, false
		}
		filter.PageSize = int32(value)
	}
	if filter.Status != "" {
		valid := false
		for _, status := range statuses {
			if filter.Status == status {
				valid = true
				break
			}
		}
		if !valid {
			return filter, false
		}
	}
	if filter.AppType != "" && filter.AppType != "uni_app" && filter.AppType != "uni_app_x" {
		return filter, false
	}
	return filter, true
}

func (h *Handler) adminFailure(r *ghttp.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, app.ErrMembershipMissing) {
		h.fail(r, http.StatusForbidden, "ACCESS.PERMISSION.DENIED", "errors.common.forbidden")
		return true
	}
	if errors.Is(err, app.ErrAppNotFound) {
		h.fail(r, http.StatusNotFound, "APP.NOT_FOUND", "errors.common.not_found")
		return true
	}
	if errors.Is(err, app.ErrInvalidInput) {
		h.fail(r, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return true
	}
	if errors.Is(err, app.ErrConflict) {
		h.fail(r, http.StatusConflict, "APP.CONFLICT", "errors.common.conflict")
		return true
	}
	h.fail(r, http.StatusServiceUnavailable, "APP.UNAVAILABLE", "errors.common.unknown")
	return true
}

func currentApp(r *ghttp.Request) (app.Application, bool) {
	value, ok := r.GetCtxVar(appContextKey).Val().(app.Application)
	return value, ok
}
func bearer(r *ghttp.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
func decode(r *ghttp.Request, out any) bool {
	raw := r.GetBody()
	return len(raw) > 0 && json.Unmarshal(raw, out) == nil
}
func (h *Handler) fail(r *ghttp.Request, status int, code, key string) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
}
