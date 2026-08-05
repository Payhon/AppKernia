package repository

import (
	"testing"

	settings "github.com/appkernia/appkernia/server/internal/modules/systemsettings/domain"
)

func TestCatalogConfigMarkerAndSemanticJSON(t *testing.T) {
	if !catalogConfig(settings.ConfigItem{ValidationSchema: []byte(`{"type":"string","x-appkernia-catalog":true}`)}) {
		t.Fatal("catalog marker was not detected")
	}
	if catalogConfig(settings.ConfigItem{ValidationSchema: []byte(`{"type":"string"}`)}) {
		t.Fatal("ordinary tenant setting was treated as a catalog item")
	}
	if !sameJSON([]byte(`{"enum":["local","s3"],"type":"string"}`), []byte(`{"type":"string","enum":["local","s3"]}`)) {
		t.Fatal("equivalent JSON metadata should compare equal")
	}
	if sameJSON([]byte(`"local"`), []byte(`"s3"`)) {
		t.Fatal("different catalog defaults should not compare equal")
	}
}
