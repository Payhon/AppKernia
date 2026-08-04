package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	"github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (repository *Postgres) CreateAvatarUpload(ctx context.Context, input domain.CreateAvatarUpload) (domain.AvatarUploadSession, error) {
	mediaType := input.MediaType
	row, err := db.New(repository.pool).CreateSelfAvatarUploadSession(ctx, db.CreateSelfAvatarUploadSessionParams{
		TenantID: input.TenantID, UserID: input.UserID, ObjectKey: input.ObjectKey,
		OriginalName: input.OriginalName, MediaType: &mediaType, ExpectedSize: input.ExpectedSize,
		ExpiresAt: pgtype.Timestamptz{Time: input.ExpiresAt, Valid: true},
	})
	if err != nil {
		return domain.AvatarUploadSession{}, fmt.Errorf("create avatar upload session: %w", err)
	}
	return domain.AvatarUploadSession{
		ID: row.ID, ObjectKey: row.ObjectKey, OriginalName: input.OriginalName,
		MediaType: input.MediaType, ExpectedSize: input.ExpectedSize, ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

func (repository *Postgres) GetAvatarUpload(ctx context.Context, principal domain.Principal, uploadID uuid.UUID) (domain.AvatarUploadSession, error) {
	row, err := db.New(repository.pool).GetSelfAvatarUploadSession(ctx, db.GetSelfAvatarUploadSessionParams{
		ID: uploadID, TenantID: principal.TenantID, UserID: principal.UserID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AvatarUploadSession{}, domain.ErrUploadNotFound
	}
	if err != nil {
		return domain.AvatarUploadSession{}, fmt.Errorf("get avatar upload session: %w", err)
	}
	return domain.AvatarUploadSession{
		ID: row.ID, ObjectKey: row.ObjectKey, OriginalName: row.OriginalName,
		MediaType: valueOrEmpty(row.MediaType), ExpectedSize: row.ExpectedSize, ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

func (repository *Postgres) CompleteAvatarUpload(ctx context.Context, input domain.CompleteAvatarUpload) (domain.AvatarCompletion, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.AvatarCompletion{}, fmt.Errorf("begin avatar completion transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	session, err := queries.LockSelfAvatarUploadSession(ctx, db.LockSelfAvatarUploadSessionParams{
		ID: input.UploadSessionID, TenantID: input.TenantID, UserID: input.UserID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AvatarCompletion{}, domain.ErrUploadNotFound
	}
	if err != nil {
		return domain.AvatarCompletion{}, fmt.Errorf("lock avatar upload session: %w", err)
	}
	if session.ObjectKey != input.ObjectKey || session.ExpectedSize != input.SizeBytes || valueOrEmpty(session.MediaType) != input.MediaType {
		return domain.AvatarCompletion{}, domain.ErrUploadInvalid
	}
	beforeFileID, err := queries.GetSelfAvatarFileIDForUpdate(ctx, input.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AvatarCompletion{}, domain.ErrUploadNotFound
	}
	if err != nil {
		return domain.AvatarCompletion{}, fmt.Errorf("lock avatar owner: %w", err)
	}
	mediaType := input.MediaType
	extension := input.Extension
	ownerID := input.UserID
	storedFile, err := queries.InsertReadySelfAvatarFile(ctx, db.InsertReadySelfAvatarFileParams{
		TenantID: input.TenantID, UserID: &ownerID, ObjectKey: input.ObjectKey,
		OriginalName: input.OriginalName, MediaType: &mediaType, Extension: &extension,
		SizeBytes: input.SizeBytes, Sha256: input.SHA256,
	})
	if err != nil {
		return domain.AvatarCompletion{}, fmt.Errorf("insert avatar file metadata: %w", err)
	}
	fileID := storedFile.ID
	if err = queries.CompleteSelfAvatarUploadSession(ctx, db.CompleteSelfAvatarUploadSessionParams{
		FileID: &fileID, ID: input.UploadSessionID, TenantID: input.TenantID, UserID: input.UserID,
	}); err != nil {
		return domain.AvatarCompletion{}, fmt.Errorf("complete avatar upload session: %w", err)
	}
	if err = queries.DeletePreviousSelfAvatarUsage(ctx, db.DeletePreviousSelfAvatarUsageParams{TenantID: input.TenantID, UserID: input.UserID}); err != nil {
		return domain.AvatarCompletion{}, fmt.Errorf("clear previous avatar usage: %w", err)
	}
	if err = queries.InsertSelfAvatarUsage(ctx, db.InsertSelfAvatarUsageParams{FileID: fileID, TenantID: input.TenantID, UserID: input.UserID}); err != nil {
		return domain.AvatarCompletion{}, fmt.Errorf("insert avatar usage: %w", err)
	}
	if err = queries.UpdateSelfAvatarFile(ctx, db.UpdateSelfAvatarFileParams{FileID: &fileID, UserID: input.UserID}); err != nil {
		return domain.AvatarCompletion{}, fmt.Errorf("update self avatar: %w", err)
	}
	resourceID := input.UserID.String()
	userAgent := strings.TrimSpace(input.UserAgent)
	var userAgentValue *string
	if userAgent != "" {
		userAgentValue = &userAgent
	}
	tenantID, userID, sessionID := input.TenantID, input.UserID, input.SessionID
	if err = queries.InsertSelfAvatarAudit(ctx, db.InsertSelfAvatarAuditParams{
		TenantID: &tenantID, UserID: &userID, SessionID: &sessionID,
		RequestID: input.RequestID, ResourceID: &resourceID, ClientIp: input.IPAddress,
		UserAgent: userAgentValue, BeforeFileID: beforeFileID, AfterFileID: fileID,
	}); err != nil {
		return domain.AvatarCompletion{}, fmt.Errorf("audit self avatar update: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.AvatarCompletion{}, fmt.Errorf("commit self avatar update: %w", err)
	}
	return domain.AvatarCompletion{FileID: fileID, ObjectKey: storedFile.ObjectKey}, nil
}

func (repository *Postgres) GetAvatarObject(ctx context.Context, principal domain.Principal) (domain.AvatarObject, error) {
	row, err := db.New(repository.pool).GetSelfAvatarObject(ctx, db.GetSelfAvatarObjectParams{
		UserID: principal.UserID, TenantID: principal.TenantID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AvatarObject{}, domain.ErrAvatarNotFound
	}
	if err != nil {
		return domain.AvatarObject{}, fmt.Errorf("get self avatar object: %w", err)
	}
	return domain.AvatarObject{
		FileID: row.ID, ObjectKey: row.ObjectKey, MediaType: valueOrEmpty(row.MediaType),
		SizeBytes: row.SizeBytes, SHA256: row.Sha256, UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
