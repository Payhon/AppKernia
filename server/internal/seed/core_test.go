package seed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadDictionaryCatalogIncludesFixedSystemLanguages(t *testing.T) {
	catalogPath := filepath.Join("..", "..", "..", "blueprint", "backend", "spec", "core-dictionaries.json")
	catalog, err := readDictionaryCatalog(catalogPath)
	if err != nil {
		t.Fatal(err)
	}

	foundType := false
	labels := map[string]string{}
	for _, definition := range catalog.Types {
		if definition.Code != "system.language" {
			continue
		}
		foundType = true
		if definition.Visibility != "internal" || definition.ExtensionPolicy != "fixed" || definition.Status != "active" {
			t.Fatalf("unexpected system language type: %+v", definition)
		}
	}
	for _, item := range catalog.Items {
		if item.TypeCode == "system.language" {
			labels[item.Value+"/"+item.Locale] = item.Label
		}
	}
	if !foundType {
		t.Fatal("system.language type is missing")
	}
	want := map[string]string{
		"zh-CN/zh-CN": "简体中文",
		"zh-CN/en-US": "Simplified Chinese",
		"en-US/zh-CN": "English",
		"en-US/en-US": "English",
	}
	if !reflect.DeepEqual(labels, want) {
		t.Fatalf("system language labels=%v want=%v", labels, want)
	}
}

func TestReadModuleCatalogAcceptsExactRepositoryCatalog(t *testing.T) {
	catalogPath := filepath.Join("..", "..", "..", "blueprint", "backend", "spec", "core-modules.json")
	catalog, err := readModuleCatalog(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"iam", "org", "sys", "storage", "notify", "jobs", "audit", "ops"}
	if len(catalog.Modules) != len(want) {
		t.Fatalf("modules=%d want=%d", len(catalog.Modules), len(want))
	}
	for index, code := range want {
		if catalog.Modules[index].Code != code {
			t.Fatalf("module[%d]=%q want=%q", index, catalog.Modules[index].Code, code)
		}
	}
}

func TestReadModuleCatalogRejectsDuplicateAndUncompiledCapability(t *testing.T) {
	cases := []struct {
		name    string
		modules []moduleDefinition
	}{
		{
			name: "duplicate code",
			modules: []moduleDefinition{
				validTestModule("iam", json.RawMessage(`{"users":true}`)),
				validTestModule("iam", json.RawMessage(`{"roles":true}`)),
			},
		},
		{
			name: "capability is not compiled",
			modules: []moduleDefinition{
				validTestModule("ops", json.RawMessage(`{"runtime_summary":true,"plugin_upload":false}`)),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invalid-modules.json")
			raw, err := json.Marshal(moduleCatalog{Version: 1, Modules: tc.modules})
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err = readModuleCatalog(path); err == nil {
				t.Fatal("invalid module catalog must be rejected")
			}
		})
	}
}

func validTestModule(code string, capabilities json.RawMessage) moduleDefinition {
	return moduleDefinition{
		Code: code, Name: code, NameKey: "ops.modules." + code + ".name",
		Description: code + " module", DescriptionKey: "ops.modules." + code + ".description",
		Capabilities: capabilities, Status: "enabled",
	}
}

func TestReadConfigCatalogRejectsSecretPlaintextAndAcceptsRepositoryCatalog(t *testing.T) {
	catalogPath := filepath.Join("..", "..", "..", "blueprint", "backend", "spec", "core-configs.json")
	catalog, err := readConfigCatalog(catalogPath)
	if err != nil || len(catalog.Categories) != 10 || len(catalog.Items) < 50 {
		t.Fatalf("repository catalog categories=%d items=%d err=%v", len(catalog.Categories), len(catalog.Items), err)
	}
	foundCaptcha := false
	for _, item := range catalog.Items {
		if item.ModuleCode == "iam" && item.ConfigGroup == "security" && item.ConfigKey == "admin.login_captcha.type" {
			foundCaptcha = item.Scope == "global" && !item.IsPublic && string(item.Value) == `"slide"` && string(item.DefaultValue) == `"slide"` && string(item.ValidationSchema) == `{"enum":["click","slide","drag","rotate"]}`
		}
		if item.ConfigKey == "site.name" && item.Scope != "tenant" {
			t.Fatalf("omitted catalog scope must default to tenant, got %q", item.Scope)
		}
	}
	if !foundCaptcha {
		t.Fatal("global Admin login CAPTCHA catalog item is missing or invalid")
	}
	invalid := `{"version":1,"categories":[{"module_code":"x","config_group":"y","name_key":"n","description_key":"d"}],"items":[{"module_code":"x","config_group":"y","config_key":"secret","display_name":"Secret","value_type":"string","value":"plaintext","is_secret":true,"validation_schema":{},"status":"active"}]}`
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err = os.WriteFile(path, []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = readConfigCatalog(path); err == nil {
		t.Fatal("secret plaintext must be rejected")
	}
	invalidScope := `{"version":1,"categories":[{"module_code":"x","config_group":"y","name_key":"n","description_key":"d"}],"items":[{"scope":"application","module_code":"x","config_group":"y","config_key":"key","display_name":"Key","value_type":"string","value":"value","is_secret":false,"validation_schema":{},"status":"active"}]}`
	if err = os.WriteFile(path, []byte(invalidScope), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = readConfigCatalog(path); err == nil {
		t.Fatal("unsupported config scope must be rejected")
	}
}

func TestReadRegionCatalogAcceptsRepositoryCatalog(t *testing.T) {
	catalogPath := filepath.Join("..", "..", "..", "blueprint", "backend", "spec", "core-regions.json")
	catalog, err := readRegionCatalog(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Source.Project != "HotGo" || catalog.Source.Commit != "c6191f7126c0ece4f4357014684f479836643822" {
		t.Fatalf("unexpected source: %+v", catalog.Source)
	}
	levels := map[int16]int{}
	for _, region := range catalog.Regions {
		levels[region.Level]++
	}
	if len(catalog.Regions) != 3663 || levels[0] != 34 || levels[1] != 357 || levels[2] != 3272 {
		t.Fatalf("unexpected catalog shape: regions=%d levels=%v", len(catalog.Regions), levels)
	}
}

func TestReadRegionCatalogRejectsUnorderedParentAndInvalidCoordinate(t *testing.T) {
	source := regionCatalogSource{Project: "test", Commit: "test", License: "MIT"}
	fullName := "Root / Child"
	parentCode := "root"
	longitude := "181"
	cases := []struct {
		name    string
		regions []regionDefinition
	}{
		{
			name: "unordered parent",
			regions: []regionDefinition{{
				Code: "child", ParentCode: &parentCode, Level: 1, Name: "Child",
				FullName: &fullName, Status: "active", Metadata: json.RawMessage(`{}`),
			}},
		},
		{
			name: "coordinate outside range",
			regions: []regionDefinition{{
				Code: "root", Level: 0, Name: "Root", FullName: &fullName,
				Longitude: &longitude, Status: "active", Metadata: json.RawMessage(`{}`),
			}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invalid-regions.json")
			raw, err := json.Marshal(regionCatalog{Version: 1, Source: source, Regions: tc.regions})
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err = readRegionCatalog(path); err == nil {
				t.Fatal("invalid region catalog must be rejected")
			}
		})
	}
}
