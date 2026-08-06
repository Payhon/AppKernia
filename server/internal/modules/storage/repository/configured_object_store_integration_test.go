//go:build integration

package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type integrationSecretOpener struct{}

func (integrationSecretOpener) Open(ciphertext []byte, _ string) ([]byte, error) {
	return bytes.Clone(ciphertext), nil
}

func TestConfiguredObjectStoreCustomS3CompatiblePutOpenDelete(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	endpoint := strings.TrimSpace(os.Getenv("AK_TEST_S3_ENDPOINT"))
	if databaseURL == "" || endpoint == "" {
		t.Skip("AK_TEST_DATABASE_URL and AK_TEST_S3_ENDPOINT are required")
	}
	accessKey := valueOrDefault(os.Getenv("AK_TEST_S3_ACCESS_KEY"), "appkernia")
	secretKey := valueOrDefault(os.Getenv("AK_TEST_S3_SECRET_KEY"), "appkernia-dev-only")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create PostgreSQL pool: %v", err)
	}
	defer pool.Close()

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	_, tenant, err := iamrepo.NewPostgres(pool).CreateIdentity(ctx, iamdomain.CreateIdentity{
		TenantCode: "s3-" + suffix[:12], TenantName: "S3 integration",
		Email: "s3-" + suffix + "@example.test", DisplayName: "S3 integration",
		Locale: "zh-CN", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	provider := "custom-" + suffix[:8]
	bucket := "ak-e2e-" + suffix[:20]
	var dictionaryTypeID uuid.UUID
	if err = pool.QueryRow(ctx, `SELECT id FROM sys.dict_types WHERE tenant_id IS NULL AND code='storage.driver' AND status='active'`).Scan(&dictionaryTypeID); err != nil {
		t.Fatalf("core storage.driver dictionary is required: %v", err)
	}
	metadata, _ := json.Marshal(map[string]string{"adapter": "s3_compatible", "provider": provider})
	if _, err = pool.Exec(ctx, `INSERT INTO sys.dict_items(dict_type_id,tenant_id,item_value,label,locale,extra,status) VALUES($1,$2,$3,$4,'zh-CN',$5,'active')`, dictionaryTypeID, tenant.ID, provider, "Integration S3", metadata); err != nil {
		t.Fatalf("register tenant S3-compatible driver: %v", err)
	}

	values := map[string]any{
		"storage.driver":           provider,
		"storage.endpoint":         endpoint,
		"storage.region":           "us-east-1",
		"storage.bucket":           bucket,
		"storage.use_ssl":          false,
		"storage.force_path_style": true,
	}
	for key, value := range values {
		raw, _ := json.Marshal(value)
		if _, err = pool.Exec(ctx, `INSERT INTO sys.config_items(tenant_id,module_code,config_group,config_key,display_name,value_type,value_json,status) VALUES($1,'storage','cloud',$2::citext,$2::text,$3,$4,'active')`, tenant.ID, key, configValueType(value), raw); err != nil {
			t.Fatalf("insert %s: %v", key, err)
		}
	}
	for key, value := range map[string]string{"storage.access_key_id": accessKey, "storage.secret_access_key": secretKey} {
		if _, err = pool.Exec(ctx, `INSERT INTO sys.config_items(tenant_id,module_code,config_group,config_key,display_name,value_type,is_secret,secret_ciphertext,secret_key_version,status) VALUES($1,'storage','cloud',$2::citext,$2::text,'string',true,$3,1,'active')`, tenant.ID, key, []byte(value)); err != nil {
			t.Fatalf("insert %s: %v", key, err)
		}
	}

	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: false})
	if err != nil {
		t.Fatalf("create MinIO client: %v", err)
	}
	if err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
		t.Fatalf("create MinIO bucket: %v", err)
	}
	t.Cleanup(func() { _ = client.RemoveBucket(context.Background(), bucket) })

	store, err := NewConfiguredObjectStore(pool, integrationSecretOpener{}, t.TempDir(), "development")
	if err != nil {
		t.Fatalf("create configured store: %v", err)
	}
	ref := domain.ObjectRef{TenantID: tenant.ID, Provider: provider, Bucket: bucket, Key: "integration/round-trip.txt"}
	want := []byte("AppKernia custom S3-compatible integration")
	if err = store.Put(ctx, ref, want); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	reader, err := store.Open(ctx, ref)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	got, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if readErr != nil || !bytes.Equal(got, want) {
		t.Fatalf("Open() content mismatch: read=%v got=%q", readErr, got)
	}
	if err = store.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err = store.Open(ctx, ref); err == nil {
		t.Fatal("Open() after Delete() unexpectedly succeeded")
	}
}

func valueOrDefault(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

func configValueType(value any) string {
	if _, ok := value.(bool); ok {
		return "boolean"
	}
	return "string"
}
