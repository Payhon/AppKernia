package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

const GitHubBrowserCallbackPath = "/api/v1/auth/oauth/github/browser-callback"

// GitHubBrowserCallbackURI is the single source of truth used by the Admin
// help field, provider token exchange and generated mobile build config.
func GitHubBrowserCallbackURI(publicBaseURI string) string {
	return strings.TrimRight(strings.TrimSpace(publicBaseURI), "/") + GitHubBrowserCallbackPath
}

var (
	wechatAppIDPattern  = regexp.MustCompile(`^wx[A-Za-z0-9]{16}$`)
	packagePattern      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z0-9_]+)+$`)
	signaturePattern    = regexp.MustCompile(`^[A-Fa-f0-9:]{16,256}$`)
	appleTeamKeyPattern = regexp.MustCompile(`^[A-Z0-9]{10}$`)
	clientIDPattern     = regexp.MustCompile(`^[A-Za-z0-9._-]{6,255}$`)
	googleClientPattern = regexp.MustCompile(`^[0-9]+-[A-Za-z0-9_-]+\.apps\.googleusercontent\.com$`)
	sha256ColonPattern  = regexp.MustCompile(`^(?:[A-F0-9]{2}:){31}[A-F0-9]{2}$`)
	dnsHostPattern      = regexp.MustCompile(`^(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)+[A-Za-z]{2,63}$`)
)

var providerCatalog = []ProviderDescriptor{
	{
		ProviderCode: ProviderWechat, DisplayNameKey: "login_provider.wechat", IconKey: "wechat",
		AuthorizationKind: "native_code", SupportedPlatforms: []string{"ios", "android", "harmony"},
		BuildVariants: []string{"ios", "android_google", "android_china", "harmony"}, ConfigSchemaVersion: 1,
		RequiresSecret: true, HelpURL: "https://developers.weixin.qq.com/doc/oplatform/Mobile_App/WeChat_Login/Development_Guide.html",
		Fields: []FieldDescriptor{
			{Name: "external_client_id", Location: "external_client_id", ValueType: "string", Required: true, MaxLength: 255, HelpKey: "login_provider.fields.wechat.app_id"},
			{Name: "android_package_name", Location: "public_config", ValueType: "string", Required: false, MaxLength: 255, HelpKey: "login_provider.fields.wechat.android_package_name"},
			{Name: "android_app_signature", Location: "public_config", ValueType: "string", Required: false, MaxLength: 256, HelpKey: "login_provider.fields.wechat.android_app_signature"},
			{Name: "ios_bundle_id", Location: "public_config", ValueType: "string", Required: false, MaxLength: 255, HelpKey: "login_provider.fields.wechat.ios_bundle_id"},
			{Name: "ios_universal_link", Location: "public_config", ValueType: "url", Required: false, MaxLength: 2048, HelpKey: "login_provider.fields.wechat.ios_universal_link"},
			{Name: "harmony_bundle_name", Location: "public_config", ValueType: "string", Required: false, MaxLength: 255, HelpKey: "login_provider.fields.wechat.harmony_bundle_name"},
			{Name: "app_secret", Location: "secret", ValueType: "string", Required: true, Secret: true, MaxLength: 512, HelpKey: "login_provider.fields.wechat.app_secret"},
		},
	},
	{
		ProviderCode: ProviderGitHub, DisplayNameKey: "login_provider.github", IconKey: "github",
		AuthorizationKind: "browser_ticket", SupportedPlatforms: []string{"ios", "android", "harmony"},
		BuildVariants: []string{"ios", "android_google", "android_china", "harmony"}, ConfigSchemaVersion: 1,
		RequiresSecret: true, HelpURL: "https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps",
		Fields: []FieldDescriptor{
			{Name: "external_client_id", Location: "external_client_id", ValueType: "string", Required: true, MaxLength: 255, HelpKey: "login_provider.fields.github.client_id"},
			{Name: "app_return_uri", Location: "public_config", ValueType: "url", Required: true, MaxLength: 2048, HelpKey: "login_provider.fields.github.app_return_uri"},
			{Name: "client_secret", Location: "secret", ValueType: "string", Required: true, Secret: true, MaxLength: 1024, HelpKey: "login_provider.fields.github.client_secret"},
		},
	},
	{
		ProviderCode: ProviderApple, DisplayNameKey: "login_provider.apple", IconKey: "apple",
		AuthorizationKind: "native_id_token", SupportedPlatforms: []string{"ios"}, BuildVariants: []string{"ios"},
		ConfigSchemaVersion: 1, RequiresSecret: true,
		HelpURL: "https://developer.apple.com/sign-in-with-apple/get-started/",
		Fields: []FieldDescriptor{
			{Name: "external_client_id", Location: "external_client_id", ValueType: "string", Required: true, MaxLength: 255, HelpKey: "login_provider.fields.apple.client_id"},
			{Name: "team_id", Location: "public_config", ValueType: "string", Required: true, MaxLength: 10, HelpKey: "login_provider.fields.apple.team_id"},
			{Name: "key_id", Location: "public_config", ValueType: "string", Required: true, MaxLength: 10, HelpKey: "login_provider.fields.apple.key_id"},
			{Name: "private_key_p8", Location: "secret", ValueType: "pem", Required: true, Secret: true, MaxLength: 8192, HelpKey: "login_provider.fields.apple.private_key_p8"},
		},
	},
	{
		ProviderCode: ProviderGoogle, DisplayNameKey: "login_provider.google", IconKey: "google",
		AuthorizationKind: "native_id_token", SupportedPlatforms: []string{"android"}, BuildVariants: []string{"android_google"},
		ConfigSchemaVersion: 1, RequiresSecret: false,
		HelpURL: "https://developer.android.com/identity/sign-in/credential-manager-siwg",
		Fields: []FieldDescriptor{
			{Name: "external_client_id", Location: "external_client_id", ValueType: "string", Required: true, MaxLength: 255, HelpKey: "login_provider.fields.google.server_client_id"},
			{Name: "android_package_name", Location: "public_config", ValueType: "string", Required: true, MaxLength: 255, HelpKey: "login_provider.fields.google.android_package_name"},
			{Name: "android_certificate_sha256", Location: "public_config", ValueType: "string_array", Required: true, MaxLength: 16, HelpKey: "login_provider.fields.google.android_certificate_sha256"},
		},
	},
}

func RegisteredCatalog() Catalog {
	items := make([]ProviderDescriptor, len(providerCatalog))
	for index := range providerCatalog {
		items[index] = providerCatalog[index]
		items[index].SupportedPlatforms = append([]string(nil), providerCatalog[index].SupportedPlatforms...)
		items[index].BuildVariants = append([]string(nil), providerCatalog[index].BuildVariants...)
		items[index].Fields = append([]FieldDescriptor(nil), providerCatalog[index].Fields...)
	}
	return Catalog{Items: items}
}

func Descriptor(providerCode string) (ProviderDescriptor, bool) {
	for _, descriptor := range providerCatalog {
		if descriptor.ProviderCode == providerCode {
			return descriptor, true
		}
	}
	return ProviderDescriptor{}, false
}

// ConfigSupportsTarget applies provider-specific enablement on top of the
// compile-time platform matrix. In particular, a partially configured WeChat
// app cannot authorize from an unconfigured native target.
func ConfigSupportsTarget(providerCode string, raw json.RawMessage, platform, buildVariant string) bool {
	switch providerCode {
	case ProviderWechat:
		value, err := decodeStrict[WechatPublicConfig](raw)
		if err != nil {
			return false
		}
		switch platform {
		case "ios":
			return buildVariant == "ios" && value.IOS.Enabled
		case "android":
			return (buildVariant == "android_google" || buildVariant == "android_china") && value.Android.Enabled
		case "harmony":
			return buildVariant == "harmony" && value.Harmony.Enabled
		default:
			return false
		}
	case ProviderGitHub:
		_, err := decodeStrict[GitHubPublicConfig](raw)
		return err == nil
	case ProviderApple:
		_, err := decodeStrict[ApplePublicConfig](raw)
		return err == nil && platform == "ios" && buildVariant == "ios"
	case ProviderGoogle:
		_, err := decodeStrict[GooglePublicConfig](raw)
		return err == nil && platform == "android" && buildVariant == "android_google"
	default:
		return false
	}
}

// CanonicalHTTPSAppLink validates the exact URL subset supported by native
// associated-domain exporters. Provider configuration and CLI export share
// this function so a config cannot become ready but remain unbuildable.
func CanonicalHTTPSAppLink(raw string, directory bool) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Port() != "" ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") {
		return "", ErrInvalid
	}
	host := strings.ToLower(parsed.Hostname())
	if !dnsHostPattern.MatchString(host) || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return "", ErrInvalid
	}
	if net.ParseIP(host) != nil || strings.Contains(parsed.EscapedPath(), "..") || path.Clean(parsed.Path) != strings.TrimSuffix(parsed.Path, "/") && parsed.Path != "/" {
		return "", ErrInvalid
	}
	if directory && !strings.HasSuffix(parsed.Path, "/") {
		return "", ErrInvalid
	}
	parsed.Host = host
	return parsed.String(), nil
}

func normalizePlatform(value *PlatformConfig) {
	value.PackageName = strings.TrimSpace(value.PackageName)
	value.AppSignature = strings.ToUpper(strings.TrimSpace(value.AppSignature))
	value.BundleID = strings.TrimSpace(value.BundleID)
	value.UniversalLink = strings.TrimSpace(value.UniversalLink)
	value.BundleName = strings.TrimSpace(value.BundleName)
}

// NormalizeConfig rejects unknown fields and normalizes the exact v1 public
// shape. When ready is true all provider-console fields needed for activation
// must be present.
func NormalizeConfig(providerCode, externalClientID string, raw json.RawMessage, ready bool) (string, json.RawMessage, error) {
	providerCode = strings.ToLower(strings.TrimSpace(providerCode))
	externalClientID = strings.TrimSpace(externalClientID)
	if len(externalClientID) < 2 || len(externalClientID) > 255 {
		return "", nil, ErrInvalid
	}
	var normalized any
	switch providerCode {
	case ProviderWechat:
		if !wechatAppIDPattern.MatchString(externalClientID) {
			return "", nil, ErrInvalid
		}
		value, err := decodeStrict[WechatPublicConfig](raw)
		if err != nil {
			return "", nil, err
		}
		normalizePlatform(&value.Android)
		normalizePlatform(&value.IOS)
		normalizePlatform(&value.Harmony)
		if ready && !value.Android.Enabled && !value.IOS.Enabled && !value.Harmony.Enabled {
			return "", nil, ErrInvalid
		}
		if value.Android.Enabled && (!packagePattern.MatchString(value.Android.PackageName) || !signaturePattern.MatchString(value.Android.AppSignature)) {
			return "", nil, ErrInvalid
		}
		if value.IOS.Enabled {
			canonicalLink, linkErr := CanonicalHTTPSAppLink(value.IOS.UniversalLink, true)
			if !packagePattern.MatchString(value.IOS.BundleID) || linkErr != nil {
				return "", nil, ErrInvalid
			}
			value.IOS.UniversalLink = canonicalLink
		}
		if value.Harmony.Enabled && !packagePattern.MatchString(value.Harmony.BundleName) {
			return "", nil, ErrInvalid
		}
		normalized = value
	case ProviderGitHub:
		if !clientIDPattern.MatchString(externalClientID) {
			return "", nil, ErrInvalid
		}
		value, err := decodeStrict[GitHubPublicConfig](raw)
		if err != nil {
			return "", nil, err
		}
		value.AppReturnURI = strings.TrimSpace(value.AppReturnURI)
		canonicalReturnURI, linkErr := CanonicalHTTPSAppLink(value.AppReturnURI, false)
		if linkErr != nil {
			return "", nil, ErrInvalid
		}
		value.AppReturnURI = canonicalReturnURI
		normalized = value
	case ProviderApple:
		if !packagePattern.MatchString(externalClientID) {
			return "", nil, ErrInvalid
		}
		value, err := decodeStrict[ApplePublicConfig](raw)
		if err != nil {
			return "", nil, err
		}
		value.TeamID, value.KeyID = strings.ToUpper(strings.TrimSpace(value.TeamID)), strings.ToUpper(strings.TrimSpace(value.KeyID))
		if !appleTeamKeyPattern.MatchString(value.TeamID) || !appleTeamKeyPattern.MatchString(value.KeyID) {
			return "", nil, ErrInvalid
		}
		normalized = value
	case ProviderGoogle:
		if !googleClientPattern.MatchString(externalClientID) {
			return "", nil, ErrInvalid
		}
		value, err := decodeStrict[GooglePublicConfig](raw)
		if err != nil {
			return "", nil, err
		}
		value.AndroidPackageName = strings.TrimSpace(value.AndroidPackageName)
		if !packagePattern.MatchString(value.AndroidPackageName) || len(value.AndroidCertificateSHA256) == 0 || len(value.AndroidCertificateSHA256) > 16 {
			return "", nil, ErrInvalid
		}
		seen := map[string]struct{}{}
		for index, fingerprint := range value.AndroidCertificateSHA256 {
			fingerprint = strings.ToUpper(strings.TrimSpace(fingerprint))
			if !sha256ColonPattern.MatchString(fingerprint) {
				return "", nil, ErrInvalid
			}
			if _, exists := seen[fingerprint]; exists {
				return "", nil, ErrInvalid
			}
			seen[fingerprint] = struct{}{}
			value.AndroidCertificateSHA256[index] = fingerprint
		}
		sort.Strings(value.AndroidCertificateSHA256)
		normalized = value
	default:
		return "", nil, ErrInvalid
	}
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return "", nil, ErrInvalid
	}
	return externalClientID, canonical, nil
}

func requiredSecretNames(providerCode string) []string {
	switch providerCode {
	case ProviderWechat:
		return []string{"app_secret"}
	case ProviderGitHub:
		return []string{"client_secret"}
	case ProviderApple:
		return []string{"private_key_p8"}
	case ProviderGoogle:
		return []string{}
	default:
		return nil
	}
}

func ValidateSecrets(providerCode string, values map[string]string) ([]string, string, map[string]string, error) {
	required := requiredSecretNames(providerCode)
	if required == nil {
		return nil, "", nil, ErrInvalid
	}
	if len(required) == 0 {
		if len(values) != 0 {
			return nil, "", nil, ErrInvalid
		}
		return []string{}, "", map[string]string{}, nil
	}
	if len(values) != len(required) {
		return nil, "", nil, ErrInvalid
	}
	clean := make(map[string]string, len(values))
	for _, name := range required {
		value, exists := values[name]
		value = strings.TrimSpace(value)
		if !exists || value == "" || len(value) > 8192 {
			return nil, "", nil, ErrInvalid
		}
		if providerCode == ProviderApple && (!strings.HasPrefix(value, "-----BEGIN PRIVATE KEY-----") || !strings.HasSuffix(value, "-----END PRIVATE KEY-----")) {
			return nil, "", nil, ErrInvalid
		}
		clean[name] = value
	}
	names := append([]string(nil), required...)
	sort.Strings(names)
	hasher := sha256.New()
	hasher.Write([]byte(providerCode))
	for _, name := range names {
		hasher.Write([]byte{0})
		hasher.Write([]byte(name))
		hasher.Write([]byte{0})
		hasher.Write([]byte(clean[name]))
	}
	return names, hex.EncodeToString(hasher.Sum(nil)), clean, nil
}

func BuildConfig(providerCode, externalClientID string, raw json.RawMessage, callbackURI string) (map[string]any, error) {
	result := map[string]any{}
	switch providerCode {
	case ProviderWechat:
		value, err := decodeStrict[WechatPublicConfig](raw)
		if err != nil {
			return nil, err
		}
		result = map[string]any{"app_id": externalClientID}
		if value.Android.Enabled {
			result["android_package_name"], result["android_app_signature"] = value.Android.PackageName, value.Android.AppSignature
		}
		if value.IOS.Enabled {
			result["ios_bundle_id"], result["ios_universal_link"] = value.IOS.BundleID, value.IOS.UniversalLink
		}
		if value.Harmony.Enabled {
			result["harmony_bundle_name"] = value.Harmony.BundleName
		}
	case ProviderGitHub:
		value, err := decodeStrict[GitHubPublicConfig](raw)
		if err != nil {
			return nil, err
		}
		result = map[string]any{"client_id": externalClientID, "callback_uri": callbackURI, "app_return_uri": value.AppReturnURI}
	case ProviderApple:
		if _, err := decodeStrict[ApplePublicConfig](raw); err != nil {
			return nil, err
		}
		result = map[string]any{"client_id": externalClientID}
	case ProviderGoogle:
		value, err := decodeStrict[GooglePublicConfig](raw)
		if err != nil {
			return nil, err
		}
		result = map[string]any{"server_client_id": externalClientID, "android_package_name": value.AndroidPackageName, "android_certificate_sha256": value.AndroidCertificateSHA256}
	default:
		return nil, ErrInvalid
	}
	return result, nil
}
