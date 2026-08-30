package application

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"

	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	share "github.com/appkernia/appkernia/server/internal/modules/shareconfig/domain"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iam.AuthenticatedContext, error)
}

type Service struct {
	auth   Authenticator
	repo   share.Repository
	sealer share.SecretSealer
}

func NewService(auth Authenticator, repo share.Repository, sealer share.SecretSealer) *Service {
	return &Service{auth: auth, repo: repo, sealer: sealer}
}

func (s *Service) authorize(ctx context.Context, token, permission string) (iam.AuthenticatedContext, error) {
	auth, err := s.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return iam.AuthenticatedContext{}, err
	}
	for _, candidate := range auth.Permissions {
		if candidate == permission {
			return auth, nil
		}
	}
	return iam.AuthenticatedContext{}, share.ErrForbidden
}

func withPrincipal(auth iam.AuthenticatedContext, p share.Principal) share.Principal {
	p.TenantID, p.UserID, p.SessionID = auth.Tenant.ID, auth.User.ID, auth.SessionID
	return p
}

func normalizeFilter(f share.ListFilter) (share.ListFilter, error) {
	f.Query, f.ProviderCode, f.Status = strings.TrimSpace(f.Query), strings.TrimSpace(f.ProviderCode), strings.TrimSpace(f.Status)
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || len([]rune(f.Query)) > 160 ||
		!oneOf(f.ProviderCode, "", share.ProviderWechat) || !oneOf(f.Status, "", "draft", "active", "disabled") {
		return f, share.ErrInvalid
	}
	return f, nil
}

var (
	wechatAppIDPattern = regexp.MustCompile(`^wx[A-Za-z0-9]{16}$`)
	packagePattern     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z0-9_]+)+$`)
	signaturePattern   = regexp.MustCompile(`^[A-Fa-f0-9:]{16,256}$`)
)

func decodeWechat(raw json.RawMessage) (share.WechatPublicConfig, error) {
	var out share.WechatPublicConfig
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if len(raw) == 0 || decoder.Decode(&out) != nil {
		return out, share.ErrInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return out, share.ErrInvalid
	}
	return out, nil
}

func normalizeConfig(in share.ConfigInput, activation bool) (share.ConfigInput, share.WechatPublicConfig, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.ProviderCode = strings.ToLower(strings.TrimSpace(in.ProviderCode))
	in.ExternalAppID = strings.TrimSpace(in.ExternalAppID)
	if in.ConfigSchemaVersion == 0 {
		in.ConfigSchemaVersion = 1
	}
	if len([]rune(in.Name)) < 1 || len([]rune(in.Name)) > 160 || len([]rune(in.Description)) > 1000 ||
		in.ProviderCode != share.ProviderWechat || !wechatAppIDPattern.MatchString(in.ExternalAppID) || in.ConfigSchemaVersion != 1 {
		return in, share.WechatPublicConfig{}, share.ErrInvalid
	}
	config, err := decodeWechat(in.PublicConfig)
	if err != nil {
		return in, config, err
	}
	normalizePlatform := func(platform *share.PlatformIdentity) {
		platform.PackageName = strings.TrimSpace(platform.PackageName)
		platform.Signature = strings.TrimSpace(platform.Signature)
		platform.BundleID = strings.TrimSpace(platform.BundleID)
		platform.UniversalLink = strings.TrimSpace(platform.UniversalLink)
		platform.BundleName = strings.TrimSpace(platform.BundleName)
	}
	normalizePlatform(&config.Android)
	normalizePlatform(&config.IOS)
	normalizePlatform(&config.Harmony)
	if err = validateWechat(config, activation); err != nil {
		return in, config, err
	}
	in.PublicConfig, err = json.Marshal(config)
	if err != nil {
		return in, config, share.ErrInvalid
	}
	return in, config, nil
}

func validateWechat(config share.WechatPublicConfig, activation bool) error {
	if !activation {
		return nil
	}
	if !config.Android.Enabled && !config.IOS.Enabled && !config.Harmony.Enabled {
		return share.ErrInvalid
	}
	if config.Android.Enabled && (!packagePattern.MatchString(config.Android.PackageName) || !signaturePattern.MatchString(config.Android.Signature)) {
		return share.ErrInvalid
	}
	if config.IOS.Enabled && (!packagePattern.MatchString(config.IOS.BundleID) || !validHTTPSOrigin(config.IOS.UniversalLink, true)) {
		return share.ErrInvalid
	}
	if config.Harmony.Enabled && !packagePattern.MatchString(config.Harmony.BundleName) {
		return share.ErrInvalid
	}
	return nil
}

func validHTTPSOrigin(raw string, allowPath bool) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (!allowPath && u.Path != "" && u.Path != "/") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast()) {
		return false
	}
	return true
}

func oneOf(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}

func (s *Service) List(ctx context.Context, token string, f share.ListFilter) (share.ConfigPage, error) {
	auth, err := s.authorize(ctx, token, "sys.share_config.read")
	if err != nil {
		return share.ConfigPage{}, err
	}
	f, err = normalizeFilter(f)
	if err != nil {
		return share.ConfigPage{}, err
	}
	return s.repo.List(ctx, auth.Tenant.ID, f)
}

func (s *Service) Get(ctx context.Context, token string, id uuid.UUID) (share.Config, error) {
	auth, err := s.authorize(ctx, token, "sys.share_config.read")
	if err != nil {
		return share.Config{}, err
	}
	if id == uuid.Nil {
		return share.Config{}, share.ErrInvalid
	}
	return s.repo.Get(ctx, auth.Tenant.ID, id)
}

func (s *Service) Create(ctx context.Context, token string, p share.Principal, in share.ConfigInput) (share.Config, error) {
	auth, err := s.authorize(ctx, token, "sys.share_config.create")
	if err != nil {
		return share.Config{}, err
	}
	in, _, err = normalizeConfig(in, false)
	if err != nil || in.LockVersion != 0 {
		return share.Config{}, share.ErrInvalid
	}
	return s.repo.Create(ctx, withPrincipal(auth, p), in)
}

func (s *Service) Update(ctx context.Context, token string, p share.Principal, id uuid.UUID, in share.ConfigInput) (share.Config, error) {
	auth, err := s.authorize(ctx, token, "sys.share_config.update")
	if err != nil {
		return share.Config{}, err
	}
	current, err := s.repo.Get(ctx, auth.Tenant.ID, id)
	if err != nil {
		return share.Config{}, err
	}
	in, _, err = normalizeConfig(in, current.Status == "active")
	if err != nil || in.LockVersion < 1 || in.ProviderCode != current.ProviderCode {
		return share.Config{}, share.ErrInvalid
	}
	return s.repo.Update(ctx, withPrincipal(auth, p), id, in)
}

func (s *Service) Activate(ctx context.Context, token string, p share.Principal, id uuid.UUID, lockVersion int32) (share.Config, error) {
	auth, err := s.authorize(ctx, token, "sys.share_config.update")
	if err != nil {
		return share.Config{}, err
	}
	current, err := s.repo.Get(ctx, auth.Tenant.ID, id)
	if err != nil {
		return share.Config{}, err
	}
	_, _, err = normalizeConfig(share.ConfigInput{Name: current.Name, Description: current.Description, ProviderCode: current.ProviderCode, ExternalAppID: current.ExternalAppID, ConfigSchemaVersion: current.ConfigSchemaVersion, PublicConfig: current.PublicConfig, LockVersion: lockVersion}, true)
	if err != nil || lockVersion < 1 || current.Status == "active" {
		return share.Config{}, share.ErrInvalid
	}
	return s.repo.SetStatus(ctx, withPrincipal(auth, p), id, lockVersion, "active")
}

func (s *Service) Disable(ctx context.Context, token string, p share.Principal, id uuid.UUID, lockVersion int32) (share.Config, error) {
	auth, err := s.authorize(ctx, token, "sys.share_config.update")
	if err != nil {
		return share.Config{}, err
	}
	if lockVersion < 1 {
		return share.Config{}, share.ErrInvalid
	}
	return s.repo.SetStatus(ctx, withPrincipal(auth, p), id, lockVersion, "disabled")
}

func (s *Service) Delete(ctx context.Context, token string, p share.Principal, id uuid.UUID, lockVersion int32) error {
	auth, err := s.authorize(ctx, token, "sys.share_config.delete")
	if err != nil {
		return err
	}
	if id == uuid.Nil || lockVersion < 1 {
		return share.ErrInvalid
	}
	return s.repo.Delete(ctx, withPrincipal(auth, p), id, lockVersion)
}

func (s *Service) RotateSecret(ctx context.Context, token string, p share.Principal, id uuid.UUID, in share.SecretInput) (share.Config, error) {
	auth, err := s.authorize(ctx, token, "sys.share_config.rotate_secret")
	if err != nil {
		return share.Config{}, err
	}
	current, err := s.repo.Get(ctx, auth.Tenant.ID, id)
	if err != nil {
		return share.Config{}, err
	}
	// The registered WeChat share provider intentionally has no AppSecret. This
	// endpoint is retained for future code-registered providers only.
	if current.ProviderCode == share.ProviderWechat || len(in.Values) == 0 || in.LockVersion < 1 || s.sealer == nil {
		return share.Config{}, share.ErrInvalid
	}
	keys := make([]string, 0, len(in.Values))
	clean := make(map[string]string, len(in.Values))
	for key, value := range in.Values {
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key == "" || value == "" || len(key) > 64 || len(value) > 4096 {
			return share.Config{}, share.ErrInvalid
		}
		keys = append(keys, key)
		clean[key] = value
	}
	sort.Strings(keys)
	plaintext, err := json.Marshal(clean)
	if err != nil {
		return share.Config{}, share.ErrInvalid
	}
	ciphertext, keyVersion, err := s.sealer.Seal(plaintext, "share-config:"+auth.Tenant.ID.String()+":"+id.String())
	if err != nil {
		return share.Config{}, err
	}
	return s.repo.RotateSecret(ctx, withPrincipal(auth, p), id, in.LockVersion, ciphertext, keyVersion, keys)
}

func normalizeBinding(provider string, in share.BindingInput) (share.BindingInput, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	in.ShareOrigin = strings.TrimRight(strings.TrimSpace(in.ShareOrigin), "/")
	in.FallbackMode = strings.ToLower(strings.TrimSpace(in.FallbackMode))
	if in.FallbackMode == "" {
		in.FallbackMode = "system"
	}
	if provider != share.ProviderWechat || in.ShareConfigID == uuid.Nil || !validHTTPSOrigin(in.ShareOrigin, false) || in.FallbackMode != "system" || len(in.Scenes) < 1 || len(in.Scenes) > 3 {
		return in, share.ErrInvalid
	}
	seen := map[string]bool{}
	for index, scene := range in.Scenes {
		scene = strings.ToLower(strings.TrimSpace(scene))
		if !oneOf(scene, "session", "timeline", "favorite") || seen[scene] {
			return in, share.ErrInvalid
		}
		seen[scene], in.Scenes[index] = true, scene
	}
	sort.Strings(in.Scenes)
	return in, nil
}

func preflightFor(config share.Config, binding share.BindingInput) share.Preflight {
	out := share.Preflight{ProviderCode: config.ProviderCode, Scenes: append([]string(nil), binding.Scenes...), Issues: []string{}, Platforms: []string{}}
	if config.Status != "active" {
		out.Issues = append(out.Issues, "share.config.not_active")
	}
	wx, err := decodeWechat(config.PublicConfig)
	if err != nil {
		out.Issues = append(out.Issues, "share.config.invalid")
	} else {
		if wx.Android.Enabled {
			out.Platforms = append(out.Platforms, "android")
		}
		if wx.IOS.Enabled {
			out.Platforms = append(out.Platforms, "ios")
		}
		if wx.Harmony.Enabled {
			out.Platforms = append(out.Platforms, "harmony")
		}
		if validateWechat(wx, true) != nil {
			out.Issues = append(out.Issues, "share.config.platform_incomplete")
		}
	}
	if !validHTTPSOrigin(binding.ShareOrigin, false) {
		out.Issues = append(out.Issues, "share.origin.invalid")
	}
	out.Ready = len(out.Issues) == 0
	return out
}

func (s *Service) ListBindings(ctx context.Context, token string, appID uuid.UUID) ([]share.Binding, error) {
	auth, err := s.authorize(ctx, token, "app.share_binding.read")
	if err != nil {
		return nil, err
	}
	if appID == uuid.Nil {
		return nil, share.ErrInvalid
	}
	exists, err := s.repo.AppExists(ctx, auth.Tenant.ID, appID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, share.ErrNotFound
	}
	return s.repo.ListBindings(ctx, auth.Tenant.ID, appID)
}

func (s *Service) Preflight(ctx context.Context, token string, appID uuid.UUID, provider string, in share.BindingInput) (share.Preflight, error) {
	auth, err := s.authorize(ctx, token, "app.share_binding.read")
	if err != nil {
		return share.Preflight{}, err
	}
	if appID == uuid.Nil {
		return share.Preflight{}, share.ErrInvalid
	}
	exists, err := s.repo.AppExists(ctx, auth.Tenant.ID, appID)
	if err != nil {
		return share.Preflight{}, err
	}
	if !exists {
		return share.Preflight{}, share.ErrNotFound
	}
	in, err = normalizeBinding(provider, in)
	if err != nil {
		return share.Preflight{}, err
	}
	config, err := s.repo.Get(ctx, auth.Tenant.ID, in.ShareConfigID)
	if err != nil {
		return share.Preflight{}, err
	}
	if config.ProviderCode != provider {
		return share.Preflight{}, share.ErrInvalid
	}
	return preflightFor(config, in), nil
}

func (s *Service) UpsertBinding(ctx context.Context, token string, p share.Principal, appID uuid.UUID, provider string, in share.BindingInput) (share.Binding, error) {
	auth, err := s.authorize(ctx, token, "app.share_binding.update")
	if err != nil {
		return share.Binding{}, err
	}
	in, err = normalizeBinding(provider, in)
	if err != nil || appID == uuid.Nil {
		return share.Binding{}, share.ErrInvalid
	}
	config, err := s.repo.Get(ctx, auth.Tenant.ID, in.ShareConfigID)
	if err != nil {
		return share.Binding{}, err
	}
	check := preflightFor(config, in)
	if config.ProviderCode != provider || !check.Ready {
		return share.Binding{}, share.ErrInvalid
	}
	return s.repo.UpsertBinding(ctx, withPrincipal(auth, p), appID, provider, in)
}

func (s *Service) DeleteBinding(ctx context.Context, token string, p share.Principal, appID uuid.UUID, provider string, lockVersion int32) error {
	auth, err := s.authorize(ctx, token, "app.share_binding.update")
	if err != nil {
		return err
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if appID == uuid.Nil || provider != share.ProviderWechat || lockVersion < 1 {
		return share.ErrInvalid
	}
	return s.repo.DeleteBinding(ctx, withPrincipal(auth, p), appID, provider, lockVersion)
}

func (s *Service) Runtime(ctx context.Context, appID uuid.UUID) ([]share.RuntimeProvider, error) {
	if appID == uuid.Nil {
		return nil, share.ErrInvalid
	}
	return s.repo.Runtime(ctx, appID)
}
