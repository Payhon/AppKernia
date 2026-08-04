package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (repository *Postgres) CreateIdentity(ctx context.Context, raw domain.CreateIdentity) (domain.User, domain.Tenant, error) {
	input := raw.Normalize()
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.User{}, domain.Tenant{}, fmt.Errorf("begin identity transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	tenant, err := queries.CreateTenant(ctx, db.CreateTenantParams{
		Code: input.TenantCode, Name: input.TenantName, Status: "active", Settings: json.RawMessage(`{}`),
	})
	if err != nil {
		return domain.User{}, domain.Tenant{}, classifyCreateError(err)
	}
	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email: &input.Email, DisplayName: input.DisplayName, Locale: input.Locale,
		TimeZone: "UTC", Status: "active", Metadata: json.RawMessage(`{}`),
	})
	if err != nil {
		return domain.User{}, domain.Tenant{}, classifyCreateError(err)
	}
	if _, err = queries.CreateUserCredential(ctx, db.CreateUserCredentialParams{UserID: user.ID, PasswordHash: input.PasswordHash}); err != nil {
		return domain.User{}, domain.Tenant{}, fmt.Errorf("create credential: %w", err)
	}
	if _, err = queries.CreateTenantMember(ctx, db.CreateTenantMemberParams{
		TenantID: tenant.ID, UserID: user.ID, DisplayName: &input.DisplayName, Status: "active",
	}); err != nil {
		return domain.User{}, domain.Tenant{}, fmt.Errorf("create tenant membership: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.User{}, domain.Tenant{}, classifyCreateError(err)
	}
	return mapUser(user), domain.Tenant{ID: tenant.ID, Code: tenant.Code, Name: tenant.Name}, nil
}

func (repository *Postgres) FindCredentialByEmail(ctx context.Context, email string) (domain.Credential, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	row, err := db.New(repository.pool).GetCredentialByEmail(ctx, &normalizedEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Credential{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.Credential{}, fmt.Errorf("find credential: %w", err)
	}
	return domain.Credential{User: domain.User{
		ID: row.UserID, Email: valueOrEmpty(row.Email), DisplayName: row.DisplayName,
		Locale: row.Locale, Status: row.Status,
	}, PasswordHash: row.PasswordHash}, nil
}

func (repository *Postgres) ListUserTenants(ctx context.Context, userID uuid.UUID) ([]domain.Tenant, error) {
	rows, err := db.New(repository.pool).ListUserTenants(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user tenants: %w", err)
	}
	result := make([]domain.Tenant, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.Tenant{ID: row.ID, Code: row.Code, Name: row.Name, Status: "active"})
	}
	return result, nil
}

func (repository *Postgres) UpdateSelfProfile(ctx context.Context, input domain.UpdateSelfProfile) (domain.User, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.User{}, fmt.Errorf("begin self profile transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	before, err := queries.GetUserByID(ctx, input.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get self profile: %w", err)
	}
	after, err := queries.UpdateSelfProfile(ctx, db.UpdateSelfProfileParams{
		ID: input.UserID, DisplayName: input.DisplayName, Locale: input.Locale, TimeZone: input.TimeZone,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("update self profile: %w", err)
	}
	beforeData, err := profileAuditData(before)
	if err != nil {
		return domain.User{}, err
	}
	afterData, err := profileAuditData(after)
	if err != nil {
		return domain.User{}, err
	}
	userAgent := strings.TrimSpace(input.UserAgent)
	var userAgentValue *string
	if userAgent != "" {
		userAgentValue = &userAgent
	}
	resourceID := input.UserID.String()
	var sessionID *uuid.UUID
	if input.SessionID != uuid.Nil {
		sessionID = &input.SessionID
	}
	if err = queries.InsertSelfProfileAudit(ctx, db.InsertSelfProfileAuditParams{
		TenantID: &input.TenantID, UserID: &input.UserID, SessionID: sessionID,
		RequestID: input.RequestID, ResourceID: &resourceID, ClientIp: input.IPAddress,
		UserAgent: userAgentValue, BeforeData: beforeData, AfterData: afterData,
	}); err != nil {
		return domain.User{}, fmt.Errorf("audit self profile update: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.User{}, fmt.Errorf("commit self profile update: %w", err)
	}
	return mapUser(after), nil
}

func (repository *Postgres) GetSelfPasswordState(ctx context.Context, userID uuid.UUID) (domain.SelfPasswordState, error) {
	queries := db.New(repository.pool)
	credential, err := queries.GetSelfPasswordState(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SelfPasswordState{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.SelfPasswordState{}, fmt.Errorf("get self password state: %w", err)
	}
	history, err := queries.ListRecentPasswordHashes(ctx, userID)
	if err != nil {
		return domain.SelfPasswordState{}, fmt.Errorf("list recent password hashes: %w", err)
	}
	return domain.SelfPasswordState{
		CurrentHash: credential.PasswordHash, CurrentVersion: credential.PasswordVersion, HistoryHashes: history,
	}, nil
}

func (repository *Postgres) ChangeSelfPassword(ctx context.Context, input domain.ChangeSelfPassword) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin self password transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	afterVersion, err := queries.UpdateSelfPasswordConditional(ctx, db.UpdateSelfPasswordConditionalParams{
		NewHash: input.NewHash, UserID: input.UserID,
		ExpectedHash: input.ExpectedHash, ExpectedVersion: input.ExpectedVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPasswordChanged
	}
	if err != nil {
		return fmt.Errorf("update self password: %w", err)
	}
	if err = queries.InsertPasswordHistory(ctx, db.InsertPasswordHistoryParams{
		UserID: input.UserID, PasswordHash: input.ExpectedHash,
	}); err != nil {
		return fmt.Errorf("insert password history: %w", err)
	}
	if err = queries.RevokeOtherSessionRefreshTokens(ctx, db.RevokeOtherSessionRefreshTokensParams{
		UserID: input.UserID, CurrentSessionID: input.SessionID,
	}); err != nil {
		return fmt.Errorf("revoke other session refresh tokens: %w", err)
	}
	revokedCount, err := queries.RevokeOtherSessions(ctx, db.RevokeOtherSessionsParams{
		UserID: input.UserID, CurrentSessionID: input.SessionID,
	})
	if err != nil {
		return fmt.Errorf("revoke other sessions: %w", err)
	}
	resourceID := input.UserID.String()
	userAgent := strings.TrimSpace(input.UserAgent)
	var userAgentValue *string
	if userAgent != "" {
		userAgentValue = &userAgent
	}
	if err = queries.InsertSelfPasswordChangeAudit(ctx, db.InsertSelfPasswordChangeAuditParams{
		TenantID: &input.TenantID, UserID: &input.UserID, SessionID: &input.SessionID,
		RequestID: input.RequestID, ResourceID: &resourceID, ClientIp: input.IPAddress,
		UserAgent: userAgentValue, BeforeVersion: input.ExpectedVersion,
		AfterVersion: afterVersion, RevokedSessionCount: revokedCount,
	}); err != nil {
		return fmt.Errorf("audit self password change: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit self password change: %w", err)
	}
	return nil
}

func classifyCreateError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" && pgError.ConstraintName == "uq_users_email_active" {
		return domain.ErrEmailAlreadyExists
	}
	return fmt.Errorf("create identity: %w", err)
}

func mapUser(user db.IamUser) domain.User {
	return domain.User{
		ID: user.ID, Email: valueOrEmpty(user.Email), DisplayName: user.DisplayName,
		Locale: user.Locale, TimeZone: user.TimeZone, Status: user.Status, AvatarFileID: user.AvatarFileID,
	}
}

func profileAuditData(user db.IamUser) (json.RawMessage, error) {
	value, err := json.Marshal(map[string]string{
		"display_name": user.DisplayName,
		"locale":       user.Locale,
		"time_zone":    user.TimeZone,
	})
	if err != nil {
		return nil, fmt.Errorf("encode self profile audit data: %w", err)
	}
	return value, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
