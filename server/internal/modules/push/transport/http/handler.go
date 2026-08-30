package http

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	pushapp "github.com/appkernia/appkernia/server/internal/modules/push/application"
	push "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *pushapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *pushapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

type lockVersionRequest struct {
	LockVersion int32 `json:"lock_version"`
}

func bearer(r *ghttp.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func principal(r *ghttp.Request) push.Principal {
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}
	return push.Principal{RequestID: httpx.RequestID(r), IPAddress: ip, UserAgent: r.Header.Get("User-Agent")}
}

func decode(r *ghttp.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target) == nil
}

func routeID(r *ghttp.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(r.GetRouter(name).String()))
	return id, err == nil
}

func (h *Handler) success(r *ghttp.Request, status int, data any) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(r)})
}

func (h *Handler) failure(r *ghttp.Request, err error) bool {
	if err == nil {
		return false
	}
	status, code, key := http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(err, iamapp.ErrInvalidAccessToken):
		status, code, key = http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized"
	case errors.Is(err, push.ErrInvalid):
		status, code, key = http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, push.ErrForbidden):
		status, code, key = http.StatusForbidden, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, push.ErrNotFound):
		status, code, key = http.StatusNotFound, "PUSH.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, push.ErrConflict):
		status, code, key = http.StatusConflict, "PUSH.CONFIG.CONFLICT", "errors.common.conflict"
	case errors.Is(err, push.ErrUnavailable):
		status, code, key = http.StatusServiceUnavailable, "PUSH.UNAVAILABLE", "errors.common.unavailable"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}

func (h *Handler) CurrentDevice(r *ghttp.Request) {
	device, err := h.service.CurrentDevice(r.Context(), bearer(r))
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, map[string]any{"device": device})
	}
}

func (h *Handler) RegisterDevice(r *ghttp.Request) {
	var input push.DeviceInput
	if !decode(r, &input) {
		h.failure(r, push.ErrInvalid)
		return
	}
	device, err := h.service.RegisterDevice(r.Context(), bearer(r), principal(r), input)
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, device)
	}
}

func (h *Handler) DisableDevice(r *ghttp.Request) {
	id, ok := routeID(r, "push_device_id")
	if !ok {
		h.failure(r, push.ErrInvalid)
		return
	}
	err := h.service.DisableDevice(r.Context(), bearer(r), principal(r), id)
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, map[string]bool{"disabled": true})
	}
}

func (h *Handler) MarkOpened(r *ghttp.Request) {
	id, ok := routeID(r, "delivery_id")
	if !ok {
		h.failure(r, push.ErrInvalid)
		return
	}
	err := h.service.MarkOpened(r.Context(), bearer(r), principal(r), id)
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, map[string]bool{"opened": true})
	}
}

func (h *Handler) Catalog(r *ghttp.Request) {
	items, err := h.service.Catalog(r.Context(), bearer(r))
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, map[string]any{"items": items})
	}
}

func (h *Handler) Configs(r *ghttp.Request) {
	appID, ok := routeID(r, "app_id")
	if !ok {
		h.failure(r, push.ErrInvalid)
		return
	}
	items, err := h.service.ListConfigs(r.Context(), bearer(r), appID, r.GetQuery("environment").String())
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, map[string]any{"items": items})
	}
}

func (h *Handler) UpsertConfig(r *ghttp.Request) {
	appID, ok := routeID(r, "app_id")
	var input push.ProviderConfigInput
	if !ok || !decode(r, &input) {
		h.failure(r, push.ErrInvalid)
		return
	}
	input.Provider = strings.TrimSpace(r.GetRouter("provider").String())
	item, err := h.service.UpsertConfig(r.Context(), bearer(r), appID, principal(r), input)
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, item)
	}
}

func (h *Handler) RotateSecret(r *ghttp.Request) {
	appID, appOK := routeID(r, "app_id")
	id, idOK := routeID(r, "id")
	var input push.SecretInput
	if !appOK || !idOK || !decode(r, &input) {
		h.failure(r, push.ErrInvalid)
		return
	}
	item, err := h.service.RotateSecret(r.Context(), bearer(r), appID, id, principal(r), input)
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, item)
	}
}

func (h *Handler) Preflight(r *ghttp.Request) {
	h.updateStatus(r, "preflight")
}

func (h *Handler) Activate(r *ghttp.Request) {
	h.updateStatus(r, "active")
}

func (h *Handler) Disable(r *ghttp.Request) {
	h.updateStatus(r, "disabled")
}

func (h *Handler) updateStatus(r *ghttp.Request, status string) {
	appID, appOK := routeID(r, "app_id")
	id, idOK := routeID(r, "id")
	var input lockVersionRequest
	if !appOK || !idOK || !decode(r, &input) {
		h.failure(r, push.ErrInvalid)
		return
	}
	var item push.ProviderConfig
	var err error
	if status == "preflight" {
		item, err = h.service.Preflight(r.Context(), bearer(r), appID, id, principal(r), input.LockVersion)
	} else {
		item, err = h.service.SetStatus(r.Context(), bearer(r), appID, id, principal(r), input.LockVersion, status)
	}
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, item)
	}
}

func (h *Handler) Test(r *ghttp.Request) {
	appID, appOK := routeID(r, "app_id")
	id, idOK := routeID(r, "id")
	var input push.TestInput
	if !appOK || !idOK || !decode(r, &input) {
		h.failure(r, push.ErrInvalid)
		return
	}
	delivery, err := h.service.Test(r.Context(), bearer(r), appID, id, principal(r), input)
	if !h.failure(r, err) {
		h.success(r, http.StatusAccepted, delivery)
	}
}

func (h *Handler) TestDevices(r *ghttp.Request) {
	appID, ok := routeID(r, "app_id")
	if !ok {
		h.failure(r, push.ErrInvalid)
		return
	}
	items, err := h.service.ListTestDevices(r.Context(), bearer(r), appID, r.GetQuery("provider").String())
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, map[string]any{"items": items})
	}
}

func (h *Handler) DeliverySummary(r *ghttp.Request) {
	appID, ok := routeID(r, "app_id")
	if !ok {
		h.failure(r, push.ErrInvalid)
		return
	}
	items, err := h.service.DeliverySummary(r.Context(), bearer(r), appID)
	if !h.failure(r, err) {
		h.success(r, http.StatusOK, map[string]any{"items": items, "window_days": 30})
	}
}
