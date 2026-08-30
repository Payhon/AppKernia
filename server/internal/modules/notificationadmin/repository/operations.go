package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/appkernia/appkernia/server/internal/platform/jobqueue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func deliveryOperationFilter(tenantID, appID uuid.UUID, f notify.OperationsFilter) (string, []any) {
	args := []any{tenantID, appID, f.From, f.To}
	where := []string{"d.tenant_id=$1", "d.app_id=$2", "d.created_at >= $3", "d.created_at < $4"}
	add := func(column string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf("%s=$%d", column, len(args)))
	}
	if f.Environment != "" {
		add("d.delivery_environment", f.Environment)
	}
	if f.Channel != "" {
		add("d.channel", f.Channel)
	}
	if f.Provider != "" {
		add("d.provider", f.Provider)
	}
	if f.Category != "" {
		args = append(args, f.Category)
		where = append(where, fmt.Sprintf("COALESCE(d.metadata->>'push_category','')=$%d", len(args)))
	}
	return strings.Join(where, " AND "), args
}

func (r *Postgres) OperationsSummary(ctx context.Context, tenantID, appID uuid.UUID, f notify.OperationsFilter) (notify.OperationsSummary, error) {
	if _, err := r.scopedApp(ctx, tenantID, appID); err != nil {
		return notify.OperationsSummary{}, err
	}
	condition, args := deliveryOperationFilter(tenantID, appID, f)
	var out notify.OperationsSummary
	err := r.pool.QueryRow(ctx, `SELECT
		count(*) FILTER(WHERE d.provider_result='accepted'),
		count(*) FILTER(WHERE d.status='failed' AND NOT d.retryable),
		count(*) FILTER(WHERE d.provider_result='invalid_token'),
		count(*) FILTER(WHERE d.opened_at IS NOT NULL)
		FROM notify.deliveries d WHERE `+condition, args...).Scan(&out.Accepted, &out.Failed, &out.InvalidTokens, &out.Opened)
	if err != nil {
		return out, err
	}
	taskArgs := []any{tenantID, appID, f.From, f.To}
	taskWhere := []string{"tenant_id=$1", "app_id=$2", "created_at >= $3", "created_at < $4", "module_code='notify'"}
	if f.TaskKind != "" {
		taskArgs = append(taskArgs, f.TaskKind)
		taskWhere = append(taskWhere, fmt.Sprintf("task_kind=$%d", len(taskArgs)))
	}
	err = r.pool.QueryRow(ctx, `SELECT
		count(*) FILTER(WHERE status IN ('queued','scheduled')),
		count(*) FILTER(WHERE status='running'),
		count(*) FILTER(WHERE status='retry_wait'),
		COALESCE(extract(epoch FROM (now()-min(scheduled_at) FILTER(WHERE status IN ('queued','scheduled'))))::bigint,0),
		COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY extract(epoch FROM (started_at-scheduled_at))*1000) FILTER(WHERE started_at IS NOT NULL),0)::bigint
		FROM jobs.task_runs WHERE `+strings.Join(taskWhere, " AND "), taskArgs...).Scan(&out.Queued, &out.Running, &out.RetryWaiting, &out.OldestWaitingSecs, &out.P95QueueDelayMS)
	if err != nil {
		return out, err
	}
	if (f.Channel != "" && f.Channel != "push") || f.Provider != "" {
		out.Skipped = 0
		if out.Accepted > 0 {
			out.OpenRate = float64(out.Opened) / float64(out.Accepted)
		}
		out.HasUnfinishedWork = out.Queued+out.Running+out.RetryWaiting > 0
		return out, nil
	}
	skipArgs := []any{tenantID, appID, f.From, f.To}
	skipWhere := []string{"r.tenant_id=$1", "r.app_id=$2", "r.push_evaluated_at >= $3", "r.push_evaluated_at < $4", "r.push_skip_reason IS NOT NULL"}
	if f.Category != "" {
		skipArgs = append(skipArgs, f.Category)
		skipWhere = append(skipWhere, fmt.Sprintf("m.push_category=$%d", len(skipArgs)))
	}
	if f.Environment != "" {
		skipArgs = append(skipArgs, f.Environment)
		skipWhere = append(skipWhere, fmt.Sprintf("r.push_environment=$%d", len(skipArgs)))
	}
	if err = r.pool.QueryRow(ctx, `SELECT count(*) FROM notify.recipients r JOIN notify.messages m ON m.id=r.message_id AND m.tenant_id=r.tenant_id AND m.app_id=r.app_id WHERE `+strings.Join(skipWhere, " AND "), skipArgs...).Scan(&out.Skipped); err != nil {
		return out, err
	}
	if out.Accepted > 0 {
		out.OpenRate = float64(out.Opened) / float64(out.Accepted)
	}
	out.HasUnfinishedWork = out.Queued+out.Running+out.RetryWaiting > 0
	return out, nil
}

func (r *Postgres) OperationsTrends(ctx context.Context, tenantID, appID uuid.UUID, f notify.OperationsFilter) ([]notify.TrendBucket, error) {
	if _, err := r.scopedApp(ctx, tenantID, appID); err != nil {
		return nil, err
	}
	condition, args := deliveryOperationFilter(tenantID, appID, f)
	rows, err := r.pool.Query(ctx, `SELECT date_trunc('day',d.created_at) AS bucket,
		count(*) FILTER(WHERE d.provider_result='accepted'),
		count(*) FILTER(WHERE d.status='failed' AND NOT d.retryable),
		count(*) FILTER(WHERE d.provider_result='invalid_token'),
		count(*) FILTER(WHERE d.opened_at IS NOT NULL)
		FROM notify.deliveries d WHERE `+condition+` GROUP BY bucket ORDER BY bucket`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]notify.TrendBucket, 0)
	byDay := map[string]int{}
	for rows.Next() {
		var item notify.TrendBucket
		if err = rows.Scan(&item.Bucket, &item.Accepted, &item.Failed, &item.InvalidTokens, &item.Opened); err != nil {
			return nil, err
		}
		item.Bucket = item.Bucket.UTC()
		byDay[item.Bucket.Format("2006-01-02")] = len(items)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	skipArgs := []any{tenantID, appID, f.From, f.To}
	skipWhere := []string{"r.tenant_id=$1", "r.app_id=$2", "r.push_evaluated_at >= $3", "r.push_evaluated_at < $4", "r.push_skip_reason IS NOT NULL"}
	if f.Category != "" {
		skipArgs = append(skipArgs, f.Category)
		skipWhere = append(skipWhere, fmt.Sprintf("m.push_category=$%d", len(skipArgs)))
	}
	if f.Environment != "" {
		skipArgs = append(skipArgs, f.Environment)
		skipWhere = append(skipWhere, fmt.Sprintf("r.push_environment=$%d", len(skipArgs)))
	}
	if (f.Channel != "" && f.Channel != "push") || f.Provider != "" {
		return items, nil
	}
	skipRows, err := r.pool.Query(ctx, `SELECT date_trunc('day',r.push_evaluated_at),count(*)
		FROM notify.recipients r JOIN notify.messages m ON m.id=r.message_id AND m.tenant_id=r.tenant_id AND m.app_id=r.app_id
		WHERE `+strings.Join(skipWhere, " AND ")+` GROUP BY 1 ORDER BY 1`, skipArgs...)
	if err != nil {
		return nil, err
	}
	defer skipRows.Close()
	for skipRows.Next() {
		var bucket time.Time
		var count int64
		if err = skipRows.Scan(&bucket, &count); err != nil {
			return nil, err
		}
		key := bucket.UTC().Format("2006-01-02")
		if index, ok := byDay[key]; ok {
			items[index].Skipped = count
		} else {
			byDay[key] = len(items)
			items = append(items, notify.TrendBucket{Bucket: bucket.UTC(), Skipped: count})
		}
	}
	if err = skipRows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Bucket.Before(items[j].Bucket) })
	return items, nil
}

const messageRunColumns = `run.id,run.message_id,m.title,m.push_category,run.trigger_type,run.status,
run.recipient_count,run.evaluated_count,run.delivery_count,run.accepted_count,run.failed_count,
run.invalid_token_count,run.opened_count,run.skipped_count,run.started_at,run.completed_at,run.created_at,run.updated_at`

func scanMessageRun(row scanner) (notify.MessageRun, error) {
	var out notify.MessageRun
	err := row.Scan(&out.ID, &out.MessageID, &out.MessageTitle, &out.Category, &out.TriggerType, &out.Status,
		&out.RecipientCount, &out.EvaluatedCount, &out.DeliveryCount, &out.AcceptedCount, &out.FailedCount,
		&out.InvalidTokenCount, &out.OpenedCount, &out.SkippedCount, &out.StartedAt, &out.CompletedAt, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *Postgres) ListMessageRuns(ctx context.Context, tenantID, appID uuid.UUID, f notify.OperationsFilter) (notify.MessageRunPage, error) {
	args := []any{tenantID, appID, f.From, f.To}
	where := []string{"run.tenant_id=$1", "run.app_id=$2", "run.created_at >= $3", "run.created_at < $4"}
	if f.Status != "" {
		args = append(args, f.Status)
		where = append(where, fmt.Sprintf("run.status=$%d", len(args)))
	}
	if f.Category != "" {
		args = append(args, f.Category)
		where = append(where, fmt.Sprintf("m.push_category=$%d", len(args)))
	}
	if f.Query != "" {
		args = append(args, f.Query)
		where = append(where, fmt.Sprintf("m.title ILIKE '%%'||$%d||'%%'", len(args)))
	}
	condition := strings.Join(where, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notify.message_runs run JOIN notify.messages m ON m.id=run.message_id WHERE `+condition, args...).Scan(&total); err != nil {
		return notify.MessageRunPage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, `SELECT `+messageRunColumns+` FROM notify.message_runs run JOIN notify.messages m ON m.id=run.message_id WHERE `+condition+fmt.Sprintf(" ORDER BY run.created_at DESC,run.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return notify.MessageRunPage{}, err
	}
	defer rows.Close()
	items := make([]notify.MessageRun, 0)
	for rows.Next() {
		item, scanErr := scanMessageRun(rows)
		if scanErr != nil {
			return notify.MessageRunPage{}, scanErr
		}
		items = append(items, item)
	}
	return notify.MessageRunPage{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, rows.Err()
}

func (r *Postgres) GetMessageRun(ctx context.Context, tenantID, appID, id uuid.UUID) (notify.MessageRun, error) {
	out, err := scanMessageRun(r.pool.QueryRow(ctx, `SELECT `+messageRunColumns+` FROM notify.message_runs run JOIN notify.messages m ON m.id=run.message_id WHERE run.tenant_id=$1 AND run.app_id=$2 AND run.id=$3`, tenantID, appID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return notify.MessageRun{}, notify.ErrNotFound
	}
	return out, err
}

const taskRunColumns = `task.id,task.task_kind,task.queue_name,task.resource_type,task.resource_id,task.correlation_id,
task.status,task.scheduled_at,task.started_at,task.finalized_at,task.next_retry_at,task.attempt_count,task.max_attempts,
COALESCE(task.last_result_class,''),COALESCE(task.last_error_code,''),COALESCE(task.last_error_summary,''),COALESCE(task.trace_id,''),
CASE WHEN task.status='failed' AND (
  (task.resource_type='notification_message' AND task.task_kind IN ('appkernia-message-publish','appkernia-push-fanout'))
  OR COALESCE(d.retryable,false)
  OR COALESCE(d.provider_result,'') IN ('throttled','transient','auth_config_error','unknown_after_write')
) THEN true ELSE false END,
COALESCE(d.retry_risk,'none'),task.created_at,task.updated_at`

func scanTaskRun(row scanner) (notify.TaskRun, error) {
	var out notify.TaskRun
	err := row.Scan(&out.ID, &out.TaskKind, &out.QueueName, &out.ResourceType, &out.ResourceID, &out.CorrelationID,
		&out.Status, &out.ScheduledAt, &out.StartedAt, &out.FinalizedAt, &out.NextRetryAt, &out.AttemptCount, &out.MaxAttempts,
		&out.LastResultClass, &out.LastErrorCode, &out.LastErrorSummary, &out.TraceID, &out.Retryable, &out.RetryRisk, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func taskOperationFilter(tenantID, appID uuid.UUID, f notify.OperationsFilter) (string, []any) {
	args := []any{tenantID, appID, f.From, f.To}
	where := []string{"task.tenant_id=$1", "task.app_id=$2", "task.module_code='notify'", "task.created_at >= $3", "task.created_at < $4"}
	add := func(column string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf("%s=$%d", column, len(args)))
	}
	if f.Status != "" {
		add("task.status", f.Status)
	}
	if f.TaskKind != "" {
		add("task.task_kind", f.TaskKind)
	}
	if f.Provider != "" {
		add("d.provider", f.Provider)
	}
	if f.Channel != "" {
		add("d.channel", f.Channel)
	}
	if f.Environment != "" {
		add("d.delivery_environment", f.Environment)
	}
	if f.Query != "" {
		args = append(args, f.Query)
		where = append(where, fmt.Sprintf("(task.task_kind ILIKE '%%'||$%[1]d||'%%' OR COALESCE(task.last_error_code,'') ILIKE '%%'||$%[1]d||'%%')", len(args)))
	}
	return strings.Join(where, " AND "), args
}

func (r *Postgres) ListTaskRuns(ctx context.Context, tenantID, appID uuid.UUID, f notify.OperationsFilter) (notify.TaskRunPage, error) {
	condition, args := taskOperationFilter(tenantID, appID, f)
	join := ` LEFT JOIN notify.deliveries d ON task.resource_type='notification_delivery' AND d.id=task.resource_id AND d.tenant_id=task.tenant_id AND d.app_id=task.app_id`
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM jobs.task_runs task`+join+` WHERE `+condition, args...).Scan(&total); err != nil {
		return notify.TaskRunPage{}, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.pool.Query(ctx, `SELECT `+taskRunColumns+` FROM jobs.task_runs task`+join+` WHERE `+condition+fmt.Sprintf(" ORDER BY task.created_at DESC,task.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return notify.TaskRunPage{}, err
	}
	defer rows.Close()
	items := make([]notify.TaskRun, 0)
	for rows.Next() {
		item, scanErr := scanTaskRun(rows)
		if scanErr != nil {
			return notify.TaskRunPage{}, scanErr
		}
		items = append(items, item)
	}
	return notify.TaskRunPage{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, rows.Err()
}

func (r *Postgres) GetTaskRun(ctx context.Context, tenantID, appID, id uuid.UUID) (notify.TaskRun, error) {
	join := ` LEFT JOIN notify.deliveries d ON task.resource_type='notification_delivery' AND d.id=task.resource_id AND d.tenant_id=task.tenant_id AND d.app_id=task.app_id`
	out, err := scanTaskRun(r.pool.QueryRow(ctx, `SELECT `+taskRunColumns+` FROM jobs.task_runs task`+join+` WHERE task.tenant_id=$1 AND task.app_id=$2 AND task.id=$3`, tenantID, appID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return notify.TaskRun{}, notify.ErrNotFound
	}
	if err != nil {
		return out, err
	}
	rows, err := r.pool.Query(ctx, `SELECT attempt_number,status,COALESCE(result_class,''),COALESCE(error_code,''),COALESCE(error_summary,''),
		COALESCE(external_request_id,''),COALESCE(trace_id,''),started_at,finished_at,duration_ms,next_retry_at
		FROM jobs.task_attempts WHERE tenant_id=$1 AND app_id=$2 AND task_run_id=$3 ORDER BY attempt_number DESC`, tenantID, appID, id)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.Attempts = make([]notify.TaskAttempt, 0)
	for rows.Next() {
		var attempt notify.TaskAttempt
		if err = rows.Scan(&attempt.AttemptNumber, &attempt.Status, &attempt.ResultClass, &attempt.ErrorCode, &attempt.ErrorSummary,
			&attempt.ExternalRequestID, &attempt.TraceID, &attempt.StartedAt, &attempt.FinishedAt, &attempt.DurationMS, &attempt.NextRetryAt); err != nil {
			return out, err
		}
		out.Attempts = append(out.Attempts, attempt)
	}
	return out, rows.Err()
}

func (r *Postgres) ListFailures(ctx context.Context, tenantID, appID uuid.UUID, f notify.OperationsFilter) (notify.FailurePage, error) {
	f.Status = "failed"
	page, err := r.ListTaskRuns(ctx, tenantID, appID, f)
	if err != nil {
		return notify.FailurePage{}, err
	}
	items := make([]notify.Failure, 0, len(page.Items))
	for _, task := range page.Items {
		item := notify.Failure{TaskRun: task}
		if task.ResourceID != nil && task.ResourceType == "notification_delivery" {
			_ = r.pool.QueryRow(ctx, `SELECT d.message_id,COALESCE(m.title,''),COALESCE(d.provider,''),d.channel
				FROM notify.deliveries d LEFT JOIN notify.messages m ON m.id=d.message_id
				WHERE d.tenant_id=$1 AND d.app_id=$2 AND d.id=$3`, tenantID, appID, *task.ResourceID).
				Scan(&item.MessageID, &item.MessageTitle, &item.Provider, &item.Channel)
		} else if task.ResourceID != nil && task.ResourceType == "notification_message" {
			item.MessageID = task.ResourceID
			_ = r.pool.QueryRow(ctx, `SELECT title FROM notify.messages WHERE tenant_id=$1 AND app_id=$2 AND id=$3`, tenantID, appID, *task.ResourceID).Scan(&item.MessageTitle)
		}
		items = append(items, item)
	}
	return notify.FailurePage{Items: items, Page: page.Page, PageSize: page.PageSize, Total: page.Total}, nil
}

func deliveryManualRetryRejection(retryable bool, risk, resultClass string, itemCount int, acknowledged bool) string {
	unknownAfterWrite := resultClass == "unknown_after_write" || risk == "duplicate_possible"
	if unknownAfterWrite && (itemCount != 1 || !acknowledged) {
		return "duplicate_risk_requires_single_acknowledgement"
	}
	manualRetryable := retryable || resultClass == "throttled" || resultClass == "transient" || resultClass == "auth_config_error"
	if !manualRetryable && !unknownAfterWrite {
		return "not_retryable"
	}
	return ""
}

func (r *Postgres) RetryTasks(ctx context.Context, p notify.Principal, input notify.RetryInput) ([]notify.RetryResult, error) {
	if r.queue == nil {
		return nil, notify.ErrDeliveryUnavailable
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	results := make([]notify.RetryResult, 0, len(input.Items))
	for _, requested := range input.Items {
		result := notify.RetryResult{TaskID: requested.TaskID}
		var kind, resourceType, status string
		var resourceID, correlationID *uuid.UUID
		err = tx.QueryRow(ctx, `SELECT task_kind,resource_type,resource_id,correlation_id,status FROM jobs.task_runs
			WHERE tenant_id=$1 AND app_id=$2 AND id=$3 FOR UPDATE`, p.TenantID, p.AppID, requested.TaskID).
			Scan(&kind, &resourceType, &resourceID, &correlationID, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			result.Reason = "not_found"
			results = append(results, result)
			continue
		}
		if err != nil {
			return nil, err
		}
		if status != "failed" || resourceID == nil {
			result.Reason = "not_retryable"
			results = append(results, result)
			continue
		}
		appID := p.AppID
		scope := jobqueue.Scope{TenantID: p.TenantID, AppID: &appID, ModuleCode: "notify", ResourceType: resourceType, ResourceID: resourceID, CorrelationID: correlationID}
		var spec jobqueue.Spec
		switch {
		case resourceType == "notification_delivery" && kind == notify.DeliveryJobKind:
			var retryable bool
			var risk, providerResult string
			var maxAttempts int
			err = tx.QueryRow(ctx, `SELECT retryable,retry_risk,COALESCE(provider_result,''),max_attempts
				FROM notify.deliveries WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND status='failed' FOR UPDATE`, p.TenantID, p.AppID, *resourceID).
				Scan(&retryable, &risk, &providerResult, &maxAttempts)
			if errors.Is(err, pgx.ErrNoRows) {
				result.Reason = "not_retryable"
				results = append(results, result)
				continue
			}
			if err != nil {
				return nil, err
			}
			if rejection := deliveryManualRetryRejection(retryable, risk, providerResult, len(input.Items), input.AcknowledgeDuplicateRisk); rejection != "" {
				result.Reason = rejection
				results = append(results, result)
				continue
			}
			if providerResult == "auth_config_error" {
				var ready bool
				err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notify.push_provider_configs c JOIN notify.deliveries d ON d.id=$3
					WHERE c.tenant_id=$1 AND c.app_id=$2 AND c.provider=d.provider AND c.environment=d.delivery_environment
					AND c.status='active' AND c.last_preflight_status='ready')`, p.TenantID, p.AppID, *resourceID).Scan(&ready)
				if err != nil {
					return nil, err
				}
				if !ready {
					result.Reason = "provider_preflight_required"
					results = append(results, result)
					continue
				}
			}
			if _, err = tx.Exec(ctx, `UPDATE notify.deliveries SET status='pending',next_attempt_at=now(),last_error=NULL,error_code=NULL,
				retryable=false,retry_risk='none',attempt_count=0 WHERE tenant_id=$1 AND app_id=$2 AND id=$3`, p.TenantID, p.AppID, *resourceID); err != nil {
				return nil, err
			}
			spec = jobqueue.Spec{Scope: scope, Args: notify.DeliveryJobArgs{DeliveryID: *resourceID}, Queue: "notifications", MaxAttempts: maxAttempts, UniqueByArgs: true}
		case resourceType == "notification_message" && (kind == notify.MessagePublishJobKind || kind == notify.PushFanoutJobKind):
			var messageStatus string
			var expiresAt *time.Time
			if err = tx.QueryRow(ctx, `SELECT status,expires_at FROM notify.messages WHERE tenant_id=$1 AND app_id=$2 AND id=$3 AND deleted_at IS NULL FOR UPDATE`, p.TenantID, p.AppID, *resourceID).Scan(&messageStatus, &expiresAt); err != nil {
				result.Reason = "message_unavailable"
				results = append(results, result)
				continue
			}
			if messageStatus == "cancelled" || expiresAt != nil && !expiresAt.After(time.Now().UTC()) || kind == notify.MessagePublishJobKind && messageStatus != "scheduled" || kind == notify.PushFanoutJobKind && messageStatus != "published" {
				result.Reason = "message_not_retryable"
				results = append(results, result)
				continue
			}
			args := jobqueue.Args(notify.PushFanoutJobArgs{TenantID: p.TenantID, AppID: p.AppID, MessageID: *resourceID})
			if kind == notify.MessagePublishJobKind {
				args = notify.MessagePublishJobArgs{TenantID: p.TenantID, AppID: p.AppID, MessageID: *resourceID}
			}
			spec = jobqueue.Spec{Scope: scope, Args: args, Queue: "notifications", MaxAttempts: 5, UniqueByArgs: false}
		default:
			result.Reason = "unsupported_task_kind"
			results = append(results, result)
			continue
		}
		newRun, enqueueErr := r.queue.EnqueueTx(ctx, tx, spec)
		if enqueueErr != nil {
			return nil, enqueueErr
		}
		if resourceType == "notification_delivery" {
			if _, err = tx.Exec(ctx, `UPDATE notify.deliveries SET task_run_id=$2 WHERE id=$1`, *resourceID, newRun.ID); err != nil {
				return nil, err
			}
		}
		if correlationID != nil {
			_, _ = tx.Exec(ctx, `UPDATE notify.message_runs SET status='running',completed_at=NULL WHERE tenant_id=$1 AND app_id=$2 AND id=$3`, p.TenantID, p.AppID, *correlationID)
		}
		result.Accepted, result.NewTaskID = true, &newRun.ID
		results = append(results, result)
	}
	if err = insertAudit(ctx, tx, p, "notify.task.retry", "notify.task", p.AppID, "POST", map[string]any{"requested_count": len(input.Items), "results": results}); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return results, nil
}
