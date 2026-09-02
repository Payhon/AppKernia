package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRegisteredCatalogIsStableAndSafe(t *testing.T) {
	catalog := RegisteredCatalog()
	if len(catalog.Items) != 4 {
		t.Fatalf("catalog items = %d", len(catalog.Items))
	}
	for index, code := range ProviderCodes {
		if catalog.Items[index].ProviderCode != code {
			t.Fatalf("catalog[%d] = %q, want %q", index, catalog.Items[index].ProviderCode, code)
		}
		if !strings.HasPrefix(catalog.Items[index].HelpURL, "https://") {
			t.Fatalf("provider %s help URL is not HTTPS", code)
		}
	}
	if catalog.Items[3].RequiresSecret {
		t.Fatal("Google native ID-token verification must not request a client secret")
	}
}

func TestBuildConfigHashUsesTypedCanonicalJSON(t *testing.T) {
	one, err := BuildConfigHash(ProviderGitHub, "client-123", json.RawMessage(`{"app_return_uri":"https://login.example.test/oauth/callback"}`))
	if err != nil {
		t.Fatal(err)
	}
	two, err := BuildConfigHash(ProviderGitHub, "client-123", json.RawMessage("{ \"app_return_uri\" : \"https://login.example.test/oauth/callback\" }"))
	if err != nil {
		t.Fatal(err)
	}
	if one != two || len(one) != 64 {
		t.Fatalf("canonical hashes differ: %q %q", one, two)
	}
	if _, err = BuildConfigHash(ProviderGitHub, "client-123", json.RawMessage(`{"app_return_uri":"https://login.example.test/oauth/callback","endpoint":"https://evil.test"}`)); err == nil {
		t.Fatal("unknown provider endpoint field was accepted")
	}
}

func TestAppleOperationalSigningMetadataDoesNotChangeBuildHash(t *testing.T) {
	one, err := BuildConfigHash(ProviderApple, "com.example.app", json.RawMessage(`{"team_id":"AAAAAAAAAA","key_id":"BBBBBBBBBB"}`))
	if err != nil {
		t.Fatal(err)
	}
	two, err := BuildConfigHash(ProviderApple, "com.example.app", json.RawMessage(`{"team_id":"CCCCCCCCCC","key_id":"DDDDDDDDDD"}`))
	if err != nil {
		t.Fatal(err)
	}
	if one != two {
		t.Fatalf("Apple signing metadata changed native build hash: %s != %s", one, two)
	}
}

func TestNormalizeConfigAndSecretsFailClosed(t *testing.T) {
	google := json.RawMessage(`{"android_package_name":"com.example.app","android_certificate_sha256":["AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA"]}`)
	if _, _, err := NormalizeConfig(ProviderGoogle, "123-example.apps.googleusercontent.com", google, true); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ValidateSecrets(ProviderGoogle, map[string]string{"client_secret": "forbidden"}); err == nil {
		t.Fatal("Google client secret was accepted")
	}
	if _, _, _, err := ValidateSecrets(ProviderApple, map[string]string{"private_key_p8": "not a key"}); err == nil {
		t.Fatal("invalid Apple p8 was accepted")
	}
}

func TestConfigSupportsOnlyEnabledWechatTargets(t *testing.T) {
	raw := json.RawMessage(`{"android":{"enabled":false},"ios":{"enabled":true,"bundle_id":"com.example.app","universal_link":"https://login.example.test/app/"},"harmony":{"enabled":false}}`)
	if !ConfigSupportsTarget(ProviderWechat, raw, "ios", "ios") {
		t.Fatal("enabled WeChat iOS target was rejected")
	}
	if ConfigSupportsTarget(ProviderWechat, raw, "android", "android_china") || ConfigSupportsTarget(ProviderWechat, raw, "harmony", "harmony") {
		t.Fatal("disabled WeChat target was accepted")
	}
}
