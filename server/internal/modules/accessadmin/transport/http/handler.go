package http

import (
	"encoding/json"
	"errors"
	"net"
	stdhttp "net/http"
	"net/netip"
	"strconv"
	"strings"

	accessapp "github.com/appkernia/appkernia/server/internal/modules/accessadmin/application"
	accessdomain "github.com/appkernia/appkernia/server/internal/modules/accessadmin/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *accessapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *accessapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

type roleRequest struct {
	ParentID    *uuid.UUID `json:"parent_id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	SortOrder   int32      `json:"sort_order"`
	Status      string     `json:"status"`
}

func (x roleRequest) input() accessdomain.RoleInput {
	return accessdomain.RoleInput{ParentID: x.ParentID, Code: x.Code, Name: x.Name, Description: x.Description, SortOrder: x.SortOrder, Status: x.Status}
}

type permissionsRequest struct {
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}
type menusRequest struct {
	MenuIDs []uuid.UUID `json:"menu_ids"`
}
type scopeRequest struct {
	DataScope string      `json:"data_scope"`
	UnitIDs   []uuid.UUID `json:"unit_ids"`
}
type menuRequest struct {
	ParentID     *uuid.UUID `json:"parent_id"`
	PermissionID *uuid.UUID `json:"permission_id"`
	Code         string     `json:"code"`
	Title        string     `json:"title"`
	I18nKey      string     `json:"i18n_key"`
	Type         string     `json:"type"`
	Path         string     `json:"path"`
	ComponentKey string     `json:"component_key"`
	Icon         string     `json:"icon"`
	ExternalURL  string     `json:"external_url"`
	OpenMode     string     `json:"open_mode"`
	Hidden       bool       `json:"hidden"`
	Affix        bool       `json:"affix"`
	SortOrder    int32      `json:"sort_order"`
	Status       string     `json:"status"`
}

func (x menuRequest) input() accessdomain.MenuInput {
	return accessdomain.MenuInput{ParentID: x.ParentID, PermissionID: x.PermissionID, Code: x.Code, Title: x.Title, I18nKey: x.I18nKey, Type: x.Type, Path: x.Path, ComponentKey: x.ComponentKey, Icon: x.Icon, ExternalURL: x.ExternalURL, OpenMode: x.OpenMode, Hidden: x.Hidden, Affix: x.Affix, SortOrder: x.SortOrder, Status: x.Status}
}

type moveRequest struct {
	ParentID  *uuid.UUID `json:"parent_id"`
	SortOrder int32      `json:"sort_order"`
}

func (h *Handler) Roles(r *ghttp.Request) {
	page, _ := strconv.Atoi(r.GetQuery("page").String())
	size, _ := strconv.Atoi(r.GetQuery("page_size").String())
	out, err := h.service.Roles(r.Context(), token(r), accessdomain.Filters{Query: r.GetQuery("q").String(), Status: r.GetQuery("status").String(), RoleType: r.GetQuery("role_type").String(), Page: int32(page), PageSize: int32(size)})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CreateRole(r *ghttp.Request) {
	var body roleRequest
	if !decode(r, &body) {
		h.fail(r, accessdomain.ErrInvalid)
		return
	}
	out, err := h.service.CreateRole(r.Context(), token(r), principal(r), body.input())
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateRole(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body roleRequest
	if !ok || !decode(r, &body) {
		h.fail(r, accessdomain.ErrInvalid)
		return
	}
	out, err := h.service.UpdateRole(r.Context(), token(r), principal(r), id, body.input())
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) DeleteRole(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, accessdomain.ErrInvalid)
		return
	}
	err := h.service.DeleteRole(r.Context(), token(r), principal(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
}
func (h *Handler) ReplacePermissions(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body permissionsRequest
	if !ok || !decode(r, &body) {
		h.fail(r, accessdomain.ErrInvalid)
		return
	}
	out, err := h.service.ReplacePermissions(r.Context(), token(r), principal(r), id, body.PermissionIDs)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) ReplaceMenus(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body menusRequest
	if !ok || !decode(r, &body) {
		h.fail(r, accessdomain.ErrInvalid)
		return
	}
	out, err := h.service.ReplaceMenus(r.Context(), token(r), principal(r), id, body.MenuIDs)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) ReplaceDataScope(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body scopeRequest
	if !ok || !decode(r, &body) {
		h.fail(r, accessdomain.ErrInvalid)
		return
	}
	out, err := h.service.ReplaceDataScope(r.Context(), token(r), principal(r), id, body.DataScope, body.UnitIDs)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Permissions(r *ghttp.Request) {
	out, err := h.service.Permissions(r.Context(), token(r), accessdomain.PermissionFilters{Query: r.GetQuery("q").String(), ModuleCode: r.GetQuery("module_code").String(), ResourceName: r.GetQuery("resource_name").String(), ActionName: r.GetQuery("action_name").String(), PermissionKind: r.GetQuery("permission_kind").String(), Status: r.GetQuery("status").String()})
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]any{"items": out, "total": len(out)})
	}
}
func (h *Handler) Menus(r *ghttp.Request) {
	out, err := h.service.Menus(r.Context(), token(r))
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]any{"items": out})
	}
}
func (h *Handler) CreateMenu(r *ghttp.Request) {
	var body menuRequest
	if !decode(r, &body) {
		h.fail(r, accessdomain.ErrInvalid)
		return
	}
	out, err := h.service.CreateMenu(r.Context(), token(r), principal(r), body.input())
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateMenu(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body menuRequest
	if !ok || !decode(r, &body) {
		h.fail(r, accessdomain.ErrInvalid)
		return
	}
	out, err := h.service.UpdateMenu(r.Context(), token(r), principal(r), id, body.input())
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) MoveMenu(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body moveRequest
	if !ok || !decode(r, &body) {
		h.fail(r, accessdomain.ErrInvalid)
		return
	}
	out, err := h.service.MoveMenu(r.Context(), token(r), principal(r), id, accessdomain.MenuMove{ParentID: body.ParentID, SortOrder: body.SortOrder})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) DeleteMenu(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, accessdomain.ErrInvalid)
		return
	}
	err := h.service.DeleteMenu(r.Context(), token(r), principal(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
}

func token(r *ghttp.Request) string {
	v := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(v) > 7 && strings.EqualFold(v[:7], "Bearer ") {
		return strings.TrimSpace(v[7:])
	}
	return ""
}
func routerID(r *ghttp.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.GetRouter(key).String())
	return id, err == nil
}
func decode(r *ghttp.Request, out any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(out) == nil
}
func principal(r *ghttp.Request) accessdomain.Principal {
	var ip *netip.Addr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if parsed, e := netip.ParseAddr(host); e == nil {
			ip = &parsed
		}
	}
	return accessdomain.Principal{RequestID: httpx.RequestID(r), IPAddress: ip, UserAgent: r.Header.Get("User-Agent")}
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
	case errors.Is(err, accessdomain.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, accessdomain.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, accessdomain.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, accessdomain.ErrMenuCycle):
		status, code, key = 409, "SYS.MENU.CYCLE", "errors.common.conflict"
	case errors.Is(err, accessdomain.ErrMenuDepth):
		status, code, key = 409, "SYS.MENU.DEPTH", "errors.common.conflict"
	case errors.Is(err, accessdomain.ErrComponentKey):
		status, code, key = 422, "SYS.MENU.COMPONENT_KEY_INVALID", "errors.validation.failed"
	case errors.Is(err, accessdomain.ErrSystemRole):
		status, code, key = 409, "IAM.ROLE.SYSTEM_IMMUTABLE", "errors.common.conflict"
	case errors.Is(err, accessdomain.ErrRoleOccupied):
		status, code, key = 409, "IAM.ROLE.OCCUPIED", "errors.common.conflict"
	case errors.Is(err, accessdomain.ErrMenuOccupied):
		status, code, key = 409, "SYS.MENU.OCCUPIED", "errors.common.conflict"
	case errors.Is(err, accessdomain.ErrConflict):
		status, code, key = 409, "COMMON.CONFLICT", "errors.common.conflict"
	}
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}

var _ = stdhttp.StatusOK
