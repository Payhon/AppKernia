package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type Postgres struct {
	pool  *pgxpool.Pool
	river *river.Client[pgx.Tx]
}

func NewPostgres(pool *pgxpool.Pool, clients ...*river.Client[pgx.Tx]) *Postgres {
	repository := &Postgres{pool: pool}
	if len(clients) > 0 {
		repository.river = clients[0]
	}
	return repository
}

type scanner interface{ Scan(...any) error }

const messageColumns = `id,app_id,message_type,title,body,body_format,status,scheduled_at,published_at,expires_at,metadata,created_at,updated_at`

func scanMessage(row scanner) (notify.Message, error) {
	var out notify.Message
	var metadata []byte
	if err := row.Scan(&out.ID, &out.AppID, &out.MessageType, &out.Title, &out.Body, &out.BodyFormat, &out.Status, &out.ScheduledAt, &out.PublishedAt, &out.ExpiresAt, &metadata, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return notify.Message{}, err
	}
	var audience struct {
		Scope   string      `json:"audience_scope"`
		UserIDs []uuid.UUID `json:"audience_user_ids"`
	}
	_ = json.Unmarshal(metadata, &audience)
	if audience.Scope == "" {
		audience.Scope = "all"
	}
	out.AudienceScope, out.AudienceUserIDs = audience.Scope, audience.UserIDs
	if out.AudienceUserIDs == nil {
		out.AudienceUserIDs = []uuid.UUID{}
	}
	return out, nil
}

func messageMeta(in notify.MessageInput) []byte {
	out, _ := json.Marshal(map[string]any{"audience_scope": in.AudienceScope, "audience_user_ids": in.AudienceUserIDs})
	return out
}

func (r *Postgres) ListMessages(ctx context.Context, tenantID, appID uuid.UUID, notice bool, f notify.PageFilter) (notify.MessagePage, error) {
	appID, err := r.scopedApp(ctx, tenantID, appID)
	if err != nil {
		return notify.MessagePage{}, err
	}
	args := []any{tenantID, appID, notice}
	where := []string{"tenant_id=$1", "app_id=$2", "deleted_at IS NULL", "($3::boolean = (message_type='notice'))"}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	if f.Query != "" {
		add("(title ILIKE '%%' || $%[1]d || '%%' OR body ILIKE '%%' || $%[1]d || '%%')", f.Query)
	}
	if f.Status != "" {
		add("status=$%d", f.Status)
	}
	if f.Type != "" {
		add("message_type=$%d", f.Type)
	}
	condition := strings.Join(where, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM notify.messages WHERE "+condition, args...).Scan(&total); err != nil {
		return notify.MessagePage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, fmt.Sprintf("SELECT %s FROM notify.messages WHERE %s ORDER BY created_at DESC,id DESC LIMIT $%d OFFSET $%d", messageColumns, condition, len(args)-1, len(args)), args...)
	if err != nil {
		return notify.MessagePage{}, err
	}
	defer rows.Close()
	items := make([]notify.Message, 0)
	for rows.Next() {
		item, scanErr := scanMessage(rows)
		if scanErr != nil {
			return notify.MessagePage{}, scanErr
		}
		items = append(items, item)
	}
	return notify.MessagePage{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, rows.Err()
}

func (r *Postgres) GetMessage(ctx context.Context, tenantID, appID, id uuid.UUID, notice bool) (notify.Message, error) {
	appID, scopeErr := r.scopedApp(ctx, tenantID, appID)
	if scopeErr != nil {
		return notify.Message{}, scopeErr
	}
	out, err := scanMessage(r.pool.QueryRow(ctx, fmt.Sprintf("SELECT %s FROM notify.messages WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND deleted_at IS NULL AND ($4::boolean = (message_type='notice'))", messageColumns), tenantID, appID, id, notice))
	if errors.Is(err, pgx.ErrNoRows) {
		return notify.Message{}, notify.ErrNotFound
	}
	return out, err
}

func (r *Postgres) CreateMessage(ctx context.Context, p notify.Principal, notice bool, in notify.MessageInput) (notify.Message, error) {
	var err error
	p.AppID, err = r.scopedApp(ctx, p.TenantID, p.AppID)
	if err != nil {
		return notify.Message{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return notify.Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	out, err := scanMessage(tx.QueryRow(ctx, fmt.Sprintf(`INSERT INTO notify.messages(tenant_id,app_id,sender_user_id,message_type,title,body,body_format,status,scheduled_at,expires_at,metadata)
		VALUES($1,$2,$3,$4,$5,$6,$7,'draft',$8,$9,$10) RETURNING %s`, messageColumns), p.TenantID, p.AppID, p.UserID, in.MessageType, in.Title, in.Body, in.BodyFormat, in.ScheduledAt, in.ExpiresAt, messageMeta(in)))
	if err != nil {
		return notify.Message{}, err
	}
	resource := "notify.message"
	if notice {
		resource = "notify.notice"
	}
	if err = insertAudit(ctx, tx, p, resource+".create", resource, out.ID, "POST", map[string]any{"status": out.Status, "message_type": out.MessageType}); err != nil {
		return notify.Message{}, err
	}
	return out, tx.Commit(ctx)
}

func (r *Postgres) UpdateMessage(ctx context.Context, p notify.Principal, id uuid.UUID, notice bool, in notify.MessageInput) (notify.Message, error) {
	var err error
	p.AppID, err = r.scopedApp(ctx, p.TenantID, p.AppID)
	if err != nil {
		return notify.Message{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return notify.Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	out, err := scanMessage(tx.QueryRow(ctx, fmt.Sprintf(`UPDATE notify.messages SET message_type=$1,title=$2,body=$3,body_format=$4,scheduled_at=$5,expires_at=$6,metadata=$7
		WHERE tenant_id=$8 AND app_id=$9 AND id=$10 AND deleted_at IS NULL AND status IN ('draft','scheduled') AND ($11::boolean=(message_type='notice')) RETURNING %s`, messageColumns), in.MessageType, in.Title, in.Body, in.BodyFormat, in.ScheduledAt, in.ExpiresAt, messageMeta(in), p.TenantID, p.AppID, id, notice))
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.GetMessage(ctx, p.TenantID, p.AppID, id, notice); getErr != nil {
			return notify.Message{}, getErr
		}
		return notify.Message{}, notify.ErrConflict
	}
	if err != nil {
		return notify.Message{}, err
	}
	resource := "notify.message"
	if notice {
		resource = "notify.notice"
	}
	if err = insertAudit(ctx, tx, p, resource+".update", resource, id, "PATCH", map[string]any{"status": out.Status, "message_type": out.MessageType}); err != nil {
		return notify.Message{}, err
	}
	return out, tx.Commit(ctx)
}

func recipientQuery(message notify.Message, count bool) (string, []any) {
	args := []any{}
	where := []string{"tm.tenant_id=$1", "am.app_id=$2", "tm.status='active'", "am.status='active'", "u.status='active'", "u.deleted_at IS NULL"}
	args = append(args, uuid.Nil, message.AppID) // tenant is replaced by caller
	if message.AudienceScope == "selected" {
		args = append(args, message.AudienceUserIDs)
		where = append(where, "u.id=ANY($3::uuid[])")
	}
	selectSQL := "count(*)"
	limit := ""
	if !count {
		selectSQL = "u.id,COALESCE(tm.display_name,u.display_name),COALESCE(u.email::text,'')"
		limit = " ORDER BY COALESCE(tm.display_name,u.display_name),u.id LIMIT 20"
	}
	return "SELECT " + selectSQL + " FROM iam.tenant_members tm JOIN app.user_memberships am ON am.tenant_id=tm.tenant_id AND am.user_id=tm.user_id JOIN iam.users u ON u.id=tm.user_id WHERE " + strings.Join(where, " AND ") + limit, args
}

func previewWith(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, tenantID uuid.UUID, message notify.Message) (notify.RecipientPreview, error) {
	countSQL, args := recipientQuery(message, true)
	args[0] = tenantID
	var total int64
	if err := q.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return notify.RecipientPreview{}, err
	}
	if message.AudienceScope == "selected" && total != int64(len(message.AudienceUserIDs)) {
		return notify.RecipientPreview{}, notify.ErrInvalid
	}
	if total == 0 {
		return notify.RecipientPreview{}, notify.ErrRecipientEmpty
	}
	listSQL, args := recipientQuery(message, false)
	args[0] = tenantID
	rows, err := q.Query(ctx, listSQL, args...)
	if err != nil {
		return notify.RecipientPreview{}, err
	}
	defer rows.Close()
	items := make([]notify.Recipient, 0)
	for rows.Next() {
		var item notify.Recipient
		var email string
		if err = rows.Scan(&item.UserID, &item.DisplayName, &email); err != nil {
			return notify.RecipientPreview{}, err
		}
		item.EmailHint = emailHint(email)
		items = append(items, item)
	}
	return notify.RecipientPreview{Count: total, Items: items}, rows.Err()
}

func (r *Postgres) PreviewRecipients(ctx context.Context, tenantID, appID uuid.UUID, message notify.Message) (notify.RecipientPreview, error) {
	appID, err := r.scopedApp(ctx, tenantID, appID)
	if err != nil {
		return notify.RecipientPreview{}, err
	}
	message.AppID = appID
	return previewWith(ctx, r.pool, tenantID, message)
}

func (r *Postgres) PublishMessage(ctx context.Context, p notify.Principal, id uuid.UUID, notice bool) (notify.Message, notify.RecipientPreview, error) {
	var err error
	p.AppID, err = r.scopedApp(ctx, p.TenantID, p.AppID)
	if err != nil {
		return notify.Message{}, notify.RecipientPreview{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return notify.Message{}, notify.RecipientPreview{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	message, err := scanMessage(tx.QueryRow(ctx, fmt.Sprintf("SELECT %s FROM notify.messages WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND deleted_at IS NULL AND ($4::boolean=(message_type='notice')) FOR UPDATE", messageColumns), p.TenantID, p.AppID, id, notice))
	if errors.Is(err, pgx.ErrNoRows) {
		return notify.Message{}, notify.RecipientPreview{}, notify.ErrNotFound
	}
	if err != nil {
		return notify.Message{}, notify.RecipientPreview{}, err
	}
	if message.Status != "draft" && message.Status != "scheduled" {
		return notify.Message{}, notify.RecipientPreview{}, notify.ErrConflict
	}
	preview, err := previewWith(ctx, tx, p.TenantID, message)
	if err != nil {
		return notify.Message{}, notify.RecipientPreview{}, err
	}
	insert := `INSERT INTO notify.recipients(tenant_id,app_id,message_id,user_id)
		SELECT $1,$2,$3,u.id FROM iam.tenant_members tm JOIN app.user_memberships am ON am.tenant_id=tm.tenant_id AND am.user_id=tm.user_id JOIN iam.users u ON u.id=tm.user_id
		WHERE tm.tenant_id=$1 AND am.app_id=$2 AND tm.status='active' AND am.status='active' AND u.status='active' AND u.deleted_at IS NULL`
	args := []any{p.TenantID, p.AppID, id}
	if message.AudienceScope == "selected" {
		insert += " AND u.id=ANY($4::uuid[])"
		args = append(args, message.AudienceUserIDs)
	}
	insert += " ON CONFLICT DO NOTHING"
	if _, err = tx.Exec(ctx, insert, args...); err != nil {
		return notify.Message{}, notify.RecipientPreview{}, err
	}
	now := time.Now().UTC()
	status := "published"
	publishedAt := &now
	if message.ScheduledAt != nil && message.ScheduledAt.After(now) {
		status, publishedAt = "scheduled", nil
	}
	message, err = scanMessage(tx.QueryRow(ctx, fmt.Sprintf("UPDATE notify.messages SET status=$1,published_at=$2 WHERE tenant_id=$3 AND app_id=$4 AND id=$5 RETURNING %s", messageColumns), status, publishedAt, p.TenantID, p.AppID, id))
	if err != nil {
		return notify.Message{}, notify.RecipientPreview{}, err
	}
	resource := "notify.message"
	if notice {
		resource = "notify.notice"
	}
	if err = insertAudit(ctx, tx, p, resource+".publish", resource, id, "POST", map[string]any{"status": status, "recipient_count": preview.Count}); err != nil {
		return notify.Message{}, notify.RecipientPreview{}, err
	}
	return message, preview, tx.Commit(ctx)
}

func (r *Postgres) CancelMessage(ctx context.Context, p notify.Principal, id uuid.UUID, notice bool) (notify.Message, error) {
	var err error
	p.AppID, err = r.scopedApp(ctx, p.TenantID, p.AppID)
	if err != nil {
		return notify.Message{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return notify.Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	message, err := scanMessage(tx.QueryRow(ctx, fmt.Sprintf(`UPDATE notify.messages SET status='cancelled' WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND deleted_at IS NULL
		AND status IN ('draft','scheduled','published') AND ($4::boolean=(message_type='notice')) RETURNING %s`, messageColumns), p.TenantID, p.AppID, id, notice))
	if errors.Is(err, pgx.ErrNoRows) {
		return notify.Message{}, notify.ErrConflict
	}
	if err != nil {
		return notify.Message{}, err
	}
	resource := "notify.message"
	if notice {
		resource = "notify.notice"
	}
	if err = insertAudit(ctx, tx, p, resource+".cancel", resource, id, "POST", map[string]any{"status": "cancelled"}); err != nil {
		return notify.Message{}, err
	}
	return message, tx.Commit(ctx)
}

func (r *Postgres) RecipientStats(ctx context.Context, tenantID, appID, id uuid.UUID, notice bool) (notify.RecipientStats, error) {
	if _, err := r.GetMessage(ctx, tenantID, appID, id, notice); err != nil {
		return notify.RecipientStats{}, err
	}
	var out notify.RecipientStats
	err := r.pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE delivery_status='pending'),count(*) FILTER(WHERE delivery_status='delivered'),count(*) FILTER(WHERE delivery_status='failed'),count(*) FILTER(WHERE read_at IS NOT NULL) FROM notify.recipients WHERE tenant_id=$1 AND app_id=$2 AND message_id=$3`, tenantID, appID, id).Scan(&out.Total, &out.Pending, &out.Delivered, &out.Failed, &out.Read)
	return out, err
}

const templateColumns = `id,tenant_id,code::text,name,channel,locale,subject_template,body_template,body_format,variables_schema,status,(tenant_id IS NULL),created_at,updated_at`

func scanTemplate(row scanner) (notify.Template, error) {
	var out notify.Template
	err := row.Scan(&out.ID, &out.TenantID, &out.Code, &out.Name, &out.Channel, &out.Locale, &out.SubjectTemplate, &out.BodyTemplate, &out.BodyFormat, &out.VariablesSchema, &out.Status, &out.IsLocked, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *Postgres) GetTemplate(ctx context.Context, tenantID, id uuid.UUID) (notify.Template, error) {
	out, err := scanTemplate(r.pool.QueryRow(ctx, fmt.Sprintf("SELECT %s FROM notify.templates WHERE id=$1 AND (tenant_id=$2 OR tenant_id IS NULL) ORDER BY tenant_id NULLS LAST LIMIT 1", templateColumns), id, tenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return notify.Template{}, notify.ErrNotFound
	}
	return out, err
}

func (r *Postgres) ListTemplates(ctx context.Context, tenantID uuid.UUID, f notify.PageFilter) (notify.TemplatePage, error) {
	args := []any{tenantID}
	where := []string{"(tenant_id=$1 OR tenant_id IS NULL)"}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	if f.Query != "" {
		add("(code::text ILIKE '%%'||$%[1]d||'%%' OR name ILIKE '%%'||$%[1]d||'%%')", f.Query)
	}
	if f.Status != "" {
		add("status=$%d", f.Status)
	}
	if f.Channel != "" {
		add("channel=$%d", f.Channel)
	}
	if f.Locale == "global" {
		where = append(where, "locale IS NULL")
	} else if f.Locale != "" {
		add("locale=$%d", f.Locale)
	}
	condition := strings.Join(where, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM notify.templates WHERE "+condition, args...).Scan(&total); err != nil {
		return notify.TemplatePage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, fmt.Sprintf("SELECT %s FROM notify.templates WHERE %s ORDER BY code,channel,locale NULLS FIRST LIMIT $%d OFFSET $%d", templateColumns, condition, len(args)-1, len(args)), args...)
	if err != nil {
		return notify.TemplatePage{}, err
	}
	defer rows.Close()
	items := make([]notify.Template, 0)
	for rows.Next() {
		item, scanErr := scanTemplate(rows)
		if scanErr != nil {
			return notify.TemplatePage{}, scanErr
		}
		items = append(items, item)
	}
	return notify.TemplatePage{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, rows.Err()
}

func (r *Postgres) CreateTemplate(ctx context.Context, p notify.Principal, in notify.TemplateInput) (notify.Template, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return notify.Template{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	out, err := scanTemplate(tx.QueryRow(ctx, fmt.Sprintf(`INSERT INTO notify.templates(tenant_id,code,name,channel,locale,subject_template,body_template,body_format,variables_schema,status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING %s`, templateColumns), p.TenantID, in.Code, in.Name, in.Channel, in.Locale, in.SubjectTemplate, in.BodyTemplate, in.BodyFormat, in.VariablesSchema, in.Status))
	if err != nil {
		if isUnique(err) {
			return notify.Template{}, notify.ErrConflict
		}
		return notify.Template{}, err
	}
	if err = insertAudit(ctx, tx, p, "notify.template.create", "notify.template", out.ID, "POST", map[string]any{"code": out.Code, "channel": out.Channel, "locale": out.Locale}); err != nil {
		return notify.Template{}, err
	}
	return out, tx.Commit(ctx)
}

func (r *Postgres) UpdateTemplate(ctx context.Context, p notify.Principal, id uuid.UUID, in notify.TemplateInput) (notify.Template, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return notify.Template{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	out, err := scanTemplate(tx.QueryRow(ctx, fmt.Sprintf(`UPDATE notify.templates SET code=$1,name=$2,channel=$3,locale=$4,subject_template=$5,body_template=$6,body_format=$7,variables_schema=$8,status=$9
		WHERE tenant_id=$10 AND id=$11 RETURNING %s`, templateColumns), in.Code, in.Name, in.Channel, in.Locale, in.SubjectTemplate, in.BodyTemplate, in.BodyFormat, in.VariablesSchema, in.Status, p.TenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return notify.Template{}, notify.ErrNotFound
	}
	if err != nil {
		if isUnique(err) {
			return notify.Template{}, notify.ErrConflict
		}
		return notify.Template{}, err
	}
	if err = insertAudit(ctx, tx, p, "notify.template.update", "notify.template", id, "PATCH", map[string]any{"code": out.Code, "channel": out.Channel, "locale": out.Locale}); err != nil {
		return notify.Template{}, err
	}
	return out, tx.Commit(ctx)
}

const deliveryColumns = `id,app_id,message_id,user_id,template_id,channel,COALESCE(target_hint,''),COALESCE(provider,''),COALESCE(provider_message_id,''),status,attempt_count,max_attempts,scheduled_at,next_attempt_at,sent_at,COALESCE(last_error,''),retryable,retry_risk,created_at,updated_at`

func scanDelivery(row scanner) (notify.Delivery, error) {
	var out notify.Delivery
	var lastError string
	err := row.Scan(&out.ID, &out.AppID, &out.MessageID, &out.UserID, &out.TemplateID, &out.Channel, &out.TargetHint, &out.Provider, &out.ProviderMessageID, &out.Status, &out.AttemptCount, &out.MaxAttempts, &out.ScheduledAt, &out.NextAttemptAt, &out.SentAt, &lastError, &out.Retryable, &out.RetryRisk, &out.CreatedAt, &out.UpdatedAt)
	if err == nil && lastError != "" {
		out.ErrorCode = "PROVIDER_DELIVERY_FAILED"
		out.ErrorSummary = safeSummary(lastError)
	}
	if err == nil {
		out.Retryable = out.Retryable && out.Status == "failed" && out.AttemptCount < out.MaxAttempts
	}
	return out, err
}

func (r *Postgres) ListDeliveries(ctx context.Context, tenantID uuid.UUID, f notify.PageFilter) (notify.DeliveryPage, error) {
	args := []any{tenantID}
	where := []string{"tenant_id=$1"}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	if f.Query != "" {
		add("(COALESCE(target_hint,'') ILIKE '%%'||$%[1]d||'%%' OR COALESCE(provider,'') ILIKE '%%'||$%[1]d||'%%')", f.Query)
	}
	if f.Status != "" {
		add("status=$%d", f.Status)
	}
	if f.Channel != "" {
		add("channel=$%d", f.Channel)
	}
	condition := strings.Join(where, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM notify.deliveries WHERE "+condition, args...).Scan(&total); err != nil {
		return notify.DeliveryPage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, fmt.Sprintf("SELECT %s FROM notify.deliveries WHERE %s ORDER BY created_at DESC,id DESC LIMIT $%d OFFSET $%d", deliveryColumns, condition, len(args)-1, len(args)), args...)
	if err != nil {
		return notify.DeliveryPage{}, err
	}
	defer rows.Close()
	items := make([]notify.Delivery, 0)
	for rows.Next() {
		item, scanErr := scanDelivery(rows)
		if scanErr != nil {
			return notify.DeliveryPage{}, scanErr
		}
		items = append(items, item)
	}
	return notify.DeliveryPage{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, rows.Err()
}

func (r *Postgres) GetDelivery(ctx context.Context, tenantID, id uuid.UUID) (notify.Delivery, error) {
	out, err := scanDelivery(r.pool.QueryRow(ctx, fmt.Sprintf("SELECT %s FROM notify.deliveries WHERE tenant_id=$1 AND id=$2", deliveryColumns), tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return notify.Delivery{}, notify.ErrNotFound
	}
	return out, err
}

func (r *Postgres) RetryDelivery(ctx context.Context, p notify.Principal, id uuid.UUID, acknowledgeDuplicateRisk bool) (notify.Delivery, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return notify.Delivery{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	out, err := scanDelivery(tx.QueryRow(ctx, fmt.Sprintf(`UPDATE notify.deliveries SET status='pending',next_attempt_at=now(),last_error=NULL,retryable=false,retry_risk='none'
		WHERE tenant_id=$1 AND id=$2 AND status='failed' AND attempt_count<max_attempts
		AND (retryable OR $3::boolean) AND (retry_risk='none' OR $3::boolean) RETURNING %s`, deliveryColumns), p.TenantID, id, acknowledgeDuplicateRisk))
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.GetDelivery(ctx, p.TenantID, id); getErr != nil {
			return notify.Delivery{}, getErr
		}
		return notify.Delivery{}, notify.ErrRetryNotAllowed
	}
	if err != nil {
		return notify.Delivery{}, err
	}
	if r.river == nil {
		return notify.Delivery{}, notify.ErrDeliveryUnavailable
	}
	if _, err = r.river.InsertTx(ctx, tx, notify.DeliveryJobArgs{DeliveryID: id}, &river.InsertOpts{Queue: "notifications", MaxAttempts: int(out.MaxAttempts), UniqueOpts: river.UniqueOpts{ByArgs: true}}); err != nil {
		return notify.Delivery{}, err
	}
	if err = insertAudit(ctx, tx, p, "notify.delivery.retry", "notify.delivery", id, "POST", map[string]any{"status": "pending", "attempt_count": out.AttemptCount, "duplicate_risk_acknowledged": acknowledgeDuplicateRisk}); err != nil {
		return notify.Delivery{}, err
	}
	return out, tx.Commit(ctx)
}

const bindingColumns = `id,template_id,provider,external_template_id,COALESCE(sign_name,''),parameter_order,status,version,created_at,updated_at`

func scanBinding(row scanner) (notify.SMSTemplateBinding, error) {
	var out notify.SMSTemplateBinding
	err := row.Scan(&out.ID, &out.TemplateID, &out.Provider, &out.ExternalTemplateID, &out.SignName, &out.ParameterOrder, &out.Status, &out.Version, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *Postgres) ListSMSTemplateBindings(ctx context.Context, tenantID, templateID uuid.UUID) ([]notify.SMSTemplateBinding, error) {
	if _, err := r.GetTemplate(ctx, tenantID, templateID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf("SELECT %s FROM notify.sms_template_bindings WHERE tenant_id=$1 AND template_id=$2 ORDER BY provider", bindingColumns), tenantID, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]notify.SMSTemplateBinding, 0)
	for rows.Next() {
		item, scanErr := scanBinding(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Postgres) UpsertSMSTemplateBinding(ctx context.Context, p notify.Principal, templateID uuid.UUID, provider string, in notify.SMSTemplateBindingInput) (notify.SMSTemplateBinding, error) {
	template, err := r.GetTemplate(ctx, p.TenantID, templateID)
	if err != nil {
		return notify.SMSTemplateBinding{}, err
	}
	if template.Channel != "sms" {
		return notify.SMSTemplateBinding{}, notify.ErrInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return notify.SMSTemplateBinding{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	out, err := scanBinding(tx.QueryRow(ctx, fmt.Sprintf(`INSERT INTO notify.sms_template_bindings(tenant_id,template_id,provider,external_template_id,sign_name,parameter_order,status,version)
		VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8)
		ON CONFLICT(tenant_id,template_id,provider) DO UPDATE SET external_template_id=EXCLUDED.external_template_id,sign_name=EXCLUDED.sign_name,parameter_order=EXCLUDED.parameter_order,status=EXCLUDED.status,version=notify.sms_template_bindings.version+1
		RETURNING %s`, bindingColumns), p.TenantID, templateID, provider, in.ExternalTemplateID, in.SignName, in.ParameterOrder, in.Status, max(1, in.Version)))
	if err != nil {
		return notify.SMSTemplateBinding{}, err
	}
	if err = insertAudit(ctx, tx, p, "notify.template.binding.update", "notify.template", templateID, "PUT", map[string]any{"provider": provider, "status": out.Status, "version": out.Version}); err != nil {
		return notify.SMSTemplateBinding{}, err
	}
	return out, tx.Commit(ctx)
}

func (r *Postgres) DeleteSMSTemplateBinding(ctx context.Context, p notify.Principal, templateID uuid.UUID, provider string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `DELETE FROM notify.sms_template_bindings WHERE tenant_id=$1 AND template_id=$2 AND provider=$3`, p.TenantID, templateID, provider)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return notify.ErrNotFound
	}
	if err = insertAudit(ctx, tx, p, "notify.template.binding.delete", "notify.template", templateID, "DELETE", map[string]any{"provider": provider}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Postgres) CreateTestDelivery(ctx context.Context, p notify.Principal, in notify.CreateDelivery) (notify.Delivery, error) {
	if r.river == nil {
		return notify.Delivery{}, notify.ErrDeliveryUnavailable
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return notify.Delivery{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	out, err := scanDelivery(tx.QueryRow(ctx, fmt.Sprintf(`INSERT INTO notify.deliveries(
		tenant_id,template_id,channel,target_ciphertext,target_hash,target_hint,target_key_version,provider,rendered_subject,rendered_body,dedupe_key,payload_ciphertext,payload_key_version,status,max_attempts)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),$10,$11,$12,$13,'pending',5)
		ON CONFLICT(tenant_id,channel,dedupe_key) WHERE dedupe_key IS NOT NULL DO UPDATE SET updated_at=notify.deliveries.updated_at
		RETURNING %s`, deliveryColumns), p.TenantID, in.TemplateID, in.Channel, in.TargetCiphertext, in.TargetHash, in.TargetHint, in.TargetKeyVersion, in.Provider, in.RenderedSubject, in.RenderedBody, in.DedupeKey, in.PayloadCiphertext, in.PayloadKeyVersion))
	if err != nil {
		return notify.Delivery{}, err
	}
	if _, err = r.river.InsertTx(ctx, tx, notify.DeliveryJobArgs{DeliveryID: out.ID}, &river.InsertOpts{Queue: "notifications", MaxAttempts: int(out.MaxAttempts), UniqueOpts: river.UniqueOpts{ByArgs: true}}); err != nil {
		return notify.Delivery{}, err
	}
	if err = insertAudit(ctx, tx, p, "notify.template.test", "notify.template", in.TemplateID, "POST", map[string]any{"delivery_id": out.ID, "channel": in.Channel, "provider": in.Provider, "target_hint": in.TargetHint}); err != nil {
		return notify.Delivery{}, err
	}
	return out, tx.Commit(ctx)
}

func insertAudit(ctx context.Context, tx pgx.Tx, p notify.Principal, action, resource string, id uuid.UUID, method string, after any) error {
	raw, _ := json.Marshal(after)
	tenant, user, session, resourceID := p.TenantID, p.UserID, p.SessionID, id.String()
	path := "/admin-api/v1/" + strings.ReplaceAll(resource, ".", "/") + "/{id}"
	status := int32(200)
	var ip *netip.Addr
	if parsed, err := netip.ParseAddr(p.IPAddress); err == nil {
		ip = &parsed
	}
	var ua *string
	if value := strings.TrimSpace(p.UserAgent); value != "" {
		ua = &value
	}
	_, err := tx.Exec(ctx, `INSERT INTO audit.operation_logs(
		tenant_id,user_id,session_id,request_id,module_code,action_name,resource_type,resource_id,
		http_method,request_path,response_status,client_ip,user_agent,after_data,succeeded)
		VALUES($1,$2,$3,$4,'notify',$5,$6,$7,$8,$9,$10,$11,$12,$13,true)`,
		&tenant, &user, &session, p.RequestID, action, &resource, &resourceID, &method, &path, &status, ip, ua, raw)
	return err
}

func (r *Postgres) scopedApp(ctx context.Context, tenantID, appID uuid.UUID) (uuid.UUID, error) {
	if appID == uuid.Nil {
		err := r.pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default AND status='active'`, tenantID).Scan(&appID)
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, notify.ErrNotFound
		}
		return appID, err
	}
	var found uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND id=$2 AND status='active'`, tenantID, appID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, notify.ErrNotFound
	}
	return found, err
}

func emailHint(value string) string {
	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return ""
	}
	visible := []rune(parts[0])
	prefix := string(visible[:1])
	return prefix + "***@" + parts[1]
}

func safeSummary(value string) string {
	value = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if r < 32 {
			return -1
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	if runes := []rune(value); len(runes) > 500 {
		value = string(runes[:500])
	}
	return value
}

func isUnique(err error) bool { return strings.Contains(err.Error(), "SQLSTATE 23505") }
