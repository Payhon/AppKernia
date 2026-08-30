package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	push "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iam.AuthenticatedContext, error)
}

type providerSpec struct {
	platforms     []string
	buildVariants []string
	required      []string
	optional      []string
	secrets       []string
}

var providerSpecs = map[string]providerSpec{
	push.ProviderAPNS: {
		platforms: []string{"ios"}, buildVariants: []string{"ios"},
		required: []string{"team_id", "key_id", "bundle_id", "apns_environment"}, secrets: []string{"private_key_p8"},
	},
	push.ProviderFCM: {
		platforms: []string{"android"}, buildVariants: []string{"android_google"},
		required: []string{"project_id", "package_name"}, secrets: []string{"service_account_json"},
	},
	push.ProviderHuaweiAndroid: {
		platforms: []string{"android"}, buildVariants: []string{"android_china"},
		required: []string{"client_id", "package_name", "notification_channel_id"}, secrets: []string{"client_secret"},
	},
	push.ProviderHonor: {
		platforms: []string{"android"}, buildVariants: []string{"android_china"},
		required: []string{"app_id", "client_id", "package_name", "notification_channel_id"}, secrets: []string{"client_secret"},
	},
	push.ProviderXiaomi: {
		platforms: []string{"android"}, buildVariants: []string{"android_china"},
		required: []string{"app_id", "app_key", "package_name", "region", "notification_channel_id"}, secrets: []string{"app_secret"},
	},
	push.ProviderOPPO: {
		platforms: []string{"android"}, buildVariants: []string{"android_china"},
		required: []string{"app_key", "package_name", "notification_channel_id"}, secrets: []string{"master_secret"},
	},
	push.ProviderVivo: {
		platforms: []string{"android"}, buildVariants: []string{"android_china"},
		required: []string{"app_id", "app_key", "package_name", "service_category", "operations_category"}, secrets: []string{"app_secret"},
	},
	push.ProviderMeizu: {
		platforms: []string{"android"}, buildVariants: []string{"android_china"},
		required: []string{"app_id", "app_key", "package_name"}, secrets: []string{"app_secret"},
	},
	push.ProviderHarmony: {
		platforms: []string{"harmony"}, buildVariants: []string{"harmony"},
		required: []string{"project_id", "client_id", "bundle_name", "service_category", "operations_category"}, secrets: []string{"service_account_json"},
	},
}

type Service struct {
	auth         Authenticator
	repo         push.Repository
	sealer       push.SecretSealer
	preflighter  push.Preflighter
	tokenHashKey []byte
	environment  string
	enabled      bool
	clock        func() time.Time
}

func NewService(auth Authenticator, repo push.Repository, sealer push.SecretSealer, tokenHashKey []byte, environment string, enabled bool, preflighter push.Preflighter) *Service {
	return &Service{auth: auth, repo: repo, sealer: sealer, preflighter: preflighter, tokenHashKey: append([]byte(nil), tokenHashKey...), environment: normalizeEnvironment(environment), enabled: enabled, clock: time.Now}
}

func Catalog() []push.ProviderCatalogItem {
	items := make([]push.ProviderCatalogItem, 0, len(push.Providers))
	for _, provider := range push.Providers {
		spec := providerSpecs[provider]
		publicFields := append(append([]string{}, spec.required...), spec.optional...)
		items = append(items, push.ProviderCatalogItem{Provider: provider, Platforms: append([]string{}, spec.platforms...), BuildVariants: append([]string{}, spec.buildVariants...), PublicFields: publicFields, SecretFields: append([]string{}, spec.secrets...), SupportsPreflight: true, SupportsTest: true, ConfigSchemaVersion: 1})
	}
	return items
}

func normalizeEnvironment(value string) string {
	if value, ok := parseEnvironment(value); ok {
		return value
	}
	return "development"
}

func parseEnvironment(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "development", "test", "staging", "production":
		return value, true
	default:
		return value, false
	}
}

func contains(values []string, value string) bool { return slices.Contains(values, value) }

func normalizeDevice(input push.DeviceInput) (push.DeviceInput, error) {
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	input.BuildVariant = strings.ToLower(strings.TrimSpace(input.BuildVariant))
	input.Token = strings.TrimSpace(input.Token)
	input.Locale = strings.TrimSpace(input.Locale)
	input.SDKVersion = strings.TrimSpace(input.SDKVersion)
	input.AppVersion = strings.TrimSpace(input.AppVersion)
	spec, ok := providerSpecs[input.Provider]
	if !ok || !contains(spec.platforms, input.Platform) || !contains(spec.buildVariants, input.BuildVariant) ||
		len(input.Token) < 16 || len(input.Token) > 8192 ||
		(input.Locale != "zh-CN" && input.Locale != "en-US") ||
		len(input.SDKVersion) > 64 || len(input.AppVersion) > 64 {
		return input, push.ErrInvalid
	}
	return input, nil
}

func (s *Service) mobile(ctx context.Context, token string) (iam.AuthenticatedContext, error) {
	auth, err := s.auth.Authenticate(ctx, token, "ak-mobile")
	if err != nil {
		return iam.AuthenticatedContext{}, err
	}
	if auth.AppID == nil || *auth.AppID == uuid.Nil {
		return iam.AuthenticatedContext{}, push.ErrForbidden
	}
	return auth, nil
}

func (s *Service) admin(ctx context.Context, token, permission string) (iam.AuthenticatedContext, error) {
	auth, err := s.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return iam.AuthenticatedContext{}, err
	}
	if !slices.Contains(auth.Permissions, permission) {
		return iam.AuthenticatedContext{}, push.ErrForbidden
	}
	return auth, nil
}

func (s *Service) mobilePrincipal(ctx context.Context, token string, request push.Principal) (push.Principal, error) {
	auth, err := s.mobile(ctx, token)
	if err != nil {
		return push.Principal{}, err
	}
	deviceID, err := s.repo.SessionDevice(ctx, auth.SessionID, auth.User.ID, *auth.AppID)
	if err != nil || deviceID == uuid.Nil {
		return push.Principal{}, push.ErrUnavailable
	}
	request.TenantID, request.AppID, request.UserID, request.SessionID, request.DeviceID = auth.Tenant.ID, *auth.AppID, auth.User.ID, auth.SessionID, deviceID
	return request, nil
}

func (s *Service) CurrentDevice(ctx context.Context, token string) (*push.Device, error) {
	principal, err := s.mobilePrincipal(ctx, token, push.Principal{})
	if err != nil {
		return nil, err
	}
	return s.repo.CurrentDevice(ctx, principal)
}

func (s *Service) RegisterDevice(ctx context.Context, token string, request push.Principal, input push.DeviceInput) (push.Device, error) {
	if !s.enabled || s.sealer == nil || len(s.tokenHashKey) < 32 {
		return push.Device{}, push.ErrUnavailable
	}
	input, err := normalizeDevice(input)
	if err != nil {
		return push.Device{}, err
	}
	principal, err := s.mobilePrincipal(ctx, token, request)
	if err != nil {
		return push.Device{}, err
	}
	consented, err := s.repo.HasCurrentLegalConsent(ctx, principal.AppID, principal.UserID)
	if err != nil {
		return push.Device{}, err
	}
	if !consented {
		return push.Device{}, push.ErrForbidden
	}
	mac := hmac.New(sha256.New, s.tokenHashKey)
	_, _ = mac.Write([]byte(principal.AppID.String()))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(input.Provider))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(input.Token))
	hash := mac.Sum(nil)
	ciphertext, keyVersion, err := s.sealer.Seal([]byte(input.Token), "push-token:"+principal.AppID.String()+":"+input.Provider)
	if err != nil {
		return push.Device{}, err
	}
	return s.repo.UpsertDevice(ctx, principal, input, hash, ciphertext, keyVersion)
}

func (s *Service) DisableDevice(ctx context.Context, token string, request push.Principal, id uuid.UUID) error {
	if id == uuid.Nil {
		return push.ErrInvalid
	}
	principal, err := s.mobilePrincipal(ctx, token, request)
	if err != nil {
		return err
	}
	return s.repo.DisableDevice(ctx, principal, id)
}

func (s *Service) MarkOpened(ctx context.Context, token string, request push.Principal, deliveryID uuid.UUID) error {
	if deliveryID == uuid.Nil {
		return push.ErrInvalid
	}
	principal, err := s.mobilePrincipal(ctx, token, request)
	if err != nil {
		return err
	}
	return s.repo.MarkOpened(ctx, principal, deliveryID)
}

func normalizeConfig(input push.ProviderConfigInput) (push.ProviderConfigInput, error) {
	environment, validEnvironment := parseEnvironment(input.Environment)
	if !validEnvironment {
		return input, push.ErrInvalid
	}
	input.Environment = environment
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	if input.ConfigSchemaVersion == 0 {
		input.ConfigSchemaVersion = 1
	}
	spec, ok := providerSpecs[input.Provider]
	if !ok || input.ConfigSchemaVersion != 1 || (input.LockVersion < 0) {
		return input, push.ErrInvalid
	}
	var fields map[string]json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(string(input.PublicConfig)))
	decoder.DisallowUnknownFields()
	if len(input.PublicConfig) == 0 || decoder.Decode(&fields) != nil || fields == nil {
		return input, push.ErrInvalid
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return input, push.ErrInvalid
	}
	allowed := append(append([]string{}, spec.required...), spec.optional...)
	for key := range fields {
		if !contains(allowed, key) {
			return input, push.ErrInvalid
		}
	}
	clean := make(map[string]string, len(fields))
	for _, key := range allowed {
		raw, present := fields[key]
		if !present {
			if contains(spec.required, key) {
				return input, push.ErrInvalid
			}
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) != nil {
			return input, push.ErrInvalid
		}
		value = strings.TrimSpace(value)
		if (len(value) == 0 && contains(spec.required, key)) || len(value) > 2048 {
			return input, push.ErrInvalid
		}
		if value != "" {
			clean[key] = value
		}
	}
	if input.Provider == push.ProviderAPNS && clean["apns_environment"] != "sandbox" && clean["apns_environment"] != "production" {
		return input, push.ErrInvalid
	}
	if input.Provider == push.ProviderXiaomi && !contains([]string{"china", "singapore", "europe", "india", "russia"}, clean["region"]) {
		return input, push.ErrInvalid
	}
	if input.Provider == push.ProviderVivo || input.Provider == push.ProviderHarmony {
		categoryPattern := regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)
		for _, key := range []string{"service_category", "operations_category"} {
			if value := clean[key]; value != "" && !categoryPattern.MatchString(value) {
				return input, push.ErrInvalid
			}
		}
	}
	encoded, err := json.Marshal(clean)
	if err != nil {
		return input, push.ErrInvalid
	}
	input.PublicConfig = encoded
	return input, nil
}

func adminPrincipal(auth iam.AuthenticatedContext, appID uuid.UUID, request push.Principal) push.Principal {
	request.TenantID, request.AppID, request.UserID, request.SessionID = auth.Tenant.ID, appID, auth.User.ID, auth.SessionID
	return request
}

func (s *Service) Catalog(ctx context.Context, token string) ([]push.ProviderCatalogItem, error) {
	if _, err := s.admin(ctx, token, "notify.push_provider.read"); err != nil {
		return nil, err
	}
	return Catalog(), nil
}

func (s *Service) ListConfigs(ctx context.Context, token string, appID uuid.UUID, environment string) ([]push.ProviderConfig, error) {
	auth, err := s.admin(ctx, token, "notify.push_provider.read")
	if err != nil {
		return nil, err
	}
	environment, validEnvironment := parseEnvironment(environment)
	if appID == uuid.Nil || !validEnvironment {
		return nil, push.ErrInvalid
	}
	return s.repo.ListConfigs(ctx, auth.Tenant.ID, appID, environment)
}

func (s *Service) UpsertConfig(ctx context.Context, token string, appID uuid.UUID, request push.Principal, input push.ProviderConfigInput) (push.ProviderConfig, error) {
	auth, err := s.admin(ctx, token, "notify.push_provider.manage")
	if err != nil {
		return push.ProviderConfig{}, err
	}
	if appID == uuid.Nil {
		return push.ProviderConfig{}, push.ErrInvalid
	}
	input, err = normalizeConfig(input)
	if err != nil {
		return push.ProviderConfig{}, err
	}
	return s.repo.UpsertConfig(ctx, adminPrincipal(auth, appID, request), input)
}

func (s *Service) RotateSecret(ctx context.Context, token string, appID, id uuid.UUID, request push.Principal, input push.SecretInput) (push.ProviderConfig, error) {
	auth, err := s.admin(ctx, token, "notify.push_provider.rotate_secret")
	if err != nil {
		return push.ProviderConfig{}, err
	}
	if appID == uuid.Nil || id == uuid.Nil || input.LockVersion < 1 || s.sealer == nil {
		return push.ProviderConfig{}, push.ErrInvalid
	}
	current, err := s.repo.GetConfig(ctx, auth.Tenant.ID, appID, id)
	if err != nil {
		return push.ProviderConfig{}, err
	}
	spec := providerSpecs[current.Provider]
	if len(input.Values) != len(spec.secrets) {
		return push.ProviderConfig{}, push.ErrInvalid
	}
	clean := make(map[string]string, len(input.Values))
	for _, key := range spec.secrets {
		value := strings.TrimSpace(input.Values[key])
		if value == "" || len(value) > 65536 {
			return push.ProviderConfig{}, push.ErrInvalid
		}
		clean[key] = value
	}
	plaintext, err := json.Marshal(clean)
	if err != nil {
		return push.ProviderConfig{}, push.ErrInvalid
	}
	ciphertext, keyVersion, err := s.sealer.Seal(plaintext, "push-config:"+appID.String()+":"+current.Provider+":"+current.Environment)
	if err != nil {
		return push.ProviderConfig{}, err
	}
	digest := sha256.Sum256(plaintext)
	fingerprint := hex.EncodeToString(digest[:8])
	return s.repo.RotateSecret(ctx, adminPrincipal(auth, appID, request), id, input.LockVersion, ciphertext, keyVersion, append([]string{}, spec.secrets...), fingerprint)
}

func (s *Service) Preflight(ctx context.Context, token string, appID, id uuid.UUID, request push.Principal, lockVersion int32) (push.ProviderConfig, error) {
	auth, err := s.admin(ctx, token, "notify.push.preflight")
	if err != nil {
		return push.ProviderConfig{}, err
	}
	if appID == uuid.Nil || id == uuid.Nil || lockVersion < 1 {
		return push.ProviderConfig{}, push.ErrInvalid
	}
	current, err := s.repo.GetConfig(ctx, auth.Tenant.ID, appID, id)
	if err != nil {
		return push.ProviderConfig{}, err
	}
	issues := []string{}
	if !current.HasSecret {
		issues = append(issues, "push.preflight.secret_missing")
	}
	if s.preflighter != nil && current.HasSecret {
		issues = append(issues, s.preflighter.Preflight(ctx, auth.Tenant.ID, appID, current.Provider)...)
	}
	sort.Strings(issues)
	checkedAt := s.clock().UTC()
	result := push.Preflight{Ready: len(issues) == 0, Provider: current.Provider, Environment: current.Environment, Issues: issues, CheckedAt: checkedAt}
	return s.repo.RecordPreflight(ctx, adminPrincipal(auth, appID, request), id, lockVersion, result)
}

func (s *Service) SetStatus(ctx context.Context, token string, appID, id uuid.UUID, request push.Principal, lockVersion int32, status string) (push.ProviderConfig, error) {
	auth, err := s.admin(ctx, token, "notify.push_provider.manage")
	if err != nil {
		return push.ProviderConfig{}, err
	}
	if appID == uuid.Nil || id == uuid.Nil || lockVersion < 1 || !contains([]string{"active", "disabled"}, status) {
		return push.ProviderConfig{}, push.ErrInvalid
	}
	current, err := s.repo.GetConfig(ctx, auth.Tenant.ID, appID, id)
	if err != nil {
		return push.ProviderConfig{}, err
	}
	if status == "active" && (!current.HasSecret || current.LastPreflightStatus != "ready") {
		return push.ProviderConfig{}, push.ErrConflict
	}
	return s.repo.SetStatus(ctx, adminPrincipal(auth, appID, request), id, lockVersion, status)
}

var routeKey = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,95}$`)

func (s *Service) Test(ctx context.Context, token string, appID, configID uuid.UUID, request push.Principal, input push.TestInput) (push.TestDelivery, error) {
	auth, err := s.admin(ctx, token, "notify.push.test")
	if err != nil {
		return push.TestDelivery{}, err
	}
	input.Title, input.Body = strings.TrimSpace(input.Title), strings.TrimSpace(input.Body)
	if appID == uuid.Nil || configID == uuid.Nil || input.PushDeviceID == uuid.Nil || len([]rune(input.Title)) < 1 || len([]rune(input.Title)) > 64 || len([]rune(input.Body)) < 1 || len([]rune(input.Body)) > 180 {
		return push.TestDelivery{}, push.ErrInvalid
	}
	config, err := s.repo.GetConfig(ctx, auth.Tenant.ID, appID, configID)
	if err != nil {
		return push.TestDelivery{}, err
	}
	if config.Status != "active" || config.Environment != s.environment || s.sealer == nil {
		return push.TestDelivery{}, push.ErrConflict
	}
	device, err := s.repo.TestDevice(ctx, auth.Tenant.ID, appID, input.PushDeviceID)
	if err != nil {
		return push.TestDelivery{}, err
	}
	if device.Provider != config.Provider {
		return push.TestDelivery{}, push.ErrInvalid
	}
	id, err := uuid.NewV7()
	if err != nil {
		return push.TestDelivery{}, err
	}
	payload := push.SendPayload{SchemaVersion: 1, DeliveryID: id, MessageID: id, Title: input.Title, Body: input.Body, Category: push.CategoryServiceSecurity, TTLSeconds: 3600, RouteKey: "push.test"}
	plain, _ := json.Marshal(payload)
	ciphertext, keyVersion, err := s.sealer.Seal(plain, "push-payload:"+appID.String()+":"+device.ID.String())
	if err != nil {
		return push.TestDelivery{}, err
	}
	if err = s.repo.QueueTestDelivery(ctx, adminPrincipal(auth, appID, request), id, configID, device.ID, ciphertext, keyVersion); err != nil {
		return push.TestDelivery{}, err
	}
	return push.TestDelivery{ID: id, Status: "pending"}, nil
}

func (s *Service) ListTestDevices(ctx context.Context, token string, appID uuid.UUID, provider string) ([]push.Device, error) {
	auth, err := s.admin(ctx, token, "notify.push.test")
	if err != nil {
		return nil, err
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if appID == uuid.Nil || !contains(push.Providers, provider) {
		return nil, push.ErrInvalid
	}
	return s.repo.ListTestDevices(ctx, auth.Tenant.ID, appID, provider)
}

func (s *Service) DeliverySummary(ctx context.Context, token string, appID uuid.UUID) ([]push.DeliverySummaryItem, error) {
	auth, err := s.admin(ctx, token, "notify.push_provider.read")
	if err != nil {
		return nil, err
	}
	if appID == uuid.Nil {
		return nil, push.ErrInvalid
	}
	return s.repo.DeliverySummary(ctx, auth.Tenant.ID, appID)
}

func (s *Service) RuntimeCapability(ctx context.Context, appID uuid.UUID) (push.RuntimeCapability, error) {
	if appID == uuid.Nil {
		return push.RuntimeCapability{}, push.ErrInvalid
	}
	capability, err := s.repo.RuntimeCapability(ctx, appID, s.environment)
	if err != nil {
		return push.RuntimeCapability{}, err
	}
	capability.Enabled = capability.Enabled && s.enabled
	return capability, nil
}

func ValidRouteKey(value string) bool { return value == "" || routeKey.MatchString(value) }
