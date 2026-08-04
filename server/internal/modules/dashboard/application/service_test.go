package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/dashboard/domain"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

type fakeAuthenticator struct {
	value iamdomain.AuthenticatedContext
	calls int
}

func (fake *fakeAuthenticator) Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error) {
	fake.calls++
	return fake.value, nil
}

type fakeRepository struct {
	summaryAccess  domain.SummaryAccess
	trendAccess    domain.TrendAccess
	activityAccess domain.ActivityAccess
}

func (fake *fakeRepository) Summary(_ context.Context, _ domain.Query, access domain.SummaryAccess) ([]domain.Metric, error) {
	fake.summaryAccess = access
	return []domain.Metric{}, nil
}

func (fake *fakeRepository) Trends(_ context.Context, _ domain.Query, access domain.TrendAccess) ([]domain.TrendSeries, error) {
	fake.trendAccess = access
	return []domain.TrendSeries{}, nil
}

func (fake *fakeRepository) Activity(_ context.Context, _ domain.Query, access domain.ActivityAccess) (domain.Activity, error) {
	fake.activityAccess = access
	return domain.Activity{}, nil
}

func TestDashboardPermissionsArePrunedBeforeRepositoryQueries(t *testing.T) {
	authenticator := &fakeAuthenticator{value: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{
		Tenant:      iamdomain.Tenant{ID: uuid.New()},
		Permissions: []string{"iam.user.read", "audit.security.read", "audit.operation.read"},
	}}}
	repository := &fakeRepository{}
	service := NewService(authenticator, repository)
	service.clock = func() time.Time { return time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC) }

	if _, err := service.Summary(context.Background(), "token", "ak-admin", "7d"); err != nil {
		t.Fatalf("summary: %v", err)
	}
	if !repository.summaryAccess.Users || !repository.summaryAccess.SecurityEvents || repository.summaryAccess.Sessions || repository.summaryAccess.FailedJobs || repository.summaryAccess.Messages {
		t.Fatalf("unexpected summary access: %#v", repository.summaryAccess)
	}
	if _, err := service.Trends(context.Background(), "token", "ak-admin", "30d"); err != nil {
		t.Fatalf("trends: %v", err)
	}
	if !repository.trendAccess.Users || !repository.trendAccess.SecurityEvents || repository.trendAccess.Logins || repository.trendAccess.FailedJobs {
		t.Fatalf("unexpected trend access: %#v", repository.trendAccess)
	}
	if _, err := service.Activity(context.Background(), "token", "ak-admin", "90d"); err != nil {
		t.Fatalf("activity: %v", err)
	}
	if !repository.activityAccess.Operations || !repository.activityAccess.SecurityEvents || repository.activityAccess.FailedJobs {
		t.Fatalf("unexpected activity access: %#v", repository.activityAccess)
	}
}

func TestDashboardRejectsUnknownRangeBeforeAuthentication(t *testing.T) {
	authenticator := &fakeAuthenticator{}
	service := NewService(authenticator, &fakeRepository{})

	_, err := service.Summary(context.Background(), "token", "ak-admin", "365d")

	if !errors.Is(err, ErrRangeInvalid) {
		t.Fatalf("expected range validation error, got %v", err)
	}
	if authenticator.calls != 0 {
		t.Fatalf("invalid range must not authenticate, got %d calls", authenticator.calls)
	}
}
