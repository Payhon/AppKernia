package application

import (
	"context"
	"slices"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	ops "github.com/appkernia/appkernia/server/internal/modules/opsadmin/domain"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}
type Service struct {
	auth    Authenticator
	repo    ops.Repository
	started time.Time
	now     func() time.Time
}

func NewService(a Authenticator, r ops.Repository) *Service {
	return &Service{auth: a, repo: r, started: time.Now().UTC(), now: time.Now}
}
func (s *Service) authorize(ctx context.Context, token string) (iamdomain.AuthenticatedContext, error) {
	a, e := s.auth.Authenticate(ctx, token, "ak-admin")
	if e != nil {
		return a, e
	}
	if !slices.Contains(a.Permissions, "ops.health.read") {
		return a, ops.ErrForbidden
	}
	return a, nil
}
func (s *Service) Health(ctx context.Context, token string) (ops.Health, error) {
	if _, e := s.authorize(ctx, token); e != nil {
		return ops.Health{}, e
	}
	return s.repo.Health(ctx), nil
}
func (s *Service) Runtime(ctx context.Context, token string) (ops.RuntimeSummary, error) {
	a, e := s.authorize(ctx, token)
	if e != nil {
		return ops.RuntimeSummary{}, e
	}
	return s.repo.Runtime(ctx, a.Tenant.ID, s.started)
}
