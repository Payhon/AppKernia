package worker

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/appkernia/appkernia/server/internal/modules/notificationadmin/jobdefs"
	push "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type PayloadSealer interface {
	Seal([]byte, string) ([]byte, int32, error)
}

type MessagePublishWorker struct {
	river.WorkerDefaults[notify.MessagePublishJobArgs]
	pool  *pgxpool.Pool
	queue jobqueue.Enqueuer
}

func NewMessagePublishWorker(pool *pgxpool.Pool, queue jobqueue.Enqueuer) *MessagePublishWorker {
	return &MessagePublishWorker{pool: pool, queue: queue}
}

func (w *MessagePublishWorker) Timeout(*river.Job[notify.MessagePublishJobArgs]) time.Duration {
	return jobdefs.PublishTimeout
}

func (w *MessagePublishWorker) Work(ctx context.Context, job *river.Job[notify.MessagePublishJobArgs]) (workErr error) {
	if err := jobqueue.StartAttempt(ctx, w.pool, job.ID, job.Attempt); err != nil {
		return err
	}
	defer func() {
		status := "succeeded"
		summary := ""
		if workErr != nil {
			status, summary = "retry_wait", workErr.Error()
			if job.Attempt >= job.MaxAttempts {
				status = "failed"
			}
		}
		_ = jobqueue.FinishAttempt(ctx, w.pool, job.ID, job.Attempt, jobqueue.Completion{Status: status, ErrorSummary: summary})
	}()
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	var messageRunID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO notify.message_runs(tenant_id,app_id,message_id,trigger_type,status,recipient_count,started_at)
		SELECT $1,$2,$3,'internal','running',count(*),now() FROM notify.recipients WHERE tenant_id=$1 AND app_id=$2 AND message_id=$3
		ON CONFLICT(tenant_id,app_id,message_id) DO UPDATE SET status='running',started_at=COALESCE(notify.message_runs.started_at,now())
		RETURNING id`, job.Args.TenantID, job.Args.AppID, job.Args.MessageID).Scan(&messageRunID)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	var scheduledAt, expiresAt *time.Time
	err = tx.QueryRow(ctx, `SELECT status,scheduled_at,expires_at FROM notify.messages
WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND deleted_at IS NULL FOR UPDATE`, job.Args.TenantID, job.Args.AppID, job.Args.MessageID).Scan(&status, &scheduledAt, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) || status != "scheduled" {
		return nil
	}
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if scheduledAt != nil && scheduledAt.After(now) {
		return river.JobSnooze(scheduledAt.Sub(now))
	}
	if expiresAt != nil && !expiresAt.After(now) {
		if _, err = tx.Exec(ctx, `UPDATE notify.messages SET status='cancelled' WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND status='scheduled'`, job.Args.TenantID, job.Args.AppID, job.Args.MessageID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE notify.message_runs SET status='expired',completed_at=now() WHERE tenant_id=$1 AND app_id=$2 AND message_id=$3`, job.Args.TenantID, job.Args.AppID, job.Args.MessageID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if _, err = tx.Exec(ctx, `UPDATE notify.messages SET status='published',published_at=now() WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND status='scheduled'`, job.Args.TenantID, job.Args.AppID, job.Args.MessageID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE notify.recipients SET delivery_status='delivered',delivered_at=COALESCE(delivered_at,now()) WHERE tenant_id=$1 AND app_id=$2 AND message_id=$3 AND delivery_status='pending'`, job.Args.TenantID, job.Args.AppID, job.Args.MessageID); err != nil {
		return err
	}
	appID, messageID := job.Args.AppID, job.Args.MessageID
	_, err = w.queue.EnqueueTx(ctx, tx, jobqueue.Spec{
		Scope: jobqueue.Scope{TenantID: job.Args.TenantID, AppID: &appID, ModuleCode: "notify", ResourceType: "notification_message", ResourceID: &messageID, CorrelationID: &messageRunID},
		Args:  notify.PushFanoutJobArgs{TenantID: job.Args.TenantID, AppID: job.Args.AppID, MessageID: job.Args.MessageID}, Queue: "notifications", MaxAttempts: 5, UniqueByArgs: true,
	})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE notify.message_runs SET status='queued',started_at=COALESCE(started_at,now()) WHERE id=$1`, messageRunID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type PushFanoutWorker struct {
	river.WorkerDefaults[notify.PushFanoutJobArgs]
	pool        *pgxpool.Pool
	queue       jobqueue.Enqueuer
	sealer      PayloadSealer
	environment string
	pushEnabled bool
}

func NewPushFanoutWorker(pool *pgxpool.Pool, queue jobqueue.Enqueuer, sealer PayloadSealer, environment string, pushEnabled bool) *PushFanoutWorker {
	return &PushFanoutWorker{pool: pool, queue: queue, sealer: sealer, environment: normalizePushEnvironment(environment), pushEnabled: pushEnabled}
}

func (w *PushFanoutWorker) Timeout(*river.Job[notify.PushFanoutJobArgs]) time.Duration {
	return jobdefs.FanoutTimeout
}

type fanoutMessage struct {
	Title, Body, BodyFormat, Category, CollapseKey, RouteKey string
	TTLSeconds                                               int
	RouteParams                                              map[string]string
	LocalizedTitles, LocalizedBodies                         map[string]string
	PushEnabled                                              bool
}

type fanoutDevice struct {
	ID                         uuid.UUID
	UserID                     uuid.UUID
	Provider, Locale           string
	TokenCiphertext, TokenHash []byte
	TokenKeyVersion            int32
}

func (w *PushFanoutWorker) Work(ctx context.Context, job *river.Job[notify.PushFanoutJobArgs]) (workErr error) {
	if err := jobqueue.StartAttempt(ctx, w.pool, job.ID, job.Attempt); err != nil {
		return err
	}
	defer func() {
		status := "succeeded"
		summary := ""
		if workErr != nil {
			status, summary = "retry_wait", workErr.Error()
			if job.Attempt >= job.MaxAttempts {
				status = "failed"
			}
		}
		_ = jobqueue.FinishAttempt(ctx, w.pool, job.ID, job.Attempt, jobqueue.Completion{Status: status, ErrorSummary: summary})
	}()
	// Keep the worker registered so already-scheduled jobs remain safe to consume,
	// while the global kill switch prevents both token reads and new deliveries.
	if !w.pushEnabled {
		_, err := w.pool.Exec(ctx, `WITH skipped AS (
			UPDATE notify.recipients SET push_skip_reason='provider_unavailable',push_evaluated_at=now(),push_environment=$4
			WHERE tenant_id=$1 AND app_id=$2 AND message_id=$3 RETURNING 1)
			UPDATE notify.message_runs SET status='completed_with_failures',evaluated_count=recipient_count,
			skipped_count=(SELECT count(*) FROM skipped),completed_at=now()
			WHERE tenant_id=$1 AND app_id=$2 AND message_id=$3`, job.Args.TenantID, job.Args.AppID, job.Args.MessageID, w.environment)
		return err
	}
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var messageRunID uuid.UUID
	if err = tx.QueryRow(ctx, `INSERT INTO notify.message_runs(tenant_id,app_id,message_id,trigger_type,status,started_at)
		VALUES($1,$2,$3,'internal','running',now())
		ON CONFLICT(tenant_id,app_id,message_id) DO UPDATE SET status='running',started_at=COALESCE(notify.message_runs.started_at,now())
		RETURNING id`, job.Args.TenantID, job.Args.AppID, job.Args.MessageID).Scan(&messageRunID); err != nil {
		return err
	}
	var message fanoutMessage
	var routeParams, localizedTitles, localizedBodies []byte
	var expiresAt *time.Time
	err = tx.QueryRow(ctx, `SELECT title,body,body_format,push_category,push_ttl_seconds,
COALESCE(push_collapse_key,''),COALESCE(push_route_key,''),push_route_params,expires_at
 ,COALESCE((metadata->>'push_enabled')::boolean,true),COALESCE(metadata->'localized_titles','{}'::jsonb),COALESCE(metadata->'localized_bodies','{}'::jsonb)
FROM notify.messages WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND status='published' AND deleted_at IS NULL`,
		job.Args.TenantID, job.Args.AppID, job.Args.MessageID).Scan(&message.Title, &message.Body, &message.BodyFormat, &message.Category,
		&message.TTLSeconds, &message.CollapseKey, &message.RouteKey, &routeParams, &expiresAt, &message.PushEnabled, &localizedTitles, &localizedBodies)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if expiresAt != nil && !expiresAt.After(time.Now().UTC()) {
		if _, err = tx.Exec(ctx, `UPDATE notify.recipients SET push_skip_reason='message_expired',push_evaluated_at=now(),push_environment=$4
WHERE tenant_id=$1 AND app_id=$2 AND message_id=$3`, job.Args.TenantID, job.Args.AppID, job.Args.MessageID, w.environment); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	_ = json.Unmarshal(routeParams, &message.RouteParams)
	_ = json.Unmarshal(localizedTitles, &message.LocalizedTitles)
	_ = json.Unmarshal(localizedBodies, &message.LocalizedBodies)
	if message.RouteParams == nil {
		message.RouteParams = map[string]string{}
	}
	if !message.PushEnabled {
		if _, err = tx.Exec(ctx, `UPDATE notify.message_runs SET status='completed',evaluated_count=recipient_count,completed_at=now()
			WHERE id=$1`, messageRunID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if _, err = tx.Exec(ctx, `UPDATE notify.recipients r SET push_skip_reason=CASE
  WHEN NOT EXISTS (SELECT 1 FROM iam.tenant_members tm JOIN app.user_memberships am ON am.tenant_id=tm.tenant_id AND am.user_id=tm.user_id
                   WHERE tm.tenant_id=r.tenant_id AND tm.user_id=r.user_id AND tm.status='active' AND am.app_id=r.app_id AND am.status='active') THEN 'membership_inactive'
  WHEN NOT COALESCE((SELECT (pref.notification_preferences->>'push')::boolean FROM iam.user_preferences pref WHERE pref.app_id=r.app_id AND pref.user_id=r.user_id),false) THEN 'push_disabled'
  WHEN $4='news_operations' AND NOT COALESCE((SELECT (pref.notification_preferences->>'push_operations')::boolean FROM iam.user_preferences pref WHERE pref.app_id=r.app_id AND pref.user_id=r.user_id),false) THEN 'category_disabled'
  WHEN $4='service_security' AND NOT COALESCE((SELECT (pref.notification_preferences->>'push_service')::boolean FROM iam.user_preferences pref WHERE pref.app_id=r.app_id AND pref.user_id=r.user_id),true) THEN 'category_disabled'
  WHEN NOT EXISTS (SELECT 1 FROM notify.push_devices d WHERE d.tenant_id=r.tenant_id AND d.app_id=r.app_id AND d.user_id=r.user_id AND d.status='active') THEN 'no_active_device'
  WHEN NOT EXISTS (SELECT 1 FROM notify.push_devices d JOIN notify.push_provider_configs c ON c.tenant_id=d.tenant_id AND c.app_id=d.app_id AND c.provider=d.provider
                   WHERE d.tenant_id=r.tenant_id AND d.app_id=r.app_id AND d.user_id=r.user_id AND d.status='active' AND c.environment=$5 AND c.status='active' AND c.last_preflight_status='ready') THEN 'provider_unavailable'
  ELSE NULL END,
  push_evaluated_at=now(),push_environment=$5
WHERE r.tenant_id=$1 AND r.app_id=$2 AND r.message_id=$3`,
		job.Args.TenantID, job.Args.AppID, job.Args.MessageID, message.Category, w.environment); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT d.id,r.user_id,d.provider,d.locale,d.token_ciphertext,d.token_hash,d.key_version
FROM notify.recipients r
JOIN iam.tenant_members tm ON tm.tenant_id=r.tenant_id AND tm.user_id=r.user_id AND tm.status='active'
JOIN app.user_memberships am ON am.tenant_id=r.tenant_id AND am.app_id=r.app_id AND am.user_id=r.user_id AND am.status='active'
JOIN notify.push_devices d ON d.tenant_id=r.tenant_id AND d.app_id=r.app_id AND d.user_id=r.user_id AND d.status='active'
LEFT JOIN iam.user_preferences pref ON pref.app_id=r.app_id AND pref.user_id=r.user_id
WHERE r.tenant_id=$1 AND r.app_id=$2 AND r.message_id=$3 AND r.delivery_status='delivered' AND d.id>$4
  AND COALESCE((pref.notification_preferences->>'push')::boolean,false)
  AND CASE WHEN $5='news_operations'
      THEN COALESCE((pref.notification_preferences->>'push_operations')::boolean,false)
      ELSE COALESCE((pref.notification_preferences->>'push_service')::boolean,true) END
  AND EXISTS (SELECT 1 FROM notify.push_provider_configs c WHERE c.tenant_id=r.tenant_id AND c.app_id=r.app_id AND c.provider=d.provider AND c.environment=$6 AND c.status='active' AND c.last_preflight_status='ready')
ORDER BY d.id LIMIT 500`, job.Args.TenantID, job.Args.AppID, job.Args.MessageID, job.Args.AfterDeviceID, message.Category, w.environment)
	if err != nil {
		return err
	}
	devices := make([]fanoutDevice, 0, 500)
	for rows.Next() {
		var device fanoutDevice
		if err = rows.Scan(&device.ID, &device.UserID, &device.Provider, &device.Locale, &device.TokenCiphertext, &device.TokenHash, &device.TokenKeyVersion); err != nil {
			rows.Close()
			return err
		}
		devices = append(devices, device)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, device := range devices {
		deliveryID, idErr := uuid.NewV7()
		if idErr != nil {
			return idErr
		}
		title, body := displayText(message, device.Locale)
		payload := push.SendPayload{SchemaVersion: 1, DeliveryID: deliveryID, MessageID: job.Args.MessageID, Title: title, Body: body,
			Category: message.Category, TTLSeconds: message.TTLSeconds, CollapseKey: message.CollapseKey, RouteKey: message.RouteKey, RouteParams: message.RouteParams}
		plain, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return marshalErr
		}
		ciphertext, keyVersion, sealErr := w.sealer.Seal(plain, "push-payload:"+job.Args.AppID.String()+":"+device.ID.String())
		if sealErr != nil {
			return sealErr
		}
		var inserted uuid.UUID
		err = tx.QueryRow(ctx, `INSERT INTO notify.deliveries(
id,tenant_id,app_id,message_id,user_id,channel,target_ciphertext,target_hash,target_hint,target_key_version,provider,
dedupe_key,payload_ciphertext,payload_key_version,push_device_id,status,max_attempts,metadata,message_run_id,delivery_environment)
VALUES($1,$2,$3,$4,$5,'push',$6,$7,$8,$9,$10,$11,$12,$13,$14,'pending',5,jsonb_build_object('push_category',$15::text),$16,$17)
ON CONFLICT(tenant_id,message_id,user_id,push_device_id) WHERE channel='push' AND message_id IS NOT NULL AND user_id IS NOT NULL AND push_device_id IS NOT NULL
DO NOTHING RETURNING id`, deliveryID, job.Args.TenantID, job.Args.AppID, job.Args.MessageID, device.UserID, device.TokenCiphertext,
			device.TokenHash, "provider:"+device.Provider, device.TokenKeyVersion, device.Provider, "push:"+job.Args.MessageID.String()+":"+device.ID.String(),
			ciphertext, keyVersion, device.ID, message.Category, messageRunID, w.environment).Scan(&inserted)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		appID, deliveryIDCopy := job.Args.AppID, inserted
		queued, enqueueErr := w.queue.EnqueueTx(ctx, tx, jobqueue.Spec{
			Scope: jobqueue.Scope{TenantID: job.Args.TenantID, AppID: &appID, ModuleCode: "notify", ResourceType: "notification_delivery", ResourceID: &deliveryIDCopy, CorrelationID: &messageRunID},
			Args:  notify.DeliveryJobArgs{DeliveryID: inserted}, Queue: "notifications", MaxAttempts: 5, UniqueByArgs: true,
		})
		if enqueueErr != nil {
			return enqueueErr
		}
		if _, err = tx.Exec(ctx, `UPDATE notify.deliveries SET task_run_id=$2 WHERE id=$1`, inserted, queued.ID); err != nil {
			return err
		}
	}
	if len(devices) == 500 {
		next := notify.PushFanoutJobArgs{TenantID: job.Args.TenantID, AppID: job.Args.AppID, MessageID: job.Args.MessageID, AfterDeviceID: devices[len(devices)-1].ID}
		appID, messageID := job.Args.AppID, job.Args.MessageID
		if _, err = w.queue.EnqueueTx(ctx, tx, jobqueue.Spec{
			Scope: jobqueue.Scope{TenantID: job.Args.TenantID, AppID: &appID, ModuleCode: "notify", ResourceType: "notification_message", ResourceID: &messageID, CorrelationID: &messageRunID},
			Args:  next, Queue: "notifications", MaxAttempts: 5, UniqueByArgs: true,
		}); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE notify.message_runs run SET
		evaluated_count=(SELECT count(*) FROM notify.recipients r WHERE r.tenant_id=run.tenant_id AND r.app_id=run.app_id AND r.message_id=run.message_id AND r.push_evaluated_at IS NOT NULL),
		skipped_count=(SELECT count(*) FROM notify.recipients r WHERE r.tenant_id=run.tenant_id AND r.app_id=run.app_id AND r.message_id=run.message_id AND r.push_skip_reason IS NOT NULL),
		delivery_count=(SELECT count(*) FROM notify.deliveries d WHERE d.tenant_id=run.tenant_id AND d.app_id=run.app_id AND d.message_run_id=run.id),
		status=CASE WHEN $2::boolean THEN 'running' WHEN EXISTS(SELECT 1 FROM notify.deliveries d WHERE d.message_run_id=run.id) THEN 'running' ELSE 'completed' END,
		completed_at=CASE WHEN $2::boolean OR EXISTS(SELECT 1 FROM notify.deliveries d WHERE d.message_run_id=run.id) THEN NULL ELSE now() END
		WHERE run.id=$1`, messageRunID, len(devices) == 500); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func normalizePushEnvironment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "development", "test", "staging", "production":
		return value
	default:
		return "development"
	}
}

var markup = regexp.MustCompile(`<[^>]+>|[#*_~` + "`" + `>]+`)

func displayText(message fanoutMessage, locale string) (string, string) {
	if message.Category == push.CategoryServiceSecurity {
		if locale == "en-US" {
			return "AppKernia security notice", "Open the app to view this protected notification."
		}
		return "AppKernia 安全通知", "请打开应用查看受保护的通知内容。"
	}
	if title := strings.TrimSpace(message.LocalizedTitles[locale]); title != "" {
		body := strings.TrimSpace(message.LocalizedBodies[locale])
		runes := []rune(body)
		if len(runes) > 180 {
			body = string(runes[:180])
		}
		return title, body
	}
	body := strings.Join(strings.Fields(markup.ReplaceAllString(message.Body, " ")), " ")
	runes := []rune(body)
	if len(runes) > 180 {
		body = string(runes[:180])
	}
	return message.Title, body
}
