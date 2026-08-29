package http

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"strconv"
	"strings"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	filesapp "github.com/appkernia/appkernia/server/internal/modules/storageadmin/application"
	files "github.com/appkernia/appkernia/server/internal/modules/storageadmin/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *filesapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *filesapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

type createUploadRequest struct {
	FileName  string `json:"file_name"`
	MediaType string `json:"media_type"`
	SizeBytes int64  `json:"size_bytes"`
}

func (h *Handler) CreateUpload(r *ghttp.Request) {
	var body createUploadRequest
	if !decode(r, &body) {
		h.fail(r, files.ErrInvalid)
		return
	}
	out, err := h.service.CreateUpload(r.Context(), token(r), httpx.RequestID(r), body.FileName, body.MediaType, body.SizeBytes)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UploadPolicy(r *ghttp.Request) {
	out, err := h.service.UploadPolicy(r.Context(), token(r))
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) GetUpload(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, files.ErrInvalid)
		return
	}
	out, err := h.service.GetUpload(r.Context(), token(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) UploadPart(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	number, e := strconv.Atoi(r.GetRouter("partNumber").String())
	if !ok || e != nil || number < 1 {
		h.fail(r, files.ErrInvalid)
		return
	}
	content, err := io.ReadAll(io.LimitReader(r.Body, files.PartSize+1))
	if err != nil || int64(len(content)) > files.PartSize {
		h.fail(r, files.ErrInvalid)
		return
	}
	out, err := h.service.UploadPart(r.Context(), token(r), id, int32(number), content)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CancelUpload(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, files.ErrInvalid)
		return
	}
	err := h.service.CancelUpload(r.Context(), token(r), httpx.RequestID(r), clientIP(r), r.Header.Get("User-Agent"), id)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]bool{"cancelled": true})
	}
}
func (h *Handler) CompleteUpload(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	var empty struct{}
	if !ok || !decodeOptional(r, &empty) {
		h.fail(r, files.ErrInvalid)
		return
	}
	out, err := h.service.CompleteUpload(r.Context(), token(r), httpx.RequestID(r), clientIP(r), r.Header.Get("User-Agent"), id)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) List(r *ghttp.Request) {
	page, e1 := strconv.Atoi(r.GetQuery("page", 1).String())
	size, e2 := strconv.Atoi(r.GetQuery("page_size", 20).String())
	createdFrom, e3 := optionalRFC3339(r.GetQuery("created_from").String())
	createdTo, e4 := optionalRFC3339(r.GetQuery("created_to").String())
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		h.fail(r, files.ErrInvalid)
		return
	}
	out, err := h.service.ListFiles(r.Context(), token(r), files.FileFilter{Query: r.GetQuery("q").String(), Status: r.GetQuery("status").String(), ScanStatus: r.GetQuery("scan_status").String(), MediaType: r.GetQuery("media_type").String(), Provider: r.GetQuery("provider").String(), CreatedFrom: createdFrom, CreatedTo: createdTo, Page: int32(page), PageSize: int32(size)})
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func optionalRFC3339(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
func (h *Handler) Get(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, files.ErrInvalid)
		return
	}
	out, err := h.service.GetFile(r.Context(), token(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Usages(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, files.ErrInvalid)
		return
	}
	out, err := h.service.ListUsages(r.Context(), token(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]any{"items": out})
	}
}
func (h *Handler) PresignDownload(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, files.ErrInvalid)
		return
	}
	file, reader, err := h.service.OpenDownload(r.Context(), token(r), id)
	if reader != nil {
		_ = reader.Close()
	}
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]any{"file_id": file.ID, "method": "GET", "download_url": "/files/" + file.ID.String() + "/content", "expires_at": time.Now().UTC().Add(5 * time.Minute)})
	}
}
func (h *Handler) Download(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, files.ErrInvalid)
		return
	}
	file, reader, err := h.service.OpenDownload(r.Context(), token(r), id)
	if h.fail(r, err) {
		return
	}
	defer func() { _ = reader.Close() }()
	r.Response.Header().Set("Content-Type", file.MediaType)
	r.Response.Header().Set("Content-Length", fmt.Sprintf("%d", file.SizeBytes))
	r.Response.Header().Set("Content-Disposition", `attachment; filename="download"; filename*=UTF-8''`+urlName(file.OriginalName))
	r.Response.Header().Set("Cache-Control", "private, no-store")
	r.Response.Header().Set("X-Content-Type-Options", "nosniff")
	if len(file.SHA256) > 0 {
		r.Response.Header().Set("ETag", `"`+hex.EncodeToString(file.SHA256)+`"`)
	}
	_, _ = io.Copy(r.Response.BufferWriter, reader)
}
func (h *Handler) Delete(r *ghttp.Request) {
	id, ok := routerID(r, "id")
	if !ok {
		h.fail(r, files.ErrInvalid)
		return
	}
	err := h.service.DeleteFile(r.Context(), token(r), httpx.RequestID(r), clientIP(r), r.Header.Get("User-Agent"), id)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
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
	case errors.Is(err, files.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, files.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, files.ErrNotFound), errors.Is(err, files.ErrUploadNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, files.ErrUploadIncomplete):
		status, code, key = 409, "STORAGE.UPLOAD.INCOMPLETE", "errors.common.conflict"
	case errors.Is(err, files.ErrScanBlocked):
		status, code, key = 409, "STORAGE.SCAN.BLOCKED", "errors.common.conflict"
	case errors.Is(err, files.ErrFileInUse):
		status, code, key = 409, "STORAGE.FILE.IN_USE", "errors.common.conflict"
	case errors.Is(err, files.ErrFeatureDisabled), errors.Is(err, files.ErrStorageConfig):
		status, code, key = 503, "STORAGE.UNAVAILABLE", "errors.common.unknown"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
func decode(r *ghttp.Request, target any) bool {
	d := json.NewDecoder(io.LimitReader(r.Body, 64*1024))
	d.DisallowUnknownFields()
	return d.Decode(target) == nil && d.Decode(&struct{}{}) == io.EOF
}
func decodeOptional(r *ghttp.Request, target any) bool {
	if r.ContentLength == 0 {
		return true
	}
	return decode(r, target)
}
func routerID(r *ghttp.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.GetRouter(key).String())
	return id, err == nil
}
func token(r *ghttp.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
func clientIP(r *ghttp.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return strings.TrimSpace(host)
}
func urlName(value string) string {
	var b strings.Builder
	for _, x := range []byte(value) {
		if x >= 'a' && x <= 'z' || x >= 'A' && x <= 'Z' || x >= '0' && x <= '9' || strings.ContainsRune("-_.~", rune(x)) {
			b.WriteByte(x)
		} else {
			fmt.Fprintf(&b, "%%%02X", x)
		}
	}
	return b.String()
}

var _ = stdhttp.StatusOK
