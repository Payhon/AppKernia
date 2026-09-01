package application

import (
	"encoding/json"
	"github.com/appkernia/appkernia/server/internal/shared/richtext"
	"github.com/google/uuid"
	"html/template"
	"net/url"
	"strconv"
	"strings"
)

func RenderBody(raw json.RawMessage, format string, id uuid.UUID) (template.HTML, error) {
	return richtext.RenderBody(raw, format, id)
}
func bodyImage(raw string, id uuid.UUID) string { return richtext.ImageURL(raw, id) }

// externalVideo returns both the browser URL and the exact CSP source. The
// repository checks the App allowlist; this rendering-boundary check rejects
// unsafe schemes and credentials before a value reaches HTML or a header.
func externalVideo(raw string) (string, string) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Hostname() == "" || u.User != nil || u.Opaque != "" || strings.ContainsAny(u.Host, " \t\r\n;'\"*\\") {
		return "", ""
	}
	if port := u.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return "", ""
		}
	}
	origin := (&url.URL{Scheme: "https", Host: strings.ToLower(u.Host)}).String()
	return u.String(), origin
}
