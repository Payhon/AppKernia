package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (repository *Postgres) RegisterAdmin(ctx context.Context, input domain.RegisterAdmin) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin admin registration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	tenant, err := queries.GetActiveTenantByCode(ctx, input.TenantCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrRegistrationTenant
	}
	if err != nil {
		return fmt.Errorf("get registration tenant: %w", err)
	}
	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email: &input.Email, DisplayName: input.DisplayName, Locale: input.Locale,
		TimeZone: "UTC", Status: "active", Metadata: json.RawMessage(`{"registration_source":"admin_self_service"}`),
	})
	if err != nil {
		return classifyCreateError(err)
	}
	if _, err = queries.CreateUserCredential(ctx, db.CreateUserCredentialParams{
		UserID: user.ID, PasswordHash: input.PasswordHash,
	}); err != nil {
		return fmt.Errorf("create registered credential: %w", err)
	}
	if _, err = queries.CreateTenantMember(ctx, db.CreateTenantMemberParams{
		TenantID: tenant.ID, UserID: user.ID, DisplayName: &input.DisplayName, Status: "active",
	}); err != nil {
		return fmt.Errorf("create registered membership: %w", err)
	}
	roleID, err := queries.GetDefaultMemberRole(ctx, tenant.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrRegistrationTenant
	}
	if err != nil {
		return fmt.Errorf("get registration role: %w", err)
	}
	if err = queries.AssignUserRole(ctx, db.AssignUserRoleParams{
		TenantID: tenant.ID, UserID: user.ID, RoleID: roleID,
	}); err != nil {
		return fmt.Errorf("assign registration role: %w", err)
	}
	resourceID := user.ID.String()
	userAgent := optionalString(input.UserAgent)
	if err = queries.InsertAdminRegistrationAudit(ctx, db.InsertAdminRegistrationAuditParams{
		TenantID: &tenant.ID, UserID: &user.ID, RequestID: optionalString(input.RequestID),
		ResourceID: &resourceID, ClientIp: input.IPAddress, UserAgent: userAgent,
	}); err != nil {
		return fmt.Errorf("audit admin registration: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return classifyCreateError(err)
	}
	return nil
}

func (repository *Postgres) PreparePasswordReset(
	ctx context.Context,
	input domain.PreparePasswordReset,
) (*domain.PasswordResetRecipient, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, fmt.Errorf("begin password reset challenge transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	credential, err := queries.GetCredentialByEmail(ctx, &input.Email)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && credential.Status != "active") {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find password reset identity: %w", err)
	}
	var tenantID uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT tenant_id FROM iam.tenant_members WHERE user_id=$1 AND status='active' ORDER BY created_at,tenant_id LIMIT 1`, credential.UserID).Scan(&tenantID); errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("resolve password reset tenant: %w", err)
	}
	userID := credential.UserID
	if _, err = queries.InsertPasswordResetChallenge(ctx, db.InsertPasswordResetChallengeParams{
		UserID: &userID, TargetHash: input.TargetHash, SecretHash: input.SecretHash,
		ExpiresAt: timestamp(input.ExpiresAt), CreatedIp: input.IPAddress,
	}); errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("create password reset challenge: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit password reset challenge: %w", err)
	}
	return &domain.PasswordResetRecipient{TenantID: tenantID, Email: valueOrEmpty(credential.Email), Locale: credential.Locale}, nil
}

func (repository *Postgres) GetPasswordResetState(ctx context.Context, tokenHash []byte) (domain.PasswordResetState, error) {
	queries := db.New(repository.pool)
	row, err := queries.GetPasswordResetState(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) || row.UserID == nil {
		return domain.PasswordResetState{}, domain.ErrResetTokenInvalid
	}
	if err != nil {
		return domain.PasswordResetState{}, fmt.Errorf("get password reset state: %w", err)
	}
	history, err := queries.ListRecentPasswordHashes(ctx, *row.UserID)
	if err != nil {
		return domain.PasswordResetState{}, fmt.Errorf("list password reset history: %w", err)
	}
	return domain.PasswordResetState{
		UserID: *row.UserID, CurrentHash: row.PasswordHash,
		CurrentVersion: row.PasswordVersion, HistoryHashes: history,
	}, nil
}

func (repository *Postgres) ResetPassword(ctx context.Context, input domain.ResetPassword) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin password reset transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	challenge, err := queries.LockPasswordResetChallenge(ctx, input.TokenHash)
	if errors.Is(err, pgx.ErrNoRows) || challenge.UserID == nil || *challenge.UserID != input.UserID {
		return domain.ErrResetTokenInvalid
	}
	if err != nil {
		return fmt.Errorf("lock password reset challenge: %w", err)
	}
	afterVersion, err := queries.UpdatePasswordAfterReset(ctx, db.UpdatePasswordAfterResetParams{
		NewHash: input.NewHash, UserID: input.UserID,
		ExpectedHash: input.ExpectedHash, ExpectedVersion: input.ExpectedVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPasswordChanged
	}
	if err != nil {
		return fmt.Errorf("update reset password: %w", err)
	}
	if err = queries.InsertPasswordHistory(ctx, db.InsertPasswordHistoryParams{
		UserID: input.UserID, PasswordHash: input.ExpectedHash,
	}); err != nil {
		return fmt.Errorf("insert reset password history: %w", err)
	}
	if err = queries.RevokeAllUserRefreshTokens(ctx, input.UserID); err != nil {
		return fmt.Errorf("revoke reset refresh tokens: %w", err)
	}
	revokedCount, err := queries.RevokeAllUserSessions(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("revoke reset sessions: %w", err)
	}
	userID := input.UserID
	if err = queries.ConsumePasswordResetChallenges(ctx, &userID); err != nil {
		return fmt.Errorf("consume password reset challenges: %w", err)
	}
	resourceID := input.UserID.String()
	if err = queries.InsertPasswordResetAudit(ctx, db.InsertPasswordResetAuditParams{
		UserID: &userID, RequestID: optionalString(input.RequestID), ResourceID: &resourceID,
		ClientIp: input.IPAddress, UserAgent: optionalString(input.UserAgent),
		BeforeVersion: input.ExpectedVersion, AfterVersion: afterVersion,
		RevokedSessionCount: revokedCount,
	}); err != nil {
		return fmt.Errorf("audit password reset: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit password reset: %w", err)
	}
	return nil
}
