package notification

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNormalizeRejectsUnsafeRoute(t *testing.T) {
	scope := Scope{TenantID: uuid.New(), AppID: uuid.New(), ActorID: uuid.New(), ActorKind: "api_client", RequestID: "request-1"}
	_, err := normalize(scope, SubmitCommand{
		IdempotencyKey: "request-key", Source: "billing.security", BusinessEventID: "invoice:1", Category: "service_security",
		Audience: Audience{Type: "users", UserIDs: []uuid.UUID{uuid.New()}},
		Content:  Content{Type: "inline", Inline: &LocalizedContent{Title: map[string]string{"zh-CN": "标题", "en-US": "Title"}, Body: map[string]string{"zh-CN": "正文", "en-US": "Body"}}},
		RouteKey: "https://unsafe.example", TTLSeconds: 300,
	}, time.Now().UTC())
	if err == nil {
		t.Fatal("expected unsafe route to be rejected")
	}
}

func TestRenderRequiresAllVariables(t *testing.T) {
	if _, err := render("Hello {{name}}", map[string]string{}); err == nil {
		t.Fatal("expected missing variable to fail")
	}
	got, err := render("Hello {{name}}", map[string]string{"name": "Kernia"})
	if err != nil || got != "Hello Kernia" {
		t.Fatalf("unexpected render: %q %v", got, err)
	}
}
