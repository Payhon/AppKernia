//go:build integration

package repository

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"testing"
	"time"

	db "github.com/appkernia/appkernia/server/internal/infrastructure/db"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	storageapp "github.com/appkernia/appkernia/server/internal/modules/storage/application"
	"github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAvatarUploadIsSelfScopedTransactionalAndPrivate(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	defer pool.Close()
	suffix := uuid.New().String()
	user, tenant, err := iamrepo.NewPostgres(pool).CreateIdentity(ctx, iamdomain.CreateIdentity{
		TenantCode: "avatar-" + suffix, TenantName: "Avatar Integration",
		Email: "avatar-" + suffix + "@example.test", DisplayName: "Avatar User",
		Locale: "zh-CN", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration",
	})
	if err != nil {
		t.Fatalf("CreateIdentity() error = %v", err)
	}
	session, err := db.New(pool).CreateSession(ctx, db.CreateSessionParams{
		UserID: user.ID, TenantID: &tenant.ID, Audience: "ak-admin",
		AbsoluteExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		IdleExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(30 * time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	principal := domain.Principal{UserID: user.ID, TenantID: tenant.ID, SessionID: session.ID}
	objects, err := NewLocalObjectStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalObjectStore() error = %v", err)
	}
	service := storageapp.NewService(NewPostgres(pool), objects, true)
	content := integrationPNG(t)
	target, err := service.CreateAvatarUpload(ctx, principal, storageapp.CreateAvatarUploadInput{
		OriginalName: "avatar.png", MediaType: "image/png", SizeBytes: int64(len(content)),
	})
	if err != nil {
		t.Fatalf("CreateAvatarUpload() error = %v", err)
	}
	fileID, err := service.UploadAvatar(ctx, principal, target.ID, content, domain.ClientMetadata{RequestID: "avatar-integration"})
	if err != nil {
		t.Fatalf("UploadAvatar() error = %v", err)
	}
	var avatarFileID uuid.UUID
	var fileStatus, scanStatus, uploadStatus string
	var usageCount, auditCount int
	err = pool.QueryRow(ctx, `
		SELECT u.avatar_file_id, f.status, f.scan_status, us.status,
		       (SELECT count(*) FROM storage.file_usages fu WHERE fu.file_id = f.id),
		       (SELECT count(*) FROM audit.operation_logs ol
		         WHERE ol.user_id = u.id AND ol.action_name = 'iam.me.avatar.update'
		           AND ol.after_data->>'avatar_file_id' = f.id::text)
		FROM iam.users u
		JOIN storage.files f ON f.id = u.avatar_file_id
		JOIN storage.upload_sessions us ON us.file_id = f.id
		WHERE u.id = $1`, user.ID,
	).Scan(&avatarFileID, &fileStatus, &scanStatus, &uploadStatus, &usageCount, &auditCount)
	if err != nil {
		t.Fatalf("query avatar state: %v", err)
	}
	if avatarFileID != fileID || fileStatus != "ready" || scanStatus != "skipped" || uploadStatus != "completed" || usageCount != 1 || auditCount != 1 {
		t.Fatalf("unexpected avatar state: file=%s statuses=%s/%s/%s usage=%d audit=%d", avatarFileID, fileStatus, scanStatus, uploadStatus, usageCount, auditCount)
	}
	previousFileID := fileID
	replacement, err := service.CreateAvatarUpload(ctx, principal, storageapp.CreateAvatarUploadInput{
		OriginalName: "replacement.png", MediaType: "image/png", SizeBytes: int64(len(content)),
	})
	if err != nil {
		t.Fatalf("CreateAvatarUpload(replacement) error = %v", err)
	}
	fileID, err = service.UploadAvatar(ctx, principal, replacement.ID, content, domain.ClientMetadata{RequestID: "avatar-replacement"})
	if err != nil {
		t.Fatalf("UploadAvatar(replacement) error = %v", err)
	}
	if fileID != previousFileID {
		t.Fatalf("deduplicated replacement created file %s instead of reusing %s", fileID, previousFileID)
	}
	var currentUsageCount, readyFileCount int
	if err = pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM storage.file_usages WHERE file_id = $1),
			(SELECT count(*) FROM storage.files
			  WHERE tenant_id = $2 AND sha256 = (SELECT sha256 FROM storage.files WHERE id = $1)
			    AND size_bytes = $3 AND status = 'ready')`,
		previousFileID, tenant.ID, len(content),
	).Scan(&currentUsageCount, &readyFileCount); err != nil {
		t.Fatalf("query deduplicated avatar state: %v", err)
	}
	if currentUsageCount != 1 || readyFileCount != 1 {
		t.Fatalf("unexpected deduplicated avatar state: usage=%d ready_files=%d", currentUsageCount, readyFileCount)
	}
	object, reader, err := service.OpenAvatar(ctx, principal)
	if err != nil {
		t.Fatalf("OpenAvatar() error = %v", err)
	}
	stored, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if readErr != nil || object.FileID != fileID || !bytes.Equal(stored, content) {
		t.Fatalf("stored avatar mismatch: object=%#v read=%v", object, readErr)
	}
	other := domain.Principal{UserID: uuid.New(), TenantID: tenant.ID, SessionID: uuid.New()}
	if _, _, err = service.OpenAvatar(ctx, other); err != domain.ErrAvatarNotFound {
		t.Fatalf("other principal OpenAvatar() error = %v", err)
	}
}

func integrationPNG(t *testing.T) []byte {
	t.Helper()
	value := image.NewRGBA(image.Rect(0, 0, 8, 8))
	value.Set(0, 0, color.RGBA{R: 15, G: 23, B: 42, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, value); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buffer.Bytes()
}
