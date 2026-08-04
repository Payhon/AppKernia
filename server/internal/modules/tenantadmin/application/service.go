package application

import (
	"context"
	"regexp"
	"strings"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	tenantdomain "github.com/appkernia/appkernia/server/internal/modules/tenantadmin/domain"
	"github.com/google/uuid"
)

const adminAudience = "ak-admin"

var tenantCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,63}$`)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}
type Service struct {
	auth       Authenticator
	repository tenantdomain.Repository
	enabled    bool
}

func NewService(auth Authenticator, repository tenantdomain.Repository, enabled bool) *Service {
	return &Service{auth: auth, repository: repository, enabled: enabled}
}
func (s *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	if !s.enabled {
		return iamdomain.AuthenticatedContext{}, tenantdomain.ErrNotFound
	}
	a, e := s.auth.Authenticate(ctx, token, adminAudience)
	if e != nil {
		return iamdomain.AuthenticatedContext{}, e
	}
	for _, p := range a.Permissions {
		if p == permission {
			return a, nil
		}
	}
	return iamdomain.AuthenticatedContext{}, tenantdomain.ErrForbidden
}
func scope(a iamdomain.AuthenticatedContext, p tenantdomain.Principal) tenantdomain.Principal {
	p.TenantID = a.Tenant.ID
	p.UserID = a.User.ID
	p.SessionID = a.SessionID
	return p
}
func (s *Service) List(ctx context.Context, token string, f tenantdomain.Filters) (tenantdomain.Page, error) {
	a, e := s.authorize(ctx, token, "iam.tenant.read")
	if e != nil {
		return tenantdomain.Page{}, e
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 {
		return tenantdomain.Page{}, tenantdomain.ErrInvalid
	}
	if (f.Status != "" && f.Status != "active" && f.Status != "disabled") || (f.Sort != "" && f.Sort != "created_desc") {
		return tenantdomain.Page{}, tenantdomain.ErrInvalid
	}
	f.Query = strings.TrimSpace(f.Query)
	return s.repository.List(ctx, a.Tenant.ID, f)
}
func (s *Service) Get(ctx context.Context, token string, id uuid.UUID) (tenantdomain.Tenant, error) {
	a, e := s.authorize(ctx, token, "iam.tenant.read")
	if e != nil {
		return tenantdomain.Tenant{}, e
	}
	return s.repository.Get(ctx, a.Tenant.ID, id)
}
func (s *Service) Create(ctx context.Context, token string, p tenantdomain.Principal, in tenantdomain.CreateInput) (tenantdomain.Tenant, error) {
	a, e := s.authorize(ctx, token, "iam.tenant.create")
	if e != nil {
		return tenantdomain.Tenant{}, e
	}
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	if !tenantCodePattern.MatchString(in.Code) || len(in.Name) < 1 || len(in.Name) > 120 {
		return tenantdomain.Tenant{}, tenantdomain.ErrInvalid
	}
	return s.repository.Create(ctx, scope(a, p), in)
}
func (s *Service) Update(ctx context.Context, token string, p tenantdomain.Principal, id uuid.UUID, in tenantdomain.UpdateInput) (tenantdomain.Tenant, error) {
	a, e := s.authorize(ctx, token, "iam.tenant.update")
	if e != nil {
		return tenantdomain.Tenant{}, e
	}
	in.Name = strings.TrimSpace(in.Name)
	if id == uuid.Nil || len(in.Name) < 1 || len(in.Name) > 120 || (in.Status != "active" && in.Status != "disabled") {
		return tenantdomain.Tenant{}, tenantdomain.ErrInvalid
	}
	return s.repository.Update(ctx, scope(a, p), id, in)
}
func (s *Service) Members(ctx context.Context, token string, id uuid.UUID) ([]tenantdomain.Member, error) {
	a, e := s.authorize(ctx, token, "iam.tenant.member.read")
	if e != nil {
		return nil, e
	}
	return s.repository.Members(ctx, a.Tenant.ID, id)
}
func (s *Service) AddMember(ctx context.Context, token string, p tenantdomain.Principal, id uuid.UUID, in tenantdomain.AddMemberInput) (tenantdomain.Member, error) {
	a, e := s.authorize(ctx, token, "iam.tenant.member.invite")
	if e != nil {
		return tenantdomain.Member{}, e
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if id == uuid.Nil || !strings.Contains(in.Email, "@") || len(in.DisplayName) > 120 {
		return tenantdomain.Member{}, tenantdomain.ErrInvalid
	}
	return s.repository.AddMember(ctx, scope(a, p), id, in)
}
func (s *Service) SetMemberStatus(ctx context.Context, token string, p tenantdomain.Principal, tenantID, userID uuid.UUID, status string) (tenantdomain.Member, error) {
	perm := "iam.tenant.member.update"
	if status == "left" {
		perm = "iam.tenant.member.remove"
	}
	a, e := s.authorize(ctx, token, perm)
	if e != nil {
		return tenantdomain.Member{}, e
	}
	if status != "active" && status != "suspended" && status != "left" {
		return tenantdomain.Member{}, tenantdomain.ErrInvalid
	}
	return s.repository.SetMemberStatus(ctx, scope(a, p), tenantID, userID, status)
}
