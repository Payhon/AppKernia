//go:build integration

package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"testing"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	iamrepo "github.com/appkernia/appkernia/server/internal/modules/iam/repository"
	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/appkernia/appkernia/server/internal/modules/notificationadmin/jobdefs"
	notifyrepo "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/repository"
	push "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	pushprovider "github.com/appkernia/appkernia/server/internal/modules/push/provider"
	pushrepo "github.com/appkernia/appkernia/server/internal/modules/push/repository"
	settingsrepo "github.com/appkernia/appkernia/server/internal/modules/systemsettings/repository"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	platformnotify "github.com/appkernia/appkernia/server/internal/platform/notification"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

func TestMockPushPipelineSubmitFanoutDeliverOpenAndObserve(t *testing.T) {
	databaseURL := os.Getenv("AK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	suffix := uuid.NewString()
	user, tenant, err := iamrepo.NewPostgres(pool).CreateIdentity(ctx, iamdomain.CreateIdentity{
		TenantCode: "push-pipeline-" + suffix, TenantName: "Push Pipeline Integration",
		Email: "push-pipeline-" + suffix + "@example.test", DisplayName: "Push Recipient", Locale: "en-US",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$integration$integration",
	})
	if err != nil {
		t.Fatal(err)
	}
	var appID uuid.UUID
	if err = pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, tenant.ID).Scan(&appID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO app.user_memberships(app_id,tenant_id,user_id,source,status)
		VALUES($1,$2,$3,'admin_created','active')`, appID, tenant.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO iam.user_preferences(app_id,user_id,locale,appearance,notification_preferences)
		VALUES($1,$2,'en-US','system','{"in_app":true,"push":true,"push_service":true,"push_operations":false,"email":true}'::jsonb)`, appID, user.ID); err != nil {
		t.Fatal(err)
	}
	var deviceID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.devices(user_id,device_key,platform,app_version,last_seen_at)
		VALUES($1,$2,'android','1.0.0',now()) RETURNING id`, user.ID, "push-pipeline-"+suffix).Scan(&deviceID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO notify.push_provider_configs(
		tenant_id,app_id,environment,provider,public_config,status,last_preflight_at,last_preflight_status)
		VALUES($1,$2,'development','fcm','{}'::jsonb,'active',now(),'ready')`, tenant.ID, appID); err != nil {
		t.Fatal(err)
	}
	sealer, err := settingsrepo.NewAESGCMSealer(bytes.Repeat([]byte{0x3c}, 32), 7)
	if err != nil {
		t.Fatal(err)
	}
	token := "mock-accepted-" + suffix
	tokenCiphertext, tokenVersion, err := sealer.Seal([]byte(token), "push-token:"+appID.String()+":fcm")
	if err != nil {
		t.Fatal(err)
	}
	tokenHash := sha256.Sum256([]byte(token))
	var pushDeviceID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO notify.push_devices(
		user_id,device_id,provider,token_hash,token_ciphertext,key_version,status,app_id,tenant_id,platform,build_variant,locale,sdk_version,app_version)
		VALUES($1,$2,'fcm',$3,$4,$5,'active',$6,$7,'android','android_google','en-US','integration','1.0.0') RETURNING id`,
		user.ID, deviceID, tokenHash[:], tokenCiphertext, tokenVersion, appID, tenant.ID).Scan(&pushDeviceID); err != nil {
		t.Fatal(err)
	}

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{SkipUnknownJobCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	queue := jobqueue.NewRiverAdapter(pool, riverClient, jobdefs.Registry())
	notificationService := platformnotify.NewPostgresService(pool, queue)
	now := time.Now().UTC().Truncate(time.Second)
	submission, err := notificationService.Submit(ctx, platformnotify.Scope{
		TenantID: tenant.ID, AppID: appID, ActorKind: "user", ActorID: user.ID, RequestID: "push-pipeline-integration", SourceIP: "127.0.0.1",
	}, platformnotify.SubmitCommand{
		IdempotencyKey: "push-pipeline-" + suffix, Source: "account.security", BusinessEventID: "security:" + suffix,
		Category: "service_security", Audience: platformnotify.Audience{Type: "users", UserIDs: []uuid.UUID{user.ID}}, Push: true,
		Content: platformnotify.Content{Type: "inline", Inline: &platformnotify.LocalizedContent{
			Title: map[string]string{"zh-CN": "安全提醒", "en-US": "Security notice"},
			Body:  map[string]string{"zh-CN": "安全设置已更新。", "en-US": "Security settings were updated."},
		}}, TTLSeconds: 3600, RouteKey: "notification_detail", ResourceID: "security-" + suffix,
	})
	if err != nil {
		t.Fatal(err)
	}

	publishJobID := trackedRiverJobID(t, ctx, pool, tenant.ID, appID, submission.RunID, notify.MessagePublishJobKind)
	publishArgs := notify.MessagePublishJobArgs{TenantID: tenant.ID, AppID: appID, MessageID: submission.MessageID}
	if err = NewMessagePublishWorker(pool, queue).Work(ctx, integrationJob(publishJobID, publishArgs)); err != nil {
		t.Fatal(err)
	}
	fanoutJobID := trackedRiverJobID(t, ctx, pool, tenant.ID, appID, submission.RunID, notify.PushFanoutJobKind)
	fanoutArgs := notify.PushFanoutJobArgs{TenantID: tenant.ID, AppID: appID, MessageID: submission.MessageID}
	if err = NewPushFanoutWorker(pool, queue, sealer, "development", true).Work(ctx, integrationJob(fanoutJobID, fanoutArgs)); err != nil {
		t.Fatal(err)
	}

	var deliveryID uuid.UUID
	var deliveryTaskRunID uuid.UUID
	var persistedTarget []byte
	if err = pool.QueryRow(ctx, `SELECT id,task_run_id,target_ciphertext FROM notify.deliveries
		WHERE tenant_id=$1 AND app_id=$2 AND message_id=$3 AND push_device_id=$4`, tenant.ID, appID, submission.MessageID, pushDeviceID).
		Scan(&deliveryID, &deliveryTaskRunID, &persistedTarget); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(persistedTarget, []byte(token)) {
		t.Fatal("push token was persisted in plaintext")
	}
	var deliveryJobID int64
	if err = pool.QueryRow(ctx, `SELECT river_job_id FROM jobs.task_runs WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND task_kind=$4`,
		tenant.ID, appID, deliveryTaskRunID, notify.DeliveryJobKind).Scan(&deliveryJobID); err != nil {
		t.Fatal(err)
	}
	deliveryArgs := notify.DeliveryJobArgs{DeliveryID: deliveryID}
	if err = NewDeliveryWorker(pool, sealer, "development", true, pushprovider.NewMockSender()).Work(ctx, integrationJob(deliveryJobID, deliveryArgs)); err != nil {
		t.Fatal(err)
	}

	var deliveryStatus, providerResult, providerMessageID string
	if err = pool.QueryRow(ctx, `SELECT status,provider_result,provider_message_id FROM notify.deliveries WHERE id=$1`, deliveryID).
		Scan(&deliveryStatus, &providerResult, &providerMessageID); err != nil {
		t.Fatal(err)
	}
	if deliveryStatus != "sent" || providerResult != "accepted" || providerMessageID != "mock-"+deliveryID.String() {
		t.Fatalf("delivery status=%s result=%s provider_message_id=%s", deliveryStatus, providerResult, providerMessageID)
	}
	pushRepository := pushrepo.NewPostgres(pool, queue)
	pushPrincipal := push.Principal{TenantID: tenant.ID, AppID: appID, UserID: user.ID, DeviceID: deviceID}
	if err = pushRepository.MarkOpened(ctx, pushPrincipal, deliveryID); err != nil {
		t.Fatal(err)
	}
	if err = pushRepository.MarkOpened(ctx, pushPrincipal, deliveryID); err != nil {
		t.Fatalf("opened must be idempotent: %v", err)
	}

	status, err := notificationService.Status(ctx, platformnotify.Scope{
		TenantID: tenant.ID, AppID: appID, ActorKind: "user", ActorID: user.ID, RequestID: "push-pipeline-status",
	}, submission.MessageID)
	if err != nil || status.Status != "completed" || status.AcceptedCount != 1 || status.OpenedCount != 1 || status.DeliveryCount != 1 {
		t.Fatalf("submission status=%#v err=%v", status, err)
	}
	operationsRepository := notifyrepo.NewPostgres(pool, queue)
	summary, err := operationsRepository.OperationsSummary(ctx, tenant.ID, appID, notify.OperationsFilter{
		From: now.Add(-time.Hour), To: now.Add(time.Hour), Environment: "development", Channel: "push", Provider: "fcm",
	})
	if err != nil || summary.Accepted != 1 || summary.Opened != 1 || summary.OpenRate != 1 {
		t.Fatalf("operations summary=%#v err=%v", summary, err)
	}
}

func trackedRiverJobID(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, appID, correlationID uuid.UUID, kind string) int64 {
	t.Helper()
	var riverJobID int64
	if err := pool.QueryRow(ctx, `SELECT river_job_id FROM jobs.task_runs
		WHERE tenant_id=$1 AND app_id=$2 AND correlation_id=$3 AND task_kind=$4 ORDER BY created_at DESC LIMIT 1`,
		tenantID, appID, correlationID, kind).Scan(&riverJobID); err != nil {
		t.Fatal(err)
	}
	return riverJobID
}

func integrationJob[T river.JobArgs](id int64, args T) *river.Job[T] {
	return &river.Job[T]{JobRow: &rivertype.JobRow{ID: id, Attempt: 1, MaxAttempts: 5}, Args: args}
}
