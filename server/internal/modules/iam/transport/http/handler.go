package http

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

const (
	refreshCookieName = "ak_admin_refresh"
	csrfCookieName    = "ak_admin_csrf"
	adminAudience     = "ak-admin"
	mobileAudience    = "ak-mobile"
)

// MobileLogin issues bearer and refresh tokens in the response body.  Unlike the
// browser-only admin flow, a native client stores its refresh token in the
// platform secure store and therefore must not receive an HTTP cookie.
func (handler *Handler) MobileLogin(request *ghttp.Request) {
	var body loginRequest
	if err := decodeSingleJSON(request, &body); err != nil || strings.TrimSpace(body.Email) == "" || body.Password == "" {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	appID, appErr := uuid.Parse(strings.TrimSpace(request.Header.Get("X-AppID")))
	if appErr != nil {
		handler.writeError(request, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	tokens, err := handler.auth.Login(request.Context(), application.LoginInput{
		Email: body.Email, Password: body.Password, Audience: mobileAudience, AppID: &appID, Client: clientMetadata(request),
	})
	if errors.Is(err, application.ErrInvalidCredentials) || errors.Is(err, application.ErrCaptchaRequired) {
		handler.writeError(request, http.StatusUnauthorized, "IAM.AUTH.INVALID_CREDENTIALS", "errors.iam.auth.invalid_credentials")
		return
	}
	if errors.Is(err, application.ErrDeviceValidation) {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
		return
	}
	handler.writeMobileSession(request, tokens)
}

type mobileRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (handler *Handler) MobileRefresh(request *ghttp.Request) {
	var body mobileRefreshRequest
	if err := decodeSingleJSON(request, &body); err != nil || strings.TrimSpace(body.RefreshToken) == "" {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	appID, appErr := uuid.Parse(strings.TrimSpace(request.Header.Get("X-AppID")))
	if appErr != nil {
		handler.writeError(request, http.StatusBadRequest, "APP.HEADER.REQUIRED", "errors.validation.failed")
		return
	}
	tokens, err := handler.auth.Refresh(request.Context(), body.RefreshToken, mobileAudience, clientMetadata(request))
	if err != nil {
		code, key := "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized"
		if errors.Is(err, domain.ErrRefreshReused) {
			code, key = "IAM.SESSION.REFRESH_REUSED", "errors.iam.auth.refresh_reused"
		}
		handler.writeError(request, http.StatusUnauthorized, code, key)
		return
	}
	if tokens.AppID == nil || *tokens.AppID != appID {
		handler.writeError(request, http.StatusUnauthorized, "APP.SESSION.MISMATCH", "errors.common.unauthorized")
		return
	}
	handler.writeMobileSession(request, tokens)
}

func (handler *Handler) MobileLogout(request *ghttp.Request) {
	if err := handler.auth.Logout(request.Context(), bearerToken(request.Header.Get("Authorization")), mobileAudience); err != nil {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteJsonExit(httpx.Success[map[string]bool]{Code: "OK", Message: handler.catalog.Translate(httpx.Locale(request), "messages.auth.signed_out", nil), Data: map[string]bool{"signed_out": true}, RequestID: httpx.RequestID(request)})
}

func (handler *Handler) MobileContext(request *ghttp.Request) {
	contextValue, err := handler.auth.Context(request.Context(), bearerToken(request.Header.Get("Authorization")), mobileAudience)
	if err != nil {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.Header().Set("Vary", "Authorization, Accept-Language")
	request.Response.WriteJsonExit(httpx.Success[map[string]any]{Code: "OK", Message: "OK", RequestID: httpx.RequestID(request), Data: map[string]any{
		"user":          map[string]any{"id": contextValue.User.ID, "email": contextValue.User.Email, "display_name": contextValue.User.DisplayName, "locale": contextValue.User.Locale, "time_zone": contextValue.TimeZone, "avatar_url": avatarURLForUser(contextValue.User)},
		"active_tenant": map[string]any{"id": contextValue.Tenant.ID, "code": contextValue.Tenant.Code, "name": contextValue.Tenant.Name},
		"roles":         contextValue.Roles, "permissions": contextValue.Permissions, "feature_flags": handler.featureFlags, "server_time": time.Now().UTC(),
	}})
}

func (handler *Handler) MobileMe(request *ghttp.Request)       { handler.mobileMe(request, false) }
func (handler *Handler) MobileUpdateMe(request *ghttp.Request) { handler.mobileMe(request, true) }

func (handler *Handler) mobileMe(request *ghttp.Request, update bool) {
	if !update {
		contextValue, err := handler.auth.Context(request.Context(), bearerToken(request.Header.Get("Authorization")), mobileAudience)
		if err != nil {
			handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
			return
		}
		handler.writeSelfProfile(request, contextValue.User, "OK")
		return
	}
	input, err := decodeSelfProfileUpdate(request)
	if err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	input.RequestID, input.Client = httpx.RequestID(request), clientMetadata(request)
	user, err := handler.auth.UpdateSelfProfile(request.Context(), bearerToken(request.Header.Get("Authorization")), mobileAudience, input)
	if errors.Is(err, application.ErrProfileValidation) {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	handler.writeSelfProfile(request, user, handler.catalog.Translate(httpx.Locale(request), "messages.profile.updated", nil))
}

func (handler *Handler) MobileSelfSessions(request *ghttp.Request) {
	handler.mobileSelfSessions(request)
}
func (handler *Handler) mobileSelfSessions(request *ghttp.Request) {
	rows, err := handler.auth.ListSelfSessions(request.Context(), bearerToken(request.Header.Get("Authorization")), mobileAudience)
	if err != nil {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	data := make([]selfSessionResponse, 0, len(rows))
	for _, row := range rows {
		var ip *string
		if row.IPAddress != nil {
			value := row.IPAddress.String()
			ip = &value
		}
		data = append(data, selfSessionResponse{ID: row.ID.String(), Audience: row.Audience, Status: row.Status, IPAddress: ip, UserAgent: row.UserAgent, LastSeenAt: row.LastSeenAt.UTC().Format(time.RFC3339Nano), AbsoluteExpiresAt: row.AbsoluteExpiresAt.UTC().Format(time.RFC3339Nano), CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339Nano), Current: row.Current})
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteJsonExit(httpx.Success[[]selfSessionResponse]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(request)})
}

func (handler *Handler) MobileRevokeSelfSession(request *ghttp.Request) {
	id, err := uuid.Parse(request.Get("id").String())
	if err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	current, err := handler.auth.RevokeSelfSession(request.Context(), bearerToken(request.Header.Get("Authorization")), mobileAudience, application.RevokeSelfSessionInput{SessionID: id, RequestID: httpx.RequestID(request), Client: clientMetadata(request)})
	if errors.Is(err, domain.ErrSessionNotFound) {
		handler.writeError(request, http.StatusNotFound, "IAM.SESSION.NOT_FOUND", "errors.iam.session.not_found")
		return
	}
	if err != nil {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	request.Response.WriteJsonExit(httpx.Success[map[string]bool]{Code: "OK", Message: handler.catalog.Translate(httpx.Locale(request), "messages.session.revoked", nil), Data: map[string]bool{"revoked": true, "current_session": current}, RequestID: httpx.RequestID(request)})
}

func (handler *Handler) MobileSelfDevices(request *ghttp.Request) {
	rows, err := handler.auth.ListSelfDevices(request.Context(), bearerToken(request.Header.Get("Authorization")), mobileAudience)
	if err != nil {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	data := make([]selfDeviceResponse, 0, len(rows))
	for _, row := range rows {
		var ip, seen *string
		if row.LastIP != nil {
			value := row.LastIP.String()
			ip = &value
		}
		if row.LastSeenAt != nil {
			value := row.LastSeenAt.UTC().Format(time.RFC3339Nano)
			seen = &value
		}
		data = append(data, selfDeviceResponse{ID: row.ID.String(), Platform: row.Platform, DeviceName: row.DeviceName, Model: row.Model, OSVersion: row.OSVersion, AppVersion: row.AppVersion, LastIP: ip, LastSeenAt: seen, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339Nano), LatestUserAgent: row.LatestUserAgent, ActiveSessionCount: row.ActiveSessionCount, Current: row.Current})
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteJsonExit(httpx.Success[[]selfDeviceResponse]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(request)})
}

func (handler *Handler) MobileRemoveSelfDevice(request *ghttp.Request) {
	id, err := uuid.Parse(request.Get("id").String())
	if err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	current, err := handler.auth.RemoveSelfDevice(request.Context(), bearerToken(request.Header.Get("Authorization")), mobileAudience, application.RemoveSelfDeviceInput{DeviceID: id, RequestID: httpx.RequestID(request), Client: clientMetadata(request)})
	if errors.Is(err, domain.ErrDeviceNotFound) {
		handler.writeError(request, http.StatusNotFound, "IAM.DEVICE.NOT_FOUND", "errors.iam.device.not_found")
		return
	}
	if err != nil {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	request.Response.WriteJsonExit(httpx.Success[map[string]bool]{Code: "OK", Message: handler.catalog.Translate(httpx.Locale(request), "messages.device.removed", nil), Data: map[string]bool{"removed": true, "current_device": current}, RequestID: httpx.RequestID(request)})
}

func (handler *Handler) MobileChangeSelfPassword(request *ghttp.Request) {
	var body passwordChangeRequest
	if err := decodeSingleJSON(request, &body); err != nil || body.CurrentPassword == "" || body.NewPassword == "" {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	err := handler.auth.ChangeSelfPassword(request.Context(), bearerToken(request.Header.Get("Authorization")), mobileAudience, application.ChangeSelfPasswordInput{CurrentPassword: body.CurrentPassword, NewPassword: body.NewPassword, RequestID: httpx.RequestID(request), Client: clientMetadata(request)})
	if errors.Is(err, application.ErrCurrentPassword) {
		handler.writeError(request, http.StatusUnprocessableEntity, "IAM.PASSWORD.CURRENT_INVALID", "errors.iam.password.current_invalid")
		return
	}
	if errors.Is(err, application.ErrPasswordReused) || errors.Is(err, domain.ErrPasswordChanged) {
		handler.writeError(request, http.StatusConflict, "IAM.PASSWORD.CHANGED", "errors.common.conflict")
		return
	}
	if err != nil {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteJsonExit(httpx.Success[map[string]bool]{Code: "OK", Message: handler.catalog.Translate(httpx.Locale(request), "messages.auth.password_changed", nil), Data: map[string]bool{"changed": true, "other_sessions_revoked": true}, RequestID: httpx.RequestID(request)})
}

type Handler struct {
	auth          *application.AuthService
	catalog       *i18n.Catalog
	allowedOrigin string
	secureCookies bool
	featureFlags  map[string]bool
}

type loginRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
}

type loginCaptchaRequest struct {
	Email string `json:"email"`
}

type loginCaptchaResponse struct {
	CaptchaID    string `json:"captcha_id"`
	ImageBase64  string `json:"image_base64"`
	MimeType     string `json:"mime_type"`
	ExpiresInSec int64  `json:"expires_in_seconds"`
}

type switchTenantRequest struct {
	TenantID string `json:"tenant_id"`
}

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Locale      string `json:"locale"`
	AcceptTerms bool   `json:"accept_terms"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	CSRFToken   string `json:"csrf_token"`
}

type csrfTokenResponse struct {
	CSRFToken string `json:"csrf_token"`
}

type mobileTokenResponse struct {
	AccessToken           string `json:"access_token"`
	TokenType             string `json:"token_type"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	SessionID             string `json:"session_id"`
	AppID                 string `json:"app_id"`
}

type selfProfileResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	Locale      string  `json:"locale"`
	TimeZone    string  `json:"time_zone"`
	AvatarURL   *string `json:"avatar_url"`
}

type selfSessionResponse struct {
	ID                string  `json:"id"`
	Audience          string  `json:"audience"`
	Status            string  `json:"status"`
	IPAddress         *string `json:"ip_address"`
	UserAgent         string  `json:"user_agent"`
	LastSeenAt        string  `json:"last_seen_at"`
	AbsoluteExpiresAt string  `json:"absolute_expires_at"`
	CreatedAt         string  `json:"created_at"`
	Current           bool    `json:"current"`
}

type selfDeviceResponse struct {
	ID                 string  `json:"id"`
	Platform           string  `json:"platform"`
	DeviceName         string  `json:"device_name"`
	Model              string  `json:"model"`
	OSVersion          string  `json:"os_version"`
	AppVersion         string  `json:"app_version"`
	LastIP             *string `json:"last_ip"`
	LastSeenAt         *string `json:"last_seen_at"`
	CreatedAt          string  `json:"created_at"`
	LatestUserAgent    string  `json:"latest_user_agent"`
	ActiveSessionCount int64   `json:"active_session_count"`
	Current            bool    `json:"current"`
}

func NewHandler(auth *application.AuthService, catalog *i18n.Catalog, allowedOrigin string, secureCookies bool, featureFlags map[string]bool) *Handler {
	return &Handler{auth: auth, catalog: catalog, allowedOrigin: allowedOrigin, secureCookies: secureCookies, featureFlags: featureFlags}
}

func (handler *Handler) Login(request *ghttp.Request) {
	var body loginRequest
	if err := decodeSingleJSON(request, &body); err != nil || strings.TrimSpace(body.Email) == "" || body.Password == "" {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var captchaID *uuid.UUID
	if strings.TrimSpace(body.CaptchaID) != "" {
		parsed, err := uuid.Parse(body.CaptchaID)
		if err != nil {
			handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
			return
		}
		captchaID = &parsed
	}
	tokens, err := handler.auth.Login(request.Context(), application.LoginInput{
		Email: body.Email, Password: body.Password, Audience: adminAudience, Client: clientMetadata(request),
		CaptchaID: captchaID, CaptchaAnswer: body.CaptchaAnswer,
	})
	switch {
	case errors.Is(err, application.ErrDeviceValidation):
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
	case errors.Is(err, application.ErrCaptchaRequired):
		handler.writeError(request, http.StatusUnauthorized, "IAM.AUTH.CAPTCHA_REQUIRED", "errors.iam.auth.captcha_required")
	case errors.Is(err, application.ErrCaptchaInvalid):
		handler.writeError(request, http.StatusUnprocessableEntity, "IAM.AUTH.CAPTCHA_INVALID", "errors.iam.auth.captcha_invalid")
	case errors.Is(err, application.ErrInvalidCredentials):
		handler.writeError(request, http.StatusUnauthorized, "IAM.AUTH.INVALID_CREDENTIALS", "errors.iam.auth.invalid_credentials")
	case err != nil:
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
	default:
		handler.writeSession(request, tokens)
	}
}

func (handler *Handler) LoginCaptcha(request *ghttp.Request) {
	var body loginCaptchaRequest
	if err := decodeSingleJSON(request, &body); err != nil || strings.TrimSpace(body.Email) == "" {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	challenge, err := handler.auth.CreateLoginCaptcha(request.Context(), body.Email, adminAudience, clientMetadata(request))
	switch {
	case errors.Is(err, application.ErrProfileValidation):
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
	case err != nil:
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
	default:
		request.Response.Header().Set("Cache-Control", "no-store")
		request.Response.WriteJsonExit(httpx.Success[loginCaptchaResponse]{
			Code: "OK", Message: "OK", RequestID: httpx.RequestID(request),
			Data: loginCaptchaResponse{
				CaptchaID: challenge.ID.String(), ImageBase64: challenge.ImageBase64,
				MimeType: challenge.MimeType, ExpiresInSec: challenge.ExpiresInSec,
			},
		})
	}
}

func (handler *Handler) SwitchTenant(request *ghttp.Request) {
	if !handler.featureFlags["multi_tenant"] {
		handler.writeError(request, http.StatusNotFound, "COMMON.NOT_FOUND", "errors.common.not_found")
		return
	}
	var body switchTenantRequest
	if err := decodeSingleJSON(request, &body); err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	tenantID, err := uuid.Parse(body.TenantID)
	if err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	tokens, err := handler.auth.SwitchTenant(request.Context(), bearerToken(request.Header.Get("Authorization")), application.SwitchTenantInput{
		TenantID: tenantID, Audience: adminAudience, Client: clientMetadata(request),
	})
	switch {
	case errors.Is(err, application.ErrTenantUnavailable):
		handler.writeError(request, http.StatusForbidden, "IAM.TENANT.UNAVAILABLE", "errors.common.forbidden")
	case errors.Is(err, application.ErrInvalidAccessToken):
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
	case err != nil:
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
	default:
		handler.writeSession(request, tokens)
	}
}

func (handler *Handler) Register(request *ghttp.Request) {
	var body registerRequest
	if err := decodeSingleJSON(request, &body); err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	err := handler.auth.Register(request.Context(), application.RegisterInput{
		Email: body.Email, DisplayName: body.DisplayName, Password: body.Password,
		Locale: body.Locale, AcceptTerms: body.AcceptTerms, RequestID: httpx.RequestID(request),
		Client: clientMetadata(request),
	})
	switch {
	case errors.Is(err, application.ErrFeatureDisabled):
		handler.writeError(request, http.StatusNotFound, "COMMON.NOT_FOUND", "errors.common.not_found")
	case errors.Is(err, application.ErrRegistrationValidation), errors.Is(err, application.ErrPasswordValidation):
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
	case err != nil:
		handler.writeError(request, http.StatusServiceUnavailable, "IAM.REGISTRATION.UNAVAILABLE", "errors.iam.registration.unavailable")
	default:
		request.Response.WriteHeader(http.StatusAccepted)
		request.Response.WriteJsonExit(httpx.Success[map[string]bool]{
			Code: "OK", Message: "OK", Data: map[string]bool{"accepted": true}, RequestID: httpx.RequestID(request),
		})
	}
}

func (handler *Handler) ForgotPassword(request *ghttp.Request) {
	var body forgotPasswordRequest
	if err := decodeSingleJSON(request, &body); err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	retryAfter, err := handler.auth.ForgotPassword(request.Context(), application.ForgotPasswordInput{
		Email: body.Email, RequestID: httpx.RequestID(request), Client: clientMetadata(request),
	})
	switch {
	case errors.Is(err, application.ErrFeatureDisabled):
		handler.writeError(request, http.StatusNotFound, "COMMON.NOT_FOUND", "errors.common.not_found")
	case errors.Is(err, application.ErrRegistrationValidation):
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
	case err != nil:
		handler.writeError(request, http.StatusServiceUnavailable, "IAM.PASSWORD.RECOVERY_UNAVAILABLE", "errors.iam.password.recovery_unavailable")
	default:
		request.Response.Header().Set("Retry-After", "60")
		request.Response.WriteHeader(http.StatusAccepted)
		request.Response.WriteJsonExit(httpx.Success[map[string]any]{
			Code: "OK", Message: "OK",
			Data:      map[string]any{"accepted": true, "retry_after_seconds": retryAfter},
			RequestID: httpx.RequestID(request),
		})
	}
}

func (handler *Handler) ResetPassword(request *ghttp.Request) {
	var body resetPasswordRequest
	if err := decodeSingleJSON(request, &body); err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	err := handler.auth.ResetPassword(request.Context(), application.ResetPasswordInput{
		Token: body.Token, NewPassword: body.NewPassword, RequestID: httpx.RequestID(request),
		Client: clientMetadata(request),
	})
	switch {
	case errors.Is(err, application.ErrFeatureDisabled):
		handler.writeError(request, http.StatusNotFound, "COMMON.NOT_FOUND", "errors.common.not_found")
	case errors.Is(err, application.ErrResetTokenInvalid), errors.Is(err, domain.ErrPasswordChanged):
		handler.writeError(request, http.StatusUnprocessableEntity, "IAM.PASSWORD.RESET_TOKEN_INVALID", "errors.iam.password.reset_token_invalid")
	case errors.Is(err, application.ErrPasswordReused):
		handler.writeError(request, http.StatusUnprocessableEntity, "IAM.PASSWORD.REUSED", "errors.iam.password.reused")
	case errors.Is(err, application.ErrPasswordValidation):
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
	case err != nil:
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
	default:
		request.Response.WriteJsonExit(httpx.Success[map[string]bool]{
			Code: "OK", Message: "OK", Data: map[string]bool{"reset": true}, RequestID: httpx.RequestID(request),
		})
	}
}

// CSRFToken exposes only the double-submit token already bound to the current
// same-site Admin refresh cookie. It lets a freshly loaded SPA recover its
// in-memory access token without widening the refresh cookie path or persisting
// bearer credentials in browser storage.
func (handler *Handler) CSRFToken(request *ghttp.Request) {
	refreshCookie := request.Cookie.Get(refreshCookieName)
	csrfCookie := request.Cookie.Get(csrfCookieName)
	if refreshCookie == nil || refreshCookie.String() == "" || csrfCookie == nil || csrfCookie.String() == "" {
		handler.clearSessionCookies(request)
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.Header().Set("Vary", "Cookie, Accept-Language")
	request.Response.WriteJsonExit(httpx.Success[csrfTokenResponse]{
		Code: "OK", Message: "OK", RequestID: httpx.RequestID(request),
		Data: csrfTokenResponse{CSRFToken: csrfCookie.String()},
	})
}

func (handler *Handler) Refresh(request *ghttp.Request) {
	if !handler.validCSRF(request) {
		handler.writeError(request, http.StatusForbidden, "AUTH.CSRF.INVALID", "errors.common.forbidden")
		return
	}
	cookieValue := request.Cookie.Get(refreshCookieName)
	if cookieValue == nil || cookieValue.String() == "" {
		handler.clearSessionCookies(request)
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	tokens, err := handler.auth.Refresh(request.Context(), cookieValue.String(), adminAudience, clientMetadata(request))
	if err != nil {
		handler.clearSessionCookies(request)
		if errors.Is(err, domain.ErrRefreshReused) {
			handler.writeError(request, http.StatusUnauthorized, "IAM.SESSION.REFRESH_REUSED", "errors.iam.auth.refresh_reused")
			return
		}
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	handler.writeSession(request, tokens)
}

func (handler *Handler) Logout(request *ghttp.Request) {
	if !handler.validCSRF(request) {
		handler.writeError(request, http.StatusForbidden, "AUTH.CSRF.INVALID", "errors.common.forbidden")
		return
	}
	accessToken := bearerToken(request.Header.Get("Authorization"))
	if accessToken == "" || handler.auth.Logout(request.Context(), accessToken, adminAudience) != nil {
		handler.clearSessionCookies(request)
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	handler.clearSessionCookies(request)
	request.Response.WriteJsonExit(httpx.Success[map[string]bool]{
		Code: "OK", Message: handler.catalog.Translate(httpx.Locale(request), "messages.auth.signed_out", nil),
		Data: map[string]bool{"signed_out": true}, RequestID: httpx.RequestID(request),
	})
}

func (handler *Handler) Context(request *ghttp.Request) {
	accessToken := bearerToken(request.Header.Get("Authorization"))
	contextValue, err := handler.auth.Context(request.Context(), accessToken, adminAudience)
	if errors.Is(err, application.ErrInvalidAccessToken) {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	if err != nil {
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
		return
	}
	request.Response.Header().Set("Vary", "Authorization, Accept-Language")
	menus := make([]map[string]any, 0, len(contextValue.Menus))
	for _, menu := range contextValue.Menus {
		menus = append(menus, map[string]any{
			"id": menu.ID, "parent_id": menu.ParentID, "code": menu.Code, "i18n_key": menu.I18nKey,
			"title": menu.Title, "type": menu.Type, "path": menu.RoutePath,
			"component_key": menu.ComponentKey, "icon": menu.Icon, "affix": menu.Affix,
			"sort": menu.SortOrder, "feature_flag": menu.FeatureFlag,
		})
	}
	availableTenants := make([]map[string]any, 0, len(contextValue.AvailableTenants))
	for _, tenant := range contextValue.AvailableTenants {
		availableTenants = append(availableTenants, map[string]any{"id": tenant.ID, "code": tenant.Code, "name": tenant.Name, "status": tenant.Status})
	}
	request.Response.WriteJsonExit(httpx.Success[map[string]any]{
		Code: "OK", Message: "OK", RequestID: httpx.RequestID(request),
		Data: map[string]any{
			"user":              map[string]any{"id": contextValue.User.ID, "email": contextValue.User.Email, "display_name": contextValue.User.DisplayName, "locale": contextValue.User.Locale, "time_zone": contextValue.TimeZone, "avatar_url": avatarURLForUser(contextValue.User)},
			"active_tenant":     map[string]any{"id": contextValue.Tenant.ID, "code": contextValue.Tenant.Code, "name": contextValue.Tenant.Name},
			"available_tenants": availableTenants,
			"roles":             contextValue.Roles, "permissions": contextValue.Permissions,
			"menus": menus, "feature_flags": handler.featureFlags, "menu_revision": 1, "permission_revision": 1,
			"server_time": time.Now().UTC(),
		},
	})
}

func (handler *Handler) Me(request *ghttp.Request) {
	accessToken := bearerToken(request.Header.Get("Authorization"))
	contextValue, err := handler.auth.Context(request.Context(), accessToken, adminAudience)
	if err != nil {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	handler.writeSelfProfile(request, contextValue.User, "OK")
}

func (handler *Handler) UpdateMe(request *ghttp.Request) {
	input, err := decodeSelfProfileUpdate(request)
	if err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	input.RequestID = httpx.RequestID(request)
	input.Client = clientMetadata(request)
	user, err := handler.auth.UpdateSelfProfile(
		request.Context(), bearerToken(request.Header.Get("Authorization")), adminAudience, input,
	)
	if errors.Is(err, application.ErrProfileValidation) {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if errors.Is(err, application.ErrInvalidAccessToken) {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	if err != nil {
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
		return
	}
	handler.writeSelfProfile(request, user, handler.catalog.Translate(httpx.Locale(request), "messages.profile.updated", nil))
}

func (handler *Handler) SelfSessions(request *ghttp.Request) {
	rows, err := handler.auth.ListSelfSessions(
		request.Context(), bearerToken(request.Header.Get("Authorization")), adminAudience,
	)
	if errors.Is(err, application.ErrInvalidAccessToken) {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	if err != nil {
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
		return
	}
	data := make([]selfSessionResponse, 0, len(rows))
	for _, row := range rows {
		var ipAddress *string
		if row.IPAddress != nil {
			value := row.IPAddress.String()
			ipAddress = &value
		}
		data = append(data, selfSessionResponse{
			ID: row.ID.String(), Audience: row.Audience, Status: row.Status, IPAddress: ipAddress,
			UserAgent: row.UserAgent, LastSeenAt: row.LastSeenAt.UTC().Format(time.RFC3339Nano),
			AbsoluteExpiresAt: row.AbsoluteExpiresAt.UTC().Format(time.RFC3339Nano),
			CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339Nano), Current: row.Current,
		})
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.Header().Set("Vary", "Authorization, Accept-Language")
	request.Response.WriteJsonExit(httpx.Success[[]selfSessionResponse]{
		Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(request),
	})
}

func (handler *Handler) RevokeSelfSession(request *ghttp.Request) {
	sessionID, err := uuid.Parse(request.Get("id").String())
	if err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	current, err := handler.auth.RevokeSelfSession(
		request.Context(), bearerToken(request.Header.Get("Authorization")), adminAudience,
		application.RevokeSelfSessionInput{
			SessionID: sessionID, RequestID: httpx.RequestID(request), Client: clientMetadata(request),
		},
	)
	if errors.Is(err, application.ErrInvalidAccessToken) {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	if errors.Is(err, domain.ErrSessionNotFound) {
		handler.writeError(request, http.StatusNotFound, "IAM.SESSION.NOT_FOUND", "errors.iam.session.not_found")
		return
	}
	if errors.Is(err, application.ErrSessionValidation) {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err != nil {
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
		return
	}
	if current {
		handler.clearSessionCookies(request)
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteJsonExit(httpx.Success[map[string]bool]{
		Code: "OK", Message: handler.catalog.Translate(httpx.Locale(request), "messages.session.revoked", nil),
		Data: map[string]bool{"revoked": true, "current_session": current}, RequestID: httpx.RequestID(request),
	})
}

func (handler *Handler) SelfDevices(request *ghttp.Request) {
	rows, err := handler.auth.ListSelfDevices(
		request.Context(), bearerToken(request.Header.Get("Authorization")), adminAudience,
	)
	if errors.Is(err, application.ErrInvalidAccessToken) {
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	}
	if err != nil {
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
		return
	}
	data := make([]selfDeviceResponse, 0, len(rows))
	for _, row := range rows {
		var lastIP *string
		if row.LastIP != nil {
			value := row.LastIP.String()
			lastIP = &value
		}
		var lastSeenAt *string
		if row.LastSeenAt != nil {
			value := row.LastSeenAt.UTC().Format(time.RFC3339Nano)
			lastSeenAt = &value
		}
		data = append(data, selfDeviceResponse{
			ID: row.ID.String(), Platform: row.Platform, DeviceName: row.DeviceName,
			Model: row.Model, OSVersion: row.OSVersion, AppVersion: row.AppVersion,
			LastIP: lastIP, LastSeenAt: lastSeenAt,
			CreatedAt:       row.CreatedAt.UTC().Format(time.RFC3339Nano),
			LatestUserAgent: row.LatestUserAgent, ActiveSessionCount: row.ActiveSessionCount,
			Current: row.Current,
		})
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.Header().Set("Vary", "Authorization, Accept-Language")
	request.Response.WriteJsonExit(httpx.Success[[]selfDeviceResponse]{
		Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(request),
	})
}

func (handler *Handler) RemoveSelfDevice(request *ghttp.Request) {
	deviceID, err := uuid.Parse(request.Get("id").String())
	if err != nil {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	current, err := handler.auth.RemoveSelfDevice(
		request.Context(), bearerToken(request.Header.Get("Authorization")), adminAudience,
		application.RemoveSelfDeviceInput{
			DeviceID: deviceID, RequestID: httpx.RequestID(request), Client: clientMetadata(request),
		},
	)
	switch {
	case errors.Is(err, application.ErrInvalidAccessToken):
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	case errors.Is(err, domain.ErrDeviceNotFound):
		handler.writeError(request, http.StatusNotFound, "IAM.DEVICE.NOT_FOUND", "errors.iam.device.not_found")
		return
	case errors.Is(err, application.ErrDeviceValidation):
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	case err != nil:
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
		return
	}
	if current {
		handler.clearSessionCookies(request)
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteJsonExit(httpx.Success[map[string]bool]{
		Code: "OK", Message: handler.catalog.Translate(httpx.Locale(request), "messages.device.removed", nil),
		Data: map[string]bool{"removed": true, "current_device": current}, RequestID: httpx.RequestID(request),
	})
}

func (handler *Handler) ChangeSelfPassword(request *ghttp.Request) {
	var body passwordChangeRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || body.CurrentPassword == "" || body.NewPassword == "" {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	err := handler.auth.ChangeSelfPassword(
		request.Context(), bearerToken(request.Header.Get("Authorization")), adminAudience,
		application.ChangeSelfPasswordInput{
			CurrentPassword: body.CurrentPassword, NewPassword: body.NewPassword,
			RequestID: httpx.RequestID(request), Client: clientMetadata(request),
		},
	)
	switch {
	case errors.Is(err, application.ErrInvalidAccessToken):
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return
	case errors.Is(err, application.ErrCurrentPassword):
		handler.writeError(request, http.StatusUnprocessableEntity, "IAM.PASSWORD.CURRENT_INVALID", "errors.iam.password.current_invalid")
		return
	case errors.Is(err, application.ErrPasswordReused):
		handler.writeError(request, http.StatusConflict, "IAM.PASSWORD.REUSED", "errors.iam.password.reused")
		return
	case errors.Is(err, domain.ErrPasswordChanged):
		handler.writeError(request, http.StatusConflict, "IAM.PASSWORD.CHANGED", "errors.common.conflict")
		return
	case errors.Is(err, application.ErrPasswordValidation):
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	case err != nil:
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
		return
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteJsonExit(httpx.Success[map[string]bool]{
		Code: "OK", Message: handler.catalog.Translate(httpx.Locale(request), "messages.auth.password_changed", nil),
		Data: map[string]bool{"changed": true, "other_sessions_revoked": true}, RequestID: httpx.RequestID(request),
	})
}

func (handler *Handler) writeSelfProfile(request *ghttp.Request, user domain.User, message string) {
	avatarURL := avatarURLForUser(user)
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.Header().Set("Vary", "Authorization, Accept-Language")
	request.Response.WriteJsonExit(httpx.Success[selfProfileResponse]{
		Code: "OK", Message: message, RequestID: httpx.RequestID(request),
		Data: selfProfileResponse{
			ID: user.ID.String(), Email: user.Email, DisplayName: user.DisplayName,
			Locale: user.Locale, TimeZone: user.TimeZone, AvatarURL: avatarURL,
		},
	})
}

func avatarURLForUser(user domain.User) *string {
	if user.AvatarFileID == nil {
		return nil
	}
	value := "/me/avatar/content?v=" + user.AvatarFileID.String()
	return &value
}

func (handler *Handler) writeSession(request *ghttp.Request, tokens application.SessionTokens) {
	csrfToken, _, err := application.NewOpaqueToken()
	if err != nil {
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
		return
	}
	request.Cookie.SetHttpCookie(&http.Cookie{
		Name: refreshCookieName, Value: tokens.RefreshToken, Path: "/admin-api/v1/auth",
		Expires: tokens.RefreshTokenExpiresAt, MaxAge: int(time.Until(tokens.RefreshTokenExpiresAt).Seconds()),
		HttpOnly: true, Secure: handler.secureCookies, SameSite: http.SameSiteStrictMode,
	})
	request.Cookie.SetHttpCookie(&http.Cookie{
		Name: csrfCookieName, Value: csrfToken, Path: "/admin-api/v1/auth",
		Expires: tokens.RefreshTokenExpiresAt, MaxAge: int(time.Until(tokens.RefreshTokenExpiresAt).Seconds()),
		HttpOnly: false, Secure: handler.secureCookies, SameSite: http.SameSiteStrictMode,
	})
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteJsonExit(httpx.Success[tokenResponse]{
		Code: "OK", Message: "OK", RequestID: httpx.RequestID(request),
		Data: tokenResponse{AccessToken: tokens.AccessToken, TokenType: "Bearer", ExpiresIn: max(0, int64(time.Until(tokens.AccessTokenExpiresAt).Seconds())), CSRFToken: csrfToken},
	})
}

func (handler *Handler) writeMobileSession(request *ghttp.Request, tokens application.SessionTokens) {
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteJsonExit(httpx.Success[mobileTokenResponse]{
		Code: "OK", Message: "OK", RequestID: httpx.RequestID(request),
		Data: mobileTokenResponse{
			AccessToken: tokens.AccessToken, TokenType: "Bearer",
			ExpiresIn:             max(0, int64(time.Until(tokens.AccessTokenExpiresAt).Seconds())),
			RefreshToken:          tokens.RefreshToken,
			RefreshTokenExpiresIn: max(0, int64(time.Until(tokens.RefreshTokenExpiresAt).Seconds())),
			SessionID:             tokens.SessionID.String(),
			AppID:                 appIDString(tokens.AppID),
		},
	})
}

func appIDString(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func (handler *Handler) validCSRF(request *ghttp.Request) bool {
	if request.Method != http.MethodPost || request.Header.Get("Origin") != handler.allowedOrigin {
		return false
	}
	cookieValue := request.Cookie.Get(csrfCookieName)
	header := request.Header.Get("X-CSRF-Token")
	return cookieValue != nil && cookieValue.String() != "" && len(cookieValue.String()) == len(header) &&
		subtle.ConstantTimeCompare([]byte(cookieValue.String()), []byte(header)) == 1
}

func (handler *Handler) clearSessionCookies(request *ghttp.Request) {
	for _, cookieName := range []string{refreshCookieName, csrfCookieName} {
		request.Cookie.SetHttpCookie(&http.Cookie{
			Name: cookieName, Value: "", Path: "/admin-api/v1/auth", MaxAge: -1,
			HttpOnly: cookieName == refreshCookieName, Secure: handler.secureCookies, SameSite: http.SameSiteStrictMode,
		})
	}
}

func (handler *Handler) writeError(request *ghttp.Request, status int, code, messageKey string) {
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Error{
		Error:     httpx.ErrorBody{Code: code, MessageKey: messageKey, Message: handler.catalog.Translate(httpx.Locale(request), messageKey, nil)},
		RequestID: httpx.RequestID(request),
	})
}

func clientMetadata(request *ghttp.Request) application.ClientMetadata {
	var address *netip.Addr
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		if parsed, parseErr := netip.ParseAddr(host); parseErr == nil {
			address = &parsed
		}
	}
	return application.ClientMetadata{
		IPAddress: address, UserAgent: request.UserAgent(), DeviceKey: request.Header.Get("X-AK-Device-Key"),
		RequestID: httpx.RequestID(request),
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func decodeSelfProfileUpdate(request *ghttp.Request) (application.UpdateSelfProfileInput, error) {
	decoder := json.NewDecoder(request.Body)
	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil || len(raw) == 0 {
		return application.UpdateSelfProfileInput{}, errors.New("invalid profile payload")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return application.UpdateSelfProfileInput{}, errors.New("multiple JSON values")
	}
	var result application.UpdateSelfProfileInput
	for key, value := range raw {
		var decoded string
		if err := json.Unmarshal(value, &decoded); err != nil {
			return application.UpdateSelfProfileInput{}, errors.New("invalid profile field")
		}
		switch key {
		case "display_name":
			result.DisplayName = &decoded
		case "locale":
			result.Locale = &decoded
		case "time_zone":
			result.TimeZone = &decoded
		default:
			return application.UpdateSelfProfileInput{}, errors.New("unknown profile field")
		}
	}
	return result, nil
}

func decodeSingleJSON(request *ghttp.Request, destination any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values")
	}
	return nil
}
