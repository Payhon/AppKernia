package http

import (
	"encoding/json"
	"errors"
	"net"
	stdhttp "net/http"
	"net/netip"
	"strconv"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	tenantapp "github.com/appkernia/appkernia/server/internal/modules/tenantadmin/application"
	tenantdomain "github.com/appkernia/appkernia/server/internal/modules/tenantadmin/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *tenantapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *tenantapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

type createRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
type updateRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}
type addMemberRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}
type memberStatusRequest struct {
	Status string `json:"status"`
}

func token(r *ghttp.Request) string {
	v := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(v) > 7 && strings.EqualFold(v[:7], "Bearer ") {
		return strings.TrimSpace(v[7:])
	}
	return ""
}
func principal(r *ghttp.Request) tenantdomain.Principal {
	var ip *netip.Addr
	if host, _, e := net.SplitHostPort(r.RemoteAddr); e == nil {
		if v, e := netip.ParseAddr(host); e == nil {
			ip = &v
		}
	}
	return tenantdomain.Principal{RequestID: httpx.RequestID(r), IPAddress: ip, UserAgent: r.Header.Get("User-Agent")}
}
func id(r *ghttp.Request, key string) (uuid.UUID, error) {
	return uuid.Parse(r.GetRouter(key).String())
}
func (h *Handler) List(r *ghttp.Request) {
	page, _ := strconv.Atoi(r.GetQuery("page").String())
	size, _ := strconv.Atoi(r.GetQuery("page_size").String())
	out, e := h.service.List(r.Context(), token(r), tenantdomain.Filters{Query: r.GetQuery("q").String(), Status: r.GetQuery("status").String(), Sort: r.GetQuery("sort").String(), Page: int32(page), PageSize: int32(size)})
	if h.fail(r, e) {
		return
	}
	h.ok(r, out)
}
func (h *Handler) Get(r *ghttp.Request) {
	v, e := id(r, "id")
	if e == nil {
		var out tenantdomain.Tenant
		out, e = h.service.Get(r.Context(), token(r), v)
		if !h.fail(r, e) {
			h.ok(r, out)
		}
	} else {
		h.fail(r, tenantdomain.ErrInvalid)
	}
}
func (h *Handler) Create(r *ghttp.Request) {
	var body createRequest
	e := json.NewDecoder(r.Body).Decode(&body)
	if e == nil {
		var out tenantdomain.Tenant
		out, e = h.service.Create(r.Context(), token(r), principal(r), tenantdomain.CreateInput{Code: body.Code, Name: body.Name})
		if !h.fail(r, e) {
			r.Response.WriteHeader(stdhttp.StatusCreated)
			h.ok(r, out)
		}
	} else {
		h.fail(r, tenantdomain.ErrInvalid)
	}
}
func (h *Handler) Update(r *ghttp.Request) {
	v, e := id(r, "id")
	var body updateRequest
	if e == nil {
		e = json.NewDecoder(r.Body).Decode(&body)
	}
	if e == nil {
		var out tenantdomain.Tenant
		out, e = h.service.Update(r.Context(), token(r), principal(r), v, tenantdomain.UpdateInput{Name: body.Name, Status: body.Status})
		if !h.fail(r, e) {
			h.ok(r, out)
		}
	} else {
		h.fail(r, tenantdomain.ErrInvalid)
	}
}
func (h *Handler) Members(r *ghttp.Request) {
	v, e := id(r, "id")
	if e == nil {
		var out []tenantdomain.Member
		out, e = h.service.Members(r.Context(), token(r), v)
		if !h.fail(r, e) {
			h.ok(r, out)
		}
	} else {
		h.fail(r, tenantdomain.ErrInvalid)
	}
}
func (h *Handler) AddMember(r *ghttp.Request) {
	v, e := id(r, "id")
	var body addMemberRequest
	if e == nil {
		e = json.NewDecoder(r.Body).Decode(&body)
	}
	if e == nil {
		var out tenantdomain.Member
		out, e = h.service.AddMember(r.Context(), token(r), principal(r), v, tenantdomain.AddMemberInput{Email: body.Email, DisplayName: body.DisplayName})
		if !h.fail(r, e) {
			r.Response.WriteHeader(stdhttp.StatusCreated)
			h.ok(r, out)
		}
	} else {
		h.fail(r, tenantdomain.ErrInvalid)
	}
}
func (h *Handler) SetMember(r *ghttp.Request) {
	tid, e := id(r, "id")
	uid, e2 := id(r, "user_id")
	var body memberStatusRequest
	if e == nil && e2 == nil {
		e = json.NewDecoder(r.Body).Decode(&body)
	}
	if e == nil {
		var out tenantdomain.Member
		out, e = h.service.SetMemberStatus(r.Context(), token(r), principal(r), tid, uid, body.Status)
		if !h.fail(r, e) {
			h.ok(r, out)
		}
	} else {
		h.fail(r, tenantdomain.ErrInvalid)
	}
}
func (h *Handler) RemoveMember(r *ghttp.Request) {
	tid, e := id(r, "id")
	uid, e2 := id(r, "user_id")
	if e == nil && e2 == nil {
		var out tenantdomain.Member
		out, e = h.service.SetMemberStatus(r.Context(), token(r), principal(r), tid, uid, "left")
		if !h.fail(r, e) {
			h.ok(r, out)
		}
	} else {
		h.fail(r, tenantdomain.ErrInvalid)
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
	status, code, key := stdhttp.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(e, iamapp.ErrInvalidAccessToken):
		status, code, key = 401, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized"
	case errors.Is(e, tenantdomain.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(e, tenantdomain.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(e, tenantdomain.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(e, tenantdomain.ErrConflict):
		status, code, key = 409, "IAM.TENANT.CONFLICT", "errors.common.conflict"
	case errors.Is(e, tenantdomain.ErrLastAdmin):
		status, code, key = 409, "IAM.TENANT.LAST_ADMIN", "errors.iam.user.last_admin"
	}
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
