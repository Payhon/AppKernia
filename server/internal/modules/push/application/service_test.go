package application

import (
	"encoding/json"
	"testing"

	push "github.com/appkernia/appkernia/server/internal/modules/push/domain"
)

func TestNormalizeDeviceBuildBoundaries(t *testing.T) {
	tests := []struct {
		name string
		in   push.DeviceInput
		ok   bool
	}{
		{"apns", push.DeviceInput{Provider: "apns", Platform: "ios", BuildVariant: "ios", Token: "0123456789abcdef", Locale: "zh-CN"}, true},
		{"fcm", push.DeviceInput{Provider: "fcm", Platform: "android", BuildVariant: "android_google", Token: "0123456789abcdef", Locale: "en-US"}, true},
		{"china-cross-contamination", push.DeviceInput{Provider: "xiaomi", Platform: "android", BuildVariant: "android_google", Token: "0123456789abcdef", Locale: "zh-CN"}, false},
		{"legacy-custom-create", push.DeviceInput{Provider: "custom", Platform: "android", BuildVariant: "android_china", Token: "0123456789abcdef", Locale: "zh-CN"}, false},
		{"unsupported-locale", push.DeviceInput{Provider: "fcm", Platform: "android", BuildVariant: "android_google", Token: "0123456789abcdef", Locale: "fr-FR"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeDevice(test.in)
			if (err == nil) != test.ok {
				t.Fatalf("err=%v ok=%v", err, test.ok)
			}
		})
	}
}

func TestNormalizeProviderConfigRejectsUnknownFields(t *testing.T) {
	input := push.ProviderConfigInput{Environment: "production", Provider: push.ProviderAPNS, ConfigSchemaVersion: 1, PublicConfig: json.RawMessage(`{"team_id":"TEAM","key_id":"KEY","bundle_id":"com.app","apns_environment":"production"}`)}
	got, err := normalizeConfig(input)
	if err != nil || string(got.PublicConfig) == "" {
		t.Fatalf("valid APNs config rejected: %v", err)
	}
	input.PublicConfig = json.RawMessage(`{"team_id":"TEAM","key_id":"KEY","bundle_id":"com.app","apns_environment":"production","private_key_p8":"secret"}`)
	if _, err = normalizeConfig(input); err == nil {
		t.Fatal("secret or unknown public fields must be rejected")
	}
}

func TestNormalizeProviderConfigRejectsUnknownEnvironment(t *testing.T) {
	input := push.ProviderConfigInput{Environment: "prod", Provider: push.ProviderAPNS, ConfigSchemaVersion: 1, PublicConfig: json.RawMessage(`{"team_id":"TEAM","key_id":"KEY","bundle_id":"com.app","apns_environment":"production"}`)}
	if _, err := normalizeConfig(input); err == nil {
		t.Fatal("unknown environments must not silently fall back to development")
	}
}

func TestProviderCatalogContainsEveryLockedProvider(t *testing.T) {
	items := Catalog()
	if len(items) != len(push.Providers) {
		t.Fatalf("catalog=%d providers=%d", len(items), len(push.Providers))
	}
	for _, item := range items {
		if item.ConfigSchemaVersion != 1 || len(item.Platforms) == 0 || len(item.BuildVariants) == 0 || len(item.SecretFields) == 0 {
			t.Fatalf("incomplete catalog item: %+v", item)
		}
	}
}

func TestNormalizeProviderCategories(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		config   string
		valid    bool
	}{
		{"vivo", push.ProviderVivo, `{"app_id":"1","app_key":"key","package_name":"com.app","service_category":"ACCOUNT","operations_category":"MARKETING"}`, true},
		{"harmony", push.ProviderHarmony, `{"project_id":"1","client_id":"2","bundle_name":"com.app","service_category":"ACCOUNT","operations_category":"MARKETING"}`, true},
		{"lowercase-rejected", push.ProviderHarmony, `{"project_id":"1","client_id":"2","bundle_name":"com.app","service_category":"account","operations_category":"MARKETING"}`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeConfig(push.ProviderConfigInput{Environment: "production", Provider: test.provider, ConfigSchemaVersion: 1, PublicConfig: json.RawMessage(test.config)})
			if (err == nil) != test.valid {
				t.Fatalf("err=%v valid=%v", err, test.valid)
			}
		})
	}
}
