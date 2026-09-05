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
	"strings"
	"time"

	share "github.com/appkernia/appkernia/server/internal/modules/shareconfig/domain"
	"github.com/appkernia/appkernia/server/internal/platform/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type shareExportRecord struct {
	AppID         uuid.UUID
	ExternalAppID string
	PublicConfig  share.WechatPublicConfig
}

type nativeShareIdentity struct {
	AndroidPackage, AndroidSignature, IOSBundleID, HarmonyBundleName string
}

func appShareCommand(program string, args []string) error {
	usage := fmt.Sprintf("usage: %s app-share export --app-id UUID --output DIR [--check] [native identity flags]", program)
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(os.Stdout, usage)
		return nil
	}
	if len(args) == 0 || args[0] != "export" {
		return &UsageError{Message: usage}
	}
	flags := flag.NewFlagSet("app-share export", flag.ContinueOnError)
	appIDValue := flags.String("app-id", "", "public App UUID")
	output := flags.String("output", "", "mobile project directory")
	check := flags.Bool("check", false, "validate generated files without writing")
	androidPackage := flags.String("android-package", strings.TrimSpace(os.Getenv("AK_ANDROID_PACKAGE")), "Android package used by this build")
	androidSignature := flags.String("android-signature", strings.TrimSpace(os.Getenv("AK_ANDROID_SIGNATURE")), "Android application signature registered with WeChat")
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
	if err = requirePostgreSQL(cfg, "app-share export"); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	record, err := loadShareExportRecord(ctx, pool, appID)
	if err != nil {
		return err
	}
	identity := nativeShareIdentity{AndroidPackage: strings.TrimSpace(*androidPackage), AndroidSignature: strings.TrimSpace(*androidSignature), IOSBundleID: strings.TrimSpace(*iosBundleID), HarmonyBundleName: strings.TrimSpace(*harmonyBundleName)}
	if err = validateNativeShareIdentity(record.PublicConfig, identity); err != nil {
		return err
	}
	manifestPath := filepath.Join(outputRoot, "manifest.json")
	manifest, err := renderShareManifest(manifestPath, record)
	if err != nil {
		return err
	}
	entitlementsPath := filepath.Join(outputRoot, "nativeResources", "ios", "UniApp.entitlements")
	var entitlements []byte
	if record.PublicConfig.IOS.Enabled {
		entitlements, err = renderAssociatedDomains(entitlementsPath, record.PublicConfig.IOS.UniversalLink)
		if err != nil {
			return err
		}
	}
	if *check {
		if err = checkGeneratedFile(manifestPath, manifest); err != nil {
			return err
		}
		if record.PublicConfig.IOS.Enabled {
			if err = checkGeneratedFile(entitlementsPath, entitlements); err != nil {
				return err
			}
		}
		fmt.Printf("app share configuration current app_id=%s provider=wechat output=%s\n", appID, outputRoot)
		return nil
	}
	if err = writeGeneratedFile(manifestPath, manifest, 0o644); err != nil {
		return err
	}
	if record.PublicConfig.IOS.Enabled {
		if err = writeGeneratedFile(entitlementsPath, entitlements, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("exported app share configuration app_id=%s provider=wechat output=%s\n", appID, outputRoot)
	return nil
}

func loadShareExportRecord(ctx context.Context, pool *pgxpool.Pool, appID uuid.UUID) (shareExportRecord, error) {
	var out shareExportRecord
	var raw json.RawMessage
	err := pool.QueryRow(ctx, `SELECT a.id,c.external_app_id,c.public_config
FROM app.applications a
JOIN app.application_share_bindings b ON b.tenant_id=a.tenant_id AND b.app_id=a.id AND b.provider_code='wechat' AND b.enabled=true
JOIN sys.share_configs c ON c.tenant_id=b.tenant_id AND c.id=b.share_config_id AND c.provider_code='wechat' AND c.status='active' AND c.deleted_at IS NULL
WHERE a.id=$1 AND a.status='active' AND a.deleted_at IS NULL`, appID).Scan(&out.AppID, &out.ExternalAppID, &raw)
	if err != nil {
		return out, fmt.Errorf("load active WeChat share binding: %w", err)
	}
	if err = json.Unmarshal(raw, &out.PublicConfig); err != nil {
		return out, fmt.Errorf("decode WeChat public configuration: %w", err)
	}
	return out, nil
}

func validateNativeShareIdentity(config share.WechatPublicConfig, build nativeShareIdentity) error {
	validate := func(platform, actual, expected string) error {
		if actual == "" {
			return fmt.Errorf("%s build identity is required for share export", platform)
		}
		if actual != expected {
			return fmt.Errorf("%s build identity does not match the active share configuration", platform)
		}
		return nil
	}
	if config.Android.Enabled {
		if err := validate("Android package", build.AndroidPackage, config.Android.PackageName); err != nil {
			return err
		}
		if err := validate("Android signature", strings.ToLower(strings.ReplaceAll(build.AndroidSignature, ":", "")), strings.ToLower(strings.ReplaceAll(config.Android.Signature, ":", ""))); err != nil {
			return err
		}
	}
	if config.IOS.Enabled {
		if err := validate("iOS Bundle ID", build.IOSBundleID, config.IOS.BundleID); err != nil {
			return err
		}
	}
	if config.Harmony.Enabled {
		if err := validate("Harmony Bundle Name", build.HarmonyBundleName, config.Harmony.BundleName); err != nil {
			return err
		}
	}
	return nil
}

func renderShareManifest(path string, record shareExportRecord) ([]byte, error) {
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
	if record.PublicConfig.Android.Enabled {
		setManifestProvider(document, "app-android", map[string]any{"appid": record.ExternalAppID})
	} else {
		removeManifestProvider(document, "app-android")
	}
	if record.PublicConfig.IOS.Enabled {
		setManifestProvider(document, "app-ios", map[string]any{"appid": record.ExternalAppID, "universalLink": record.PublicConfig.IOS.UniversalLink})
	} else {
		removeManifestProvider(document, "app-ios")
	}
	if record.PublicConfig.Harmony.Enabled {
		setManifestProvider(document, "app-harmony", map[string]any{"appid": record.ExternalAppID})
	} else {
		removeManifestProvider(document, "app-harmony")
	}
	out, err := json.MarshalIndent(document, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("encode mobile manifest: %w", err)
	}
	return append(out, '\n'), nil
}

func manifestMap(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	value := map[string]any{}
	parent[key] = value
	return value
}

func setManifestProvider(document map[string]any, platform string, weixin map[string]any) {
	distribute := manifestMap(manifestMap(document, platform), "distribute")
	modules := manifestMap(distribute, "modules")
	shareModule := manifestMap(modules, "uni-share")
	shareModule["weixin"] = weixin
}

func removeManifestProvider(document map[string]any, platform string) {
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
	shareModule, ok := modules["uni-share"].(map[string]any)
	if !ok {
		return
	}
	delete(shareModule, "weixin")
	if len(shareModule) == 0 {
		delete(modules, "uni-share")
	}
}

func renderAssociatedDomains(path, universalLink string) ([]byte, error) {
	parsed, err := url.Parse(strings.TrimSpace(universalLink))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("invalid iOS Universal Link")
	}
	domain := "applinks:" + parsed.Hostname()
	content, err := os.ReadFile(path)
	if errorsIsNotExist(err) {
		return []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1.0\">\n<dict>\n\t<key>com.apple.developer.associated-domains</key>\n\t<array>\n\t\t<string>" + html.EscapeString(domain) + "</string>\n\t</array>\n</dict>\n</plist>\n"), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read iOS entitlements: %w", err)
	}
	if bytes.Contains(content, []byte("<string>"+html.EscapeString(domain)+"</string>")) {
		return content, nil
	}
	key := []byte("<key>com.apple.developer.associated-domains</key>")
	position := bytes.Index(content, key)
	if position >= 0 {
		arrayEnd := bytes.Index(content[position:], []byte("</array>"))
		if arrayEnd < 0 {
			return nil, fmt.Errorf("associated domains entitlement has no array")
		}
		insert := position + arrayEnd
		return append(append(append([]byte{}, content[:insert]...), []byte("\t\t<string>"+html.EscapeString(domain)+"</string>\n\t")...), content[insert:]...), nil
	}
	dictEnd := bytes.LastIndex(content, []byte("</dict>"))
	if dictEnd < 0 {
		return nil, fmt.Errorf("iOS entitlements has no root dictionary")
	}
	entry := []byte("\t<key>com.apple.developer.associated-domains</key>\n\t<array>\n\t\t<string>" + html.EscapeString(domain) + "</string>\n\t</array>\n")
	return append(append(append([]byte{}, content[:dictEnd]...), entry...), content[dictEnd:]...), nil
}

func errorsIsNotExist(err error) bool { return err != nil && os.IsNotExist(err) }

func checkGeneratedFile(path string, expected []byte) error {
	actual, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("generated share file missing or unreadable %s: %w", path, err)
	}
	if !bytes.Equal(actual, expected) {
		return fmt.Errorf("generated share file drifted: %s", path)
	}
	return nil
}
