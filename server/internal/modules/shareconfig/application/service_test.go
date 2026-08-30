package application

import (
	"encoding/json"
	"testing"

	share "github.com/appkernia/appkernia/server/internal/modules/shareconfig/domain"
)

func validConfigInput(t *testing.T) share.ConfigInput {
	t.Helper()
	raw, err := json.Marshal(share.WechatPublicConfig{
		Android: share.PlatformIdentity{Enabled: true, PackageName: "com.appkernia.mobile", Signature: "AABBCCDDEEFF0011"},
		IOS:     share.PlatformIdentity{Enabled: true, BundleID: "com.appkernia.mobile", UniversalLink: "https://share.example.com/app/"},
		Harmony: share.PlatformIdentity{Enabled: true, BundleName: "com.appkernia.mobile"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return share.ConfigInput{Name: "Official WeChat", ProviderCode: "wechat", ExternalAppID: "wx1234567890abcdef", ConfigSchemaVersion: 1, PublicConfig: raw}
}

func TestNormalizeConfigActivation(t *testing.T) {
	input := validConfigInput(t)
	normalized, _, err := normalizeConfig(input, true)
	if err != nil {
		t.Fatalf("valid active configuration rejected: %v", err)
	}
	if normalized.Name != input.Name || !json.Valid(normalized.PublicConfig) {
		t.Fatal("configuration was not normalized")
	}

	var document map[string]any
	if err = json.Unmarshal(input.PublicConfig, &document); err != nil {
		t.Fatal(err)
	}
	document["unknown"] = true
	input.PublicConfig, _ = json.Marshal(document)
	if _, _, err = normalizeConfig(input, true); err == nil {
		t.Fatal("unknown provider field accepted")
	}
}

func TestNormalizeBindingRejectsUnsafeOriginAndDuplicateScenes(t *testing.T) {
	input := share.BindingInput{Scenes: []string{"session", "session"}, ShareOrigin: "https://share.example.com", FallbackMode: "system"}
	input.ShareConfigID[0] = 1
	if _, err := normalizeBinding("wechat", input); err == nil {
		t.Fatal("duplicate scenes accepted")
	}
	input.Scenes = []string{"session"}
	input.ShareOrigin = "https://127.0.0.1"
	if _, err := normalizeBinding("wechat", input); err == nil {
		t.Fatal("private share origin accepted")
	}
}

func TestPreflightNeverReturnsProviderIdentity(t *testing.T) {
	input := validConfigInput(t)
	config := share.Config{ProviderCode: "wechat", Status: "active", PublicConfig: input.PublicConfig}
	binding := share.BindingInput{Scenes: []string{"session", "timeline"}, ShareOrigin: "https://share.example.com", FallbackMode: "system"}
	result := preflightFor(config, binding)
	if !result.Ready || len(result.Issues) != 0 || len(result.Platforms) != 3 {
		t.Fatalf("unexpected preflight: %#v", result)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" || json.Valid(encoded) == false {
		t.Fatal("preflight is not serializable")
	}
}
