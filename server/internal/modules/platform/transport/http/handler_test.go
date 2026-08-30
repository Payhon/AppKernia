package http

import (
	"encoding/json"
	"testing"

	"github.com/appkernia/appkernia/server/internal/shared/i18n"
)

func TestAppPublicConfigIncludesOnlyRepositoryProvidedSettings(t *testing.T) {
	values := map[string]json.RawMessage{
		"core.ui.page_size": json.RawMessage(`20`),
	}
	config := appPublicConfigData(i18n.LocaleEnUS, values)
	if config["locale"] != i18n.LocaleEnUS || config["default_locale"] != i18n.LocaleZhCN {
		t.Fatalf("unexpected locale contract: %#v", config)
	}
	settings, ok := config["settings"].(map[string]json.RawMessage)
	if !ok || string(settings["core.ui.page_size"]) != "20" {
		t.Fatalf("unexpected public settings: %#v", config["settings"])
	}
}

func TestAdminPublicConfigFailsClosedAndPreservesLocale(t *testing.T) {
	config := adminPublicConfigData(i18n.LocaleEnUS, nil)
	if config["locale"] != i18n.LocaleEnUS || config["default_locale"] != i18n.LocaleZhCN {
		t.Fatalf("unexpected locale contract: %#v", config)
	}
	flags, ok := config["feature_flags"].(map[string]bool)
	if !ok {
		t.Fatalf("unexpected feature flag type: %T", config["feature_flags"])
	}
	for _, key := range []string{"admin_registration", "password_recovery", "oauth"} {
		if flags[key] {
			t.Fatalf("expected %s to fail closed", key)
		}
	}
}

func TestAdminPublicConfigEnablesOnlyKnownConfiguredFeatures(t *testing.T) {
	config := adminPublicConfigData(i18n.LocaleZhCN, map[string]bool{
		"admin_registration": true,
		"password_recovery":  true,
		"unknown":            true,
	})
	flags := config["feature_flags"].(map[string]bool)
	if !flags["admin_registration"] || !flags["password_recovery"] || flags["oauth"] {
		t.Fatalf("unexpected configured flags: %#v", flags)
	}
	if _, exists := flags["unknown"]; exists {
		t.Fatalf("unknown feature flag must not be exposed: %#v", flags)
	}
}

func TestPrometheusLabelEscapesUntrustedText(t *testing.T) {
	if got, want := prometheusLabel("provider\\\"\nvalue"), "provider\\\\\\\"\\nvalue"; got != want {
		t.Fatalf("label=%q want %q", got, want)
	}
}
