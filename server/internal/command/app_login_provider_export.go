package command

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	loginprovider "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type loginProviderExportRecord struct {
	AppID, ConfigID     uuid.UUID
	ProviderCode        string
	ExternalClientID    string
	ConfigSchemaVersion int32
	PublicConfig        json.RawMessage
	Enabled             bool
	SortOrder           int32
}

type nativeLoginProviderIdentity struct {
	AndroidPackage, AndroidSignature, AndroidCertificateSHA256 string
	IOSBundleID, HarmonyBundleName                             string
}

type loginProviderSnapshot struct {
	SchemaVersion        int                             `json:"schema_version"`
	AppID                string                          `json:"app_id"`
	BuildVariants        []string                        `json:"build_variants"`
	IOSAssociatedDomains []string                        `json:"ios_associated_domains"`
	Providers            []loginProviderSnapshotProvider `json:"providers"`
}

type loginProviderSnapshotProvider struct {
	ProviderCode        string          `json:"provider_code"`
	ExternalClientID    string          `json:"external_client_id"`
	ConfigSchemaVersion int32           `json:"config_schema_version"`
	BuildConfigHash     string          `json:"build_config_hash"`
	Enabled             bool            `json:"enabled"`
	SortOrder           int32           `json:"sort_order"`
	Platforms           []string        `json:"platforms"`
	BuildVariants       []string        `json:"build_variants"`
	BuildConfig         json.RawMessage `json:"build_config"`
}

type harmonyLoginProviderOverlay struct {
	SchemaVersion int                     `json:"schema_version"`
	QuerySchemes  []string                `json:"query_schemes"`
	Actions       []string                `json:"actions"`
	HTTPSLinks    []harmonyLoginHTTPSLink `json:"https_links"`
}

type harmonyLoginHTTPSLink struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	Path   string `json:"path"`
}

type loginProviderExportPlan struct {
	Snapshot        loginProviderSnapshot
	Wechat          *loginprovider.WechatPublicConfig
	WechatClientID  string
	GitHubReturnURI string
	AppleEnabled    bool
	AppleClientID   string
	Google          *loginprovider.GooglePublicConfig
}

func appLoginProviderCommand(program string, args []string) error {
	usage := fmt.Sprintf("usage: %s app-login-provider export --app-id UUID --output DIR [--check] [native identity flags]", program)
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(os.Stdout, usage)
		return nil
	}
	if len(args) == 0 || args[0] != "export" {
		return &UsageError{Message: usage}
	}
	flags := flag.NewFlagSet("app-login-provider export", flag.ContinueOnError)
	appIDValue := flags.String("app-id", "", "public App UUID")
	output := flags.String("output", "", "mobile project directory")
	check := flags.Bool("check", false, "validate generated files without writing")
	androidPackage := flags.String("android-package", strings.TrimSpace(os.Getenv("AK_ANDROID_PACKAGE")), "Android package used by this build")
	androidSignature := flags.String("android-signature", strings.TrimSpace(os.Getenv("AK_ANDROID_SIGNATURE")), "Android application signature registered with WeChat")
	androidCertificateSHA256 := flags.String("android-certificate-sha256", strings.TrimSpace(os.Getenv("AK_ANDROID_CERTIFICATE_SHA256")), "Android SHA-256 signing certificate fingerprint registered with Google")
	iosBundleID := flags.String("ios-bundle-id", strings.TrimSpace(os.Getenv("AK_IOS_BUNDLE_ID")), "iOS Bundle ID used by this build")
	harmonyBundleName := flags.String("harmony-bundle-name", strings.TrimSpace(os.Getenv("AK_HARMONY_BUNDLE_NAME")), "HarmonyOS Bundle Name used by this build")
	if err := parseCommandFlags(flags, args[1:], usage); err != nil {
		return err
	}
	appID, err := uuid.Parse(strings.TrimSpace(*appIDValue))
	if err != nil || strings.TrimSpace(*output) == "" {
		return fmt.Errorf("--app-id must be a UUID and --output is required")
	}
	outputRoot, err := filepath.Abs(*output)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err = requirePostgreSQL(cfg, "app-login-provider export"); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	records, err := loadLoginProviderExportRecords(ctx, pool, appID)
	if err != nil {
		return err
	}
	plan, err := prepareLoginProviderExport(appID, records)
	if err != nil {
		return err
	}
	identity := nativeLoginProviderIdentity{
		AndroidPackage: strings.TrimSpace(*androidPackage), AndroidSignature: strings.TrimSpace(*androidSignature),
		AndroidCertificateSHA256: strings.TrimSpace(*androidCertificateSHA256), IOSBundleID: strings.TrimSpace(*iosBundleID),
		HarmonyBundleName: strings.TrimSpace(*harmonyBundleName),
	}
	if err = validateNativeLoginProviderIdentity(plan, identity); err != nil {
		return err
	}

	snapshotPath := filepath.Join(outputRoot, "config", "login-providers.generated.json")
	previous, err := readLoginProviderSnapshot(snapshotPath)
	if err != nil {
		return err
	}
	if (previous.SchemaVersion != 0 || previous.AppID != "") && (previous.SchemaVersion != 1 || previous.AppID != appID.String()) {
		return fmt.Errorf("existing login provider snapshot belongs to a different App or schema version")
	}
	snapshot, err := marshalGeneratedJSON(plan.Snapshot)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(outputRoot, "manifest.json")
	manifest, err := renderLoginProviderManifest(manifestPath, plan)
	if err != nil {
		return err
	}
	entitlementsPath := filepath.Join(outputRoot, "nativeResources", "ios", "UniApp.entitlements")
	entitlements, writeEntitlements, err := renderLoginProviderEntitlements(entitlementsPath, previous, plan.Snapshot, shareAssociatedDomains(manifest))
	if err != nil {
		return err
	}
	androidPath := filepath.Join(outputRoot, "nativeResources", "android", "AndroidManifest.xml")
	androidManifest, writeAndroidManifest, err := renderAndroidLoginProviderManifest(androidPath, previousGitHubReturnURI(previous), plan.GitHubReturnURI)
	if err != nil {
		return err
	}
	harmonyPath := filepath.Join(outputRoot, "harmony-configs", "entry", "src", "main", "oauth-links.generated.json")
	harmonyOverlay, err := renderHarmonyLoginProviderOverlay(plan)
	if err != nil {
		return err
	}

	generated := []struct {
		path    string
		content []byte
		write   bool
	}{
		{snapshotPath, snapshot, true},
		{manifestPath, manifest, true},
		{entitlementsPath, entitlements, writeEntitlements},
		{androidPath, androidManifest, writeAndroidManifest},
		{harmonyPath, harmonyOverlay, true},
	}
	for _, file := range generated {
		if !file.write {
			continue
		}
		if *check {
			err = checkGeneratedFile(file.path, file.content)
		} else {
			err = writeGeneratedFile(file.path, file.content, 0o644)
		}
		if err != nil {
			return err
		}
	}
	action := "exported"
	if *check {
		action = "current"
	}
	fmt.Printf("app login provider configuration %s app_id=%s providers=%d output=%s\n", action, appID, len(plan.Snapshot.Providers), outputRoot)
	return nil
}

func loadLoginProviderExportRecords(ctx context.Context, pool *pgxpool.Pool, appID uuid.UUID) ([]loginProviderExportRecord, error) {
	rows, err := pool.Query(ctx, `SELECT a.id,c.id,b.provider_code,c.external_client_id,c.config_schema_version,c.public_config,
       (b.enabled AND c.status='active' AND c.last_preflight_status='ready'),b.sort_order
FROM app.applications a
JOIN app.application_login_provider_bindings b ON b.tenant_id=a.tenant_id AND b.app_id=a.id
JOIN sys.login_provider_configs c ON c.tenant_id=b.tenant_id AND c.id=b.login_provider_config_id AND c.provider_code=b.provider_code
WHERE a.id=$1 AND a.status='active' AND a.deleted_at IS NULL
  AND c.deleted_at IS NULL
ORDER BY b.sort_order,b.provider_code`, appID)
	if err != nil {
		return nil, fmt.Errorf("load selected login provider configurations: %w", err)
	}
	defer rows.Close()
	var records []loginProviderExportRecord
	for rows.Next() {
		var record loginProviderExportRecord
		if err = rows.Scan(&record.AppID, &record.ConfigID, &record.ProviderCode, &record.ExternalClientID, &record.ConfigSchemaVersion, &record.PublicConfig, &record.Enabled, &record.SortOrder); err != nil {
			return nil, fmt.Errorf("scan selected login provider configuration: %w", err)
		}
		records = append(records, record)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read selected login provider configurations: %w", err)
	}
	if len(records) == 0 {
		var exists bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications WHERE id=$1 AND status='active' AND deleted_at IS NULL)`, appID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("validate App for login provider export: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("active App not found for login provider export")
		}
	}
	return records, nil
}

func prepareLoginProviderExport(appID uuid.UUID, records []loginProviderExportRecord) (loginProviderExportPlan, error) {
	plan := loginProviderExportPlan{Snapshot: loginProviderSnapshot{SchemaVersion: 1, AppID: appID.String(), Providers: []loginProviderSnapshotProvider{}, BuildVariants: []string{}, IOSAssociatedDomains: []string{}}}
	seen := map[string]bool{}
	iosConsumerLoginEnabled, appleLoginEnabled := false, false
	for _, record := range records {
		if record.AppID != appID || record.ConfigSchemaVersion != 1 || seen[record.ProviderCode] || strings.TrimSpace(record.ExternalClientID) == "" {
			return plan, fmt.Errorf("invalid selected login provider configuration")
		}
		seen[record.ProviderCode] = true
		canonical, err := loginprovider.CanonicalPublicConfig(record.ProviderCode, record.PublicConfig)
		if err != nil {
			return plan, fmt.Errorf("decode %s build configuration: %w", record.ProviderCode, err)
		}
		hash, err := loginprovider.BuildConfigHash(record.ProviderCode, record.ExternalClientID, canonical)
		if err != nil {
			return plan, fmt.Errorf("hash %s build configuration: %w", record.ProviderCode, err)
		}
		buildConfig, err := loginprovider.BuildConfig(record.ProviderCode, record.ExternalClientID, canonical, "")
		if err != nil {
			return plan, fmt.Errorf("project %s build configuration: %w", record.ProviderCode, err)
		}
		buildConfigJSON, err := json.Marshal(buildConfig)
		if err != nil {
			return plan, fmt.Errorf("encode %s build configuration: %w", record.ProviderCode, err)
		}
		provider := loginProviderSnapshotProvider{ProviderCode: record.ProviderCode, ExternalClientID: record.ExternalClientID, ConfigSchemaVersion: record.ConfigSchemaVersion, BuildConfigHash: hash, Enabled: record.Enabled, SortOrder: record.SortOrder, BuildConfig: buildConfigJSON}
		switch record.ProviderCode {
		case loginprovider.ProviderWechat:
			var value loginprovider.WechatPublicConfig
			if err = json.Unmarshal(canonical, &value); err != nil {
				return plan, err
			}
			if err = validateWechatBuildConfig(value); err != nil {
				return plan, err
			}
			plan.Wechat, plan.WechatClientID = &value, record.ExternalClientID
			if value.Android.Enabled {
				provider.Platforms = append(provider.Platforms, "android")
				provider.BuildVariants = append(provider.BuildVariants, "android_china", "android_google")
			}
			if value.IOS.Enabled {
				iosConsumerLoginEnabled = iosConsumerLoginEnabled || record.Enabled
				provider.Platforms = append(provider.Platforms, "ios")
				provider.BuildVariants = append(provider.BuildVariants, "ios")
				plan.Snapshot.IOSAssociatedDomains = append(plan.Snapshot.IOSAssociatedDomains, associatedDomain(value.IOS.UniversalLink))
			}
			if value.Harmony.Enabled {
				provider.Platforms = append(provider.Platforms, "harmony")
				provider.BuildVariants = append(provider.BuildVariants, "harmony")
			}
		case loginprovider.ProviderGitHub:
			var value loginprovider.GitHubPublicConfig
			if err = json.Unmarshal(canonical, &value); err != nil {
				return plan, err
			}
			if _, err = parseHTTPSReturnURI(value.AppReturnURI); err != nil {
				return plan, fmt.Errorf("invalid GitHub App return URI: %w", err)
			}
			plan.GitHubReturnURI = value.AppReturnURI
			iosConsumerLoginEnabled = iosConsumerLoginEnabled || record.Enabled
			provider.Platforms = []string{"android", "harmony", "ios"}
			provider.BuildVariants = []string{"android_china", "android_google", "harmony", "ios"}
			plan.Snapshot.IOSAssociatedDomains = append(plan.Snapshot.IOSAssociatedDomains, associatedDomain(value.AppReturnURI))
		case loginprovider.ProviderApple:
			var value loginprovider.ApplePublicConfig
			if err = json.Unmarshal(canonical, &value); err != nil || strings.TrimSpace(value.TeamID) == "" || strings.TrimSpace(value.KeyID) == "" {
				return plan, fmt.Errorf("invalid Apple build configuration")
			}
			plan.AppleEnabled, plan.AppleClientID = true, record.ExternalClientID
			appleLoginEnabled = record.Enabled
			provider.Platforms = []string{"ios"}
			provider.BuildVariants = []string{"ios"}
		case loginprovider.ProviderGoogle:
			var value loginprovider.GooglePublicConfig
			if err = json.Unmarshal(canonical, &value); err != nil || strings.TrimSpace(value.AndroidPackageName) == "" || len(value.AndroidCertificateSHA256) == 0 {
				return plan, fmt.Errorf("invalid Google build configuration")
			}
			plan.Google = &value
			provider.Platforms = []string{"android"}
			provider.BuildVariants = []string{"android_google"}
		default:
			return plan, fmt.Errorf("unsupported login provider %q", record.ProviderCode)
		}
		plan.Snapshot.Providers = append(plan.Snapshot.Providers, provider)
		plan.Snapshot.BuildVariants = append(plan.Snapshot.BuildVariants, provider.BuildVariants...)
	}
	if iosConsumerLoginEnabled && !appleLoginEnabled {
		return plan, fmt.Errorf("enabled iOS consumer login providers require an enabled Apple login binding")
	}
	plan.Snapshot.BuildVariants = uniqueSorted(plan.Snapshot.BuildVariants)
	plan.Snapshot.IOSAssociatedDomains = uniqueSorted(plan.Snapshot.IOSAssociatedDomains)
	return plan, nil
}

func validateWechatBuildConfig(value loginprovider.WechatPublicConfig) error {
	if value.Android.Enabled && (strings.TrimSpace(value.Android.PackageName) == "" || strings.TrimSpace(value.Android.AppSignature) == "") {
		return fmt.Errorf("invalid WeChat Android build configuration")
	}
	if value.IOS.Enabled {
		if strings.TrimSpace(value.IOS.BundleID) == "" {
			return fmt.Errorf("invalid WeChat iOS build configuration")
		}
		if _, err := parseHTTPSDirectoryURI(value.IOS.UniversalLink); err != nil {
			return fmt.Errorf("invalid WeChat iOS Universal Link: %w", err)
		}
	}
	if value.Harmony.Enabled && strings.TrimSpace(value.Harmony.BundleName) == "" {
		return fmt.Errorf("invalid WeChat HarmonyOS build configuration")
	}
	return nil
}

func validateNativeLoginProviderIdentity(plan loginProviderExportPlan, build nativeLoginProviderIdentity) error {
	match := func(label, actual, expected string) error {
		if actual == "" {
			return fmt.Errorf("%s build identity is required for login provider export", label)
		}
		if actual != expected {
			return fmt.Errorf("%s build identity does not match the selected login provider configuration", label)
		}
		return nil
	}
	if plan.Wechat != nil {
		if plan.Wechat.Android.Enabled {
			if err := match("Android package", build.AndroidPackage, plan.Wechat.Android.PackageName); err != nil {
				return err
			}
			if err := match("Android WeChat signature", normalizeFingerprint(build.AndroidSignature), normalizeFingerprint(plan.Wechat.Android.AppSignature)); err != nil {
				return err
			}
		}
		if plan.Wechat.IOS.Enabled {
			if err := match("iOS Bundle ID", build.IOSBundleID, plan.Wechat.IOS.BundleID); err != nil {
				return err
			}
		}
		if plan.Wechat.Harmony.Enabled {
			if err := match("HarmonyOS Bundle Name", build.HarmonyBundleName, plan.Wechat.Harmony.BundleName); err != nil {
				return err
			}
		}
	}
	if plan.AppleEnabled {
		if err := match("Apple iOS Bundle ID", build.IOSBundleID, plan.AppleClientID); err != nil {
			return err
		}
	}
	if plan.Google != nil {
		if err := match("Google Android package", build.AndroidPackage, plan.Google.AndroidPackageName); err != nil {
			return err
		}
		actual := normalizeFingerprint(build.AndroidCertificateSHA256)
		if actual == "" {
			return fmt.Errorf("Google Android SHA-256 certificate build identity is required for login provider export")
		}
		for _, expected := range plan.Google.AndroidCertificateSHA256 {
			if actual == normalizeFingerprint(expected) {
				return nil
			}
		}
		return fmt.Errorf("Google Android SHA-256 certificate does not match the selected login provider configuration")
	}
	return nil
}

func normalizeFingerprint(value string) string {
	return strings.ToLower(strings.NewReplacer(":", "", " ", "", "-", "").Replace(strings.TrimSpace(value)))
}

func renderLoginProviderManifest(path string, plan loginProviderExportPlan) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mobile manifest: %w", err)
	}
	var document map[string]any
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err = decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode mobile manifest: %w", err)
	}
	platforms := []struct {
		key     string
		enabled bool
		config  map[string]any
	}{
		{key: "app-android"}, {key: "app-ios"}, {key: "app-harmony"},
	}
	if plan.Wechat != nil {
		platforms[0].enabled, platforms[0].config = plan.Wechat.Android.Enabled, map[string]any{"appid": plan.WechatClientID}
		platforms[1].enabled, platforms[1].config = plan.Wechat.IOS.Enabled, map[string]any{"appid": plan.WechatClientID, "universalLink": plan.Wechat.IOS.UniversalLink}
		platforms[2].enabled, platforms[2].config = plan.Wechat.Harmony.Enabled, map[string]any{"appid": plan.WechatClientID}
	}
	for _, platform := range platforms {
		if platform.enabled {
			modules := manifestMap(manifestMap(manifestMap(document, platform.key), "distribute"), "modules")
			manifestMap(modules, "uni-oauth")["weixin"] = platform.config
		} else {
			removeLoginManifestProvider(document, platform.key)
		}
	}
	out, err := json.MarshalIndent(document, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("encode mobile manifest: %w", err)
	}
	return append(out, '\n'), nil
}

func removeLoginManifestProvider(document map[string]any, platform string) {
	platformMap, ok := document[platform].(map[string]any)
	if !ok {
		return
	}
	distribute, ok := platformMap["distribute"].(map[string]any)
	if !ok {
		return
	}
	modules, ok := distribute["modules"].(map[string]any)
	if !ok {
		return
	}
	oauth, ok := modules["uni-oauth"].(map[string]any)
	if !ok {
		return
	}
	delete(oauth, "weixin")
	if len(oauth) == 0 {
		delete(modules, "uni-oauth")
	}
}

func marshalGeneratedJSON(value any) ([]byte, error) {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode generated login provider configuration: %w", err)
	}
	return append(out, '\n'), nil
}

func readLoginProviderSnapshot(path string) (loginProviderSnapshot, error) {
	var snapshot loginProviderSnapshot
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return snapshot, nil
	}
	if err != nil {
		return snapshot, fmt.Errorf("read previous login provider snapshot: %w", err)
	}
	if err = json.Unmarshal(content, &snapshot); err != nil {
		return snapshot, fmt.Errorf("decode previous login provider snapshot: %w", err)
	}
	return snapshot, nil
}

func shareAssociatedDomains(manifest []byte) []string {
	var document map[string]any
	if json.Unmarshal(manifest, &document) != nil {
		return nil
	}
	value := document
	for _, key := range []string{"app-ios", "distribute", "modules", "uni-share", "weixin"} {
		next, ok := value[key].(map[string]any)
		if !ok {
			return nil
		}
		value = next
	}
	link, _ := value["universalLink"].(string)
	if link == "" {
		return nil
	}
	return []string{associatedDomain(link)}
}

func renderLoginProviderEntitlements(path string, previous, current loginProviderSnapshot, preserveDomains []string) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if len(current.IOSAssociatedDomains) == 0 && !snapshotHasProvider(current, loginprovider.ProviderApple) {
			return nil, false, nil
		}
		content = []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1.0\">\n<dict>\n</dict>\n</plist>\n")
	} else if err != nil {
		return nil, false, fmt.Errorf("read iOS entitlements: %w", err)
	}
	preserve := stringSet(append(append([]string{}, current.IOSAssociatedDomains...), preserveDomains...))
	for _, domain := range previous.IOSAssociatedDomains {
		if !preserve[domain] {
			content, err = removePlistArrayValue(content, "com.apple.developer.associated-domains", domain)
			if err != nil {
				return nil, false, err
			}
		}
	}
	for _, domain := range current.IOSAssociatedDomains {
		content, err = ensurePlistArrayValue(content, "com.apple.developer.associated-domains", domain)
		if err != nil {
			return nil, false, err
		}
	}
	appleNow := snapshotHasProvider(current, loginprovider.ProviderApple)
	if appleNow {
		content, err = ensurePlistArrayValue(content, "com.apple.developer.applesignin", "Default")
	} else if snapshotHasProvider(previous, loginprovider.ProviderApple) {
		content, err = removePlistArrayValue(content, "com.apple.developer.applesignin", "Default")
	}
	if err != nil {
		return nil, false, err
	}
	return content, true, nil
}

func ensurePlistArrayValue(content []byte, key, value string) ([]byte, error) {
	escaped := html.EscapeString(value)
	needle := []byte("<string>" + escaped + "</string>")
	keyNeedle := []byte("<key>" + key + "</key>")
	keyIndex := bytes.Index(content, keyNeedle)
	if keyIndex >= 0 {
		arrayStart := bytes.Index(content[keyIndex+len(keyNeedle):], []byte("<array>"))
		if arrayStart < 0 {
			return nil, fmt.Errorf("iOS entitlement %s has no array", key)
		}
		arrayStart += keyIndex + len(keyNeedle)
		arrayEnd := bytes.Index(content[arrayStart:], []byte("</array>"))
		if arrayEnd < 0 {
			return nil, fmt.Errorf("iOS entitlement %s has no array end", key)
		}
		arrayEnd += arrayStart
		if bytes.Contains(content[arrayStart:arrayEnd], needle) {
			return content, nil
		}
		return append(append(append([]byte{}, content[:arrayEnd]...), []byte("\t\t<string>"+escaped+"</string>\n\t")...), content[arrayEnd:]...), nil
	}
	dictEnd := bytes.LastIndex(content, []byte("</dict>"))
	if dictEnd < 0 {
		return nil, fmt.Errorf("iOS entitlements has no root dictionary")
	}
	entry := []byte("\t<key>" + key + "</key>\n\t<array>\n\t\t<string>" + escaped + "</string>\n\t</array>\n")
	return append(append(append([]byte{}, content[:dictEnd]...), entry...), content[dictEnd:]...), nil
}

func removePlistArrayValue(content []byte, key, value string) ([]byte, error) {
	keyNeedle := []byte("<key>" + key + "</key>")
	keyIndex := bytes.Index(content, keyNeedle)
	if keyIndex < 0 {
		return content, nil
	}
	arrayStart := bytes.Index(content[keyIndex+len(keyNeedle):], []byte("<array>"))
	if arrayStart < 0 {
		return nil, fmt.Errorf("iOS entitlement %s has no array", key)
	}
	arrayStart += keyIndex + len(keyNeedle)
	arrayEnd := bytes.Index(content[arrayStart:], []byte("</array>"))
	if arrayEnd < 0 {
		return nil, fmt.Errorf("iOS entitlement %s has no array end", key)
	}
	arrayEnd += arrayStart
	needle := []byte("<string>" + html.EscapeString(value) + "</string>")
	valueIndex := bytes.Index(content[arrayStart:arrayEnd], needle)
	if valueIndex < 0 {
		return content, nil
	}
	valueIndex += arrayStart
	lineStart := bytes.LastIndex(content[:valueIndex], []byte("\n")) + 1
	lineEndOffset := bytes.Index(content[valueIndex+len(needle):], []byte("\n"))
	lineEnd := valueIndex + len(needle)
	if lineEndOffset >= 0 {
		lineEnd += lineEndOffset + 1
	}
	content = append(append([]byte{}, content[:lineStart]...), content[lineEnd:]...)
	keyIndex = bytes.Index(content, keyNeedle)
	arrayStart = bytes.Index(content[keyIndex+len(keyNeedle):], []byte("<array>")) + keyIndex + len(keyNeedle)
	arrayEnd = bytes.Index(content[arrayStart:], []byte("</array>")) + arrayStart
	if strings.TrimSpace(string(content[arrayStart+len("<array>"):arrayEnd])) != "" {
		return content, nil
	}
	blockStart := bytes.LastIndex(content[:keyIndex], []byte("\n")) + 1
	blockEndOffset := bytes.Index(content[arrayEnd+len("</array>"):], []byte("\n"))
	blockEnd := arrayEnd + len("</array>")
	if blockEndOffset >= 0 {
		blockEnd += blockEndOffset + 1
	}
	return append(append([]byte{}, content[:blockStart]...), content[blockEnd:]...), nil
}

const androidLoginProviderStart = "<!-- appkernia-login-provider:start -->"
const androidLoginProviderEnd = "<!-- appkernia-login-provider:end -->"

func renderAndroidLoginProviderManifest(path, previousReturnURI, currentReturnURI string) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) && currentReturnURI == "" && previousReturnURI == "" {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read Android native manifest: %w", err)
	}
	start := bytes.Index(content, []byte(androidLoginProviderStart))
	if start >= 0 {
		start = bytes.LastIndex(content[:start], []byte("\n")) + 1
		endOffset := bytes.Index(content[start:], []byte(androidLoginProviderEnd))
		if endOffset < 0 {
			return nil, false, fmt.Errorf("Android login provider managed block is incomplete")
		}
		end := start + endOffset + len(androidLoginProviderEnd)
		if end < len(content) && content[end] == '\n' {
			end++
		}
		content = append(append([]byte{}, content[:start]...), content[end:]...)
	}
	if currentReturnURI == "" {
		return content, true, nil
	}
	parsed, err := parseHTTPSReturnURI(currentReturnURI)
	if err != nil {
		return nil, false, err
	}
	block := []byte("        " + androidLoginProviderStart + "\n" +
		"        <activity android:name=\"io.dcloud.uniapp.UniLaunchProxyActivity\" android:exported=\"true\">\n" +
		"            <intent-filter android:autoVerify=\"true\">\n" +
		"                <action android:name=\"android.intent.action.VIEW\" />\n" +
		"                <category android:name=\"android.intent.category.DEFAULT\" />\n" +
		"                <category android:name=\"android.intent.category.BROWSABLE\" />\n" +
		"                <data android:scheme=\"https\" android:host=\"" + html.EscapeString(parsed.Hostname()) + "\" android:path=\"" + html.EscapeString(parsed.EscapedPath()) + "\" />\n" +
		"            </intent-filter>\n" +
		"        </activity>\n" +
		"        " + androidLoginProviderEnd + "\n")
	applicationEnd := bytes.LastIndex(content, []byte("</application>"))
	if applicationEnd >= 0 {
		insertAt := applicationEnd
		lineStart := bytes.LastIndex(content[:applicationEnd], []byte("\n")) + 1
		if lineStart > 0 && len(bytes.TrimSpace(content[lineStart:applicationEnd])) == 0 {
			insertAt = lineStart
		}
		return append(append(append([]byte{}, content[:insertAt]...), block...), content[insertAt:]...), true, nil
	}
	applicationStart := bytes.Index(content, []byte("<application"))
	if applicationStart < 0 {
		return nil, false, fmt.Errorf("Android native manifest has no application")
	}
	selfCloseOffset := bytes.Index(content[applicationStart:], []byte("/>"))
	if selfCloseOffset < 0 {
		return nil, false, fmt.Errorf("Android native application has no close tag")
	}
	selfClose := applicationStart + selfCloseOffset
	replacement := append([]byte(">\n"), block...)
	replacement = append(replacement, []byte("    </application>")...)
	return append(append(append([]byte{}, content[:selfClose]...), replacement...), content[selfClose+2:]...), true, nil
}

func renderHarmonyLoginProviderOverlay(plan loginProviderExportPlan) ([]byte, error) {
	overlay := harmonyLoginProviderOverlay{SchemaVersion: 1, QuerySchemes: []string{}, Actions: []string{}, HTTPSLinks: []harmonyLoginHTTPSLink{}}
	if plan.Wechat != nil && plan.Wechat.Harmony.Enabled {
		overlay.QuerySchemes = []string{"weixin"}
		overlay.Actions = []string{"action.system.home", "wxentity.action.open"}
	}
	if plan.GitHubReturnURI != "" {
		parsed, err := parseHTTPSReturnURI(plan.GitHubReturnURI)
		if err != nil {
			return nil, err
		}
		overlay.HTTPSLinks = append(overlay.HTTPSLinks, harmonyLoginHTTPSLink{Scheme: "https", Host: parsed.Hostname(), Path: parsed.EscapedPath()})
	}
	return marshalGeneratedJSON(overlay)
}

func parseHTTPSReturnURI(raw string) (*url.URL, error) {
	return parseHTTPSAppLink(raw, false)
}

func parseHTTPSDirectoryURI(raw string) (*url.URL, error) {
	return parseHTTPSAppLink(raw, true)
}

func parseHTTPSAppLink(raw string, directory bool) (*url.URL, error) {
	canonical, err := loginprovider.CanonicalHTTPSAppLink(raw, directory)
	if err != nil {
		return nil, err
	}
	return url.Parse(canonical)
}

func associatedDomain(raw string) string {
	parsed, err := parseHTTPSReturnURI(raw)
	if err != nil {
		return ""
	}
	return "applinks:" + parsed.Hostname()
}

func previousGitHubReturnURI(snapshot loginProviderSnapshot) string {
	for _, provider := range snapshot.Providers {
		if provider.ProviderCode != loginprovider.ProviderGitHub {
			continue
		}
		var value loginprovider.GitHubPublicConfig
		if json.Unmarshal(provider.BuildConfig, &value) == nil {
			return value.AppReturnURI
		}
	}
	return ""
}

func snapshotHasProvider(snapshot loginProviderSnapshot, code string) bool {
	for _, provider := range snapshot.Providers {
		if provider.ProviderCode == code {
			return true
		}
	}
	return false
}

func stringSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func uniqueSorted(values []string) []string {
	set := stringSet(values)
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
