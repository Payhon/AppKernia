package application

import (
	"context"
	"net/netip"
	"slices"
	"strings"
	"time"

	blocks "github.com/appkernia/appkernia/server/internal/modules/blockruleadmin/domain"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}
type Service struct {
	auth Authenticator
	repo blocks.Repository
	now  func() time.Time
}

func NewService(a Authenticator, r blocks.Repository) *Service {
	return &Service{auth: a, repo: r, now: time.Now}
}
func (s *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	a, err := s.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return a, err
	}
	if !slices.Contains(a.Permissions, permission) {
		return a, blocks.ErrForbidden
	}
	return a, nil
}
func principal(a iamdomain.AuthenticatedContext, p blocks.Principal) blocks.Principal {
	p.TenantID, p.UserID, p.SessionID = a.Tenant.ID, a.User.ID, a.SessionID
	return p
}
func validSubject(kind, value string) bool {
	switch kind {
	case "ip":
		_, e := netip.ParseAddr(value)
		return e == nil
	case "cidr":
		_, e := netip.ParsePrefix(value)
		return e == nil
	case "user":
		_, e := uuid.Parse(value)
		return e == nil
	case "device", "identifier":
		return len(value) >= 4 && len(value) <= 512
	}
	return false
}
func validSubjectType(v string) bool {
	return v == "ip" || v == "cidr" || v == "user" || v == "device" || v == "identifier"
}
func validAction(v string) bool { return v == "deny" || v == "challenge" || v == "rate_limit" }
func validStatus(v string) bool { return v == "active" || v == "disabled" }
func (s *Service) List(ctx context.Context, token string, f blocks.Filter) (blocks.Page, error) {
	a, e := s.authorize(ctx, token, "iam.block_rule.read")
	if e != nil {
		return blocks.Page{}, e
	}
	f.SubjectType = strings.TrimSpace(f.SubjectType)
	f.SubjectHint = strings.TrimSpace(f.SubjectHint)
	f.Scope = strings.TrimSpace(f.Scope)
	f.Status = strings.TrimSpace(f.Status)
	f.Expiry = strings.TrimSpace(f.Expiry)
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || len(f.SubjectHint) > 160 || (f.SubjectType != "" && !validSubjectType(f.SubjectType)) || (f.Scope != "" && f.Scope != "tenant") || (f.Status != "" && !validStatus(f.Status)) || (f.Expiry != "" && f.Expiry != "active" && f.Expiry != "expired" && f.Expiry != "never") {
		return blocks.Page{}, blocks.ErrInvalid
	}
	return s.repo.List(ctx, a.Tenant.ID, f)
}
func (s *Service) Create(ctx context.Context, token string, p blocks.Principal, in blocks.CreateInput) (blocks.Rule, error) {
	a, e := s.authorize(ctx, token, "iam.block_rule.create")
	if e != nil {
		return blocks.Rule{}, e
	}
	in.SubjectType = strings.TrimSpace(in.SubjectType)
	in.SubjectValue = strings.TrimSpace(in.SubjectValue)
	in.Action = strings.TrimSpace(in.Action)
	in.Reason = strings.TrimSpace(in.Reason)
	in.Status = strings.TrimSpace(in.Status)
	if in.Action == "" {
		in.Action = "deny"
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.StartsAt == nil {
		v := s.now().UTC()
		in.StartsAt = &v
	} else {
		v := in.StartsAt.UTC()
		in.StartsAt = &v
	}
	if in.ExpiresAt != nil {
		v := in.ExpiresAt.UTC()
		in.ExpiresAt = &v
	}
	if !validSubject(in.SubjectType, in.SubjectValue) || !validAction(in.Action) || !validStatus(in.Status) || len(in.Reason) > 500 || in.ExpiresAt != nil && !in.ExpiresAt.After(*in.StartsAt) || strings.TrimSpace(p.RequestID) == "" {
		return blocks.Rule{}, blocks.ErrInvalid
	}
	return s.repo.Create(ctx, principal(a, p), in)
}
func (s *Service) Update(ctx context.Context, token string, p blocks.Principal, id uuid.UUID, in blocks.UpdateInput) (blocks.Rule, error) {
	a, e := s.authorize(ctx, token, "iam.block_rule.update")
	if e != nil {
		return blocks.Rule{}, e
	}
	in.Action = strings.TrimSpace(in.Action)
	in.Reason = strings.TrimSpace(in.Reason)
	in.Status = strings.TrimSpace(in.Status)
	in.StartsAt = in.StartsAt.UTC()
	if in.ExpiresAt != nil {
		v := in.ExpiresAt.UTC()
		in.ExpiresAt = &v
	}
	if id == uuid.Nil || !validAction(in.Action) || !validStatus(in.Status) || in.StartsAt.IsZero() || len(in.Reason) > 500 || in.ExpiresAt != nil && !in.ExpiresAt.After(in.StartsAt) || strings.TrimSpace(p.RequestID) == "" {
		return blocks.Rule{}, blocks.ErrInvalid
	}
	return s.repo.Update(ctx, principal(a, p), id, in)
}
func (s *Service) Revoke(ctx context.Context, token string, p blocks.Principal, id uuid.UUID) (blocks.RevokeResult, error) {
	a, e := s.authorize(ctx, token, "iam.block_rule.delete")
	if e != nil {
		return blocks.RevokeResult{}, e
	}
	if id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" {
		return blocks.RevokeResult{}, blocks.ErrInvalid
	}
	return s.repo.Revoke(ctx, principal(a, p), id)
}
