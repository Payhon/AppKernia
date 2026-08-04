package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	webhookapp "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/application"
	webhooks "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *webhookapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *webhookapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service, catalog}
}
func (h *Handler) List(r *ghttp.Request) {
	page, e1 := strconv.Atoi(r.GetQuery("page", 1).String())
	size, e2 := strconv.Atoi(r.GetQuery("page_size", 20).String())
	if e1 != nil || e2 != nil {
		h.fail(r, webhooks.ErrInvalid)
		return
	}
	out, err := h.service.List(r.Context(), token(r), webhooks.Filter{Query: r.GetQuery("q").String(), Status: r.GetQuery("status").String(), EventType: r.GetQuery("event_type").String(), Page: int32(page), PageSize: int32(size)})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Create(r *ghttp.Request) {
	var in webhooks.Input
	if !decode(r, &in) {
		h.fail(r, webhooks.ErrInvalid)
		return
	}
	out, err := h.service.Create(r.Context(), token(r), principal(r), in)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) Update(r *ghttp.Request) {
	id, ok := id(r)
	var in webhooks.Input
	if !ok || !decode(r, &in) {
		h.fail(r, webhooks.ErrInvalid)
		return
	}
	out, err := h.service.Update(r.Context(), token(r), principal(r), id, in)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Test(r *ghttp.Request) {
	id, ok := id(r)
	var in webhooks.TestInput
	if !ok || !decode(r, &in) {
		h.fail(r, webhooks.ErrInvalid)
		return
	}
	out, err := h.service.Test(r.Context(), token(r), principal(r), id, r.Header.Get("Idempotency-Key"), in)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Deliveries(r *ghttp.Request) {
	id, ok := id(r)
	page, e1 := strconv.Atoi(r.GetQuery("page", 1).String())
	size, e2 := strconv.Atoi(r.GetQuery("page_size", 20).String())
	if !ok || e1 != nil || e2 != nil {
		h.fail(r, webhooks.ErrInvalid)
		return
	}
	out, err := h.service.Deliveries(r.Context(), token(r), id, int32(page), int32(size))
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
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
		status, code, key = 401, "AUTH.TOKEN.INVALID", "errors.common.unauthorized"
	case errors.Is(err, webhooks.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, webhooks.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, webhooks.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, webhooks.ErrConflict):
		status, code, key = 409, "SYS.WEBHOOK.CONFLICT", "errors.common.conflict"
	case errors.Is(err, webhooks.ErrDelivery):
		status, code, key = 502, "SYS.WEBHOOK.DELIVERY_FAILED", "errors.common.unknown"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
func decode(r *ghttp.Request, target any) bool {
	d := json.NewDecoder(io.LimitReader(r.Body, 256*1024))
	d.DisallowUnknownFields()
	return d.Decode(target) == nil && d.Decode(&struct{}{}) == io.EOF
}
func id(r *ghttp.Request) (uuid.UUID, bool) {
	v, err := uuid.Parse(r.GetRouter("id").String())
	return v, err == nil
}
func token(r *ghttp.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
func principal(r *ghttp.Request) webhooks.Principal {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return webhooks.Principal{RequestID: httpx.RequestID(r), IPAddress: strings.TrimSpace(host), UserAgent: strings.TrimSpace(r.Header.Get("User-Agent"))}
}
