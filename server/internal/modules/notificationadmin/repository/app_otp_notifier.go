package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	app "github.com/appkernia/appkernia/server/internal/modules/appmanagement/application"
	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AppOTPNotifier writes its delivery and River job into the caller's challenge
// transaction.  The OTP is only present in payload_ciphertext; rendered email
// fields remain empty until the worker decrypts and renders the template.
type AppOTPNotifier struct {
	queue  jobqueue.Enqueuer
	sealer notify.TargetSealer
}

// Queue implements loginprovider/repository.OTPDispatcher for App-scoped
// email and SMS login, binding and step-up challenges. Plain targets and OTPs
// only exist in memory and encrypted delivery fields.
func (n *AppOTPNotifier) Queue(ctx context.Context, tx pgx.Tx, input login.OTPChallenge) error {
	if tx == nil || input.AppID == uuid.Nil || input.TenantID == uuid.Nil || input.ID == uuid.Nil ||
		(input.IdentifierType != "email" && input.IdentifierType != "mobile") || strings.TrimSpace(input.NormalizedValue) == "" || len(input.Code) != 6 {
		return errors.New("login OTP notification is invalid")
	}
	locale := input.Locale
	if locale != "en-US" {
		locale = "zh-CN"
	}
	channel, provider := "email", "smtp"
	if input.IdentifierType == "mobile" {
		channel = "sms"
		var enabledRaw, providerRaw json.RawMessage
		if err := tx.QueryRow(ctx, `SELECT
COALESCE((SELECT COALESCE(value_json,default_value_json) FROM sys.config_items WHERE tenant_id=$1 AND module_code='notifications' AND config_group='sms' AND config_key='sms.enabled' AND status='active' LIMIT 1),'false'::jsonb),
COALESCE((SELECT COALESCE(value_json,default_value_json) FROM sys.config_items WHERE tenant_id=$1 AND module_code='notifications' AND config_group='sms' AND config_key='sms.provider' AND status='active' LIMIT 1),'"none"'::jsonb)`, input.TenantID).Scan(&enabledRaw, &providerRaw); err != nil {
			return errors.New("SMS delivery configuration is unavailable")
		}
		var enabled bool
		if json.Unmarshal(enabledRaw, &enabled) != nil || json.Unmarshal(providerRaw, &provider) != nil || !enabled || (provider != "aliyun" && provider != "tencent") {
			return errors.New("SMS delivery configuration is unavailable")
		}
	}
	var templateID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM notify.templates
WHERE code='verification_code' AND channel=$1 AND status='active' AND (tenant_id=$2 OR tenant_id IS NULL)
ORDER BY CASE WHEN locale=$3 THEN 0 WHEN locale IS NULL THEN 1 WHEN locale='zh-CN' THEN 2 ELSE 3 END,
CASE WHEN tenant_id=$2 THEN 0 ELSE 1 END,id LIMIT 1`, channel, input.TenantID, locale).Scan(&templateID); err != nil {
		return errors.New("verification code template is unavailable")
	}
	target := strings.TrimSpace(input.NormalizedValue)
	if channel == "email" {
		target = strings.ToLower(target)
	}
	targetCiphertext, targetVersion, err := n.sealer.Seal([]byte(target), input.TenantID.String())
	if err != nil {
		return errors.New("login OTP target encryption failed")
	}
	payload, _ := json.Marshal(map[string]any{
		"code": input.Code, "expires_minutes": 10, "app_id": input.AppID.String(), "purpose": input.Purpose,
	})
	payloadCiphertext, payloadVersion, err := n.sealer.Seal(payload, input.TenantID.String()+":notification-payload")
	if err != nil {
		return errors.New("login OTP payload encryption failed")
	}
	targetHash := sha256.Sum256([]byte(target))
	hint := input.DisplayHint
	dedupe := "login-otp:" + input.ID.String()
	var deliveryID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO notify.deliveries(
app_id,tenant_id,template_id,channel,target_ciphertext,target_hash,target_hint,target_key_version,provider,
rendered_subject,rendered_body,dedupe_key,payload_ciphertext,payload_key_version,status,max_attempts)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'','',$10,$11,$12,'pending',5) RETURNING id`,
		input.AppID, input.TenantID, templateID, channel, targetCiphertext, targetHash[:], hint, targetVersion,
		provider, dedupe, payloadCiphertext, payloadVersion).Scan(&deliveryID)
	if err != nil {
		return fmt.Errorf("create login OTP delivery: %w", err)
	}
	appID := input.AppID
	run, err := n.queue.EnqueueTx(ctx, tx, jobqueue.Spec{
		Scope: jobqueue.Scope{TenantID: input.TenantID, AppID: &appID, ModuleCode: "notify", ResourceType: "notification_delivery", ResourceID: &deliveryID},
		Args:  notify.DeliveryJobArgs{DeliveryID: deliveryID}, Queue: "notifications", MaxAttempts: 5, UniqueByArgs: true,
	})
	if err != nil {
		return errors.New("enqueue login OTP delivery failed")
	}
	if _, err = tx.Exec(ctx, `UPDATE notify.deliveries SET task_run_id=$2 WHERE id=$1`, deliveryID, run.ID); err != nil {
		return errors.New("record login OTP task failed")
	}
	return nil
}

func NewAppOTPNotifier(queue jobqueue.Enqueuer, sealer notify.TargetSealer) (*AppOTPNotifier, error) {
	if queue == nil || sealer == nil {
		return nil, errors.New("app OTP notifier dependencies are incomplete")
	}
	return &AppOTPNotifier{queue: queue, sealer: sealer}, nil
}

func (n *AppOTPNotifier) QueueOTP(ctx context.Context, tx pgx.Tx, input app.OTPNotification) error {
	if tx == nil || input.AppID == uuid.Nil || input.TenantID == uuid.Nil || strings.TrimSpace(input.Email) == "" || len(input.Code) != 6 || (input.Purpose != "email_otp" && input.Purpose != "password_reset" && input.Purpose != "account_delete") {
		return errors.New("app OTP notification is invalid")
	}
	locale := input.Locale
	if locale != "en-US" {
		locale = "zh-CN"
	}
	var templateID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM notify.templates WHERE code='verification_code' AND channel='email' AND status='active' AND (tenant_id=$1 OR tenant_id IS NULL) ORDER BY CASE WHEN locale=$2 THEN 0 WHEN locale IS NULL THEN 1 WHEN locale='zh-CN' THEN 2 ELSE 3 END, CASE WHEN tenant_id=$1 THEN 0 ELSE 1 END, id LIMIT 1`, input.TenantID, locale).Scan(&templateID)
	if err != nil {
		return fmt.Errorf("verification code template is unavailable: %w", err)
	}
	target := strings.ToLower(strings.TrimSpace(input.Email))
	targetCiphertext, targetVersion, err := n.sealer.Seal([]byte(target), input.TenantID.String())
	if err != nil {
		return errors.New("app OTP target encryption failed")
	}
	payload, err := json.Marshal(map[string]any{"code": input.Code, "expires_minutes": input.ExpiresMinutes, "app_id": input.AppID.String(), "purpose": input.Purpose})
	if err != nil {
		return err
	}
	payloadCiphertext, payloadVersion, err := n.sealer.Seal(payload, input.TenantID.String()+":notification-payload")
	if err != nil {
		return errors.New("app OTP payload encryption failed")
	}
	targetHash, codeHash := sha256.Sum256([]byte(target)), sha256.Sum256([]byte(input.Code))
	dedupe := "app-otp:" + input.AppID.String() + ":" + input.Purpose + ":" + hex.EncodeToString(codeHash[:])
	var deliveryID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO notify.deliveries(app_id,tenant_id,template_id,channel,target_ciphertext,target_hash,target_hint,target_key_version,provider,rendered_subject,rendered_body,dedupe_key,payload_ciphertext,payload_key_version,status,max_attempts)
	VALUES($1,$2,$3,'email',$4,$5,$6,$7,'smtp','','',$8,$9,$10,'pending',5) RETURNING id`, input.AppID, input.TenantID, templateID, targetCiphertext, targetHash[:], emailHint(target), targetVersion, dedupe, payloadCiphertext, payloadVersion).Scan(&deliveryID)
	if err != nil {
		return fmt.Errorf("create app OTP delivery: %w", err)
	}
	appID := input.AppID
	run, enqueueErr := n.queue.EnqueueTx(ctx, tx, jobqueue.Spec{
		Scope: jobqueue.Scope{TenantID: input.TenantID, AppID: &appID, ModuleCode: "notify", ResourceType: "notification_delivery", ResourceID: &deliveryID},
		Args:  notify.DeliveryJobArgs{DeliveryID: deliveryID}, Queue: "notifications", MaxAttempts: 5, UniqueByArgs: true,
	})
	if enqueueErr != nil {
		return fmt.Errorf("enqueue app OTP delivery: %w", enqueueErr)
	}
	if _, err = tx.Exec(ctx, `UPDATE notify.deliveries SET task_run_id=$2 WHERE id=$1`, deliveryID, run.ID); err != nil {
		return fmt.Errorf("enqueue app OTP delivery: %w", err)
	}
	return nil
}
