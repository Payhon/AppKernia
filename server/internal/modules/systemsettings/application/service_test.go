package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	settings "github.com/appkernia/appkernia/server/internal/modules/systemsettings/domain"
	"github.com/google/uuid"
)

type fakeAuthenticator struct {
	auth iamdomain.AuthenticatedContext
}

func (f fakeAuthenticator) Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error) {
	return f.auth, nil
}

type fakeRepository struct {
	principal settings.Principal
	input     settings.ConfigInput
	sealed    []byte
	version   int32
}

func (*fakeRepository) ListPublicConfigs(context.Context) (map[string]json.RawMessage, error) {
	return map[string]json.RawMessage{}, nil
}
func (*fakeRepository) ListRegions(context.Context, settings.RegionFilter) ([]settings.Region, error) {
	return nil, nil
}
func (*fakeRepository) ListModules(context.Context, settings.ModuleFilter) ([]settings.Module, error) {
	return nil, nil
}

func (*fakeRepository) ListConfigs(context.Context, uuid.UUID, settings.PageFilter) (settings.ConfigPage, error) {
	return settings.ConfigPage{}, nil
}
func (f *fakeRepository) CreateConfig(_ context.Context, p settings.Principal, in settings.ConfigInput, sealed []byte, version int32) (settings.ConfigItem, error) {
	f.principal, f.input, f.sealed, f.version = p, in, sealed, version
	return settings.ConfigItem{ID: uuid.New(), IsSecret: in.IsSecret, SecretConfigured: len(sealed) > 0}, nil
}
func (*fakeRepository) UpdateConfig(context.Context, settings.Principal, uuid.UUID, settings.ConfigInput) (settings.ConfigItem, error) {
	return settings.ConfigItem{}, nil
}
func (*fakeRepository) RotateSecret(context.Context, settings.Principal, uuid.UUID, int32, []byte, int32) (settings.ConfigItem, error) {
	return settings.ConfigItem{}, nil
}
func (*fakeRepository) ListDictTypes(context.Context, uuid.UUID, settings.PageFilter) (settings.DictTypePage, error) {
	return settings.DictTypePage{}, nil
}
func (*fakeRepository) CreateDictType(context.Context, settings.Principal, settings.DictTypeInput) (settings.DictType, error) {
	return settings.DictType{}, nil
}
func (*fakeRepository) UpdateDictType(context.Context, settings.Principal, uuid.UUID, settings.DictTypeInput) (settings.DictType, error) {
	return settings.DictType{}, nil
}
func (*fakeRepository) ListDictItems(context.Context, uuid.UUID, uuid.UUID, settings.DictItemFilter) (settings.DictItemPage, error) {
	return settings.DictItemPage{}, nil
}
func (*fakeRepository) CreateDictItem(context.Context, settings.Principal, uuid.UUID, settings.DictItemInput) (settings.DictItem, error) {
	return settings.DictItem{}, nil
}
func (*fakeRepository) UpdateDictItem(context.Context, settings.Principal, uuid.UUID, settings.DictItemInput) (settings.DictItem, error) {
	return settings.DictItem{}, nil
}
func (*fakeRepository) DeleteDictItem(context.Context, settings.Principal, uuid.UUID) error {
	return nil
}

type fakeSealer struct{ aad string }

func (f *fakeSealer) Seal(plaintext []byte, aad string) ([]byte, int32, error) {
	f.aad = aad
	return append([]byte("sealed:"), plaintext...), 4, nil
}

func authContext(permissions ...string) iamdomain.AuthenticatedContext {
	return iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{User: iamdomain.User{ID: uuid.New()}, Tenant: iamdomain.Tenant{ID: uuid.New()}, Permissions: permissions}, SessionID: uuid.New()}
}

func TestServiceRequiresExactPermission(t *testing.T) {
	service := NewService(fakeAuthenticator{auth: authContext("sys.config.reader")}, &fakeRepository{}, &fakeSealer{})
	if _, err := service.ListConfigs(context.Background(), "token", settings.PageFilter{}); !errors.Is(err, settings.ErrForbidden) {
		t.Fatalf("prefix permission authorized list: %v", err)
	}
}

func TestCatalogRequiresExactPermissionsAndValidatesLargeTreeFilters(t *testing.T) {
	service := NewService(fakeAuthenticator{auth: authContext("sys.region.reader", "sys.module.reader")}, &fakeRepository{}, &fakeSealer{})
	if _, err := service.ListRegions(context.Background(), "token", settings.RegionFilter{}); !errors.Is(err, settings.ErrForbidden) {
		t.Fatalf("region prefix permission authorized: %v", err)
	}
	service = NewService(fakeAuthenticator{auth: authContext("sys.region.read", "sys.module.read")}, &fakeRepository{}, &fakeSealer{})
	tooLarge := int16(11)
	if _, err := service.ListRegions(context.Background(), "token", settings.RegionFilter{Level: &tooLarge}); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("invalid region level: %v", err)
	}
	if _, err := service.ListModules(context.Background(), "token", settings.ModuleFilter{Status: "installed"}); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("runtime module status accepted: %v", err)
	}
}

func TestServiceSealsSecretAndScopesPrincipal(t *testing.T) {
	auth := authContext("sys.config.create")
	repository := &fakeRepository{}
	sealer := &fakeSealer{}
	service := NewService(fakeAuthenticator{auth: auth}, repository, sealer)
	item, err := service.CreateConfig(context.Background(), "token", settings.Principal{RequestID: "request"}, settings.ConfigInput{ModuleCode: "core", ConfigGroup: "mail", ConfigKey: "mail.api_key", DisplayName: "Mail API key", ValueType: "string", IsSecret: true, SecretValue: "only-once", Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	if !item.SecretConfigured || repository.input.SecretValue != "" || string(repository.sealed) != "sealed:only-once" || repository.version != 4 {
		t.Fatalf("secret write = %#v, input=%#v", item, repository.input)
	}
	if sealer.aad != auth.Tenant.ID.String() || repository.principal.TenantID != auth.Tenant.ID || repository.principal.UserID != auth.User.ID || repository.principal.SessionID != auth.SessionID {
		t.Fatalf("scoped principal=%#v aad=%q", repository.principal, sealer.aad)
	}
}

func TestServiceValidatesTypedValuesSecretPolicyAndLocales(t *testing.T) {
	service := NewService(fakeAuthenticator{auth: authContext("sys.config.create", "sys.dictionary.create")}, &fakeRepository{}, &fakeSealer{})
	base := settings.ConfigInput{ModuleCode: "core", ConfigGroup: "limits", ConfigKey: "core.limit", DisplayName: "Limit", ValueType: "integer", Status: "active"}
	base.Value = []byte(`"8"`)
	if _, err := service.CreateConfig(context.Background(), "token", settings.Principal{}, base); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("string integer error=%v", err)
	}
	base.Value, base.ValidationSchema = []byte(`0`), []byte(`{"minimum":1}`)
	if _, err := service.CreateConfig(context.Background(), "token", settings.Principal{}, base); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("minimum error=%v", err)
	}
	base.Value, base.ValidationSchema, base.IsSecret, base.IsPublic, base.SecretValue = nil, nil, true, true, "secret"
	if _, err := service.CreateConfig(context.Background(), "token", settings.Principal{}, base); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("public secret error=%v", err)
	}
	badLocale := "fr-FR"
	if _, err := service.CreateDictItem(context.Background(), "token", settings.Principal{}, uuid.New(), settings.DictItemInput{ItemValue: "ready", Label: "Ready", Locale: &badLocale, Extra: []byte(`{}`), Status: "active"}); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("non-contract locale error=%v", err)
	}
}
