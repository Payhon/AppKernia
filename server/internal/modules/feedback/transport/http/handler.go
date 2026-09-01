package http

import (
	"encoding/json"
	"errors"
	app "github.com/appkernia/appkernia/server/internal/modules/feedback/application"
	f "github.com/appkernia/appkernia/server/internal/modules/feedback/domain"
	iam "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	service *app.Service
	catalog *i18n.Catalog
}

func NewHandler(s *app.Service, c *i18n.Catalog) *Handler { return &Handler{s, c} }
func (h *Handler) fail(r *ghttp.Request, e error) bool {
	if e == nil {
		return false
	}
	status, code, key := 500, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(e, iam.ErrInvalidAccessToken):
		status, code, key = 401, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized"
	case errors.Is(e, f.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(e, f.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(e, f.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(e, f.ErrConflict):
		status, code, key = 409, "COMMON.CONFLICT", "errors.common.conflict"
	case errors.Is(e, f.ErrStorage):
		status, code, key = 503, "STORAGE.UNAVAILABLE", "errors.storage.feedback.unavailable"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
func (h *Handler) ok(r *ghttp.Request, status int, data any) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(r)})
}
func decode(r *ghttp.Request, x any) bool {
	d := json.NewDecoder(io.LimitReader(r.Body, 65537))
	d.DisallowUnknownFields()
	return d.Decode(x) == nil && d.Decode(&struct{}{}) == io.EOF
}
func id(r *ghttp.Request, key string) uuid.UUID {
	x, _ := uuid.Parse(r.GetRouter(key).String())
	return x
}
func (h *Handler) scope(r *ghttp.Request, permission string) (f.Scope, bool) {
	admin := strings.HasPrefix(r.URL.Path, "/admin-api/")
	appID := id(r, "app_id")
	if !admin {
		appID, _ = uuid.Parse(r.Header.Get("X-AppID"))
		permission = ""
	}
	auth := strings.Fields(r.Header.Get("Authorization"))
	token := ""
	if len(auth) == 2 && strings.EqualFold(auth[0], "Bearer") {
		token = auth[1]
	}
	p, e := h.service.Scope(r.Context(), token, appID, permission, httpx.RequestID(r))
	return p, !h.fail(r, e)
}
func (h *Handler) Collection(r *ghttp.Request) {
	p, ok := h.scope(r, "app.feedback.read")
	if !ok {
		return
	}
	if r.Method == "POST" {
		var x f.Input
		key, e := uuid.Parse(r.Header.Get("Idempotency-Key"))
		if e != nil || !decode(r, &x) {
			h.fail(r, f.ErrInvalid)
			return
		}
		out, e := h.service.Create(r.Context(), p, x, key)
		if !h.fail(r, e) {
			h.ok(r, 201, out)
		}
		return
	}
	page, e := strconv.ParseInt(r.GetQuery("page", 1).String(), 10, 32)
	size, e2 := strconv.ParseInt(r.GetQuery("page_size", 20).String(), 10, 32)
	if e != nil || e2 != nil {
		h.fail(r, f.ErrInvalid)
		return
	}
	q := f.Filter{Query: r.GetQuery("q").String(), Status: r.GetQuery("status").String(), Page: int32(page), PageSize: int32(size)}
	for key, target := range map[string]**time.Time{"created_from": &q.From, "created_to": &q.To} {
		if raw := r.GetQuery(key).String(); raw != "" {
			t, e := time.Parse(time.RFC3339, raw)
			if e != nil {
				h.fail(r, f.ErrInvalid)
				return
			}
			*target = &t
		}
	}
	out, e := h.service.List(r.Context(), p, q)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Detail(r *ghttp.Request) {
	permission := "app.feedback.read"
	if r.Method == "PATCH" {
		permission = "app.feedback.update"
	}
	p, ok := h.scope(r, permission)
	if !ok {
		return
	}
	feedbackID := id(r, "id")
	if feedbackID == uuid.Nil {
		h.fail(r, f.ErrInvalid)
		return
	}
	if r.Method == "PATCH" {
		var x f.StatusInput
		if !decode(r, &x) {
			h.fail(r, f.ErrInvalid)
			return
		}
		out, e := h.service.Change(r.Context(), p, feedbackID, x)
		if !h.fail(r, e) {
			h.ok(r, 200, out)
		}
		return
	}
	out, e := h.service.Get(r.Context(), p, feedbackID)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Reply(r *ghttp.Request) {
	p, ok := h.scope(r, "app.feedback.reply")
	if !ok {
		return
	}
	var x f.ReplyInput
	key, e := uuid.Parse(r.Header.Get("Idempotency-Key"))
	if e != nil || !decode(r, &x) {
		h.fail(r, f.ErrInvalid)
		return
	}
	out, e := h.service.Reply(r.Context(), p, id(r, "id"), x, key)
	if !h.fail(r, e) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) CreateUpload(r *ghttp.Request) {
	p, ok := h.scope(r, "")
	if !ok {
		return
	}
	var x f.UploadInput
	if !decode(r, &x) {
		h.fail(r, f.ErrInvalid)
		return
	}
	out, e := h.service.CreateUpload(r.Context(), p, x)
	if !h.fail(r, e) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) CancelUpload(r *ghttp.Request) {
	p, ok := h.scope(r, "")
	if !ok {
		return
	}
	if !h.fail(r, h.service.CancelUpload(r.Context(), p, id(r, "id"))) {
		h.ok(r, 200, map[string]bool{"cancelled": true})
	}
}
func (h *Handler) Upload(r *ghttp.Request) {
	p, ok := h.scope(r, "")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(nil, r.Body, f.MaxImageBytes+65536)
	reader, e := r.MultipartReader()
	if e != nil {
		h.fail(r, f.ErrInvalid)
		return
	}
	part, e := reader.NextPart()
	if e != nil || part.FormName() != "file" {
		h.fail(r, f.ErrInvalid)
		return
	}
	data, e := io.ReadAll(io.LimitReader(part, f.MaxImageBytes+1))
	_ = part.Close()
	if e != nil || int64(len(data)) > f.MaxImageBytes {
		h.fail(r, f.ErrInvalid)
		return
	}
	if _, e = reader.NextPart(); e != io.EOF {
		h.fail(r, f.ErrInvalid)
		return
	}
	fileID, e := h.service.Upload(r.Context(), p, id(r, "id"), data)
	if !h.fail(r, e) {
		h.ok(r, 200, map[string]uuid.UUID{"file_id": fileID})
	}
}
func (h *Handler) File(r *ghttp.Request) {
	p, ok := h.scope(r, "app.feedback.read")
	if !ok {
		return
	}
	x, reader, e := h.service.OpenFile(r.Context(), p, id(r, "id"), id(r, "file_id"))
	if h.fail(r, e) {
		return
	}
	defer func() { _ = reader.Close() }()
	r.Response.Header().Set("Content-Type", x.MediaType)
	r.Response.Header().Set("Cache-Control", "private, no-store")
	r.Response.Header().Set("X-Content-Type-Options", "nosniff")
	r.Response.Header().Set("Content-Length", strconv.FormatInt(x.SizeBytes, 10))
	_, _ = io.Copy(r.Response.BufferWriter, reader)
}
