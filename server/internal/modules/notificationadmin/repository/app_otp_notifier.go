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
