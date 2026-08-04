package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/appkernia/appkernia/server/internal/modules/platform/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
)

type Handler struct {
	health            *application.HealthService
	catalog           *i18n.Catalog
	adminFeatureFlags map[string]bool
	publicConfigs     publicConfigStore
}

type publicConfigStore interface {
	ListPublicConfigs(context.Context) (map[string]json.RawMessage, error)
}

func NewHandler(health *application.HealthService, catalog *i18n.Catalog, adminFeatureFlags map[string]bool, publicConfigs publicConfigStore) *Handler {
	return &Handler{health: health, catalog: catalog, adminFeatureFlags: adminFeatureFlags, publicConfigs: publicConfigs}
}

func (h *Handler) Live(r *ghttp.Request) {
	r.Response.WriteJsonExit(httpx.Success[map[string]string]{
		Code:      "OK",
		Message:   "OK",
		Data:      map[string]string{"status": "live"},
		RequestID: httpx.RequestID(r),
	})
}

func (h *Handler) Ready(r *ghttp.Request) {
	if err := h.health.Ready(r.Context()); err != nil {
		locale := httpx.Locale(r)
		r.Response.WriteHeader(http.StatusServiceUnavailable)
		r.Response.WriteJsonExit(httpx.Error{
			Error: httpx.ErrorBody{
				Code:       "PLATFORM.DATABASE.UNAVAILABLE",
				MessageKey: "errors.common.unknown",
				Message:    h.catalog.Translate(locale, "errors.common.unknown", nil),
			},
			RequestID: httpx.RequestID(r),
		})
		return
	}
	r.Response.WriteJsonExit(httpx.Success[map[string]string]{
		Code:      "OK",
		Message:   "OK",
		Data:      map[string]string{"status": "ready"},
		RequestID: httpx.RequestID(r),
	})
}

func (h *Handler) Metrics(r *ghttp.Request) {
	r.Response.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	r.Response.Write("# HELP appkernia_up Whether the API process is running.\n")
	r.Response.Write("# TYPE appkernia_up gauge\n")
	r.Response.Write("appkernia_up 1\n")
}

func (h *Handler) PublicConfig(r *ghttp.Request) {
	values, err := h.publicConfigs.ListPublicConfigs(r.Context())
	if err != nil {
		locale := httpx.Locale(r)
		r.Response.WriteHeader(http.StatusServiceUnavailable)
		r.Response.WriteJsonExit(httpx.Error{
			Error: httpx.ErrorBody{
				Code:       "PLATFORM.CONFIG.UNAVAILABLE",
				MessageKey: "errors.common.unknown",
				Message:    h.catalog.Translate(locale, "errors.common.unknown", nil),
			},
			RequestID: httpx.RequestID(r),
		})
		return
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.Header().Set("Vary", "Accept-Language")
	locale := httpx.Locale(r)
	r.Response.WriteJsonExit(httpx.Success[map[string]any]{
		Code:      "OK",
		Message:   "OK",
		Data:      appPublicConfigData(locale, values),
		RequestID: httpx.RequestID(r),
	})
}

func appPublicConfigData(locale i18n.Locale, values map[string]json.RawMessage) map[string]any {
	return map[string]any{
		"locale":            locale,
		"default_locale":    i18n.LocaleZhCN,
		"supported_locales": i18n.Supported(),
		"feature_flags": map[string]bool{
			"registration_enabled": true,
			"otp_login_enabled":    false,
			"dark_mode":            false,
		},
		"settings": values,
	}
}

func (h *Handler) AdminPublicConfig(r *ghttp.Request) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.Header().Set("Vary", "Accept-Language")
	r.Response.WriteJsonExit(httpx.Success[map[string]any]{
		Code:      "OK",
		Message:   "OK",
		Data:      adminPublicConfigData(httpx.Locale(r), h.adminFeatureFlags),
		RequestID: httpx.RequestID(r),
	})
}

func adminPublicConfigData(locale i18n.Locale, configured map[string]bool) map[string]any {
	flags := map[string]bool{
		"admin_registration": false,
		"password_recovery":  false,
		"avatar_upload":      false,
		"oauth":              false,
	}
	for key := range flags {
		if configured[key] {
			flags[key] = true
		}
	}
	return map[string]any{
		"locale":            locale,
		"default_locale":    i18n.LocaleZhCN,
		"supported_locales": i18n.Supported(),
		"feature_flags":     flags,
		"settings":          map[string]json.RawMessage{},
	}
}
