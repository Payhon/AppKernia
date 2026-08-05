package repository

import (
	"encoding/json"
	"testing"
)

func TestNormalizeEndpointAndObjectKeySafety(t *testing.T) {
	host, secure, err := normalizeEndpoint("https://s3.example.test", false)
	if err != nil || host != "s3.example.test" || !secure {
		t.Fatalf("normalized endpoint = %q secure=%v err=%v", host, secure, err)
	}
	if _, _, err = normalizeEndpoint("https://user:secret@s3.example.test/path", true); err == nil {
		t.Fatal("endpoint with credentials and path must be rejected")
	}
	for _, key := range []string{"../escape", "/absolute", `bad\key`} {
		if validObjectKey(key) {
			t.Fatalf("unsafe object key accepted: %q", key)
		}
	}
	if !validObjectKey("appkernia/files/tenant/object") {
		t.Fatal("safe object key rejected")
	}
}

func TestPolicyValueParsingIsBoundedAndDeduplicated(t *testing.T) {
	values := map[string]json.RawMessage{
		"limit": []byte(`209715200`),
		"types": []byte(`["image/png","image/png"," text/plain ",""]`),
	}
	if got := boundedInt64(values, "limit", 12, 100); got != 12 {
		t.Fatalf("bounded value = %d", got)
	}
	types := stringSlice(values, "types", []string{"fallback"})
	if len(types) != 2 || types[0] != "image/png" || types[1] != "text/plain" {
		t.Fatalf("types = %#v", types)
	}
}
