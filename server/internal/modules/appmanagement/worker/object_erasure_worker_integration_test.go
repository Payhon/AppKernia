//go:build integration

package worker

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	storagedomain "github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type erasureObjectStore struct {
	deleteErr error
	refs      []storagedomain.ObjectRef
}

func (*erasureObjectStore) ResolvePolicy(context.Context, uuid.UUID) (storagedomain.UploadPolicy, error) {
	return storagedomain.UploadPolicy{}, errors.New("not used")
}

func (*erasureObjectStore) Put(context.Context, storagedomain.ObjectRef, []byte) error {
	return errors.New("not used")
}

func (*erasureObjectStore) Open(context.Context, storagedomain.ObjectRef) (io.ReadCloser, error) {
	return nil, errors.New("not used")
}

func (s *erasureObjectStore) Delete(_ context.Context, ref storagedomain.ObjectRef) error {
	s.refs = append(s.refs, ref)
	return s.deleteErr
}

func TestObjectErasureRetriesAndIsIdempotent(t *testing.T) {
	pool := objectErasureTestPool(t)
	ctx := context.Background()
	tenantID, appID := createObjectErasureScope(t, pool)
	defer cleanupObjectErasureScope(pool, tenantID)
	eventID, objectID := createObjectErasureRecord(t, pool, tenantID, appID, "retry/"+uuid.NewString())
	store := &erasureObjectStore{deleteErr: errors.New("temporary storage failure")}
	worker := NewObjectErasureWorker(pool, store)

	if _, code, err := worker.eraseObject(ctx, objectID, 1, 3); err == nil || code != "STORAGE.OBJECT.DELETE_FAILED" {
		t.Fatalf("first attempt code=%q err=%v", code, err)
	}
	assertErasureState(t, pool, objectID, "pending", 1, "STORAGE.OBJECT.DELETE_FAILED")

	store.deleteErr = nil
	if result, code, err := worker.eraseObject(ctx, objectID, 2, 3); err != nil || code != "" || result != "object_deleted" {
		t.Fatalf("retry result=%q code=%q err=%v", result, code, err)
	}
	assertErasureState(t, pool, objectID, "succeeded", 2, "")
	var eventStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM audit.privacy_erasure_events WHERE id=$1`, eventID).Scan(&eventStatus); err != nil || eventStatus != "completed" {
		t.Fatalf("event status=%q err=%v", eventStatus, err)
	}

	if result, code, err := worker.eraseObject(ctx, objectID, 3, 3); err != nil || code != "" || result != "already_erased" {
		t.Fatalf("idempotent attempt result=%q code=%q err=%v", result, code, err)
	}
	if len(store.refs) != 2 {
		t.Fatalf("object store delete calls=%d, want 2", len(store.refs))
	}
}

func TestObjectErasureMarksTerminalFailure(t *testing.T) {
	pool := objectErasureTestPool(t)
	ctx := context.Background()
	tenantID, appID := createObjectErasureScope(t, pool)
	defer cleanupObjectErasureScope(pool, tenantID)
	eventID, objectID := createObjectErasureRecord(t, pool, tenantID, appID, "terminal/"+uuid.NewString())
	store := &erasureObjectStore{deleteErr: errors.New("permanent storage failure")}
	worker := NewObjectErasureWorker(pool, store)

	if _, code, err := worker.eraseObject(ctx, objectID, 10, 10); err == nil || code != "STORAGE.OBJECT.DELETE_FAILED" {
		t.Fatalf("terminal attempt code=%q err=%v", code, err)
	}
	assertErasureState(t, pool, objectID, "failed", 10, "STORAGE.OBJECT.DELETE_FAILED")
	var eventStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM audit.privacy_erasure_events WHERE id=$1`, eventID).Scan(&eventStatus); err != nil || eventStatus != "failed" {
		t.Fatalf("terminal event status=%q err=%v", eventStatus, err)
	}
}

func objectErasureTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("AK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func createObjectErasureScope(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	code := "erasure-worker-" + uuid.NewString()
	var tenantID, appID uuid.UUID
	if err := pool.QueryRow(context.Background(), `INSERT INTO iam.tenants(code,name) VALUES($1,$2) RETURNING id`, code, code).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, tenantID).Scan(&appID); err != nil {
		t.Fatal(err)
	}
	return tenantID, appID
}

func createObjectErasureRecord(t *testing.T, pool *pgxpool.Pool, tenantID, appID uuid.UUID, key string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	var eventID, objectID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO audit.privacy_erasure_events(tenant_id,app_id,status) VALUES($1,$2,'pending_objects') RETURNING id`, tenantID, appID).Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO audit.privacy_erasure_objects(event_id,tenant_id,provider,bucket_name,object_key) VALUES($1,$2,'local','privacy-test',$3) RETURNING id`, eventID, tenantID, key).Scan(&objectID); err != nil {
		t.Fatal(err)
	}
	return eventID, objectID
}

func assertErasureState(t *testing.T, pool *pgxpool.Pool, objectID uuid.UUID, wantStatus string, wantAttempts int, wantCode string) {
	t.Helper()
	var status string
	var attempts int
	var errorCode *string
	if err := pool.QueryRow(context.Background(), `SELECT status,attempt_count,last_error_code FROM audit.privacy_erasure_objects WHERE id=$1`, objectID).Scan(&status, &attempts, &errorCode); err != nil {
		t.Fatal(err)
	}
	if status != wantStatus || attempts != wantAttempts || (errorCode != nil && *errorCode != wantCode) || (errorCode == nil && wantCode != "") {
		t.Fatalf("erasure status=%q attempts=%d error=%v; want status=%q attempts=%d error=%q", status, attempts, errorCode, wantStatus, wantAttempts, wantCode)
	}
}

func cleanupObjectErasureScope(pool *pgxpool.Pool, tenantID uuid.UUID) {
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `DELETE FROM audit.privacy_erasure_events WHERE tenant_id=$1`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM iam.tenants WHERE id=$1`, tenantID)
}
