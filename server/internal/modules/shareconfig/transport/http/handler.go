package http

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	shareapp "github.com/appkernia/appkernia/server/internal/modules/shareconfig/application"
	share "github.com/appkernia/appkernia/server/internal/modules/shareconfig/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *shareapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *shareapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

type statusRequest struct {
	LockVersion int32 `json:"lock_version"`
}

func token(r *ghttp.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) > 7 && strings.EqualFold(value[:7], "Bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return ""
}

func principal(r *ghttp.Request) share.Principal {
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}
	return share.Principal{RequestID: httpx.RequestID(r), IPAddress: ip, UserAgent: r.Header.Get("User-Agent")}
}

func decode(r *ghttp.Request, out any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(out) == nil
}

func idParam(r *ghttp.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(r.GetRouter(key).String()))
	return id, err == nil
}

func (h *Handler) ok(r *ghttp.Request, status int, data any) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(r)})
}

func (h *Handler) fail(r *ghttp.Request, err error) bool {
	if err == nil {
		return false
	}
	status, code, key := 500, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(err, iamapp.ErrInvalidAccessToken):
		status, code, key = 401, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized"
	case errors.Is(err, share.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, share.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, share.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, share.ErrInUse):
		status, code, key = 409, "SHARE.CONFIG.IN_USE", "errors.common.conflict"
	case errors.Is(err, share.ErrConflict):
		status, code, key = 409, "COMMON.CONFLICT", "errors.common.conflict"
	}
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}

func (h *Handler) Configs(r *ghttp.Request) {
	page, pageErr := strconv.Atoi(r.GetQuery("page", 1).String())
	pageSize, sizeErr := strconv.Atoi(r.GetQuery("page_size", 20).String())
	if pageErr != nil || sizeErr != nil {
		h.fail(r, share.ErrInvalid)
		return
	}
	out, err := h.service.List(r.Context(), token(r), share.ListFilter{Query: r.GetQuery("q").String(), ProviderCode: r.GetQuery("provider_code").String(), Status: r.GetQuery("status").String(), Page: int32(page), PageSize: int32(pageSize)})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) CreateConfig(r *ghttp.Request) {
	var body share.ConfigInput
	if !decode(r, &body) {
		h.fail(r, share.ErrInvalid)
		return
	}
	out, err := h.service.Create(r.Context(), token(r), principal(r), body)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}

func (h *Handler) Config(r *ghttp.Request) {
	id, ok := idParam(r, "id")
	if !ok {
		h.fail(r, share.ErrInvalid)
		return
	}
	switch r.Method {
	case "GET":
		out, err := h.service.Get(r.Context(), token(r), id)
		if !h.fail(r, err) {
			h.ok(r, 200, out)
		}
	case "PATCH":
		var body share.ConfigInput
		if !decode(r, &body) {
			h.fail(r, share.ErrInvalid)
			return
		}
		out, err := h.service.Update(r.Context(), token(r), principal(r), id, body)
		if !h.fail(r, err) {
			h.ok(r, 200, out)
		}
	case "DELETE":
		version, err := strconv.Atoi(r.GetQuery("lock_version").String())
		if err != nil {
			h.fail(r, share.ErrInvalid)
			return
		}
		err = h.service.Delete(r.Context(), token(r), principal(r), id, int32(version))
		if !h.fail(r, err) {
			h.ok(r, 200, map[string]bool{"deleted": true})
		}
	default:
		h.fail(r, share.ErrInvalid)
	}
}

func (h *Handler) Activate(r *ghttp.Request) {
	id, ok := idParam(r, "id")
	var body statusRequest
	if !ok || !decode(r, &body) {
		h.fail(r, share.ErrInvalid)
		return
	}
	out, err := h.service.Activate(r.Context(), token(r), principal(r), id, body.LockVersion)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) Disable(r *ghttp.Request) {
	id, ok := idParam(r, "id")
	var body statusRequest
	if !ok || !decode(r, &body) {
		h.fail(r, share.ErrInvalid)
		return
	}
	out, err := h.service.Disable(r.Context(), token(r), principal(r), id, body.LockVersion)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) RotateSecret(r *ghttp.Request) {
	id, ok := idParam(r, "id")
	var body share.SecretInput
	if !ok || !decode(r, &body) {
		h.fail(r, share.ErrInvalid)
		return
	}
	out, err := h.service.RotateSecret(r.Context(), token(r), principal(r), id, body)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) Bindings(r *ghttp.Request) {
	appID, ok := idParam(r, "app_id")
	if !ok {
		h.fail(r, share.ErrInvalid)
		return
	}
	out, err := h.service.ListBindings(r.Context(), token(r), appID)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) Binding(r *ghttp.Request) {
	appID, ok := idParam(r, "app_id")
	provider := strings.TrimSpace(r.GetRouter("provider_code").String())
	if !ok {
		h.fail(r, share.ErrInvalid)
		return
	}
	switch r.Method {
	case "PUT":
		var body share.BindingInput
		if !decode(r, &body) {
			h.fail(r, share.ErrInvalid)
			return
		}
		out, err := h.service.UpsertBinding(r.Context(), token(r), principal(r), appID, provider, body)
		if !h.fail(r, err) {
			h.ok(r, 200, out)
		}
	case "DELETE":
		version, err := strconv.Atoi(r.GetQuery("lock_version").String())
		if err != nil {
			h.fail(r, share.ErrInvalid)
			return
		}
		err = h.service.DeleteBinding(r.Context(), token(r), principal(r), appID, provider, int32(version))
		if !h.fail(r, err) {
			h.ok(r, 200, map[string]bool{"deleted": true})
		}
	default:
		h.fail(r, share.ErrInvalid)
	}
}

func (h *Handler) Preflight(r *ghttp.Request) {
	appID, ok := idParam(r, "app_id")
	provider := strings.TrimSpace(r.GetRouter("provider_code").String())
	var body share.BindingInput
	if !ok || !decode(r, &body) {
		h.fail(r, share.ErrInvalid)
		return
	}
	out, err := h.service.Preflight(r.Context(), token(r), appID, provider, body)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
