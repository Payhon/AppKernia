//go:build integration

package repository

import (
	"bytes"
	"context"
	"os"
	"testing"

	app "github.com/appkernia/appkernia/server/internal/modules/appmanagement/application"
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

func TestAppOTPNotifierEncryptsAndAtomicallyQueues(t *testing.T) {
	dsn := os.Getenv("AK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	suffix := uuid.NewString()
	email := "otp-delivery-" + suffix + "@example.test"
	_, tenant, err := iamrepo.NewPostgres(pool).CreateIdentity(ctx, iamdomain.CreateIdentity{TenantCode: "otp-delivery-" + suffix, TenantName: "OTP Delivery", Email: email, DisplayName: "OTP Delivery", Locale: "en-US", PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM river_job WHERE args->>'delivery_id' IN (SELECT id::text FROM notify.deliveries WHERE tenant_id=$1)`, tenant.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM iam.tenants WHERE id=$1`, tenant.ID)
	}()
	var appID uuid.UUID
	if err = pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, tenant.ID).Scan(&appID); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notify.templates WHERE code='verification_code' AND channel='email' AND status='active')`).Scan(&exists); err != nil || !exists {
		t.Skip("verification_code template is not seeded")
	}
	sealer, err := settingsrepo.NewAESGCMSealer(bytes.Repeat([]byte{0x5a}, 32), 11)
	if err != nil {
		t.Fatal(err)
	}
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{SkipUnknownJobCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	trackedQueue := jobqueue.NewRiverAdapter(pool, riverClient, jobdefs.Registry())
	n, err := NewAppOTPNotifier(trackedQueue, sealer)
	if err != nil {
		t.Fatal(err)
	}
	code := "731942"
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err = n.QueueOTP(ctx, tx, app.OTPNotification{AppID: appID, TenantID: tenant.ID, Email: email, Code: code, Purpose: "email_otp", Locale: "en-US", ExpiresMinutes: 10}); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var id uuid.UUID
	var foundApp uuid.UUID
	var target, payload []byte
	var subject, body string
	if err = pool.QueryRow(ctx, `SELECT id,app_id,target_ciphertext,payload_ciphertext,COALESCE(rendered_subject,''),COALESCE(rendered_body,'') FROM notify.deliveries WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT 1`, tenant.ID).Scan(&id, &foundApp, &target, &payload, &subject, &body); err != nil {
		t.Fatal(err)
	}
	if foundApp != appID || subject != "" || body != "" || bytes.Contains(target, []byte(email)) || bytes.Contains(payload, []byte(code)) {
		t.Fatalf("OTP persisted unsafely app=%s subject=%q body=%q", foundApp, subject, body)
	}
	var jobs int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE args->>'delivery_id'=$1`, id.String()).Scan(&jobs); err != nil || jobs != 1 {
		t.Fatalf("jobs=%d err=%v", jobs, err)
	}
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.QueueOTP(ctx, tx, app.OTPNotification{AppID: appID, TenantID: tenant.ID, Email: "rollback-" + email, Code: "111222", Purpose: "password_reset", Locale: "en-US", ExpiresMinutes: 10}); err != nil {
		t.Fatal(err)
	}
	_ = tx.Rollback(ctx)
	var leaked int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM notify.deliveries WHERE tenant_id=$1 AND target_hint LIKE 'r***%'`, tenant.ID).Scan(&leaked); err != nil {
		t.Fatal(err)
	}
	if leaked != 0 {
		t.Fatalf("rollback leaked deliveries=%d", leaked)
	}
}
