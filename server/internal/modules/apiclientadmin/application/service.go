package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/netip"
	"regexp"
	"slices"
	"strings"
	"time"

	clients "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}
type TokenIssuer interface {
	Issue(uuid.UUID, uuid.UUID, uuid.UUID, string, int32) (string, time.Time, error)
}
type Service struct {
	auth   Authenticator
	repo   clients.Repository
	issuer TokenIssuer
	clock  func() time.Time
}

func NewService(a Authenticator, r clients.Repository, i TokenIssuer) *Service {
	return &Service{auth: a, repo: r, issuer: i, clock: time.Now}
}
func (s *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	a, e := s.auth.Authenticate(ctx, token, "ak-admin")
	if e != nil {
		return a, e
	}
	if !slices.Contains(a.Permissions, permission) {
		return a, clients.ErrForbidden
	}
	return a, nil
}
func principal(a iamdomain.AuthenticatedContext, p clients.Principal) clients.Principal {
	p.TenantID = a.Tenant.ID
	p.UserID = a.User.ID
	p.SessionID = a.SessionID
	return p
}
func normalizeFilter(f clients.Filter) (clients.Filter, error) {
	f.Query = strings.TrimSpace(f.Query)
	f.Status = strings.TrimSpace(f.Status)
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || len(f.Query) > 160 || (f.Status != "" && f.Status != "active" && f.Status != "disabled") {
		return f, clients.ErrInvalid
	}
	return f, nil
}

var namePattern = regexp.MustCompile(`^[\pL\pN][\pL\pN _.-]{0,159}$`)

func normalizeInput(in clients.Input, now time.Time) (clients.Input, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Status == "" {
		in.Status = "active"
	}
	if !namePattern.MatchString(in.Name) || len([]rune(in.Description)) > 500 || (in.Status != "active" && in.Status != "disabled") || (in.ExpiresAt != nil && !in.ExpiresAt.After(now)) {
		return in, clients.ErrInvalid
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(in.AllowedCIDRs))
	if len(in.AllowedCIDRs) > 64 {
		return in, clients.ErrInvalid
	}
	for _, raw := range in.AllowedCIDRs {
		p, e := netip.ParsePrefix(strings.TrimSpace(raw))
		if e != nil {
			return in, clients.ErrInvalid
		}
		p = p.Masked()
		if !seen[p.String()] {
			seen[p.String()] = true
			out = append(out, p.String())
		}
	}
	slices.Sort(out)
	in.AllowedCIDRs = out
	return in, nil
}
func opaque(n int) (string, error) {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func (s *Service) List(ctx context.Context, t string, f clients.Filter) (clients.Page, error) {
	a, e := s.authorize(ctx, t, "sys.api_client.read")
	if e != nil {
		return clients.Page{}, e
	}
	f, e = normalizeFilter(f)
	if e != nil {
		return clients.Page{}, e
	}
	return s.repo.List(ctx, a.Tenant.ID, f)
}
func (s *Service) Get(ctx context.Context, t string, id uuid.UUID) (clients.Client, error) {
	a, e := s.authorize(ctx, t, "sys.api_client.read")
	if e != nil {
		return clients.Client{}, e
	}
	if id == uuid.Nil {
		return clients.Client{}, clients.ErrInvalid
	}
	return s.repo.Get(ctx, a.Tenant.ID, id)
}
func (s *Service) Create(ctx context.Context, t string, p clients.Principal, in clients.Input) (clients.Client, error) {
	a, e := s.authorize(ctx, t, "sys.api_client.create")
	if e != nil {
		return clients.Client{}, e
	}
	in, e = normalizeInput(in, s.clock().UTC())
	if e != nil || strings.TrimSpace(p.RequestID) == "" {
		if e == nil {
			e = clients.ErrInvalid
		}
		return clients.Client{}, e
	}
	raw, e := opaque(18)
	if e != nil {
		return clients.Client{}, e
	}
	return s.repo.Create(ctx, principal(a, p), "ak_"+raw, in)
}
func (s *Service) Update(ctx context.Context, t string, p clients.Principal, id uuid.UUID, in clients.Input) (clients.Client, error) {
	a, e := s.authorize(ctx, t, "sys.api_client.update")
	if e != nil {
		return clients.Client{}, e
	}
	in, e = normalizeInput(in, s.clock().UTC())
	if e != nil || id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" {
		if e == nil {
			e = clients.ErrInvalid
		}
		return clients.Client{}, e
	}
	return s.repo.Update(ctx, principal(a, p), id, in)
}
func (s *Service) CreateSecret(ctx context.Context, t string, p clients.Principal, id uuid.UUID, expires *time.Time) (clients.CreatedSecret, error) {
	a, e := s.authorize(ctx, t, "sys.api_client.rotate_secret")
	if e != nil {
		return clients.CreatedSecret{}, e
	}
	now := s.clock().UTC()
	if id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" || (expires != nil && !expires.After(now)) {
		return clients.CreatedSecret{}, clients.ErrInvalid
	}
	prefixRaw, e := opaque(6)
	if e != nil {
		return clients.CreatedSecret{}, e
	}
	body, e := opaque(32)
	if e != nil {
		return clients.CreatedSecret{}, e
	}
	prefix := "aks_" + prefixRaw
	plain := prefix + "_" + body
	digest := sha256.Sum256([]byte(plain))
	meta, e := s.repo.CreateSecret(ctx, principal(a, p), id, prefix, digest[:], expires)
	if e != nil {
		return clients.CreatedSecret{}, e
	}
	return clients.CreatedSecret{Secret: meta, Plaintext: plain}, nil
}
func (s *Service) RevokeSecret(ctx context.Context, t string, p clients.Principal, id, sid uuid.UUID) error {
	a, e := s.authorize(ctx, t, "sys.api_client.revoke_secret")
	if e != nil {
		return e
	}
	if id == uuid.Nil || sid == uuid.Nil || strings.TrimSpace(p.RequestID) == "" {
		return clients.ErrInvalid
	}
	return s.repo.RevokeSecret(ctx, principal(a, p), id, sid)
}
func (s *Service) Permissions(ctx context.Context, t string, p clients.Principal, id uuid.UUID, codes []string) (clients.Client, error) {
	a, e := s.authorize(ctx, t, "sys.api_client.assign_permission")
	if e != nil {
		return clients.Client{}, e
	}
	if id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" || len(codes) > 256 {
		return clients.Client{}, clients.ErrInvalid
	}
	for i := range codes {
		codes[i] = strings.TrimSpace(codes[i])
		if codes[i] == "" {
			return clients.Client{}, clients.ErrInvalid
		}
	}
	slices.Sort(codes)
	codes = slices.Compact(codes)
	return s.repo.ReplacePermissions(ctx, principal(a, p), id, codes)
}
func (s *Service) Token(ctx context.Context, clientID, secret, ip string) (string, time.Time, error) {
	clientID = strings.TrimSpace(clientID)
	secret = strings.TrimSpace(secret)
	if clientID == "" || len(secret) < 32 {
		return "", time.Time{}, clients.ErrCredential
	}
	digest := iamapp.HashOpaqueToken(secret)
	c, e := s.repo.Authenticate(ctx, clients.Credential{ClientID: clientID, SecretHash: digest, IPAddress: ip})
	if e != nil {
		return "", time.Time{}, e
	}
	return s.issuer.Issue(c.ID, c.ID, c.ID, "ak-api", 1)
}
