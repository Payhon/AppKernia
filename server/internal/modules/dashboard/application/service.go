package application

import (
	"context"
	"errors"
	"strings"
	"time"

	dashboarddomain "github.com/appkernia/appkernia/server/internal/modules/dashboard/domain"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
)

var ErrRangeInvalid = errors.New("dashboard range is invalid")

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Result[T any] struct {
	Range   string
	StartAt time.Time
	EndAt   time.Time
	Data    T
}

type Service struct {
	authenticator Authenticator
	repository    dashboarddomain.Repository
	clock         func() time.Time
}

func NewService(authenticator Authenticator, repository dashboarddomain.Repository) *Service {
	return &Service{authenticator: authenticator, repository: repository, clock: time.Now}
}

func (service *Service) Summary(ctx context.Context, accessToken, audience, rawRange string) (Result[[]dashboarddomain.Metric], error) {
	authenticated, query, normalizedRange, err := service.scope(ctx, accessToken, audience, rawRange)
	if err != nil {
		return Result[[]dashboarddomain.Metric]{}, err
	}
	permissions := permissionSet(authenticated.Permissions)
	data, err := service.repository.Summary(ctx, query, dashboarddomain.SummaryAccess{
		Users: permissions["iam.user.read"], Sessions: permissions["iam.session.read"],
		FailedJobs: permissions["jobs.run.read"], SecurityEvents: permissions["audit.security.read"],
		Messages: permissions["notify.notice.read"],
	})
	return Result[[]dashboarddomain.Metric]{Range: normalizedRange, StartAt: query.StartAt, EndAt: query.EndAt, Data: data}, err
}

func (service *Service) Trends(ctx context.Context, accessToken, audience, rawRange string) (Result[[]dashboarddomain.TrendSeries], error) {
	authenticated, query, normalizedRange, err := service.scope(ctx, accessToken, audience, rawRange)
	if err != nil {
		return Result[[]dashboarddomain.TrendSeries]{}, err
	}
	permissions := permissionSet(authenticated.Permissions)
	data, err := service.repository.Trends(ctx, query, dashboarddomain.TrendAccess{
		Logins: permissions["audit.login.read"], Users: permissions["iam.user.read"],
		FailedJobs: permissions["jobs.run.read"], SecurityEvents: permissions["audit.security.read"],
	})
	return Result[[]dashboarddomain.TrendSeries]{Range: normalizedRange, StartAt: query.StartAt, EndAt: query.EndAt, Data: data}, err
}

func (service *Service) Activity(ctx context.Context, accessToken, audience, rawRange string) (Result[dashboarddomain.Activity], error) {
	authenticated, query, normalizedRange, err := service.scope(ctx, accessToken, audience, rawRange)
	if err != nil {
		return Result[dashboarddomain.Activity]{}, err
	}
	permissions := permissionSet(authenticated.Permissions)
	data, err := service.repository.Activity(ctx, query, dashboarddomain.ActivityAccess{
		Operations: permissions["audit.operation.read"], FailedJobs: permissions["jobs.run.read"],
		SecurityEvents: permissions["audit.security.read"],
	})
	return Result[dashboarddomain.Activity]{Range: normalizedRange, StartAt: query.StartAt, EndAt: query.EndAt, Data: data}, err
}

func (service *Service) scope(
	ctx context.Context,
	accessToken string,
	audience string,
	rawRange string,
) (iamdomain.AuthenticatedContext, dashboarddomain.Query, string, error) {
	days, normalizedRange, err := normalizeRange(rawRange)
	if err != nil {
		return iamdomain.AuthenticatedContext{}, dashboarddomain.Query{}, "", err
	}
	authenticated, err := service.authenticator.Authenticate(ctx, accessToken, audience)
	if err != nil {
		return iamdomain.AuthenticatedContext{}, dashboarddomain.Query{}, "", err
	}
	endAt := service.clock().UTC()
	startAt := time.Date(endAt.Year(), endAt.Month(), endAt.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(days - 1))
	return authenticated, dashboarddomain.Query{
		TenantID: authenticated.Tenant.ID, StartAt: startAt, EndAt: endAt,
	}, normalizedRange, nil
}

func normalizeRange(raw string) (int, string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = "30d"
	}
	switch value {
	case "7d":
		return 7, value, nil
	case "30d":
		return 30, value, nil
	case "90d":
		return 90, value, nil
	default:
		return 0, "", ErrRangeInvalid
	}
}

func permissionSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}
