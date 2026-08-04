package http

import (
	"errors"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	auditapp "github.com/appkernia/appkernia/server/internal/modules/auditadmin/application"
	auditdomain "github.com/appkernia/appkernia/server/internal/modules/auditadmin/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *auditapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *auditapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func (h *Handler) Operations(r *ghttp.Request) {
	base, ok := pageFilter(r)
	if !ok {
		h.fail(r, auditdomain.ErrInvalid)
		return
	}
	out, err := h.service.Operations(r.Context(), token(r), auditdomain.OperationFilter{PageFilter: base, ModuleCode: r.GetQuery("module_code").String(), Result: r.GetQuery("result").String()})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) Logins(r *ghttp.Request) {
	base, ok := pageFilter(r)
	if !ok {
		h.fail(r, auditdomain.ErrInvalid)
		return
	}
	out, err := h.service.Logins(r.Context(), token(r), auditdomain.LoginFilter{PageFilter: base, Result: r.GetQuery("result").String(), Audience: r.GetQuery("audience").String(), AuthMethod: r.GetQuery("auth_method").String()})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) SecurityEvents(r *ghttp.Request) {
	base, ok := pageFilter(r)
	if !ok {
		h.fail(r, auditdomain.ErrInvalid)
		return
	}
	out, err := h.service.SecurityEvents(r.Context(), token(r), auditdomain.SecurityFilter{PageFilter: base, Severity: r.GetQuery("severity").String(), Source: r.GetQuery("source").String(), Status: r.GetQuery("status").String()})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) SecurityEvent(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, auditdomain.ErrInvalid)
		return
	}
	out, err := h.service.SecurityEvent(r.Context(), token(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) ResolveSecurityEvent(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, auditdomain.ErrInvalid)
		return
	}
	out, err := h.service.ResolveSecurityEvent(r.Context(), token(r), principal(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func pageFilter(r *ghttp.Request) (auditdomain.PageFilter, bool) {
	page, errPage := strconv.Atoi(r.GetQuery("page", 1).String())
	size, errSize := strconv.Atoi(r.GetQuery("page_size", 20).String())
	fromAt, fromOK := queryTime(r.GetQuery("from").String(), false)
	toAt, toOK := queryTime(r.GetQuery("to").String(), true)
	return auditdomain.PageFilter{Query: r.GetQuery("q").String(), FromAt: fromAt, ToAt: toAt, Page: int32(page), PageSize: int32(size)}, errPage == nil && errSize == nil && fromOK && toOK
}

func queryTime(value string, endOfDay bool) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, true
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, true
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, false
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return parsed, true
}

func token(r *ghttp.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) > 7 && strings.EqualFold(value[:7], "Bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return ""
}

func routerID(r *ghttp.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.GetRouter(key).String())
	return id, err == nil
}

func principal(r *ghttp.Request) auditdomain.Principal {
	var ip *netip.Addr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if parsed, parseErr := netip.ParseAddr(host); parseErr == nil {
			ip = &parsed
		}
	}
	return auditdomain.Principal{RequestID: httpx.RequestID(r), IPAddress: ip, UserAgent: r.Header.Get("User-Agent")}
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
	case errors.Is(err, auditdomain.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, auditdomain.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, auditdomain.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, auditdomain.ErrAlreadyResolved):
		status, code, key = 409, "AUDIT.SECURITY.ALREADY_RESOLVED", "errors.common.conflict"
	}
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
