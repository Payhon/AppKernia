package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLoadDevelopmentDefaults(t *testing.T) {
	t.Setenv("AK_ENV", "development")
	t.Setenv("AK_HTTP_ADDR", "")
	t.Setenv("AK_DATABASE_DRIVER", "")
	t.Setenv("AK_DATABASE_URL", "")
	t.Setenv("AK_SQLITE_PATH", "")
	t.Setenv("AK_LOGIN_PROTECTION_KEY_BASE64", "")
	t.Setenv("AK_SHUTDOWN_TIMEOUT", "")
	t.Setenv("AK_ADMIN_REGISTRATION_ENABLED", "")
	t.Setenv("AK_PLATFORM_TENANT_CODE", "")
	t.Setenv("AK_PASSWORD_RECOVERY_ENABLED", "")
	t.Setenv("AK_PASSWORD_RECOVERY_ADAPTER", "")
	t.Setenv("AK_AVATAR_UPLOAD_ENABLED", "")
	t.Setenv("AK_FILE_STORAGE_ENABLED", "")
	t.Setenv("AK_MULTI_TENANT_ENABLED", "")
	t.Setenv("AK_MFA_ENABLED", "")
	t.Setenv("AK_OAUTH_ENABLED", "")
	t.Setenv("AK_OAUTH_ADAPTER", "")
	t.Setenv("AK_OTP_ADAPTER", "")
	t.Setenv("AK_OBJECT_STORAGE_ADAPTER", "")
	t.Setenv("AK_LOCAL_OBJECT_STORAGE_DIR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:8080" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		t.Fatal(err)
	}
	wantSQLitePath := filepath.Join(filepath.Dir(executable), "data", "appkernia.db")
	if cfg.DatabaseDriver != "sqlite" || cfg.DatabaseURL != "" || cfg.SQLitePath != wantSQLitePath {
		t.Fatalf("unexpected database defaults: driver=%q url=%q sqlite_path=%q", cfg.DatabaseDriver, cfg.DatabaseURL, cfg.SQLitePath)
	}
	if cfg.AdminOrigin != "http://127.0.0.1:8080" {
		t.Fatalf("AdminOrigin = %q", cfg.AdminOrigin)
	}
	if cfg.AdminBaseURL() != "http://127.0.0.1:8080/admin" {
		t.Fatalf("AdminBaseURL = %q", cfg.AdminBaseURL())
	}
	if cfg.PlatformTenantCode != "local" {
		t.Fatalf("PlatformTenantCode = %q", cfg.PlatformTenantCode)
	}
	if cfg.AdminRegistrationEnabled || cfg.PasswordRecoveryEnabled || cfg.AvatarUploadEnabled || cfg.FileStorageEnabled || cfg.MultiTenantEnabled || cfg.MFAEnabled || cfg.OAuthEnabled || cfg.PasswordRecoveryAdapter != "local" || cfg.ObjectStorageAdapter != "configured" || cfg.OAuthAdapter != "local-mock" || cfg.OTPAdapter != "database-capture" {
		t.Fatalf("anonymous auth defaults must fail closed with the development adapter available: %#v", cfg)
	}
	if cfg.LocalObjectStorageDir == "" {
		t.Fatal("development local object storage directory must be configured")
	}
}

func TestLoadYAMLWithEnvironmentAndExplicitOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "akone.yml")
	content := []byte(`environment: development
server:
  listen: 127.0.0.1:18080
  shutdown_timeout: 9s
database:
  url: postgres://yaml.example/appkernia
admin:
  path: /console
  static_dir: ./admin-dist
log:
  level: warn
  format: json
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AK_HTTP_ADDR", "127.0.0.1:28080")
	t.Setenv("AK_DATABASE_URL", "")
	t.Setenv("AK_ADMIN_PATH", "")
	t.Setenv("AK_LOG_LEVEL", "")
	cfg, err := Load(Options{Path: path, Overrides: Overrides{HTTPAddr: "127.0.0.1:38080"}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != "127.0.0.1:38080" || cfg.DatabaseDriver != "postgresql" || cfg.DatabaseURL != "postgres://yaml.example/appkernia" || cfg.AdminPath != "/console" {
		t.Fatalf("unexpected precedence result: %#v", cfg)
	}
	if cfg.AdminStaticDir != filepath.Join(filepath.Dir(path), "admin-dist") || cfg.ShutdownTimeout != 9*time.Second || cfg.LogLevel != "warn" || cfg.LogFormat != "json" {
		t.Fatalf("unexpected YAML result: %#v", cfg)
	}
	if cfg.AdminBaseURL() != "http://127.0.0.1:8080/console" {
		t.Fatalf("AdminBaseURL = %q", cfg.AdminBaseURL())
	}
}

func TestLoadSQLitePathSourceBasesAndOverrides(t *testing.T) {
	configDir := t.TempDir()
	path := filepath.Join(configDir, "akone.yml")
	content := []byte("environment: development\ndatabase:\n  driver: sqlite\n  sqlite_path: ./yaml/appkernia.db\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AK_DATABASE_DRIVER", "")
	t.Setenv("AK_DATABASE_URL", "")
	t.Setenv("AK_SQLITE_PATH", "")
	cfg, err := Load(Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(configDir, "yaml", "appkernia.db"); cfg.SQLitePath != want {
		t.Fatalf("YAML SQLitePath = %q, want %q", cfg.SQLitePath, want)
	}

	t.Setenv("AK_SQLITE_PATH", filepath.Join("environment", "appkernia.db"))
	cfg, err = Load(Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	wantEnvironment, err := filepath.Abs(filepath.Join("environment", "appkernia.db"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SQLitePath != wantEnvironment {
		t.Fatalf("environment SQLitePath = %q, want %q", cfg.SQLitePath, wantEnvironment)
	}

	cfg, err = Load(Options{Path: path, Overrides: Overrides{DatabaseDriver: "sqlite", SQLitePath: filepath.Join("override", "appkernia.db")}})
	if err != nil {
		t.Fatal(err)
	}
	wantOverride, err := filepath.Abs(filepath.Join("override", "appkernia.db"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseDriver != "sqlite" || cfg.SQLitePath != wantOverride {
		t.Fatalf("override database = driver %q path %q, want sqlite %q", cfg.DatabaseDriver, cfg.SQLitePath, wantOverride)
	}
}

func TestLoadDatabaseDriverInferenceAndValidation(t *testing.T) {
	t.Run("postgresql inferred from URL", func(t *testing.T) {
		t.Setenv("AK_DATABASE_DRIVER", "")
		t.Setenv("AK_DATABASE_URL", "postgres://example.invalid/appkernia")
		t.Setenv("AK_SQLITE_PATH", "")
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DatabaseDriver != "postgresql" {
			t.Fatalf("DatabaseDriver = %q", cfg.DatabaseDriver)
		}
	})
	t.Run("unknown driver", func(t *testing.T) {
		t.Setenv("AK_DATABASE_DRIVER", "mysql")
		t.Setenv("AK_DATABASE_URL", "")
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "sqlite or postgresql") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("postgresql URL required", func(t *testing.T) {
		t.Setenv("AK_DATABASE_DRIVER", "postgresql")
		t.Setenv("AK_DATABASE_URL", "")
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AK_DATABASE_URL") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("environment URL overrides YAML SQLite selection", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "akone.yml")
		if err := os.WriteFile(path, []byte("environment: development\ndatabase:\n  driver: sqlite\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AK_DATABASE_DRIVER", "")
		t.Setenv("AK_DATABASE_URL", "postgres://environment.example/appkernia")
		t.Setenv("AK_SQLITE_PATH", "")
		cfg, err := Load(Options{Path: path})
		if err != nil || cfg.DatabaseDriver != DatabaseDriverPostgreSQL {
			t.Fatalf("environment URL did not select PostgreSQL: cfg=%#v err=%v", cfg, err)
		}
	})
	t.Run("environment SQLite path overrides YAML PostgreSQL selection", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "akone.yml")
		if err := os.WriteFile(path, []byte("environment: development\ndatabase:\n  driver: postgresql\n  url: postgres://yaml.example/appkernia\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AK_DATABASE_DRIVER", "")
		t.Setenv("AK_DATABASE_URL", "")
		t.Setenv("AK_SQLITE_PATH", filepath.Join("environment", "appkernia.db"))
		cfg, err := Load(Options{Path: path})
		if err != nil || cfg.DatabaseDriver != DatabaseDriverSQLite {
			t.Fatalf("environment SQLite path did not select SQLite: cfg=%#v err=%v", cfg, err)
		}
	})
	t.Run("SQLite rejects PostgreSQL-only features", func(t *testing.T) {
		t.Setenv("AK_DATABASE_DRIVER", DatabaseDriverSQLite)
		t.Setenv("AK_DATABASE_URL", "")
		t.Setenv("AK_SQLITE_PATH", filepath.Join(t.TempDir(), "appkernia.db"))
		t.Setenv("AK_API_CLIENTS_ENABLED", "true")
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "api_clients") {
			t.Fatalf("unsupported SQLite feature error = %v", err)
		}
	})
	t.Run("ambiguous environment database selection", func(t *testing.T) {
		t.Setenv("AK_DATABASE_DRIVER", "")
		t.Setenv("AK_DATABASE_URL", "postgres://environment.example/appkernia")
		t.Setenv("AK_SQLITE_PATH", filepath.Join(t.TempDir(), "appkernia.db"))
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "explicit AK_DATABASE_DRIVER") {
			t.Fatalf("ambiguous database environment error = %v", err)
		}
		cfg, err := Load(Options{Overrides: Overrides{DatabaseDriver: DatabaseDriverSQLite, SQLitePath: filepath.Join(t.TempDir(), "override.db")}})
		if err != nil || cfg.DatabaseDriver != DatabaseDriverSQLite {
			t.Fatalf("explicit CLI-equivalent override did not win: cfg=%#v err=%v", cfg, err)
		}
	})
	t.Run("ambiguous YAML database selection", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "akone.yml")
		content := []byte("environment: development\ndatabase:\n  url: postgres://yaml.example/appkernia\n  sqlite_path: ./appkernia.db\n")
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AK_DATABASE_DRIVER", "")
		t.Setenv("AK_DATABASE_URL", "")
		t.Setenv("AK_SQLITE_PATH", "")
		if _, err := Load(Options{Path: path}); err == nil || !strings.Contains(err.Error(), "explicit database.driver") {
			t.Fatalf("ambiguous YAML database error = %v", err)
		}
	})
}

func TestLoadConfigurationPathAndBooleanEnvironmentPrecedence(t *testing.T) {
	environmentPath := filepath.Join(t.TempDir(), "environment.yml")
	explicitPath := filepath.Join(t.TempDir(), "explicit.yml")
	for path, content := range map[string]string{
		environmentPath: "environment: development\nadmin:\n  path: /from-environment-file\n",
		explicitPath:    "environment: development\nadmin:\n  path: /from-explicit-file\nfeatures:\n  api_clients: true\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("AK_CONFIG_FILE", environmentPath)
	cfg, err := Load()
	if err != nil || cfg.FilePath != environmentPath || cfg.AdminPath != "/from-environment-file" {
		t.Fatalf("environment configuration path not selected: cfg=%#v err=%v", cfg, err)
	}
	t.Setenv("AK_API_CLIENTS_ENABLED", "false")
	cfg, err = Load(Options{Path: explicitPath})
	if err != nil || cfg.FilePath != explicitPath || cfg.AdminPath != "/from-explicit-file" || cfg.APIClientsEnabled {
		t.Fatalf("explicit path or boolean environment precedence failed: cfg=%#v err=%v", cfg, err)
	}
}

func TestLoadYAMLRejectsUnknownFieldsAndMultipleDocuments(t *testing.T) {
	for name, content := range map[string]string{
		"unknown":   "environment: development\nunknown: true\n",
		"documents": "environment: development\n---\nenvironment: production\n",
		"duration":  "environment: development\nserver:\n  shutdown_timeout: tomorrow\n",
		"log-level": "environment: development\nlog:\n  level: verbose\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "akone.yml")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(Options{Path: path}); err == nil {
				t.Fatal("invalid YAML was accepted")
			}
		})
	}
}

func TestLoadRejectsUnsafeAdminPaths(t *testing.T) {
	for _, value := range []string{"/", "admin", "/admin/", "/admin//nested", "/admin/../api", "/admin%2fapi", "/admin-api"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("AK_ENV", "development")
			t.Setenv("AK_ADMIN_PATH", value)
			if _, err := Load(); err == nil {
				t.Fatalf("unsafe path %q was accepted", value)
			}
		})
	}
}

func TestWriteDefaultPreservesExistingFileAndRedactsSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "akone.yml")
	written, err := WriteDefault(path, false)
	if err != nil || written != path {
		t.Fatalf("WriteDefault() path=%q err=%v", written, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("configuration mode=%v err=%v", info.Mode().Perm(), err)
	}
	if _, err = WriteDefault(path, false); err == nil {
		t.Fatal("existing configuration was overwritten")
	}
	cfg, err := Load(Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIClientsEnabled {
		t.Fatal("generated SQLite starter configuration must fail closed for PostgreSQL-only API Clients")
	}
	if cfg.DatabaseDriver != "sqlite" || cfg.SQLitePath == "" {
		t.Fatalf("generated starter configuration must select SQLite: %#v", cfg)
	}
	cfg.DatabaseURL = "postgres://user:secret@example.test/appkernia"
	cfg.DatabaseDriver = "postgresql"
	cfg.SQLitePath = "safe/appkernia.db"
	cfg.JWTPrivateKey = "private"
	redacted, err := RedactedYAML(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(redacted), "postgres://") || strings.Contains(string(redacted), "jwt_private_key_base64: private") || !strings.Contains(string(redacted), "driver: postgresql") || !strings.Contains(string(redacted), "sqlite_path: safe/appkernia.db") {
		t.Fatalf("redacted configuration leaked a secret:\n%s", redacted)
	}
}

func TestLoadRejectsInsecureConfigurationFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows permissions are enforced by the user profile ACL")
	}
	path := filepath.Join(t.TempDir(), "akone.yml")
	if err := os.WriteFile(path, []byte("environment: development\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(Options{Path: path}); err == nil || !strings.Contains(err.Error(), "mode 0600") {
		t.Fatalf("insecure configuration mode was accepted: %v", err)
	}
}

func TestLoadNormalizesConfiguredPlatformTenantCode(t *testing.T) {
	t.Setenv("AK_ENV", "development")
	t.Setenv("AK_PLATFORM_TENANT_CODE", "  Platform-OPS  ")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.PlatformTenantCode != "platform-ops" {
		t.Fatalf("PlatformTenantCode = %q", cfg.PlatformTenantCode)
	}
}

func TestLoadOTPCaptureFailsClosedOutsideDevelopment(t *testing.T) {
	t.Setenv("AK_ENV", "production")
	t.Setenv("AK_DATABASE_URL", "postgres://example.invalid/appkernia")
	t.Setenv("AK_JWT_PRIVATE_KEY_BASE64", "test-signing-key")
	t.Setenv("AK_LOGIN_PROTECTION_KEY_BASE64", developmentLoginProtectionKeyBase64)
	t.Setenv("AK_CONFIG_MASTER_KEY_BASE64", "dGVzdC1jb25maWctbWFzdGVyLWtleS0zMmJ5dGVzIQ==")
	t.Setenv("AK_OTP_ADAPTER", "database-capture")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted plaintext OTP capture in production")
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
	t.Setenv("AK_DATABASE_DRIVER", "postgresql")
	t.Setenv("AK_DATABASE_URL", "")
	t.Setenv("AK_LOGIN_PROTECTION_KEY_BASE64", developmentLoginProtectionKeyBase64)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AK_DATABASE_URL") {
		t.Fatalf("Load() expected a database URL error, got %v", err)
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
