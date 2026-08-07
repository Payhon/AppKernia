package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func decodePreferences(row db.GetMobilePreferencesRow) (domain.Preferences, error) {
	result := domain.Preferences{Locale: row.Locale, Appearance: row.Appearance}
	if err := json.Unmarshal(row.NotificationPreferences, &result.NotificationPreferences); err != nil {
		return domain.Preferences{}, fmt.Errorf("decode notification preferences: %w", err)
	}
	return result, nil
}
func (repository *Postgres) GetPreferences(ctx context.Context, appID, userID uuid.UUID) (domain.Preferences, error) {
	row, err := db.New(repository.pool).GetMobilePreferences(ctx, db.GetMobilePreferencesParams{AppID: appID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Preferences{}, fmt.Errorf("mobile user not found")
	}
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("get mobile preferences: %w", err)
	}
	return decodePreferences(row)
}
func (repository *Postgres) UpdatePreferences(ctx context.Context, input domain.PreferenceUpdate) (domain.Preferences, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("begin mobile preferences transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	row, err := queries.GetMobilePreferences(ctx, db.GetMobilePreferencesParams{AppID: input.AppID, UserID: input.UserID})
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("get mobile preferences: %w", err)
	}
	current, err := decodePreferences(row)
	if err != nil {
		return domain.Preferences{}, err
	}
	if input.Locale != nil {
		current.Locale = *input.Locale
	}
	if input.Appearance != nil {
		current.Appearance = *input.Appearance
	}
	for key, value := range input.NotificationPreferences {
		current.NotificationPreferences[key] = value
	}
	raw, err := json.Marshal(current.NotificationPreferences)
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("encode notification preferences: %w", err)
	}
	if err = queries.UpsertMobilePreferences(ctx, db.UpsertMobilePreferencesParams{AppID: input.AppID, UserID: input.UserID, Locale: current.Locale, Appearance: current.Appearance, NotificationPreferences: raw}); err != nil {
		return domain.Preferences{}, fmt.Errorf("update mobile preferences: %w", err)
	}
	after, err := json.Marshal(map[string]any{"locale": current.Locale, "appearance": current.Appearance, "notification_preferences": current.NotificationPreferences})
	if err != nil {
		return domain.Preferences{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs (tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,after_data,succeeded) VALUES ($1,$2,$3,$4,'iam','mobile_preferences.update','iam.preference.manage_self','iam.user_preferences',$5,'PATCH','/api/v1/me/preferences',$6,true)`, input.TenantID, input.UserID, input.SessionID, input.RequestID, input.AppID.String()+":"+input.UserID.String(), after); err != nil {
		return domain.Preferences{}, fmt.Errorf("audit mobile preferences: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Preferences{}, fmt.Errorf("commit mobile preferences: %w", err)
	}
	return current, nil
}
func (repository *Postgres) UnreadCount(ctx context.Context, userID, tenantID, appID uuid.UUID) (int64, error) {
	return db.New(repository.pool).CountMobileUnreadNotifications(ctx, db.CountMobileUnreadNotificationsParams{TenantID: tenantID, AppID: appID, UserID: userID})
}
func (repository *Postgres) LoginEvents(ctx context.Context, userID, appID uuid.UUID) ([]domain.LoginEvent, error) {
	rows, err := db.New(repository.pool).ListMobileLoginEvents(ctx, db.ListMobileLoginEventsParams{UserID: &userID, AppID: &appID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.LoginEvent, 0, len(rows))
	for _, row := range rows {
		var ip *string
		if row.ClientIp != nil {
			value := row.ClientIp.String()
			ip = &value
		}
		out = append(out, domain.LoginEvent{ID: row.ID, AuthMethod: row.AuthMethod, Result: row.Result, OccurredAt: row.OccurredAt.Time, IPAddress: ip})
	}
	return out, nil
}
func (repository *Postgres) SecurityEvents(ctx context.Context, userID, appID uuid.UUID) ([]domain.SecurityEvent, error) {
	rows, err := db.New(repository.pool).ListMobileSecurityEvents(ctx, db.ListMobileSecurityEventsParams{UserID: &userID, AppID: &appID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SecurityEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.SecurityEvent{ID: row.ID, EventType: row.EventType, Severity: row.Severity, OccurredAt: row.OccurredAt.Time})
	}
	return out, nil
}

type notificationCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

func decodeCursor(raw string) (notificationCursor, error) {
	if raw == "" {
		return notificationCursor{CreatedAt: time.Now().UTC().Add(time.Second), ID: uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")}, nil
	}
	encoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return notificationCursor{}, err
	}
	var cursor notificationCursor
	if err = json.Unmarshal(encoded, &cursor); err != nil || cursor.ID == uuid.Nil || cursor.CreatedAt.IsZero() {
		return notificationCursor{}, errors.New("invalid notification cursor")
	}
	return cursor, nil
}
func encodeCursor(createdAt time.Time, id uuid.UUID) *string {
	raw, _ := json.Marshal(notificationCursor{CreatedAt: createdAt.UTC(), ID: id})
	value := base64.RawURLEncoding.EncodeToString(raw)
	return &value
}
func (repository *Postgres) Notifications(ctx context.Context, userID, tenantID, appID uuid.UUID, cursor string, limit int) (domain.NotificationPage, error) {
	parsed, err := decodeCursor(cursor)
	if err != nil {
		return domain.NotificationPage{}, err
	}
	rows, err := db.New(repository.pool).ListMobileNotifications(ctx, db.ListMobileNotificationsParams{TenantID: tenantID, AppID: appID, UserID: userID, CursorCreatedAt: pgtype.Timestamptz{Time: parsed.CreatedAt, Valid: true}, CursorID: parsed.ID, PageLimit: int32(limit + 1)})
	if err != nil {
		return domain.NotificationPage{}, err
	}
	items := make([]domain.Notification, 0, min(limit, len(rows)))
	var next *string
	for index, row := range rows {
		if index == limit {
			last := items[len(items)-1]
			next = encodeCursor(last.CreatedAt, last.ID)
			break
		}
		item := domain.Notification{ID: row.ID, Title: row.Title, Body: row.Body, BodyFormat: row.BodyFormat, MessageType: row.MessageType, CreatedAt: row.CreatedAt.Time}
		if row.ReadAt.Valid {
			value := row.ReadAt.Time
			item.ReadAt = &value
		}
		items = append(items, item)
	}
	return domain.NotificationPage{Items: items, NextCursor: next}, nil
}
func (repository *Postgres) MarkNotificationRead(ctx context.Context, userID, tenantID, appID, sessionID, messageID uuid.UUID, requestID string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	affected, err := db.New(tx).MarkMobileNotificationRead(ctx, db.MarkMobileNotificationReadParams{TenantID: tenantID, AppID: appID, UserID: userID, MessageID: messageID})
	if err != nil {
		return err
	}
	if affected != 1 {
		return domain.ErrNotificationNotFound
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,resource_type,resource_id,http_method,request_path,succeeded) VALUES($1,$2,$3,$4,'notify','recipient.read','notify.recipient',$5,'PATCH','/api/v1/me/notifications/{id}/read',true)`, tenantID, userID, sessionID, requestID, messageID.String()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
