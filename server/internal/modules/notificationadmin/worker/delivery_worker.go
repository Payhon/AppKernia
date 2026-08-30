package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysms "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/appkernia/appkernia/server/internal/modules/notificationadmin/jobdefs"
	push "github.com/appkernia/appkernia/server/internal/modules/push/domain"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tencentsms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	mail "github.com/wneessen/go-mail"
)

type SecretOpener interface {
	Open([]byte, string) ([]byte, error)
}

type DeliveryWorker struct {
	river.WorkerDefaults[notify.DeliveryJobArgs]
	pool        *pgxpool.Pool
	secrets     SecretOpener
	environment string
	cacheMu     sync.Mutex
	configs     map[uuid.UUID]cachedConfig
	tencent     map[string]*tencentsms.Client
	aliyun      map[string]*dysms.Client
	pushSender  push.Sender
	pushEnabled bool
}

func NewDeliveryWorker(pool *pgxpool.Pool, secrets SecretOpener, environment string, pushEnabled bool, pushSenders ...push.Sender) *DeliveryWorker {
	worker := &DeliveryWorker{
		pool: pool, secrets: secrets, environment: strings.ToLower(strings.TrimSpace(environment)),
		configs: map[uuid.UUID]cachedConfig{}, tencent: map[string]*tencentsms.Client{}, aliyun: map[string]*dysms.Client{},
		pushEnabled: pushEnabled,
	}
	if len(pushSenders) > 0 {
		worker.pushSender = pushSenders[0]
	}
	return worker
}

func (w *DeliveryWorker) Timeout(*river.Job[notify.DeliveryJobArgs]) time.Duration {
	return jobdefs.DeliveryTimeout
}

type storedDelivery struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	AppID             uuid.UUID
	TemplateID        uuid.UUID
	PushDeviceID      uuid.UUID
	MessageID         uuid.UUID
	MessageRunID      uuid.UUID
	Channel           string
	Provider          string
	TargetCiphertext  []byte
	PayloadCiphertext []byte
	RenderedSubject   string
	RenderedBody      string
	AttemptCount      int32
	MaxAttempts       int32
}

type deliveryResult struct {
	messageID  string
	retryable  bool
	risk       string
	class      string
	errorCode  string
	retryAfter time.Duration
	err        error
	cancelled  bool
}

func (w *DeliveryWorker) Work(ctx context.Context, job *river.Job[notify.DeliveryJobArgs]) error {
	if err := jobqueue.StartAttempt(ctx, w.pool, job.ID, job.Attempt); err != nil {
		return err
	}
	finish := func(completion jobqueue.Completion) {
		_ = jobqueue.FinishAttempt(ctx, w.pool, job.ID, job.Attempt, completion)
	}
	delivery, err := w.claim(ctx, job.Args.DeliveryID)
	if errors.Is(err, pgx.ErrNoRows) {
		finish(jobqueue.Completion{Status: "succeeded"})
		return nil
	}
	if err != nil {
		err = fmt.Errorf("claim notification delivery: %w", err)
		finish(jobqueue.Completion{Status: retryStatus(job.Attempt, job.MaxAttempts), ErrorSummary: err.Error()})
		return err
	}
	result := w.send(ctx, delivery)
	if result.cancelled {
		_, err = w.pool.Exec(ctx, `UPDATE notify.deliveries SET status='cancelled',next_attempt_at=NULL,retryable=false,retry_risk='none',last_error='message cancelled or expired before provider write' WHERE id=$1`, delivery.ID)
		if err == nil {
			_ = w.refreshMessageRun(ctx, delivery.MessageRunID)
			finish(jobqueue.Completion{Status: "cancelled", ResultClass: "cancelled"})
		} else {
			finish(jobqueue.Completion{Status: retryStatus(job.Attempt, job.MaxAttempts), ErrorSummary: err.Error()})
		}
		return err
	}
	if result.err == nil {
		_, err = w.pool.Exec(ctx, `UPDATE notify.deliveries SET status='sent',sent_at=now(),
			accepted_at=CASE WHEN channel='push' THEN now() ELSE accepted_at END,next_attempt_at=NULL,last_error=NULL,error_code=NULL,
			retryable=false,retry_risk='none',provider_result=CASE WHEN channel='push' THEN 'accepted' ELSE provider_result END,
			provider_message_id=NULLIF($2,'') WHERE id=$1`, delivery.ID, result.messageID)
		if err == nil {
			_ = w.refreshMessageRun(ctx, delivery.MessageRunID)
			finish(jobqueue.Completion{Status: "succeeded", ResultClass: result.class, ExternalRequestID: result.messageID})
		} else {
			finish(jobqueue.Completion{Status: retryStatus(job.Attempt, job.MaxAttempts), ErrorSummary: err.Error()})
		}
		return err
	}
	summary := safeError(result.err)
	nextDelay := result.retryAfter
	if delivery.Channel == "push" {
		nextDelay = pushRetryDelay(delivery.ID, delivery.AttemptCount, result.retryAfter)
	} else if nextDelay <= 0 {
		nextDelay = 30 * time.Second
	}
	shouldRetry := result.retryable && delivery.AttemptCount < delivery.MaxAttempts
	_, updateErr := w.pool.Exec(ctx, `UPDATE notify.deliveries SET status='failed',last_error=$2,error_code=NULLIF($3,''),provider_result=NULLIF($4,''),
		retryable=$5,retry_risk=$6,next_attempt_at=CASE WHEN $5 THEN now() + $7::interval ELSE NULL END WHERE id=$1`,
		delivery.ID, summary, result.errorCode, result.class, shouldRetry, result.risk, nextDelay.String())
	if updateErr != nil {
		err = fmt.Errorf("record notification delivery failure: %w", updateErr)
		finish(jobqueue.Completion{Status: retryStatus(job.Attempt, job.MaxAttempts), ErrorSummary: err.Error()})
		return err
	}
	if delivery.Channel == "push" && result.class == "invalid_token" {
		_, _ = w.pool.Exec(ctx, `UPDATE notify.push_devices SET status='invalid',invalidated_at=now(),invalid_reason=NULLIF($2,'') WHERE id=$1 AND status='active'`, delivery.PushDeviceID, result.errorCode)
	}
	if delivery.Channel == "push" && result.class == "auth_config_error" {
		_, _ = w.pool.Exec(ctx, `UPDATE notify.push_provider_configs SET status='faulted' WHERE tenant_id=$1 AND app_id=$2 AND provider=$3 AND environment=$4 AND status='active'`, delivery.TenantID, delivery.AppID, delivery.Provider, w.environment)
	}
	_ = w.refreshMessageRun(ctx, delivery.MessageRunID)
	if shouldRetry {
		nextRetryAt := time.Now().UTC().Add(nextDelay)
		finish(jobqueue.Completion{Status: "retry_wait", ResultClass: result.class, ErrorCode: result.errorCode, ErrorSummary: summary, NextRetryAt: &nextRetryAt})
		return river.JobSnooze(nextDelay)
	}
	finish(jobqueue.Completion{Status: "failed", ResultClass: result.class, ErrorCode: result.errorCode, ErrorSummary: summary})
	return nil
}

func retryStatus(attempt, maxAttempts int) string {
	if attempt >= maxAttempts {
		return "failed"
	}
	return "retry_wait"
}

func (w *DeliveryWorker) refreshMessageRun(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return nil
	}
	_, err := w.pool.Exec(ctx, `UPDATE notify.message_runs run SET
		delivery_count=(SELECT count(*) FROM notify.deliveries d WHERE d.message_run_id=run.id),
		accepted_count=(SELECT count(*) FROM notify.deliveries d WHERE d.message_run_id=run.id AND d.provider_result='accepted'),
		failed_count=(SELECT count(*) FROM notify.deliveries d WHERE d.message_run_id=run.id AND d.status='failed' AND NOT d.retryable),
		invalid_token_count=(SELECT count(*) FROM notify.deliveries d WHERE d.message_run_id=run.id AND d.provider_result='invalid_token'),
		opened_count=(SELECT count(*) FROM notify.deliveries d WHERE d.message_run_id=run.id AND d.opened_at IS NOT NULL),
		status=CASE
			WHEN EXISTS(SELECT 1 FROM notify.deliveries d WHERE d.message_run_id=run.id AND (d.status IN ('pending','processing') OR (d.status='failed' AND d.retryable))) THEN 'running'
			WHEN EXISTS(SELECT 1 FROM notify.deliveries d WHERE d.message_run_id=run.id AND d.status='failed') THEN 'completed_with_failures'
			ELSE 'completed' END,
		completed_at=CASE
			WHEN EXISTS(SELECT 1 FROM notify.deliveries d WHERE d.message_run_id=run.id AND (d.status IN ('pending','processing') OR (d.status='failed' AND d.retryable))) THEN NULL
			ELSE now() END
		WHERE run.id=$1`, id)
	return err
}

func pushRetryDelay(deliveryID uuid.UUID, attempt int32, providerDelay time.Duration) time.Duration {
	base := 30 * time.Second
	if attempt > 1 {
		shift := min(attempt-1, 5)
		base *= time.Duration(1 << shift)
	}
	if base > 15*time.Minute {
		base = 15 * time.Minute
	}
	if providerDelay > base {
		base = providerDelay
	}
	// Stable per-delivery jitter prevents a failed provider from causing a
	// synchronized retry wave while keeping tests deterministic.
	jitterPercent := int64(deliveryID[0]^byte(attempt)) % 21
	return base + time.Duration(int64(base)*jitterPercent/100)
}

func (w *DeliveryWorker) claim(ctx context.Context, id uuid.UUID) (storedDelivery, error) {
	var out storedDelivery
	if _, err := w.pool.Exec(ctx, `UPDATE notify.deliveries
		SET status='failed',retryable=false,
		    retry_risk=CASE WHEN channel='sms' THEN 'duplicate_possible' ELSE 'manual_review' END,
		    next_attempt_at=NULL,
		    last_error=CASE WHEN channel='sms' THEN 'delivery outcome is uncertain after worker interruption' ELSE 'delivery requires review after worker interruption' END
		WHERE id=$1 AND status='processing'`, id); err != nil {
		return out, err
	}
	if _, err := w.pool.Exec(ctx, `UPDATE notify.deliveries d
		SET status='cancelled',retryable=false,retry_risk='none',next_attempt_at=NULL,last_error='message cancelled or expired before provider write'
		WHERE d.id=$1 AND d.channel='push' AND d.message_id IS NOT NULL AND d.status IN ('pending','failed') AND d.retry_risk='none'
		  AND NOT EXISTS (SELECT 1 FROM notify.messages m WHERE m.tenant_id=d.tenant_id AND m.app_id=d.app_id AND m.id=d.message_id
		                  AND m.status='published' AND m.deleted_at IS NULL AND (m.expires_at IS NULL OR m.expires_at>now()))`, id); err != nil {
		return out, err
	}
	err := w.pool.QueryRow(ctx, `UPDATE notify.deliveries SET status='processing',attempt_count=attempt_count+1,next_attempt_at=NULL
		WHERE id=$1 AND status IN ('pending','failed') AND retry_risk='none' AND attempt_count<max_attempts
		RETURNING id,tenant_id,COALESCE(app_id,'00000000-0000-0000-0000-000000000000'::uuid),
		COALESCE(template_id,'00000000-0000-0000-0000-000000000000'::uuid),COALESCE(push_device_id,'00000000-0000-0000-0000-000000000000'::uuid),COALESCE(message_id,'00000000-0000-0000-0000-000000000000'::uuid),
		COALESCE(message_run_id,'00000000-0000-0000-0000-000000000000'::uuid),
		channel,COALESCE(provider,''),target_ciphertext,COALESCE(payload_ciphertext,'\\x'::bytea),COALESCE(rendered_subject,''),COALESCE(rendered_body,''),attempt_count,max_attempts`, id).Scan(
		&out.ID, &out.TenantID, &out.AppID, &out.TemplateID, &out.PushDeviceID, &out.MessageID, &out.MessageRunID, &out.Channel, &out.Provider, &out.TargetCiphertext, &out.PayloadCiphertext, &out.RenderedSubject, &out.RenderedBody, &out.AttemptCount, &out.MaxAttempts)
	return out, err
}

func (w *DeliveryWorker) send(ctx context.Context, delivery storedDelivery) deliveryResult {
	if delivery.Channel == "push" {
		return w.sendPush(ctx, delivery)
	}
	target, err := w.secrets.Open(delivery.TargetCiphertext, delivery.TenantID.String())
	if err != nil {
		return permanent(errors.New("encrypted delivery target cannot be opened"))
	}
	config, err := w.loadConfig(ctx, delivery.TenantID)
	if err != nil {
		return permanent(err)
	}
	if delivery.Channel == "email" {
		if delivery.RenderedSubject == "" && delivery.RenderedBody == "" {
			var renderErr error
			delivery, renderErr = w.renderEncryptedEmail(ctx, delivery)
			if renderErr != nil {
				return permanent(renderErr)
			}
		}
		return w.sendEmail(ctx, string(target), delivery, config)
	}
	if delivery.Channel != "sms" {
		return permanent(errors.New("unsupported delivery channel"))
	}
	payload, err := w.secrets.Open(delivery.PayloadCiphertext, delivery.TenantID.String()+":notification-payload")
	if err != nil {
		return permanent(errors.New("encrypted delivery variables cannot be opened"))
	}
	variables := map[string]string{}
	if json.Unmarshal(payload, &variables) != nil {
		return permanent(errors.New("delivery variables are invalid"))
	}
	binding, err := w.loadBinding(ctx, delivery.TenantID, delivery.TemplateID, delivery.Provider)
	if err != nil {
		return permanent(err)
	}
	switch delivery.Provider {
	case "tencent":
		return w.sendTencentSMS(ctx, delivery.TenantID, string(target), variables, binding, config)
	case "aliyun":
		return w.sendAliyunSMS(delivery.TenantID, string(target), variables, binding, config)
	default:
		return permanent(errors.New("SMS provider is not registered"))
	}
}

func (w *DeliveryWorker) sendPush(ctx context.Context, delivery storedDelivery) deliveryResult {
	if !w.pushEnabled {
		return permanent(errors.New("push delivery disabled by global kill switch"))
	}
	if w.pushSender == nil || delivery.AppID == uuid.Nil || delivery.PushDeviceID == uuid.Nil {
		return permanent(errors.New("push delivery adapter unavailable"))
	}
	if delivery.MessageID != uuid.Nil {
		var deliverable bool
		if err := w.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notify.messages WHERE tenant_id=$1 AND app_id=$2 AND id=$3
			AND status='published' AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at>now()))`, delivery.TenantID, delivery.AppID, delivery.MessageID).Scan(&deliverable); err != nil {
			return deliveryResult{retryable: true, risk: "none", class: "transient", errorCode: "PUSH.MESSAGE_STATE_UNAVAILABLE", retryAfter: 15 * time.Second, err: errors.New("push message state is temporarily unavailable")}
		}
		if !deliverable {
			return deliveryResult{cancelled: true}
		}
	}
	token, err := w.secrets.Open(delivery.TargetCiphertext, "push-token:"+delivery.AppID.String()+":"+delivery.Provider)
	if err != nil {
		return permanent(errors.New("encrypted push token cannot be opened"))
	}
	plaintext, err := w.secrets.Open(delivery.PayloadCiphertext, "push-payload:"+delivery.AppID.String()+":"+delivery.PushDeviceID.String())
	if err != nil {
		return permanent(errors.New("encrypted push payload cannot be opened"))
	}
	var payload push.SendPayload
	if json.Unmarshal(plaintext, &payload) != nil || payload.SchemaVersion != 1 || payload.DeliveryID != delivery.ID {
		return permanent(errors.New("push payload is invalid"))
	}
	result := w.pushSender.Send(ctx, delivery.TenantID, delivery.AppID, delivery.Provider, string(token), payload)
	if result.Class == "accepted" {
		return deliveryResult{messageID: result.ProviderMessageID, class: result.Class}
	}
	risk := "none"
	if result.Class == "unknown_after_write" {
		risk = "manual_review"
	}
	retryable := result.Class == "throttled" || result.Class == "transient"
	return deliveryResult{retryable: retryable, risk: risk, class: result.Class, errorCode: result.ErrorCode, retryAfter: result.RetryAfter, err: errors.New(result.SafeSummary)}
}

var notificationPlaceholder = regexp.MustCompile(`\{\{\s*([a-zA-Z][a-zA-Z0-9_.-]*)\s*\}\}`)

func (w *DeliveryWorker) renderEncryptedEmail(ctx context.Context, delivery storedDelivery) (storedDelivery, error) {
	variables, err := decryptNotificationVariables(w.secrets, delivery.PayloadCiphertext, delivery.TenantID)
	if err != nil {
		return delivery, err
	}
	var subject, body string
	if err = w.pool.QueryRow(ctx, `SELECT COALESCE(subject_template,''),body_template FROM notify.templates WHERE id=$1 AND channel='email' AND status='active'`, delivery.TemplateID).Scan(&subject, &body); err != nil {
		return delivery, errors.New("email template is unavailable")
	}
	if delivery.RenderedSubject, delivery.RenderedBody, err = renderNotificationTemplate(subject, body, variables); err != nil {
		return delivery, err
	}
	return delivery, nil
}

func decryptNotificationVariables(opener SecretOpener, ciphertext []byte, tenantID uuid.UUID) (map[string]any, error) {
	payload, err := opener.Open(ciphertext, tenantID.String()+":notification-payload")
	if err != nil {
		return nil, errors.New("encrypted email variables cannot be opened")
	}
	variables := map[string]any{}
	if json.Unmarshal(payload, &variables) != nil {
		return nil, errors.New("encrypted email variables are invalid")
	}
	return variables, nil
}

func renderNotificationTemplate(subject, body string, variables map[string]any) (string, string, error) {
	render := func(raw string) (string, error) {
		missing := false
		out := notificationPlaceholder.ReplaceAllStringFunc(raw, func(match string) string {
			key := notificationPlaceholder.FindStringSubmatch(match)[1]
			value, ok := variables[key]
			if !ok {
				missing = true
				return ""
			}
			return fmt.Sprint(value)
		})
		if missing {
			return "", errors.New("email template variables are incomplete")
		}
		return out, nil
	}
	renderedSubject, err := render(subject)
	if err != nil {
		return "", "", err
	}
	renderedBody, err := render(body)
	if err != nil {
		return "", "", err
	}
	return renderedSubject, renderedBody, nil
}

type configValues struct {
	values  map[string]json.RawMessage
	secrets map[string]string
	version int32
}

type cachedConfig struct {
	value     configValues
	expiresAt time.Time
}

func (w *DeliveryWorker) loadConfig(ctx context.Context, tenantID uuid.UUID) (configValues, error) {
	var version int32
	if err := w.pool.QueryRow(ctx, `SELECT COALESCE(max(version),0) FROM sys.config_items WHERE tenant_id=$1 AND module_code='notifications' AND config_group IN ('email','sms') AND status='active'`, tenantID).Scan(&version); err != nil {
		return configValues{}, errors.New("notification configuration version cannot be loaded")
	}
	w.cacheMu.Lock()
	if cached, ok := w.configs[tenantID]; ok && cached.value.version == version && time.Now().Before(cached.expiresAt) {
		w.cacheMu.Unlock()
		return cached.value, nil
	}
	w.cacheMu.Unlock()
	rows, err := w.pool.Query(ctx, `SELECT config_key::text,COALESCE(value_json,default_value_json),is_secret,secret_ciphertext FROM sys.config_items WHERE tenant_id=$1 AND module_code='notifications' AND config_group IN ('email','sms') AND status='active'`, tenantID)
	if err != nil {
		return configValues{}, errors.New("notification configuration cannot be loaded")
	}
	defer rows.Close()
	out := configValues{values: map[string]json.RawMessage{}, secrets: map[string]string{}, version: version}
	for rows.Next() {
		var key string
		var raw json.RawMessage
		var secret bool
		var ciphertext []byte
		if rows.Scan(&key, &raw, &secret, &ciphertext) != nil {
			return configValues{}, errors.New("notification configuration is invalid")
		}
		if secret {
			if len(ciphertext) == 0 {
				continue
			}
			plaintext, openErr := w.secrets.Open(ciphertext, tenantID.String())
			if openErr != nil {
				return configValues{}, errors.New("notification secret cannot be opened")
			}
			out.secrets[key] = strings.TrimSpace(string(plaintext))
		} else {
			out.values[key] = raw
		}
	}
	if err = rows.Err(); err != nil {
		return configValues{}, err
	}
	w.cacheMu.Lock()
	w.configs[tenantID] = cachedConfig{value: out, expiresAt: time.Now().Add(5 * time.Minute)}
	w.cacheMu.Unlock()
	return out, nil
}

type smsBinding struct {
	templateID     string
	signName       string
	parameterOrder []string
}

func (w *DeliveryWorker) loadBinding(ctx context.Context, tenantID, templateID uuid.UUID, provider string) (smsBinding, error) {
	var out smsBinding
	var order []byte
	err := w.pool.QueryRow(ctx, `SELECT external_template_id,COALESCE(sign_name,''),parameter_order FROM notify.sms_template_bindings WHERE tenant_id=$1 AND template_id=$2 AND provider=$3 AND status='active'`, tenantID, templateID, provider).Scan(&out.templateID, &out.signName, &order)
	if errors.Is(err, pgx.ErrNoRows) {
		return smsBinding{}, errors.New("active SMS template binding is missing")
	}
	if err != nil || json.Unmarshal(order, &out.parameterOrder) != nil {
		return smsBinding{}, errors.New("SMS template binding is invalid")
	}
	return out, nil
}

func (w *DeliveryWorker) sendEmail(ctx context.Context, target string, delivery storedDelivery, config configValues) deliveryResult {
	if !boolConfig(config.values, "email.enabled", false) {
		return permanent(errors.New("email delivery is disabled"))
	}
	host := stringConfig(config.values, "email.smtp_host", "")
	port := intConfig(config.values, "email.smtp_port", 587)
	username := stringConfig(config.values, "email.smtp_username", "")
	password := config.secrets["email.smtp_password"]
	from := stringConfig(config.values, "email.from_address", "")
	fromName := stringConfig(config.values, "email.from_name", "AppKernia")
	mode := stringConfig(config.values, "email.tls_mode", "")
	if mode == "" {
		if boolConfig(config.values, "email.use_tls", true) {
			mode = "starttls"
		} else {
			mode = "none"
		}
	}
	if host == "" || from == "" || port < 1 || port > 65535 || !oneOf(mode, "starttls", "implicit", "none") || w.environment == "production" && mode == "none" {
		return permanent(errors.New("email configuration is incomplete"))
	}
	timeout := time.Duration(intConfig(config.values, "email.timeout_seconds", 15)) * time.Second
	opts := []mail.Option{mail.WithPort(port), mail.WithTimeout(timeout)}
	switch mode {
	case "implicit":
		opts = append(opts, mail.WithSSL())
	case "starttls":
		opts = append(opts, mail.WithTLSPortPolicy(mail.TLSMandatory))
	case "none":
		opts = append(opts, mail.WithTLSPortPolicy(mail.NoTLS))
	}
	if username != "" {
		if password == "" {
			return permanent(errors.New("SMTP credential is incomplete"))
		}
		opts = append(opts, mail.WithUsername(username), mail.WithPassword(password), mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover))
	}
	client, err := mail.NewClient(host, opts...)
	if err != nil {
		return permanent(errors.New("SMTP client configuration is invalid"))
	}
	message := mail.NewMsg()
	if message.FromFormat(fromName, from) != nil || message.To(target) != nil {
		return permanent(errors.New("email address configuration is invalid"))
	}
	message.Subject(delivery.RenderedSubject)
	contentType := mail.TypeTextPlain
	var bodyFormat string
	_ = w.pool.QueryRow(ctx, `SELECT body_format FROM notify.templates WHERE id=$1`, delivery.TemplateID).Scan(&bodyFormat)
	if bodyFormat == "html" {
		contentType = mail.TypeTextHTML
	}
	message.SetBodyString(contentType, delivery.RenderedBody)
	err = client.DialAndSendWithContext(ctx, message)
	if err == nil {
		return deliveryResult{messageID: message.GetMessageID(), risk: "none"}
	}
	if message.SendErrorIsTemp() {
		return deliveryResult{retryable: true, risk: "none", err: errors.New("temporary SMTP rejection")}
	}
	return permanent(errors.New("SMTP delivery rejected"))
}

func (w *DeliveryWorker) sendTencentSMS(ctx context.Context, tenantID uuid.UUID, target string, variables map[string]string, binding smsBinding, config configValues) deliveryResult {
	if !boolConfig(config.values, "sms.enabled", false) || stringConfig(config.values, "sms.provider", "none") != "tencent" {
		return permanent(errors.New("Tencent SMS is not enabled"))
	}
	secretID, secretKey := config.secrets["sms.tencent.secret_id"], config.secrets["sms.tencent.secret_key"]
	appID := stringConfig(config.values, "sms.tencent.sdk_app_id", "")
	sign := binding.signName
	if sign == "" {
		sign = stringConfig(config.values, "sms.tencent.sign_name", "")
	}
	if secretID == "" || secretKey == "" || appID == "" || sign == "" || binding.templateID == "" {
		return permanent(errors.New("Tencent SMS configuration is incomplete"))
	}
	parameters := make([]*string, 0, len(binding.parameterOrder))
	for _, key := range binding.parameterOrder {
		value, ok := variables[key]
		if !ok {
			return permanent(errors.New("Tencent SMS template parameter is missing"))
		}
		parameters = append(parameters, common.StringPtr(value))
	}
	cacheKey := tenantID.String() + ":tencent:" + fmt.Sprint(config.version)
	w.cacheMu.Lock()
	client := w.tencent[cacheKey]
	w.cacheMu.Unlock()
	if client == nil {
		clientProfile := profile.NewClientProfile()
		clientProfile.HttpProfile.Endpoint = stringConfig(config.values, "sms.tencent.endpoint", "sms.tencentcloudapi.com")
		var clientErr error
		client, clientErr = tencentsms.NewClient(common.NewCredential(secretID, secretKey), stringConfig(config.values, "sms.tencent.region", "ap-guangzhou"), clientProfile)
		if clientErr != nil {
			return permanent(errors.New("Tencent SMS client configuration is invalid"))
		}
		w.cacheMu.Lock()
		w.tencent[cacheKey] = client
		w.cacheMu.Unlock()
	}
	request := tencentsms.NewSendSmsRequest()
	request.PhoneNumberSet = []*string{common.StringPtr(target)}
	request.SmsSdkAppId = common.StringPtr(appID)
	request.SignName = common.StringPtr(sign)
	request.TemplateId = common.StringPtr(binding.templateID)
	request.TemplateParamSet = parameters
	response, err := client.SendSmsWithContext(ctx, request)
	if err != nil {
		return uncertain(errors.New("Tencent SMS result is uncertain"))
	}
	if response == nil || response.Response == nil || len(response.Response.SendStatusSet) != 1 || response.Response.SendStatusSet[0] == nil {
		return uncertain(errors.New("Tencent SMS response is incomplete"))
	}
	status := response.Response.SendStatusSet[0]
	code := stringValue(status.Code)
	if strings.EqualFold(code, "Ok") {
		return deliveryResult{messageID: stringValue(status.SerialNo), risk: "none"}
	}
	if strings.HasPrefix(code, "LimitExceeded.") {
		return deliveryResult{retryable: true, risk: "none", err: errors.New("Tencent SMS rate limit rejection")}
	}
	return permanent(errors.New("Tencent SMS rejected the request"))
}

func (w *DeliveryWorker) sendAliyunSMS(tenantID uuid.UUID, target string, variables map[string]string, binding smsBinding, config configValues) deliveryResult {
	if !boolConfig(config.values, "sms.enabled", false) || stringConfig(config.values, "sms.provider", "none") != "aliyun" {
		return permanent(errors.New("Alibaba Cloud SMS is not enabled"))
	}
	accessKeyID, accessKeySecret := config.secrets["sms.aliyun.access_key_id"], config.secrets["sms.aliyun.access_key_secret"]
	sign := binding.signName
	if sign == "" {
		sign = stringConfig(config.values, "sms.aliyun.sign_name", "")
	}
	if accessKeyID == "" || accessKeySecret == "" || sign == "" || binding.templateID == "" {
		return permanent(errors.New("Alibaba Cloud SMS configuration is incomplete"))
	}
	params, _ := json.Marshal(variables)
	cacheKey := tenantID.String() + ":aliyun:" + fmt.Sprint(config.version)
	w.cacheMu.Lock()
	client := w.aliyun[cacheKey]
	w.cacheMu.Unlock()
	if client == nil {
		clientConfig := (&openapi.Config{}).SetAccessKeyId(accessKeyID).SetAccessKeySecret(accessKeySecret).SetEndpoint(stringConfig(config.values, "sms.aliyun.endpoint", "dysmsapi.aliyuncs.com"))
		var clientErr error
		client, clientErr = dysms.NewClient(clientConfig)
		if clientErr != nil {
			return permanent(errors.New("Alibaba Cloud SMS client configuration is invalid"))
		}
		w.cacheMu.Lock()
		w.aliyun[cacheKey] = client
		w.cacheMu.Unlock()
	}
	request := (&dysms.SendSmsRequest{}).SetPhoneNumbers(target).SetSignName(sign).SetTemplateCode(binding.templateID).SetTemplateParam(string(params))
	runtime := &service.RuntimeOptions{Autoretry: tea.Bool(false), ReadTimeout: tea.Int(30_000), ConnectTimeout: tea.Int(10_000)}
	response, err := client.SendSmsWithOptions(request, runtime)
	if err != nil {
		return uncertain(errors.New("Alibaba Cloud SMS result is uncertain"))
	}
	if response == nil || response.Body == nil {
		return uncertain(errors.New("Alibaba Cloud SMS response is incomplete"))
	}
	code := stringValue(response.Body.Code)
	if strings.EqualFold(code, "OK") {
		return deliveryResult{messageID: stringValue(response.Body.BizId), risk: "none"}
	}
	if strings.Contains(strings.ToLower(code), "limit") || strings.Contains(strings.ToLower(code), "throttl") {
		return deliveryResult{retryable: true, risk: "none", err: errors.New("Alibaba Cloud SMS rate limit rejection")}
	}
	return permanent(errors.New("Alibaba Cloud SMS rejected the request"))
}

func permanent(err error) deliveryResult { return deliveryResult{risk: "manual_review", err: err} }
func uncertain(err error) deliveryResult { return deliveryResult{risk: "duplicate_possible", err: err} }

func stringConfig(values map[string]json.RawMessage, key, fallback string) string {
	var out string
	if json.Unmarshal(values[key], &out) == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out)
	}
	return fallback
}

func intConfig(values map[string]json.RawMessage, key string, fallback int) int {
	var out int
	if json.Unmarshal(values[key], &out) == nil && out > 0 {
		return out
	}
	return fallback
}

func boolConfig(values map[string]json.RawMessage, key string, fallback bool) bool {
	var out bool
	if json.Unmarshal(values[key], &out) == nil {
		return out
	}
	return fallback
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	value := strings.Join(strings.Fields(err.Error()), " ")
	if len(value) > 500 {
		value = value[:500]
	}
	return value
}
