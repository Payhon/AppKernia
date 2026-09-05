package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	clientapp "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/application"
	clients "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *clientapp.Service
	catalog *i18n.Catalog
}

func NewHandler(s *clientapp.Service, c *i18n.Catalog) *Handler { return &Handler{s, c} }
func (h *Handler) List(r *ghttp.Request) {
	p, e1 := strconv.Atoi(r.GetQuery("page", 1).String())
	z, e2 := strconv.Atoi(r.GetQuery("page_size", 20).String())
	if e1 != nil || e2 != nil {
		h.fail(r, clients.ErrInvalid)
		return
	}
	out, e := h.service.List(r.Context(), token(r), clients.Filter{Query: r.GetQuery("q").String(), Status: r.GetQuery("status").String(), Page: int32(p), PageSize: int32(z)})
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Get(r *ghttp.Request) {
	clientID, ok := id(r, "id")
	if !ok {
		h.fail(r, clients.ErrInvalid)
		return
	}
	out, e := h.service.Get(r.Context(), token(r), clientID)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Create(r *ghttp.Request) {
	var in clients.Input
	if !decode(r, &in) {
		h.fail(r, clients.ErrInvalid)
		return
	}
	out, e := h.service.Create(r.Context(), token(r), principal(r), in)
	if !h.fail(r, e) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) Update(r *ghttp.Request) {
	id, ok := id(r, "id")
	var in clients.Input
	if !ok || !decode(r, &in) {
		h.fail(r, clients.ErrInvalid)
		return
	}
	out, e := h.service.Update(r.Context(), token(r), principal(r), id, in)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CreateSecret(r *ghttp.Request) {
	id, ok := id(r, "id")
	var body struct {
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if !ok || !decodeOptional(r, &body) {
		h.fail(r, clients.ErrInvalid)
		return
	}
	out, e := h.service.CreateSecret(r.Context(), token(r), principal(r), id, body.ExpiresAt)
	if !h.fail(r, e) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) RevokeSecret(r *ghttp.Request) {
	cid, cok := id(r, "id")
	sid, sok := id(r, "secret_id")
	if !cok || !sok {
		h.fail(r, clients.ErrInvalid)
		return
	}
	e := h.service.RevokeSecret(r.Context(), token(r), principal(r), cid, sid)
	if !h.fail(r, e) {
		h.ok(r, 200, map[string]bool{"revoked": true})
	}
}
func (h *Handler) Permissions(r *ghttp.Request) {
	cid, ok := id(r, "id")
	var body struct {
		PermissionCodes []string `json:"permission_codes"`
	}
	if !ok || !decode(r, &body) {
		h.fail(r, clients.ErrInvalid)
		return
	}
	out, e := h.service.Permissions(r.Context(), token(r), principal(r), cid, body.PermissionCodes)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Applications(r *ghttp.Request) {
	cid, ok := id(r, "id")
	var body struct {
		AppIDs []uuid.UUID `json:"app_ids"`
	}
	if !ok || !decode(r, &body) {
		h.fail(r, clients.ErrInvalid)
		return
	}
	out, e := h.service.Applications(r.Context(), token(r), principal(r), cid, body.AppIDs)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Token(r *ghttp.Request) {
	var body struct {
		ClientID string `json:"client_id"`
		Secret   string `json:"client_secret"`
	}
	if !decode(r, &body) {
		h.fail(r, clients.ErrCredential)
		return
	}
	access, expires, e := h.service.Token(r.Context(), body.ClientID, body.Secret, clients.TokenMetadata{
		RequestID: httpx.RequestID(r), IPAddress: remoteIP(r), UserAgent: r.UserAgent(),
	})
	if !h.fail(r, e) {
		h.ok(r, 200, map[string]any{"access_token": access, "token_type": "Bearer", "audience": "ak-api", "expires_at": expires})
	}
}
func remoteIP(r *ghttp.Request) string {
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		host = r.RemoteAddr
	}
	return strings.TrimSpace(host)
}
func principal(r *ghttp.Request) clients.Principal {
	return clients.Principal{RequestID: httpx.RequestID(r), IPAddress: remoteIP(r), UserAgent: strings.TrimSpace(r.Header.Get("User-Agent"))}
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
	case errors.Is(e, iamapp.ErrInvalidAccessToken), errors.Is(e, clients.ErrCredential):
		status, code, key = 401, "AUTH.CLIENT.INVALID", "errors.common.unauthorized"
	case errors.Is(e, clients.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(e, clients.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(e, clients.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(e, clients.ErrConflict):
		status, code, key = 409, "SYS.API_CLIENT.CONFLICT", "errors.common.conflict"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
func decode(r *ghttp.Request, target any) bool {
	d := json.NewDecoder(io.LimitReader(r.Body, 128*1024))
	d.DisallowUnknownFields()
	return d.Decode(target) == nil && d.Decode(&struct{}{}) == io.EOF
}
func decodeOptional(r *ghttp.Request, target any) bool {
	return r.ContentLength == 0 || decode(r, target)
}
func id(r *ghttp.Request, key string) (uuid.UUID, bool) {
	v, e := uuid.Parse(r.GetRouter(key).String())
	return v, e == nil
}
func token(r *ghttp.Request) string {
	p := strings.Fields(r.Header.Get("Authorization"))
	if len(p) == 2 && strings.EqualFold(p[0], "Bearer") {
		return p[1]
	}
	return ""
}
