package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	jobapp "github.com/appkernia/appkernia/server/internal/modules/jobadmin/application"
	jobs "github.com/appkernia/appkernia/server/internal/modules/jobadmin/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *jobapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *jobapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func (h *Handler) Handlers(r *ghttp.Request) {
	out, err := h.service.Handlers(r.Context(), token(r))
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) Preview(r *ghttp.Request) {
	var body jobs.ScheduleInput
	if !decode(r, &body) {
		h.fail(r, jobs.ErrInvalid)
		return
	}
	out, err := h.service.Preview(r.Context(), token(r), body)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) List(r *ghttp.Request) {
	filter, ok := pageFilter(r)
	if !ok {
		h.fail(r, jobs.ErrInvalid)
		return
	}
	filter.Query = r.GetQuery("q").String()
	filter.Status = r.GetQuery("status").String()
	filter.TimeZone = r.GetQuery("time_zone").String()
	out, err := h.service.List(r.Context(), token(r), filter)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) Create(r *ghttp.Request) {
	var body jobs.ScheduleInput
	if !decode(r, &body) {
		h.fail(r, jobs.ErrInvalid)
		return
	}
	out, err := h.service.Create(r.Context(), token(r), requestPrincipal(r), body)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}

func (h *Handler) Update(r *ghttp.Request) {
	id, ok := routerID(r)
	var body jobs.ScheduleInput
	if !ok || !decode(r, &body) {
		h.fail(r, jobs.ErrInvalid)
		return
	}
	out, err := h.service.Update(r.Context(), token(r), requestPrincipal(r), id, body)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) Pause(r *ghttp.Request)  { h.changeStatus(r, "paused") }
func (h *Handler) Resume(r *ghttp.Request) { h.changeStatus(r, "active") }

func (h *Handler) changeStatus(r *ghttp.Request, target string) {
	id, ok := routerID(r)
	var body struct{}
	if !ok || !decodeOptional(r, &body) {
		h.fail(r, jobs.ErrInvalid)
		return
	}
	out, err := h.service.ChangeStatus(r.Context(), token(r), requestPrincipal(r), id, target)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) Execute(r *ghttp.Request) {
	id, ok := routerID(r)
	var body struct{}
	if !ok || !decodeOptional(r, &body) {
		h.fail(r, jobs.ErrInvalid)
		return
	}
	out, err := h.service.Execute(r.Context(), token(r), requestPrincipal(r), id, r.Header.Get("Idempotency-Key"))
	if !h.fail(r, err) {
		h.ok(r, 202, out)
	}
}

func (h *Handler) Runs(r *ghttp.Request) {
	id, ok := routerID(r)
	filter, pageOK := pageFilter(r)
	if !ok || !pageOK {
		h.fail(r, jobs.ErrInvalid)
		return
	}
	filter.Status = r.GetQuery("status").String()
	out, err := h.service.Runs(r.Context(), token(r), id, filter)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func pageFilter(r *ghttp.Request) (jobs.PageFilter, bool) {
	page, pageErr := strconv.Atoi(r.GetQuery("page", 1).String())
	size, sizeErr := strconv.Atoi(r.GetQuery("page_size", 20).String())
	return jobs.PageFilter{Page: int32(page), PageSize: int32(size)}, pageErr == nil && sizeErr == nil
}

func requestPrincipal(r *ghttp.Request) jobs.Principal {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return jobs.Principal{RequestID: httpx.RequestID(r), IPAddress: strings.TrimSpace(host), UserAgent: strings.TrimSpace(r.Header.Get("User-Agent"))}
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
	case errors.Is(err, jobs.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, jobs.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, jobs.ErrHandlerUnknown):
		status, code, key = 422, "JOBS.HANDLER.UNKNOWN", "errors.validation.failed"
	case errors.Is(err, jobs.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, jobs.ErrConflict):
		status, code, key = 409, "JOBS.SCHEDULE.CONFLICT", "errors.common.conflict"
	case errors.Is(err, jobs.ErrTransition):
		status, code, key = 409, "JOBS.SCHEDULE.TRANSITION_NOT_ALLOWED", "errors.common.conflict"
	case errors.Is(err, jobs.ErrExecutionDenied):
		status, code, key = 409, "JOBS.SCHEDULE.EXECUTION_NOT_ALLOWED", "errors.common.conflict"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}

func decode(r *ghttp.Request, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 128*1024))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target) == nil && decoder.Decode(&struct{}{}) == io.EOF
}

func decodeOptional(r *ghttp.Request, target any) bool {
	return r.ContentLength == 0 || decode(r, target)
}

func routerID(r *ghttp.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.GetRouter("id").String())
	return id, err == nil
}

func token(r *ghttp.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
