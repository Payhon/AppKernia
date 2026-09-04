package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	stdhttp "net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	loginapp "github.com/appkernia/appkernia/server/internal/modules/loginprovider/application"
	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	platformcaptcha "github.com/appkernia/appkernia/server/internal/platform/captcha"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *loginapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *loginapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func token(request *ghttp.Request) string {
	value := strings.TrimSpace(request.Header.Get("Authorization"))
	if len(value) > 7 && strings.EqualFold(value[:7], "Bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return ""
}

func decode(request *ghttp.Request, out any) bool {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil {
		return false
	}
	return errors.Is(decoder.Decode(&struct{}{}), io.EOF)
}

func routeUUID(request *ghttp.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(request.GetRouter(name).String()))
	return id, err == nil
}

func appID(request *ghttp.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(request.Header.Get("X-AppID")))
	return id, err == nil
}

func client(request *ghttp.Request) iamapp.ClientMetadata {
	var address *netip.Addr
	host := request.RemoteAddr
	if parsedHost, _, err := net.SplitHostPort(request.RemoteAddr); err == nil {
		host = parsedHost
	}
	if parsed, err := netip.ParseAddr(host); err == nil {
		address = &parsed
	}
	return iamapp.ClientMetadata{IPAddress: address, UserAgent: request.UserAgent(),
		DeviceKey: request.Header.Get("X-AK-Device-Key"), RequestID: httpx.RequestID(request)}
}

func principal(request *ghttp.Request) login.Principal {
	metadata := client(request)
	ip := ""
	if metadata.IPAddress != nil {
		ip = metadata.IPAddress.String()
	}
	return login.Principal{RequestID: metadata.RequestID, IPAddress: ip, UserAgent: metadata.UserAgent}
}

func (handler *Handler) ok(request *ghttp.Request, status int, data any) {
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(request)})
}

func (handler *Handler) fail(request *ghttp.Request, err error) bool {
	if err == nil {
		return false
	}
	status, code, key := 500, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(err, login.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, login.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, iamapp.ErrInvalidAccessToken):
		status, code, key = 401, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized"
	case errors.Is(err, login.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, login.ErrInUse):
		status, code, key = 409, "LOGIN_PROVIDER.CONFIG.IN_USE", "errors.common.conflict"
	case errors.Is(err, login.ErrConflict):
		status, code, key = 409, "COMMON.CONFLICT", "errors.common.conflict"
	case errors.Is(err, login.ErrProviderUnavailable):
		status, code, key = 503, "IAM.OAUTH.PROVIDER_UNAVAILABLE", "errors.iam.oauth.provider_unavailable"
	case errors.Is(err, login.ErrConfigStale):
		status, code, key = 409, "IAM.OAUTH.CONFIG_STALE", "errors.iam.oauth.config_stale"
	case errors.Is(err, login.ErrFlowExpired):
		status, code, key = 410, "IAM.OAUTH.FLOW_EXPIRED", "errors.common.unauthorized"
	case errors.Is(err, login.ErrFlowConsumed):
		status, code, key = 409, "IAM.OAUTH.FLOW_CONSUMED", "errors.common.conflict"
	case errors.Is(err, login.ErrFlowInvalid), errors.Is(err, login.ErrCallbackInvalid):
		status, code, key = 401, "IAM.OAUTH.CALLBACK_INVALID", "errors.iam.oauth.callback_invalid"
	case errors.Is(err, login.ErrAuthorizationDenied):
		status, code, key = 401, "IAM.OAUTH.AUTHORIZATION_DENIED", "errors.iam.oauth.authorization_denied"
	case errors.Is(err, login.ErrIdentityConflict):
		status, code, key = 409, "IAM.OAUTH.IDENTITY_CONFLICT", "errors.common.conflict"
	case errors.Is(err, login.ErrLinkRequired):
		status, code, key = 409, "ACCOUNT_LINK_REQUIRED", "errors.iam.oauth.account_link_required"
	case errors.Is(err, login.ErrLastLoginMethod):
		status, code, key = 409, "AUTH.LAST_LOGIN_METHOD", "errors.iam.login_method.last"
	case errors.Is(err, login.ErrIdentifierConflict):
		status, code, key = 409, "AUTH.IDENTIFIER.CONFLICT", "errors.common.conflict"
	case errors.Is(err, login.ErrAccountExists):
		status, code, key = 409, "ACCOUNT_ALREADY_EXISTS", "errors.common.conflict"
	case errors.Is(err, login.ErrOTPInvalid):
		status, code, key = 401, "IAM.OTP.INVALID", "errors.iam.otp.invalid"
	case errors.Is(err, login.ErrCaptchaRequired):
		status, code, key = 428, "IAM.CAPTCHA.REQUIRED", "errors.iam.captcha.required"
	case errors.Is(err, login.ErrCaptchaInvalid):
		status, code, key = 422, "IAM.CAPTCHA.INVALID", "errors.iam.captcha.invalid"
	case errors.Is(err, login.ErrCaptchaCooldown):
		request.Response.Header().Set("Retry-After", "2")
		status, code, key = 429, "IAM.CAPTCHA.COOLDOWN", "errors.iam.captcha.cooldown"
	case errors.Is(err, login.ErrCaptchaUnavailable):
		status, code, key = 503, "IAM.CAPTCHA.UNAVAILABLE", "errors.iam.captcha.unavailable"
	case errors.Is(err, login.ErrDeliveryUnavailable):
		status, code, key = 503, "IAM.OTP.DELIVERY_UNAVAILABLE", "errors.iam.otp.delivery_unavailable"
	case errors.Is(err, login.ErrStepUpRequired):
		status, code, key = 428, "IAM.STEP_UP.REQUIRED", "errors.iam.step_up.required"
	case errors.Is(err, login.ErrStepUpInvalid):
		status, code, key = 401, "IAM.STEP_UP.INVALID", "errors.iam.step_up.invalid"
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key,
		Message: handler.catalog.Translate(httpx.Locale(request), key, nil)}, RequestID: httpx.RequestID(request)})
	return true
}

func (handler *Handler) Catalog(request *ghttp.Request) {
	out, err := handler.service.Catalog(request.Context(), token(request))
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, out)
	}
}

func (handler *Handler) Configs(request *ghttp.Request) {
	switch request.Method {
	case stdhttp.MethodGet:
		page, pageErr := strconv.Atoi(request.GetQuery("page", 1).String())
		pageSize, sizeErr := strconv.Atoi(request.GetQuery("page_size", 20).String())
		if pageErr != nil || sizeErr != nil {
			handler.fail(request, login.ErrInvalid)
			return
		}
		out, err := handler.service.ListConfigs(request.Context(), token(request), login.ListFilter{
			Query: request.GetQuery("q").String(), ProviderCode: request.GetQuery("provider_code").String(),
			Status: request.GetQuery("status").String(), Page: int32(page), PageSize: int32(pageSize),
		})
		if !handler.fail(request, err) {
			handler.ok(request, stdhttp.StatusOK, out)
		}
	case stdhttp.MethodPost:
		var body login.ConfigInput
		if !decode(request, &body) {
			handler.fail(request, login.ErrInvalid)
			return
		}
		out, err := handler.service.CreateConfig(request.Context(), token(request), principal(request), body)
		if !handler.fail(request, err) {
			handler.ok(request, stdhttp.StatusCreated, out)
		}
	default:
		handler.fail(request, login.ErrInvalid)
	}
}

func (handler *Handler) Config(request *ghttp.Request) {
	id, ok := routeUUID(request, "id")
	if !ok {
		handler.fail(request, login.ErrInvalid)
		return
	}
	switch request.Method {
	case stdhttp.MethodGet:
		out, err := handler.service.GetConfig(request.Context(), token(request), id)
		if !handler.fail(request, err) {
			handler.ok(request, stdhttp.StatusOK, out)
		}
	case stdhttp.MethodPatch:
		var body login.ConfigInput
		if !decode(request, &body) {
			handler.fail(request, login.ErrInvalid)
			return
		}
		out, err := handler.service.UpdateConfig(request.Context(), token(request), principal(request), id, body)
		if !handler.fail(request, err) {
			handler.ok(request, stdhttp.StatusOK, out)
		}
	case stdhttp.MethodDelete:
		version, err := strconv.Atoi(request.GetQuery("lock_version").String())
		if err != nil {
			handler.fail(request, login.ErrInvalid)
			return
		}
		err = handler.service.DeleteConfig(request.Context(), token(request), principal(request), id, int32(version))
		if !handler.fail(request, err) {
			handler.ok(request, stdhttp.StatusOK, map[string]bool{"deleted": true})
		}
	default:
		handler.fail(request, login.ErrInvalid)
	}
}

type statusInput struct {
	LockVersion int32 `json:"lock_version"`
}

func (handler *Handler) configAction(request *ghttp.Request, action string) {
	id, ok := routeUUID(request, "id")
	var body statusInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	var out login.Config
	var err error
	switch action {
	case "preflight":
		out, err = handler.service.Preflight(request.Context(), token(request), principal(request), id, body.LockVersion)
	case "activate":
		out, err = handler.service.Activate(request.Context(), token(request), principal(request), id, body.LockVersion)
	case "disable":
		out, err = handler.service.Disable(request.Context(), token(request), principal(request), id, body.LockVersion)
	default:
		err = login.ErrInvalid
	}
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, out)
	}
}

func (handler *Handler) Preflight(request *ghttp.Request) { handler.configAction(request, "preflight") }
func (handler *Handler) Activate(request *ghttp.Request)  { handler.configAction(request, "activate") }
func (handler *Handler) Disable(request *ghttp.Request)   { handler.configAction(request, "disable") }

func (handler *Handler) RotateSecret(request *ghttp.Request) {
	id, ok := routeUUID(request, "id")
	var body login.SecretInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.RotateSecret(request.Context(), token(request), principal(request), id, body)
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, out)
	}
}

func (handler *Handler) Bindings(request *ghttp.Request) {
	id, ok := routeUUID(request, "app_id")
	if !ok {
		handler.fail(request, login.ErrInvalid)
		return
	}
	if request.Method == stdhttp.MethodGet {
		out, err := handler.service.ListBindings(request.Context(), token(request), id)
		if !handler.fail(request, err) {
			handler.ok(request, stdhttp.StatusOK, out)
		}
		return
	}
	if request.Method != stdhttp.MethodPut {
		handler.fail(request, login.ErrInvalid)
		return
	}
	var body login.BulkBindingInput
	if !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.ReplaceBindings(request.Context(), token(request), principal(request), id, body)
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, out)
	}
}

func (handler *Handler) Providers(request *ghttp.Request) {
	id, ok := appID(request)
	if !ok {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.MobileProviders(request.Context(), id, request.GetQuery("platform").String(), request.GetQuery("build_variant").String())
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, out)
	}
}

func (handler *Handler) Authorize(request *ghttp.Request) {
	id, ok := appID(request)
	var body login.AuthorizeInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.Authorize(request.Context(), token(request), id, request.GetRouter("provider").String(), request.Header.Get("X-AK-Device-Key"), body)
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, out)
	}
}

type mobileSessionResponse struct {
	AccessToken           string `json:"access_token"`
	TokenType             string `json:"token_type"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	SessionID             string `json:"session_id"`
	AppID                 string `json:"app_id"`
}

func sessionResponse(tokens iamapp.SessionTokens) mobileSessionResponse {
	app := ""
	if tokens.AppID != nil {
		app = tokens.AppID.String()
	}
	return mobileSessionResponse{AccessToken: tokens.AccessToken, TokenType: "Bearer", ExpiresIn: max(0, int64(time.Until(tokens.AccessTokenExpiresAt).Seconds())),
		RefreshToken: tokens.RefreshToken, RefreshTokenExpiresIn: max(0, int64(time.Until(tokens.RefreshTokenExpiresAt).Seconds())),
		SessionID: tokens.SessionID.String(), AppID: app}
}

func (handler *Handler) Callback(request *ghttp.Request) {
	id, ok := appID(request)
	var body login.CallbackInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.Callback(request.Context(), token(request), id, request.GetRouter("provider").String(), request.Header.Get("X-AK-Device-Key"), body, client(request))
	if handler.fail(request, err) {
		return
	}
	data := map[string]any{"mode": out.Mode}
	if out.Account != nil {
		data["oauth_account"] = out.Account
	}
	if out.Session != nil {
		data["session"] = sessionResponse(*out.Session)
	}
	if out.StepUpToken != "" {
		data["step_up_token"], data["step_up_expires_at"] = out.StepUpToken, out.StepUpExpiresAt
	}
	handler.ok(request, stdhttp.StatusOK, data)
}

func (handler *Handler) GitHubBrowserCallback(request *ghttp.Request) {
	providerCode := strings.ToLower(strings.TrimSpace(request.GetRouter("provider").String()))
	descriptor, registered := login.Descriptor(providerCode)
	if !registered || descriptor.AuthorizationKind != "browser_ticket" || providerCode != login.ProviderGitHub {
		handler.fail(request, login.ErrProviderUnavailable)
		return
	}
	out, err := handler.service.GitHubBrowserCallback(request.Context(), request.GetQuery("code").String(), request.GetQuery("state").String())
	if handler.fail(request, err) {
		return
	}
	setBrowserRedirectHeaders(request.Response.Header())
	request.Response.RedirectTo(out.RedirectURI, stdhttp.StatusFound)
}

func setBrowserRedirectHeaders(header stdhttp.Header) {
	header.Set("Cache-Control", "no-store")
	header.Set("Pragma", "no-cache")
	header.Set("Referrer-Policy", "no-referrer")
}

func (handler *Handler) LoginMethods(request *ghttp.Request) {
	id, ok := appID(request)
	if !ok {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.LoginMethods(request.Context(), token(request), id)
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, out)
	}
}

func (handler *Handler) SetPassword(request *ghttp.Request) {
	id, ok := appID(request)
	var body struct {
		NewPassword string `json:"new_password"`
		StepUpToken string `json:"step_up_token"`
	}
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	err := handler.service.SetPassword(request.Context(), token(request), id, body.NewPassword, body.StepUpToken, client(request))
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, map[string]bool{"password_set": true})
	}
}

func (handler *Handler) OAuthAccounts(request *ghttp.Request) {
	id, ok := appID(request)
	if !ok {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.OAuthAccounts(request.Context(), token(request), id)
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, map[string]any{"items": out})
	}
}

type stepTokenInput struct {
	StepUpToken string `json:"step_up_token"`
}

func (handler *Handler) DeleteOAuthAccount(request *ghttp.Request) {
	id, ok := appID(request)
	accountID, accountOK := routeUUID(request, "account_id")
	var body stepTokenInput
	if !ok || !accountOK || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	err := handler.service.DeleteOAuthAccount(request.Context(), token(request), id, accountID, body.StepUpToken, client(request))
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, map[string]bool{"unbound": true})
	}
}

type loginCodeInput struct {
	Email   string                    `json:"email"`
	Mobile  string                    `json:"mobile"`
	Captcha *iamapp.LoginCaptchaInput `json:"captcha,omitempty"`
}

type interactiveCaptchaResponse struct {
	CaptchaID      string                 `json:"captcha_id"`
	CaptchaToken   string                 `json:"captcha_token"`
	Type           platformcaptcha.Type   `json:"type"`
	ExpiresInSec   int64                  `json:"expires_in_seconds"`
	Image          platformcaptcha.Image  `json:"image"`
	PromptImage    *platformcaptcha.Image `json:"prompt_image,omitempty"`
	RequiredPoints int                    `json:"required_points,omitempty"`
	TileImage      *platformcaptcha.Image `json:"tile_image,omitempty"`
	InitialPoint   *platformcaptcha.Point `json:"initial_point,omitempty"`
	ThumbImage     *platformcaptcha.Image `json:"thumb_image,omitempty"`
}

func captchaResponse(value iamapp.LoginCaptcha) interactiveCaptchaResponse {
	return interactiveCaptchaResponse{CaptchaID: value.ID.String(), CaptchaToken: value.Token, Type: value.Type,
		ExpiresInSec: value.ExpiresInSec, Image: value.Image, PromptImage: value.PromptImage, RequiredPoints: value.RequiredPoints,
		TileImage: value.TileImage, InitialPoint: value.InitialPoint, ThumbImage: value.ThumbImage}
}

func (handler *Handler) SMSCaptcha(request *ghttp.Request) {
	id, ok := appID(request)
	var body loginapp.SMSCaptchaInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.CreateSMSCaptcha(request.Context(), token(request), id, body, client(request))
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, captchaResponse(out))
	}
}

func (handler *Handler) SendEmailCode(request *ghttp.Request) {
	handler.sendLoginCode(request, "email")
}
func (handler *Handler) SendMobileCode(request *ghttp.Request) {
	handler.sendLoginCode(request, "mobile")
}

func (handler *Handler) sendLoginCode(request *ghttp.Request, identifierType string) {
	id, ok := appID(request)
	var body loginCodeInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	identifier := body.Email
	if identifierType == "mobile" {
		identifier = body.Mobile
	}
	out, err := handler.service.SendLoginCode(request.Context(), id, identifierType, identifier, string(httpx.Locale(request)), client(request), body.Captcha)
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusAccepted, out)
	}
}

func (handler *Handler) OTPLogin(request *ghttp.Request) {
	id, ok := appID(request)
	var body loginapp.OTPLoginInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.OTPLogin(request.Context(), id, body, client(request))
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, sessionResponse(out))
	}
}

func (handler *Handler) SendRegistrationCode(request *ghttp.Request) {
	id, ok := appID(request)
	var body struct {
		IdentifierType string                    `json:"identifier_type"`
		Identifier     string                    `json:"identifier"`
		Captcha        *iamapp.LoginCaptchaInput `json:"captcha,omitempty"`
	}
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.SendRegistrationCode(request.Context(), id, body.IdentifierType, body.Identifier, string(httpx.Locale(request)), client(request), body.Captcha)
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusAccepted, out)
	}
}

func (handler *Handler) RegisterOTP(request *ghttp.Request) {
	id, ok := appID(request)
	var body loginapp.OTPRegistrationInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.RegisterOTP(request.Context(), id, body, client(request))
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, sessionResponse(out))
	}
}

func (handler *Handler) ForgotPassword(request *ghttp.Request) {
	id, ok := appID(request)
	var body struct {
		IdentifierType string                    `json:"identifier_type"`
		Identifier     string                    `json:"identifier"`
		Email          string                    `json:"email"`
		Captcha        *iamapp.LoginCaptchaInput `json:"captcha,omitempty"`
	}
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	if body.Identifier == "" {
		body.Identifier, body.IdentifierType = body.Email, "email"
	}
	out, err := handler.service.SendPasswordResetCode(request.Context(), id, body.IdentifierType, body.Identifier, string(httpx.Locale(request)), client(request), body.Captcha)
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusAccepted, out)
	}
}

func (handler *Handler) ResetPassword(request *ghttp.Request) {
	id, ok := appID(request)
	var body struct {
		loginapp.PasswordResetInput
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	if body.Identifier == "" {
		body.Identifier, body.IdentifierType = body.Email, "email"
	}
	if body.VerificationCode == "" {
		body.VerificationCode = body.Code
	}
	err := handler.service.ResetPassword(request.Context(), id, body.PasswordResetInput)
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, map[string]bool{"reset": true})
	}
}

func (handler *Handler) IdentifierChallenge(request *ghttp.Request) {
	id, ok := appID(request)
	var body loginapp.IdentifierCodeInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.SendIdentifierCode(request.Context(), token(request), id, request.GetRouter("type").String(), body, client(request))
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusAccepted, out)
	}
}

func (handler *Handler) Identifier(request *ghttp.Request) {
	id, ok := appID(request)
	identifierType := request.GetRouter("type").String()
	if !ok {
		handler.fail(request, login.ErrInvalid)
		return
	}
	if request.Method == stdhttp.MethodPut {
		var body loginapp.IdentifierVerifyInput
		if !decode(request, &body) {
			handler.fail(request, login.ErrInvalid)
			return
		}
		out, err := handler.service.VerifyIdentifier(request.Context(), token(request), id, identifierType, body, client(request))
		if !handler.fail(request, err) {
			handler.ok(request, stdhttp.StatusOK, map[string]any{"identifier": out})
		}
		return
	}
	if request.Method == stdhttp.MethodDelete {
		var body stepTokenInput
		if !decode(request, &body) {
			handler.fail(request, login.ErrInvalid)
			return
		}
		err := handler.service.DeleteIdentifier(request.Context(), token(request), id, identifierType, body.StepUpToken, client(request))
		if !handler.fail(request, err) {
			handler.ok(request, stdhttp.StatusOK, map[string]bool{"unbound": true})
		}
		return
	}
	handler.fail(request, login.ErrInvalid)
}

func (handler *Handler) StepUpCode(request *ghttp.Request) {
	id, ok := appID(request)
	var body loginapp.StepUpCodeInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.SendStepUpCode(request.Context(), token(request), id, body, client(request))
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusAccepted, out)
	}
}

func (handler *Handler) StepUp(request *ghttp.Request) {
	id, ok := appID(request)
	var body loginapp.StepUpInput
	if !ok || !decode(request, &body) {
		handler.fail(request, login.ErrInvalid)
		return
	}
	out, err := handler.service.StepUp(request.Context(), token(request), id, body, client(request))
	if !handler.fail(request, err) {
		handler.ok(request, stdhttp.StatusOK, out)
	}
}
