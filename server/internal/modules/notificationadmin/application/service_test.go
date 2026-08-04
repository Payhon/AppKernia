package application

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/google/uuid"
)

func TestSanitizeHTMLAllowlist(t *testing.T) {
	raw := `<p onclick="steal()">Hello <strong>team</strong><script>alert(1)</script><a href="javascript:steal()" style="color:red">bad</a><a href="https://example.com" target="_blank">safe</a></p>`
	got := SanitizeHTML(raw)
	for _, forbidden := range []string{"onclick", "script", "alert", "javascript", "style", "target"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("sanitized HTML retained %q: %s", forbidden, got)
		}
	}
	for _, wanted := range []string{"<p>", "<strong>team</strong>", `<a>bad</a>`, `<a href="https://example.com">safe</a>`} {
		if !strings.Contains(got, wanted) {
			t.Fatalf("sanitized HTML lost %q: %s", wanted, got)
		}
	}
}

func TestNormalizeMessageAudienceAndSchedule(t *testing.T) {
	now := time.Date(2026, 8, 3, 8, 0, 0, 0, time.UTC)
	first := uuid.New()
	in, err := normalizeMessage(notify.MessageInput{Title: " Update ", Body: "<p>Ready</p><iframe src='x'>drop</iframe>", BodyFormat: "html", AudienceScope: "selected", AudienceUserIDs: []uuid.UUID{first, first}, ScheduledAt: ptrTime(now.Add(time.Hour))}, false, now)
	if err != nil {
		t.Fatalf("normalize valid message: %v", err)
	}
	if in.Title != "Update" || in.MessageType != "system" || len(in.AudienceUserIDs) != 1 || strings.Contains(in.Body, "iframe") {
		t.Fatalf("unexpected normalized message: %#v", in)
	}
	if _, err = normalizeMessage(notify.MessageInput{Title: "No audience", Body: "body", AudienceScope: "selected"}, false, now); err == nil {
		t.Fatal("selected audience without users must fail")
	}
}

func TestNormalizeTemplateRequiresDeclaredVariablesAndCanonicalLocale(t *testing.T) {
	locale := "en-US"
	schema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
	in, err := normalizeTemplate(notify.TemplateInput{Code: "account.welcome", Name: "Welcome", Channel: "email", Locale: &locale, BodyTemplate: "Hello {{ name }}", VariablesSchema: schema})
	if err != nil || in.Status != "active" {
		t.Fatalf("normalize valid template: %#v %v", in, err)
	}
	badLocale := "en"
	if _, err = normalizeTemplate(notify.TemplateInput{Code: "account.welcome", Name: "Welcome", Channel: "email", Locale: &badLocale, BodyTemplate: "Hello", VariablesSchema: schema}); err == nil {
		t.Fatal("non-canonical locale must fail")
	}
	if _, err = normalizeTemplate(notify.TemplateInput{Code: "account.welcome", Name: "Welcome", Channel: "email", Locale: &locale, BodyTemplate: "Hello {{ missing }}", VariablesSchema: schema}); err == nil {
		t.Fatal("undeclared placeholder must fail")
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
