// Package http adapts the content application service to the two HTTP
// audiences. It intentionally contains no database objects or SQL.
package http

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	contentapp "github.com/appkernia/appkernia/server/internal/modules/content/application"
	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *contentapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *contentapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func (h *Handler) Articles(r *ghttp.Request) {
	featured, ok := optionalBool(r, "featured")
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	limit, e := strconv.Atoi(r.GetQuery("limit", 20).String())
	if e != nil {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, e := h.service.ListPublished(r.Context(), token(r), locale(r), content.PublicFilter{Query: r.GetQuery("q").String(), CategorySlug: r.GetQuery("category").String(), Featured: featured, Cursor: r.GetQuery("cursor").String(), Limit: int32(limit)})
	if !h.fail(r, e) {
		h.okMeta(r, 200, map[string]any{"items": out.Items}, map[string]any{"next_cursor": out.NextCursor})
	}
}
func (h *Handler) ArticleCategories(r *ghttp.Request) {
	out, e := h.service.ListPublishedCategories(r.Context(), token(r), locale(r))
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) ArticleAsset(r *ghttp.Request) {
	id, ok := idAt(r, "file_id")
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	asset, reader, e := h.service.OpenArticleAsset(r.Context(), token(r), id)
	if h.fail(r, e) {
		return
	}
	defer func() { _ = reader.Close() }()
	digest := asset.SHA256
	if len(digest) == 0 {
		fallback := sha256.Sum256([]byte(asset.FileID.String()))
		digest = fallback[:]
	}
	r.Response.Header().Set("Content-Type", asset.MediaType)
	r.Response.Header().Set("Content-Length", fmt.Sprintf("%d", asset.SizeBytes))
	r.Response.Header().Set("Cache-Control", "private, max-age=300")
	r.Response.Header().Set("ETag", `"`+hex.EncodeToString(digest)+`"`)
	r.Response.Header().Set("X-Content-Type-Options", "nosniff")
	r.Response.Header().Set("Vary", "Authorization")
	_, _ = io.Copy(r.Response.BufferWriter, reader)
}
func (h *Handler) Article(r *ghttp.Request) {
	out, e := h.service.GetPublished(r.Context(), token(r), locale(r), r.GetRouter("slug").String())
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Bookmark(r *ghttp.Request) {
	id, ok := idAt(r, "article_id")
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	if !h.fail(r, h.service.Bookmark(r.Context(), token(r), id)) {
		h.ok(r, 200, map[string]bool{"bookmarked": true})
	}
}
func (h *Handler) RemoveBookmark(r *ghttp.Request) {
	id, ok := idAt(r, "article_id")
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	if !h.fail(r, h.service.RemoveBookmark(r.Context(), token(r), id)) {
		h.ok(r, 200, map[string]bool{"bookmarked": false})
	}
}

func (h *Handler) Categories(r *ghttp.Request) {
	f, ok := pageFilter(r)
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	f.Status = r.GetQuery("status").String()
	f.Sort = r.GetQuery("sort").String()
	out, e := h.service.ListCategories(r.Context(), token(r), appID(r), f)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Category(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, e := h.service.GetCategory(r.Context(), token(r), appID(r), id)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CreateCategory(r *ghttp.Request) {
	var x content.Category
	if !decode(r, &x) {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, e := h.service.CreateCategory(r.Context(), token(r), appID(r), principal(r), x)
	if !h.fail(r, e) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateCategory(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	var x content.Category
	if !ok || !decode(r, &x) {
		h.fail(r, content.ErrInvalid)
		return
	}
	x.ID = id
	out, e := h.service.UpdateCategory(r.Context(), token(r), appID(r), principal(r), x)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) DeleteCategory(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	v, e := strconv.Atoi(r.GetQuery("lock_version").String())
	if !ok || e != nil {
		h.fail(r, content.ErrInvalid)
		return
	}
	if !h.fail(r, h.service.DeleteCategory(r.Context(), token(r), appID(r), principal(r), id, int32(v))) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
}

func (h *Handler) AdminArticles(r *ghttp.Request) {
	f, ok := pageFilter(r)
	featured, featuredOK := optionalBool(r, "featured")
	category, categoryOK := optionalID(r, "category_id")
	topic, topicOK := optionalID(r, "topic_id")
	if !ok || !featuredOK || !categoryOK || !topicOK {
		h.fail(r, content.ErrInvalid)
		return
	}
	f.Status, f.Sort, f.Featured, f.CategoryID, f.TopicID, f.ContentType = r.GetQuery("status").String(), r.GetQuery("sort").String(), featured, category, topic, r.GetQuery("type").String()
	out, e := h.service.ListArticles(r.Context(), token(r), appID(r), f)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) AdminArticle(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, e := h.service.GetArticle(r.Context(), token(r), appID(r), id)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CreateArticle(r *ghttp.Request) {
	var x content.Article
	if !decode(r, &x) {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, e := h.service.CreateArticle(r.Context(), token(r), appID(r), principal(r), x)
	if !h.fail(r, e) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateArticle(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	var x content.Article
	if !ok || !decode(r, &x) {
		h.fail(r, content.ErrInvalid)
		return
	}
	x.ID = id
	out, e := h.service.UpdateArticle(r.Context(), token(r), appID(r), principal(r), x)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) DeleteArticle(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	v, e := strconv.Atoi(r.GetQuery("lock_version").String())
	if !ok || e != nil {
		h.fail(r, content.ErrInvalid)
		return
	}
	if !h.fail(r, h.service.DeleteArticle(r.Context(), token(r), appID(r), principal(r), id, int32(v))) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
}
func (h *Handler) Publish(r *ghttp.Request)   { h.transition(r, "published") }
func (h *Handler) Unpublish(r *ghttp.Request) { h.transition(r, "draft") }
func (h *Handler) Archive(r *ghttp.Request)   { h.transition(r, "archived") }
func (h *Handler) transition(r *ghttp.Request, state string) {
	id, ok := idAt(r, "id")
	var input struct {
		LockVersion int32 `json:"lock_version"`
	}
	if !ok || !decode(r, &input) {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, e := h.service.TransitionArticle(r.Context(), token(r), appID(r), principal(r), id, input.LockVersion, state)
	if !h.fail(r, e) {
		h.ok(r, 200, out)
	}
}

func pageFilter(r *ghttp.Request) (content.PageFilter, bool) {
	page, e1 := strconv.Atoi(r.GetQuery("page", 1).String())
	size, e2 := strconv.Atoi(r.GetQuery("page_size", 20).String())
	return content.PageFilter{Query: r.GetQuery("q").String(), Page: int32(page), PageSize: int32(size)}, e1 == nil && e2 == nil
}
func optionalBool(r *ghttp.Request, key string) (*bool, bool) {
	raw := strings.TrimSpace(r.GetQuery(key).String())
	if raw == "" {
		return nil, true
	}
	v, e := strconv.ParseBool(raw)
	if e != nil {
		return nil, false
	}
	return &v, true
}
func optionalID(r *ghttp.Request, key string) (*uuid.UUID, bool) {
	raw := strings.TrimSpace(r.GetQuery(key).String())
	if raw == "" {
		return nil, true
	}
	id, e := uuid.Parse(raw)
	return &id, e == nil
}
func idAt(r *ghttp.Request, key string) (uuid.UUID, bool) {
	id, e := uuid.Parse(r.GetRouter(key).String())
	return id, e == nil
}
func appID(r *ghttp.Request) uuid.UUID {
	id, err := uuid.Parse(r.GetRouter("app_id").String())
	if err != nil {
		return uuid.Nil // Legacy Admin endpoints stay scoped to the tenant's default App.
	}
	return id
}
func token(r *ghttp.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
func principal(r *ghttp.Request) content.Principal {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return content.Principal{RequestID: httpx.RequestID(r), IPAddress: strings.TrimSpace(host), UserAgent: strings.TrimSpace(r.Header.Get("User-Agent"))}
}
func locale(r *ghttp.Request) string {
	value := "zh-CN"
	if strings.Contains(strings.ToLower(r.Header.Get("Accept-Language")), "en") {
		value = "en-US"
	}
	r.Response.Header().Set("Content-Language", value)
	r.Response.Header().Set("Vary", "Authorization, Accept-Language")
	return value
}
func decode(r *ghttp.Request, target any) bool {
	d := json.NewDecoder(io.LimitReader(r.Body, 512*1024))
	d.DisallowUnknownFields()
	return d.Decode(target) == nil && d.Decode(&struct{}{}) == io.EOF
}
func (h *Handler) ok(r *ghttp.Request, status int, data any) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(r)})
}
func (h *Handler) okMeta(r *ghttp.Request, status int, data, meta any) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Data      any    `json:"data"`
		Meta      any    `json:"meta"`
		RequestID string `json:"request_id"`
	}{"OK", "OK", data, meta, httpx.RequestID(r)})
}
func (h *Handler) fail(r *ghttp.Request, e error) bool {
	if e == nil {
		return false
	}
	status, code, key := 500, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(e, iamapp.ErrInvalidAccessToken):
		status, code, key = 401, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized"
	case errors.Is(e, content.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(e, content.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(e, content.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(e, content.ErrConflict):
		status, code, key = 409, "CONTENT.CONFLICT", "errors.common.conflict"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}
