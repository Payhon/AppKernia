package application

import (
	"context"
	"slices"
	"strings"
	"time"

	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/google/uuid"
)

func normalizeOperationsFilter(f notify.OperationsFilter, now time.Time) (notify.OperationsFilter, error) {
	f.Environment = strings.TrimSpace(f.Environment)
	f.Category = strings.TrimSpace(f.Category)
	f.Channel = strings.TrimSpace(f.Channel)
	f.Provider = strings.TrimSpace(f.Provider)
	f.TaskKind = strings.TrimSpace(f.TaskKind)
	f.Status = strings.TrimSpace(f.Status)
	f.Query = strings.TrimSpace(f.Query)
	if f.To.IsZero() {
		f.To = now.UTC().Add(time.Second)
	}
	if f.From.IsZero() {
		f.From = f.To.Add(-30 * 24 * time.Hour)
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	providers := []string{"", "apns", "fcm", "huawei_android", "honor", "xiaomi", "oppo", "vivo", "meizu", "harmony", "smtp", "aliyun", "tencent"}
	statuses := []string{"", "scheduled", "queued", "running", "retry_wait", "succeeded", "failed", "cancelled", "completed", "completed_with_failures", "expired"}
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || len([]rune(f.Query)) > 160 ||
		!f.From.Before(f.To) || f.To.Sub(f.From) > 400*24*time.Hour ||
		!oneOf(f.Environment, "", "development", "test", "staging", "production") ||
		!oneOf(f.Category, "", "service_security", "news_operations") ||
		!oneOf(f.Channel, "", "in_app", "email", "sms", "push", "webhook") ||
		!slices.Contains(providers, f.Provider) || !slices.Contains(statuses, f.Status) || len(f.TaskKind) > 160 {
		return f, notify.ErrInvalid
	}
	return f, nil
}

func (s *Service) OperationsSummary(ctx context.Context, token string, appID uuid.UUID, f notify.OperationsFilter) (notify.OperationsSummary, error) {
	auth, err := s.authorize(ctx, token, "notify.observability.read")
	if err != nil {
		return notify.OperationsSummary{}, err
	}
	f, err = normalizeOperationsFilter(f, s.clock())
	if err != nil || appID == uuid.Nil {
		return notify.OperationsSummary{}, notify.ErrInvalid
	}
	return s.repo.OperationsSummary(ctx, auth.Tenant.ID, appID, f)
}

func (s *Service) OperationsTrends(ctx context.Context, token string, appID uuid.UUID, f notify.OperationsFilter) ([]notify.TrendBucket, error) {
	auth, err := s.authorize(ctx, token, "notify.observability.read")
	if err != nil {
		return nil, err
	}
	f, err = normalizeOperationsFilter(f, s.clock())
	if err != nil || appID == uuid.Nil {
		return nil, notify.ErrInvalid
	}
	return s.repo.OperationsTrends(ctx, auth.Tenant.ID, appID, f)
}

func (s *Service) ListMessageRuns(ctx context.Context, token string, appID uuid.UUID, f notify.OperationsFilter) (notify.MessageRunPage, error) {
	auth, err := s.authorize(ctx, token, "notify.observability.read")
	if err != nil {
		return notify.MessageRunPage{}, err
	}
	f, err = normalizeOperationsFilter(f, s.clock())
	if err != nil || appID == uuid.Nil {
		return notify.MessageRunPage{}, notify.ErrInvalid
	}
	return s.repo.ListMessageRuns(ctx, auth.Tenant.ID, appID, f)
}

func (s *Service) GetMessageRun(ctx context.Context, token string, appID, id uuid.UUID) (notify.MessageRun, error) {
	auth, err := s.authorize(ctx, token, "notify.observability.read")
	if err != nil {
		return notify.MessageRun{}, err
	}
	if appID == uuid.Nil || id == uuid.Nil {
		return notify.MessageRun{}, notify.ErrInvalid
	}
	return s.repo.GetMessageRun(ctx, auth.Tenant.ID, appID, id)
}

func (s *Service) ListTaskRuns(ctx context.Context, token string, appID uuid.UUID, f notify.OperationsFilter) (notify.TaskRunPage, error) {
	auth, err := s.authorize(ctx, token, "notify.observability.read")
	if err != nil {
		return notify.TaskRunPage{}, err
	}
	f, err = normalizeOperationsFilter(f, s.clock())
	if err != nil || appID == uuid.Nil {
		return notify.TaskRunPage{}, notify.ErrInvalid
	}
	return s.repo.ListTaskRuns(ctx, auth.Tenant.ID, appID, f)
}

func (s *Service) GetTaskRun(ctx context.Context, token string, appID, id uuid.UUID) (notify.TaskRun, error) {
	auth, err := s.authorize(ctx, token, "notify.observability.read")
	if err != nil {
		return notify.TaskRun{}, err
	}
	if appID == uuid.Nil || id == uuid.Nil {
		return notify.TaskRun{}, notify.ErrInvalid
	}
	return s.repo.GetTaskRun(ctx, auth.Tenant.ID, appID, id)
}

func (s *Service) ListFailures(ctx context.Context, token string, appID uuid.UUID, f notify.OperationsFilter) (notify.FailurePage, error) {
	auth, err := s.authorize(ctx, token, "notify.observability.read")
	if err != nil {
		return notify.FailurePage{}, err
	}
	f, err = normalizeOperationsFilter(f, s.clock())
	if err != nil || appID == uuid.Nil {
		return notify.FailurePage{}, notify.ErrInvalid
	}
	return s.repo.ListFailures(ctx, auth.Tenant.ID, appID, f)
}

func (s *Service) RetryTasks(ctx context.Context, token string, appID uuid.UUID, p notify.Principal, input notify.RetryInput) ([]notify.RetryResult, error) {
	auth, err := s.authorize(ctx, token, "notify.task.retry")
	if err != nil {
		return nil, err
	}
	if appID == uuid.Nil || strings.TrimSpace(p.RequestID) == "" || len(input.Items) < 1 || len(input.Items) > 100 {
		return nil, notify.ErrInvalid
	}
	seen := map[uuid.UUID]bool{}
	for _, item := range input.Items {
		if item.TaskID == uuid.Nil || seen[item.TaskID] {
			return nil, notify.ErrInvalid
		}
		seen[item.TaskID] = true
	}
	return s.repo.RetryTasks(ctx, principal(auth, appID, p), input)
}
