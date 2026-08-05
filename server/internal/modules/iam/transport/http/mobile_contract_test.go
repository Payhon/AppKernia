package http

import (
	"os"
	"strings"
	"testing"
)

func TestMobileSessionWriterDoesNotSetAdminCookies(t *testing.T) {
	raw, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	start := strings.Index(source, "func (handler *Handler) writeMobileSession")
	if start < 0 {
		t.Fatal("writeMobileSession not found")
	}
	tail := source[start:]
	end := strings.Index(tail[1:], "\nfunc ")
	if end < 0 {
		t.Fatal("writeMobileSession boundary not found")
	}
	body := tail[:end+1]
	if strings.Contains(body, "Cookie.Set") || strings.Contains(body, refreshCookieName) || strings.Contains(body, csrfCookieName) {
		t.Fatalf("mobile session writer must not emit Admin cookies: %s", body)
	}
	if !strings.Contains(body, "RefreshToken") {
		t.Fatal("mobile session writer must return the rotated refresh token in its native-client payload")
	}
}

func TestMobileHandlersPinMobileAudience(t *testing.T) {
	raw, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, name := range []string{"MobileLogin", "MobileRefresh", "MobileLogout", "MobileContext", "MobileMe", "MobileSelfSessions"} {
		start := strings.Index(source, "func (handler *Handler) "+name)
		if start < 0 {
			t.Fatalf("%s not found", name)
		}
		fragment := source[start:min(len(source), start+1800)]
		if !strings.Contains(fragment, "mobileAudience") {
			t.Fatalf("%s is not pinned to mobile audience", name)
		}
	}
}
