package http

import (
	"errors"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	opsapp "github.com/appkernia/appkernia/server/internal/modules/opsadmin/application"
	ops "github.com/appkernia/appkernia/server/internal/modules/opsadmin/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
)

type Handler struct {
	service *opsapp.Service
	catalog *i18n.Catalog
}

func NewHandler(s *opsapp.Service, c *i18n.Catalog) *Handler { return &Handler{s, c} }
func (h *Handler) Health(r *ghttp.Request) {
	out, e := h.service.Health(r.Context(), token(r))
	if !h.fail(r, e) {
		h.ok(r, out)
	}
}
func (h *Handler) Runtime(r *ghttp.Request) {
	out, e := h.service.Runtime(r.Context(), token(r))
	if !h.fail(r, e) {
		h.ok(r, out)
	}
}
func (h *Handler) ok(r *ghttp.Request, data any) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(r)})
}
func (h *Handler) fail(r *ghttp.Request, e error) bool {
	if e == nil {
		return false
	}
	status, code, key := 500, "COMMON.UNKNOWN", "errors.common.unknown"
	if errors.Is(e, iamapp.ErrInvalidAccessToken) {
		status, code, key = 401, "AUTH.TOKEN.INVALID", "errors.common.unauthorized"
	} else if errors.Is(e, ops.ErrForbidden) {
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
func token(r *ghttp.Request) string {
	v := strings.Fields(r.Header.Get("Authorization"))
	if len(v) == 2 && strings.EqualFold(v[0], "Bearer") {
		return v[1]
	}
	return ""
}
