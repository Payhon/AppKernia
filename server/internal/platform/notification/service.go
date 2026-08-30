package notification

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalid             = errors.New("notification submission invalid")
	ErrNotFound            = errors.New("notification submission not found")
	ErrConflict            = errors.New("notification submission conflict")
	ErrIdempotencyConflict = errors.New("idempotency key already used with different payload")
)

type Scope struct {
	TenantID  uuid.UUID
	AppID     uuid.UUID
	ActorKind string
	ActorID   uuid.UUID
	RequestID string
	SourceIP  string
	UserAgent string
}

type Audience struct {
	Type    string      `json:"type"`
	UserIDs []uuid.UUID `json:"user_ids,omitempty"`
}

type LocalizedContent struct {
	Title map[string]string `json:"title"`
	Body  map[string]string `json:"body"`
}

type TemplateContent struct {
	Code      string            `json:"code"`
	Variables map[string]string `json:"variables,omitempty"`
}

type Content struct {
	Type     string            `json:"type"`
	Inline   *LocalizedContent `json:"inline,omitempty"`
	Template *TemplateContent  `json:"template,omitempty"`
}

type SubmitCommand struct {
	IdempotencyKey  string            `json:"-"`
	Source          string            `json:"source"`
	BusinessEventID string            `json:"business_event_id"`
	Category        string            `json:"category"`
	Audience        Audience          `json:"audience"`
	Content         Content           `json:"content"`
	Push            bool              `json:"push"`
	ScheduledAt     *time.Time        `json:"scheduled_at,omitempty"`
	ExpiresAt       *time.Time        `json:"expires_at,omitempty"`
	TTLSeconds      int32             `json:"ttl_seconds,omitempty"`
	CollapseKey     string            `json:"collapse_key,omitempty"`
	ThreadKey       string            `json:"thread_key,omitempty"`
	RouteKey        string            `json:"route_key,omitempty"`
	ResourceID      string            `json:"resource_id,omitempty"`
	RouteParams     map[string]string `json:"route_params,omitempty"`
}

type Submission struct {
	MessageID uuid.UUID `json:"message_id"`
	RunID     uuid.UUID `json:"run_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type SubmissionStatus struct {
	Submission
	RecipientCount    int64      `json:"recipient_count"`
	EvaluatedCount    int64      `json:"evaluated_count"`
	DeliveryCount     int64      `json:"delivery_count"`
	AcceptedCount     int64      `json:"accepted_count"`
	FailedCount       int64      `json:"failed_count"`
	InvalidTokenCount int64      `json:"invalid_token_count"`
	OpenedCount       int64      `json:"opened_count"`
	SkippedCount      int64      `json:"skipped_count"`
	ScheduledAt       *time.Time `json:"scheduled_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

type Service interface {
	Submit(context.Context, Scope, SubmitCommand) (Submission, error)
	SubmitTx(context.Context, pgx.Tx, Scope, SubmitCommand) (Submission, error)
	Status(context.Context, Scope, uuid.UUID) (SubmissionStatus, error)
	Cancel(context.Context, Scope, uuid.UUID) error
}

type PostgresService struct {
	pool  *pgxpool.Pool
	queue jobqueue.Enqueuer
	clock func() time.Time
}

func NewPostgresService(pool *pgxpool.Pool, queue jobqueue.Enqueuer) *PostgresService {
	return &PostgresService{pool: pool, queue: queue, clock: time.Now}
}

func (s *PostgresService) Submit(ctx context.Context, scope Scope, cmd SubmitCommand) (Submission, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Submission{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	out, err := s.SubmitTx(ctx, tx, scope, cmd)
	if err != nil {
		return Submission{}, err
	}
	return out, tx.Commit(ctx)
}

func (s *PostgresService) SubmitTx(ctx context.Context, tx pgx.Tx, scope Scope, cmd SubmitCommand) (Submission, error) {
	now := s.clock().UTC()
	cmd, err := normalize(scope, cmd, now)
	if err != nil {
		return Submission{}, err
	}
	requestJSON, err := json.Marshal(cmd)
	if err != nil {
		return Submission{}, err
	}
	digest := sha256.Sum256(requestJSON)
	var idemResponse []byte
	var existingHash []byte
	inserted := false
	err = tx.QueryRow(ctx, `INSERT INTO sys.idempotency_keys(tenant_id,identity_type,identity_id,idempotency_key,request_hash,locked_until,expires_at)
		VALUES($1,$2,$3,$4,$5,now()+interval '2 minutes',now()+interval '24 hours')
		ON CONFLICT(tenant_id,identity_type,identity_id,idempotency_key) DO NOTHING
		RETURNING true`, scope.TenantID, scope.ActorKind, scope.ActorID, cmd.IdempotencyKey, digest[:]).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT request_hash,response_body FROM sys.idempotency_keys
			WHERE tenant_id=$1 AND identity_type=$2 AND identity_id=$3 AND idempotency_key=$4 FOR UPDATE`, scope.TenantID, scope.ActorKind, scope.ActorID, cmd.IdempotencyKey).Scan(&existingHash, &idemResponse)
		if err != nil {
			return Submission{}, err
		}
		if !slices.Equal(existingHash, digest[:]) {
			return Submission{}, ErrIdempotencyConflict
		}
		if len(idemResponse) == 0 {
			return Submission{}, ErrConflict
		}
		var prior Submission
		if err = json.Unmarshal(idemResponse, &prior); err != nil {
			return Submission{}, err
		}
		return prior, nil
	}
	if err != nil {
		return Submission{}, err
	}

	var appExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.applications WHERE tenant_id=$1 AND id=$2 AND status='active' AND deleted_at IS NULL)`, scope.TenantID, scope.AppID).Scan(&appExists); err != nil {
		return Submission{}, err
	}
	if !appExists {
		return Submission{}, ErrNotFound
	}

	titles, bodies, templateMeta, err := resolveContent(ctx, tx, scope.TenantID, cmd.Content)
	if err != nil {
		return Submission{}, err
	}
	metadata := map[string]any{
		"source": cmd.Source, "business_event_id": cmd.BusinessEventID,
		"push_enabled": cmd.Push, "localized_titles": titles, "localized_bodies": bodies,
		"audience_scope": mapAudienceScope(cmd.Audience.Type), "audience_user_ids": cmd.Audience.UserIDs,
	}
	if templateMeta != nil {
		metadata["template"] = templateMeta
	}
	if cmd.ThreadKey != "" {
		metadata["thread_key"] = cmd.ThreadKey
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return Submission{}, err
	}
	routeParams := make(map[string]string, len(cmd.RouteParams)+1)
	for key, value := range cmd.RouteParams {
		routeParams[key] = value
	}
	if cmd.ResourceID != "" {
		routeParams["resource_id"] = cmd.ResourceID
	}
	routeJSON, err := json.Marshal(routeParams)
	if err != nil {
		return Submission{}, err
	}
	messageType := "system"
	if cmd.Category == "news_operations" {
		messageType = "marketing"
	} else if strings.Contains(cmd.Source, "security") {
		messageType = "security"
	}
	scheduledAt := now
	if cmd.ScheduledAt != nil {
		scheduledAt = cmd.ScheduledAt.UTC()
	}
	messageID, err := uuid.NewV7()
	if err != nil {
		return Submission{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO notify.messages(
		id,tenant_id,app_id,message_type,title,body,body_format,status,
		scheduled_at,expires_at,metadata,push_category,push_ttl_seconds,push_collapse_key,push_route_key,push_route_params)
		VALUES($1,$2,$3,$4,$5,$6,'plain','scheduled',$7,$8,$9,$10,$11,NULLIF($12,''),NULLIF($13,''),$14)`,
		messageID, scope.TenantID, scope.AppID, messageType, titles["zh-CN"], bodies["zh-CN"],
		scheduledAt, cmd.ExpiresAt, metadataJSON, cmd.Category, cmd.TTLSeconds, cmd.CollapseKey, cmd.RouteKey, routeJSON)
	if err != nil {
		return Submission{}, err
	}

	var recipientCount int64
	if cmd.Audience.Type == "users" {
		tag, insertErr := tx.Exec(ctx, `INSERT INTO notify.recipients(tenant_id,app_id,message_id,user_id)
			SELECT $1,$2,$3,member_id FROM unnest($4::uuid[]) member_id
			WHERE EXISTS(SELECT 1 FROM iam.tenant_members tm JOIN app.user_memberships am ON am.tenant_id=tm.tenant_id AND am.user_id=tm.user_id
				WHERE tm.tenant_id=$1 AND tm.user_id=member_id AND tm.status='active' AND am.app_id=$2 AND am.status='active')`,
			scope.TenantID, scope.AppID, messageID, cmd.Audience.UserIDs)
		if insertErr != nil {
			return Submission{}, insertErr
		}
		recipientCount = tag.RowsAffected()
		if recipientCount != int64(len(cmd.Audience.UserIDs)) {
			return Submission{}, ErrInvalid
		}
	} else {
		tag, insertErr := tx.Exec(ctx, `INSERT INTO notify.recipients(tenant_id,app_id,message_id,user_id)
			SELECT tm.tenant_id,$2,$3,tm.user_id FROM iam.tenant_members tm
			JOIN app.user_memberships am ON am.tenant_id=tm.tenant_id AND am.user_id=tm.user_id AND am.app_id=$2 AND am.status='active'
			WHERE tm.tenant_id=$1 AND tm.status='active'`, scope.TenantID, scope.AppID, messageID)
		if insertErr != nil {
			return Submission{}, insertErr
		}
		recipientCount = tag.RowsAffected()
	}
	if recipientCount == 0 {
		return Submission{}, ErrInvalid
	}
	runID, err := uuid.NewV7()
	if err != nil {
		return Submission{}, err
	}
	runStatus := "queued"
	if scheduledAt.After(now) {
		runStatus = "scheduled"
	}
	_, err = tx.Exec(ctx, `INSERT INTO notify.message_runs(id,tenant_id,app_id,message_id,trigger_type,status,recipient_count)
		VALUES($1,$2,$3,$4,'api_client',$5,$6)`, runID, scope.TenantID, scope.AppID, messageID, runStatus, recipientCount)
	if err != nil {
		return Submission{}, err
	}
	appID, resourceID := scope.AppID, messageID
	var scheduledFor *time.Time
	if scheduledAt.After(now) {
		value := scheduledAt
		scheduledFor = &value
	}
	_, err = s.queue.EnqueueTx(ctx, tx, jobqueue.Spec{
		Scope: jobqueue.Scope{TenantID: scope.TenantID, AppID: &appID, ModuleCode: "notify", ResourceType: "notification_message", ResourceID: &resourceID, CorrelationID: &runID},
		Args:  notify.MessagePublishJobArgs{TenantID: scope.TenantID, AppID: scope.AppID, MessageID: messageID}, Queue: "notifications", MaxAttempts: 5,
		ScheduledAt: scheduledFor, UniqueByArgs: true,
	})
	if err != nil {
		return Submission{}, err
	}
	created := Submission{MessageID: messageID, RunID: runID, Status: runStatus, CreatedAt: now}
	responseJSON, err := json.Marshal(created)
	if err != nil {
		return Submission{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE sys.idempotency_keys SET response_status=202,response_headers='{"content-type":"application/json"}'::jsonb,
		response_body=$5,completed_at=now(),locked_until=NULL WHERE tenant_id=$1 AND identity_type=$2 AND identity_id=$3 AND idempotency_key=$4`,
		scope.TenantID, scope.ActorKind, scope.ActorID, cmd.IdempotencyKey, responseJSON)
	if err != nil {
		return Submission{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,after_data,succeeded)
		VALUES($1,$2,'notify','notify.message.submit','notify.message.submit','notification_message',$3,'POST',$4,202,NULLIF($5,'')::inet,NULLIF($6,''),jsonb_build_object('app_id',$7::text,'actor_kind',$8::text,'actor_id',$9::text,'run_id',$10::text),true)`,
		scope.TenantID, scope.RequestID, messageID.String(), "/api/v1/apps/"+scope.AppID.String()+"/notifications", scope.SourceIP, scope.UserAgent, scope.AppID, scope.ActorKind, scope.ActorID, runID)
	if err != nil {
		return Submission{}, err
	}
	return created, nil
}

func (s *PostgresService) Status(ctx context.Context, scope Scope, messageID uuid.UUID) (SubmissionStatus, error) {
	if err := validateScope(scope); err != nil || messageID == uuid.Nil {
		return SubmissionStatus{}, ErrInvalid
	}
	var out SubmissionStatus
	err := s.pool.QueryRow(ctx, `SELECT m.id,r.id,r.status,r.created_at,r.recipient_count,r.evaluated_count,r.delivery_count,
		r.accepted_count,r.failed_count,r.invalid_token_count,r.opened_count,r.skipped_count,m.scheduled_at,r.completed_at
		FROM notify.messages m JOIN notify.message_runs r ON r.tenant_id=m.tenant_id AND r.app_id=m.app_id AND r.message_id=m.id
		WHERE m.tenant_id=$1 AND m.app_id=$2 AND m.id=$3 AND m.deleted_at IS NULL`, scope.TenantID, scope.AppID, messageID).Scan(
		&out.MessageID, &out.RunID, &out.Status, &out.CreatedAt, &out.RecipientCount, &out.EvaluatedCount, &out.DeliveryCount,
		&out.AcceptedCount, &out.FailedCount, &out.InvalidTokenCount, &out.OpenedCount, &out.SkippedCount, &out.ScheduledAt, &out.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SubmissionStatus{}, ErrNotFound
	}
	return out, err
}

func (s *PostgresService) Cancel(ctx context.Context, scope Scope, messageID uuid.UUID) error {
	if err := validateScope(scope); err != nil || messageID == uuid.Nil {
		return ErrInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE notify.messages SET status='cancelled',updated_at=now()
		WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND status IN ('draft','scheduled') AND deleted_at IS NULL`, scope.TenantID, scope.AppID, messageID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notify.messages WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND deleted_at IS NULL)`, scope.TenantID, scope.AppID, messageID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return ErrConflict
	}
	_, err = tx.Exec(ctx, `UPDATE notify.message_runs SET status='cancelled',completed_at=now() WHERE tenant_id=$1 AND app_id=$2 AND message_id=$3`, scope.TenantID, scope.AppID, messageID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE jobs.task_runs SET status='cancelled',finalized_at=now()
		WHERE tenant_id=$1 AND app_id=$2 AND resource_type='notification_message' AND resource_id=$3 AND status IN ('scheduled','queued','retry_wait')`, scope.TenantID, scope.AppID, messageID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var (
	codePattern      = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,95}$`)
	eventPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,159}$`)
	templateVariable = regexp.MustCompile(`\{\{\s*([A-Za-z][A-Za-z0-9_.-]*)\s*\}\}`)
)

func normalize(scope Scope, cmd SubmitCommand, now time.Time) (SubmitCommand, error) {
	if err := validateScope(scope); err != nil {
		return cmd, err
	}
	cmd.IdempotencyKey = strings.TrimSpace(cmd.IdempotencyKey)
	cmd.Source = strings.TrimSpace(cmd.Source)
	cmd.BusinessEventID = strings.TrimSpace(cmd.BusinessEventID)
	cmd.Category = strings.TrimSpace(cmd.Category)
	cmd.Audience.Type = strings.TrimSpace(cmd.Audience.Type)
	cmd.Content.Type = strings.TrimSpace(cmd.Content.Type)
	cmd.CollapseKey = strings.TrimSpace(cmd.CollapseKey)
	cmd.ThreadKey = strings.TrimSpace(cmd.ThreadKey)
	cmd.RouteKey = strings.TrimSpace(cmd.RouteKey)
	cmd.ResourceID = strings.TrimSpace(cmd.ResourceID)
	if len(cmd.IdempotencyKey) < 8 || len(cmd.IdempotencyKey) > 255 || !codePattern.MatchString(cmd.Source) || !eventPattern.MatchString(cmd.BusinessEventID) ||
		(cmd.Category != "service_security" && cmd.Category != "news_operations") || (cmd.Audience.Type != "users" && cmd.Audience.Type != "all_active_app_users") ||
		(cmd.Content.Type != "inline" && cmd.Content.Type != "template") {
		return cmd, ErrInvalid
	}
	if cmd.TTLSeconds == 0 {
		cmd.TTLSeconds = 86400
	}
	if cmd.TTLSeconds < 300 || cmd.TTLSeconds > 604800 || len(cmd.CollapseKey) > 128 || len(cmd.ThreadKey) > 128 ||
		(cmd.RouteKey != "" && !codePattern.MatchString(cmd.RouteKey)) || !opaqueValue(cmd.ResourceID, 160) || len(cmd.RouteParams) > 8 {
		return cmd, ErrInvalid
	}
	for key, value := range cmd.RouteParams {
		if !codePattern.MatchString(key) || !opaqueValue(strings.TrimSpace(value), 160) {
			return cmd, ErrInvalid
		}
		cmd.RouteParams[key] = strings.TrimSpace(value)
	}
	if cmd.ScheduledAt != nil {
		value := cmd.ScheduledAt.UTC()
		cmd.ScheduledAt = &value
		if value.Before(now.Add(-time.Minute)) || value.After(now.AddDate(1, 0, 0)) {
			return cmd, ErrInvalid
		}
	}
	if cmd.ExpiresAt != nil {
		value := cmd.ExpiresAt.UTC()
		cmd.ExpiresAt = &value
		if !value.After(now) || cmd.ScheduledAt != nil && !value.After(*cmd.ScheduledAt) {
			return cmd, ErrInvalid
		}
	}
	if cmd.Audience.Type == "users" {
		if len(cmd.Audience.UserIDs) == 0 || len(cmd.Audience.UserIDs) > 500 {
			return cmd, ErrInvalid
		}
		seen := make(map[uuid.UUID]struct{}, len(cmd.Audience.UserIDs))
		out := make([]uuid.UUID, 0, len(cmd.Audience.UserIDs))
		for _, id := range cmd.Audience.UserIDs {
			if id == uuid.Nil {
				return cmd, ErrInvalid
			}
			if _, exists := seen[id]; !exists {
				seen[id] = struct{}{}
				out = append(out, id)
			}
		}
		cmd.Audience.UserIDs = out
	} else {
		cmd.Audience.UserIDs = nil
	}
	if cmd.Content.Type == "inline" {
		if cmd.Content.Inline == nil || cmd.Content.Template != nil || !validLocalized(cmd.Content.Inline.Title, 300) || !validLocalized(cmd.Content.Inline.Body, 100_000) {
			return cmd, ErrInvalid
		}
	} else if cmd.Content.Template == nil || cmd.Content.Inline != nil || !codePattern.MatchString(strings.TrimSpace(cmd.Content.Template.Code)) || len(cmd.Content.Template.Variables) > 100 {
		return cmd, ErrInvalid
	}
	return cmd, nil
}

func validateScope(scope Scope) error {
	if scope.TenantID == uuid.Nil || scope.AppID == uuid.Nil || scope.ActorID == uuid.Nil || (scope.ActorKind != "api_client" && scope.ActorKind != "user") || strings.TrimSpace(scope.RequestID) == "" {
		return ErrInvalid
	}
	return nil
}

func validLocalized(value map[string]string, max int) bool {
	if len(value) != 2 {
		return false
	}
	for _, locale := range []string{"zh-CN", "en-US"} {
		text := strings.TrimSpace(value[locale])
		if text == "" || utf8.RuneCountInString(text) > max {
			return false
		}
		value[locale] = text
	}
	return true
}

func opaqueValue(value string, max int) bool {
	return len(value) <= max && !strings.Contains(value, "://") && !strings.ContainsAny(value, "<>{}\n\r\t")
}

func mapAudienceScope(value string) string {
	if value == "users" {
		return "selected"
	}
	return "all"
}

func resolveContent(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, content Content) (map[string]string, map[string]string, map[string]any, error) {
	if content.Type == "inline" {
		return content.Inline.Title, content.Inline.Body, nil, nil
	}
	code := strings.TrimSpace(content.Template.Code)
	titles := map[string]string{}
	bodies := map[string]string{}
	for _, locale := range []string{"zh-CN", "en-US"} {
		var subject *string
		var body string
		err := tx.QueryRow(ctx, `SELECT subject_template,body_template FROM notify.templates
			WHERE (tenant_id=$1 OR tenant_id IS NULL) AND code=$2 AND channel='in_app' AND status='active' AND (locale=$3 OR locale IS NULL)
			ORDER BY tenant_id NULLS LAST,(locale=$3) DESC LIMIT 1`, tenantID, code, locale).Scan(&subject, &body)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, ErrNotFound
		}
		if err != nil {
			return nil, nil, nil, err
		}
		title := code
		if subject != nil {
			title = *subject
		}
		var renderErr error
		titles[locale], renderErr = render(title, content.Template.Variables)
		if renderErr != nil {
			return nil, nil, nil, renderErr
		}
		bodies[locale], renderErr = render(body, content.Template.Variables)
		if renderErr != nil {
			return nil, nil, nil, renderErr
		}
	}
	return titles, bodies, map[string]any{"code": code, "variable_names": sortedKeys(content.Template.Variables)}, nil
}

func render(template string, variables map[string]string) (string, error) {
	invalid := false
	out := templateVariable.ReplaceAllStringFunc(template, func(match string) string {
		parts := templateVariable.FindStringSubmatch(match)
		value, ok := variables[parts[1]]
		if !ok || len(value) > 4000 {
			invalid = true
			return ""
		}
		return value
	})
	if invalid || templateVariable.MatchString(out) || strings.TrimSpace(out) == "" {
		return "", ErrInvalid
	}
	return strings.TrimSpace(out), nil
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
