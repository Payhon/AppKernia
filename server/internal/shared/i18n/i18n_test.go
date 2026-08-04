package i18n

import "testing"

func TestNormalizeAliases(t *testing.T) {
	cases := map[string]Locale{
		"zh-Hans": LocaleZhCN,
		"zh_CN":   LocaleZhCN,
		"en":      LocaleEnUS,
		"fr-FR":   LocaleZhCN,
	}
	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveAcceptLanguageQuality(t *testing.T) {
	if got := ResolveAcceptLanguage("en-US;q=0.9, zh-CN;q=0.5"); got != LocaleEnUS {
		t.Fatalf("ResolveAcceptLanguage() = %q", got)
	}
}

func TestCatalogParityAndFallback(t *testing.T) {
	catalog, err := LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	if got := catalog.Translate(LocaleEnUS, "validation.required", map[string]string{"name": "Email"}); got != "Enter Email" {
		t.Fatalf("Translate() = %q", got)
	}
	if got := catalog.Translate(Locale("fr-FR"), "common.actions.save", nil); got != "保存" {
		t.Fatalf("fallback Translate() = %q", got)
	}
}
