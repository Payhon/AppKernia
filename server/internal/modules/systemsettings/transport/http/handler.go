package http

import (
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"strconv"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	settingsapp "github.com/appkernia/appkernia/server/internal/modules/systemsettings/application"
	settings "github.com/appkernia/appkernia/server/internal/modules/systemsettings/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *settingsapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *settingsapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

type configRequest struct {
	ModuleCode       string          `json:"module_code"`
	ConfigGroup      string          `json:"config_group"`
	ConfigKey        string          `json:"config_key"`
	DisplayName      string          `json:"display_name"`
	ValueType        string          `json:"value_type"`
	Value            json.RawMessage `json:"value"`
	DefaultValue     json.RawMessage `json:"default_value"`
	SecretValue      string          `json:"secret_value"`
	IsSecret         bool            `json:"is_secret"`
	IsPublic         bool            `json:"is_public"`
	ValidationSchema json.RawMessage `json:"validation_schema"`
	Description      string          `json:"description"`
	SortOrder        int32           `json:"sort_order"`
	Status           string          `json:"status"`
	Version          int32           `json:"version"`
}

func (x configRequest) input() settings.ConfigInput {
	return settings.ConfigInput{ModuleCode: x.ModuleCode, ConfigGroup: x.ConfigGroup, ConfigKey: x.ConfigKey, DisplayName: x.DisplayName, ValueType: x.ValueType, Value: x.Value, DefaultValue: x.DefaultValue, SecretValue: x.SecretValue, IsSecret: x.IsSecret, IsPublic: x.IsPublic, ValidationSchema: x.ValidationSchema, Description: x.Description, SortOrder: x.SortOrder, Status: x.Status, Version: x.Version}
}

type rotateRequest struct {
	SecretValue string `json:"secret_value"`
	Version     int32  `json:"version"`
}
type typeRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}
type itemRequest struct {
	ItemValue string          `json:"item_value"`
	Label     string          `json:"label"`
	Locale    *string         `json:"locale"`
	Color     string          `json:"color"`
	CSSClass  string          `json:"css_class"`
	SortOrder int32           `json:"sort_order"`
	IsDefault bool            `json:"is_default"`
	Extra     json.RawMessage `json:"extra"`
	Status    string          `json:"status"`
}

type regionCreateRequest struct {
	Code       string   `json:"code"`
	ParentCode string   `json:"parent_code"`
	Name       string   `json:"name"`
	FullName   string   `json:"full_name"`
	PostalCode string   `json:"postal_code"`
	Longitude  *float64 `json:"longitude"`
	Latitude   *float64 `json:"latitude"`
	Status     string   `json:"status"`
}

func (x regionCreateRequest) input() settings.RegionCreateInput {
	return settings.RegionCreateInput{Code: x.Code, ParentCode: x.ParentCode, Name: x.Name, FullName: x.FullName, PostalCode: x.PostalCode, Longitude: x.Longitude, Latitude: x.Latitude, Status: x.Status}
}

type regionUpdateRequest struct {
	Name       string   `json:"name"`
	FullName   string   `json:"full_name"`
	PostalCode string   `json:"postal_code"`
	Longitude  *float64 `json:"longitude"`
	Latitude   *float64 `json:"latitude"`
	Status     string   `json:"status"`
	Version    int32    `json:"version"`
}

func (x regionUpdateRequest) input() settings.RegionUpdateInput {
	return settings.RegionUpdateInput{Name: x.Name, FullName: x.FullName, PostalCode: x.PostalCode, Longitude: x.Longitude, Latitude: x.Latitude, Status: x.Status, Version: x.Version}
}

func (x itemRequest) input() settings.DictItemInput {
	return settings.DictItemInput{ItemValue: x.ItemValue, Label: x.Label, Locale: x.Locale, Color: x.Color, CSSClass: x.CSSClass, SortOrder: x.SortOrder, IsDefault: x.IsDefault, Extra: x.Extra, Status: x.Status}
}

func page(r *ghttp.Request) (int32, int32, bool) {
	p, e1 := strconv.Atoi(r.GetQuery("page", 1).String())
	s, e2 := strconv.Atoi(r.GetQuery("page_size", 20).String())
	return int32(p), int32(s), e1 == nil && e2 == nil
}
func optionalBool(r *ghttp.Request, key string) (*bool, bool) {
	raw := strings.TrimSpace(r.GetQuery(key).String())
	if raw == "" {
		return nil, true
	}
	v, err := strconv.ParseBool(raw)
	return &v, err == nil
}
func (h *Handler) Configs(r *ghttp.Request) {
	p, s, ok := page(r)
	pub, pubOK := optionalBool(r, "is_public")
	secret, secretOK := optionalBool(r, "is_secret")
	if !ok || !pubOK || !secretOK {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.ListConfigs(r.Context(), token(r), settings.PageFilter{Query: r.GetQuery("q").String(), ModuleCode: r.GetQuery("module_code").String(), Group: r.GetQuery("config_group").String(), ValueType: r.GetQuery("value_type").String(), Status: r.GetQuery("status").String(), Sort: r.GetQuery("sort").String(), IsPublic: pub, IsSecret: secret, Page: p, PageSize: s})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CreateConfig(r *ghttp.Request) {
	var body configRequest
	if !decode(r, &body) {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.CreateConfig(r.Context(), token(r), principal(r), body.input())
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateConfig(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body configRequest
	if !ok || !decode(r, &body) {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.UpdateConfig(r.Context(), token(r), principal(r), id, body.input())
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) RotateSecret(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body rotateRequest
	if !ok || !decode(r, &body) {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.RotateSecret(r.Context(), token(r), principal(r), id, body.Version, body.SecretValue)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) DictTypes(r *ghttp.Request) {
	p, s, ok := page(r)
	if !ok {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.ListDictTypes(r.Context(), token(r), settings.PageFilter{Query: r.GetQuery("q").String(), Status: r.GetQuery("status").String(), Sort: r.GetQuery("sort").String(), Page: p, PageSize: s})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CreateDictType(r *ghttp.Request) {
	var body typeRequest
	if !decode(r, &body) {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.CreateDictType(r.Context(), token(r), principal(r), settings.DictTypeInput{Code: body.Code, Name: body.Name, Description: body.Description, Status: body.Status})
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateDictType(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body typeRequest
	if !ok || !decode(r, &body) {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.UpdateDictType(r.Context(), token(r), principal(r), id, settings.DictTypeInput{Code: body.Code, Name: body.Name, Description: body.Description, Status: body.Status})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) DictItems(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	p, s, pageOK := page(r)
	if !ok || !pageOK {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.ListDictItems(r.Context(), token(r), id, settings.DictItemFilter{Query: r.GetQuery("q").String(), Locale: r.GetQuery("locale").String(), Status: r.GetQuery("status").String(), Sort: r.GetQuery("sort").String(), Page: p, PageSize: s})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CreateDictItem(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body itemRequest
	if !ok || !decode(r, &body) {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.CreateDictItem(r.Context(), token(r), principal(r), id, body.input())
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateDictItem(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var body itemRequest
	if !ok || !decode(r, &body) {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.UpdateDictItem(r.Context(), token(r), principal(r), id, body.input())
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) DeleteDictItem(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, settings.ErrInvalid)
		return
	}
	err := h.service.DeleteDictItem(r.Context(), token(r), principal(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
}

func regionFilter(r *ghttp.Request) (settings.RegionFilter, bool) {
	f := settings.RegionFilter{Query: r.GetQuery("q").String(), ParentCode: r.GetQuery("parent_code").String(), Status: r.GetQuery("status").String()}
	limit, err := strconv.Atoi(r.GetQuery("limit", 100).String())
	if err != nil {
		return f, false
	}
	f.Limit = int32(limit)
	if raw := strings.TrimSpace(r.GetQuery("level").String()); raw != "" {
		level, e := strconv.Atoi(raw)
		if e != nil || level < 0 || level > 10 {
			return f, false
		}
		v := int16(level)
		f.Level = &v
	}
	return f, true
}
func (h *Handler) Regions(r *ghttp.Request) {
	f, ok := regionFilter(r)
	if !ok {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.ListRegions(r.Context(), token(r), f)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]any{"items": out})
	}
}
func (h *Handler) CreateRegion(r *ghttp.Request) {
	var body regionCreateRequest
	if !decode(r, &body) {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.CreateRegion(r.Context(), token(r), principal(r), body.input())
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateRegion(r *ghttp.Request) {
	code := strings.TrimSpace(r.GetRouter("code").String())
	var body regionUpdateRequest
	if code == "" || !decode(r, &body) {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.UpdateRegion(r.Context(), token(r), principal(r), code, body.input())
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) DeleteRegion(r *ghttp.Request) {
	code := strings.TrimSpace(r.GetRouter("code").String())
	if code == "" {
		h.fail(r, settings.ErrInvalid)
		return
	}
	err := h.service.DeleteRegion(r.Context(), token(r), principal(r), code)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
}
func (h *Handler) PublicRegions(r *ghttp.Request) {
	f, ok := regionFilter(r)
	if !ok {
		h.fail(r, settings.ErrInvalid)
		return
	}
	out, err := h.service.ListPublicRegions(r.Context(), f)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]any{"items": out})
	}
}
func (h *Handler) AdminDictionary(r *ghttp.Request) {
	out, err := h.service.ResolveAdminDictionary(r.Context(), token(r), r.GetRouter("code").String(), string(httpx.Locale(r)))
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) PublicDictionary(r *ghttp.Request) {
	out, err := h.service.ResolvePublicDictionary(r.Context(), r.GetRouter("code").String(), string(httpx.Locale(r)))
	if !h.fail(r, err) {
		h.ok(r, 200, out)
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
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(out) == nil
}
func principal(r *ghttp.Request) settings.Principal {
	var ip *netip.Addr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if parsed, e := netip.ParseAddr(host); e == nil {
			ip = &parsed
		}
	}
	return settings.Principal{RequestID: httpx.RequestID(r), IPAddress: ip, UserAgent: r.Header.Get("User-Agent")}
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
	case errors.Is(err, settings.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, settings.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, settings.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, settings.ErrLocked):
		status, code, key = 409, "SYS.SETTINGS.LOCKED", "errors.common.conflict"
	case errors.Is(err, settings.ErrConflict):
		status, code, key = 409, "COMMON.CONFLICT", "errors.common.conflict"
	case errors.Is(err, settings.ErrSecretUnavailable):
		status, code, key = 503, "SYS.CONFIG.SECRET_UNAVAILABLE", "errors.common.unknown"
	}
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
