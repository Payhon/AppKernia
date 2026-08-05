package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddr                     = ":8080"
	defaultShutdown                     = 15 * time.Second
	developmentDatabaseDSN              = "postgres://appkernia:appkernia-dev-only@localhost:55432/appkernia?sslmode=disable"
	developmentLoginProtectionKeyBase64 = "YXBwa2VybmlhLWRldi1sb2dpbi1rZXktMzJieXRlcyE="
)

type Config struct {
	Environment                 string
	HTTPAddr                    string
	DatabaseURL                 string
	AdminOrigin                 string
	JWTKeyID                    string
	JWTPrivateKey               string
	LoginProtectionKeyBase64    string
	AdminRegistrationEnabled    bool
	AdminRegistrationTenantCode string
	PasswordRecoveryEnabled     bool
	PasswordRecoveryAdapter     string
	AvatarUploadEnabled         bool
	FileStorageEnabled          bool
	MultiTenantEnabled          bool
	APIClientsEnabled           bool
	WebhooksEnabled             bool
	WebhookAdapter              string
	MFAEnabled                  bool
	OAuthEnabled                bool
	OAuthAdapter                string
	ConfigMasterKeyBase64       string
	ConfigMasterKeyVersion      int32
	ObjectStorageAdapter        string
	LocalObjectStorageDir       string
	ShutdownTimeout             time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Environment:                 os.Getenv("AK_ENV"),
		HTTPAddr:                    os.Getenv("AK_HTTP_ADDR"),
		DatabaseURL:                 os.Getenv("AK_DATABASE_URL"),
		AdminOrigin:                 os.Getenv("AK_ADMIN_ORIGIN"),
		JWTKeyID:                    os.Getenv("AK_JWT_KEY_ID"),
		JWTPrivateKey:               os.Getenv("AK_JWT_PRIVATE_KEY_BASE64"),
		LoginProtectionKeyBase64:    strings.TrimSpace(os.Getenv("AK_LOGIN_PROTECTION_KEY_BASE64")),
		AdminRegistrationTenantCode: strings.TrimSpace(os.Getenv("AK_ADMIN_REGISTRATION_TENANT_CODE")),
		PasswordRecoveryAdapter:     strings.ToLower(strings.TrimSpace(os.Getenv("AK_PASSWORD_RECOVERY_ADAPTER"))),
		ObjectStorageAdapter:        strings.ToLower(strings.TrimSpace(os.Getenv("AK_OBJECT_STORAGE_ADAPTER"))),
		WebhookAdapter:              strings.ToLower(strings.TrimSpace(os.Getenv("AK_WEBHOOK_ADAPTER"))),
		OAuthAdapter:                strings.ToLower(strings.TrimSpace(os.Getenv("AK_OAUTH_ADAPTER"))),
		LocalObjectStorageDir:       strings.TrimSpace(os.Getenv("AK_LOCAL_OBJECT_STORAGE_DIR")),
		ConfigMasterKeyBase64:       strings.TrimSpace(os.Getenv("AK_CONFIG_MASTER_KEY_BASE64")),
	}
	var err error
	if cfg.AdminRegistrationEnabled, err = optionalBool("AK_ADMIN_REGISTRATION_ENABLED"); err != nil {
		return Config{}, err
	}
	if cfg.PasswordRecoveryEnabled, err = optionalBool("AK_PASSWORD_RECOVERY_ENABLED"); err != nil {
		return Config{}, err
	}
	if cfg.AvatarUploadEnabled, err = optionalBool("AK_AVATAR_UPLOAD_ENABLED"); err != nil {
		return Config{}, err
	}
	if cfg.FileStorageEnabled, err = optionalBool("AK_FILE_STORAGE_ENABLED"); err != nil {
		return Config{}, err
	}
	if cfg.MultiTenantEnabled, err = optionalBool("AK_MULTI_TENANT_ENABLED"); err != nil {
		return Config{}, err
	}
	if cfg.APIClientsEnabled, err = optionalBool("AK_API_CLIENTS_ENABLED"); err != nil {
		return Config{}, err
	}
	if cfg.WebhooksEnabled, err = optionalBool("AK_WEBHOOKS_ENABLED"); err != nil {
		return Config{}, err
	}
	if cfg.MFAEnabled, err = optionalBool("AK_MFA_ENABLED"); err != nil {
		return Config{}, err
	}
	if cfg.OAuthEnabled, err = optionalBool("AK_OAUTH_ENABLED"); err != nil {
		return Config{}, err
	}
	if raw := strings.TrimSpace(os.Getenv("AK_CONFIG_MASTER_KEY_VERSION")); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 32)
		if parseErr != nil || value < 1 {
			return Config{}, errors.New("AK_CONFIG_MASTER_KEY_VERSION must be a positive integer")
		}
		cfg.ConfigMasterKeyVersion = int32(value)
	} else {
		cfg.ConfigMasterKeyVersion = 1
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = defaultHTTPAddr
	}
	if cfg.AdminOrigin == "" {
		cfg.AdminOrigin = "http://localhost:4173"
	}
	if cfg.JWTKeyID == "" {
		cfg.JWTKeyID = "development-ephemeral"
	}
	if cfg.LoginProtectionKeyBase64 == "" && cfg.Environment == "development" {
		cfg.LoginProtectionKeyBase64 = developmentLoginProtectionKeyBase64
	}
	if cfg.AdminRegistrationTenantCode == "" {
		cfg.AdminRegistrationTenantCode = "local"
	}
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
	if cfg.OAuthAdapter == "" && cfg.Environment == "development" {
		cfg.OAuthAdapter = "local-mock"
	}
	if cfg.LocalObjectStorageDir == "" && cfg.Environment == "development" {
		cfg.LocalObjectStorageDir = "./var/object-storage"
	}
	if cfg.DatabaseURL == "" && cfg.Environment == "development" {
		cfg.DatabaseURL = developmentDatabaseDSN
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("AK_DATABASE_URL is required outside development")
	}
	if cfg.Environment != "development" && cfg.JWTPrivateKey == "" {
		return Config{}, errors.New("AK_JWT_PRIVATE_KEY_BASE64 is required outside development")
	}
	if cfg.Environment != "development" && cfg.LoginProtectionKeyBase64 == "" {
		return Config{}, errors.New("AK_LOGIN_PROTECTION_KEY_BASE64 is required outside development")
	}
	if key, decodeErr := base64.StdEncoding.DecodeString(cfg.LoginProtectionKeyBase64); decodeErr != nil || len(key) != 32 {
		return Config{}, errors.New("AK_LOGIN_PROTECTION_KEY_BASE64 must encode exactly 32 bytes")
	}
	if cfg.Environment != "development" && cfg.ConfigMasterKeyBase64 == "" {
		return Config{}, errors.New("AK_CONFIG_MASTER_KEY_BASE64 is required outside development")
	}
	if cfg.PasswordRecoveryEnabled && cfg.PasswordRecoveryAdapter == "" {
		return Config{}, errors.New("AK_PASSWORD_RECOVERY_ADAPTER is required when password recovery is enabled")
	}
	if cfg.PasswordRecoveryAdapter == "local" && cfg.Environment != "development" {
		return Config{}, errors.New("AK_PASSWORD_RECOVERY_ADAPTER=local is allowed only in development")
	}
	if cfg.PasswordRecoveryAdapter != "" && cfg.PasswordRecoveryAdapter != "local" {
		return Config{}, fmt.Errorf("AK_PASSWORD_RECOVERY_ADAPTER %q is not configured in this build", cfg.PasswordRecoveryAdapter)
	}
	if (cfg.AvatarUploadEnabled || cfg.FileStorageEnabled) && cfg.ObjectStorageAdapter == "" {
		return Config{}, errors.New("AK_OBJECT_STORAGE_ADAPTER is required when file storage is enabled")
	}
	if cfg.ObjectStorageAdapter == "local" && cfg.Environment != "development" {
		return Config{}, errors.New("AK_OBJECT_STORAGE_ADAPTER=local is allowed only in development")
	}
	if cfg.ObjectStorageAdapter != "" && cfg.ObjectStorageAdapter != "local" && cfg.ObjectStorageAdapter != "configured" {
		return Config{}, fmt.Errorf("AK_OBJECT_STORAGE_ADAPTER %q is not configured in this build", cfg.ObjectStorageAdapter)
	}
	if (cfg.AvatarUploadEnabled || cfg.FileStorageEnabled) && (cfg.ObjectStorageAdapter == "local" || (cfg.ObjectStorageAdapter == "configured" && cfg.Environment == "development")) && cfg.LocalObjectStorageDir == "" {
		return Config{}, errors.New("AK_LOCAL_OBJECT_STORAGE_DIR is required when development storage can use the local driver")
	}
	if cfg.WebhooksEnabled && cfg.WebhookAdapter != "http" && cfg.WebhookAdapter != "local-mock" {
		return Config{}, fmt.Errorf("AK_WEBHOOK_ADAPTER %q is not configured in this build", cfg.WebhookAdapter)
	}
	if cfg.WebhookAdapter == "local-mock" && cfg.Environment != "development" {
		return Config{}, errors.New("AK_WEBHOOK_ADAPTER=local-mock is allowed only in development")
	}
	if cfg.OAuthEnabled && cfg.OAuthAdapter == "" {
		return Config{}, errors.New("AK_OAUTH_ADAPTER is required when OAuth is enabled")
	}
	if cfg.OAuthAdapter == "local-mock" && cfg.Environment != "development" {
		return Config{}, errors.New("AK_OAUTH_ADAPTER=local-mock is allowed only in development")
	}
	if cfg.OAuthAdapter != "" && cfg.OAuthAdapter != "local-mock" {
		return Config{}, fmt.Errorf("AK_OAUTH_ADAPTER %q is not configured in this build", cfg.OAuthAdapter)
	}
	rawTimeout := os.Getenv("AK_SHUTDOWN_TIMEOUT")
	if rawTimeout == "" {
		cfg.ShutdownTimeout = defaultShutdown
	} else {
		timeout, err := time.ParseDuration(rawTimeout)
		if err != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("AK_SHUTDOWN_TIMEOUT must be a positive duration")
		}
		cfg.ShutdownTimeout = timeout
	}
	return cfg, nil
}

func optionalBool(name string) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", name)
	}
	return value, nil
}
