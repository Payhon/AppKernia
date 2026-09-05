package http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	dashboardapp "github.com/appkernia/appkernia/server/internal/modules/dashboard/application"
	"github.com/appkernia/appkernia/server/internal/modules/dashboard/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
)

const adminAudience = "ak-admin"

type Handler struct {
	service *dashboardapp.Service
	catalog *i18n.Catalog
}

type metricResponse struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

type summaryResponse struct {
	Range   string           `json:"range"`
	StartAt string           `json:"start_at"`
	EndAt   string           `json:"end_at"`
	Metrics []metricResponse `json:"metrics"`
}

type trendPointResponse struct {
	Day   string `json:"day"`
	Value int64  `json:"value"`
}

type trendSeriesResponse struct {
	Key    string               `json:"key"`
	Points []trendPointResponse `json:"points"`
}

type trendsResponse struct {
	Range   string                `json:"range"`
	StartAt string                `json:"start_at"`
	EndAt   string                `json:"end_at"`
	Series  []trendSeriesResponse `json:"series"`
}

type operationActivityResponse struct {
	ID           string `json:"id"`
	ModuleCode   string `json:"module_code"`
	ActionName   string `json:"action_name"`
	ResourceType string `json:"resource_type"`
	Succeeded    bool   `json:"succeeded"`
	ErrorCode    string `json:"error_code"`
	OccurredAt   string `json:"occurred_at"`
}

type failedJobActivityResponse struct {
	ID           string `json:"id"`
	ScheduleCode string `json:"schedule_code"`
	ScheduleName string `json:"schedule_name"`
	ErrorCode    string `json:"error_code"`
	OccurredAt   string `json:"occurred_at"`
}

type securityActivityResponse struct {
	ID         string `json:"id"`
	EventType  string `json:"event_type"`
	Severity   string `json:"severity"`
	Source     string `json:"source"`
	OccurredAt string `json:"occurred_at"`
}

type activityResponse struct {
	Range          string                      `json:"range"`
	StartAt        string                      `json:"start_at"`
	EndAt          string                      `json:"end_at"`
	Operations     []operationActivityResponse `json:"operations"`
	FailedJobs     []failedJobActivityResponse `json:"failed_jobs"`
	SecurityEvents []securityActivityResponse  `json:"security_events"`
}

func NewHandler(service *dashboardapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func (handler *Handler) Summary(request *ghttp.Request) {
	result, err := handler.service.Summary(request.Context(), bearerToken(request.Header.Get("Authorization")), adminAudience, request.GetQuery("range").String())
	if handler.writeServiceError(request, err) {
		return
	}
	metrics := make([]metricResponse, 0, len(result.Data))
	for _, metric := range result.Data {
		metrics = append(metrics, metricResponse{Key: metric.Key, Value: metric.Value})
	}
	handler.writeSuccess(request, summaryResponse{
		Range: result.Range, StartAt: formatTime(result.StartAt), EndAt: formatTime(result.EndAt), Metrics: metrics,
	})
}

func (handler *Handler) Trends(request *ghttp.Request) {
	result, err := handler.service.Trends(request.Context(), bearerToken(request.Header.Get("Authorization")), adminAudience, request.GetQuery("range").String())
	if handler.writeServiceError(request, err) {
		return
	}
	series := make([]trendSeriesResponse, 0, len(result.Data))
	for _, item := range result.Data {
		points := make([]trendPointResponse, 0, len(item.Points))
		for _, point := range item.Points {
			points = append(points, trendPointResponse{Day: point.Day.UTC().Format("2006-01-02"), Value: point.Value})
		}
		series = append(series, trendSeriesResponse{Key: item.Key, Points: points})
	}
	handler.writeSuccess(request, trendsResponse{
		Range: result.Range, StartAt: formatTime(result.StartAt), EndAt: formatTime(result.EndAt), Series: series,
	})
}

func (handler *Handler) Activity(request *ghttp.Request) {
	result, err := handler.service.Activity(request.Context(), bearerToken(request.Header.Get("Authorization")), adminAudience, request.GetQuery("range").String())
	if handler.writeServiceError(request, err) {
		return
	}
	handler.writeSuccess(request, mapActivity(result.Range, result.StartAt, result.EndAt, result.Data))
}

func (handler *Handler) writeServiceError(request *ghttp.Request, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, dashboardapp.ErrRangeInvalid):
		handler.writeError(request, http.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
	case errors.Is(err, iamapp.ErrInvalidAccessToken):
		handler.writeError(request, http.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
	case errors.Is(err, iamapp.ErrAccessDenied):
		handler.writeError(request, http.StatusForbidden, "COMMON.FORBIDDEN", "errors.common.forbidden")
	default:
		handler.writeError(request, http.StatusInternalServerError, "COMMON.UNKNOWN", "errors.common.unknown")
	}
	return true
}

func (handler *Handler) writeSuccess(request *ghttp.Request, data any) {
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.Header().Set("Vary", "Authorization, Accept-Language")
	request.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(request)})
}

func (handler *Handler) writeError(request *ghttp.Request, status int, code, messageKey string) {
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Error{
		Error:     httpx.ErrorBody{Code: code, MessageKey: messageKey, Message: handler.catalog.Translate(httpx.Locale(request), messageKey, nil)},
		RequestID: httpx.RequestID(request),
	})
}

func mapActivity(rawRange string, startAt, endAt time.Time, data domain.Activity) activityResponse {
	result := activityResponse{
		Range: rawRange, StartAt: formatTime(startAt), EndAt: formatTime(endAt),
		Operations: []operationActivityResponse{}, FailedJobs: []failedJobActivityResponse{},
		SecurityEvents: []securityActivityResponse{},
	}
	for _, item := range data.Operations {
		result.Operations = append(result.Operations, operationActivityResponse{
			ID: item.ID.String(), ModuleCode: item.ModuleCode, ActionName: item.ActionName,
			ResourceType: item.ResourceType, Succeeded: item.Succeeded, ErrorCode: item.ErrorCode,
			OccurredAt: formatTime(item.OccurredAt),
		})
	}
	for _, item := range data.FailedJobs {
		result.FailedJobs = append(result.FailedJobs, failedJobActivityResponse{
			ID: item.ID.String(), ScheduleCode: item.ScheduleCode, ScheduleName: item.ScheduleName,
			ErrorCode: item.ErrorCode, OccurredAt: formatTime(item.OccurredAt),
		})
	}
	for _, item := range data.SecurityEvents {
		result.SecurityEvents = append(result.SecurityEvents, securityActivityResponse{
			ID: item.ID.String(), EventType: item.EventType, Severity: item.Severity,
			Source: item.Source, OccurredAt: formatTime(item.OccurredAt),
		})
	}
	return result
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
