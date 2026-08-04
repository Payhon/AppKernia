package http

import (
	"errors"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	sessionapp "github.com/appkernia/appkernia/server/internal/modules/sessionadmin/application"
	sessiondomain "github.com/appkernia/appkernia/server/internal/modules/sessionadmin/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *sessionapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *sessionapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func (h *Handler) List(r *ghttp.Request) {
	page, pageErr := strconv.Atoi(r.GetQuery("page", 1).String())
	size, sizeErr := strconv.Atoi(r.GetQuery("page_size", 20).String())
	fromAt, fromOK := parseTime(r.GetQuery("from").String(), false)
	toAt, toOK := parseTime(r.GetQuery("to").String(), true)
	if pageErr != nil || sizeErr != nil || !fromOK || !toOK {
		h.fail(r, sessiondomain.ErrInvalid)
		return
	}
	out, err := h.service.List(r.Context(), bearer(r), sessiondomain.Filter{Query: r.GetQuery("q").String(), Audience: r.GetQuery("audience").String(), Platform: r.GetQuery("platform").String(), Status: r.GetQuery("status").String(), IP: r.GetQuery("ip").String(), FromAt: fromAt, ToAt: toAt, Page: int32(page), PageSize: int32(size)})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) Revoke(r *ghttp.Request) {
	id, err := uuid.Parse(r.GetRouter("id").String())
	if err != nil {
		h.fail(r, sessiondomain.ErrInvalid)
		return
	}
	out, err := h.service.Revoke(r.Context(), bearer(r), principal(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func bearer(r *ghttp.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) > 7 && strings.EqualFold(value[:7], "Bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return ""
}

func principal(r *ghttp.Request) sessiondomain.Principal {
	var ip *netip.Addr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if parsed, parseErr := netip.ParseAddr(host); parseErr == nil {
			ip = &parsed
		}
	}
	return sessiondomain.Principal{RequestID: httpx.RequestID(r), IPAddress: ip, UserAgent: r.Header.Get("User-Agent")}
}

func parseTime(value string, end bool) (time.Time, bool) {
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
	if end {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return parsed, true
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
	case errors.Is(err, sessiondomain.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, sessiondomain.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, sessiondomain.ErrSessionAbsent):
		status, code, key = 409, "IAM.SESSION.NOT_ACTIVE", "errors.common.conflict"
	}
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
