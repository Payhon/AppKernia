package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	identityapp "github.com/appkernia/appkernia/server/internal/modules/identitysecurity/application"
	identity "github.com/appkernia/appkernia/server/internal/modules/identitysecurity/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
)

type Handler struct {
	service *identityapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *identityapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func (handler *Handler) MFAStatus(request *ghttp.Request) {
	result, err := handler.service.MFAStatus(request.Context(), token(request))
	if !handler.fail(request, err) {
		handler.ok(request, 200, result)
	}
}

func (handler *Handler) EnrollTOTP(request *ghttp.Request) {
	result, err := handler.service.EnrollTOTP(request.Context(), token(request), principal(request))
	if !handler.fail(request, err) {
		handler.ok(request, 201, result)
	}
}

func (handler *Handler) VerifyTOTP(request *ghttp.Request) {
	var input identity.VerifyTOTPInput
	if !decode(request, &input) {
		handler.fail(request, identity.ErrInvalid)
		return
	}
	result, err := handler.service.VerifyTOTP(request.Context(), token(request), principal(request), input)
	if !handler.fail(request, err) {
		handler.ok(request, 200, result)
	}
}

func (handler *Handler) DisableTOTP(request *ghttp.Request) {
	var input identity.StepUpInput
	if !decode(request, &input) {
		handler.fail(request, identity.ErrInvalid)
		return
	}
	err := handler.service.DisableTOTP(request.Context(), token(request), principal(request), input)
	if !handler.fail(request, err) {
		handler.ok(request, 200, map[string]bool{"disabled": true})
	}
}

func (handler *Handler) RotateRecoveryCodes(request *ghttp.Request) {
	var input identity.StepUpInput
	if !decode(request, &input) {
		handler.fail(request, identity.ErrInvalid)
		return
	}
	result, err := handler.service.RotateRecoveryCodes(request.Context(), token(request), principal(request), input)
	if !handler.fail(request, err) {
		handler.ok(request, 200, result)
	}
}

func (handler *Handler) OAuthAccounts(request *ghttp.Request) {
	result, err := handler.service.OAuthAccounts(request.Context(), token(request))
	if !handler.fail(request, err) {
		handler.ok(request, 200, map[string]any{"items": result})
	}
}

func (handler *Handler) StartOAuth(request *ghttp.Request) {
	result, err := handler.service.StartOAuth(request.Context(), token(request), principal(request), request.GetRouter("provider").String())
	if !handler.fail(request, err) {
		handler.ok(request, 201, result)
	}
}

func (handler *Handler) CompleteOAuth(request *ghttp.Request) {
	var input identity.OAuthCompleteInput
	if !decode(request, &input) {
		handler.fail(request, identity.ErrInvalid)
		return
	}
	result, err := handler.service.CompleteOAuth(request.Context(), token(request), principal(request), request.GetRouter("provider").String(), input)
	if !handler.fail(request, err) {
		handler.ok(request, 200, result)
	}
}

func (handler *Handler) DeleteOAuth(request *ghttp.Request) {
	err := handler.service.DeleteOAuth(request.Context(), token(request), principal(request), request.GetRouter("provider").String())
	if !handler.fail(request, err) {
		handler.ok(request, 200, map[string]bool{"unbound": true})
	}
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
	case errors.Is(err, iamapp.ErrInvalidAccessToken):
		status, code, key = 401, "AUTH.TOKEN.INVALID", "errors.common.unauthorized"
	case errors.Is(err, identity.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, identity.ErrInvalid):
		status, code, key = 422, "IAM.MFA.CODE_INVALID", "errors.iam.mfa.code_invalid"
	case errors.Is(err, identity.ErrStepUpRequired):
		status, code, key = 403, "IAM.STEP_UP.REQUIRED", "errors.iam.step_up.required"
	case errors.Is(err, identity.ErrFeatureDisabled), errors.Is(err, identity.ErrProviderDisabled):
		status, code, key = 503, "COMMON.FEATURE_DISABLED", "errors.common.feature_disabled"
	case errors.Is(err, identity.ErrOAuthState):
		status, code, key = 422, "IAM.OAUTH.STATE_INVALID", "errors.iam.oauth.state_invalid"
	case errors.Is(err, identity.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, identity.ErrConflict):
		status, code, key = 409, "IAM.SECURITY.STATE_CONFLICT", "errors.iam.security.state_conflict"
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: handler.catalog.Translate(httpx.Locale(request), key, nil)}, RequestID: httpx.RequestID(request)})
	return true
}

func decode(request *ghttp.Request, value any) bool {
	decoder := json.NewDecoder(io.LimitReader(request.Body, 64*1024))
	decoder.DisallowUnknownFields()
	return decoder.Decode(value) == nil && decoder.Decode(&struct{}{}) == io.EOF
}

func token(request *ghttp.Request) string {
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func principal(request *ghttp.Request) identity.Principal {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	return identity.Principal{RequestID: httpx.RequestID(request), IPAddress: strings.TrimSpace(host), UserAgent: strings.TrimSpace(request.UserAgent())}
}
