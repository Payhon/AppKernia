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
	raw := `<p onclick="steal()">Hello <strong>team</strong><script>alert(1)</script><a href="javascript:steal()" style="color:red">bad</a><a href="https://example.com" target="_blank">safe</a><a href="{{reset_url}}">reset</a></p>`
	got := SanitizeHTML(raw)
	for _, forbidden := range []string{"onclick", "script", "alert", "javascript", "style", "target"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("sanitized HTML retained %q: %s", forbidden, got)
		}
	}
	for _, wanted := range []string{"<p>", "<strong>team</strong>", `<a>bad</a>`, `<a href="https://example.com">safe</a>`, `<a href="{{reset_url}}">reset</a>`} {
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

func TestRenderTemplateEscapesHTMLAndRejectsMissingRequiredVariables(t *testing.T) {
	subject := "Hello {{name}}"
	template := notify.Template{
		SubjectTemplate: &subject,
		BodyTemplate:    `<p>Code: <strong>{{code}}</strong>; {{name}}</p>`,
		BodyFormat:      "html",
		VariablesSchema: json.RawMessage(`{"type":"object","properties":{"code":{"type":"string"},"name":{"type":"string"}},"required":["code","name"]}`),
	}
	gotSubject, gotBody, err := renderTemplate(template, map[string]string{"code": `<123>`, "name": `A&B`})
	if err != nil {
		t.Fatalf("render valid HTML template: %v", err)
	}
	if gotSubject != "Hello A&B" || !strings.Contains(gotBody, `&lt;123&gt;`) || !strings.Contains(gotBody, `A&amp;B`) {
		t.Fatalf("unexpected rendered template: subject=%q body=%q", gotSubject, gotBody)
	}
	if _, _, err = renderTemplate(template, map[string]string{"code": "123"}); err == nil {
		t.Fatal("missing required variable must fail")
	}
	if _, _, err = renderTemplate(template, map[string]string{"code": "123", "name": "A", "undeclared": "x"}); err == nil {
		t.Fatal("undeclared template variable must fail")
	}
}

func TestNormalizeBindingUsesRegisteredProvidersAndUniqueParameterOrder(t *testing.T) {
	input := notify.SMSTemplateBindingInput{ExternalTemplateID: " SMS_123 ", ParameterOrder: json.RawMessage(`["code","expires_minutes"]`)}
	provider, normalized, err := normalizeBinding(" tencent ", input)
	if err != nil || provider != "tencent" || normalized.ExternalTemplateID != "SMS_123" || normalized.Status != "active" {
		t.Fatalf("normalize valid binding: provider=%q input=%#v err=%v", provider, normalized, err)
	}
	input.ParameterOrder = json.RawMessage(`["code","code"]`)
	if _, _, err = normalizeBinding("tencent", input); err == nil {
		t.Fatal("duplicate Tencent parameter name must fail")
	}
	if _, _, err = normalizeBinding("unregistered", notify.SMSTemplateBindingInput{ExternalTemplateID: "x", ParameterOrder: json.RawMessage(`[]`)}); err == nil {
		t.Fatal("unregistered provider must fail")
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
