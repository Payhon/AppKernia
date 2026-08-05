package seed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfigCatalogRejectsSecretPlaintextAndAcceptsRepositoryCatalog(t *testing.T) {
	catalogPath := filepath.Join("..", "..", "..", "blueprint", "backend", "spec", "core-configs.json")
	catalog, err := readConfigCatalog(catalogPath)
	if err != nil || len(catalog.Categories) != 9 || len(catalog.Items) < 50 {
		t.Fatalf("repository catalog categories=%d items=%d err=%v", len(catalog.Categories), len(catalog.Items), err)
	}
	invalid := `{"version":1,"categories":[{"module_code":"x","config_group":"y","name_key":"n","description_key":"d"}],"items":[{"module_code":"x","config_group":"y","config_key":"secret","display_name":"Secret","value_type":"string","value":"plaintext","is_secret":true,"validation_schema":{},"status":"active"}]}`
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err = os.WriteFile(path, []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = readConfigCatalog(path); err == nil {
		t.Fatal("secret plaintext must be rejected")
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
