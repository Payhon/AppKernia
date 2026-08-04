package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"

	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
)

const (
	requestIDKey = "appkernia.request_id"
	localeKey    = "appkernia.locale"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

func RequestContext(r *ghttp.Request) {
	requestID := r.Header.Get("X-Request-ID")
	if !requestIDPattern.MatchString(requestID) {
		requestID = newRequestID()
	}
	locale := i18n.ResolveAcceptLanguage(r.Header.Get("Accept-Language"))
	r.SetCtxVar(requestIDKey, requestID)
	r.SetCtxVar(localeKey, string(locale))
	r.Response.Header().Set("X-Request-ID", requestID)
	r.Response.Header().Set("Content-Language", string(locale))
	r.Middleware.Next()
}

func RequestID(r *ghttp.Request) string {
	return r.GetCtxVar(requestIDKey).String()
}

func Locale(r *ghttp.Request) i18n.Locale {
	return i18n.Normalize(r.GetCtxVar(localeKey).String())
}

func newRequestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(value[:])
}
