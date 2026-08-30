package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"slices"
	"strings"

	clients "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/domain"
	"github.com/appkernia/appkernia/server/internal/platform/notification"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type MachineAuthenticator interface {
	Authenticate(context.Context, string, uuid.UUID, string, string) (clients.MachinePrincipal, error)
}

type Handler struct {
	auth    MachineAuthenticator
	service notification.Service
	catalog *i18n.Catalog
}

func NewHandler(auth MachineAuthenticator, service notification.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{auth: auth, service: service, catalog: catalog}
}

func (h *Handler) Submit(r *ghttp.Request) {
	appID, ok := routeID(r, "app_id")
	if !ok {
		h.fail(r, notification.ErrInvalid)
		return
	}
	var command notification.SubmitCommand
	if !decode(r, &command) {
		h.fail(r, notification.ErrInvalid)
		return
	}
	command.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	principal, err := h.auth.Authenticate(r.Context(), bearer(r), appID, "notify.message.submit", remoteIP(r))
	if err != nil {
		h.fail(r, err)
		return
	}
	if command.Audience.Type == "all_active_app_users" && !slices.Contains(principal.Permissions, "notify.message.broadcast") ||
		command.Category == "news_operations" && !slices.Contains(principal.Permissions, "notify.operations.publish") {
		h.fail(r, clients.ErrForbidden)
		return
	}
	scope := notification.Scope{TenantID: principal.TenantID, AppID: appID, ActorKind: "api_client", ActorID: principal.ClientID,
		RequestID: httpx.RequestID(r), SourceIP: remoteIP(r), UserAgent: strings.TrimSpace(r.Header.Get("User-Agent"))}
	out, err := h.service.Submit(r.Context(), scope, command)
	if err != nil {
		h.fail(r, err)
		return
	}
	statusURL := "/api/v1/apps/" + appID.String() + "/notifications/" + out.MessageID.String()
	r.Response.Header().Set("Location", statusURL)
	h.ok(r, 202, map[string]any{"message_id": out.MessageID, "run_id": out.RunID, "status": out.Status, "status_url": statusURL, "created_at": out.CreatedAt})
}

func (h *Handler) Status(r *ghttp.Request) {
	appID, appOK := routeID(r, "app_id")
	messageID, messageOK := routeID(r, "message_id")
	if !appOK || !messageOK {
		h.fail(r, notification.ErrInvalid)
		return
	}
	principal, err := h.auth.Authenticate(r.Context(), bearer(r), appID, "notify.message.status.read", remoteIP(r))
	if err != nil {
		h.fail(r, err)
		return
	}
	out, err := h.service.Status(r.Context(), notification.Scope{TenantID: principal.TenantID, AppID: appID, ActorKind: "api_client", ActorID: principal.ClientID, RequestID: httpx.RequestID(r)}, messageID)
	if err != nil {
		h.fail(r, err)
		return
	}
	h.ok(r, 200, out)
}

func (h *Handler) Cancel(r *ghttp.Request) {
	appID, appOK := routeID(r, "app_id")
	messageID, messageOK := routeID(r, "message_id")
	if !appOK || !messageOK {
		h.fail(r, notification.ErrInvalid)
		return
	}
	principal, err := h.auth.Authenticate(r.Context(), bearer(r), appID, "notify.message.submit", remoteIP(r))
	if err != nil {
		h.fail(r, err)
		return
	}
	err = h.service.Cancel(r.Context(), notification.Scope{TenantID: principal.TenantID, AppID: appID, ActorKind: "api_client", ActorID: principal.ClientID, RequestID: httpx.RequestID(r)}, messageID)
	if err != nil {
		h.fail(r, err)
		return
	}
	h.ok(r, 200, map[string]any{"cancelled": true, "message_id": messageID})
}

func (h *Handler) ok(r *ghttp.Request, status int, data any) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(r)})
}

func (h *Handler) fail(r *ghttp.Request, err error) {
	status, code, key := 500, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(err, clients.ErrCredential):
		status, code, key = 401, "AUTH.CLIENT.INVALID", "errors.common.unauthorized"
	case errors.Is(err, clients.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, notification.ErrInvalid):
		status, code, key = 422, "NOTIFY.SUBMISSION.INVALID", "errors.validation.failed"
	case errors.Is(err, notification.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, notification.ErrIdempotencyConflict):
		status, code, key = 409, "NOTIFY.IDEMPOTENCY.CONFLICT", "errors.common.conflict"
	case errors.Is(err, notification.ErrConflict):
		status, code, key = 409, "NOTIFY.SUBMISSION.CONFLICT", "errors.common.conflict"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
}

func decode(r *ghttp.Request, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 256*1024))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target) == nil && decoder.Decode(&struct{}{}) == io.EOF
}

func routeID(r *ghttp.Request, key string) (uuid.UUID, bool) {
	value, err := uuid.Parse(r.GetRouter(key).String())
	return value, err == nil
}

func bearer(r *ghttp.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func remoteIP(r *ghttp.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return strings.TrimSpace(host)
}
