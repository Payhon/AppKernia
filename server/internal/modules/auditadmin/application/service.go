package application

import (
	"context"
	"strings"
	"time"

	auditdomain "github.com/appkernia/appkernia/server/internal/modules/auditadmin/domain"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

const adminAudience = "ak-admin"

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Service struct {
	auth       Authenticator
	repository auditdomain.Repository
	now        func() time.Time
}

func NewService(auth Authenticator, repository auditdomain.Repository) *Service {
	return &Service{auth: auth, repository: repository, now: time.Now}
}

func (s *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	a, err := s.auth.Authenticate(ctx, token, adminAudience)
	if err != nil {
		return iamdomain.AuthenticatedContext{}, err
	}
	for _, candidate := range a.Permissions {
		if candidate == permission {
			return a, nil
		}
	}
	return iamdomain.AuthenticatedContext{}, auditdomain.ErrForbidden
}

func (s *Service) normalizePage(f auditdomain.PageFilter) (auditdomain.PageFilter, error) {
	f.Query = strings.TrimSpace(f.Query)
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	now := s.now().UTC()
	if f.ToAt.IsZero() {
		f.ToAt = now
	}
	if f.FromAt.IsZero() {
		f.FromAt = f.ToAt.AddDate(0, 0, -30)
	}
	f.FromAt, f.ToAt = f.FromAt.UTC(), f.ToAt.UTC()
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || len(f.Query) > 160 || f.FromAt.After(f.ToAt) || f.ToAt.Sub(f.FromAt) > 180*24*time.Hour {
		return auditdomain.PageFilter{}, auditdomain.ErrInvalid
	}
	return f, nil
}

func (s *Service) Operations(ctx context.Context, token string, f auditdomain.OperationFilter) (auditdomain.Page[auditdomain.Operation], error) {
	a, err := s.authorize(ctx, token, "audit.operation.read")
	if err != nil {
		return auditdomain.Page[auditdomain.Operation]{}, err
	}
	f.PageFilter, err = s.normalizePage(f.PageFilter)
	f.ModuleCode, f.Result = strings.TrimSpace(f.ModuleCode), strings.TrimSpace(f.Result)
	if err != nil || !oneOf(f.Result, "", "success", "failure") || len(f.ModuleCode) > 64 {
		return auditdomain.Page[auditdomain.Operation]{}, auditdomain.ErrInvalid
	}
	return s.repository.ListOperations(ctx, a.Tenant.ID, f)
}

func (s *Service) Logins(ctx context.Context, token string, f auditdomain.LoginFilter) (auditdomain.Page[auditdomain.Login], error) {
	a, err := s.authorize(ctx, token, "audit.login.read")
	if err != nil {
		return auditdomain.Page[auditdomain.Login]{}, err
	}
	f.PageFilter, err = s.normalizePage(f.PageFilter)
	f.Result, f.Audience, f.AuthMethod = strings.TrimSpace(f.Result), strings.TrimSpace(f.Audience), strings.TrimSpace(f.AuthMethod)
	if err != nil || !oneOf(f.Result, "", "success", "failure", "blocked") || !oneOf(f.Audience, "", "ak-mobile", "ak-admin", "ak-api") || !oneOf(f.AuthMethod, "", "password", "email_otp", "sms_otp", "oauth", "refresh_token", "api_secret", "mfa", "tenant_switch") {
		return auditdomain.Page[auditdomain.Login]{}, auditdomain.ErrInvalid
	}
	return s.repository.ListLogins(ctx, a.Tenant.ID, f)
}

func (s *Service) SecurityEvents(ctx context.Context, token string, f auditdomain.SecurityFilter) (auditdomain.Page[auditdomain.SecurityEvent], error) {
	a, err := s.authorize(ctx, token, "audit.security.read")
	if err != nil {
		return auditdomain.Page[auditdomain.SecurityEvent]{}, err
	}
	f.PageFilter, err = s.normalizePage(f.PageFilter)
	f.Severity, f.Source, f.Status = strings.TrimSpace(f.Severity), strings.TrimSpace(f.Source), strings.TrimSpace(f.Status)
	if err != nil || !oneOf(f.Severity, "", "info", "low", "medium", "high", "critical") || !oneOf(f.Status, "", "open", "resolved") || len(f.Source) > 64 {
		return auditdomain.Page[auditdomain.SecurityEvent]{}, auditdomain.ErrInvalid
	}
	return s.repository.ListSecurityEvents(ctx, a.Tenant.ID, f)
}

func (s *Service) SecurityEvent(ctx context.Context, token string, id uuid.UUID) (auditdomain.SecurityEvent, error) {
	a, err := s.authorize(ctx, token, "audit.security.read")
	if err != nil {
		return auditdomain.SecurityEvent{}, err
	}
	if id == uuid.Nil {
		return auditdomain.SecurityEvent{}, auditdomain.ErrInvalid
	}
	return s.repository.GetSecurityEvent(ctx, a.Tenant.ID, id)
}

func (s *Service) ResolveSecurityEvent(ctx context.Context, token string, p auditdomain.Principal, id uuid.UUID) (auditdomain.SecurityEvent, error) {
	a, err := s.authorize(ctx, token, "audit.security.resolve")
	if err != nil {
		return auditdomain.SecurityEvent{}, err
	}
	if id == uuid.Nil {
		return auditdomain.SecurityEvent{}, auditdomain.ErrInvalid
	}
	p.TenantID, p.UserID, p.SessionID = a.Tenant.ID, a.User.ID, a.SessionID
	return s.repository.ResolveSecurityEvent(ctx, p, id)
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
