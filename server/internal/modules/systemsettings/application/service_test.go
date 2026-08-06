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
	principal         settings.Principal
	input             settings.ConfigInput
	regionCreateInput settings.RegionCreateInput
	regionUpdateInput settings.RegionUpdateInput
	regionCode        string
	sealed            []byte
	version           int32
}

func (f *fakeRepository) ResolveDictionary(_ context.Context, _ *uuid.UUID, code, locale string, _ bool) (settings.ResolvedDictionary, error) {
	return settings.ResolvedDictionary{Code: code, Locale: locale, ExtensionPolicy: "open", Items: []settings.DictionaryOption{}}, nil
}

func (*fakeRepository) ListPublicConfigs(context.Context) (map[string]json.RawMessage, error) {
	return map[string]json.RawMessage{}, nil
}
func (*fakeRepository) ListRegions(context.Context, settings.RegionFilter) ([]settings.Region, error) {
	return nil, nil
}
func (f *fakeRepository) CreateRegion(_ context.Context, p settings.Principal, in settings.RegionCreateInput) (settings.Region, error) {
	f.principal, f.regionCreateInput = p, in
	return settings.Region{Code: in.Code, ParentCode: &in.ParentCode, Name: in.Name, Version: 1}, nil
}
func (f *fakeRepository) UpdateRegion(_ context.Context, p settings.Principal, code string, in settings.RegionUpdateInput) (settings.Region, error) {
	f.principal, f.regionCode, f.regionUpdateInput = p, code, in
	return settings.Region{Code: code, Name: in.Name, Version: in.Version + 1}, nil
}
func (f *fakeRepository) DeleteRegion(_ context.Context, p settings.Principal, code string) error {
	f.principal, f.regionCode = p, code
	return nil
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
	service := NewService(fakeAuthenticator{auth: authContext("sys.region.reader")}, &fakeRepository{}, &fakeSealer{})
	if _, err := service.ListRegions(context.Background(), "token", settings.RegionFilter{}); !errors.Is(err, settings.ErrForbidden) {
		t.Fatalf("region prefix permission authorized: %v", err)
	}
	service = NewService(fakeAuthenticator{auth: authContext("sys.region.read")}, &fakeRepository{}, &fakeSealer{})
	tooLarge := int16(11)
	if _, err := service.ListRegions(context.Background(), "token", settings.RegionFilter{Level: &tooLarge}); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("invalid region level: %v", err)
	}
}

func TestRegionWritesRequireExactPermissionsAndValidateImmutableHierarchyInputs(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(fakeAuthenticator{auth: authContext("sys.region.creator")}, repository, &fakeSealer{})
	create := settings.RegionCreateInput{Code: "990100", ParentCode: "990000", Name: "测试市", FullName: "测试省 / 测试市", Status: "active"}
	if _, err := service.CreateRegion(context.Background(), "token", settings.Principal{}, create); !errors.Is(err, settings.ErrForbidden) {
		t.Fatalf("prefix create permission authorized: %v", err)
	}

	service = NewService(fakeAuthenticator{auth: authContext("sys.region.create")}, repository, &fakeSealer{})
	created, err := service.CreateRegion(context.Background(), "token", settings.Principal{RequestID: "region-create"}, create)
	if err != nil || created.Code != create.Code || repository.principal.TenantID == uuid.Nil || repository.regionCreateInput.ParentCode != create.ParentCode {
		t.Fatalf("create region=%+v input=%+v principal=%+v err=%v", created, repository.regionCreateInput, repository.principal, err)
	}
	invalid := create
	invalid.ParentCode = invalid.Code
	if _, err = service.CreateRegion(context.Background(), "token", settings.Principal{}, invalid); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("self-parent create error=%v", err)
	}

	service = NewService(fakeAuthenticator{auth: authContext("sys.region.update")}, repository, &fakeSealer{})
	update := settings.RegionUpdateInput{Name: "测试市（新）", FullName: "测试省 / 测试市（新）", Status: "active", Version: 1}
	updated, err := service.UpdateRegion(context.Background(), "token", settings.Principal{}, create.Code, update)
	if err != nil || updated.Version != 2 || repository.regionCode != create.Code {
		t.Fatalf("update region=%+v code=%s err=%v", updated, repository.regionCode, err)
	}
	update.Version = 0
	if _, err = service.UpdateRegion(context.Background(), "token", settings.Principal{}, create.Code, update); !errors.Is(err, settings.ErrInvalid) {
		t.Fatalf("zero version update error=%v", err)
	}

	service = NewService(fakeAuthenticator{auth: authContext("sys.region.delete")}, repository, &fakeSealer{})
	if err = service.DeleteRegion(context.Background(), "token", settings.Principal{}, create.Code); err != nil || repository.regionCode != create.Code {
		t.Fatalf("delete code=%s err=%v", repository.regionCode, err)
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
