package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	userapp "github.com/appkernia/appkernia/server/internal/modules/useradmin/application"
	userdomain "github.com/appkernia/appkernia/server/internal/modules/useradmin/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *userapp.Service
	catalog *i18n.Catalog
}

type createRequest struct {
	Email             string `json:"email"`
	DisplayName       string `json:"display_name"`
	Locale            string `json:"locale"`
	TimeZone          string `json:"time_zone"`
	TemporaryPassword string `json:"temporary_password"`
}
type updateRequest struct {
	DisplayName string `json:"display_name"`
	Locale      string `json:"locale"`
	TimeZone    string `json:"time_zone"`
}
type passwordRequest struct {
	TemporaryPassword string `json:"temporary_password"`
}
type rolesRequest struct {
	RoleIDs []uuid.UUID `json:"role_ids"`
}
type assignmentsRequest struct {
	UnitIDs           []uuid.UUID `json:"unit_ids"`
	PrimaryUnitID     *uuid.UUID  `json:"primary_unit_id"`
	PositionIDs       []uuid.UUID `json:"position_ids"`
	PrimaryPositionID *uuid.UUID  `json:"primary_position_id"`
}
type filtersRequest struct {
	Query       string     `json:"q"`
	Status      string     `json:"status"`
	UnitID      *uuid.UUID `json:"unit_id"`
	PositionID  *uuid.UUID `json:"position_id"`
	RoleID      *uuid.UUID `json:"role_id"`
	CreatedFrom *time.Time `json:"created_from"`
	CreatedTo   *time.Time `json:"created_to"`
	Sort        string     `json:"sort"`
}

func NewHandler(service *userapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func (handler *Handler) RoleOptions(request *ghttp.Request) {
	items, err := handler.service.RoleOptions(request.Context(), bearerToken(request))
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, map[string]any{"items": items})
}

func (handler *Handler) Users(request *ghttp.Request) {
	filters, ok := queryFilters(request)
	if !ok {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	page, err := handler.service.List(request.Context(), bearerToken(request), filters)
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, page)
}
func (handler *Handler) User(request *ghttp.Request) {
	id, ok := routerID(request, "id")
	if !ok {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	user, err := handler.service.Get(request.Context(), bearerToken(request), id)
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, user)
}
func (handler *Handler) Create(request *ghttp.Request) {
	var body createRequest
	if !decode(request, &body) {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	user, err := handler.service.Create(request.Context(), bearerToken(request), principal(request), userdomain.CreateInput{Email: body.Email, DisplayName: body.DisplayName, Locale: body.Locale, TimeZone: body.TimeZone, TemporaryPassword: body.TemporaryPassword})
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 201, user)
}
func (handler *Handler) Update(request *ghttp.Request) {
	id, ok := routerID(request, "id")
	var body updateRequest
	if !ok || !decode(request, &body) {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	user, err := handler.service.Update(request.Context(), bearerToken(request), principal(request), id, userdomain.UpdateInput{DisplayName: body.DisplayName, Locale: body.Locale, TimeZone: body.TimeZone})
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, user)
}
func (handler *Handler) Enable(request *ghttp.Request)  { handler.status(request, true) }
func (handler *Handler) Disable(request *ghttp.Request) { handler.status(request, false) }
func (handler *Handler) status(request *ghttp.Request, enabled bool) {
	id, ok := routerID(request, "id")
	if !ok {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	user, err := handler.service.SetStatus(request.Context(), bearerToken(request), principal(request), id, enabled)
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, user)
}
func (handler *Handler) Unlock(request *ghttp.Request) {
	id, ok := routerID(request, "id")
	if !ok {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	err := handler.service.Unlock(request.Context(), bearerToken(request), principal(request), id)
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, map[string]bool{"unlocked": true})
}
func (handler *Handler) ResetPassword(request *ghttp.Request) {
	id, ok := routerID(request, "id")
	var body passwordRequest
	if !ok || !decode(request, &body) {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	count, err := handler.service.ResetPassword(request.Context(), bearerToken(request), principal(request), id, body.TemporaryPassword)
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, map[string]any{"reset": true, "sessions_revoked": count, "force_password_change": true})
}
func (handler *Handler) ReplaceRoles(request *ghttp.Request) {
	id, ok := routerID(request, "id")
	var body rolesRequest
	if !ok || !decode(request, &body) {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	user, err := handler.service.ReplaceRoles(request.Context(), bearerToken(request), principal(request), id, body.RoleIDs)
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, user)
}
func (handler *Handler) ReplaceAssignments(request *ghttp.Request) {
	id, ok := routerID(request, "user_id")
	var body assignmentsRequest
	if !ok || !decode(request, &body) {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	user, err := handler.service.ReplaceAssignments(request.Context(), bearerToken(request), principal(request), id, userdomain.AssignmentInput{UnitIDs: body.UnitIDs, PrimaryUnitID: body.PrimaryUnitID, PositionIDs: body.PositionIDs, PrimaryPositionID: body.PrimaryPositionID})
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, user)
}
func (handler *Handler) Sessions(request *ghttp.Request) {
	id, ok := routerID(request, "id")
	if !ok {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	items, err := handler.service.Sessions(request.Context(), bearerToken(request), id)
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, map[string]any{"items": items, "total": len(items)})
}
func (handler *Handler) RevokeSession(request *ghttp.Request) {
	id, ok := routerID(request, "id")
	sessionID, sessionOK := routerID(request, "session_id")
	if !ok || !sessionOK {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	err := handler.service.RevokeSession(request.Context(), bearerToken(request), principal(request), id, sessionID)
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, map[string]bool{"revoked": true})
}
func (handler *Handler) Import(request *ghttp.Request) {
	result, err := handler.service.Import(request.Context(), bearerToken(request), principal(request), request.Body)
	if handler.writeError(request, err) {
		return
	}
	handler.success(request, 200, result)
}
func (handler *Handler) Export(request *ghttp.Request) {
	var body filtersRequest
	if !decode(request, &body) {
		handler.writeError(request, userdomain.ErrInvalid)
		return
	}
	var output bytes.Buffer
	err := handler.service.Export(request.Context(), bearerToken(request), body.filters(), &output)
	if handler.writeError(request, err) {
		return
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.Header().Set("Content-Type", "text/csv; charset=utf-8")
	request.Response.Header().Set("Content-Disposition", `attachment; filename="users.csv"`)
	request.Response.WriteHeader(200)
	request.Response.WriteExit(output.Bytes())
}

func queryFilters(request *ghttp.Request) (userdomain.Filters, bool) {
	page, err := positiveInt(request.GetQuery("page").String(), 1)
	if err != nil {
		return userdomain.Filters{}, false
	}
	pageSize, err := positiveInt(request.GetQuery("page_size").String(), 20)
	if err != nil {
		return userdomain.Filters{}, false
	}
	unitID, ok := optionalID(request.GetQuery("unit_id").String())
	if !ok {
		return userdomain.Filters{}, false
	}
	positionID, ok := optionalID(request.GetQuery("position_id").String())
	if !ok {
		return userdomain.Filters{}, false
	}
	roleID, ok := optionalID(request.GetQuery("role_id").String())
	if !ok {
		return userdomain.Filters{}, false
	}
	createdFrom, ok := optionalTime(request.GetQuery("created_from").String())
	if !ok {
		return userdomain.Filters{}, false
	}
	createdTo, ok := optionalTime(request.GetQuery("created_to").String())
	if !ok {
		return userdomain.Filters{}, false
	}
	return userdomain.Filters{Query: request.GetQuery("q").String(), Status: request.GetQuery("status").String(), UnitID: unitID, PositionID: positionID, RoleID: roleID, CreatedFrom: createdFrom, CreatedTo: createdTo, Page: int32(page), PageSize: int32(pageSize), Sort: request.GetQuery("sort").String()}, true
}
func (body filtersRequest) filters() userdomain.Filters {
	return userdomain.Filters{Query: body.Query, Status: body.Status, UnitID: body.UnitID, PositionID: body.PositionID, RoleID: body.RoleID, CreatedFrom: body.CreatedFrom, CreatedTo: body.CreatedTo, Sort: body.Sort}
}
func positiveInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, userdomain.ErrInvalid
	}
	return value, nil
}
func optionalID(raw string) (*uuid.UUID, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, false
	}
	return &id, true
}
func optionalTime(raw string) (*time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, false
	}
	return &value, true
}
func routerID(request *ghttp.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(request.GetRouter(name).String())
	return id, err == nil
}
func decode(request *ghttp.Request, target any) bool {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target) == nil && decoder.Decode(&struct{}{}) == io.EOF
}
func bearerToken(request *ghttp.Request) string {
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
func principal(request *ghttp.Request) userdomain.Principal {
	return userdomain.Principal{RequestID: httpx.RequestID(request), IPAddress: clientIP(request), UserAgent: request.Header.Get("User-Agent")}
}
func clientIP(request *ghttp.Request) *netip.Addr {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	address, err := netip.ParseAddr(strings.TrimSpace(host))
	if err != nil {
		return nil
	}
	return &address
}
func (handler *Handler) success(request *ghttp.Request, status int, data any) {
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.Header().Set("Vary", "Authorization, Accept-Language")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(request)})
}
func (handler *Handler) writeError(request *ghttp.Request, err error) bool {
	if err == nil {
		return false
	}
	status, code, key := 500, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(err, iamapp.ErrInvalidAccessToken):
		status, code, key = 401, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized"
	case errors.Is(err, userdomain.ErrForbidden):
		status, code, key = 403, "IAM.PERMISSION.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, userdomain.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, userdomain.ErrNotFound):
		status, code, key = 404, "IAM.USER.NOT_FOUND", "errors.iam.user.not_found"
	case errors.Is(err, userdomain.ErrEmailConflict):
		status, code, key = 409, "IAM.USER.EMAIL_EXISTS", "errors.iam.user.email_exists"
	case errors.Is(err, userdomain.ErrRoleInvalid):
		status, code, key = 422, "IAM.USER.ROLE_INVALID", "errors.iam.user.role_invalid"
	case errors.Is(err, userdomain.ErrOrgInvalid):
		status, code, key = 422, "IAM.USER.ORG_INVALID", "errors.iam.user.org_invalid"
	case errors.Is(err, userdomain.ErrLastAdmin):
		status, code, key = 409, "IAM.USER.LAST_ADMIN", "errors.iam.user.last_admin"
	case errors.Is(err, userdomain.ErrSessionAbsent):
		status, code, key = 404, "IAM.SESSION.NOT_FOUND", "errors.iam.session.not_found"
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: handler.catalog.Translate(httpx.Locale(request), key, nil)}, RequestID: httpx.RequestID(request)})
	return true
}
