package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type LoginSettings struct {
	AppID           uuid.UUID `json:"app_id"`
	PasswordEnabled bool      `json:"password_enabled"`
	OTPEnabled      bool      `json:"otp_enabled"`
	EmailOTPEnabled bool      `json:"email_otp_enabled"`
	SMSOTPEnabled   bool      `json:"sms_otp_enabled"`
	OAuthEnabled    bool      `json:"oauth_enabled"`
	LockVersion     int32     `json:"lock_version"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type LoginSettingsInput struct {
	OTPEnabled      bool
	EmailOTPEnabled bool
	SMSOTPEnabled   bool
	LockVersion     int32
}

func validateLoginSettings(input LoginSettingsInput) error {
	if input.LockVersion < 0 || input.OTPEnabled && !input.EmailOTPEnabled && !input.SMSOTPEnabled {
		return ErrInvalidInput
	}
	return nil
}

func (s *Service) GetAdminLoginSettings(ctx context.Context, token string, appID uuid.UUID) (LoginSettings, error) {
	principal, err := s.authorizeAdmin(ctx, token, "app.login_settings.read")
	if err != nil {
		return LoginSettings{}, err
	}
	return s.loginSettings(ctx, principal.Tenant.ID, appID)
}

func (s *Service) loginSettings(ctx context.Context, tenantID, appID uuid.UUID) (LoginSettings, error) {
	if appID == uuid.Nil {
		return LoginSettings{}, ErrInvalidInput
	}
	out := LoginSettings{PasswordEnabled: true, EmailOTPEnabled: true}
	err := s.pool.QueryRow(ctx, `SELECT a.id,
COALESCE(c.otp_login_enabled,false),COALESCE(c.email_otp_enabled,true),COALESCE(c.sms_otp_enabled,false),
EXISTS(SELECT 1 FROM app.application_login_provider_bindings b
 JOIN sys.login_provider_configs p ON p.tenant_id=b.tenant_id AND p.id=b.login_provider_config_id
 WHERE b.app_id=a.id AND b.enabled AND p.status='active' AND p.last_preflight_status='ready' AND p.deleted_at IS NULL),
COALESCE(c.lock_version,0),COALESCE(c.updated_at,a.updated_at)
FROM app.applications a
LEFT JOIN app.application_login_settings c ON c.tenant_id=a.tenant_id AND c.app_id=a.id
WHERE a.tenant_id=$1 AND a.id=$2 AND a.deleted_at IS NULL`, tenantID, appID).Scan(
		&out.AppID, &out.OTPEnabled, &out.EmailOTPEnabled, &out.SMSOTPEnabled,
		&out.OAuthEnabled, &out.LockVersion, &out.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginSettings{}, ErrAppNotFound
	}
	return out, err
}

func (s *Service) PublicLoginSettings(ctx context.Context, appID uuid.UUID) (LoginSettings, error) {
	var tenantID uuid.UUID
	if err := s.pool.QueryRow(ctx, `SELECT tenant_id FROM app.applications WHERE id=$1 AND deleted_at IS NULL`, appID).Scan(&tenantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginSettings{}, ErrAppNotFound
		}
		return LoginSettings{}, err
	}
	return s.loginSettings(ctx, tenantID, appID)
}

func (s *Service) UpdateAdminLoginSettings(ctx context.Context, token string, appID uuid.UUID, input LoginSettingsInput, requestID string) (LoginSettings, error) {
	principal, err := s.authorizeAdmin(ctx, token, "app.login_settings.update")
	if err != nil {
		return LoginSettings{}, err
	}
	if appID == uuid.Nil || validateLoginSettings(input) != nil {
		return LoginSettings{}, ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return LoginSettings{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL)`, principal.Tenant.ID, appID).Scan(&exists); err != nil {
		return LoginSettings{}, err
	}
	if !exists {
		return LoginSettings{}, ErrAppNotFound
	}
	var before LoginSettings
	err = tx.QueryRow(ctx, `SELECT app_id,otp_login_enabled,email_otp_enabled,sms_otp_enabled,lock_version,updated_at
FROM app.application_login_settings WHERE tenant_id=$1 AND app_id=$2 FOR UPDATE`, principal.Tenant.ID, appID).Scan(
		&before.AppID, &before.OTPEnabled, &before.EmailOTPEnabled, &before.SMSOTPEnabled, &before.LockVersion, &before.UpdatedAt)
	missing := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !missing {
		return LoginSettings{}, err
	}
	if missing {
		if input.LockVersion != 0 {
			return LoginSettings{}, ErrConflict
		}
		_, err = tx.Exec(ctx, `INSERT INTO app.application_login_settings(
tenant_id,app_id,otp_login_enabled,email_otp_enabled,sms_otp_enabled,created_by,updated_by)
VALUES($1,$2,$3,$4,$5,$6,$6)`, principal.Tenant.ID, appID, input.OTPEnabled, input.EmailOTPEnabled, input.SMSOTPEnabled, principal.User.ID)
	} else {
		if before.LockVersion != input.LockVersion {
			return LoginSettings{}, ErrConflict
		}
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE app.application_login_settings
SET otp_login_enabled=$3,email_otp_enabled=$4,sms_otp_enabled=$5,lock_version=lock_version+1,updated_by=$6
WHERE tenant_id=$1 AND app_id=$2 AND lock_version=$7`, principal.Tenant.ID, appID, input.OTPEnabled, input.EmailOTPEnabled, input.SMSOTPEnabled, principal.User.ID, input.LockVersion)
		if err == nil && tag.RowsAffected() != 1 {
			return LoginSettings{}, ErrConflict
		}
	}
	if err != nil {
		return LoginSettings{}, err
	}
	var after LoginSettings
	if err = tx.QueryRow(ctx, `SELECT app_id,true,otp_login_enabled,email_otp_enabled,sms_otp_enabled,false,lock_version,updated_at
FROM app.application_login_settings WHERE tenant_id=$1 AND app_id=$2`, principal.Tenant.ID, appID).Scan(
		&after.AppID, &after.PasswordEnabled, &after.OTPEnabled, &after.EmailOTPEnabled, &after.SMSOTPEnabled, &after.OAuthEnabled, &after.LockVersion, &after.UpdatedAt); err != nil {
		return LoginSettings{}, err
	}
	var beforeJSON []byte
	if !missing {
		before.PasswordEnabled = true
		beforeJSON, _ = json.Marshal(before)
	}
	afterJSON, _ := json.Marshal(after)
	if _, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(
tenant_id,user_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,before_data,after_data,succeeded)
VALUES($1,$2,NULLIF($3,''),'app','app.login_settings.update','app.login_settings.update','application_login_settings',$4::text,
'PUT','/admin-api/v1/apps/'||$4::text||'/login-settings',$5,$6,true)`, principal.Tenant.ID, principal.User.ID, strings.TrimSpace(requestID), appID.String(), beforeJSON, afterJSON); err != nil {
		return LoginSettings{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return LoginSettings{}, err
	}
	return after, nil
}
