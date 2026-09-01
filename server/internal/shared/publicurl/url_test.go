package publicurl

import (
	"github.com/google/uuid"
	"testing"
)

func TestTrustedOrigin(t *testing.T) {
	for _, base := range []string{"https://public.example", "https://public.example:8443/", ""} {
		if err := Validate(base, false); err != nil {
			t.Fatal(err)
		}
	}
	for _, base := range []string{"http://public.example", "https://user:password@example.test", "https://example.test/subpath", "https://example.test?redirect=evil", "https://example.test#fragment", "//example.test", "javascript:alert(1)"} {
		if Validate(base, false) == nil {
			t.Errorf("accepted %s", base)
		}
	}
	if Validate("http://localhost:18080", true) != nil {
		t.Fatal("development loopback rejected")
	}
	if Validate("http://0.0.0.0:18080", true) == nil {
		t.Fatal("nonloopback origin accepted")
	}
	id := uuid.New()
	if Link("", id, "/download", "zh-CN") != "" {
		t.Fatal("missing origin built link")
	}
	if got := Link("https://public.example/", id, "/download", "en-US"); got != "https://public.example/h5/apps/"+id.String()+"/download?lang=en-US" {
		t.Fatal(got)
	}
}
