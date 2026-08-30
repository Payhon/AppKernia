package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/platform/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	health            *application.HealthService
	catalog           *i18n.Catalog
	adminFeatureFlags map[string]bool
	publicConfigs     publicConfigStore
	pushMetrics       pushMetricStore
}

type publicConfigStore interface {
	ListPublicConfigs(context.Context) (map[string]json.RawMessage, error)
}

type pushMetricStore interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func NewHandler(health *application.HealthService, catalog *i18n.Catalog, adminFeatureFlags map[string]bool, publicConfigs publicConfigStore, metricStores ...pushMetricStore) *Handler {
	handler := &Handler{health: health, catalog: catalog, adminFeatureFlags: adminFeatureFlags, publicConfigs: publicConfigs}
	if len(metricStores) > 0 {
		handler.pushMetrics = metricStores[0]
	}
	return handler
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
	var output strings.Builder
	output.WriteString("# HELP appkernia_up Whether the API process is running.\n")
	output.WriteString("# TYPE appkernia_up gauge\n")
	output.WriteString("appkernia_up 1\n")
	if h.pushMetrics != nil {
		writePushMetrics(r.Context(), h.pushMetrics, &output)
	}
	r.Response.Write(output.String())
}

func writePushMetrics(ctx context.Context, store pushMetricStore, output *strings.Builder) {
	metricContext, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	rows, err := store.Query(metricContext, `SELECT app_id::text,COALESCE(provider,'unknown'),COALESCE(metadata->>'push_category','unknown'),
COALESCE(provider_result,status),count(*),count(*) FILTER (WHERE opened_at IS NOT NULL),
COALESCE(sum(extract(epoch FROM accepted_at-created_at)) FILTER (WHERE accepted_at IS NOT NULL),0),
count(*) FILTER (WHERE accepted_at IS NOT NULL)
FROM notify.deliveries WHERE channel='push' AND created_at>=now()-interval '30 days'
GROUP BY app_id,provider,metadata->>'push_category',provider_result,status ORDER BY app_id,provider,3,4`)
	if err != nil {
		output.WriteString("# HELP appkernia_push_metrics_scrape_error Whether push metrics could not be read.\n# TYPE appkernia_push_metrics_scrape_error gauge\nappkernia_push_metrics_scrape_error 1\n")
		return
	}
	defer rows.Close()
	output.WriteString("# HELP appkernia_push_deliveries_total Push delivery outcomes in the last 30 days.\n# TYPE appkernia_push_deliveries_total gauge\n")
	output.WriteString("# HELP appkernia_push_opened_total Push notifications opened in the last 30 days.\n# TYPE appkernia_push_opened_total gauge\n")
	output.WriteString("# HELP appkernia_push_queue_delay_seconds_sum Time from delivery creation to provider acceptance.\n# TYPE appkernia_push_queue_delay_seconds_sum gauge\n")
	output.WriteString("# HELP appkernia_push_queue_delay_seconds_count Accepted deliveries represented by the queue-delay sum.\n# TYPE appkernia_push_queue_delay_seconds_count gauge\n")
	for rows.Next() {
		var appID, provider, category, result string
		var deliveries, opened, delayCount int64
		var delaySum float64
		if rows.Scan(&appID, &provider, &category, &result, &deliveries, &opened, &delaySum, &delayCount) != nil {
			output.WriteString("# HELP appkernia_push_metrics_scrape_error Whether push metrics could not be read.\n# TYPE appkernia_push_metrics_scrape_error gauge\nappkernia_push_metrics_scrape_error 1\n")
			return
		}
		labels := pushMetricLabels(appID, provider, category, result)
		_, _ = fmt.Fprintf(output, "appkernia_push_deliveries_total{%s} %d\n", labels, deliveries)
		_, _ = fmt.Fprintf(output, "appkernia_push_opened_total{%s} %d\n", labels, opened)
		_, _ = fmt.Fprintf(output, "appkernia_push_queue_delay_seconds_sum{%s} %g\n", labels, delaySum)
		_, _ = fmt.Fprintf(output, "appkernia_push_queue_delay_seconds_count{%s} %d\n", labels, delayCount)
	}
	if rows.Err() != nil {
		output.WriteString("# HELP appkernia_push_metrics_scrape_error Whether push metrics could not be read.\n# TYPE appkernia_push_metrics_scrape_error gauge\nappkernia_push_metrics_scrape_error 1\n")
		return
	}
	writePushBacklogMetrics(metricContext, store, output)
	writePushProviderFaultMetrics(metricContext, store, output)
	writeNotificationTaskMetrics(metricContext, store, output)
	writeNotificationPipelineMetrics(metricContext, store, output)
}

func writeNotificationTaskMetrics(ctx context.Context, store pushMetricStore, output *strings.Builder) {
	rows, err := store.Query(ctx, `SELECT COALESCE(app_id::text,'none'),task_kind,status,count(*),
		COALESCE(max(extract(epoch FROM (now()-scheduled_at))) FILTER(WHERE status IN ('scheduled','queued','retry_wait')),0),
		COALESCE(sum(extract(epoch FROM (finalized_at-started_at))) FILTER(WHERE finalized_at IS NOT NULL AND started_at IS NOT NULL),0),
		count(*) FILTER(WHERE finalized_at IS NOT NULL AND started_at IS NOT NULL),
		COALESCE(sum(GREATEST(attempt_count-1,0)),0),
		count(*) FILTER(WHERE status='failed' AND COALESCE(last_result_class,'') NOT IN ('throttled','transient'))
		FROM jobs.task_runs WHERE module_code='notify' AND created_at>=now()-interval '30 days'
		GROUP BY app_id,task_kind,status ORDER BY app_id,task_kind,status`)
	if err != nil {
		return
	}
	defer rows.Close()
	output.WriteString("# HELP appkernia_notification_task_total Notification task count by current state.\n# TYPE appkernia_notification_task_total gauge\n")
	output.WriteString("# HELP appkernia_notification_task_oldest_wait_seconds Oldest unfinished notification task age.\n# TYPE appkernia_notification_task_oldest_wait_seconds gauge\n")
	output.WriteString("# HELP appkernia_notification_task_duration_seconds_sum Completed notification task execution time.\n# TYPE appkernia_notification_task_duration_seconds_sum gauge\n")
	output.WriteString("# HELP appkernia_notification_task_duration_seconds_count Completed notification tasks represented by the duration sum.\n# TYPE appkernia_notification_task_duration_seconds_count gauge\n")
	output.WriteString("# HELP appkernia_notification_task_retries_total Retry attempts for notification tasks in the last 30 days.\n# TYPE appkernia_notification_task_retries_total gauge\n")
	output.WriteString("# HELP appkernia_notification_task_permanent_failures_total Terminal non-transient notification task failures in the last 30 days.\n# TYPE appkernia_notification_task_permanent_failures_total gauge\n")
	for rows.Next() {
		var appID, kind, status string
		var count, durationCount, retries, permanentFailures int64
		var oldest, durationSum float64
		if rows.Scan(&appID, &kind, &status, &count, &oldest, &durationSum, &durationCount, &retries, &permanentFailures) != nil {
			return
		}
		labels := fmt.Sprintf("app_id=\"%s\",kind=\"%s\",status=\"%s\"", prometheusLabel(appID), prometheusLabel(kind), prometheusLabel(status))
		_, _ = fmt.Fprintf(output, "appkernia_notification_task_total{%s} %d\n", labels, count)
		_, _ = fmt.Fprintf(output, "appkernia_notification_task_oldest_wait_seconds{%s} %g\n", labels, oldest)
		_, _ = fmt.Fprintf(output, "appkernia_notification_task_duration_seconds_sum{%s} %g\n", labels, durationSum)
		_, _ = fmt.Fprintf(output, "appkernia_notification_task_duration_seconds_count{%s} %d\n", labels, durationCount)
		_, _ = fmt.Fprintf(output, "appkernia_notification_task_retries_total{%s} %d\n", labels, retries)
		_, _ = fmt.Fprintf(output, "appkernia_notification_task_permanent_failures_total{%s} %d\n", labels, permanentFailures)
	}
}

func writeNotificationPipelineMetrics(ctx context.Context, store pushMetricStore, output *strings.Builder) {
	rows, err := store.Query(ctx, `SELECT app_id::text,status,count(*),
		COALESCE(sum(extract(epoch FROM (completed_at-created_at))) FILTER(WHERE completed_at IS NOT NULL),0),
		count(*) FILTER(WHERE completed_at IS NOT NULL)
		FROM notify.message_runs WHERE created_at>=now()-interval '30 days'
		GROUP BY app_id,status ORDER BY app_id,status`)
	if err != nil {
		return
	}
	defer rows.Close()
	output.WriteString("# HELP appkernia_notification_pipeline_total Message pipeline runs by current state in the last 30 days.\n# TYPE appkernia_notification_pipeline_total gauge\n")
	output.WriteString("# HELP appkernia_notification_pipeline_duration_seconds_sum Completed message pipeline duration.\n# TYPE appkernia_notification_pipeline_duration_seconds_sum gauge\n")
	output.WriteString("# HELP appkernia_notification_pipeline_duration_seconds_count Completed message pipelines represented by the duration sum.\n# TYPE appkernia_notification_pipeline_duration_seconds_count gauge\n")
	for rows.Next() {
		var appID, status string
		var count, durationCount int64
		var durationSum float64
		if rows.Scan(&appID, &status, &count, &durationSum, &durationCount) != nil {
			return
		}
		labels := fmt.Sprintf("app_id=\"%s\",status=\"%s\"", prometheusLabel(appID), prometheusLabel(status))
		_, _ = fmt.Fprintf(output, "appkernia_notification_pipeline_total{%s} %d\n", labels, count)
		_, _ = fmt.Fprintf(output, "appkernia_notification_pipeline_duration_seconds_sum{%s} %g\n", labels, durationSum)
		_, _ = fmt.Fprintf(output, "appkernia_notification_pipeline_duration_seconds_count{%s} %d\n", labels, durationCount)
	}
}

func writePushBacklogMetrics(ctx context.Context, store pushMetricStore, output *strings.Builder) {
	rows, err := store.Query(ctx, `SELECT app_id::text,COALESCE(provider,'unknown'),status,count(*)
FROM notify.deliveries WHERE channel='push' AND status IN ('pending','processing') GROUP BY app_id,provider,status ORDER BY app_id,provider,status`)
	if err != nil {
		return
	}
	defer rows.Close()
	output.WriteString("# HELP appkernia_push_queue_backlog Pending and processing push deliveries.\n# TYPE appkernia_push_queue_backlog gauge\n")
	for rows.Next() {
		var appID, provider, status string
		var count int64
		if rows.Scan(&appID, &provider, &status, &count) == nil {
			_, _ = fmt.Fprintf(output, "appkernia_push_queue_backlog{app_id=\"%s\",provider=\"%s\",status=\"%s\"} %d\n", prometheusLabel(appID), prometheusLabel(provider), prometheusLabel(status), count)
		}
	}
}

func writePushProviderFaultMetrics(ctx context.Context, store pushMetricStore, output *strings.Builder) {
	rows, err := store.Query(ctx, `SELECT app_id::text,provider,environment,count(*)
FROM notify.push_provider_configs WHERE status='faulted' GROUP BY app_id,provider,environment ORDER BY app_id,provider,environment`)
	if err != nil {
		return
	}
	defer rows.Close()
	output.WriteString("# HELP appkernia_push_provider_fault Faulted push provider configurations.\n# TYPE appkernia_push_provider_fault gauge\n")
	for rows.Next() {
		var appID, provider, environment string
		var count int64
		if rows.Scan(&appID, &provider, &environment, &count) == nil {
			_, _ = fmt.Fprintf(output, "appkernia_push_provider_fault{app_id=\"%s\",provider=\"%s\",environment=\"%s\"} %d\n", prometheusLabel(appID), prometheusLabel(provider), prometheusLabel(environment), count)
		}
	}
}

func pushMetricLabels(appID, provider, category, result string) string {
	return fmt.Sprintf("app_id=\"%s\",provider=\"%s\",category=\"%s\",result=\"%s\"", prometheusLabel(appID), prometheusLabel(provider), prometheusLabel(category), prometheusLabel(result))
}

func prometheusLabel(value string) string {
	return strings.NewReplacer("\\", "\\\\", "\n", "\\n", "\"", "\\\"").Replace(value)
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
