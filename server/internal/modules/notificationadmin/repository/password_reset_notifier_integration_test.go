//go:build integration

package repository

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	"github.com/appkernia/appkernia/server/internal/modules/notificationadmin/jobdefs"
	settingsrepo "github.com/appkernia/appkernia/server/internal/modules/systemsettings/repository"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func TestPasswordResetNotifierEncryptsAndAtomicallyQueuesDelivery(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	suffix := uuid.NewString()
	email := "reset-notifier-" + suffix + "@example.test"
	hash, err := iamapp.HashPassword("notifier integration password 2026!")
	if err != nil {
		t.Fatal(err)
	}
	identity, tenant, err := iamrepo.NewPostgres(pool).CreateIdentity(ctx, iamdomain.CreateIdentity{
		TenantCode:   "reset-notifier-" + suffix,
		TenantName:   "Reset Notifier",
		Email:        email,
		DisplayName:  "Reset Notifier",
		Locale:       "en-US",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = identity
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM river_job WHERE args->>'delivery_id' IN (SELECT id::text FROM notify.deliveries WHERE tenant_id=$1)`, tenant.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM iam.tenants WHERE id=$1`, tenant.ID)
	}()

	sealer, err := settingsrepo.NewAESGCMSealer(bytes.Repeat([]byte{0x4a}, 32), 9)
	if err != nil {
		t.Fatal(err)
	}
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{SkipUnknownJobCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	trackedQueue := jobqueue.NewRiverAdapter(pool, riverClient, jobdefs.Registry())
	notifier, err := NewPasswordResetNotifier(pool, trackedQueue, sealer, "https://admin.example.test")
	if err != nil {
		t.Fatal(err)
	}
	token := "opaque-password-reset-token-" + suffix
	if err = notifier.SendPasswordReset(ctx, iamapp.PasswordResetNotification{TenantID: tenant.ID, Email: email, Locale: "en-US", Token: token}); err != nil {
		t.Fatal(err)
	}

	var deliveryID uuid.UUID
	var targetCiphertext, payloadCiphertext []byte
	var targetVersion, payloadVersion int32
	var status string
	if err = pool.QueryRow(ctx, `SELECT id,target_ciphertext,payload_ciphertext,target_key_version,payload_key_version,status FROM notify.deliveries WHERE tenant_id=$1 AND channel='email' ORDER BY created_at DESC LIMIT 1`, tenant.ID).Scan(&deliveryID, &targetCiphertext, &payloadCiphertext, &targetVersion, &payloadVersion, &status); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || targetVersion != 9 || payloadVersion != 9 || bytes.Contains(targetCiphertext, []byte(email)) || bytes.Contains(payloadCiphertext, []byte(token)) {
		t.Fatalf("password reset delivery was not safely persisted: status=%s target_version=%d payload_version=%d", status, targetVersion, payloadVersion)
	}
	openedTarget, err := sealer.Open(targetCiphertext, tenant.ID.String())
	if err != nil || string(openedTarget) != email {
		t.Fatalf("open encrypted target: value=%q error=%v", openedTarget, err)
	}
	var queued int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE args->>'delivery_id'=$1`, deliveryID.String()).Scan(&queued); err != nil || queued != 1 {
		t.Fatalf("queued River jobs=%d error=%v", queued, err)
	}

	// The production code uses the same pgx transaction for delivery and River insert.
	// Keep this compile-time assertion close to the test so a driver type change cannot silently split it.
	var _ *river.Client[pgx.Tx] = riverClient
}
