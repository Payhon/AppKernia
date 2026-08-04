package repository

import (
	"context"
	"testing"
	"time"
)

func TestLocalMockAdapterRequiresSafeHostAndSignatureEnvelope(t *testing.T) {
	adapter := NewLocalMockAdapter()
	headers := map[string]string{"X-AK-Webhook-Signature": "v1=abc", "X-AK-Webhook-Timestamp": "123", "X-AK-Webhook-Event-ID": "event"}
	result, err := adapter.Deliver(context.Background(), "https://hooks.example.test/events", headers, []byte(`{"ok":true}`), time.Second)
	if err != nil || result.StatusCode != 204 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if _, err = adapter.Deliver(context.Background(), "https://example.com/events", headers, []byte(`{"ok":true}`), time.Second); err == nil {
		t.Fatal("expected non-mock host rejection")
	}
	if _, err = adapter.Deliver(context.Background(), "https://hooks.example.test/events", map[string]string{}, []byte(`{}`), time.Second); err == nil {
		t.Fatal("expected missing envelope rejection")
	}
}
