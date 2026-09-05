package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/appkernia/appkernia/server/internal/shared/publicurl"
	"go.yaml.in/yaml/v3"
)

const (
	DatabaseDriverSQLite                = "sqlite"
	DatabaseDriverPostgreSQL            = "postgresql"
	maxConfigBytes                      = 1 << 20
	defaultHTTPAddr                     = "127.0.0.1:8080"
	defaultAdminPath                    = "/admin"
	defaultShutdown                     = 15 * time.Second
	developmentLoginProtectionKeyBase64 = "YXBwa2VybmlhLWRldi1sb2dpbi1rZXktMzJieXRlcyE="
	developmentConfigMasterKeyBase64    = "YXBwa2VybmlhLWRldi1jb25maWcta2V5LTMyYnl0ZSE="
)

// Config is the fully resolved runtime configuration. Load applies values in
// this order: explicit overrides, AK_* environment variables, YAML, defaults.
type Config struct {
	Environment                 string
	StateDir                    string
	HTTPAddr                    string
	PublicWebBaseURL            string
	DatabaseDriver              string
	DatabaseURL                 string
	SQLitePath                  string
	AdminOrigin                 string
	AdminPath                   string
	AdminStaticDir              string
	JWTKeyID                    string
	JWTPrivateKey               string
	LoginProtectionKeyBase64    string
	AdminRegistrationEnabled    bool
	AdminRegistrationTenantCode string
	PlatformTenantCode          string
	PasswordRecoveryEnabled     bool
	PasswordRecoveryAdapter     string
	AvatarUploadEnabled         bool
	FileStorageEnabled          bool
	MultiTenantEnabled          bool
	APIClientsEnabled           bool
	WebhooksEnabled             bool
	PushEnabled                 bool
	PushAdapter                 string
	WebhookAdapter              string
	MFAEnabled                  bool
	OAuthEnabled                bool
	OAuthAdapter                string
	OTPAdapter                  string
	ConfigMasterKeyBase64       string
	ConfigMasterKeyVersion      int32
	ObjectStorageAdapter        string
	LocalObjectStorageDir       string
	FeedbackClamdSocket         string
	ShutdownTimeout             time.Duration
	LogLevel                    string
	LogFormat                   string
	LogFile                     string
	FilePath                    string
}

// Overrides contains non-secret values that may safely be supplied as command
// line flags. Empty fields do not override environment or YAML values.
type Overrides struct {
	HTTPAddr       string
	DatabaseDriver string
	SQLitePath     string
	AdminPath      string
	AdminStaticDir string
	LogLevel       string
	LogFormat      string
	LogFile        string
}

type Options struct {
	Path      string
	Overrides Overrides
}

func (cfg Config) AdminBaseURL() string {
	return strings.TrimRight(cfg.AdminOrigin, "/") + cfg.AdminPath
}

type fileConfig struct {
	Environment string `yaml:"environment"`
	StateDir    string `yaml:"state_dir"`
	Server      struct {
		Listen           string `yaml:"listen"`
		PublicWebBaseURL string `yaml:"public_web_base_url"`
		ShutdownTimeout  string `yaml:"shutdown_timeout"`
	} `yaml:"server"`
	Database struct {
		Driver     string `yaml:"driver"`
		URL        string `yaml:"url"`
		SQLitePath string `yaml:"sqlite_path"`
	} `yaml:"database"`
	Admin struct {
		Origin                 string `yaml:"origin"`
		Path                   string `yaml:"path"`
		StaticDir              string `yaml:"static_dir"`
		RegistrationEnabled    bool   `yaml:"registration_enabled"`
		RegistrationTenantCode string `yaml:"registration_tenant_code"`
	} `yaml:"admin"`
	Auth struct {
		JWTKeyID                 string `yaml:"jwt_key_id"`
		JWTPrivateKeyBase64      string `yaml:"jwt_private_key_base64"`
		LoginProtectionKeyBase64 string `yaml:"login_protection_key_base64"`
		ConfigMasterKeyBase64    string `yaml:"config_master_key_base64"`
		ConfigMasterKeyVersion   int32  `yaml:"config_master_key_version"`
	} `yaml:"auth"`
	Features struct {
		PasswordRecovery bool `yaml:"password_recovery"`
		AvatarUpload     bool `yaml:"avatar_upload"`
		FileStorage      bool `yaml:"file_storage"`
		MultiTenant      bool `yaml:"multi_tenant"`
		APIClients       bool `yaml:"api_clients"`
		Webhooks         bool `yaml:"webhooks"`
		Push             bool `yaml:"push"`
		MFA              bool `yaml:"mfa"`
		OAuth            bool `yaml:"oauth"`
	} `yaml:"features"`
	Adapters struct {
		PasswordRecovery string `yaml:"password_recovery"`
		ObjectStorage    string `yaml:"object_storage"`
		Webhook          string `yaml:"webhook"`
		Push             string `yaml:"push"`
		OAuth            string `yaml:"oauth"`
		OTP              string `yaml:"otp"`
	} `yaml:"adapters"`
	Storage struct {
		LocalDir string `yaml:"local_dir"`
	} `yaml:"storage"`
	Feedback struct {
		ClamdSocket string `yaml:"clamd_socket"`
	} `yaml:"feedback"`
	Platform struct {
		TenantCode string `yaml:"tenant_code"`
	} `yaml:"platform"`
	Log struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
		File   string `yaml:"file"`
	} `yaml:"log"`
}

// DefaultPath returns the native per-user configuration location.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user configuration directory: %w", err)
	}
	return filepath.Join(dir, "AppKernia", "akone.yml"), nil
}

func Load(options ...Options) (Config, error) {
	var option Options
	if len(options) > 1 {
		return Config{}, errors.New("config.Load accepts at most one Options value")
	}
	if len(options) == 1 {
		option = options[0]
	}
	path, explicit, err := resolveConfigPath(option.Path)
	if err != nil {
		return Config{}, err
	}
	fileValues, loaded, err := readFile(path, explicit)
	if err != nil {
		return Config{}, err
	}
	cfg, err := configFromFile(fileValues)
	if err != nil {
		return Config{}, err
	}
	if loaded {
		cfg.FilePath = path
		resolveFilePaths(&cfg, filepath.Dir(path))
	}
	if err = applyEnvironment(&cfg, strings.TrimSpace(option.Overrides.DatabaseDriver) != ""); err != nil {
		return Config{}, err
	}
	if err = applyOverrides(&cfg, option.Overrides); err != nil {
		return Config{}, err
	}
	if err = applyDefaultsAndValidate(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func resolveConfigPath(optionPath string) (string, bool, error) {
	path := strings.TrimSpace(optionPath)
	explicit := path != ""
	if path == "" {
		path = strings.TrimSpace(os.Getenv("AK_CONFIG_FILE"))
		explicit = path != ""
	}
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return "", false, err
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false, fmt.Errorf("resolve configuration file: %w", err)
	}
	return abs, explicit, nil
}

func readFile(path string, required bool) (fileConfig, bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) && !required {
		return fileConfig{}, false, nil
	}
	if err != nil {
		return fileConfig{}, false, fmt.Errorf("inspect configuration file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fileConfig{}, false, errors.New("configuration path must be a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return fileConfig{}, false, errors.New("configuration file must be readable and writable only by its owner (mode 0600)")
	}
	file, err := os.Open(path)
	if err != nil {
		return fileConfig{}, false, fmt.Errorf("read configuration file: %w", err)
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	if err != nil {
		return fileConfig{}, false, fmt.Errorf("read configuration file: %w", err)
	}
	if len(content) > maxConfigBytes {
		return fileConfig{}, false, fmt.Errorf("configuration file exceeds %d bytes", maxConfigBytes)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	var values fileConfig
	if err = decoder.Decode(&values); err != nil {
		return fileConfig{}, false, fmt.Errorf("decode configuration file: %w", err)
	}
	var extra any
	if err = decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fileConfig{}, false, errors.New("configuration file must contain one YAML document")
		}
		return fileConfig{}, false, fmt.Errorf("decode configuration file: %w", err)
	}
	return values, true, nil
}

func configFromFile(values fileConfig) (Config, error) {
	var shutdownTimeout time.Duration
	if raw := strings.TrimSpace(values.Server.ShutdownTimeout); raw != "" {
		var err error
		shutdownTimeout, err = time.ParseDuration(raw)
		if err != nil || shutdownTimeout <= 0 {
			return Config{}, errors.New("server.shutdown_timeout must be a positive duration")
		}
	}
	return Config{
		Environment: values.Environment, StateDir: values.StateDir,
		HTTPAddr: values.Server.Listen, PublicWebBaseURL: values.Server.PublicWebBaseURL,
		DatabaseDriver: values.Database.Driver, DatabaseURL: values.Database.URL,
		SQLitePath: values.Database.SQLitePath, AdminOrigin: values.Admin.Origin,
		AdminPath: values.Admin.Path, AdminStaticDir: values.Admin.StaticDir,
		JWTKeyID: values.Auth.JWTKeyID, JWTPrivateKey: values.Auth.JWTPrivateKeyBase64,
		LoginProtectionKeyBase64:    values.Auth.LoginProtectionKeyBase64,
		ConfigMasterKeyBase64:       values.Auth.ConfigMasterKeyBase64,
		ConfigMasterKeyVersion:      values.Auth.ConfigMasterKeyVersion,
		AdminRegistrationEnabled:    values.Admin.RegistrationEnabled,
		AdminRegistrationTenantCode: values.Admin.RegistrationTenantCode,
		PlatformTenantCode:          values.Platform.TenantCode,
		PasswordRecoveryEnabled:     values.Features.PasswordRecovery,
		AvatarUploadEnabled:         values.Features.AvatarUpload,
		FileStorageEnabled:          values.Features.FileStorage,
		MultiTenantEnabled:          values.Features.MultiTenant,
		APIClientsEnabled:           values.Features.APIClients,
		WebhooksEnabled:             values.Features.Webhooks, PushEnabled: values.Features.Push,
		MFAEnabled: values.Features.MFA, OAuthEnabled: values.Features.OAuth,
		PasswordRecoveryAdapter: values.Adapters.PasswordRecovery,
		ObjectStorageAdapter:    values.Adapters.ObjectStorage,
		WebhookAdapter:          values.Adapters.Webhook, PushAdapter: values.Adapters.Push,
		OAuthAdapter: values.Adapters.OAuth, OTPAdapter: values.Adapters.OTP,
		LocalObjectStorageDir: values.Storage.LocalDir,
		FeedbackClamdSocket:   values.Feedback.ClamdSocket,
		LogLevel:              values.Log.Level, LogFormat: values.Log.Format, LogFile: values.Log.File,
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

func resolveFilePaths(cfg *Config, base string) {
	for _, value := range []*string{&cfg.StateDir, &cfg.SQLitePath, &cfg.AdminStaticDir, &cfg.LocalObjectStorageDir, &cfg.LogFile} {
		if *value != "" && !filepath.IsAbs(*value) {
			*value = filepath.Join(base, *value)
		}
	}
}

func applyEnvironment(cfg *Config, databaseOverridden bool) error {
	databaseDriver := strings.TrimSpace(os.Getenv("AK_DATABASE_DRIVER"))
	databaseURL := strings.TrimSpace(os.Getenv("AK_DATABASE_URL"))
	sqlitePath := strings.TrimSpace(os.Getenv("AK_SQLITE_PATH"))
	stringsByEnv := map[string]*string{
		"AK_ENV": &cfg.Environment, "AK_STATE_DIR": &cfg.StateDir,
		"AK_HTTP_ADDR": &cfg.HTTPAddr, "AK_PUBLIC_WEB_BASE_URL": &cfg.PublicWebBaseURL,
		"AK_DATABASE_DRIVER": &cfg.DatabaseDriver, "AK_DATABASE_URL": &cfg.DatabaseURL,
		"AK_ADMIN_ORIGIN": &cfg.AdminOrigin,
		"AK_ADMIN_PATH":   &cfg.AdminPath, "AK_ADMIN_STATIC_DIR": &cfg.AdminStaticDir,
		"AK_JWT_KEY_ID": &cfg.JWTKeyID, "AK_JWT_PRIVATE_KEY_BASE64": &cfg.JWTPrivateKey,
		"AK_LOGIN_PROTECTION_KEY_BASE64":    &cfg.LoginProtectionKeyBase64,
		"AK_ADMIN_REGISTRATION_TENANT_CODE": &cfg.AdminRegistrationTenantCode,
		"AK_PLATFORM_TENANT_CODE":           &cfg.PlatformTenantCode,
		"AK_PASSWORD_RECOVERY_ADAPTER":      &cfg.PasswordRecoveryAdapter,
		"AK_OBJECT_STORAGE_ADAPTER":         &cfg.ObjectStorageAdapter,
		"AK_WEBHOOK_ADAPTER":                &cfg.WebhookAdapter, "AK_PUSH_ADAPTER": &cfg.PushAdapter,
		"AK_OAUTH_ADAPTER": &cfg.OAuthAdapter, "AK_OTP_ADAPTER": &cfg.OTPAdapter,
		"AK_LOCAL_OBJECT_STORAGE_DIR": &cfg.LocalObjectStorageDir,
		"AK_FEEDBACK_CLAMD_SOCKET":    &cfg.FeedbackClamdSocket,
		"AK_CONFIG_MASTER_KEY_BASE64": &cfg.ConfigMasterKeyBase64,
		"AK_LOG_LEVEL":                &cfg.LogLevel, "AK_LOG_FORMAT": &cfg.LogFormat, "AK_LOG_FILE": &cfg.LogFile,
	}
	for name, target := range stringsByEnv {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			*target = value
		}
	}
	if sqlitePath != "" {
		absolute, err := filepath.Abs(sqlitePath)
		if err != nil {
			return fmt.Errorf("resolve AK_SQLITE_PATH: %w", err)
		}
		cfg.SQLitePath = absolute
	}
	if databaseDriver == "" {
		if !databaseOverridden && sqlitePath != "" && databaseURL != "" {
			return errors.New("AK_SQLITE_PATH and AK_DATABASE_URL require an explicit AK_DATABASE_DRIVER when both are set")
		}
		switch {
		case sqlitePath != "":
			cfg.DatabaseDriver = DatabaseDriverSQLite
		case databaseURL != "":
			cfg.DatabaseDriver = DatabaseDriverPostgreSQL
		}
	}
	boolsByEnv := map[string]*bool{
		"AK_ADMIN_REGISTRATION_ENABLED": &cfg.AdminRegistrationEnabled,
		"AK_PASSWORD_RECOVERY_ENABLED":  &cfg.PasswordRecoveryEnabled,
		"AK_AVATAR_UPLOAD_ENABLED":      &cfg.AvatarUploadEnabled,
		"AK_FILE_STORAGE_ENABLED":       &cfg.FileStorageEnabled,
		"AK_MULTI_TENANT_ENABLED":       &cfg.MultiTenantEnabled,
		"AK_API_CLIENTS_ENABLED":        &cfg.APIClientsEnabled,
		"AK_WEBHOOKS_ENABLED":           &cfg.WebhooksEnabled, "AK_PUSH_ENABLED": &cfg.PushEnabled,
		"AK_MFA_ENABLED": &cfg.MFAEnabled, "AK_OAUTH_ENABLED": &cfg.OAuthEnabled,
	}
	for name, target := range boolsByEnv {
		if err := optionalBool(target, name); err != nil {
			return err
		}
	}
	if raw := strings.TrimSpace(os.Getenv("AK_CONFIG_MASTER_KEY_VERSION")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || value < 1 {
			return errors.New("AK_CONFIG_MASTER_KEY_VERSION must be a positive integer")
		}
		cfg.ConfigMasterKeyVersion = int32(value)
	}
	if raw := strings.TrimSpace(os.Getenv("AK_SHUTDOWN_TIMEOUT")); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			return errors.New("AK_SHUTDOWN_TIMEOUT must be a positive duration")
		}
		cfg.ShutdownTimeout = value
	}
	return nil
}

func applyOverrides(cfg *Config, values Overrides) error {
	if value := strings.TrimSpace(values.HTTPAddr); value != "" {
		cfg.HTTPAddr = value
	}
	if value := strings.TrimSpace(values.DatabaseDriver); value != "" {
		cfg.DatabaseDriver = value
	}
	if value := strings.TrimSpace(values.SQLitePath); value != "" {
		absolute, err := filepath.Abs(value)
		if err != nil {
			return fmt.Errorf("resolve SQLite database path: %w", err)
		}
		cfg.SQLitePath = absolute
	}
	if value := strings.TrimSpace(values.AdminPath); value != "" {
		cfg.AdminPath = value
	}
	if value := strings.TrimSpace(values.AdminStaticDir); value != "" {
		cfg.AdminStaticDir = value
	}
	if value := strings.TrimSpace(values.LogLevel); value != "" {
		cfg.LogLevel = value
	}
	if value := strings.TrimSpace(values.LogFormat); value != "" {
		cfg.LogFormat = value
	}
	if value := strings.TrimSpace(values.LogFile); value != "" {
		cfg.LogFile = value
	}
	return nil
}

func applyDefaultsAndValidate(cfg *Config) error {
	cfg.Environment = strings.ToLower(strings.TrimSpace(cfg.Environment))
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = defaultHTTPAddr
	}
	if cfg.PublicWebBaseURL == "" && cfg.Environment == "development" {
		cfg.PublicWebBaseURL = "http://127.0.0.1:8080"
	}
	cfg.PublicWebBaseURL = strings.TrimRight(strings.TrimSpace(cfg.PublicWebBaseURL), "/")
	if err := publicurl.Validate(cfg.PublicWebBaseURL, cfg.Environment == "development" || cfg.Environment == "test"); err != nil {
		return err
	}
	if cfg.AdminOrigin == "" {
		cfg.AdminOrigin = "http://127.0.0.1:8080"
	}
	if cfg.AdminPath == "" {
		cfg.AdminPath = defaultAdminPath
	}
	var err error
	if cfg.AdminPath, err = normalizeAdminPath(cfg.AdminPath); err != nil {
		return err
	}
	if cfg.JWTKeyID == "" {
		cfg.JWTKeyID = "development-ephemeral"
	}
	if cfg.LoginProtectionKeyBase64 == "" && cfg.Environment == "development" {
		cfg.LoginProtectionKeyBase64 = developmentLoginProtectionKeyBase64
	}
	if cfg.ConfigMasterKeyBase64 == "" && cfg.Environment == "development" {
		cfg.ConfigMasterKeyBase64 = developmentConfigMasterKeyBase64
	}
	if cfg.ConfigMasterKeyVersion == 0 {
		cfg.ConfigMasterKeyVersion = 1
	}
	if cfg.ConfigMasterKeyVersion < 1 {
		return errors.New("config master key version must be a positive integer")
	}
	if cfg.AdminRegistrationTenantCode == "" {
		cfg.AdminRegistrationTenantCode = "local"
	}
	cfg.PlatformTenantCode = strings.ToLower(strings.TrimSpace(cfg.PlatformTenantCode))
	if cfg.PlatformTenantCode == "" {
		cfg.PlatformTenantCode = "local"
	}
	cfg.PasswordRecoveryAdapter = strings.ToLower(strings.TrimSpace(cfg.PasswordRecoveryAdapter))
	cfg.ObjectStorageAdapter = strings.ToLower(strings.TrimSpace(cfg.ObjectStorageAdapter))
	cfg.WebhookAdapter = strings.ToLower(strings.TrimSpace(cfg.WebhookAdapter))
	cfg.PushAdapter = strings.ToLower(strings.TrimSpace(cfg.PushAdapter))
	cfg.OAuthAdapter = strings.ToLower(strings.TrimSpace(cfg.OAuthAdapter))
	cfg.OTPAdapter = strings.ToLower(strings.TrimSpace(cfg.OTPAdapter))
	if cfg.PasswordRecoveryAdapter == "" && cfg.Environment == "development" {
		cfg.PasswordRecoveryAdapter = "local"
	}
	if cfg.ObjectStorageAdapter == "" && cfg.Environment == "development" {
		cfg.ObjectStorageAdapter = "configured"
	}
	if cfg.WebhookAdapter == "" {
		if cfg.Environment == "development" {
			cfg.WebhookAdapter = "local-mock"
		} else {
			cfg.WebhookAdapter = "http"
		}
	}
	if cfg.PushAdapter == "" {
		if cfg.Environment == "development" {
			cfg.PushAdapter = "local-mock"
		} else {
			cfg.PushAdapter = "official"
		}
	}
	if cfg.OAuthAdapter == "" && cfg.Environment == "development" {
		cfg.OAuthAdapter = "local-mock"
	}
	if cfg.OTPAdapter == "" {
		if cfg.Environment == "development" || cfg.Environment == "test" {
			cfg.OTPAdapter = "database-capture"
		} else {
			cfg.OTPAdapter = "notification"
		}
	}
	if cfg.LocalObjectStorageDir == "" && cfg.Environment == "development" {
		cfg.LocalObjectStorageDir = "./var/object-storage"
	}
	cfg.DatabaseDriver = strings.ToLower(strings.TrimSpace(cfg.DatabaseDriver))
	cfg.DatabaseURL = strings.TrimSpace(cfg.DatabaseURL)
	cfg.SQLitePath = strings.TrimSpace(cfg.SQLitePath)
	if cfg.DatabaseDriver == "" {
		if cfg.DatabaseURL != "" && cfg.SQLitePath != "" {
			return errors.New("database.url and database.sqlite_path require an explicit database.driver when both are set")
		}
		if cfg.DatabaseURL != "" {
			cfg.DatabaseDriver = DatabaseDriverPostgreSQL
		} else {
			cfg.DatabaseDriver = DatabaseDriverSQLite
		}
	}
	if cfg.DatabaseDriver != DatabaseDriverSQLite && cfg.DatabaseDriver != DatabaseDriverPostgreSQL {
		return errors.New("database driver must be sqlite or postgresql")
	}
	if cfg.DatabaseDriver == DatabaseDriverPostgreSQL && cfg.DatabaseURL == "" {
		return errors.New("AK_DATABASE_URL is required when database driver is postgresql")
	}
	if cfg.DatabaseDriver == DatabaseDriverSQLite && cfg.SQLitePath == "" {
		var err error
		cfg.SQLitePath, err = defaultSQLitePath()
		if err != nil {
			return err
		}
	}
	if cfg.DatabaseDriver == DatabaseDriverSQLite {
		unsupported := make([]string, 0, 7)
		for name, enabled := range map[string]bool{
			"api_clients": cfg.APIClientsEnabled, "avatar_upload": cfg.AvatarUploadEnabled,
			"file_storage": cfg.FileStorageEnabled, "mfa": cfg.MFAEnabled,
			"oauth": cfg.OAuthEnabled, "push": cfg.PushEnabled, "webhooks": cfg.WebhooksEnabled,
		} {
			if enabled {
				unsupported = append(unsupported, name)
			}
		}
		if cfg.PasswordRecoveryEnabled && (cfg.Environment != "development" || cfg.PasswordRecoveryAdapter != "local") {
			unsupported = append(unsupported, "password_recovery")
		}
		if len(unsupported) != 0 {
			slices.Sort(unsupported)
			return fmt.Errorf("SQLite standalone does not support enabled features: %s", strings.Join(unsupported, ", "))
		}
	}
	if cfg.Environment != "development" && cfg.JWTPrivateKey == "" {
		return errors.New("AK_JWT_PRIVATE_KEY_BASE64 is required outside development")
	}
	if cfg.Environment != "development" && cfg.LoginProtectionKeyBase64 == "" {
		return errors.New("AK_LOGIN_PROTECTION_KEY_BASE64 is required outside development")
	}
	if key, decodeErr := base64.StdEncoding.DecodeString(cfg.LoginProtectionKeyBase64); decodeErr != nil || len(key) != 32 {
		return errors.New("AK_LOGIN_PROTECTION_KEY_BASE64 must encode exactly 32 bytes")
	}
	if cfg.Environment != "development" && cfg.ConfigMasterKeyBase64 == "" {
		return errors.New("AK_CONFIG_MASTER_KEY_BASE64 is required outside development")
	}
	if cfg.PasswordRecoveryEnabled && cfg.PasswordRecoveryAdapter == "" {
		return errors.New("AK_PASSWORD_RECOVERY_ADAPTER is required when password recovery is enabled")
	}
	if cfg.PasswordRecoveryAdapter == "local" && cfg.Environment != "development" {
		return errors.New("AK_PASSWORD_RECOVERY_ADAPTER=local is allowed only in development")
	}
	if cfg.PasswordRecoveryAdapter != "" && cfg.PasswordRecoveryAdapter != "local" && cfg.PasswordRecoveryAdapter != "notification" {
		return fmt.Errorf("AK_PASSWORD_RECOVERY_ADAPTER %q is not configured in this build", cfg.PasswordRecoveryAdapter)
	}
	if (cfg.AvatarUploadEnabled || cfg.FileStorageEnabled) && cfg.ObjectStorageAdapter == "" {
		return errors.New("AK_OBJECT_STORAGE_ADAPTER is required when file storage is enabled")
	}
	if cfg.ObjectStorageAdapter == "local" && cfg.Environment != "development" {
		return errors.New("AK_OBJECT_STORAGE_ADAPTER=local is allowed only in development")
	}
	if cfg.ObjectStorageAdapter != "" && cfg.ObjectStorageAdapter != "local" && cfg.ObjectStorageAdapter != "configured" {
		return fmt.Errorf("AK_OBJECT_STORAGE_ADAPTER %q is not configured in this build", cfg.ObjectStorageAdapter)
	}
	if (cfg.AvatarUploadEnabled || cfg.FileStorageEnabled) && (cfg.ObjectStorageAdapter == "local" || (cfg.ObjectStorageAdapter == "configured" && cfg.Environment == "development")) && cfg.LocalObjectStorageDir == "" {
		return errors.New("AK_LOCAL_OBJECT_STORAGE_DIR is required when development storage can use the local driver")
	}
	if cfg.WebhooksEnabled && cfg.WebhookAdapter != "http" && cfg.WebhookAdapter != "local-mock" {
		return fmt.Errorf("AK_WEBHOOK_ADAPTER %q is not configured in this build", cfg.WebhookAdapter)
	}
	if cfg.WebhookAdapter == "local-mock" && cfg.Environment != "development" {
		return errors.New("AK_WEBHOOK_ADAPTER=local-mock is allowed only in development")
	}
	if cfg.PushAdapter != "local-mock" && cfg.PushAdapter != "official" {
		return fmt.Errorf("AK_PUSH_ADAPTER %q is not configured in this build", cfg.PushAdapter)
	}
	if cfg.PushAdapter == "local-mock" && cfg.Environment != "development" {
		return errors.New("AK_PUSH_ADAPTER=local-mock is allowed only in development")
	}
	if cfg.OAuthEnabled && cfg.OAuthAdapter == "" {
		return errors.New("AK_OAUTH_ADAPTER is required when OAuth is enabled")
	}
	if cfg.OAuthAdapter == "local-mock" && cfg.Environment != "development" {
		return errors.New("AK_OAUTH_ADAPTER=local-mock is allowed only in development")
	}
	if cfg.OAuthAdapter != "" && cfg.OAuthAdapter != "local-mock" {
		return fmt.Errorf("AK_OAUTH_ADAPTER %q is not configured in this build", cfg.OAuthAdapter)
	}
	if cfg.OTPAdapter == "database-capture" && cfg.Environment != "development" && cfg.Environment != "test" {
		return errors.New("AK_OTP_ADAPTER=database-capture is allowed only in development or test")
	}
	if cfg.OTPAdapter != "database-capture" && cfg.OTPAdapter != "notification" {
		return fmt.Errorf("AK_OTP_ADAPTER %q is not configured in this build", cfg.OTPAdapter)
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = defaultShutdown
	}
	if cfg.ShutdownTimeout < 0 {
		return errors.New("shutdown timeout must be a positive duration")
	}
	cfg.LogLevel = strings.ToLower(strings.TrimSpace(cfg.LogLevel))
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.LogLevel != "debug" && cfg.LogLevel != "info" && cfg.LogLevel != "warn" && cfg.LogLevel != "error" {
		return errors.New("log level must be debug, info, warn, or error")
	}
	cfg.LogFormat = strings.ToLower(strings.TrimSpace(cfg.LogFormat))
	if cfg.LogFormat == "" {
		cfg.LogFormat = "text"
	}
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		return errors.New("log format must be text or json")
	}
	return nil
}

func defaultSQLitePath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path for SQLite database: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return "", fmt.Errorf("resolve executable path for SQLite database: %w", err)
	}
	return filepath.Join(filepath.Dir(executable), "data", "appkernia.db"), nil
}

func normalizeAdminPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultAdminPath, nil
	}
	if value == "/" || !strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.ContainsAny(value, "\\?#%") {
		return "", errors.New("AK_ADMIN_PATH must be an absolute URL path without a trailing slash")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", errors.New("AK_ADMIN_PATH contains a control character")
		}
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "." || segment == ".." || segment == "" && strings.Contains(value, "//") {
			return "", errors.New("AK_ADMIN_PATH contains an invalid path segment")
		}
	}
	for _, reserved := range []string{"/api", "/admin-api", "/ws", "/internal", "/h5", "/s", "/assets", "/brand", "/openapi"} {
		if value == reserved || strings.HasPrefix(value, reserved+"/") {
			return "", fmt.Errorf("AK_ADMIN_PATH conflicts with reserved path %s", reserved)
		}
	}
	return value, nil
}

func optionalBool(target *bool, name string) error {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fmt.Errorf("%s must be a boolean", name)
	}
	*target = value
	return nil
}
