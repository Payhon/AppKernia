package application

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/netip"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	webhooks "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/domain"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}
type Service struct {
	auth    Authenticator
	repo    webhooks.Repository
	sealer  webhooks.Sealer
	adapter webhooks.Adapter
	clock   func() time.Time
}

func NewService(a Authenticator, r webhooks.Repository, s webhooks.Sealer, adapter webhooks.Adapter) *Service {
	return &Service{auth: a, repo: r, sealer: s, adapter: adapter, clock: time.Now}
}
func (s *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	a, err := s.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return a, err
	}
	if !slices.Contains(a.Permissions, permission) {
		return a, webhooks.ErrForbidden
	}
	return a, nil
}
func principal(a iamdomain.AuthenticatedContext, p webhooks.Principal) webhooks.Principal {
	p.TenantID, p.UserID, p.SessionID = a.Tenant.ID, a.User.ID, a.SessionID
	return p
}

var (
	namePattern  = regexp.MustCompile(`^[\pL\pN][\pL\pN _.-]{0,159}$`)
	eventPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,159}$`)
)

func normalizeInput(in webhooks.Input) (webhooks.Input, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.EndpointURL = strings.TrimSpace(in.EndpointURL)
	if in.Status == "" {
		in.Status = "active"
	}
	if in.MaxAttempts == 0 {
		in.MaxAttempts = 8
	}
	if in.TimeoutSeconds == 0 {
		in.TimeoutSeconds = 10
	}
	if !namePattern.MatchString(in.Name) || (in.Status != "active" && in.Status != "disabled") || in.MaxAttempts < 1 || in.MaxAttempts > 100 || in.TimeoutSeconds < 1 || in.TimeoutSeconds > 60 || validateEndpointURL(in.EndpointURL) != nil || len(in.EventTypes) == 0 || len(in.EventTypes) > 64 {
		return in, webhooks.ErrInvalid
	}
	seen := map[string]bool{}
	events := make([]string, 0, len(in.EventTypes))
	for _, raw := range in.EventTypes {
		v := strings.ToLower(strings.TrimSpace(raw))
		if !eventPattern.MatchString(v) {
			return in, webhooks.ErrInvalid
		}
		if !seen[v] {
			seen[v] = true
			events = append(events, v)
		}
	}
	slices.Sort(events)
	in.EventTypes = events
	return in, nil
}
func normalizeFilter(f webhooks.Filter) (webhooks.Filter, error) {
	f.Query, f.Status, f.EventType = strings.TrimSpace(f.Query), strings.TrimSpace(f.Status), strings.ToLower(strings.TrimSpace(f.EventType))
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || len(f.Query) > 160 || (f.Status != "" && f.Status != "active" && f.Status != "disabled") || (f.EventType != "" && !eventPattern.MatchString(f.EventType)) {
		return f, webhooks.ErrInvalid
	}
	return f, nil
}
func validateEndpointURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return webhooks.ErrInvalid
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "localhost" || !strings.Contains(host, ".") {
		return webhooks.ErrInvalid
	}
	if ip, err := netip.ParseAddr(host); err == nil && !isPublicIP(ip) {
		return webhooks.ErrInvalid
	}
	return nil
}
func isPublicIP(ip netip.Addr) bool {
	return ip.IsValid() && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}
func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "whsec_" + base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Service) List(ctx context.Context, token string, f webhooks.Filter) (webhooks.EndpointPage, error) {
	a, err := s.authorize(ctx, token, "sys.webhook.read")
	if err != nil {
		return webhooks.EndpointPage{}, err
	}
	f, err = normalizeFilter(f)
	if err != nil {
		return webhooks.EndpointPage{}, err
	}
	return s.repo.List(ctx, a.Tenant.ID, f)
}
func (s *Service) Create(ctx context.Context, token string, p webhooks.Principal, in webhooks.Input) (webhooks.CreatedEndpoint, error) {
	a, err := s.authorize(ctx, token, "sys.webhook.create")
	if err != nil {
		return webhooks.CreatedEndpoint{}, err
	}
	in, err = normalizeInput(in)
	if err != nil || strings.TrimSpace(p.RequestID) == "" {
		return webhooks.CreatedEndpoint{}, webhooks.ErrInvalid
	}
	secret, err := randomSecret()
	if err != nil {
		return webhooks.CreatedEndpoint{}, err
	}
	sealed, version, err := s.sealer.Seal([]byte(secret), a.Tenant.ID.String())
	if err != nil {
		return webhooks.CreatedEndpoint{}, err
	}
	endpoint, err := s.repo.Create(ctx, principal(a, p), in, sealed, version)
	if err != nil {
		return webhooks.CreatedEndpoint{}, err
	}
	return webhooks.CreatedEndpoint{Endpoint: endpoint, SigningSecret: secret}, nil
}
func (s *Service) Update(ctx context.Context, token string, p webhooks.Principal, id uuid.UUID, in webhooks.Input) (webhooks.Endpoint, error) {
	a, err := s.authorize(ctx, token, "sys.webhook.update")
	if err != nil {
		return webhooks.Endpoint{}, err
	}
	in, err = normalizeInput(in)
	if err != nil || id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" {
		return webhooks.Endpoint{}, webhooks.ErrInvalid
	}
	return s.repo.Update(ctx, principal(a, p), id, in)
}
func (s *Service) Test(ctx context.Context, token string, p webhooks.Principal, id uuid.UUID, idem string, in webhooks.TestInput) (webhooks.Delivery, error) {
	a, err := s.authorize(ctx, token, "sys.webhook.test")
	if err != nil {
		return webhooks.Delivery{}, err
	}
	idem, in.EventType = strings.TrimSpace(idem), strings.ToLower(strings.TrimSpace(in.EventType))
	if id == uuid.Nil || len(idem) < 16 || len(idem) > 128 || !eventPattern.MatchString(in.EventType) || in.Payload == nil || strings.TrimSpace(p.RequestID) == "" {
		return webhooks.Delivery{}, webhooks.ErrInvalid
	}
	stored, err := s.repo.GetStored(ctx, a.Tenant.ID, id)
	if err != nil {
		return webhooks.Delivery{}, err
	}
	if stored.Status != "active" || !slices.Contains(stored.EventTypes, in.EventType) {
		return webhooks.Delivery{}, webhooks.ErrInvalid
	}
	eventID := uuid.NewSHA1(id, []byte(idem))
	delivery, created, err := s.repo.CreateTestDelivery(ctx, principal(a, p), id, idem, eventID, in.EventType, in.Payload)
	if err != nil || !created {
		return delivery, err
	}
	secret, err := s.sealer.Open(stored.SecretCiphertext, a.Tenant.ID.String())
	if err != nil {
		_, _ = s.repo.CompleteDelivery(ctx, principal(a, p), id, delivery.ID, webhooks.DeliveryResult{}, webhooks.ErrDelivery)
		return webhooks.Delivery{}, err
	}
	now := s.clock().UTC()
	body, err := json.Marshal(map[string]any{"event_id": eventID, "event_type": in.EventType, "occurred_at": now, "payload": in.Payload})
	if err != nil {
		return webhooks.Delivery{}, err
	}
	timestamp := strconv.FormatInt(now.Unix(), 10)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(timestamp + "." + string(body)))
	headers := map[string]string{"Content-Type": "application/json", "User-Agent": "AppKernia-Webhooks/1.0", "X-AK-Webhook-Timestamp": timestamp, "X-AK-Webhook-Event-ID": eventID.String(), "X-AK-Webhook-Signature": "v1=" + hex.EncodeToString(mac.Sum(nil))}
	result, deliverErr := s.adapter.Deliver(ctx, stored.EndpointURL, headers, body, time.Duration(stored.TimeoutSeconds)*time.Second)
	return s.repo.CompleteDelivery(ctx, principal(a, p), id, delivery.ID, result, deliverErr)
}
func (s *Service) Deliveries(ctx context.Context, token string, endpointID uuid.UUID, page, pageSize int32) (webhooks.DeliveryPage, error) {
	a, err := s.authorize(ctx, token, "sys.webhook.delivery.read")
	if err != nil {
		return webhooks.DeliveryPage{}, err
	}
	if endpointID == uuid.Nil {
		return webhooks.DeliveryPage{}, webhooks.ErrInvalid
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return webhooks.DeliveryPage{}, webhooks.ErrInvalid
	}
	return s.repo.Deliveries(ctx, a.Tenant.ID, endpointID, page, pageSize)
}
