package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	loginprovider "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/google/uuid"
)

func TestPrepareLoginProviderExportUsesSharedHashAndNoSecrets(t *testing.T) {
	appID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	records := testLoginProviderRecords(appID)
	plan, err := prepareLoginProviderExport(appID, records)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Snapshot.Providers) != 4 || strings.Join(plan.Snapshot.BuildVariants, ",") != "android_china,android_google,harmony,ios" {
		t.Fatalf("unexpected provider snapshot: %#v", plan.Snapshot)
	}
	expected, err := loginprovider.BuildConfigHash(records[0].ProviderCode, records[0].ExternalClientID, records[0].PublicConfig)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Snapshot.Providers[0].BuildConfigHash != expected {
		t.Fatal("exporter hash drifted from the runtime hash")
	}
	encoded, err := marshalGeneratedJSON(plan.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(encoded))
	for _, forbidden := range []string{"client_secret", "private_key", "app_secret", "id_token", "access_token"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("generated snapshot contains secret field %q", forbidden)
		}
	}
	var snapshot map[string]any
	if err = json.Unmarshal(encoded, &snapshot); err != nil {
		t.Fatal(err)
	}
	providers := snapshot["providers"].([]any)
	wechat := providers[0].(map[string]any)["build_config"].(map[string]any)
	if wechat["app_id"] != records[0].ExternalClientID || wechat["android_package_name"] != "com.appkernia.mobile" {
		t.Fatalf("exported snapshot did not use the runtime build projection: %#v", wechat)
	}
	if _, nested := wechat["android"]; nested {
		t.Fatalf("exported snapshot leaked the storage shape: %#v", wechat)
	}
	identity := nativeLoginProviderIdentity{
		AndroidPackage: "com.appkernia.mobile", AndroidSignature: "aabbccdd",
		AndroidCertificateSHA256: "aa:bb:cc:dd", IOSBundleID: "com.appkernia.mobile",
		HarmonyBundleName: "com.appkernia.mobile",
	}
	if err = validateNativeLoginProviderIdentity(plan, identity); err != nil {
		t.Fatalf("valid native identity rejected: %v", err)
	}
	identity.AndroidCertificateSHA256 = "11:22:33:44"
	if err = validateNativeLoginProviderIdentity(plan, identity); err == nil {
		t.Fatal("mismatched Google certificate accepted")
	}
}

func TestRenderLoginProviderNativeFilesIsIdempotent(t *testing.T) {
	appID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	plan, err := prepareLoginProviderExport(appID, testLoginProviderRecords(appID))
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	manifestPath := filepath.Join(directory, "manifest.json")
	manifestInput := []byte(`{"name":"keep","app-android":{"distribute":{"modules":{"uni-video":{}}}},"app-ios":{"distribute":{"modules":{"uni-share":{"weixin":{"universalLink":"https://old.example.com/share/"}}}}},"app-harmony":{"distribute":{"modules":{}}}}`)
	if err = os.WriteFile(manifestPath, manifestInput, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := renderLoginProviderManifest(manifestPath, plan)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(manifest), `"uni-oauth"`) != 3 || !strings.Contains(string(manifest), `"uni-video"`) || !strings.Contains(string(manifest), `"uni-share"`) {
		t.Fatalf("unexpected login provider manifest:\n%s", manifest)
	}

	entitlementsPath := filepath.Join(directory, "UniApp.entitlements")
	entitlementsInput := []byte("<?xml version=\"1.0\"?><plist><dict>\n\t<key>com.apple.developer.associated-domains</key>\n\t<array>\n\t\t<string>applinks:old.example.com</string>\n\t\t<string>applinks:remove.example.com</string>\n\t</array>\n</dict></plist>\n")
	if err = os.WriteFile(entitlementsPath, entitlementsInput, 0o600); err != nil {
		t.Fatal(err)
	}
	previous := loginProviderSnapshot{
		IOSAssociatedDomains: []string{"applinks:old.example.com", "applinks:remove.example.com"},
		Providers:            []loginProviderSnapshotProvider{{ProviderCode: loginprovider.ProviderApple}},
	}
	entitlements, write, err := renderLoginProviderEntitlements(entitlementsPath, previous, plan.Snapshot, shareAssociatedDomains(manifest))
	if err != nil || !write {
		t.Fatalf("render entitlements: write=%t err=%v", write, err)
	}
	text := string(entitlements)
	for _, expected := range []string{"applinks:old.example.com", "applinks:login.example.com", "applinks:github.example.com", "com.apple.developer.applesignin", "<string>Default</string>"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing entitlement %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "remove.example.com") {
		t.Fatalf("stale login domain was retained:\n%s", text)
	}
	if err = os.WriteFile(entitlementsPath, entitlements, 0o600); err != nil {
		t.Fatal(err)
	}
	second, _, err := renderLoginProviderEntitlements(entitlementsPath, plan.Snapshot, plan.Snapshot, shareAssociatedDomains(manifest))
	if err != nil || string(second) != string(entitlements) {
		t.Fatal("iOS entitlement export is not idempotent")
	}

	androidPath := filepath.Join(directory, "AndroidManifest.xml")
	androidInput := []byte("<?xml version=\"1.0\"?><manifest xmlns:android=\"http://schemas.android.com/apk/res/android\"><application android:label=\"Keep\" /></manifest>\n")
	if err = os.WriteFile(androidPath, androidInput, 0o600); err != nil {
		t.Fatal(err)
	}
	android, write, err := renderAndroidLoginProviderManifest(androidPath, "", plan.GitHubReturnURI)
	if err != nil || !write || !strings.Contains(string(android), `android:autoVerify="true"`) || !strings.Contains(string(android), `android:host="github.example.com"`) || !strings.Contains(string(android), `android:label="Keep"`) {
		t.Fatalf("unexpected Android App Link:\n%s\nerr=%v", android, err)
	}
	if err = os.WriteFile(androidPath, android, 0o600); err != nil {
		t.Fatal(err)
	}
	androidSecond, _, err := renderAndroidLoginProviderManifest(androidPath, plan.GitHubReturnURI, plan.GitHubReturnURI)
	if err != nil || string(androidSecond) != string(android) {
		t.Fatalf("Android App Link export is not idempotent:\n%s", androidSecond)
	}

	harmony, err := renderHarmonyLoginProviderOverlay(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(harmony), `"weixin"`) || !strings.Contains(string(harmony), `"wxentity.action.open"`) || !strings.Contains(string(harmony), `"github.example.com"`) {
		t.Fatalf("unexpected Harmony overlay:\n%s", harmony)
	}
}

func TestLoginProviderReturnURIRejectsUnsafeValues(t *testing.T) {
	for _, value := range []string{"http://example.com/oauth/callback", "https://user@example.com/oauth/callback", "https://example.com:8443/oauth/callback", "https://example.com/oauth/callback?ticket=fixed", "custom://oauth/callback"} {
		if _, err := parseHTTPSReturnURI(value); err == nil {
			t.Fatalf("unsafe return URI accepted: %s", value)
		}
	}
}

func TestPrepareLoginProviderExportRequiresAppleForEnabledIOSConsumerLogin(t *testing.T) {
	appID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	records := testLoginProviderRecords(appID)
	records[2].Enabled = false
	if _, err := prepareLoginProviderExport(appID, records); err == nil || !strings.Contains(err.Error(), "require an enabled Apple") {
		t.Fatalf("enabled iOS social login without Apple was accepted: %v", err)
	}
	records[2].Enabled = true
	if _, err := prepareLoginProviderExport(appID, records); err != nil {
		t.Fatalf("enabled Apple binding was rejected: %v", err)
	}
}

func TestDisabledSelectedProviderPreservesBuildProjection(t *testing.T) {
	appID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	records := testLoginProviderRecords(appID)
	enabled, err := prepareLoginProviderExport(appID, records)
	if err != nil {
		t.Fatal(err)
	}
	records[3].Enabled = false
	disabled, err := prepareLoginProviderExport(appID, records)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Snapshot.Providers[3].Enabled {
		t.Fatal("disabled selected configuration remained runtime-enabled")
	}
	if disabled.Snapshot.Providers[3].BuildConfigHash != enabled.Snapshot.Providers[3].BuildConfigHash ||
		string(disabled.Snapshot.Providers[3].BuildConfig) != string(enabled.Snapshot.Providers[3].BuildConfig) {
		t.Fatal("runtime disablement changed the native build projection")
	}
}

func testLoginProviderRecords(appID uuid.UUID) []loginProviderExportRecord {
	return []loginProviderExportRecord{
		{AppID: appID, ConfigID: uuid.New(), ProviderCode: loginprovider.ProviderWechat, ExternalClientID: "wx123", ConfigSchemaVersion: 1, Enabled: true, SortOrder: 10, PublicConfig: json.RawMessage(`{"android":{"enabled":true,"package_name":"com.appkernia.mobile","app_signature":"AA:BB:CC:DD"},"ios":{"enabled":true,"bundle_id":"com.appkernia.mobile","universal_link":"https://login.example.com/wechat/"},"harmony":{"enabled":true,"bundle_name":"com.appkernia.mobile"}}`)},
		{AppID: appID, ConfigID: uuid.New(), ProviderCode: loginprovider.ProviderGitHub, ExternalClientID: "github-client", ConfigSchemaVersion: 1, Enabled: true, SortOrder: 20, PublicConfig: json.RawMessage(`{"app_return_uri":"https://github.example.com/oauth/callback"}`)},
		{AppID: appID, ConfigID: uuid.New(), ProviderCode: loginprovider.ProviderApple, ExternalClientID: "com.appkernia.mobile", ConfigSchemaVersion: 1, Enabled: true, SortOrder: 30, PublicConfig: json.RawMessage(`{"team_id":"TEAM123","key_id":"KEY123"}`)},
		{AppID: appID, ConfigID: uuid.New(), ProviderCode: loginprovider.ProviderGoogle, ExternalClientID: "google-server-client", ConfigSchemaVersion: 1, Enabled: true, SortOrder: 40, PublicConfig: json.RawMessage(`{"android_package_name":"com.appkernia.mobile","android_certificate_sha256":["AA:BB:CC:DD"]}`)},
	}
}
