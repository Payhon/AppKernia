package adminui

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gogf/gf/v2/net/ghttp"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		value string
		want  string
		ok    bool
	}{
		{"", "/admin", true},
		{"/admin", "/admin", true},
		{"/ops/admin", "/ops/admin", true},
		{"/", "", false},
		{"admin", "", false},
		{"/admin/../api", "", false},
		{"/admin%2Fapi", "", false},
		{"/admin-api", "", false},
		{"/assets/admin", "", false},
	} {
		test := test
		t.Run(test.value, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizePath(test.value)
			if test.ok && (err != nil || got != test.want) {
				t.Fatalf("NormalizePath(%q) = %q, %v; want %q", test.value, got, err, test.want)
			}
			if !test.ok && err == nil {
				t.Fatalf("NormalizePath(%q) unexpectedly succeeded with %q", test.value, got)
			}
		})
	}
}

func TestNewHandlerInjectsBasePath(t *testing.T) {
	t.Parallel()
	handler, err := newHandler("/ops/admin", fstest.MapFS{
		"index.html": {Data: []byte("<head>" + basePathMarker + "</head>")},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(handler.index); got != `<head><meta name="ak-admin-base-path" content="/ops/admin" /></head>` {
		t.Fatalf("unexpected index: %s", got)
	}
}

func TestNewHandlerRejectsMissingMarker(t *testing.T) {
	t.Parallel()
	_, err := newHandler("/admin", fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}, "")
	if err == nil {
		t.Fatal("expected missing marker error")
	}
}

func TestHandlerRoutesAdminAndStaticFiles(t *testing.T) {
	handler, err := newHandler("/admin", fstest.MapFS{
		"index.html":             {Data: []byte("<head>" + basePathMarker + "</head>")},
		"assets/app-deadbeef.js": {Data: []byte("export{}")},
		"openapi/index.html":     {Data: []byte("openapi")},
	}, "https://public.example.test")
	if err != nil {
		t.Fatal(err)
	}
	server := ghttp.GetServer(t.Name())
	handler.Register(server)
	server.SetPort(0)
	server.SetDumpRouterMap(false)
	if err = server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Shutdown() })
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	for _, test := range []struct {
		method, path, accept string
		status               int
		body, cache          string
	}{
		{http.MethodGet, "/admin", "text/html", http.StatusPermanentRedirect, "", ""},
		{http.MethodGet, "/admin/", "text/html", http.StatusOK, `content="/admin"`, "no-cache, must-revalidate"},
		{http.MethodGet, "/admin/dashboard", "text/html", http.StatusOK, `content="/admin"`, "no-cache, must-revalidate"},
		{http.MethodGet, "/assets/app-deadbeef.js", "*/*", http.StatusOK, "export{}", "public, max-age=31536000, immutable"},
		{http.MethodGet, "/assets/missing.js", "text/html", http.StatusNotFound, "", ""},
		{http.MethodGet, "/openapi/", "text/html", http.StatusOK, "openapi", "no-cache, must-revalidate"},
		{http.MethodGet, "/openapi/openapi.yaml", "application/yaml", http.StatusOK, "openapi: 3.1.0", "no-cache, must-revalidate"},
		{http.MethodPost, "/admin/", "text/html", http.StatusMethodNotAllowed, "", ""},
	} {
		request, requestErr := http.NewRequest(test.method, fmt.Sprintf("http://127.0.0.1:%d%s", server.GetListenedPort(), test.path), nil)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		request.Header.Set("Accept", test.accept)
		response, requestErr := client.Do(request)
		if requestErr != nil {
			t.Fatalf("%s %s: %v", test.method, test.path, requestErr)
		}
		content, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if response.StatusCode != test.status || (test.body != "" && !strings.Contains(string(content), test.body)) {
			t.Fatalf("%s %s: status=%d location=%q body=%q", test.method, test.path, response.StatusCode, response.Header.Get("Location"), content)
		}
		if test.cache != "" && response.Header.Get("Cache-Control") != test.cache {
			t.Fatalf("%s: Cache-Control=%q", test.path, response.Header.Get("Cache-Control"))
		}
		if test.status == http.StatusOK && !strings.Contains(response.Header.Get("Content-Security-Policy"), "frame-src 'self' https://public.example.test") {
			t.Fatalf("%s: public preview origin missing from CSP", test.path)
		}
	}
}
