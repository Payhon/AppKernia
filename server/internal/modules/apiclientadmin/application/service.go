package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
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
type TokenVerifier interface {
	Verify(string, string) (iamapp.AccessClaims, error)
}

type DelegatedIdentityResolver interface {
	ResolveDelegatedContext(context.Context, uuid.UUID, uuid.UUID) (iamdomain.AuthenticatedContext, error)
}

type machineRepository interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (clients.Client, error)
	AuditAgentAuthentication(context.Context, clients.AgentAudit) error
}

type AgentCall struct {
	OperationID string
	Method      string
	Path        string
	RequestID   string
	IPAddress   string
	UserAgent   string
	AppID       *uuid.UUID
}

type agentCallContextKey struct{}

func WithAgentCall(ctx context.Context, call AgentCall) context.Context {
	return context.WithValue(ctx, agentCallContextKey{}, call)
}

func agentCallFromContext(ctx context.Context) (AgentCall, bool) {
	call, ok := ctx.Value(agentCallContextKey{}).(AgentCall)
	return call, ok
}

var agentCallableOperations = map[string]struct{}{
	"getAdminDashboardSummary":         {},
	"getAdminDashboardTrends":          {},
	"getAdminDashboardActivity":        {},
	"listAdminAppContentCategories":    {},
	"createAdminAppContentCategory":    {},
	"getAdminAppContentCategory":       {},
	"updateAdminAppContentCategory":    {},
	"deleteAdminAppContentCategory":    {},
	"listAdminAppContentArticles":      {},
	"createAdminAppContentArticle":     {},
	"getAdminAppContentArticle":        {},
	"updateAdminAppContentArticle":     {},
	"deleteAdminAppContentArticle":     {},
	"transitionAdminAppContentArticle": {},
}

func IsAgentCallable(operationID string) bool {
	_, ok := agentCallableOperations[operationID]
	return ok
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
func (s *Service) Applications(ctx context.Context, t string, p clients.Principal, id uuid.UUID, appIDs []uuid.UUID) (clients.Client, error) {
	a, e := s.authorize(ctx, t, "sys.api_client.update")
	if e != nil {
		return clients.Client{}, e
	}
	if id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" || len(appIDs) > 256 {
		return clients.Client{}, clients.ErrInvalid
	}
	for _, appID := range appIDs {
		if appID == uuid.Nil {
			return clients.Client{}, clients.ErrInvalid
		}
	}
	slices.SortFunc(appIDs, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	appIDs = slices.Compact(appIDs)
	return s.repo.ReplaceApps(ctx, principal(a, p), id, appIDs)
}
func (s *Service) Token(ctx context.Context, clientID, secret string, metadata clients.TokenMetadata) (string, time.Time, error) {
	clientID = strings.TrimSpace(clientID)
	secret = strings.TrimSpace(secret)
	identifierHash := sha256.Sum256([]byte(clientID))
	audit := clients.TokenExchangeAudit{
		ClientID: clientID, IdentifierHash: identifierHash[:], RequestID: strings.TrimSpace(metadata.RequestID),
		IPAddress: strings.TrimSpace(metadata.IPAddress), UserAgent: strings.TrimSpace(metadata.UserAgent),
	}
	if clientID == "" || len(secret) < 32 {
		audit.Result, audit.FailureReason = "failure", "invalid_credentials"
		_ = s.repo.AuditTokenExchange(ctx, audit)
		return "", time.Time{}, clients.ErrCredential
	}
	digest := iamapp.HashOpaqueToken(secret)
	c, e := s.repo.Authenticate(ctx, clients.Credential{ClientID: clientID, SecretHash: digest, IPAddress: audit.IPAddress})
	if e != nil {
		if errors.Is(e, clients.ErrCredential) {
			audit.Result, audit.FailureReason = "failure", "invalid_credentials"
			_ = s.repo.AuditTokenExchange(ctx, audit)
		}
		return "", time.Time{}, e
	}
	accessToken, expiresAt, e := s.issuer.Issue(c.ID, c.TenantID, c.ID, "ak-api", 1)
	if e != nil {
		return "", time.Time{}, e
	}
	audit.TenantID, audit.Result = &c.TenantID, "success"
	if e = s.repo.AuditTokenExchange(ctx, audit); e != nil {
		return "", time.Time{}, e
	}
	return accessToken, expiresAt, nil
}

type MachineAuthenticator struct {
	repo     machineRepository
	verifier TokenVerifier
	resolver DelegatedIdentityResolver
	clock    func() time.Time
}

func NewMachineAuthenticator(repo machineRepository, verifier TokenVerifier, resolver ...DelegatedIdentityResolver) *MachineAuthenticator {
	authenticator := &MachineAuthenticator{repo: repo, verifier: verifier, clock: time.Now}
	if len(resolver) > 0 {
		authenticator.resolver = resolver[0]
	}
	return authenticator
}

func (a *MachineAuthenticator) Authenticate(ctx context.Context, rawToken string, appID uuid.UUID, permission, ipAddress string) (clients.MachinePrincipal, error) {
	if strings.TrimSpace(rawToken) == "" || appID == uuid.Nil || strings.TrimSpace(permission) == "" {
		return clients.MachinePrincipal{}, clients.ErrCredential
	}
	claims, err := a.verifier.Verify(rawToken, "ak-api")
	if err != nil {
		return clients.MachinePrincipal{}, clients.ErrCredential
	}
	clientID, err := uuid.Parse(claims.Subject)
	if err != nil || clientID == uuid.Nil || claims.TenantID == uuid.Nil || claims.SessionID != clientID {
		return clients.MachinePrincipal{}, clients.ErrCredential
	}
	client, err := a.repo.Get(ctx, claims.TenantID, clientID)
	if err != nil || client.Status != "active" || (client.ExpiresAt != nil && !client.ExpiresAt.After(a.clock().UTC())) {
		return clients.MachinePrincipal{}, clients.ErrCredential
	}
	if !slices.Contains(client.Permissions, permission) || !slices.Contains(client.AppIDs, appID) {
		return clients.MachinePrincipal{}, clients.ErrForbidden
	}
	if !allowsIP(client.AllowedCIDRs, ipAddress) {
		return clients.MachinePrincipal{}, clients.ErrCredential
	}
	return clients.MachinePrincipal{TenantID: claims.TenantID, ClientID: clientID, AppID: appID, Permissions: client.Permissions}, nil
}

func (a *MachineAuthenticator) AuthenticateDelegated(ctx context.Context, rawToken string, call AgentCall) (iamdomain.AuthenticatedContext, error) {
	if strings.TrimSpace(rawToken) == "" || !IsAgentCallable(call.OperationID) || a.resolver == nil {
		return iamdomain.AuthenticatedContext{}, clients.ErrCredential
	}
	claims, err := a.verifier.Verify(rawToken, "ak-api")
	if err != nil {
		return iamdomain.AuthenticatedContext{}, clients.ErrCredential
	}
	clientID, err := uuid.Parse(claims.Subject)
	if err != nil || clientID == uuid.Nil || claims.TenantID == uuid.Nil || claims.SessionID != clientID {
		return iamdomain.AuthenticatedContext{}, clients.ErrCredential
	}
	client, err := a.repo.Get(ctx, claims.TenantID, clientID)
	if err != nil || client.Status != "active" || client.BoundUserID == nil || *client.BoundUserID == uuid.Nil || (client.ExpiresAt != nil && !client.ExpiresAt.After(a.clock().UTC())) {
		return iamdomain.AuthenticatedContext{}, clients.ErrCredential
	}
	if !allowsIP(client.AllowedCIDRs, call.IPAddress) {
		return iamdomain.AuthenticatedContext{}, clients.ErrCredential
	}
	if call.AppID != nil && !slices.Contains(client.AppIDs, *call.AppID) {
		return iamdomain.AuthenticatedContext{}, clients.ErrForbidden
	}
	authenticated, err := a.resolver.ResolveDelegatedContext(ctx, *client.BoundUserID, claims.TenantID)
	if err != nil {
		return iamdomain.AuthenticatedContext{}, err
	}
	clientPermissions := make(map[string]struct{}, len(client.Permissions))
	for _, permission := range client.Permissions {
		clientPermissions[strings.ToLower(permission)] = struct{}{}
	}
	effective := make([]string, 0, len(authenticated.Permissions))
	for _, permission := range authenticated.Permissions {
		if _, ok := clientPermissions[strings.ToLower(permission)]; ok {
			effective = append(effective, permission)
		}
	}
	authenticated.Permissions = effective
	authenticated.SessionID = uuid.Nil
	authenticated.APIClientID = &clientID
	method := strings.ToUpper(strings.TrimSpace(call.Method))
	if err = a.repo.AuditAgentAuthentication(ctx, clients.AgentAudit{
		TenantID: claims.TenantID, UserID: *client.BoundUserID, ClientID: clientID,
		RequestID: call.RequestID, Operation: call.OperationID, Method: method,
		Path: call.Path, IPAddress: call.IPAddress, UserAgent: call.UserAgent,
	}); err != nil {
		return iamdomain.AuthenticatedContext{}, err
	}
	return authenticated, nil
}

type DelegatedAuthenticator struct {
	user    Authenticator
	machine *MachineAuthenticator
}

func NewDelegatedAuthenticator(user Authenticator, machine *MachineAuthenticator) *DelegatedAuthenticator {
	return &DelegatedAuthenticator{user: user, machine: machine}
}

func (a *DelegatedAuthenticator) Authenticate(ctx context.Context, rawToken, audience string) (iamdomain.AuthenticatedContext, error) {
	authenticated, userErr := a.user.Authenticate(ctx, rawToken, audience)
	if userErr == nil || audience != "ak-admin" {
		return authenticated, userErr
	}
	call, ok := agentCallFromContext(ctx)
	if !ok {
		return iamdomain.AuthenticatedContext{}, userErr
	}
	authenticated, err := a.machine.AuthenticateDelegated(ctx, rawToken, call)
	switch {
	case err == nil:
		return authenticated, nil
	case errors.Is(err, clients.ErrForbidden):
		return iamdomain.AuthenticatedContext{}, iamapp.ErrAccessDenied
	case errors.Is(err, clients.ErrCredential), errors.Is(err, clients.ErrNotFound):
		return iamdomain.AuthenticatedContext{}, iamapp.ErrInvalidAccessToken
	default:
		return iamdomain.AuthenticatedContext{}, err
	}
}

func allowsIP(cidrs []string, rawIP string) bool {
	if len(cidrs) == 0 {
		return true
	}
	ip, err := netip.ParseAddr(strings.TrimSpace(rawIP))
	if err != nil {
		return false
	}
	for _, raw := range cidrs {
		prefix, err := netip.ParsePrefix(raw)
		if err == nil && prefix.Contains(ip) {
			return true
		}
	}
	return false
}
