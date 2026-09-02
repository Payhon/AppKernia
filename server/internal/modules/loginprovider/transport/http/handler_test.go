package http

import (
	stdhttp "net/http"
	"testing"
)

func TestBrowserRedirectHeadersPreventTicketReferrerLeakage(t *testing.T) {
	header := stdhttp.Header{}
	setBrowserRedirectHeaders(header)
	for name, expected := range map[string]string{
		"Cache-Control":   "no-store",
		"Pragma":          "no-cache",
		"Referrer-Policy": "no-referrer",
	} {
		if actual := header.Get(name); actual != expected {
			t.Fatalf("%s=%q, want %q", name, actual, expected)
		}
	}
}
