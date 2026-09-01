package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	apps "github.com/appkernia/appkernia/server/internal/modules/appmanagement/application"
	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	releaseapp "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/application"
	releases "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	web "github.com/appkernia/appkernia/server/internal/modules/publicweb/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/appkernia/appkernia/server/internal/shared/publicurl"
	"github.com/appkernia/appkernia/server/resource"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"
)

type asset struct {
	body  []byte
	media string
}
type Handler struct {
	service                *web.Service
	catalog                *i18n.Catalog
	templates              map[string]*template.Template
	assets                 map[string]asset
	css, js, mediaJS       string
	previewJS, adminOrigin string
}

func NewHandler(service *web.Service, catalog *i18n.Catalog, adminOrigin string, development bool) (*Handler, error) {
	h := &Handler{service: service, catalog: catalog, templates: map[string]*template.Template{}, assets: map[string]asset{}, adminOrigin: previewOrigin(adminOrigin, development)}
	if adminOrigin != "" && h.adminOrigin == "" {
		slog.Warn("public web preview disabled: invalid AK_ADMIN_ORIGIN")
	}
	for _, kind := range []string{"article", "document", "download", "error"} {
		file := kind
		if kind == "document" {
			file = "article"
		}
		t, e := template.New("base").Option("missingkey=error").ParseFS(resource.Files, "tpl/layouts/base.html", "tpl/components/*.html", "tpl/pages/"+file+".html")
		if e != nil {
			return nil, e
		}
		h.templates[kind] = t
	}
	for _, file := range []string{"css/public-web.css", "js/download.js", "js/media.js", "js/preview.js"} {
		body, e := resource.Files.ReadFile("static/" + file)
		if e != nil {
			return nil, e
		}
		sum := sha256.Sum256(body)
		ext := "css"
		media := "text/css; charset=utf-8"
		if strings.HasSuffix(file, ".js") {
			ext = "js"
			media = "text/javascript; charset=utf-8"
		}
		name := hex.EncodeToString(sum[:12]) + "." + ext
		h.assets[name] = asset{body, media}
		if ext == "css" {
			h.css = "/h5/static/" + name
		} else if file == "js/preview.js" {
			h.previewJS = "/h5/static/" + name
		} else if file == "js/media.js" {
			h.mediaJS = "/h5/static/" + name
		} else {
			h.js = "/h5/static/" + name
		}
	}
	return h, nil
}

// This value becomes a CSP source, not a pattern or a request-derived origin.
func previewOrigin(raw string, development bool) string {
	if raw == "" || strings.ContainsAny(raw, " \t\r\n;'\"*\\") || publicurl.Validate(raw, development) != nil {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port := u.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return ""
		}
		if !(u.Scheme == "https" && port == "443") && !(u.Scheme == "http" && port == "80") {
			host += ":" + port
		}
	}
	return u.Scheme + "://" + host
}

func (h *Handler) documentHeaders(r *ghttp.Request, mediaOrigin string) {
	headers(r)
	if mediaOrigin != "" {
		policy := r.Response.Header().Get("Content-Security-Policy")
		r.Response.Header().Set("Content-Security-Policy", strings.Replace(policy, "media-src 'self'", "media-src 'self' "+mediaOrigin, 1))
	}
	if h.adminOrigin != "" {
		policy := r.Response.Header().Get("Content-Security-Policy")
		r.Response.Header().Set("Content-Security-Policy", strings.Replace(policy, "frame-ancestors 'none'", "frame-ancestors "+h.adminOrigin, 1))
	}
}
func (h *Handler) Register(group *ghttp.RouterGroup) {
	group.GET("/h5/static/{name}", h.Static)
	group.GET("/h5/apps/{app_id}/articles/{slug}", h.Article)
	group.GET("/h5/apps/{app_id}/pages/{slug}", h.Page)
	group.GET("/h5/apps/{app_id}/download", h.Download)
	group.GET("/h5/apps/{app_id}/assets/{file_id}", h.Asset)
	group.GET("/h5/apps/{app_id}/apk", h.APK)
	group.GET("/h5/apps/{app_id}/packages/{release_id}/{file_id}", h.Package)
	group.GET("/s/{slug}", h.LegacyArticle)
	group.GET("/s/assets/{app_id}/{file_id}", h.Asset)
}
func language(r *ghttp.Request) string {
	raw := r.GetQuery("lang").String()
	if raw != "" {
		return string(i18n.Normalize(raw))
	}
	return string(httpx.Locale(r))
}
func appID(r *ghttp.Request) (uuid.UUID, error) { return uuid.Parse(r.GetRouter("app_id").String()) }
func headers(r *ghttp.Request) {
	r.Response.Header().Set("X-Content-Type-Options", "nosniff")
	r.Response.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	r.Response.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; media-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'self'")
	r.Response.Header().Set("Cache-Control", "no-cache")
	r.Response.Header().Set("Vary", "Accept-Language")
}
func (h *Handler) Render(v web.View) ([]byte, error) {
	v.CSS, v.JS = h.css, h.js
	v.MediaJS = h.mediaJS
	if h.adminOrigin != "" {
		v.PreviewJS, v.PreviewOrigin = h.previewJS, h.adminOrigin
	}
	v.Labels = map[string]string{}
	for _, key := range []string{"skip", "other_language", "page_actions", "switch_language", "download", "version", "article_footer", "official", "get_app", "screenshots", "screenshot", "choose", "download_title", "unknown", "recommended", "wechat", "all", "ios", "android", "harmony", "apk", "download_apk", "visit_store", "other_methods", "no_download", "qr_alt", "scan", "scan_hint", "copy", "copied", "copy_failed", "documents", "unavailable", "gallery", "previous_image", "next_image", "video_layout", "horizontal_video", "vertical_video", "video_unavailable"} {
		v.Labels[key] = h.catalog.Translate(i18n.Normalize(v.Locale), "public_web."+key, nil)
	}
	var out bytes.Buffer
	t, ok := h.templates[v.Kind]
	if !ok {
		return nil, fmt.Errorf("unknown page kind")
	}
	e := t.ExecuteTemplate(&out, "base", v)
	return out.Bytes(), e
}
func (h *Handler) respond(r *ghttp.Request, v web.View, err error) {
	if err != nil {
		h.fail(r, err)
		return
	}
	body, e := h.Render(v)
	if e != nil {
		h.fail(r, e)
		return
	}
	h.documentHeaders(r, v.MediaOrigin)
	r.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	r.Response.Header().Set("Content-Language", v.Locale)
	r.Response.WriteHeader(200)
	r.Response.Write(body)
}
func (h *Handler) fail(r *ghttp.Request, err error) {
	code := http.StatusServiceUnavailable
	key := "error_service"
	if errors.Is(err, web.ErrNotFound) || errors.Is(err, apps.ErrAppNotFound) || errors.Is(err, apps.ErrAppDisabled) || errors.Is(err, content.ErrNotFound) || errors.Is(err, content.ErrForbidden) || errors.Is(err, content.ErrInvalid) || errors.Is(err, releases.ErrReleaseNotFound) || errors.Is(err, releases.ErrReleaseFileInvalid) || errors.Is(err, releaseapp.ErrInvalidRelease) {
		code = 404
		key = "error_not_found"
	}
	if code == 503 {
		slog.ErrorContext(r.Context(), "public web request failed", "request_id", httpx.RequestID(r), "error", err)
	}
	loc := language(r)
	v := web.View{Kind: "error", Locale: loc, Title: h.catalog.Translate(i18n.Normalize(loc), "public_web."+key, nil), Summary: h.catalog.Translate(i18n.Normalize(loc), "public_web.error_hint", nil)}
	body, e := h.Render(v)
	h.documentHeaders(r, "")
	r.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	r.Response.Header().Set("Content-Language", loc)
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.Header().Set("X-Robots-Tag", "noindex")
	r.Response.WriteHeader(code)
	if e == nil {
		r.Response.Write(body)
	}
}
func (h *Handler) Article(r *ghttp.Request) {
	id, e := appID(r)
	if e != nil {
		h.fail(r, web.ErrNotFound)
		return
	}
	v, e := h.service.Article(r.Context(), id, r.GetRouter("slug").String(), language(r))
	h.respond(r, v, e)
}
func (h *Handler) LegacyArticle(r *ghttp.Request) {
	id, e := uuid.Parse(r.GetQuery("app_id").String())
	if e != nil {
		h.fail(r, web.ErrNotFound)
		return
	}
	v, e := h.service.Article(r.Context(), id, r.GetRouter("slug").String(), language(r))
	h.respond(r, v, e)
}
func (h *Handler) Page(r *ghttp.Request) {
	id, e := appID(r)
	if e != nil {
		h.fail(r, web.ErrNotFound)
		return
	}
	v, e := h.service.Page(r.Context(), id, r.GetRouter("slug").String(), language(r))
	h.respond(r, v, e)
}
func (h *Handler) Download(r *ghttp.Request) {
	id, e := appID(r)
	if e != nil {
		h.fail(r, web.ErrNotFound)
		return
	}
	v, e := h.service.Download(r.Context(), id, language(r))
	if e != nil {
		h.fail(r, e)
		return
	}
	if r.GetQuery("format").String() == "qr" {
		body, err := qrcode.Encode(v.Canonical, qrcode.Medium, 256)
		if err != nil {
			h.fail(r, err)
			return
		}
		headers(r)
		r.Response.Header().Set("Content-Type", "image/png")
		r.Response.Write(body)
		return
	}
	h.respond(r, v, nil)
}
func (h *Handler) Static(r *ghttp.Request) {
	x, ok := h.assets[r.GetRouter("name").String()]
	if !ok {
		h.fail(r, web.ErrNotFound)
		return
	}
	headers(r)
	r.Response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	r.Response.Header().Set("Content-Type", x.media)
	r.Response.Write(x.body)
}
func (h *Handler) Asset(r *ghttp.Request) {
	id, e := appID(r)
	file, fe := uuid.Parse(r.GetRouter("file_id").String())
	if e != nil || fe != nil {
		h.fail(r, web.ErrNotFound)
		return
	}
	media, size, reader, e := h.service.Asset(r.Context(), id, file)
	if e != nil {
		h.fail(r, e)
		return
	}
	defer reader.Close()
	headers(r)
	r.Response.Header().Set("Content-Type", media)
	r.Response.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	if _, e = io.Copy(r.Response.BufferWriter, reader); e != nil {
		slog.WarnContext(r.Context(), "public asset stream interrupted", "request_id", httpx.RequestID(r))
	}
}
func (h *Handler) APK(r *ghttp.Request) {
	id, e := appID(r)
	if e != nil {
		h.fail(r, web.ErrNotFound)
		return
	}
	target, e := h.service.PackageURL(r.Context(), id)
	if e != nil {
		h.fail(r, e)
		return
	}
	headers(r)
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.Header().Set("Location", target)
	r.Response.WriteHeader(http.StatusFound)
}
func (h *Handler) Package(r *ghttp.Request) {
	id, e := appID(r)
	release, re := uuid.Parse(r.GetRouter("release_id").String())
	file, fe := uuid.Parse(r.GetRouter("file_id").String())
	expires, xe := strconv.ParseInt(r.GetQuery("expires").String(), 10, 64)
	if e != nil || re != nil || fe != nil || xe != nil {
		h.fail(r, web.ErrNotFound)
		return
	}
	meta, reader, e := h.service.Package(r.Context(), id, release, file, expires, r.GetQuery("signature").String())
	if e != nil {
		h.fail(r, e)
		return
	}
	defer reader.Close()
	headers(r)
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.Header().Set("Content-Type", "application/vnd.android.package-archive")
	r.Response.Header().Set("Content-Disposition", `attachment; filename="app.apk"`)
	r.Response.Header().Set("Content-Length", strconv.FormatInt(meta.SizeBytes, 10))
	if _, e = io.Copy(r.Response.BufferWriter, reader); e != nil {
		slog.WarnContext(r.Context(), "package stream interrupted", "request_id", httpx.RequestID(r))
	}
}
