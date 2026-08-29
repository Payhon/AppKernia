package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	files "github.com/appkernia/appkernia/server/internal/modules/storageadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (r *Postgres) CreateUpload(ctx context.Context, in files.CreateUpload) (files.UploadSession, error) {
	media, partSize := in.MediaType, files.PartSize
	row, err := db.New(r.pool).CreateAdminFileUploadSession(ctx, db.CreateAdminFileUploadSessionParams{TenantID: in.TenantID, UserID: in.UserID, Provider: in.Provider, BucketName: in.Bucket, ObjectKey: in.ObjectKey, OriginalName: in.OriginalName, MediaType: &media, ExpectedSize: in.ExpectedSize, PartSize: &partSize, ExpiresAt: pgtype.Timestamptz{Time: in.ExpiresAt, Valid: true}})
	if err != nil {
		return files.UploadSession{}, fmt.Errorf("create admin upload: %w", err)
	}
	return uploadFrom(row.ID, row.OriginalName, stringValue(row.MediaType), row.ExpectedSize, int64Value(row.PartSize), row.Status, row.Provider, row.BucketName, row.ObjectKey, row.ExpiresAt, nil), nil
}

func (r *Postgres) GetUpload(ctx context.Context, tenantID, id uuid.UUID) (files.UploadSession, error) {
	q := db.New(r.pool)
	row, err := q.GetAdminFileUploadSession(ctx, db.GetAdminFileUploadSessionParams{ID: id, TenantID: tenantID})
	if errors.Is(err, pgx.ErrNoRows) {
		return files.UploadSession{}, files.ErrUploadNotFound
	}
	if err != nil {
		return files.UploadSession{}, fmt.Errorf("get admin upload: %w", err)
	}
	parts, err := q.ListAdminFileUploadParts(ctx, id)
	if err != nil {
		return files.UploadSession{}, fmt.Errorf("list upload parts: %w", err)
	}
	return uploadFrom(row.ID, row.OriginalName, stringValue(row.MediaType), row.ExpectedSize, int64Value(row.PartSize), row.Status, row.Provider, row.BucketName, row.ObjectKey, row.ExpiresAt, mapParts(parts)), nil
}

func (r *Postgres) UpsertPart(ctx context.Context, tenantID, id uuid.UUID, part files.Part) error {
	q := db.New(r.pool)
	if err := q.UpsertAdminFileUploadPart(ctx, db.UpsertAdminFileUploadPartParams{PartNumber: part.PartNumber, Etag: part.ETag, SizeBytes: part.SizeBytes, ChecksumSha256: part.Checksum, UploadSessionID: id, TenantID: tenantID}); err != nil {
		return fmt.Errorf("upsert upload part: %w", err)
	}
	if err := q.MarkAdminUploadUploading(ctx, db.MarkAdminUploadUploadingParams{ID: id, TenantID: tenantID}); err != nil {
		return fmt.Errorf("mark upload active: %w", err)
	}
	return nil
}

func (r *Postgres) AbortUpload(ctx context.Context, p files.Principal, id uuid.UUID) (files.UploadSession, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return files.UploadSession{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.New(tx)
	row, err := q.LockAdminFileUploadSession(ctx, db.LockAdminFileUploadSessionParams{ID: id, TenantID: p.TenantID})
	if errors.Is(err, pgx.ErrNoRows) {
		return files.UploadSession{}, files.ErrUploadNotFound
	}
	if err != nil {
		return files.UploadSession{}, err
	}
	parts, err := q.ListAdminFileUploadParts(ctx, id)
	if err != nil {
		return files.UploadSession{}, err
	}
	if affected, abortErr := q.AbortAdminFileUploadSession(ctx, db.AbortAdminFileUploadSessionParams{ID: id, TenantID: p.TenantID}); abortErr != nil || affected != 1 {
		return files.UploadSession{}, files.ErrUploadNotFound
	}
	if err = insertAudit(ctx, q, p, "storage.file.upload.cancel", "storage.upload_session", id.String(), "DELETE", "/admin-api/v1/files/upload-sessions/{id}", 200, map[string]any{"status": "aborted"}); err != nil {
		return files.UploadSession{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return files.UploadSession{}, err
	}
	return uploadFrom(row.ID, row.OriginalName, stringValue(row.MediaType), row.ExpectedSize, int64Value(row.PartSize), row.Status, row.Provider, row.BucketName, row.ObjectKey, row.ExpiresAt, mapParts(parts)), nil
}

func (r *Postgres) CompleteUpload(ctx context.Context, in files.CompleteUpload) (files.File, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return files.File{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.New(tx)
	session, err := q.LockAdminFileUploadSession(ctx, db.LockAdminFileUploadSessionParams{ID: in.UploadID, TenantID: in.TenantID})
	if errors.Is(err, pgx.ErrNoRows) {
		return files.File{}, files.ErrUploadNotFound
	}
	if err != nil {
		return files.File{}, err
	}
	if session.ObjectKey != in.ObjectKey || session.Provider != in.Provider || session.BucketName != in.Bucket || session.ExpectedSize != in.SizeBytes {
		return files.File{}, files.ErrInvalid
	}
	media, extension, owner := in.MediaType, in.Extension, session.UserID
	row, err := q.InsertReadyAdminFile(ctx, db.InsertReadyAdminFileParams{TenantID: in.TenantID, OwnerUserID: &owner, Provider: in.Provider, BucketName: in.Bucket, ObjectKey: in.ObjectKey, OriginalName: session.OriginalName, MediaType: &media, Extension: &extension, SizeBytes: in.SizeBytes, Sha256: in.SHA256, ScanStatus: in.ScanStatus})
	if err != nil {
		return files.File{}, fmt.Errorf("insert completed file: %w", err)
	}
	fileID := row.ID
	if affected, updateErr := q.CompleteAdminFileUploadSession(ctx, db.CompleteAdminFileUploadSessionParams{FileID: &fileID, ID: in.UploadID, TenantID: in.TenantID}); updateErr != nil || affected != 1 {
		return files.File{}, files.ErrUploadNotFound
	}
	if err = insertAudit(ctx, q, in.Principal, "storage.file.upload.complete", "storage.file", row.ID.String(), "POST", "/admin-api/v1/files/upload-sessions/{id}/complete", 201, map[string]any{"file_id": row.ID, "size_bytes": row.SizeBytes, "scan_status": row.ScanStatus}); err != nil {
		return files.File{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return files.File{}, err
	}
	return fileFromReady(row), nil
}

func (r *Postgres) ListFiles(ctx context.Context, tenantID uuid.UUID, f files.FileFilter) (files.FilePage, error) {
	q := db.New(r.pool)
	params := db.ListAdminFilesParams{TenantID: tenantID, Query: f.Query, Status: f.Status, ScanStatus: f.ScanStatus, MediaType: f.MediaType, Provider: f.Provider, CreatedFrom: nullableTimestamp(f.CreatedFrom), CreatedTo: nullableTimestamp(f.CreatedTo), PageOffset: (f.Page - 1) * f.PageSize, PageSize: f.PageSize}
	rows, err := q.ListAdminFiles(ctx, params)
	if err != nil {
		return files.FilePage{}, err
	}
	total, err := q.CountAdminFiles(ctx, db.CountAdminFilesParams{TenantID: tenantID, Query: f.Query, Status: f.Status, ScanStatus: f.ScanStatus, MediaType: f.MediaType, Provider: f.Provider, CreatedFrom: nullableTimestamp(f.CreatedFrom), CreatedTo: nullableTimestamp(f.CreatedTo)})
	if err != nil {
		return files.FilePage{}, err
	}
	items := make([]files.File, 0, len(rows))
	for _, row := range rows {
		items = append(items, files.File{ID: row.ID, OwnerUserID: row.OwnerUserID, OriginalName: row.OriginalName, MediaType: stringValue(row.MediaType), Extension: stringValue(row.Extension), SizeBytes: row.SizeBytes, Provider: row.Provider, Status: row.Status, ScanStatus: row.ScanStatus, UsageCount: row.UsageCount, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time})
	}
	return files.FilePage{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, nil
}

func nullableTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func (r *Postgres) GetFile(ctx context.Context, tenantID, id uuid.UUID) (files.File, error) {
	row, err := db.New(r.pool).GetAdminFile(ctx, db.GetAdminFileParams{ID: id, TenantID: tenantID})
	if errors.Is(err, pgx.ErrNoRows) {
		return files.File{}, files.ErrNotFound
	}
	if err != nil {
		return files.File{}, err
	}
	return files.File{ID: row.ID, OwnerUserID: row.OwnerUserID, OriginalName: row.OriginalName, MediaType: stringValue(row.MediaType), Extension: stringValue(row.Extension), SizeBytes: row.SizeBytes, Provider: row.Provider, Status: row.Status, ScanStatus: row.ScanStatus, UsageCount: row.UsageCount, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time, ObjectKey: row.ObjectKey, Bucket: row.BucketName, SHA256: row.Sha256}, nil
}
func (r *Postgres) ListUsages(ctx context.Context, tenantID, id uuid.UUID) ([]files.Usage, error) {
	if _, err := r.GetFile(ctx, tenantID, id); err != nil {
		return nil, err
	}
	rows, err := db.New(r.pool).ListAdminFileUsages(ctx, db.ListAdminFileUsagesParams{TenantID: tenantID, FileID: id})
	if err != nil {
		return nil, err
	}
	out := make([]files.Usage, 0, len(rows))
	for _, x := range rows {
		out = append(out, files.Usage{ID: x.ID, ModuleCode: x.ModuleCode, EntityType: x.EntityType, EntityID: x.EntityID, FieldName: x.FieldName, CreatedAt: x.CreatedAt.Time})
	}
	return out, nil
}
func (r *Postgres) DeleteFile(ctx context.Context, p files.Principal, id uuid.UUID) (files.File, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return files.File{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.New(tx)
	row, err := q.LockAdminFileForDelete(ctx, db.LockAdminFileForDeleteParams{ID: id, TenantID: p.TenantID})
	if errors.Is(err, pgx.ErrNoRows) {
		return files.File{}, files.ErrNotFound
	}
	if err != nil {
		return files.File{}, err
	}
	if row.UsageCount > 0 {
		return files.File{}, files.ErrFileInUse
	}
	if affected, e := q.SoftDeleteAdminFile(ctx, db.SoftDeleteAdminFileParams{ID: id, TenantID: p.TenantID}); e != nil || affected != 1 {
		return files.File{}, files.ErrNotFound
	}
	if err = insertAudit(ctx, q, p, "storage.file.delete", "storage.file", id.String(), "DELETE", "/admin-api/v1/files/{id}", 200, map[string]any{"file_id": id, "original_name": row.OriginalName}); err != nil {
		return files.File{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return files.File{}, err
	}
	return files.File{ID: row.ID, OriginalName: row.OriginalName, Provider: row.Provider, Bucket: row.BucketName, ObjectKey: row.ObjectKey}, nil
}

func uploadFrom(id uuid.UUID, name, media string, size, partSize int64, status, provider, bucket, key string, expires pgtype.Timestamptz, parts []files.Part) files.UploadSession {
	if parts == nil {
		parts = []files.Part{}
	}
	return files.UploadSession{ID: id, OriginalName: name, MediaType: media, ExpectedSize: size, PartSize: partSize, Status: status, Provider: provider, Bucket: bucket, ObjectKey: key, ExpiresAt: expires.Time, UploadedParts: parts}
}
func mapParts(rows []db.ListAdminFileUploadPartsRow) []files.Part {
	out := make([]files.Part, 0, len(rows))
	for _, x := range rows {
		out = append(out, files.Part{PartNumber: x.PartNumber, SizeBytes: x.SizeBytes, ETag: x.Etag, Checksum: x.ChecksumSha256})
	}
	return out
}
func fileFromReady(row db.InsertReadyAdminFileRow) files.File {
	return files.File{ID: row.ID, OwnerUserID: row.OwnerUserID, OriginalName: row.OriginalName, MediaType: stringValue(row.MediaType), Extension: stringValue(row.Extension), SizeBytes: row.SizeBytes, Provider: row.Provider, Bucket: row.BucketName, Status: row.Status, ScanStatus: row.ScanStatus, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time, ObjectKey: row.ObjectKey, SHA256: row.Sha256}
}
func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func int64Value(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
func insertAudit(ctx context.Context, q *db.Queries, p files.Principal, action, resourceType, resourceID, method, path string, status int32, data any) error {
	raw, _ := json.Marshal(data)
	tenant, user, session := p.TenantID, p.UserID, p.SessionID
	var ip *netip.Addr
	if parsed, e := netip.ParseAddr(p.IPAddress); e == nil {
		ip = &parsed
	}
	var ua *string
	if x := strings.TrimSpace(p.UserAgent); x != "" {
		ua = &x
	}
	return q.InsertAdminStorageAudit(ctx, db.InsertAdminStorageAuditParams{TenantID: &tenant, UserID: &user, SessionID: &session, RequestID: p.RequestID, ActionName: action, ResourceType: &resourceType, ResourceID: &resourceID, HttpMethod: &method, RequestPath: &path, ResponseStatus: &status, ClientIp: ip, UserAgent: ua, AfterData: raw})
}
