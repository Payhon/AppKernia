package adminui

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/appkernia/appkernia/server/internal/platform/runtimeassets"
	"github.com/gogf/gf/v2/net/ghttp"
)

const basePathMarker = `<meta name="ak-admin-base-path" content="/" />`

var reservedPaths = []string{
	"/admin-api", "/api", "/assets", "/brand", "/h5", "/internal", "/openapi", "/s",
}

type Handler struct {
	adminPath string
	files     fs.FS
	root      *os.Root
	index     []byte
	frameSrc  string
	openAPI   []byte
}

func New(adminPath, staticDir, publicWebBaseURL string) (*Handler, error) {
	normalized, err := NormalizePath(adminPath)
	if err != nil {
		return nil, err
	}
	frameSrc, err := frameSource(publicWebBaseURL)
	if err != nil {
		return nil, err
	}
	var source fs.FS
	var root *os.Root
	if staticDir != "" {
		root, err = os.OpenRoot(staticDir)
		if err != nil {
			return nil, fmt.Errorf("open Admin static directory: %w", err)
		}
		source = root.FS()
	} else {
		var available bool
		source, available, err = embeddedFiles()
		if err != nil {
			return nil, fmt.Errorf("open embedded Admin files: %w", err)
		}
		if !available {
			return nil, nil
		}
	}
	handler, err := newHandler(normalized, source, frameSrc)
	if err != nil {
		if root != nil {
			_ = root.Close()
		}
		return nil, err
	}
	handler.root = root
	return handler, nil
}

func newHandler(adminPath string, source fs.FS, frameSrc string) (*Handler, error) {
	index, err := fs.ReadFile(source, "index.html")
	if err != nil {
		return nil, fmt.Errorf("read Admin index.html: %w", err)
	}
	if !bytes.Contains(index, []byte(basePathMarker)) {
		return nil, errors.New("Admin index.html is missing the ak-admin-base-path marker")
	}
	openAPI, err := runtimeassets.OpenAPI()
	if err != nil {
		return nil, err
	}
	index = bytes.Replace(index, []byte(basePathMarker), []byte(`<meta name="ak-admin-base-path" content="`+adminPath+`" />`), 1)
	return &Handler{adminPath: adminPath, files: source, index: index, frameSrc: frameSrc, openAPI: openAPI}, nil
}

func frameSource(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return "", errors.New("public web base URL is invalid for the Admin content security policy")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func NormalizePath(value string) (string, error) {
	if value == "" {
		value = "/admin"
	}
	if value == "/" || !strings.HasPrefix(value, "/") || strings.ContainsAny(value, "\\%?#\x00\r\n\t") {
		return "", fmt.Errorf("invalid Admin path %q", value)
	}
	normalized := path.Clean(value)
	if normalized != value && value != normalized+"/" {
		return "", fmt.Errorf("Admin path must be normalized: %q", value)
	}
	for _, reserved := range reservedPaths {
		if normalized == reserved || strings.HasPrefix(normalized, reserved+"/") {
			return "", fmt.Errorf("Admin path %q conflicts with reserved route %q", value, reserved)
		}
	}
	return normalized, nil
}

func (h *Handler) Close() error {
	if h == nil || h.root == nil {
		return nil
	}
	return h.root.Close()
}

func (h *Handler) Register(server *ghttp.Server) {
	server.BindHandler(h.adminPath, h.Serve)
	server.BindHandler(h.adminPath+"/*resource", h.Serve)
	server.BindHandler("/assets/*resource", h.Serve)
	server.BindHandler("/brand/*resource", h.Serve)
	server.BindHandler("/openapi", h.Serve)
	server.BindHandler("/openapi/*resource", h.Serve)
}

func (h *Handler) Serve(request *ghttp.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		request.Response.Header().Set("Allow", "GET, HEAD")
		request.Response.WriteStatus(http.StatusMethodNotAllowed)
		return
	}
	requestPath := request.URL.Path
	if original, err := url.ParseRequestURI(request.RequestURI); err == nil {
		requestPath = original.Path
	}
	if requestPath == h.adminPath || requestPath == "/openapi" {
		request.Response.Header().Set("Location", requestPath+"/")
		request.Response.WriteStatus(http.StatusPermanentRedirect)
		return
	}
	if requestPath == "/openapi/" {
		if !h.writeFile(request, "openapi/index.html") {
			request.Response.WriteStatus(http.StatusNotFound)
		}
		return
	}
	if requestPath == "/openapi/openapi.yaml" {
		h.writeOpenAPI(request)
		return
	}
	if strings.HasPrefix(requestPath, h.adminPath+"/") {
		relative := strings.TrimPrefix(requestPath, h.adminPath+"/")
		if relative == "" {
			h.writeIndex(request)
			return
		}
		if h.writeFile(request, relative) {
			return
		}
		if strings.Contains(request.Header.Get("Accept"), "text/html") && path.Ext(relative) == "" {
			h.writeIndex(request)
			return
		}
		request.Response.WriteStatus(http.StatusNotFound)
		return
	}
	relative := strings.TrimPrefix(requestPath, "/")
	if !h.writeFile(request, relative) {
		request.Response.WriteStatus(http.StatusNotFound)
	}
}

func (h *Handler) writeOpenAPI(request *ghttp.Request) {
	h.securityHeaders(request)
	request.Response.Header().Set("Cache-Control", "no-cache, must-revalidate")
	request.Response.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	request.Response.WriteHeader(http.StatusOK)
	if request.Method == http.MethodGet {
		request.Response.Write(h.openAPI)
	}
}

func (h *Handler) writeIndex(request *ghttp.Request) {
	h.securityHeaders(request)
	request.Response.Header().Set("Cache-Control", "no-cache, must-revalidate")
	request.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	request.Response.WriteHeader(http.StatusOK)
	if request.Method == http.MethodGet {
		request.Response.Write(h.index)
	}
}

func (h *Handler) writeFile(request *ghttp.Request, name string) bool {
	if !validFileName(name) {
		return false
	}
	body, err := fs.ReadFile(h.files, name)
	if err != nil {
		return false
	}
	h.securityHeaders(request)
	if strings.HasPrefix(name, "assets/") {
		request.Response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		request.Response.Header().Set("Cache-Control", "no-cache, must-revalidate")
	}
	contentType := mime.TypeByExtension(path.Ext(name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	request.Response.Header().Set("Content-Type", contentType)
	request.Response.WriteHeader(http.StatusOK)
	if request.Method == http.MethodGet {
		request.Response.Write(body)
	}
	return true
}

func validFileName(name string) bool {
	if !fs.ValidPath(name) || name == "." {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") {
			return false
		}
	}
	return true
}

func (h *Handler) securityHeaders(request *ghttp.Request) {
	header := request.Response.Header()
	frameSources := "'self'"
	if h.frameSrc != "" {
		frameSources += " " + h.frameSrc
	}
	header.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob: https:; connect-src 'self'; frame-src "+frameSources+"; font-src 'self' data:; worker-src 'self' blob:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
	header.Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
}
