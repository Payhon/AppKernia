package application

import (
	"context"
	"strings"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	sessiondomain "github.com/appkernia/appkernia/server/internal/modules/sessionadmin/domain"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Service struct {
	auth Authenticator
	repo sessiondomain.Repository
	now  func() time.Time
}

func NewService(auth Authenticator, repo sessiondomain.Repository) *Service {
	return &Service{auth: auth, repo: repo, now: time.Now}
}

func (s *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	auth, err := s.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return iamdomain.AuthenticatedContext{}, err
	}
	for _, candidate := range auth.Permissions {
		if candidate == permission {
			return auth, nil
		}
	}
	return iamdomain.AuthenticatedContext{}, sessiondomain.ErrForbidden
}

func (s *Service) List(ctx context.Context, token string, filter sessiondomain.Filter) (sessiondomain.Page, error) {
	auth, err := s.authorize(ctx, token, "iam.session.read")
	if err != nil {
		return sessiondomain.Page{}, err
	}
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Audience = strings.TrimSpace(filter.Audience)
	filter.Platform = strings.TrimSpace(filter.Platform)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.IP = strings.TrimSpace(filter.IP)
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	if filter.ToAt.IsZero() {
		filter.ToAt = s.now().UTC()
	}
	if filter.FromAt.IsZero() {
		filter.FromAt = filter.ToAt.AddDate(0, 0, -30)
	}
	filter.FromAt, filter.ToAt = filter.FromAt.UTC(), filter.ToAt.UTC()
	if filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 || len(filter.Query) > 160 || len(filter.IP) > 64 || filter.FromAt.After(filter.ToAt) || filter.ToAt.Sub(filter.FromAt) > 180*24*time.Hour || !oneOf(filter.Audience, "", "ak-mobile", "ak-admin", "ak-api") || !oneOf(filter.Platform, "", "android", "ios", "harmonyos", "web", "desktop", "unknown") || !oneOf(filter.Status, "", "active", "revoked", "expired") {
		return sessiondomain.Page{}, sessiondomain.ErrInvalid
	}
	return s.repo.List(ctx, auth.Tenant.ID, auth.SessionID, filter)
}

func (s *Service) Revoke(ctx context.Context, token string, principal sessiondomain.Principal, id uuid.UUID) (sessiondomain.RevokeResult, error) {
	auth, err := s.authorize(ctx, token, "iam.session.revoke")
	if err != nil {
		return sessiondomain.RevokeResult{}, err
	}
	if id == uuid.Nil {
		return sessiondomain.RevokeResult{}, sessiondomain.ErrInvalid
	}
	principal.TenantID, principal.UserID, principal.SessionID = auth.Tenant.ID, auth.User.ID, auth.SessionID
	return s.repo.Revoke(ctx, principal, id)
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
