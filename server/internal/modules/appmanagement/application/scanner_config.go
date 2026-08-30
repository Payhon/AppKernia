package application

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/net/publicsuffix"
)

const maxScannerHostPatterns = 100

var scannerHostLabelPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

type ScannerConfigValidationError struct {
	Field string
	Index *int
}

func (e *ScannerConfigValidationError) Error() string { return "invalid scanner configuration" }
func (e *ScannerConfigValidationError) Unwrap() error { return ErrInvalidInput }

type ScannerConfig struct {
	AppID               uuid.UUID `json:"app_id"`
	WebViewEnabled      bool      `json:"webview_enabled"`
	AllowedHostPatterns []string  `json:"allowed_host_patterns"`
	LockVersion         int32     `json:"lock_version"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ScannerConfigInput struct {
	WebViewEnabled      bool
	AllowedHostPatterns []string
	LockVersion         int32
}

type PublicScannerWebViewConfig struct {
	Enabled             bool     `json:"enabled"`
	AllowedHostPatterns []string `json:"allowed_host_patterns"`
}

type PublicScannerConfig struct {
	WebView PublicScannerWebViewConfig `json:"webview"`
}

func normalizeScannerConfigInput(input ScannerConfigInput) (ScannerConfigInput, error) {
	if input.LockVersion < 0 || len(input.AllowedHostPatterns) > maxScannerHostPatterns {
		return input, ErrInvalidInput
	}
	seen := make(map[string]struct{}, len(input.AllowedHostPatterns))
	normalized := make([]string, 0, len(input.AllowedHostPatterns))
	for index, candidate := range input.AllowedHostPatterns {
		value, err := normalizeScannerHostPattern(candidate)
		if err != nil {
			return input, &ScannerConfigValidationError{Field: "allowed_host_patterns", Index: &index}
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	if input.WebViewEnabled && len(normalized) == 0 {
		return input, &ScannerConfigValidationError{Field: "allowed_host_patterns"}
	}
	input.AllowedHostPatterns = normalized
	return input, nil
}

func normalizeScannerHostPattern(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(raw), "."))
	wildcard := strings.HasPrefix(value, "*.")
	if wildcard {
		value = strings.TrimPrefix(value, "*.")
	}
	if value == "" || len(value) > 253 || !isASCII(value) || net.ParseIP(value) != nil || value == "localhost" || strings.HasSuffix(value, ".localhost") || strings.HasSuffix(value, ".local") {
		return "", ErrInvalidInput
	}
	labels := strings.Split(value, ".")
	if len(labels) < 2 {
		return "", ErrInvalidInput
	}
	for _, label := range labels {
		if !scannerHostLabelPattern.MatchString(label) {
			return "", ErrInvalidInput
		}
	}
	registrable, err := publicsuffix.EffectiveTLDPlusOne(value)
	if err != nil || registrable == "" {
		return "", ErrInvalidInput
	}
	if wildcard {
		return "*." + value, nil
	}
	return value, nil
}

func isASCII(value string) bool {
	for _, candidate := range []byte(value) {
		if candidate > 0x7f {
			return false
		}
	}
	return true
}

func (s *Service) GetAdminScannerConfig(ctx context.Context, token string, appID uuid.UUID) (ScannerConfig, error) {
	principal, err := s.authorizeAdmin(ctx, token, "app.scanner_config.read")
	if err != nil {
		return ScannerConfig{}, err
	}
	if appID == uuid.Nil {
		return ScannerConfig{}, ErrInvalidInput
	}
	var out ScannerConfig
	err = s.pool.QueryRow(ctx, `SELECT a.id,COALESCE(c.webview_enabled,false),COALESCE(c.allowed_host_patterns,'{}'::text[]),COALESCE(c.lock_version,0),COALESCE(c.updated_at,a.updated_at)
FROM app.applications a
LEFT JOIN app.application_scanner_configs c ON c.tenant_id=a.tenant_id AND c.app_id=a.id
WHERE a.tenant_id=$1 AND a.id=$2 AND a.deleted_at IS NULL`, principal.Tenant.ID, appID).Scan(
		&out.AppID, &out.WebViewEnabled, &out.AllowedHostPatterns, &out.LockVersion, &out.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ScannerConfig{}, ErrAppNotFound
	}
	return out, err
}

func (s *Service) UpdateAdminScannerConfig(ctx context.Context, token string, appID uuid.UUID, input ScannerConfigInput, requestID string) (ScannerConfig, error) {
	principal, err := s.authorizeAdmin(ctx, token, "app.scanner_config.update")
	if err != nil {
		return ScannerConfig{}, err
	}
	if appID == uuid.Nil {
		return ScannerConfig{}, ErrInvalidInput
	}
	input, err = normalizeScannerConfigInput(input)
	if err != nil {
		return ScannerConfig{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ScannerConfig{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var appExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL)`, principal.Tenant.ID, appID).Scan(&appExists); err != nil {
		return ScannerConfig{}, err
	}
	if !appExists {
		return ScannerConfig{}, ErrAppNotFound
	}
	var before ScannerConfig
	err = tx.QueryRow(ctx, `SELECT app_id,webview_enabled,allowed_host_patterns,lock_version,updated_at
FROM app.application_scanner_configs WHERE tenant_id=$1 AND app_id=$2 FOR UPDATE`, principal.Tenant.ID, appID).Scan(
		&before.AppID, &before.WebViewEnabled, &before.AllowedHostPatterns, &before.LockVersion, &before.UpdatedAt,
	)
	missing := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !missing {
		return ScannerConfig{}, err
	}
	if missing {
		if input.LockVersion != 0 {
			return ScannerConfig{}, ErrConflict
		}
		_, err = tx.Exec(ctx, `INSERT INTO app.application_scanner_configs(
tenant_id,app_id,webview_enabled,allowed_host_patterns,created_by,updated_by)
VALUES($1,$2,$3,$4,$5,$5)`, principal.Tenant.ID, appID, input.WebViewEnabled, input.AllowedHostPatterns, principal.User.ID)
	} else {
		if before.LockVersion != input.LockVersion {
			return ScannerConfig{}, ErrConflict
		}
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE app.application_scanner_configs
SET webview_enabled=$3,allowed_host_patterns=$4,lock_version=lock_version+1,updated_by=$5
WHERE tenant_id=$1 AND app_id=$2 AND lock_version=$6`, principal.Tenant.ID, appID, input.WebViewEnabled, input.AllowedHostPatterns, principal.User.ID, input.LockVersion)
		if err == nil && tag.RowsAffected() != 1 {
			return ScannerConfig{}, ErrConflict
		}
	}
	if err != nil {
		return ScannerConfig{}, err
	}
	var after ScannerConfig
	if err = tx.QueryRow(ctx, `SELECT app_id,webview_enabled,allowed_host_patterns,lock_version,updated_at
FROM app.application_scanner_configs WHERE tenant_id=$1 AND app_id=$2`, principal.Tenant.ID, appID).Scan(
		&after.AppID, &after.WebViewEnabled, &after.AllowedHostPatterns, &after.LockVersion, &after.UpdatedAt,
	); err != nil {
		return ScannerConfig{}, err
	}
	var beforeJSON []byte
	if !missing {
		beforeJSON, _ = json.Marshal(scannerConfigAudit(before))
	}
	afterJSON, _ := json.Marshal(scannerConfigAudit(after))
	if _, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(
tenant_id,user_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,
http_method,request_path,before_data,after_data,succeeded)
VALUES($1,$2,NULLIF($3,''),'app','app.scanner_config.update','app.scanner_config.update','application_scanner_config',$4::text,
'PUT','/admin-api/v1/apps/'||$4::text||'/scanner-config',$5,$6,true)`, principal.Tenant.ID, principal.User.ID, strings.TrimSpace(requestID), appID.String(), beforeJSON, afterJSON); err != nil {
		return ScannerConfig{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ScannerConfig{}, err
	}
	return after, nil
}

func (s *Service) PublicScannerConfig(ctx context.Context, appID uuid.UUID) (PublicScannerConfig, error) {
	out := PublicScannerConfig{WebView: PublicScannerWebViewConfig{AllowedHostPatterns: []string{}}}
	if appID == uuid.Nil {
		return out, ErrInvalidInput
	}
	err := s.pool.QueryRow(ctx, `SELECT webview_enabled,allowed_host_patterns
FROM app.application_scanner_configs WHERE app_id=$1`, appID).Scan(&out.WebView.Enabled, &out.WebView.AllowedHostPatterns)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	return out, err
}

func scannerConfigAudit(item ScannerConfig) map[string]any {
	return map[string]any{
		"app_id":                item.AppID,
		"webview_enabled":       item.WebViewEnabled,
		"allowed_host_patterns": item.AllowedHostPatterns,
		"lock_version":          item.LockVersion,
	}
}
