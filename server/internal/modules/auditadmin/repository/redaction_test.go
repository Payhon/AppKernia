package repository

import "testing"

func TestDecodeRedactedRemovesNestedSensitiveFields(t *testing.T) {
	value := decodeRedacted([]byte(`{"email":"masked@example.test","password":"plain","nested":{"access_token":"jwt","safe":"visible"},"headers":{"Authorization":"Bearer private"},"items":[{"api-key":"key"},"Bearer raw"]}`))
	if value["email"] != "masked@example.test" || value["password"] != redactedValue {
		t.Fatalf("top-level redaction=%#v", value)
	}
	nested := value["nested"].(map[string]any)
	if nested["access_token"] != redactedValue || nested["safe"] != "visible" {
		t.Fatalf("nested redaction=%#v", nested)
	}
	headers := value["headers"].(map[string]any)
	if headers["Authorization"] != redactedValue {
		t.Fatalf("authorization redaction=%#v", headers)
	}
	items := value["items"].([]any)
	if items[0].(map[string]any)["api-key"] != redactedValue || items[1] != redactedValue {
		t.Fatalf("array redaction=%#v", items)
	}
}

func TestDecodeRedactedRejectsMalformedJSON(t *testing.T) {
	if value := decodeRedacted([]byte(`{"password"`)); len(value) != 0 {
		t.Fatalf("malformed JSON should not be exposed: %#v", value)
	}
}
