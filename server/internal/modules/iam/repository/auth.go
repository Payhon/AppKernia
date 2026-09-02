package repository

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (repository *Postgres) CreateSession(ctx context.Context, input domain.CreateSession) (domain.Session, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.Session{}, fmt.Errorf("begin session transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	var deviceID *uuid.UUID
	if input.DeviceKey != "" {
		id, upsertErr := queries.UpsertWebDevice(ctx, db.UpsertWebDeviceParams{
			UserID: input.UserID, DeviceKey: input.DeviceKey, LastIp: input.IPAddress,
		})
		if upsertErr != nil {
			return domain.Session{}, fmt.Errorf("upsert web device: %w", upsertErr)
		}
		deviceID = &id
	}
	userAgent := input.UserAgent
	session, err := queries.CreateSession(ctx, db.CreateSessionParams{
		UserID: input.UserID, TenantID: &input.TenantID, AppID: input.AppID, DeviceID: deviceID, Audience: input.Audience,
		AbsoluteExpiresAt: timestamp(input.AbsoluteExpiresAt), IdleExpiresAt: timestamp(input.IdleExpiresAt),
		IpAddress: input.IPAddress, UserAgent: &userAgent,
	})
	if err != nil {
		return domain.Session{}, fmt.Errorf("create session: %w", err)
	}
	if _, err = queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		SessionID: session.ID, TokenHash: input.RefreshTokenHash,
		ExpiresAt: timestamp(input.RefreshExpiresAt), CreatedIp: input.IPAddress,
	}); err != nil {
		return domain.Session{}, fmt.Errorf("create refresh token: %w", err)
	}
	requestID := optionalString(input.RequestID)
	userAgentValue := optionalString(input.UserAgent)
	authMethod := strings.TrimSpace(input.AuthMethod)
	if authMethod == "" {
		authMethod = "password"
	}
	userID, tenantID, sessionID := input.UserID, input.TenantID, session.ID
	if err = queries.InsertSuccessfulLoginEvent(ctx, db.InsertSuccessfulLoginEventParams{
		TenantID: &tenantID, UserID: &userID, SessionID: &sessionID, AppID: input.AppID, RequestID: requestID,
		AuthMethod: authMethod, Audience: input.Audience, ClientIp: input.IPAddress, UserAgent: userAgentValue,
		DeviceRegistered: input.DeviceKey != "",
	}); err != nil {
		return domain.Session{}, fmt.Errorf("record successful login: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Session{}, fmt.Errorf("commit session transaction: %w", err)
	}
	return mapSession(session), nil
}

func (repository *Postgres) LoginCaptchaRequired(ctx context.Context, scopeHash []byte, now time.Time) (bool, error) {
	count, err := db.New(repository.pool).GetActiveLoginFailureCount(ctx, db.GetActiveLoginFailureCountParams{
		ScopeHash: scopeHash, NowAt: timestamp(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get login failure state: %w", err)
	}
	return count >= 3, nil
}

func (repository *Postgres) RecordLoginFailure(ctx context.Context, input domain.LoginFailure) (int32, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin failed login transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	count, err := queries.UpsertLoginFailureState(ctx, db.UpsertLoginFailureStateParams{
		ScopeHash: input.ScopeHash, NowAt: timestamp(input.FailedAt), ExpiresAt: timestamp(input.ExpiresAt),
	})
	if err != nil {
		return 0, fmt.Errorf("update login failure state: %w", err)
	}
	if err := queries.InsertFailedLoginEvent(ctx, db.InsertFailedLoginEventParams{
		UserID: input.UserID, AppID: input.AppID, RequestID: optionalString(input.RequestID), Audience: input.Audience,
		ClientIp: input.IPAddress, UserAgent: optionalString(input.UserAgent),
	}); err != nil {
		return 0, fmt.Errorf("record failed login: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit failed login transaction: %w", err)
	}
	return count, nil
}

func (repository *Postgres) ResetLoginFailures(ctx context.Context, scopeHash []byte) error {
	if err := db.New(repository.pool).DeleteLoginFailureState(ctx, scopeHash); err != nil {
		return fmt.Errorf("reset login failure state: %w", err)
	}
	return nil
}

func (repository *Postgres) CreateLoginCaptcha(ctx context.Context, input domain.LoginCaptchaChallenge) (uuid.UUID, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin login captcha transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	if err = queries.InvalidateActiveLoginCaptchas(ctx, db.InvalidateActiveLoginCaptchasParams{
		ScopeHash: input.ScopeHash, NowAt: timestamp(input.CreatedAt),
	}); err != nil {
		return uuid.Nil, fmt.Errorf("invalidate active login captcha: %w", err)
	}
	challenge, err := queries.InsertLoginCaptchaChallenge(ctx, db.InsertLoginCaptchaChallengeParams{
		ScopeHash: input.ScopeHash, AnswerSalt: input.AnswerSalt, AnswerHash: input.AnswerHash,
		ExpiresAt: timestamp(input.ExpiresAt), CreatedAt: timestamp(input.CreatedAt),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert login captcha: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit login captcha transaction: %w", err)
	}
	return challenge.ID, nil
}

func (repository *Postgres) VerifyLoginCaptcha(ctx context.Context, input domain.LoginCaptchaAttempt) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin captcha verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	challenge, err := queries.GetLoginCaptchaForUpdate(ctx, db.GetLoginCaptchaForUpdateParams{
		ID: input.ID, ScopeHash: input.ScopeHash, NowAt: timestamp(input.Now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrLoginCaptchaInvalid
	}
	if err != nil {
		return fmt.Errorf("lock login captcha: %w", err)
	}
	hasher := sha256.New()
	hasher.Write(challenge.AnswerSalt)
	hasher.Write([]byte(strings.TrimSpace(input.Answer)))
	actualHash := hasher.Sum(nil)
	valid := len(actualHash) == len(challenge.AnswerHash) && subtle.ConstantTimeCompare(actualHash, challenge.AnswerHash) == 1
	if err = queries.CompleteLoginCaptchaAttempt(ctx, db.CompleteLoginCaptchaAttemptParams{
		ID: input.ID, Consume: valid, NowAt: timestamp(input.Now),
	}); err != nil {
		return fmt.Errorf("complete login captcha attempt: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit captcha verification: %w", err)
	}
	if !valid {
		return domain.ErrLoginCaptchaInvalid
	}
	return nil
}

func (repository *Postgres) RotateRefreshToken(ctx context.Context, oldHash, newHash []byte, ipAddress *netip.Addr) (domain.Session, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.Session{}, fmt.Errorf("begin refresh transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	row, err := queries.GetRefreshTokenForUpdate(ctx, oldHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Session{}, domain.ErrRefreshInvalid
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("lock refresh token: %w", err)
	}
	if row.ConsumedAt.Valid {
		if err = repository.revokeReusedRefresh(ctx, queries, row, ipAddress); err != nil {
			return domain.Session{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return domain.Session{}, fmt.Errorf("commit refresh reuse revocation: %w", err)
		}
		return domain.Session{}, domain.ErrRefreshReused
	}
	now := time.Now().UTC()
	if row.RefreshRevokedAt.Valid || row.SessionRevokedAt.Valid || row.SessionStatus != "active" ||
		!row.RefreshExpiresAt.Valid || !row.RefreshExpiresAt.Time.After(now) || row.TenantID == nil {
		return domain.Session{}, domain.ErrRefreshInvalid
	}
	newToken, err := queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		SessionID: row.SessionID, TokenHash: newHash, ParentTokenID: &row.RefreshTokenID,
		ExpiresAt: row.RefreshExpiresAt, CreatedIp: ipAddress,
	})
	if err != nil {
		return domain.Session{}, fmt.Errorf("create rotated refresh token: %w", err)
	}
	if err = queries.MarkRefreshTokenConsumed(ctx, db.MarkRefreshTokenConsumedParams{
		ID: row.RefreshTokenID, ReplacedByTokenID: &newToken.ID,
	}); err != nil {
		return domain.Session{}, fmt.Errorf("consume refresh token: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Session{}, fmt.Errorf("commit refresh rotation: %w", err)
	}
	return domain.Session{
		ID: row.SessionID, UserID: row.UserID, TenantID: *row.TenantID,
		AppID:    row.AppID,
		Audience: row.Audience, AccessTokenVersion: row.AccessTokenVersion,
		AbsoluteExpiresAt: row.AbsoluteExpiresAt.Time,
	}, nil
}

func (repository *Postgres) ResolveActiveMobileAppMembership(ctx context.Context, appID, userID uuid.UUID) (domain.Tenant, error) {
	var tenant domain.Tenant
	err := repository.pool.QueryRow(ctx, `
SELECT t.id, t.code, t.name, t.status
FROM app.user_memberships m
JOIN app.applications a ON a.id=m.app_id AND a.tenant_id=m.tenant_id AND a.status='active'
JOIN iam.tenants t ON t.id=m.tenant_id AND t.status='active' AND t.deleted_at IS NULL
JOIN iam.users u ON u.id=m.user_id AND u.status='active' AND u.deleted_at IS NULL
WHERE m.app_id=$1 AND m.user_id=$2 AND m.status='active'`, appID, userID).Scan(&tenant.ID, &tenant.Code, &tenant.Name, &tenant.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Tenant{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.Tenant{}, fmt.Errorf("resolve mobile app membership: %w", err)
	}
	return tenant, nil
}

func (repository *Postgres) revokeReusedRefresh(
	ctx context.Context,
	queries *db.Queries,
	row db.GetRefreshTokenForUpdateRow,
	ipAddress *netip.Addr,
) error {
	reason := "refresh_token_reuse"
	if err := queries.MarkRefreshTokenReused(ctx, row.RefreshTokenID); err != nil {
		return fmt.Errorf("mark refresh token reuse: %w", err)
	}
	if err := queries.RevokeSession(ctx, db.RevokeSessionParams{ID: row.SessionID, RevokeReason: &reason}); err != nil {
		return fmt.Errorf("revoke reused session: %w", err)
	}
	if err := queries.RevokeSessionRefreshTokens(ctx, row.SessionID); err != nil {
		return fmt.Errorf("revoke refresh token family: %w", err)
	}
	userID, sessionID := row.UserID, row.SessionID
	if err := queries.InsertRefreshReuseSecurityEvent(ctx, db.InsertRefreshReuseSecurityEventParams{
		TenantID: row.TenantID, UserID: &userID, SessionID: &sessionID, AppID: row.AppID, ClientIp: ipAddress,
	}); err != nil {
		return fmt.Errorf("record refresh reuse security event: %w", err)
	}
	return nil
}

func (repository *Postgres) RevokeSession(ctx context.Context, sessionID uuid.UUID, reason string) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin revoke transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	if _, err = tx.Exec(ctx, `UPDATE notify.push_devices d
SET status='disabled',invalidated_at=COALESCE(invalidated_at,now()),invalid_reason=COALESCE(invalid_reason,'session_revoked'),updated_at=now()
FROM iam.sessions s
WHERE s.id=$1 AND s.audience='ak-mobile' AND s.app_id IS NOT NULL AND s.device_id IS NOT NULL
  AND d.tenant_id=s.tenant_id AND d.app_id=s.app_id AND d.user_id=s.user_id AND d.device_id=s.device_id AND d.status='active'`, sessionID); err != nil {
		return fmt.Errorf("disable push binding for revoked mobile session: %w", err)
	}
	if err = queries.RevokeSession(ctx, db.RevokeSessionParams{ID: sessionID, RevokeReason: &reason}); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if err = queries.RevokeSessionRefreshTokens(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke session refresh tokens: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit session revocation: %w", err)
	}
	return nil
}

func (repository *Postgres) ValidateSession(ctx context.Context, session domain.Session) error {
	_, err := db.New(repository.pool).GetActiveSession(ctx, db.GetActiveSessionParams{
		ID: session.ID, UserID: session.UserID, TenantID: &session.TenantID,
		Audience: session.Audience, AccessTokenVersion: session.AccessTokenVersion, AppID: session.AppID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrRefreshInvalid
	}
	if err != nil {
		return fmt.Errorf("validate active session: %w", err)
	}
	return nil
}

func (repository *Postgres) GetAuthContext(ctx context.Context, userID, tenantID uuid.UUID) (domain.AuthContext, error) {
	queries := db.New(repository.pool)
	row, err := queries.GetAuthContextUser(ctx, db.GetAuthContextUserParams{ID: userID, TenantID: tenantID})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AuthContext{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.AuthContext{}, fmt.Errorf("get auth context user: %w", err)
	}
	roles, err := queries.ListEffectiveRoleCodes(ctx, db.ListEffectiveRoleCodesParams{TenantID: tenantID, UserID: userID})
	if err != nil {
		return domain.AuthContext{}, fmt.Errorf("list auth context roles: %w", err)
	}
	permissions, err := queries.ListEffectivePermissionCodes(ctx, db.ListEffectivePermissionCodesParams{TenantID: tenantID, UserID: userID})
	if err != nil {
		return domain.AuthContext{}, fmt.Errorf("list auth context permissions: %w", err)
	}
	menuRows, err := queries.ListEffectiveMenus(ctx, db.ListEffectiveMenusParams{TenantID: tenantID, UserID: userID})
	if err != nil {
		return domain.AuthContext{}, fmt.Errorf("list auth context menus: %w", err)
	}
	menus := make([]domain.Menu, 0, len(menuRows))
	for _, menuRow := range menuRows {
		var metadata struct {
			I18nKey     string `json:"i18n_key"`
			FeatureFlag string `json:"feature_flag"`
		}
		if err = json.Unmarshal(menuRow.Metadata, &metadata); err != nil || metadata.I18nKey == "" {
			return domain.AuthContext{}, fmt.Errorf("decode menu %s metadata", menuRow.Code)
		}
		menus = append(menus, domain.Menu{
			ID: menuRow.ID, ParentID: menuRow.ParentID, Code: menuRow.Code, I18nKey: metadata.I18nKey,
			Title: menuRow.Title, Type: menuRow.MenuType, RoutePath: menuRow.RoutePath,
			ComponentKey: menuRow.ComponentKey, Icon: menuRow.Icon, Affix: menuRow.Affix,
			SortOrder: menuRow.SortOrder, FeatureFlag: metadata.FeatureFlag,
		})
	}
	return domain.AuthContext{
		User: domain.User{
			ID: row.ID, Email: valueOrEmpty(row.Email), DisplayName: row.DisplayName,
			Locale: row.Locale, TimeZone: row.TimeZone, Status: "active", AvatarFileID: row.AvatarFileID,
		},
		Tenant:   domain.Tenant{ID: row.TenantID, Code: row.TenantCode, Name: row.TenantName, Status: "active"},
		TimeZone: row.TimeZone, Roles: roles, Permissions: permissions, Menus: menus,
	}, nil
}

func (repository *Postgres) ListSelfSessions(ctx context.Context, scope domain.SelfSessionScope) ([]domain.SelfSession, error) {
	rows, err := db.New(repository.pool).ListSelfSessions(ctx, db.ListSelfSessionsParams{
		UserID: scope.UserID, TenantID: &scope.TenantID, Audience: scope.Audience,
	})
	if err != nil {
		return nil, fmt.Errorf("list self sessions: %w", err)
	}
	result := make([]domain.SelfSession, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.SelfSession{
			ID: row.ID, Audience: row.Audience, Status: row.Status, IPAddress: row.IpAddress,
			UserAgent: valueOrEmpty(row.UserAgent), LastSeenAt: row.LastSeenAt.Time,
			AbsoluteExpiresAt: row.AbsoluteExpiresAt.Time, CreatedAt: row.CreatedAt.Time,
		})
	}
	return result, nil
}

func (repository *Postgres) RevokeSelfSession(ctx context.Context, input domain.RevokeSelfSession) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin self session revocation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	_, err = queries.LockSelfSessionForRevoke(ctx, db.LockSelfSessionForRevokeParams{
		SessionID: input.SessionID, UserID: input.UserID,
		TenantID: &input.TenantID, Audience: input.Audience,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrSessionNotFound
	}
	if err != nil {
		return fmt.Errorf("lock self session: %w", err)
	}
	reason := input.RevocationReason
	if err = queries.RevokeSession(ctx, db.RevokeSessionParams{ID: input.SessionID, RevokeReason: &reason}); err != nil {
		return fmt.Errorf("revoke self session: %w", err)
	}
	if err = queries.RevokeSessionRefreshTokens(ctx, input.SessionID); err != nil {
		return fmt.Errorf("revoke self session refresh tokens: %w", err)
	}
	resourceID := input.SessionID.String()
	requestPath := "/admin-api/v1/me/sessions/" + resourceID
	userAgent := strings.TrimSpace(input.UserAgent)
	var userAgentValue *string
	if userAgent != "" {
		userAgentValue = &userAgent
	}
	if err = queries.InsertSelfSessionRevokeAudit(ctx, db.InsertSelfSessionRevokeAuditParams{
		TenantID: &input.TenantID, UserID: &input.UserID, ActorSessionID: &input.ActorSessionID,
		RequestID: input.RequestID, ResourceID: &resourceID, RequestPath: &requestPath,
		ClientIp: input.IPAddress, UserAgent: userAgentValue,
	}); err != nil {
		return fmt.Errorf("audit self session revocation: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit self session revocation: %w", err)
	}
	return nil
}

func (repository *Postgres) ListSelfDevices(ctx context.Context, scope domain.SelfDeviceScope) ([]domain.SelfDevice, error) {
	rows, err := db.New(repository.pool).ListSelfDevices(ctx, db.ListSelfDevicesParams{
		UserID: scope.UserID, TenantID: &scope.TenantID,
		CurrentSessionID: scope.SessionID, Audience: scope.Audience,
	})
	if err != nil {
		return nil, fmt.Errorf("list self devices: %w", err)
	}
	result := make([]domain.SelfDevice, 0, len(rows))
	for _, row := range rows {
		var lastSeenAt *time.Time
		if row.LastSeenAt.Valid {
			value := row.LastSeenAt.Time.UTC()
			lastSeenAt = &value
		}
		result = append(result, domain.SelfDevice{
			ID: row.ID, Platform: row.Platform, DeviceName: valueOrEmpty(row.DeviceName),
			Model: valueOrEmpty(row.Model), OSVersion: valueOrEmpty(row.OsVersion),
			AppVersion: valueOrEmpty(row.AppVersion), LastIP: row.LastIp, LastSeenAt: lastSeenAt,
			CreatedAt: row.CreatedAt.Time.UTC(), LatestUserAgent: row.LatestUserAgent,
			ActiveSessionCount: row.ActiveSessionCount, Current: row.Current,
		})
	}
	return result, nil
}

func (repository *Postgres) RemoveSelfDevice(ctx context.Context, input domain.RemoveSelfDevice) (bool, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return false, fmt.Errorf("begin self device removal: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	locked, err := queries.LockSelfDeviceForRemove(ctx, db.LockSelfDeviceForRemoveParams{
		CurrentSessionID: input.SessionID, DeviceID: input.DeviceID, UserID: input.UserID,
		TenantID: &input.TenantID, Audience: input.Audience,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, domain.ErrDeviceNotFound
	}
	if err != nil {
		return false, fmt.Errorf("lock self device: %w", err)
	}
	deviceID := input.DeviceID
	if err = queries.RevokeSelfDeviceRefreshTokens(ctx, db.RevokeSelfDeviceRefreshTokensParams{
		UserID: input.UserID, DeviceID: &deviceID,
	}); err != nil {
		return false, fmt.Errorf("revoke self device refresh tokens: %w", err)
	}
	revokedCount, err := queries.RevokeSelfDeviceSessions(ctx, db.RevokeSelfDeviceSessionsParams{
		UserID: input.UserID, DeviceID: &deviceID,
	})
	if err != nil {
		return false, fmt.Errorf("revoke self device sessions: %w", err)
	}
	resourceID := input.DeviceID.String()
	requestPath := "/admin-api/v1/me/devices/" + resourceID
	userAgent := strings.TrimSpace(input.UserAgent)
	var userAgentValue *string
	if userAgent != "" {
		userAgentValue = &userAgent
	}
	if err = queries.InsertSelfDeviceRemoveAudit(ctx, db.InsertSelfDeviceRemoveAuditParams{
		TenantID: &input.TenantID, UserID: &input.UserID, ActorSessionID: &input.SessionID,
		RequestID: input.RequestID, ResourceID: &resourceID, RequestPath: &requestPath,
		ClientIp: input.IPAddress, UserAgent: userAgentValue, RevokedSessionCount: revokedCount,
	}); err != nil {
		return false, fmt.Errorf("audit self device removal: %w", err)
	}
	if err = queries.DeleteSelfDevice(ctx, db.DeleteSelfDeviceParams{
		DeviceID: input.DeviceID, UserID: input.UserID,
	}); err != nil {
		return false, fmt.Errorf("delete self device: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit self device removal: %w", err)
	}
	return locked.Current, nil
}

func mapSession(session db.IamSession) domain.Session {
	tenantID := uuid.Nil
	if session.TenantID != nil {
		tenantID = *session.TenantID
	}
	return domain.Session{
		ID: session.ID, UserID: session.UserID, TenantID: tenantID, Audience: session.Audience,
		AppID:              session.AppID,
		AccessTokenVersion: session.AccessTokenVersion, AbsoluteExpiresAt: session.AbsoluteExpiresAt.Time,
	}
}

func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func optionalString(value string) *string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return nil
	}
	return &normalized
}
