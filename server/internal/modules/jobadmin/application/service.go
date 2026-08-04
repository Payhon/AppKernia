package application

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	jobs "github.com/appkernia/appkernia/server/internal/modules/jobadmin/domain"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Service struct {
	auth  Authenticator
	repo  jobs.Repository
	clock func() time.Time
}

func NewService(auth Authenticator, repo jobs.Repository) *Service {
	return &Service{auth: auth, repo: repo, clock: time.Now}
}

func (s *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	auth, err := s.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return iamdomain.AuthenticatedContext{}, err
	}
	if !slices.Contains(auth.Permissions, permission) {
		return iamdomain.AuthenticatedContext{}, jobs.ErrForbidden
	}
	return auth, nil
}

func principal(auth iamdomain.AuthenticatedContext, p jobs.Principal) jobs.Principal {
	p.TenantID, p.UserID, p.SessionID = auth.Tenant.ID, auth.User.ID, auth.SessionID
	return p
}

func normalizePage(in jobs.PageFilter) (jobs.PageFilter, error) {
	in.Query, in.Status, in.TimeZone = strings.TrimSpace(in.Query), strings.TrimSpace(in.Status), strings.TrimSpace(in.TimeZone)
	if in.Page == 0 {
		in.Page = 1
	}
	if in.PageSize == 0 {
		in.PageSize = 20
	}
	if in.Page < 1 || in.PageSize < 1 || in.PageSize > 100 || len([]rune(in.Query)) > 160 || !oneOf(in.Status, "", "active", "paused", "disabled") {
		return in, jobs.ErrInvalid
	}
	if in.TimeZone != "" {
		if _, err := time.LoadLocation(in.TimeZone); err != nil {
			return in, jobs.ErrInvalid
		}
	}
	return in, nil
}

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,95}$`)
var queuePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)

func normalizeInput(in jobs.ScheduleInput, now time.Time) (jobs.ScheduleInput, jobs.CronPreview, error) {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.HandlerKey = strings.TrimSpace(in.HandlerKey)
	in.CronExpression = strings.Join(strings.Fields(in.CronExpression), " ")
	in.TimeZone = strings.TrimSpace(in.TimeZone)
	in.QueueName = strings.TrimSpace(in.QueueName)
	in.OverlapPolicy = strings.TrimSpace(in.OverlapPolicy)
	in.MisfirePolicy = strings.TrimSpace(in.MisfirePolicy)
	definition, found := jobs.FindCompiledHandler(in.HandlerKey)
	if !found {
		return in, jobs.CronPreview{}, jobs.ErrHandlerUnknown
	}
	if in.TimeZone == "" {
		in.TimeZone = "UTC"
	}
	if in.QueueName == "" {
		in.QueueName = definition.DefaultQueue
	}
	if in.OverlapPolicy == "" {
		in.OverlapPolicy = "skip"
	}
	if in.MisfirePolicy == "" {
		in.MisfirePolicy = "fire_once"
	}
	if in.TimeoutSeconds == 0 {
		in.TimeoutSeconds = 300
	}
	if in.MaxAttempts == 0 {
		in.MaxAttempts = 3
	}
	if !codePattern.MatchString(in.Code) || len([]rune(in.Name)) < 1 || len([]rune(in.Name)) > 160 || !queuePattern.MatchString(in.QueueName) || in.QueueName != definition.DefaultQueue || !oneOf(in.OverlapPolicy, "allow", "skip", "replace") || !oneOf(in.MisfirePolicy, "ignore", "fire_once", "catch_up") || in.TimeoutSeconds < 1 || in.TimeoutSeconds > 86400 || in.MaxAttempts < 1 || in.MaxAttempts > 100 {
		return in, jobs.CronPreview{}, jobs.ErrInvalid
	}
	if len(in.Payload) == 0 {
		in.Payload = json.RawMessage(`{}`)
	}
	trimmedPayload := strings.TrimSpace(string(in.Payload))
	if len(in.Payload) > 64*1024 || !json.Valid(in.Payload) || !strings.HasPrefix(trimmedPayload, "{") {
		return in, jobs.CronPreview{}, jobs.ErrInvalid
	}
	var payload map[string]json.RawMessage
	if json.Unmarshal(in.Payload, &payload) != nil {
		return in, jobs.CronPreview{}, jobs.ErrInvalid
	}
	allowed := make(map[string]struct{}, len(definition.AllowedPayloads))
	for _, key := range definition.AllowedPayloads {
		allowed[key] = struct{}{}
	}
	for key := range payload {
		if _, ok := allowed[key]; !ok {
			return in, jobs.CronPreview{}, jobs.ErrInvalid
		}
	}
	preview, err := PreviewCron(in.CronExpression, in.TimeZone, now)
	if err != nil {
		return in, jobs.CronPreview{}, err
	}
	return in, preview, nil
}

func PreviewCron(expression, timeZone string, now time.Time) (jobs.CronPreview, error) {
	expression = strings.Join(strings.Fields(expression), " ")
	timeZone = strings.TrimSpace(timeZone)
	if timeZone == "" {
		timeZone = "UTC"
	}
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return jobs.CronPreview{}, jobs.ErrInvalid
	}
	schedule, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse(expression)
	if err != nil {
		return jobs.CronPreview{}, jobs.ErrInvalid
	}
	cursor := now.UTC().In(location)
	next := make([]time.Time, 0, 5)
	for range 5 {
		cursor = schedule.Next(cursor)
		if cursor.IsZero() {
			return jobs.CronPreview{}, jobs.ErrInvalid
		}
		next = append(next, cursor.UTC())
	}
	return jobs.CronPreview{CronExpression: expression, TimeZone: timeZone, NextRuns: next}, nil
}

func (s *Service) Handlers(ctx context.Context, token string) ([]jobs.HandlerDefinition, error) {
	if _, err := s.authorize(ctx, token, "jobs.schedule.read"); err != nil {
		return nil, err
	}
	return jobs.CompiledHandlers(), nil
}

func (s *Service) Preview(ctx context.Context, token string, input jobs.ScheduleInput) (jobs.CronPreview, error) {
	if _, err := s.authorize(ctx, token, "jobs.schedule.read"); err != nil {
		return jobs.CronPreview{}, err
	}
	_, preview, err := normalizeInput(input, s.clock().UTC())
	return preview, err
}

func (s *Service) List(ctx context.Context, token string, filter jobs.PageFilter) (jobs.SchedulePage, error) {
	auth, err := s.authorize(ctx, token, "jobs.schedule.read")
	if err != nil {
		return jobs.SchedulePage{}, err
	}
	filter, err = normalizePage(filter)
	if err != nil {
		return jobs.SchedulePage{}, err
	}
	return s.repo.ListSchedules(ctx, auth.Tenant.ID, filter)
}

func (s *Service) Create(ctx context.Context, token string, p jobs.Principal, input jobs.ScheduleInput) (jobs.Schedule, error) {
	auth, err := s.authorize(ctx, token, "jobs.schedule.create")
	if err != nil {
		return jobs.Schedule{}, err
	}
	input, preview, err := normalizeInput(input, s.clock().UTC())
	if err != nil {
		return jobs.Schedule{}, err
	}
	if strings.TrimSpace(p.RequestID) == "" {
		return jobs.Schedule{}, jobs.ErrInvalid
	}
	return s.repo.CreateSchedule(ctx, principal(auth, p), input, preview.NextRuns[0])
}

func (s *Service) Update(ctx context.Context, token string, p jobs.Principal, id uuid.UUID, input jobs.ScheduleInput) (jobs.Schedule, error) {
	auth, err := s.authorize(ctx, token, "jobs.schedule.update")
	if err != nil {
		return jobs.Schedule{}, err
	}
	input, preview, err := normalizeInput(input, s.clock().UTC())
	if err != nil {
		return jobs.Schedule{}, err
	}
	if id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" {
		return jobs.Schedule{}, jobs.ErrInvalid
	}
	return s.repo.UpdateSchedule(ctx, principal(auth, p), id, input, preview.NextRuns[0])
}

func (s *Service) ChangeStatus(ctx context.Context, token string, p jobs.Principal, id uuid.UUID, target string) (jobs.Schedule, error) {
	auth, err := s.authorize(ctx, token, "jobs.schedule.pause")
	if err != nil {
		return jobs.Schedule{}, err
	}
	if id == uuid.Nil || !oneOf(target, "active", "paused") || strings.TrimSpace(p.RequestID) == "" {
		return jobs.Schedule{}, jobs.ErrInvalid
	}
	var next *time.Time
	if target == "active" {
		item, getErr := s.repo.GetSchedule(ctx, auth.Tenant.ID, id)
		if getErr != nil {
			return jobs.Schedule{}, getErr
		}
		preview, previewErr := PreviewCron(item.CronExpression, item.TimeZone, s.clock().UTC())
		if previewErr != nil {
			return jobs.Schedule{}, previewErr
		}
		next = &preview.NextRuns[0]
	}
	return s.repo.ChangeStatus(ctx, principal(auth, p), id, target, next)
}

func (s *Service) Execute(ctx context.Context, token string, p jobs.Principal, id uuid.UUID, idempotencyKey string) (jobs.Run, error) {
	auth, err := s.authorize(ctx, token, "jobs.schedule.execute")
	if err != nil {
		return jobs.Run{}, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" || len(idempotencyKey) < 16 || len(idempotencyKey) > 128 {
		return jobs.Run{}, jobs.ErrInvalid
	}
	return s.repo.Execute(ctx, principal(auth, p), id, idempotencyKey, s.clock().UTC())
}

func (s *Service) Runs(ctx context.Context, token string, id uuid.UUID, filter jobs.PageFilter) (jobs.RunPage, error) {
	auth, err := s.authorize(ctx, token, "jobs.run.read")
	if err != nil {
		return jobs.RunPage{}, err
	}
	filter.Query, filter.TimeZone = "", ""
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	if filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 || !oneOf(filter.Status, "", "queued", "running", "succeeded", "failed", "cancelled", "skipped") || id == uuid.Nil {
		return jobs.RunPage{}, jobs.ErrInvalid
	}
	return s.repo.ListRuns(ctx, auth.Tenant.ID, id, filter)
}

func oneOf(value string, options ...string) bool {
	return slices.Contains(options, value)
}
