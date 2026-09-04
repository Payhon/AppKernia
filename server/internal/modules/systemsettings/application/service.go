package application

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	settings "github.com/appkernia/appkernia/server/internal/modules/systemsettings/domain"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Service struct {
	auth               Authenticator
	repo               settings.Repository
	sealer             settings.SecretSealer
	platformTenantCode string
}

func NewService(auth Authenticator, repo settings.Repository, sealer settings.SecretSealer, platformTenantCode ...string) *Service {
	code := "local"
	if len(platformTenantCode) > 0 && strings.TrimSpace(platformTenantCode[0]) != "" {
		code = strings.ToLower(strings.TrimSpace(platformTenantCode[0]))
	}
	return &Service{auth: auth, repo: repo, sealer: sealer, platformTenantCode: code}
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
	return iamdomain.AuthenticatedContext{}, settings.ErrForbidden
}

func pageFilter(f settings.PageFilter) (settings.PageFilter, error) {
	f.Query, f.ModuleCode, f.Group, f.ValueType, f.Status, f.Sort = strings.TrimSpace(f.Query), strings.TrimSpace(f.ModuleCode), strings.TrimSpace(f.Group), strings.TrimSpace(f.ValueType), strings.TrimSpace(f.Status), strings.TrimSpace(f.Sort)
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Sort == "" {
		f.Sort = "sort_order"
	}
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || len(f.Query) > 160 || !oneOf(f.Status, "", "active", "disabled") || !oneOf(f.Sort, "sort_order", "updated_desc", "key") {
		return f, settings.ErrInvalid
	}
	return f, nil
}

func principal(auth iamdomain.AuthenticatedContext, p settings.Principal) settings.Principal {
	p.TenantID, p.UserID, p.SessionID = auth.Tenant.ID, auth.User.ID, auth.SessionID
	return p
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (s *Service) canUpdateGlobalConfigs(auth iamdomain.AuthenticatedContext) bool {
	return strings.EqualFold(strings.TrimSpace(auth.Tenant.Code), s.platformTenantCode) &&
		contains(auth.Roles, "super-admin") &&
		contains(auth.Permissions, "sys.config.update") &&
		contains(auth.Permissions, "sys.platform_config.update")
}

func (s *Service) configPrincipal(auth iamdomain.AuthenticatedContext, p settings.Principal) settings.Principal {
	p = principal(auth, p)
	p.CanUpdateGlobalConfigs = s.canUpdateGlobalConfigs(auth)
	return p
}

func unlockGlobalConfigs(page *settings.ConfigPage, allowed bool) {
	if !allowed {
		return
	}
	for index := range page.Items {
		if page.Items[index].TenantID == nil {
			page.Items[index].IsLocked = false
		}
	}
}

func (s *Service) ListConfigs(ctx context.Context, token string, f settings.PageFilter) (settings.ConfigPage, error) {
	auth, err := s.authorize(ctx, token, "sys.config.read")
	if err != nil {
		return settings.ConfigPage{}, err
	}
	f, err = pageFilter(f)
	if err != nil || !oneOf(f.ValueType, "", "string", "integer", "decimal", "boolean", "json", "datetime") {
		return settings.ConfigPage{}, settings.ErrInvalid
	}
	page, err := s.repo.ListConfigs(ctx, auth.Tenant.ID, f)
	if err != nil {
		return settings.ConfigPage{}, err
	}
	unlockGlobalConfigs(&page, s.canUpdateGlobalConfigs(auth))
	return page, nil
}

func normalizeRegionFilter(f settings.RegionFilter) (settings.RegionFilter, error) {
	f.Query, f.ParentCode, f.Status = strings.TrimSpace(f.Query), strings.TrimSpace(f.ParentCode), strings.TrimSpace(f.Status)
	if f.Limit == 0 {
		f.Limit = 100
	}
	if f.Limit < 1 || f.Limit > 200 || len(f.Query) > 160 || len(f.ParentCode) > 32 || !oneOf(f.Status, "", "active", "disabled") || f.Level != nil && (*f.Level < 0 || *f.Level > 10) {
		return f, settings.ErrInvalid
	}
	return f, nil
}

func (s *Service) ListRegions(ctx context.Context, token string, f settings.RegionFilter) ([]settings.Region, error) {
	if _, err := s.authorize(ctx, token, "sys.region.read"); err != nil {
		return nil, err
	}
	normalized, err := normalizeRegionFilter(f)
	if err != nil {
		return nil, err
	}
	return s.repo.ListRegions(ctx, normalized)
}
func (s *Service) ListPublicRegions(ctx context.Context, f settings.RegionFilter) ([]settings.Region, error) {
	normalized, err := normalizeRegionFilter(f)
	if err != nil {
		return nil, err
	}
	normalized.Status = "active"
	return s.repo.ListRegions(ctx, normalized)
}

func normalizeRegionCreate(in settings.RegionCreateInput) (settings.RegionCreateInput, error) {
	in.Code, in.ParentCode = strings.TrimSpace(in.Code), strings.TrimSpace(in.ParentCode)
	in.Name, in.FullName = strings.TrimSpace(in.Name), strings.TrimSpace(in.FullName)
	in.PostalCode, in.Status = strings.TrimSpace(in.PostalCode), strings.TrimSpace(in.Status)
	if in.Status == "" {
		in.Status = "active"
	}
	if !validRegionCode(in.Code) || !validRegionCode(in.ParentCode) || in.Code == in.ParentCode || in.Name == "" || len(in.Name) > 160 || in.FullName == "" || len(in.FullName) > 500 || len(in.PostalCode) > 24 || !oneOf(in.Status, "active", "disabled") || !validCoordinates(in.Longitude, in.Latitude) {
		return in, settings.ErrInvalid
	}
	return in, nil
}

func normalizeRegionUpdate(in settings.RegionUpdateInput) (settings.RegionUpdateInput, error) {
	in.Name, in.FullName = strings.TrimSpace(in.Name), strings.TrimSpace(in.FullName)
	in.PostalCode, in.Status = strings.TrimSpace(in.PostalCode), strings.TrimSpace(in.Status)
	if in.Name == "" || len(in.Name) > 160 || in.FullName == "" || len(in.FullName) > 500 || len(in.PostalCode) > 24 || !oneOf(in.Status, "active", "disabled") || in.Version < 1 || !validCoordinates(in.Longitude, in.Latitude) {
		return in, settings.ErrInvalid
	}
	return in, nil
}

func validCoordinates(longitude, latitude *float64) bool {
	return (longitude == nil || *longitude >= -180 && *longitude <= 180) && (latitude == nil || *latitude >= -90 && *latitude <= 90)
}

func validRegionCode(code string) bool {
	code = strings.TrimSpace(code)
	return regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$`).MatchString(code)
}

func (s *Service) CreateRegion(ctx context.Context, token string, p settings.Principal, in settings.RegionCreateInput) (settings.Region, error) {
	auth, err := s.authorize(ctx, token, "sys.region.create")
	if err != nil {
		return settings.Region{}, err
	}
	in, err = normalizeRegionCreate(in)
	if err != nil {
		return settings.Region{}, err
	}
	return s.repo.CreateRegion(ctx, principal(auth, p), in)
}

func (s *Service) UpdateRegion(ctx context.Context, token string, p settings.Principal, code string, in settings.RegionUpdateInput) (settings.Region, error) {
	auth, err := s.authorize(ctx, token, "sys.region.update")
	if err != nil {
		return settings.Region{}, err
	}
	code = strings.TrimSpace(code)
	in, err = normalizeRegionUpdate(in)
	if err != nil || !validRegionCode(code) {
		return settings.Region{}, settings.ErrInvalid
	}
	return s.repo.UpdateRegion(ctx, principal(auth, p), code, in)
}

func (s *Service) DeleteRegion(ctx context.Context, token string, p settings.Principal, code string) error {
	auth, err := s.authorize(ctx, token, "sys.region.delete")
	if err != nil {
		return err
	}
	code = strings.TrimSpace(code)
	if !validRegionCode(code) {
		return settings.ErrInvalid
	}
	return s.repo.DeleteRegion(ctx, principal(auth, p), code)
}

func normalizeConfig(in settings.ConfigInput, creating bool) (settings.ConfigInput, error) {
	in.ModuleCode, in.ConfigGroup, in.ConfigKey, in.DisplayName = strings.TrimSpace(in.ModuleCode), strings.TrimSpace(in.ConfigGroup), strings.TrimSpace(in.ConfigKey), strings.TrimSpace(in.DisplayName)
	in.ValueType, in.Description, in.Status, in.SecretValue = strings.TrimSpace(in.ValueType), strings.TrimSpace(in.Description), strings.TrimSpace(in.Status), strings.TrimSpace(in.SecretValue)
	if in.Status == "" {
		in.Status = "active"
	}
	if len(in.ModuleCode) < 1 || len(in.ModuleCode) > 64 || len(in.ConfigGroup) < 1 || len(in.ConfigGroup) > 96 || len(in.ConfigKey) < 1 || len(in.ConfigKey) > 160 || len(in.DisplayName) < 1 || len(in.DisplayName) > 160 || len(in.Description) > 1000 || !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`).MatchString(in.ConfigKey) || !oneOf(in.ValueType, "string", "integer", "decimal", "boolean", "json", "datetime") || !oneOf(in.Status, "active", "disabled") || in.IsSecret && in.IsPublic {
		return in, settings.ErrInvalid
	}
	if len(in.ValidationSchema) == 0 {
		in.ValidationSchema = json.RawMessage(`{}`)
	}
	if !json.Valid(in.ValidationSchema) {
		return in, settings.ErrInvalid
	}
	if in.IsSecret {
		in.Value, in.DefaultValue = nil, nil
		if creating && in.SecretValue == "" {
			return in, settings.ErrInvalid
		}
	} else {
		if in.SecretValue != "" || validateValue(in.ValueType, in.Value, in.ValidationSchema) != nil || (len(in.DefaultValue) > 0 && validateValue(in.ValueType, in.DefaultValue, in.ValidationSchema) != nil) {
			return in, settings.ErrInvalid
		}
	}
	return in, nil
}

func isPlatformManagedConfig(in settings.ConfigInput) bool {
	return strings.EqualFold(in.ModuleCode, "iam") &&
		strings.EqualFold(in.ConfigGroup, "security") &&
		strings.EqualFold(in.ConfigKey, "interactive_captcha.type")
}

func validateValue(kind string, raw json.RawMessage, schemaRaw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if !json.Valid(raw) {
		return settings.ErrInvalid
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return settings.ErrInvalid
	}
	switch kind {
	case "string", "datetime":
		if _, ok := value.(string); !ok {
			return settings.ErrInvalid
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return settings.ErrInvalid
		}
	case "integer":
		n, ok := value.(float64)
		if !ok || n != float64(int64(n)) {
			return settings.ErrInvalid
		}
	case "decimal":
		if _, ok := value.(float64); !ok {
			return settings.ErrInvalid
		}
	case "json":
	}
	var schema map[string]any
	if json.Unmarshal(schemaRaw, &schema) != nil {
		return settings.ErrInvalid
	}
	if enum, ok := schema["enum"].([]any); ok {
		matched := false
		for _, item := range enum {
			if string(mustJSON(item)) == string(raw) {
				matched = true
			}
		}
		if !matched {
			return settings.ErrInvalid
		}
	}
	if text, ok := value.(string); ok {
		if min, ok := schema["minLength"].(float64); ok && len([]rune(text)) < int(min) {
			return settings.ErrInvalid
		}
		if max, ok := schema["maxLength"].(float64); ok && len([]rune(text)) > int(max) {
			return settings.ErrInvalid
		}
		if pattern, ok := schema["pattern"].(string); ok {
			re, err := regexp.Compile(pattern)
			if err != nil || !re.MatchString(text) {
				return settings.ErrInvalid
			}
		}
	}
	if number, ok := value.(float64); ok {
		if min, ok := schema["minimum"].(float64); ok && number < min {
			return settings.ErrInvalid
		}
		if max, ok := schema["maximum"].(float64); ok && number > max {
			return settings.ErrInvalid
		}
	}
	return nil
}
func mustJSON(value any) []byte { out, _ := json.Marshal(value); return out }

func (s *Service) validateDictionaryConfig(ctx context.Context, tenantID uuid.UUID, in settings.ConfigInput) error {
	var schema map[string]any
	if json.Unmarshal(in.ValidationSchema, &schema) != nil {
		return settings.ErrInvalid
	}
	code, _ := schema["x-appkernia-dictionary"].(string)
	if code == "" || in.IsSecret || len(in.Value) == 0 || string(in.Value) == "null" {
		return nil
	}
	var selected string
	if json.Unmarshal(in.Value, &selected) != nil || selected == "" {
		return settings.ErrInvalid
	}
	dictionary, err := s.repo.ResolveDictionary(ctx, &tenantID, code, "zh-CN", false)
	if err != nil {
		return settings.ErrInvalid
	}
	for _, item := range dictionary.Items {
		if item.Value == selected {
			return nil
		}
	}
	return settings.ErrInvalid
}

func (s *Service) CreateConfig(ctx context.Context, token string, p settings.Principal, in settings.ConfigInput) (settings.ConfigItem, error) {
	auth, err := s.authorize(ctx, token, "sys.config.create")
	if err != nil {
		return settings.ConfigItem{}, err
	}
	in, err = normalizeConfig(in, true)
	if err != nil {
		return settings.ConfigItem{}, err
	}
	if isPlatformManagedConfig(in) {
		return settings.ConfigItem{}, settings.ErrInvalid
	}
	if err = s.validateDictionaryConfig(ctx, auth.Tenant.ID, in); err != nil {
		return settings.ConfigItem{}, err
	}
	var sealed []byte
	var keyVersion int32
	if in.IsSecret {
		if s.sealer == nil {
			return settings.ConfigItem{}, settings.ErrSecretUnavailable
		}
		sealed, keyVersion, err = s.sealer.Seal([]byte(in.SecretValue), auth.Tenant.ID.String())
		if err != nil {
			return settings.ConfigItem{}, settings.ErrSecretUnavailable
		}
		in.SecretValue = ""
	}
	return s.repo.CreateConfig(ctx, principal(auth, p), in, sealed, keyVersion)
}

func (s *Service) UpdateConfig(ctx context.Context, token string, p settings.Principal, id uuid.UUID, in settings.ConfigInput) (settings.ConfigItem, error) {
	auth, err := s.authorize(ctx, token, "sys.config.update")
	if err != nil {
		return settings.ConfigItem{}, err
	}
	if id == uuid.Nil || in.Version < 1 || in.SecretValue != "" {
		return settings.ConfigItem{}, settings.ErrInvalid
	}
	in, err = normalizeConfig(in, false)
	if err != nil {
		return settings.ConfigItem{}, err
	}
	if err = s.validateDictionaryConfig(ctx, auth.Tenant.ID, in); err != nil {
		return settings.ConfigItem{}, err
	}
	item, err := s.repo.UpdateConfig(ctx, s.configPrincipal(auth, p), id, in)
	if err != nil {
		return settings.ConfigItem{}, err
	}
	if item.TenantID == nil && s.canUpdateGlobalConfigs(auth) {
		item.IsLocked = false
	}
	return item, nil
}

func (s *Service) RotateSecret(ctx context.Context, token string, p settings.Principal, id uuid.UUID, version int32, secret string) (settings.ConfigItem, error) {
	auth, err := s.authorize(ctx, token, "sys.config.rotate_secret")
	if err != nil {
		return settings.ConfigItem{}, err
	}
	secret = strings.TrimSpace(secret)
	if id == uuid.Nil || version < 1 || secret == "" || len(secret) > 16384 || s.sealer == nil {
		return settings.ConfigItem{}, settings.ErrInvalid
	}
	sealed, keyVersion, err := s.sealer.Seal([]byte(secret), auth.Tenant.ID.String())
	if err != nil {
		return settings.ConfigItem{}, settings.ErrSecretUnavailable
	}
	return s.repo.RotateSecret(ctx, principal(auth, p), id, version, sealed, keyVersion)
}

func (s *Service) ListDictTypes(ctx context.Context, token string, f settings.PageFilter) (settings.DictTypePage, error) {
	auth, err := s.authorize(ctx, token, "sys.dictionary.read")
	if err != nil {
		return settings.DictTypePage{}, err
	}
	f, err = pageFilter(f)
	if err != nil {
		return settings.DictTypePage{}, err
	}
	return s.repo.ListDictTypes(ctx, auth.Tenant.ID, f)
}

func canonicalDictionaryLocale(locale string) string {
	locale = strings.TrimSpace(locale)
	if locale == "en-US" {
		return locale
	}
	return "zh-CN"
}

func (s *Service) ResolveAdminDictionary(ctx context.Context, token, code, locale string) (settings.ResolvedDictionary, error) {
	auth, err := s.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return settings.ResolvedDictionary{}, err
	}
	code = strings.TrimSpace(code)
	if !regexp.MustCompile(`^[a-z][a-z0-9._-]{1,159}$`).MatchString(code) {
		return settings.ResolvedDictionary{}, settings.ErrInvalid
	}
	return s.repo.ResolveDictionary(ctx, &auth.Tenant.ID, code, canonicalDictionaryLocale(locale), false)
}

func (s *Service) ResolvePublicDictionary(ctx context.Context, code, locale string) (settings.ResolvedDictionary, error) {
	code = strings.TrimSpace(code)
	if !regexp.MustCompile(`^[a-z][a-z0-9._-]{1,159}$`).MatchString(code) {
		return settings.ResolvedDictionary{}, settings.ErrInvalid
	}
	return s.repo.ResolveDictionary(ctx, nil, code, canonicalDictionaryLocale(locale), true)
}
func normalizeType(in settings.DictTypeInput) (settings.DictTypeInput, error) {
	in.Code, in.Name, in.Description, in.Status = strings.TrimSpace(in.Code), strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), strings.TrimSpace(in.Status)
	if in.Status == "" {
		in.Status = "active"
	}
	if len(in.Code) < 1 || len(in.Code) > 160 || len(in.Name) < 1 || len(in.Name) > 160 || len(in.Description) > 500 || !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`).MatchString(in.Code) || !oneOf(in.Status, "active", "disabled") {
		return in, settings.ErrInvalid
	}
	return in, nil
}
func (s *Service) CreateDictType(ctx context.Context, token string, p settings.Principal, in settings.DictTypeInput) (settings.DictType, error) {
	auth, err := s.authorize(ctx, token, "sys.dictionary.create")
	if err != nil {
		return settings.DictType{}, err
	}
	in, err = normalizeType(in)
	if err != nil {
		return settings.DictType{}, err
	}
	return s.repo.CreateDictType(ctx, principal(auth, p), in)
}
func (s *Service) UpdateDictType(ctx context.Context, token string, p settings.Principal, id uuid.UUID, in settings.DictTypeInput) (settings.DictType, error) {
	auth, err := s.authorize(ctx, token, "sys.dictionary.update")
	if err != nil {
		return settings.DictType{}, err
	}
	in, err = normalizeType(in)
	if err != nil || id == uuid.Nil {
		return settings.DictType{}, settings.ErrInvalid
	}
	return s.repo.UpdateDictType(ctx, principal(auth, p), id, in)
}
func normalizeItem(in settings.DictItemInput) (settings.DictItemInput, error) {
	in.ItemValue, in.Label, in.Color, in.CSSClass, in.Status = strings.TrimSpace(in.ItemValue), strings.TrimSpace(in.Label), strings.TrimSpace(in.Color), strings.TrimSpace(in.CSSClass), strings.TrimSpace(in.Status)
	if in.Status == "" {
		in.Status = "active"
	}
	if in.Locale != nil {
		v := strings.TrimSpace(*in.Locale)
		if !oneOf(v, "zh-CN", "en-US") {
			return in, settings.ErrInvalid
		}
		in.Locale = &v
	}
	if len(in.ItemValue) < 1 || len(in.ItemValue) > 255 || len(in.Label) < 1 || len(in.Label) > 255 || len(in.Color) > 64 || len(in.CSSClass) > 128 || !oneOf(in.Status, "active", "disabled") {
		return in, settings.ErrInvalid
	}
	if len(in.Extra) == 0 {
		in.Extra = json.RawMessage(`{}`)
	}
	if !json.Valid(in.Extra) {
		return in, settings.ErrInvalid
	}
	return in, nil
}
func (s *Service) ListDictItems(ctx context.Context, token string, typeID uuid.UUID, f settings.DictItemFilter) (settings.DictItemPage, error) {
	auth, err := s.authorize(ctx, token, "sys.dictionary.read")
	if err != nil {
		return settings.DictItemPage{}, err
	}
	f.Query, f.Locale, f.Status, f.Sort = strings.TrimSpace(f.Query), strings.TrimSpace(f.Locale), strings.TrimSpace(f.Status), strings.TrimSpace(f.Sort)
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Sort == "" {
		f.Sort = "sort_order"
	}
	if typeID == uuid.Nil || f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || !oneOf(f.Locale, "", "zh-CN", "en-US", "neutral") || !oneOf(f.Status, "", "active", "disabled") || !oneOf(f.Sort, "sort_order", "label", "updated_desc") {
		return settings.DictItemPage{}, settings.ErrInvalid
	}
	return s.repo.ListDictItems(ctx, auth.Tenant.ID, typeID, f)
}
func (s *Service) CreateDictItem(ctx context.Context, token string, p settings.Principal, typeID uuid.UUID, in settings.DictItemInput) (settings.DictItem, error) {
	auth, err := s.authorize(ctx, token, "sys.dictionary.create")
	if err != nil {
		return settings.DictItem{}, err
	}
	in, err = normalizeItem(in)
	if err != nil || typeID == uuid.Nil {
		return settings.DictItem{}, settings.ErrInvalid
	}
	return s.repo.CreateDictItem(ctx, principal(auth, p), typeID, in)
}
func (s *Service) UpdateDictItem(ctx context.Context, token string, p settings.Principal, id uuid.UUID, in settings.DictItemInput) (settings.DictItem, error) {
	auth, err := s.authorize(ctx, token, "sys.dictionary.update")
	if err != nil {
		return settings.DictItem{}, err
	}
	in, err = normalizeItem(in)
	if err != nil || id == uuid.Nil {
		return settings.DictItem{}, settings.ErrInvalid
	}
	return s.repo.UpdateDictItem(ctx, principal(auth, p), id, in)
}
func (s *Service) DeleteDictItem(ctx context.Context, token string, p settings.Principal, id uuid.UUID) error {
	auth, err := s.authorize(ctx, token, "sys.dictionary.delete")
	if err != nil {
		return err
	}
	if id == uuid.Nil {
		return settings.ErrInvalid
	}
	return s.repo.DeleteDictItem(ctx, principal(auth, p), id)
}
func oneOf(v string, choices ...string) bool {
	for _, x := range choices {
		if v == x {
			return true
		}
	}
	return false
}
