// Package publicurl builds links only from an operator-controlled origin.
package publicurl

import (
	"fmt"
	"github.com/google/uuid"
	"net/url"
	"strings"
)

func Validate(base string, development bool) error {
	if base == "" {
		return nil
	}
	u, err := url.Parse(base)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("invalid public web origin")
	}
	if u.Scheme == "https" {
		return nil
	}
	if development && u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1") {
		return nil
	}
	return fmt.Errorf("public web origin must use HTTPS")
}
func Path(appID uuid.UUID, suffix string) string { return "/h5/apps/" + appID.String() + suffix }
func Link(base string, appID uuid.UUID, suffix, locale string) string {
	if base == "" {
		return ""
	}
	out := strings.TrimRight(base, "/") + Path(appID, suffix)
	if locale != "" {
		out += "?lang=" + url.QueryEscape(locale)
	}
	return out
}
