package application

import (
	"reflect"
	"testing"
)

func TestNormalizeScannerConfigInput(t *testing.T) {
	input, err := normalizeScannerConfigInput(ScannerConfigInput{
		WebViewEnabled: true,
		AllowedHostPatterns: []string{
			"Example.COM.",
			"*.Sub.Example.com",
			"example.com",
		},
	})
	if err != nil {
		t.Fatalf("normalize valid scanner config: %v", err)
	}
	want := []string{"*.sub.example.com", "example.com"}
	if !reflect.DeepEqual(input.AllowedHostPatterns, want) {
		t.Fatalf("patterns=%v want=%v", input.AllowedHostPatterns, want)
	}
}

func TestNormalizeScannerHostPatternRejectsUnsafeValues(t *testing.T) {
	values := []string{
		"https://example.com",
		"example.com/path",
		"user@example.com",
		"example.com:8443",
		"127.0.0.1",
		"localhost",
		"*.com",
		"*.co.uk",
		"*.github.io",
		"co.uk",
		"*.localhost",
		"例子.测试",
		"-bad.example.com",
	}
	for _, value := range values {
		if _, err := normalizeScannerHostPattern(value); err == nil {
			t.Fatalf("unsafe pattern accepted: %q", value)
		}
	}
}

func TestNormalizeScannerHostPatternAcceptsExactWildcardAndPunycode(t *testing.T) {
	tests := map[string]string{
		" EXAMPLE.com. ":                   "example.com",
		"*.Sub.Example.COM.":               "*.sub.example.com",
		"xn--fsqu00a.xn--0zwm56d":          "xn--fsqu00a.xn--0zwm56d",
		"*.xn--fsqu00a.xn--0zwm56d":        "*.xn--fsqu00a.xn--0zwm56d",
		"deep.service.example-company.com": "deep.service.example-company.com",
	}
	for input, want := range tests {
		got, err := normalizeScannerHostPattern(input)
		if err != nil || got != want {
			t.Fatalf("normalize %q = %q, %v; want %q", input, got, err, want)
		}
	}
}

func TestNormalizeScannerConfigRequiresHostsWhenEnabled(t *testing.T) {
	if _, err := normalizeScannerConfigInput(ScannerConfigInput{WebViewEnabled: true}); err == nil {
		t.Fatal("enabled scanner webview accepted without allowed hosts")
	}
}

func TestNormalizeScannerConfigLimitsAndRetainsHostsWhenDisabled(t *testing.T) {
	patterns := make([]string, maxScannerHostPatterns+1)
	for index := range patterns {
		patterns[index] = "host" + string(rune('a'+index%26)) + ".example.com"
	}
	if _, err := normalizeScannerConfigInput(ScannerConfigInput{AllowedHostPatterns: patterns}); err == nil {
		t.Fatal("scanner config accepted more than 100 input rows")
	}
	disabled, err := normalizeScannerConfigInput(ScannerConfigInput{AllowedHostPatterns: []string{"*.example.com"}})
	if err != nil || !reflect.DeepEqual(disabled.AllowedHostPatterns, []string{"*.example.com"}) {
		t.Fatalf("disabled scanner config did not retain hosts: %#v err=%v", disabled, err)
	}
}
