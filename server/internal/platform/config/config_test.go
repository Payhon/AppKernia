package config

import "testing"

func TestLoadDevelopmentDefaults(t *testing.T) {
	t.Setenv("AK_ENV", "development")
	t.Setenv("AK_HTTP_ADDR", "")
	t.Setenv("AK_DATABASE_URL", "")
	t.Setenv("AK_LOGIN_PROTECTION_KEY_BASE64", "")
	t.Setenv("AK_SHUTDOWN_TIMEOUT", "")
	t.Setenv("AK_ADMIN_REGISTRATION_ENABLED", "")
	t.Setenv("AK_PASSWORD_RECOVERY_ENABLED", "")
	t.Setenv("AK_PASSWORD_RECOVERY_ADAPTER", "")
	t.Setenv("AK_AVATAR_UPLOAD_ENABLED", "")
	t.Setenv("AK_FILE_STORAGE_ENABLED", "")
	t.Setenv("AK_MULTI_TENANT_ENABLED", "")
	t.Setenv("AK_MFA_ENABLED", "")
	t.Setenv("AK_OAUTH_ENABLED", "")
	t.Setenv("AK_OAUTH_ADAPTER", "")
	t.Setenv("AK_OBJECT_STORAGE_ADAPTER", "")
	t.Setenv("AK_LOCAL_OBJECT_STORAGE_DIR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("DatabaseURL must have a development default")
	}
	if cfg.AdminOrigin != "http://localhost:4173" {
		t.Fatalf("AdminOrigin = %q", cfg.AdminOrigin)
	}
	if cfg.AdminRegistrationEnabled || cfg.PasswordRecoveryEnabled || cfg.AvatarUploadEnabled || cfg.FileStorageEnabled || cfg.MultiTenantEnabled || cfg.MFAEnabled || cfg.OAuthEnabled || cfg.PasswordRecoveryAdapter != "local" || cfg.ObjectStorageAdapter != "configured" || cfg.OAuthAdapter != "local-mock" {
		t.Fatalf("anonymous auth defaults must fail closed with the development adapter available: %#v", cfg)
	}
	if cfg.LocalObjectStorageDir == "" {
		t.Fatal("development local object storage directory must be configured")
	}
}

func TestLoadOAuthConfigurationFailsClosed(t *testing.T) {
	t.Setenv("AK_ENV", "production")
	t.Setenv("AK_DATABASE_URL", "postgres://example.invalid/appkernia")
	t.Setenv("AK_JWT_PRIVATE_KEY_BASE64", "test-signing-key")
	t.Setenv("AK_LOGIN_PROTECTION_KEY_BASE64", developmentLoginProtectionKeyBase64)
	t.Setenv("AK_CONFIG_MASTER_KEY_BASE64", "dGVzdC1jb25maWctbWFzdGVyLWtleS0zMmJ5dGVzIQ==")
	t.Setenv("AK_OAUTH_ENABLED", "true")
	t.Setenv("AK_OAUTH_ADAPTER", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected a missing OAuth adapter error")
	}
	t.Setenv("AK_OAUTH_ADAPTER", "local-mock")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected the local OAuth adapter to be rejected outside development")
	}
}

func TestLoadAvatarStorageConfigurationFailsClosed(t *testing.T) {
	t.Setenv("AK_ENV", "production")
	t.Setenv("AK_DATABASE_URL", "postgres://example.invalid/appkernia")
	t.Setenv("AK_JWT_PRIVATE_KEY_BASE64", "test-signing-key")
	t.Setenv("AK_LOGIN_PROTECTION_KEY_BASE64", developmentLoginProtectionKeyBase64)
	t.Setenv("AK_CONFIG_MASTER_KEY_BASE64", "configured-test-master-key")
	t.Setenv("AK_AVATAR_UPLOAD_ENABLED", "true")
	t.Setenv("AK_OBJECT_STORAGE_ADAPTER", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected a missing object storage adapter error")
	}
	t.Setenv("AK_OBJECT_STORAGE_ADAPTER", "local")
	t.Setenv("AK_LOCAL_OBJECT_STORAGE_DIR", "/tmp/appkernia-test")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected the local object storage adapter to be rejected outside development")
	}
	t.Setenv("AK_OBJECT_STORAGE_ADAPTER", "configured")
	if _, err := Load(); err != nil {
		t.Fatalf("Load() configured object storage error = %v", err)
	}
}

func TestLoadAnonymousAuthConfigurationFailsClosed(t *testing.T) {
	t.Setenv("AK_ENV", "production")
	t.Setenv("AK_DATABASE_URL", "postgres://example.invalid/appkernia")
	t.Setenv("AK_JWT_PRIVATE_KEY_BASE64", "test-signing-key")
	t.Setenv("AK_LOGIN_PROTECTION_KEY_BASE64", developmentLoginProtectionKeyBase64)
	t.Setenv("AK_PASSWORD_RECOVERY_ENABLED", "true")
	t.Setenv("AK_PASSWORD_RECOVERY_ADAPTER", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected a missing recovery adapter error")
	}
	t.Setenv("AK_PASSWORD_RECOVERY_ADAPTER", "local")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected the local adapter to be rejected outside development")
	}
}

func TestLoadProductionRequiresSigningKey(t *testing.T) {
	t.Setenv("AK_ENV", "production")
	t.Setenv("AK_DATABASE_URL", "postgres://example.invalid/appkernia")
	t.Setenv("AK_JWT_PRIVATE_KEY_BASE64", "")
	t.Setenv("AK_LOGIN_PROTECTION_KEY_BASE64", developmentLoginProtectionKeyBase64)
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected a signing key error")
	}
}

func TestLoadProductionRequiresDatabaseURL(t *testing.T) {
	t.Setenv("AK_ENV", "production")
	t.Setenv("AK_DATABASE_URL", "")
	t.Setenv("AK_LOGIN_PROTECTION_KEY_BASE64", developmentLoginProtectionKeyBase64)
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error")
	}
}

func TestLoadProductionRequiresLoginProtectionKey(t *testing.T) {
	t.Setenv("AK_ENV", "production")
	t.Setenv("AK_DATABASE_URL", "postgres://example.invalid/appkernia")
	t.Setenv("AK_JWT_PRIVATE_KEY_BASE64", "test-signing-key")
	t.Setenv("AK_CONFIG_MASTER_KEY_BASE64", "dGVzdC1jb25maWctbWFzdGVyLWtleS0zMmJ5dGVzIQ==")
	t.Setenv("AK_LOGIN_PROTECTION_KEY_BASE64", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected a login protection key error")
	}
}
