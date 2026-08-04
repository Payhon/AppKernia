package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"

	blockapp "github.com/appkernia/appkernia/server/internal/modules/blockruleadmin/application"
	blocks "github.com/appkernia/appkernia/server/internal/modules/blockruleadmin/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *blockapp.Service
	catalog *i18n.Catalog
}

func NewHandler(s *blockapp.Service, c *i18n.Catalog) *Handler { return &Handler{s, c} }
func (h *Handler) List(r *ghttp.Request) {
	p, e1 := strconv.Atoi(r.GetQuery("page", 1).String())
	z, e2 := strconv.Atoi(r.GetQuery("page_size", 20).String())
	if e1 != nil || e2 != nil {
		h.fail(r, blocks.ErrInvalid)
		return
	}
	out, e := h.service.List(r.Context(), token(r), blocks.Filter{SubjectType: r.GetQuery("subject_type").String(), SubjectHint: r.GetQuery("subject_hint").String(), Scope: r.GetQuery("scope").String(), Status: r.GetQuery("status").String(), Expiry: r.GetQuery("expiry").String(), Page: int32(p), PageSize: int32(z)})
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Create(r *ghttp.Request) {
	var in blocks.CreateInput
	if !decode(r, &in) {
		h.fail(r, blocks.ErrInvalid)
		return
	}
	out, e := h.service.Create(r.Context(), token(r), principal(r), in)
	if !h.fail(r, e) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) Update(r *ghttp.Request) {
	id, ok := id(r)
	var in blocks.UpdateInput
	if !ok || !decode(r, &in) {
		h.fail(r, blocks.ErrInvalid)
		return
	}
	out, e := h.service.Update(r.Context(), token(r), principal(r), id, in)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Revoke(r *ghttp.Request) {
	id, ok := id(r)
	if !ok {
		h.fail(r, blocks.ErrInvalid)
		return
	}
	out, e := h.service.Revoke(r.Context(), token(r), principal(r), id)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) ok(r *ghttp.Request, status int, data any) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(r)})
}
func (h *Handler) fail(r *ghttp.Request, e error) bool {
	if e == nil {
		return false
	}
	status, code, key := 500, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(e, iamapp.ErrInvalidAccessToken):
		status, code, key = 401, "AUTH.TOKEN.INVALID", "errors.common.unauthorized"
	case errors.Is(e, blocks.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(e, blocks.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(e, blocks.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
func decode(r *ghttp.Request, v any) bool {
	d := json.NewDecoder(io.LimitReader(r.Body, 256*1024))
	d.DisallowUnknownFields()
	return d.Decode(v) == nil && d.Decode(&struct{}{}) == io.EOF
}
func id(r *ghttp.Request) (uuid.UUID, bool) {
	v, e := uuid.Parse(r.GetRouter("id").String())
	return v, e == nil
}
func token(r *ghttp.Request) string {
	v := strings.Fields(r.Header.Get("Authorization"))
	if len(v) == 2 && strings.EqualFold(v[0], "Bearer") {
		return v[1]
	}
	return ""
}
func principal(r *ghttp.Request) blocks.Principal {
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		host = r.RemoteAddr
	}
	return blocks.Principal{RequestID: httpx.RequestID(r), IPAddress: strings.TrimSpace(host), UserAgent: strings.TrimSpace(r.UserAgent())}
}
