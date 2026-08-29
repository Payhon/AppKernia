//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	files "github.com/appkernia/appkernia/server/internal/modules/storageadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAdminFileLifecycleIsTenantScopedAuditedAndUsageProtected(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	suffix := uuid.New().String()
	user, tenant, err := iamrepo.NewPostgres(pool).CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "files-" + suffix, TenantName: "Files Integration", Email: "files-" + suffix + "@example.test", DisplayName: "Files User", Locale: "zh-CN", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := db.New(pool).CreateSession(ctx, db.CreateSessionParams{UserID: user.ID, TenantID: &tenant.ID, Audience: "ak-admin", AbsoluteExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}, IdleExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(30 * time.Minute), Valid: true}})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(pool)
	principal := files.Principal{TenantID: tenant.ID, UserID: user.ID, SessionID: session.ID, RequestID: "files-integration", IPAddress: "127.0.0.1", UserAgent: "integration"}
	content := []byte("verified file content")
	digest := sha256.Sum256(content)
	upload, err := repo.CreateUpload(ctx, files.CreateUpload{Principal: principal, OriginalName: "report.txt", MediaType: "text/plain", ExpectedSize: int64(len(content)), ObjectKey: "files/" + tenant.ID.String() + "/" + uuid.New().String(), ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.UpsertPart(ctx, tenant.ID, upload.ID, files.Part{PartNumber: 1, SizeBytes: int64(len(content)), ETag: "etag", Checksum: digest[:]}); err != nil {
		t.Fatal(err)
	}
	file, err := repo.CompleteUpload(ctx, files.CompleteUpload{Principal: principal, UploadID: upload.ID, ObjectKey: upload.ObjectKey, MediaType: "text/plain", Extension: "txt", SizeBytes: int64(len(content)), SHA256: digest[:], ScanStatus: "skipped"})
	if err != nil {
		t.Fatal(err)
	}
	page, err := repo.ListFiles(ctx, tenant.ID, files.FileFilter{Page: 1, PageSize: 20})
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("page=%#v err=%v", page, err)
	}
	future := time.Now().Add(time.Hour)
	page, err = repo.ListFiles(ctx, tenant.ID, files.FileFilter{CreatedFrom: &future, Page: 1, PageSize: 20})
	if err != nil || page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("future-filtered page=%#v err=%v", page, err)
	}
	if _, err = repo.GetFile(ctx, uuid.New(), file.ID); !errors.Is(err, files.ErrNotFound) {
		t.Fatalf("cross tenant error=%v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO storage.file_usages(file_id,tenant_id,module_code,entity_type,entity_id,field_name) VALUES($1,$2,'test','iam.user',$3,'attachment')`, file.ID, tenant.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	usages, err := repo.ListUsages(ctx, tenant.ID, file.ID)
	if err != nil || len(usages) != 1 {
		t.Fatalf("usages=%#v err=%v", usages, err)
	}
	if _, err = repo.DeleteFile(ctx, principal, file.ID); !errors.Is(err, files.ErrFileInUse) {
		t.Fatalf("in-use delete error=%v", err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM storage.file_usages WHERE tenant_id=$1 AND file_id=$2`, tenant.ID, file.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.DeleteFile(ctx, principal, file.ID); err != nil {
		t.Fatal(err)
	}
	var completed, deleted, audits int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM storage.upload_sessions WHERE id=$1 AND status='completed'),(SELECT count(*) FROM storage.files WHERE id=$2 AND status='deleted' AND deleted_at IS NOT NULL),(SELECT count(*) FROM audit.operation_logs WHERE tenant_id=$3 AND action_name IN ('storage.file.upload.complete','storage.file.delete'))`, upload.ID, file.ID, tenant.ID).Scan(&completed, &deleted, &audits); err != nil {
		t.Fatal(err)
	}
	if completed != 1 || deleted != 1 || audits != 2 {
		t.Fatalf("completed=%d deleted=%d audits=%d", completed, deleted, audits)
	}
}
