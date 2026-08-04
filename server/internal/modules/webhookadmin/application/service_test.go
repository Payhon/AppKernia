package application

import (
	"testing"

	webhooks "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/domain"
)

func TestValidateEndpointURLRejectsSSRFPrimitives(t *testing.T) {
	tests := []struct {
		raw   string
		valid bool
	}{
		{"https://hooks.example.test/events", true},
		{"https://8.8.8.8/events", true},
		{"http://hooks.example.test/events", false},
		{"https://localhost/events", false},
		{"https://127.0.0.1/events", false},
		{"https://10.0.0.2/events", false},
		{"https://169.254.169.254/latest/meta-data", false},
		{"https://[::1]/events", false},
		{"https://user:password@hooks.example.test/events", false},
		{"https://internal/events", false},
		{"https://hooks.example.test/events#fragment", false},
	}
	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			if got := validateEndpointURL(test.raw) == nil; got != test.valid {
				t.Fatalf("valid=%v, want %v", got, test.valid)
			}
		})
	}
}

func TestNormalizeInputDeduplicatesStableEventCodes(t *testing.T) {
	in, err := normalizeInput(webhookInput("https://hooks.example.test/events", []string{"order.created", " order.created ", "USER.UPDATED"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(in.EventTypes) != 2 || in.EventTypes[0] != "order.created" || in.EventTypes[1] != "user.updated" {
		t.Fatalf("events=%v", in.EventTypes)
	}
}

func webhookInput(raw string, events []string) webhooks.Input {
	return webhooks.Input{Name: "Receiver", EndpointURL: raw, EventTypes: events, MaxAttempts: 8, TimeoutSeconds: 10, Status: "active"}
}
