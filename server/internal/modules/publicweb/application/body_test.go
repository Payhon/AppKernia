package application

import (
	"encoding/json"
	"github.com/google/uuid"
	"strings"
	"testing"
)

func TestRenderBodySafetyAndTypography(t *testing.T) {
	app, file := uuid.New(), uuid.New()
	markdown := "# Title\n\nParagraph **bold**.\n\n> Quote\n\n| One | Two |\n| --- | --- |\n| a | b |\n\n```go\nfmt.Println(\"hello\")\n```\n\n![safe](/api/v1/public/content/assets/" + file.String() + ")\n\n![remote](https://tracking.example/a.png)\n\n<script>alert(1)</script>\n\n<div hx-get=\"/admin-api/v1/users\" data-hx-post=\"/bad\" onclick=\"alert(1)\">unsafe</div>\n\n[bad](javascript:alert(1))"
	raw, _ := json.Marshal(markdown)
	out, err := RenderBody(raw, "markdown", app)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"<strong>bold</strong>", "<blockquote>", "<table>", "<pre><code", "language-go", "/h5/apps/" + app.String() + "/assets/" + file.String()} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %s", want, s)
		}
	}
	for _, bad := range []string{"<script", "hx-get", "data-hx-post", "onclick", "javascript:", "https://tracking.example"} {
		if strings.Contains(s, bad) {
			t.Errorf("unsafe %q in %s", bad, s)
		}
	}
}
func TestRenderBlocksAndRejectInvalidContent(t *testing.T) {
	raw := json.RawMessage(`{"type":"doc","content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Hello <script>"}]},{"type":"paragraph","content":[{"type":"text","text":"Link","marks":[{"type":"link","attrs":{"href":"javascript:alert(1)"}}]}]}]}`)
	out, err := RenderBody(raw, "blocks", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "<h2>Hello &lt;script&gt;</h2>") || strings.Contains(string(out), "javascript:") {
		t.Fatalf("unsafe blocks: %s", out)
	}
	for _, c := range []struct {
		raw    json.RawMessage
		format string
	}{{raw, "html"}, {raw, "markdown"}, {json.RawMessage(`"broken`), "markdown"}, {json.RawMessage(strings.Repeat("x", (1<<20)+1)), "markdown"}} {
		if _, e := RenderBody(c.raw, c.format, uuid.New()); e == nil {
			t.Fatal("invalid content accepted")
		}
	}
}
func TestBodyImagesAreScoped(t *testing.T) {
	app, file := uuid.New(), uuid.New()
	want := "/h5/apps/" + app.String() + "/assets/" + file.String()
	for _, raw := range []string{"/api/v1/public/content/assets/" + file.String(), "/s/assets/" + app.String() + "/" + file.String(), want} {
		if bodyImage(raw, app) != want {
			t.Errorf("rejected %s", raw)
		}
	}
	for _, raw := range []string{"https://example.test/a.png", "//evil.test/x", "data:image/png;base64,AAAA", want + "?app_id=" + uuid.NewString(), "/h5/apps/" + uuid.NewString() + "/assets/" + file.String(), "/api/v1/public/content/assets/../private"} {
		if bodyImage(raw, app) != "" {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestExternalVideoPolicy(t *testing.T) {
	for _, tc := range []struct{ raw, url, origin string }{
		{"https://media.example.test:8443/video.mp4?quality=hd", "https://media.example.test:8443/video.mp4?quality=hd", "https://media.example.test:8443"},
		{"http://media.example.test/video.mp4", "", ""},
		{"https://user@media.example.test/video.mp4", "", ""},
		{"javascript:alert(1)", "", ""},
	} {
		gotURL, gotOrigin := externalVideo(tc.raw)
		if gotURL != tc.url || gotOrigin != tc.origin {
			t.Fatalf("externalVideo(%q)=(%q,%q), want (%q,%q)", tc.raw, gotURL, gotOrigin, tc.url, tc.origin)
		}
	}
}
