package worker

import (
	"context"
	"errors"
	storage "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"time"
)

// Cleanup shares the worker lifecycle and configured object storage. Expired
// unreferenced uploads remain retryable if either deletion or commit fails.
type Cleanup struct {
	pool    *pgxpool.Pool
	objects storage.ObjectStore
}

func NewCleanup(p *pgxpool.Pool, o storage.ObjectStore) *Cleanup { return &Cleanup{p, o} }
func (w *Cleanup) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if e := w.Sweep(ctx); e != nil && ctx.Err() == nil {
			slog.Warn("feedback upload cleanup failed", "code", "FEEDBACK.CLEANUP.FAILED")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (w *Cleanup) Sweep(ctx context.Context) error {
	if w.objects == nil {
		return nil
	}
	for n := 0; n < 100; n++ {
		found, e := w.one(ctx)
		if e != nil || !found {
			return e
		}
	}
	return nil
}
func (w *Cleanup) one(ctx context.Context) (bool, error) {
	tx, e := w.pool.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	var fileID *uuid.UUID
	var ref storage.ObjectRef
	e = tx.QueryRow(ctx, `SELECT u.id,u.file_id,u.tenant_id,u.provider,u.bucket_name,u.object_key FROM storage.upload_sessions u WHERE u.purpose='feedback' AND u.expires_at<=now() AND u.status IN ('initiated','completed') AND NOT EXISTS(SELECT 1 FROM storage.file_usages a WHERE a.file_id=u.file_id) ORDER BY u.expires_at,u.id LIMIT 1 FOR UPDATE OF u SKIP LOCKED`).Scan(&id, &fileID, &ref.TenantID, &ref.Provider, &ref.Bucket, &ref.Key)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if e = w.objects.Delete(ctx, ref); e != nil && !errors.Is(e, storage.ErrObjectNotFound) {
		return false, e
	}
	if fileID != nil {
		if _, e = tx.Exec(ctx, `UPDATE storage.files SET status='deleted',deleted_at=now() WHERE id=$1 AND tenant_id=$2 AND metadata->>'purpose'='feedback' AND NOT EXISTS(SELECT 1 FROM storage.file_usages a WHERE a.file_id=$1)`, *fileID, ref.TenantID); e != nil {
			return false, e
		}
	}
	if _, e = tx.Exec(ctx, `UPDATE storage.upload_sessions SET status='expired' WHERE id=$1 AND tenant_id=$2 AND purpose='feedback'`, id, ref.TenantID); e != nil {
		return false, e
	}
	return true, tx.Commit(ctx)
}
