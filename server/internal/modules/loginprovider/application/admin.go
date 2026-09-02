package application

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"sort"
	"strings"
	"time"

	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func validAppleCredential(raw string, public login.ApplePublicConfig, clientID string, now time.Time) bool {
	block, _ := pem.Decode([]byte(raw))
	if block == nil || block.Type != "PRIVATE KEY" {
		return false
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	key, ok := parsed.(*ecdsa.PrivateKey)
	if err != nil || !ok || key.Curve != elliptic.P256() {
		return false
	}
	claims := jwt.MapClaims{
		"iss": public.TeamID, "sub": clientID, "aud": "https://appleid.apple.com",
		"iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = public.KeyID
	signed, err := token.SignedString(key)
	if err != nil {
		return false
	}
	verified, err := jwt.Parse(signed, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodES256.Alg() {
			return nil, login.ErrInvalid
		}
		return &key.PublicKey, nil
	}, jwt.WithAudience("https://appleid.apple.com"), jwt.WithIssuer(public.TeamID), jwt.WithExpirationRequired(), jwt.WithTimeFunc(func() time.Time { return now }))
	return err == nil && verified.Valid
}

type Authenticator interface {
	Authenticate(context.Context, string, string) (iam.AuthenticatedContext, error)
}

type Service struct {
	auth        Authenticator
	repository  login.Repository
	sealer      login.SecretSealer
	adapter     login.ProviderAdapter
	callbackURI string
	stepUpKey   []byte
	clock       func() time.Time
}

func NewService(auth Authenticator, repository login.Repository, sealer login.SecretSealer, adapter login.ProviderAdapter, callbackURI string, stepUpKey []byte) *Service {
	return &Service{
		auth: auth, repository: repository, sealer: sealer, adapter: adapter,
		callbackURI: strings.TrimRight(strings.TrimSpace(callbackURI), "/"), stepUpKey: append([]byte(nil), stepUpKey...), clock: time.Now,
	}
}

func (service *Service) authorizeAdmin(ctx context.Context, token, permission string) (iam.AuthenticatedContext, error) {
	principal, err := service.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return iam.AuthenticatedContext{}, err
	}
	for _, candidate := range principal.Permissions {
		if candidate == permission {
			return principal, nil
		}
	}
	return iam.AuthenticatedContext{}, login.ErrForbidden
}

func withPrincipal(principal iam.AuthenticatedContext, input login.Principal) login.Principal {
	input.TenantID, input.UserID, input.SessionID = principal.Tenant.ID, principal.User.ID, principal.SessionID
	return input
}

func normalizeListFilter(filter login.ListFilter) (login.ListFilter, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.ProviderCode = strings.ToLower(strings.TrimSpace(filter.ProviderCode))
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	if filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 || len([]rune(filter.Query)) > 160 ||
		filter.ProviderCode != "" && !registered(filter.ProviderCode) ||
		filter.Status != "" && filter.Status != "draft" && filter.Status != "active" && filter.Status != "disabled" {
		return filter, login.ErrInvalid
	}
	return filter, nil
}

func normalizeConfigInput(input login.ConfigInput) (login.ConfigInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.ProviderCode = strings.ToLower(strings.TrimSpace(input.ProviderCode))
	if input.ConfigSchemaVersion == 0 {
		input.ConfigSchemaVersion = 1
	}
	if len([]rune(input.Name)) == 0 || len([]rune(input.Name)) > 160 || len([]rune(input.Description)) > 2000 || input.ConfigSchemaVersion != 1 {
		return input, login.ErrInvalid
	}
	externalClientID, canonical, err := login.NormalizeConfig(input.ProviderCode, input.ExternalClientID, input.PublicConfig, false)
	if err != nil {
		return input, err
	}
	input.ExternalClientID, input.PublicConfig = externalClientID, canonical
	return input, nil
}

func registered(provider string) bool {
	_, ok := login.Descriptor(provider)
	return ok
}

func (service *Service) decorateConfig(config login.Config) login.Config {
	if config.ProviderCode == login.ProviderGitHub {
		config.CallbackURI = login.GitHubBrowserCallbackURI(service.callbackURI)
	}
	config.SecretCiphertext, config.SecretKeyVersion = nil, nil
	return config
}

func (service *Service) Catalog(ctx context.Context, token string) (login.Catalog, error) {
	if _, err := service.authorizeAdmin(ctx, token, "sys.login_provider_config.read"); err != nil {
		return login.Catalog{}, err
	}
	return login.RegisteredCatalog(), nil
}

func (service *Service) ListConfigs(ctx context.Context, token string, filter login.ListFilter) (login.ConfigPage, error) {
	principal, err := service.authorizeAdmin(ctx, token, "sys.login_provider_config.read")
	if err != nil {
		return login.ConfigPage{}, err
	}
	filter, err = normalizeListFilter(filter)
	if err != nil {
		return login.ConfigPage{}, err
	}
	page, err := service.repository.ListConfigs(ctx, principal.Tenant.ID, filter)
	if err != nil {
		return page, err
	}
	for index := range page.Items {
		page.Items[index] = service.decorateConfig(page.Items[index])
	}
	return page, nil
}

func (service *Service) GetConfig(ctx context.Context, token string, id uuid.UUID) (login.Config, error) {
	principal, err := service.authorizeAdmin(ctx, token, "sys.login_provider_config.read")
	if err != nil {
		return login.Config{}, err
	}
	if id == uuid.Nil {
		return login.Config{}, login.ErrInvalid
	}
	config, err := service.repository.GetConfig(ctx, principal.Tenant.ID, id)
	return service.decorateConfig(config), err
}

func (service *Service) CreateConfig(ctx context.Context, token string, principal login.Principal, input login.ConfigInput) (login.Config, error) {
	authenticated, err := service.authorizeAdmin(ctx, token, "sys.login_provider_config.create")
	if err != nil {
		return login.Config{}, err
	}
	input, err = normalizeConfigInput(input)
	if err != nil || input.LockVersion != 0 {
		return login.Config{}, login.ErrInvalid
	}
	config, err := service.repository.CreateConfig(ctx, withPrincipal(authenticated, principal), input)
	return service.decorateConfig(config), err
}

func (service *Service) UpdateConfig(ctx context.Context, token string, principal login.Principal, id uuid.UUID, input login.ConfigInput) (login.Config, error) {
	authenticated, err := service.authorizeAdmin(ctx, token, "sys.login_provider_config.update")
	if err != nil {
		return login.Config{}, err
	}
	input, err = normalizeConfigInput(input)
	if err != nil || id == uuid.Nil || input.LockVersion < 1 {
		return login.Config{}, login.ErrInvalid
	}
	current, err := service.repository.GetConfig(ctx, authenticated.Tenant.ID, id)
	if err != nil {
		return login.Config{}, err
	}
	if current.ProviderCode != input.ProviderCode {
		return login.Config{}, login.ErrInvalid
	}
	config, err := service.repository.UpdateConfig(ctx, withPrincipal(authenticated, principal), id, input)
	return service.decorateConfig(config), err
}

func (service *Service) RotateSecret(ctx context.Context, token string, principal login.Principal, id uuid.UUID, input login.SecretInput) (login.Config, error) {
	authenticated, err := service.authorizeAdmin(ctx, token, "sys.login_provider_config.rotate_secret")
	if err != nil {
		return login.Config{}, err
	}
	if id == uuid.Nil || input.LockVersion < 1 || service.sealer == nil {
		return login.Config{}, login.ErrInvalid
	}
	current, err := service.repository.GetConfig(ctx, authenticated.Tenant.ID, id)
	if err != nil {
		return login.Config{}, err
	}
	names, fingerprint, values, err := login.ValidateSecrets(current.ProviderCode, input.Values)
	if err != nil || len(names) == 0 {
		return login.Config{}, login.ErrInvalid
	}
	plaintext, err := json.Marshal(values)
	if err != nil {
		return login.Config{}, login.ErrInvalid
	}
	ciphertext, keyVersion, err := service.sealer.Seal(plaintext, configAAD(authenticated.Tenant.ID, id))
	if err != nil {
		return login.Config{}, err
	}
	config, err := service.repository.RotateSecret(ctx, withPrincipal(authenticated, principal), id, input.LockVersion, ciphertext, keyVersion, names, fingerprint)
	return service.decorateConfig(config), err
}

func configAAD(tenantID, configID uuid.UUID) string {
	return "login-provider-config:" + tenantID.String() + ":" + configID.String()
}

func (service *Service) Preflight(ctx context.Context, token string, principal login.Principal, id uuid.UUID, lockVersion int32) (login.Config, error) {
	authenticated, err := service.authorizeAdmin(ctx, token, "sys.login_provider_config.preflight")
	if err != nil {
		return login.Config{}, err
	}
	if id == uuid.Nil || lockVersion < 1 {
		return login.Config{}, login.ErrInvalid
	}
	current, err := service.repository.GetConfigSecret(ctx, authenticated.Tenant.ID, id)
	if err != nil {
		return login.Config{}, err
	}
	issues := []string{}
	if _, _, normalizeErr := login.NormalizeConfig(current.ProviderCode, current.ExternalClientID, current.PublicConfig, true); normalizeErr != nil {
		issues = append(issues, "login_provider.config.invalid")
	}
	descriptor, _ := login.Descriptor(current.ProviderCode)
	if descriptor.RequiresSecret {
		if !current.HasSecret || service.sealer == nil {
			issues = append(issues, "login_provider.secret.missing")
		} else {
			plaintext, openErr := service.sealer.Open(current.SecretCiphertext, configAAD(current.TenantID, current.ID))
			var values map[string]string
			if openErr != nil || json.Unmarshal(plaintext, &values) != nil {
				issues = append(issues, "login_provider.secret.invalid")
			} else if _, _, values, validateErr := login.ValidateSecrets(current.ProviderCode, values); validateErr != nil {
				issues = append(issues, "login_provider.secret.invalid")
			} else if current.ProviderCode == login.ProviderApple {
				var public login.ApplePublicConfig
				if json.Unmarshal(current.PublicConfig, &public) != nil || !validAppleCredential(values["private_key_p8"], public, current.ExternalClientID, service.clock().UTC()) {
					issues = append(issues, "login_provider.apple.private_key_invalid")
				}
			}
		}
	}
	sort.Strings(issues)
	config, err := service.repository.SetPreflight(ctx, withPrincipal(authenticated, principal), id, lockVersion, len(issues) == 0, issues)
	return service.decorateConfig(config), err
}

func (service *Service) Activate(ctx context.Context, token string, principal login.Principal, id uuid.UUID, lockVersion int32) (login.Config, error) {
	authenticated, err := service.authorizeAdmin(ctx, token, "sys.login_provider_config.update")
	if err != nil {
		return login.Config{}, err
	}
	if id == uuid.Nil || lockVersion < 1 {
		return login.Config{}, login.ErrInvalid
	}
	config, err := service.repository.SetStatus(ctx, withPrincipal(authenticated, principal), id, lockVersion, "active")
	return service.decorateConfig(config), err
}

func (service *Service) Disable(ctx context.Context, token string, principal login.Principal, id uuid.UUID, lockVersion int32) (login.Config, error) {
	authenticated, err := service.authorizeAdmin(ctx, token, "sys.login_provider_config.update")
	if err != nil {
		return login.Config{}, err
	}
	if id == uuid.Nil || lockVersion < 1 {
		return login.Config{}, login.ErrInvalid
	}
	config, err := service.repository.SetStatus(ctx, withPrincipal(authenticated, principal), id, lockVersion, "disabled")
	return service.decorateConfig(config), err
}

func (service *Service) DeleteConfig(ctx context.Context, token string, principal login.Principal, id uuid.UUID, lockVersion int32) error {
	authenticated, err := service.authorizeAdmin(ctx, token, "sys.login_provider_config.delete")
	if err != nil {
		return err
	}
	if id == uuid.Nil || lockVersion < 1 {
		return login.ErrInvalid
	}
	return service.repository.DeleteConfig(ctx, withPrincipal(authenticated, principal), id, lockVersion)
}

func completeBindings(appID uuid.UUID, stored []login.Binding) []login.Binding {
	byProvider := make(map[string]login.Binding, len(stored))
	for _, item := range stored {
		byProvider[item.ProviderCode] = item
	}
	items := make([]login.Binding, 0, len(login.ProviderCodes))
	for index, provider := range login.ProviderCodes {
		item, exists := byProvider[provider]
		if !exists {
			item = login.Binding{AppID: appID, ProviderCode: provider, SortOrder: int32((index + 1) * 100)}
		}
		items = append(items, item)
	}
	return items
}

func (service *Service) ListBindings(ctx context.Context, token string, appID uuid.UUID) (login.BindingList, error) {
	authenticated, err := service.authorizeAdmin(ctx, token, "app.login_provider_binding.read")
	if err != nil {
		return login.BindingList{}, err
	}
	if appID == uuid.Nil {
		return login.BindingList{}, login.ErrInvalid
	}
	items, err := service.repository.ListBindings(ctx, authenticated.Tenant.ID, appID)
	return login.BindingList{Items: completeBindings(appID, items)}, err
}

func normalizeBindings(input login.BulkBindingInput) ([]login.BindingInput, error) {
	if len(input.Bindings) != len(login.ProviderCodes) {
		return nil, login.ErrInvalid
	}
	seen := map[string]bool{}
	items := append([]login.BindingInput(nil), input.Bindings...)
	for index := range items {
		item := &items[index]
		item.ProviderCode = strings.ToLower(strings.TrimSpace(item.ProviderCode))
		if !registered(item.ProviderCode) || seen[item.ProviderCode] || item.SortOrder < 0 || item.SortOrder > 1000 || item.LockVersion < 0 || item.LoginProviderConfigID == nil && item.Enabled {
			return nil, login.ErrInvalid
		}
		seen[item.ProviderCode] = true
	}
	for _, provider := range login.ProviderCodes {
		if !seen[provider] {
			return nil, login.ErrInvalid
		}
	}
	sort.Slice(items, func(left, right int) bool { return items[left].ProviderCode < items[right].ProviderCode })
	return items, nil
}

func (service *Service) ReplaceBindings(ctx context.Context, token string, principal login.Principal, appID uuid.UUID, input login.BulkBindingInput) (login.BindingList, error) {
	authenticated, err := service.authorizeAdmin(ctx, token, "app.login_provider_binding.update")
	if err != nil {
		return login.BindingList{}, err
	}
	items, err := normalizeBindings(input)
	if err != nil || appID == uuid.Nil {
		return login.BindingList{}, login.ErrInvalid
	}
	stored, err := service.repository.ReplaceBindings(ctx, withPrincipal(authenticated, principal), appID, items)
	return login.BindingList{Items: completeBindings(appID, stored)}, err
}

func isIdentityError(err error) bool {
	return errors.Is(err, login.ErrIdentityConflict) || errors.Is(err, login.ErrLinkRequired)
}
