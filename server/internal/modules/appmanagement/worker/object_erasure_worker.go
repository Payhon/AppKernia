package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	appdomain "github.com/appkernia/appkernia/server/internal/modules/appmanagement/domain"
	"github.com/appkernia/appkernia/server/internal/modules/appmanagement/jobdefs"
	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type ObjectErasureWorker struct {
	river.WorkerDefaults[appdomain.ObjectErasureJobArgs]
	pool    *pgxpool.Pool
	objects storagedomain.ObjectStore
}

func NewObjectErasureWorker(pool *pgxpool.Pool, objects storagedomain.ObjectStore) *ObjectErasureWorker {
	return &ObjectErasureWorker{pool: pool, objects: objects}
}

func (w *ObjectErasureWorker) Timeout(*river.Job[appdomain.ObjectErasureJobArgs]) time.Duration {
	return jobdefs.Timeout
}

func (w *ObjectErasureWorker) Work(ctx context.Context, job *river.Job[appdomain.ObjectErasureJobArgs]) error {
	if err := jobqueue.StartAttempt(ctx, w.pool, job.ID, job.Attempt); err != nil {
		return err
	}
	finish := func(completion jobqueue.Completion) {
		_ = jobqueue.FinishAttempt(ctx, w.pool, job.ID, job.Attempt, completion)
	}
	resultClass, errorCode, err := w.eraseObject(ctx, job.Args.ObjectID, job.Attempt, job.MaxAttempts)
	if err != nil {
		finish(jobqueue.Completion{Status: erasureRetryStatus(job.Attempt, job.MaxAttempts), ErrorCode: errorCode, ErrorSummary: err.Error()})
		return err
	}
	finish(jobqueue.Completion{Status: "succeeded", ResultClass: resultClass})
	return nil
}

func (w *ObjectErasureWorker) eraseObject(ctx context.Context, objectID uuid.UUID, attempt, maxAttempts int) (string, string, error) {
	if w.objects == nil {
		err := errors.New("object storage adapter is unavailable")
		if recordErr := w.recordFailure(ctx, objectID, attempt, attempt >= maxAttempts, "STORAGE.ADAPTER.UNAVAILABLE"); recordErr != nil {
			return "", "PRIVACY.ERASURE.UPDATE_FAILED", errors.Join(err, recordErr)
		}
		return "", "STORAGE.ADAPTER.UNAVAILABLE", err
	}
	var eventID, tenantID uuid.UUID
	var provider, bucket, key, status string
	err := w.pool.QueryRow(ctx, `UPDATE audit.privacy_erasure_objects
		SET attempt_count=GREATEST(attempt_count,$2)
		WHERE id=$1 AND status='pending'
		RETURNING event_id,tenant_id,provider,bucket_name,object_key,status`, objectID, attempt).
		Scan(&eventID, &tenantID, &provider, &bucket, &key, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "already_erased", "", nil
	}
	if err != nil {
		return "", "PRIVACY.ERASURE.LOAD_FAILED", err
	}
	err = w.objects.Delete(ctx, storagedomain.ObjectRef{TenantID: tenantID, Provider: provider, Bucket: bucket, Key: key})
	if err != nil && !errors.Is(err, storagedomain.ErrObjectNotFound) {
		terminal := attempt >= maxAttempts
		wrapped := fmt.Errorf("delete privacy erasure object: %w", err)
		if recordErr := w.recordFailure(ctx, objectID, attempt, terminal, "STORAGE.OBJECT.DELETE_FAILED"); recordErr != nil {
			return "", "PRIVACY.ERASURE.UPDATE_FAILED", errors.Join(wrapped, recordErr)
		}
		return "", "STORAGE.OBJECT.DELETE_FAILED", wrapped
	}
	if _, err = w.pool.Exec(ctx, `UPDATE audit.privacy_erasure_objects SET status='succeeded',completed_at=now(),last_error_code=NULL WHERE id=$1`, objectID); err != nil {
		return "", "PRIVACY.ERASURE.UPDATE_FAILED", err
	}
	_, _ = w.pool.Exec(ctx, `UPDATE audit.privacy_erasure_events e SET status='completed',completed_at=now()
		WHERE e.id=$1 AND NOT EXISTS (SELECT 1 FROM audit.privacy_erasure_objects o WHERE o.event_id=e.id AND o.status<>'succeeded')`, eventID)
	return "object_deleted", "", nil
}

func (w *ObjectErasureWorker) recordFailure(ctx context.Context, objectID uuid.UUID, attempt int, terminal bool, errorCode string) error {
	status := "pending"
	if terminal {
		status = "failed"
	}
	var eventID uuid.UUID
	if err := w.pool.QueryRow(ctx, `UPDATE audit.privacy_erasure_objects SET status=$2::varchar,last_error_code=$4::varchar,
		completed_at=CASE WHEN $2::text='failed' THEN now() ELSE NULL END,attempt_count=GREATEST(attempt_count,$3)
		WHERE id=$1 RETURNING event_id`, objectID, status, attempt, errorCode).Scan(&eventID); err != nil {
		return fmt.Errorf("record privacy erasure failure: %w", err)
	}
	if terminal && eventID != uuid.Nil {
		if _, err := w.pool.Exec(ctx, `UPDATE audit.privacy_erasure_events SET status='failed',completed_at=now() WHERE id=$1`, eventID); err != nil {
			return fmt.Errorf("record privacy erasure event failure: %w", err)
		}
	}
	return nil
}

func erasureRetryStatus(attempt, maxAttempts int) string {
	if attempt >= maxAttempts {
		return "failed"
	}
	return "retry_wait"
}
