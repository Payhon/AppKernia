package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/infrastructure/db"
	sessiondomain "github.com/appkernia/appkernia/server/internal/modules/sessionadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (r *Postgres) List(ctx context.Context, tenantID, current uuid.UUID, filter sessiondomain.Filter) (sessiondomain.Page, error) {
	queries := db.New(r.pool)
	params := db.SessionAdminListParams{TenantID: &tenantID, CurrentSessionID: current, FromAt: pgTime(filter.FromAt), ToAt: pgTime(filter.ToAt), Query: filter.Query, Audience: filter.Audience, Platform: filter.Platform, Ip: filter.IP, Status: filter.Status, PageSize: filter.PageSize, PageOffset: (filter.Page - 1) * filter.PageSize}
	rows, err := queries.SessionAdminList(ctx, params)
	if err != nil {
		return sessiondomain.Page{}, err
	}
	items := make([]sessiondomain.Session, 0, len(rows))
	for _, row := range rows {
		items = append(items, sessiondomain.Session{ID: row.ID, UserID: row.UserID, UserHint: maskEmail(row.Email), DisplayName: row.DisplayName, Audience: row.Audience, Platform: row.Platform, DeviceHint: safeHint(row.DeviceHint), IPHint: maskIP(row.IpAddress), Status: row.EffectiveStatus, Current: row.Current, LastSeenAt: row.LastSeenAt.Time, ExpiresAt: row.AbsoluteExpiresAt.Time, RevokedAt: timePtr(row.RevokedAt)})
	}
	total, err := queries.SessionAdminCount(ctx, db.SessionAdminCountParams{TenantID: &tenantID, FromAt: pgTime(filter.FromAt), ToAt: pgTime(filter.ToAt), Query: filter.Query, Audience: filter.Audience, Platform: filter.Platform, Ip: filter.IP, Status: filter.Status})
	if err != nil {
		return sessiondomain.Page{}, err
	}
	return sessiondomain.Page{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (r *Postgres) Revoke(ctx context.Context, principal sessiondomain.Principal, id uuid.UUID) (sessiondomain.RevokeResult, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return sessiondomain.RevokeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	locked, err := queries.SessionAdminLockForRevoke(ctx, db.SessionAdminLockForRevokeParams{ID: id, TenantID: &principal.TenantID, CurrentSessionID: principal.SessionID})
	if errors.Is(err, pgx.ErrNoRows) {
		return sessiondomain.RevokeResult{}, sessiondomain.ErrSessionAbsent
	}
	if err != nil {
		return sessiondomain.RevokeResult{}, err
	}
	if count, execErr := queries.SessionAdminRevoke(ctx, db.SessionAdminRevokeParams{ID: id, TenantID: &principal.TenantID}); execErr != nil || count != 1 {
		if execErr != nil {
			return sessiondomain.RevokeResult{}, execErr
		}
		return sessiondomain.RevokeResult{}, sessiondomain.ErrSessionAbsent
	}
	if err = queries.SessionAdminRevokeRefreshTokens(ctx, id); err != nil {
		return sessiondomain.RevokeResult{}, err
	}
	resourceID := id.String()
	if err = queries.SessionAdminInsertAudit(ctx, db.SessionAdminInsertAuditParams{TenantID: &principal.TenantID, ActorID: &principal.UserID, ActorSessionID: &principal.SessionID, RequestID: principal.RequestID, ResourceID: &resourceID, ClientIp: principal.IPAddress, UserAgent: stringPtr(principal.UserAgent), Current: locked.Current}); err != nil {
		return sessiondomain.RevokeResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return sessiondomain.RevokeResult{}, err
	}
	return sessiondomain.RevokeResult{ID: id, Revoked: true, Current: locked.Current}, nil
}

func maskEmail(value string) string {
	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "***"
	}
	return string([]rune(parts[0])[0]) + "***@" + parts[1]
}

func maskIP(value string) string {
	if value == "" {
		return ""
	}
	if strings.Contains(value, ".") {
		parts := strings.Split(value, ".")
		if len(parts) == 4 {
			return strings.Join(parts[:3], ".") + ".***"
		}
	}
	if index := strings.LastIndex(value, ":"); index >= 0 {
		return value[:index+1] + "***"
	}
	return "***"
}

func safeHint(value string) string {
	value = strings.TrimSpace(value)
	return value
}
func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
func pgTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}
