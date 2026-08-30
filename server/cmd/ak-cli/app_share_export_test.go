package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	share "github.com/appkernia/appkernia/server/internal/modules/shareconfig/domain"
)

func TestValidateNativeShareIdentity(t *testing.T) {
	config := share.WechatPublicConfig{
		Android: share.PlatformIdentity{Enabled: true, PackageName: "com.appkernia.mobile", Signature: "AA:BB:CC:DD:EE:FF:00:11"},
		IOS:     share.PlatformIdentity{Enabled: true, BundleID: "com.appkernia.mobile", UniversalLink: "https://share.example.com/app/"},
		Harmony: share.PlatformIdentity{Enabled: true, BundleName: "com.appkernia.mobile"},
	}
	valid := nativeShareIdentity{AndroidPackage: "com.appkernia.mobile", AndroidSignature: "aabbccddeeff0011", IOSBundleID: "com.appkernia.mobile", HarmonyBundleName: "com.appkernia.mobile"}
	if err := validateNativeShareIdentity(config, valid); err != nil {
		t.Fatalf("valid identity rejected: %v", err)
	}
	valid.IOSBundleID = "com.other.app"
	if err := validateNativeShareIdentity(config, valid); err == nil {
		t.Fatal("mismatched iOS identity accepted")
	}
}

func TestRenderShareManifestPreservesUnrelatedConfiguration(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "manifest.json")
	input := []byte(`{"name":"App","unrelated":{"keep":true},"app-android":{"distribute":{"modules":{"uni-video":{}}}},"app-ios":{"distribute":{"modules":{}}},"app-harmony":{"distribute":{"modules":{}}}}`)
	if err := os.WriteFile(path, input, 0o600); err != nil {
		t.Fatal(err)
	}
	record := shareExportRecord{ExternalAppID: "wx1234567890abcdef", PublicConfig: share.WechatPublicConfig{
		Android: share.PlatformIdentity{Enabled: true},
		IOS:     share.PlatformIdentity{Enabled: true, UniversalLink: "https://share.example.com/app/"},
		Harmony: share.PlatformIdentity{Enabled: true},
	}}
	first, err := renderShareManifest(path, record)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), `"unrelated": {`) || !strings.Contains(string(first), `"uni-video": {}`) || strings.Count(string(first), `"appid": "wx1234567890abcdef"`) != 3 {
		t.Fatalf("unexpected manifest:\n%s", first)
	}
	if err = os.WriteFile(path, first, 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := renderShareManifest(path, record)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("share manifest export is not idempotent")
	}
}

func TestRenderShareManifestRemovesOnlyWechatProvider(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "manifest.json")
	input := []byte(`{"app-android":{"distribute":{"modules":{"uni-share":{"weixin":{"appid":"old"},"future-provider":{"clientId":"keep"}}}}}}`)
	if err := os.WriteFile(path, input, 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := renderShareManifest(path, shareExportRecord{ExternalAppID: "wx1234567890abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), `"weixin"`) || !strings.Contains(string(out), `"future-provider"`) {
		t.Fatalf("provider cleanup changed unrelated settings:\n%s", out)
	}
}

func TestRenderAssociatedDomainsMergesAndIsIdempotent(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "UniApp.entitlements")
	input := []byte("<?xml version=\"1.0\"?><plist><dict><key>aps-environment</key><string>development</string></dict></plist>")
	if err := os.WriteFile(path, input, 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := renderAssociatedDomains(path, "https://share.example.com:8443/app/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), "aps-environment") || !strings.Contains(string(first), "applinks:share.example.com") {
		t.Fatalf("unrelated entitlement was lost: %s", first)
	}
	if err = os.WriteFile(path, first, 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := renderAssociatedDomains(path, "https://share.example.com/app/")
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("associated domains merge is not idempotent")
	}
}
