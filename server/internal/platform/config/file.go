package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

const defaultYAML = `environment: development
state_dir: ""

server:
  listen: 127.0.0.1:8080
  public_web_base_url: http://127.0.0.1:8080
  shutdown_timeout: 15s

database:
  driver: sqlite
  url: ""
  sqlite_path: ""

admin:
  origin: http://127.0.0.1:8080
  path: /admin
  static_dir: ""
  registration_enabled: false
  registration_tenant_code: local

auth:
  jwt_key_id: development-ephemeral
  jwt_private_key_base64: ""
  login_protection_key_base64: ""
  config_master_key_base64: ""
  config_master_key_version: 1

features:
  password_recovery: false
  avatar_upload: false
  file_storage: false
  multi_tenant: false
  api_clients: false
  webhooks: false
  push: false
  mfa: false
  oauth: false

adapters:
  password_recovery: ""
  object_storage: ""
  webhook: ""
  push: ""
  oauth: ""
  otp: ""

storage:
  local_dir: ./var/object-storage

feedback:
  clamd_socket: ""

platform:
  tenant_code: local

log:
  level: info
  format: text
  file: ""
`

// WriteDefault writes a non-secret starter configuration with owner-only
// permissions. Existing files are preserved unless force is true.
func WriteDefault(path string, force bool) (string, error) {
	var err error
	if strings.TrimSpace(path) == "" {
		path, err = DefaultPath()
		if err != nil {
			return "", err
		}
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve configuration file: %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create configuration directory: %w", err)
	}
	if !force {
		file, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if openErr != nil {
			if errors.Is(openErr, os.ErrExist) {
				return "", fmt.Errorf("configuration file already exists: %s", path)
			}
			return "", fmt.Errorf("create configuration file: %w", openErr)
		}
		if _, err = file.WriteString(defaultYAML); err == nil {
			err = file.Sync()
		}
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
			return "", fmt.Errorf("write configuration file: %w", err)
		}
		return path, nil
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".akone-config-*")
	if err != nil {
		return "", fmt.Errorf("create temporary configuration file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err = temporary.Chmod(0o600); err == nil {
		_, err = temporary.WriteString(defaultYAML)
	}
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("write temporary configuration file: %w", err)
	}
	if err = os.Rename(temporaryPath, path); err != nil {
		return "", fmt.Errorf("replace configuration file: %w", err)
	}
	return path, nil
}

// RedactedYAML renders the effective configuration without credentials or key
// material, making it safe for diagnostics and support requests.
func RedactedYAML(cfg Config) ([]byte, error) {
	values := fileConfig{}
	values.Environment, values.StateDir = cfg.Environment, cfg.StateDir
	values.Server.Listen = cfg.HTTPAddr
	values.Server.PublicWebBaseURL = cfg.PublicWebBaseURL
	values.Server.ShutdownTimeout = cfg.ShutdownTimeout.String()
	values.Database.Driver = cfg.DatabaseDriver
	if cfg.DatabaseURL != "" {
		values.Database.URL = "<redacted>"
	}
	values.Database.SQLitePath = cfg.SQLitePath
	values.Admin.Origin, values.Admin.Path, values.Admin.StaticDir = cfg.AdminOrigin, cfg.AdminPath, cfg.AdminStaticDir
	values.Admin.RegistrationEnabled, values.Admin.RegistrationTenantCode = cfg.AdminRegistrationEnabled, cfg.AdminRegistrationTenantCode
	values.Auth.JWTKeyID = cfg.JWTKeyID
	values.Auth.ConfigMasterKeyVersion = cfg.ConfigMasterKeyVersion
	if cfg.JWTPrivateKey != "" {
		values.Auth.JWTPrivateKeyBase64 = "<redacted>"
	}
	if cfg.LoginProtectionKeyBase64 != "" {
		values.Auth.LoginProtectionKeyBase64 = "<redacted>"
	}
	if cfg.ConfigMasterKeyBase64 != "" {
		values.Auth.ConfigMasterKeyBase64 = "<redacted>"
	}
	values.Features.PasswordRecovery, values.Features.AvatarUpload = cfg.PasswordRecoveryEnabled, cfg.AvatarUploadEnabled
	values.Features.FileStorage, values.Features.MultiTenant = cfg.FileStorageEnabled, cfg.MultiTenantEnabled
	values.Features.APIClients, values.Features.Webhooks = cfg.APIClientsEnabled, cfg.WebhooksEnabled
	values.Features.Push, values.Features.MFA, values.Features.OAuth = cfg.PushEnabled, cfg.MFAEnabled, cfg.OAuthEnabled
	values.Adapters.PasswordRecovery, values.Adapters.ObjectStorage = cfg.PasswordRecoveryAdapter, cfg.ObjectStorageAdapter
	values.Adapters.Webhook, values.Adapters.Push = cfg.WebhookAdapter, cfg.PushAdapter
	values.Adapters.OAuth, values.Adapters.OTP = cfg.OAuthAdapter, cfg.OTPAdapter
	values.Storage.LocalDir, values.Feedback.ClamdSocket = cfg.LocalObjectStorageDir, cfg.FeedbackClamdSocket
	values.Platform.TenantCode = cfg.PlatformTenantCode
	values.Log.Level, values.Log.Format, values.Log.File = cfg.LogLevel, cfg.LogFormat, cfg.LogFile
	return yaml.Marshal(values)
}
