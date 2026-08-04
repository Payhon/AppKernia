package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	stdhttp "net/http"
	"net/netip"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	orgapp "github.com/appkernia/appkernia/server/internal/modules/org/application"
	orgdomain "github.com/appkernia/appkernia/server/internal/modules/org/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *orgapp.Service
	catalog *i18n.Catalog
}

type unitRequest struct {
	ParentID  *uuid.UUID `json:"parent_id"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	UnitType  string     `json:"unit_type"`
	Phone     string     `json:"phone"`
	Email     string     `json:"email"`
	SortOrder int32      `json:"sort_order"`
	Status    string     `json:"status"`
}
type moveRequest struct {
	ParentID  *uuid.UUID `json:"parent_id"`
	SortOrder int32      `json:"sort_order"`
}
type positionRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int32  `json:"sort_order"`
	Status      string `json:"status"`
}

func NewHandler(service *orgapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func (handler *Handler) UnitTree(request *ghttp.Request) {
	units, err := handler.service.ListUnits(request.Context(), bearerToken(request), request.GetQuery("q").String(), request.GetQuery("status").String())
	if handler.writeError(request, err, nil) {
		return
	}
	handler.success(request, stdhttp.StatusOK, mapUnits(units))
}
func (handler *Handler) CreateUnit(request *ghttp.Request) {
	var body unitRequest
	if !decode(request, &body) {
		handler.writeError(request, orgdomain.ErrInvalid, nil)
		return
	}
	unit, err := handler.service.CreateUnit(request.Context(), bearerToken(request), principal(request), body.unit())
	if handler.writeError(request, err, nil) {
		return
	}
	handler.success(request, stdhttp.StatusCreated, mapUnit(unit))
}
func (handler *Handler) UpdateUnit(request *ghttp.Request) {
	id, ok := pathID(request)
	if !ok {
		handler.writeError(request, orgdomain.ErrInvalid, nil)
		return
	}
	var body unitRequest
	if !decode(request, &body) {
		handler.writeError(request, orgdomain.ErrInvalid, nil)
		return
	}
	unit, err := handler.service.UpdateUnit(request.Context(), bearerToken(request), principal(request), id, body.unit())
	if handler.writeError(request, err, nil) {
		return
	}
	handler.success(request, stdhttp.StatusOK, mapUnit(unit))
}
func (handler *Handler) MoveUnit(request *ghttp.Request) {
	id, ok := pathID(request)
	if !ok {
		handler.writeError(request, orgdomain.ErrInvalid, nil)
		return
	}
	var body moveRequest
	if !decode(request, &body) {
		handler.writeError(request, orgdomain.ErrInvalid, nil)
		return
	}
	unit, err := handler.service.MoveUnit(request.Context(), bearerToken(request), principal(request), id, orgdomain.UnitMove{ParentID: body.ParentID, SortOrder: body.SortOrder})
	if handler.writeError(request, err, nil) {
		return
	}
	handler.success(request, stdhttp.StatusOK, mapUnit(unit))
}
func (handler *Handler) DeleteUnit(request *ghttp.Request) {
	id, ok := pathID(request)
	if !ok {
		handler.writeError(request, orgdomain.ErrInvalid, nil)
		return
	}
	occupancy, err := handler.service.DeleteUnit(request.Context(), bearerToken(request), principal(request), id)
	details := map[string]any{"child_count": occupancy.ChildCount, "member_count": occupancy.MemberCount}
	if handler.writeError(request, err, details) {
		return
	}
	handler.success(request, stdhttp.StatusOK, map[string]any{"deleted": true})
}

func (handler *Handler) Positions(request *ghttp.Request) {
	var unitID *uuid.UUID
	if raw := strings.TrimSpace(request.GetQuery("unit_id").String()); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			handler.writeError(request, orgdomain.ErrInvalid, nil)
			return
		}
		unitID = &parsed
	}
	positions, err := handler.service.ListPositions(request.Context(), bearerToken(request), request.GetQuery("q").String(), request.GetQuery("status").String(), unitID)
	if handler.writeError(request, err, nil) {
		return
	}
	items := make([]map[string]any, 0, len(positions))
	for _, item := range positions {
		items = append(items, mapPosition(item))
	}
	handler.success(request, stdhttp.StatusOK, map[string]any{"items": items, "total": len(items)})
}
func (handler *Handler) CreatePosition(request *ghttp.Request) {
	var body positionRequest
	if !decode(request, &body) {
		handler.writeError(request, orgdomain.ErrInvalid, nil)
		return
	}
	position, err := handler.service.CreatePosition(request.Context(), bearerToken(request), principal(request), body.position())
	if handler.writeError(request, err, nil) {
		return
	}
	handler.success(request, stdhttp.StatusCreated, mapPosition(position))
}
func (handler *Handler) UpdatePosition(request *ghttp.Request) {
	id, ok := pathID(request)
	if !ok {
		handler.writeError(request, orgdomain.ErrInvalid, nil)
		return
	}
	var body positionRequest
	if !decode(request, &body) {
		handler.writeError(request, orgdomain.ErrInvalid, nil)
		return
	}
	position, err := handler.service.UpdatePosition(request.Context(), bearerToken(request), principal(request), id, body.position())
	if handler.writeError(request, err, nil) {
		return
	}
	handler.success(request, stdhttp.StatusOK, mapPosition(position))
}
func (handler *Handler) DeletePosition(request *ghttp.Request) {
	id, ok := pathID(request)
	if !ok {
		handler.writeError(request, orgdomain.ErrInvalid, nil)
		return
	}
	count, err := handler.service.DeletePosition(request.Context(), bearerToken(request), principal(request), id)
	if handler.writeError(request, err, map[string]any{"member_count": count}) {
		return
	}
	handler.success(request, stdhttp.StatusOK, map[string]any{"deleted": true})
}

func (body unitRequest) unit() orgdomain.UnitInput {
	return orgdomain.UnitInput{ParentID: body.ParentID, Code: body.Code, Name: body.Name, UnitType: body.UnitType, Phone: body.Phone, Email: body.Email, SortOrder: body.SortOrder, Status: body.Status}
}
func (body positionRequest) position() orgdomain.PositionInput {
	return orgdomain.PositionInput{Code: body.Code, Name: body.Name, Description: body.Description, SortOrder: body.SortOrder, Status: body.Status}
}
func mapUnits(units []orgdomain.Unit) []map[string]any {
	result := make([]map[string]any, 0, len(units))
	for _, unit := range units {
		result = append(result, mapUnit(unit))
	}
	return result
}
func mapUnit(unit orgdomain.Unit) map[string]any {
	return map[string]any{"id": unit.ID, "parent_id": unit.ParentID, "code": unit.Code, "name": unit.Name, "unit_type": unit.UnitType, "phone": unit.Phone, "email": unit.Email, "sort_order": unit.SortOrder, "status": unit.Status, "direct_member_count": unit.DirectMemberCount, "child_count": unit.ChildCount, "updated_at": unit.UpdatedAt, "children": mapUnits(unit.Children)}
}
func mapPosition(item orgdomain.Position) map[string]any {
	return map[string]any{"id": item.ID, "code": item.Code, "name": item.Name, "description": item.Description, "sort_order": item.SortOrder, "status": item.Status, "member_count": item.MemberCount, "updated_at": item.UpdatedAt}
}

func (handler *Handler) success(request *ghttp.Request, status int, data any) {
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.Header().Set("Vary", "Authorization, Accept-Language")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(request)})
}
func (handler *Handler) writeError(request *ghttp.Request, err error, details map[string]any) bool {
	if err == nil {
		return false
	}
	status, code, key := stdhttp.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(err, iamapp.ErrInvalidAccessToken):
		status, code, key = 401, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized"
	case errors.Is(err, orgdomain.ErrForbidden):
		status, code, key = 403, "ORG.PERMISSION.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, orgdomain.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, orgdomain.ErrUnitNotFound):
		status, code, key = 404, "ORG.UNIT.NOT_FOUND", "errors.org.unit.not_found"
	case errors.Is(err, orgdomain.ErrUnitCodeConflict):
		status, code, key = 409, "ORG.UNIT.CODE_CONFLICT", "errors.org.unit.code_conflict"
	case errors.Is(err, orgdomain.ErrUnitCycle):
		status, code, key = 409, "ORG.UNIT.CYCLE", "errors.org.unit.cycle"
	case errors.Is(err, orgdomain.ErrUnitOccupied):
		status, code, key = 409, "ORG.UNIT.OCCUPIED", "errors.org.unit.occupied"
	case errors.Is(err, orgdomain.ErrPositionNotFound):
		status, code, key = 404, "ORG.POSITION.NOT_FOUND", "errors.org.position.not_found"
	case errors.Is(err, orgdomain.ErrPositionConflict):
		status, code, key = 409, "ORG.POSITION.CODE_CONFLICT", "errors.org.position.code_conflict"
	case errors.Is(err, orgdomain.ErrPositionOccupied):
		status, code, key = 409, "ORG.POSITION.OCCUPIED", "errors.org.position.occupied"
	}
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: handler.catalog.Translate(httpx.Locale(request), key, nil), Details: details}, RequestID: httpx.RequestID(request)})
	return true
}
func decode(request *ghttp.Request, target any) bool {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target) == nil && decoder.Decode(&struct{}{}) == io.EOF
}
func pathID(request *ghttp.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(request.GetRouter("id").String())
	return id, err == nil
}
func bearerToken(request *ghttp.Request) string {
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
func principal(request *ghttp.Request) orgdomain.Principal {
	return orgdomain.Principal{RequestID: httpx.RequestID(request), IPAddress: clientIP(request), UserAgent: request.Header.Get("User-Agent")}
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
