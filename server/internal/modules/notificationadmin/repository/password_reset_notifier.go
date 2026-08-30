package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	stdhtml "html"
	"net/url"
	"regexp"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PasswordResetNotifier struct {
	pool   *pgxpool.Pool
	queue  jobqueue.Enqueuer
	sealer notify.TargetSealer
	origin *url.URL
}

func NewPasswordResetNotifier(pool *pgxpool.Pool, queue jobqueue.Enqueuer, sealer notify.TargetSealer, adminOrigin string) (*PasswordResetNotifier, error) {
	origin, err := url.Parse(strings.TrimSpace(adminOrigin))
	if err != nil || origin.Scheme == "" || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || (origin.Scheme != "https" && origin.Hostname() != "localhost") {
		return nil, errors.New("password reset notifier requires a secure admin origin")
	}
	if pool == nil || queue == nil || sealer == nil {
		return nil, errors.New("password reset notifier dependencies are incomplete")
	}
	return &PasswordResetNotifier{pool: pool, queue: queue, sealer: sealer, origin: origin}, nil
}

var resetPlaceholder = regexp.MustCompile(`\{\{\s*([a-zA-Z][a-zA-Z0-9_.-]*)\s*\}\}`)

func (n *PasswordResetNotifier) SendPasswordReset(ctx context.Context, input iamapp.PasswordResetNotification) error {
	if input.TenantID == uuid.Nil || strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Token) == "" {
		return errors.New("password reset notification is invalid")
	}
	template, err := n.passwordResetTemplate(ctx, input.TenantID, input.Locale)
	if err != nil {
		return err
	}
	resetURL := *n.origin
	resetURL.Path = strings.TrimRight(resetURL.Path, "/") + "/reset-password"
	query := resetURL.Query()
	query.Set("token", input.Token)
	resetURL.RawQuery = query.Encode()
	variables := map[string]string{"reset_url": resetURL.String(), "expires_minutes": "15"}
	subject, err := renderResetTemplate(template.subject, variables, false)
	if err != nil {
		return err
	}
	body, err := renderResetTemplate(template.body, variables, template.bodyFormat == "html")
	if err != nil {
		return err
	}
	target := strings.ToLower(strings.TrimSpace(input.Email))
	targetCiphertext, targetVersion, err := n.sealer.Seal([]byte(target), input.TenantID.String())
	if err != nil {
		return errors.New("password reset target encryption failed")
	}
	payload, _ := json.Marshal(variables)
	payloadCiphertext, payloadVersion, err := n.sealer.Seal(payload, input.TenantID.String()+":notification-payload")
	if err != nil {
		return errors.New("password reset payload encryption failed")
	}
	targetHash := sha256.Sum256([]byte(target))
	tokenHash := sha256.Sum256([]byte(input.Token))
	dedupeKey := "password-reset:" + hex.EncodeToString(tokenHash[:])
	tx, err := n.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var deliveryID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO notify.deliveries(
		tenant_id,template_id,channel,target_ciphertext,target_hash,target_hint,target_key_version,provider,rendered_subject,rendered_body,dedupe_key,payload_ciphertext,payload_key_version,status,max_attempts)
		VALUES($1,$2,'email',$3,$4,$5,$6,'smtp',$7,$8,$9,$10,$11,'pending',5)
		ON CONFLICT(tenant_id,channel,dedupe_key) WHERE dedupe_key IS NOT NULL DO UPDATE SET updated_at=notify.deliveries.updated_at
		RETURNING id`, input.TenantID, template.id, targetCiphertext, targetHash[:], emailHint(target), targetVersion, subject, body, dedupeKey, payloadCiphertext, payloadVersion).Scan(&deliveryID)
	if err != nil {
		return fmt.Errorf("create password reset delivery: %w", err)
	}
	run, enqueueErr := n.queue.EnqueueTx(ctx, tx, jobqueue.Spec{
		Scope: jobqueue.Scope{TenantID: input.TenantID, ModuleCode: "notify", ResourceType: "notification_delivery", ResourceID: &deliveryID},
		Args:  notify.DeliveryJobArgs{DeliveryID: deliveryID}, Queue: "notifications", MaxAttempts: 5, UniqueByArgs: true,
	})
	if enqueueErr != nil {
		return fmt.Errorf("enqueue password reset delivery: %w", enqueueErr)
	}
	if _, err = tx.Exec(ctx, `UPDATE notify.deliveries SET task_run_id=$2 WHERE id=$1`, deliveryID, run.ID); err != nil {
		return fmt.Errorf("record password reset task: %w", err)
	}
	return tx.Commit(ctx)
}

type resetTemplate struct {
	id         uuid.UUID
	subject    string
	body       string
	bodyFormat string
}

func (n *PasswordResetNotifier) passwordResetTemplate(ctx context.Context, tenantID uuid.UUID, locale string) (resetTemplate, error) {
	if locale != "en-US" {
		locale = "zh-CN"
	}
	var out resetTemplate
	err := n.pool.QueryRow(ctx, `SELECT id,COALESCE(subject_template,''),body_template,body_format
		FROM notify.templates
		WHERE code='password_reset' AND channel='email' AND status='active' AND (tenant_id=$1 OR tenant_id IS NULL)
		ORDER BY CASE WHEN locale=$2 THEN 0 WHEN locale IS NULL THEN 1 WHEN locale='zh-CN' THEN 2 ELSE 3 END,
		         CASE WHEN tenant_id=$1 THEN 0 ELSE 1 END, id
		LIMIT 1`, tenantID, locale).Scan(&out.id, &out.subject, &out.body, &out.bodyFormat)
	if errors.Is(err, pgx.ErrNoRows) {
		return resetTemplate{}, errors.New("password reset email template is unavailable")
	}
	return out, err
}

func renderResetTemplate(raw string, variables map[string]string, escape bool) (string, error) {
	missing := false
	out := resetPlaceholder.ReplaceAllStringFunc(raw, func(match string) string {
		parts := resetPlaceholder.FindStringSubmatch(match)
		value, ok := variables[parts[1]]
		if !ok || value == "" {
			missing = true
			return ""
		}
		if escape {
			return stdhtml.EscapeString(value)
		}
		return value
	})
	if missing {
		return "", errors.New("password reset template variable is missing")
	}
	return out, nil
}
