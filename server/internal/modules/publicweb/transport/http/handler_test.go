package http

import (
	"fmt"
	web "github.com/appkernia/appkernia/server/internal/modules/publicweb/application"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPreviewOriginRejectsUntrustedSources(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		development       bool
	}{
		{"production", "https://admin.example.test", "https://admin.example.test", false},
		{"slash", "https://admin.example.test/", "https://admin.example.test", false},
		{"browser origin normalization", "https://ADMIN.example.test:443/", "https://admin.example.test", false},
		{"invalid port", "https://admin.example.test:65536", "", false},
		{"local", "http://localhost:4173", "http://localhost:4173", true},
		{"ipv6", "http://[::1]:4173", "http://[::1]:4173", true},
		{"empty", "", "", false},
		{"production http", "http://localhost:4173", "", false},
		{"remote http", "http://admin.example.test", "", true},
		{"wildcard", "https://*.example.test", "", false},
		{"credentials", "https://user@admin.example.test", "", false},
		{"path", "https://admin.example.test/admin", "", false},
		{"query", "https://admin.example.test?x=1", "", false},
		{"fragment", "https://admin.example.test#x", "", false},
		{"directive injection", "https://admin.example.test;script-src *", "", false},
		{"header injection", "https://admin.example.test\r\nX-Evil: yes", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := previewOrigin(tc.input, tc.development); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPreviewDocumentAndAssetPolicies(t *testing.T) {
	catalog, err := i18n.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(nil, catalog, "https://admin.example.test", false)
	if err != nil {
		t.Fatal(err)
	}
	s := ghttp.GetServer(t.Name())
	s.Group("/", h.Register)
	s.BindHandler("/preview-fixture", func(r *ghttp.Request) {
		h.respond(r, web.View{Kind: "article", Locale: "en-US", Title: "Fixture", MediaOrigin: "https://media.example.test"}, nil)
	})
	s.SetPort(0)
	s.SetDumpRouterMap(false)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Shutdown() })
	for _, tc := range []struct {
		path string
		code int
		html bool
	}{
		{"/preview-fixture", 200, true},
		{"/h5/apps/invalid/download", 404, true},
		{h.previewJS, 200, false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", s.GetListenedPort(), tc.path), nil)
			req.Host = "attacker.example.test"
			req.Header.Set("X-Forwarded-Host", "attacker.example.test")
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tc.code {
				t.Fatalf("status %d", response.StatusCode)
			}
			policy := response.Header.Get("Content-Security-Policy")
			ancestor := "frame-ancestors 'none'"
			if tc.html {
				ancestor = "frame-ancestors https://admin.example.test"
			}
			if !strings.Contains(policy, ancestor) || strings.Contains(policy, "attacker") || !strings.Contains(policy, "script-src 'self'") || (tc.path == "/preview-fixture" && !strings.Contains(policy, "media-src 'self' https://media.example.test")) {
				t.Fatalf("policy: %s", policy)
			}
			if tc.html && (!strings.Contains(string(body), `data-preview-origin="https://admin.example.test"`) || !strings.Contains(string(body), h.previewJS)) {
				t.Fatal("missing trusted preview configuration")
			}
			if !tc.html && response.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" {
				t.Fatal("asset not immutable")
			}
		})
	}
}

func TestEmbeddedTemplatesEscapeAndLocalize(t *testing.T) {
	catalog, err := i18n.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(nil, catalog, "", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"zh-CN", "en-US"} {
		for _, kind := range []string{"article", "document", "download", "error"} {
			out, err := h.Render(web.View{Kind: kind, Locale: locale, Title: `<script>alert(1)</script>`, Summary: `" onload="alert(1)`, AppName: "Fixture App", Canonical: "https://public.example/download?lang=" + locale})
			if err != nil {
				t.Fatalf("%s %s: %v", kind, locale, err)
			}
			s := string(out)
			if strings.Contains(s, "<script>alert") || strings.Contains(s, "public_web.") {
				t.Fatalf("unescaped or untranslated output: %s", s)
			}
			if !strings.Contains(s, `lang="`+locale+`"`) || !strings.Contains(s, h.css) {
				t.Fatal("locale or hashed css missing")
			}
			if (kind == "download" || kind == "article") != strings.Contains(s, `<script type="module"`) {
				t.Fatal("page-specific JS loading policy broken")
			}
		}
	}
	if len(h.assets) != 4 {
		t.Fatal("missing embedded assets")
	}
}

func TestLegacyOriginUsesRelativeDisplayImageAndAbsoluteShareImage(t *testing.T) {
	catalog, err := i18n.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(nil, catalog, "", false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := h.Render(web.View{Kind: "article", Locale: "zh-CN", Title: "Article", CoverURL: "/h5/apps/test/assets/image", ShareImageURL: "https://public.example/h5/apps/test/assets/image"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `<img src="/h5/apps/test/assets/image"`) || !strings.Contains(string(out), `property="og:image" content="https://public.example/h5/apps/test/assets/image"`) {
		t.Fatalf("legacy origin image policy broken: %s", out)
	}
}

func TestVideoAndPageControlsUseAccessibleIconsAndConfiguredPromotion(t *testing.T) {
	catalog, err := i18n.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(nil, catalog, "", false)
	if err != nil {
		t.Fatal(err)
	}
	view := web.View{Kind: "article", ContentType: "video", Locale: "en-US", Title: "Video", AppName: "Fixture", OtherLanguageURL: "https://public.example/article?lang=zh-CN", DownloadURL: "https://public.example/download?lang=en-US", PromotionEnabled: true, PromotionTitle: "Get Fixture", PromotionDescription: "Continue in the app", PromotionButtonLabel: "Install now"}
	out, err := h.Render(view)
	if err != nil {
		t.Fatal(err)
	}
	html := string(out)
	for _, expected := range []string{`class="icon-button language-switch"`, `aria-label="Switch to 简体中文"`, `data-video-mode="horizontal"`, `data-video-mode="vertical"`, `<svg aria-hidden="true"`, `Get Fixture`, `Continue in the app`, `Install now`} {
		if !strings.Contains(html, expected) {
			t.Fatalf("missing %q in %s", expected, html)
		}
	}
	if strings.Contains(html, `<footer class="site-footer"><span>Fixture</span><a`) {
		t.Fatal("language switch remained in the footer")
	}
	view.PromotionEnabled = false
	out, err = h.Render(view)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), `class="article-app"`) || strings.Contains(string(out), `class="header-download"`) {
		t.Fatal("disabled promotion was rendered")
	}
}
