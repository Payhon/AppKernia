package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
	"time"

	jobs "github.com/appkernia/appkernia/server/internal/modules/jobadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type Postgres struct {
	pool  *pgxpool.Pool
	river *river.Client[pgx.Tx]
}

func NewPostgres(pool *pgxpool.Pool, riverClient *river.Client[pgx.Tx]) *Postgres {
	return &Postgres{pool: pool, river: riverClient}
}

type scanner interface{ Scan(...any) error }

const scheduleColumns = `id,code,name,handler_key,cron_expression,time_zone,payload,queue_name,overlap_policy,misfire_policy,timeout_seconds,max_attempts,status,last_enqueued_at,next_run_at,created_at,updated_at`

func scanSchedule(row scanner) (jobs.Schedule, error) {
	var item jobs.Schedule
	if err := row.Scan(&item.ID, &item.Code, &item.Name, &item.HandlerKey, &item.CronExpression, &item.TimeZone, &item.Payload, &item.QueueName, &item.OverlapPolicy, &item.MisfirePolicy, &item.TimeoutSeconds, &item.MaxAttempts, &item.Status, &item.LastEnqueuedAt, &item.NextRunAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return jobs.Schedule{}, err
	}
	if item.Payload == nil {
		item.Payload = json.RawMessage(`{}`)
	}
	return item, nil
}

func (r *Postgres) ListSchedules(ctx context.Context, tenantID uuid.UUID, filter jobs.PageFilter) (jobs.SchedulePage, error) {
	args := []any{tenantID}
	where := []string{"tenant_id=$1"}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	if filter.Query != "" {
		add("(name ILIKE '%%' || $%[1]d || '%%' OR code::text ILIKE '%%' || $%[1]d || '%%' OR handler_key ILIKE '%%' || $%[1]d || '%%')", filter.Query)
	}
	if filter.Status != "" {
		add("status=$%d", filter.Status)
	}
	if filter.TimeZone != "" {
		add("time_zone=$%d", filter.TimeZone)
	}
	condition := strings.Join(where, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM jobs.schedules WHERE "+condition, args...).Scan(&total); err != nil {
		return jobs.SchedulePage{}, err
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := r.pool.Query(ctx, fmt.Sprintf("SELECT %s FROM jobs.schedules WHERE %s ORDER BY updated_at DESC,id DESC LIMIT $%d OFFSET $%d", scheduleColumns, condition, len(args)-1, len(args)), args...)
	if err != nil {
		return jobs.SchedulePage{}, err
	}
	defer rows.Close()
	items := make([]jobs.Schedule, 0)
	for rows.Next() {
		item, scanErr := scanSchedule(rows)
		if scanErr != nil {
			return jobs.SchedulePage{}, scanErr
		}
		items = append(items, item)
	}
	return jobs.SchedulePage{Items: items, Page: filter.Page, PageSize: filter.PageSize, Total: total}, rows.Err()
}

func (r *Postgres) GetSchedule(ctx context.Context, tenantID, id uuid.UUID) (jobs.Schedule, error) {
	item, err := scanSchedule(r.pool.QueryRow(ctx, "SELECT "+scheduleColumns+" FROM jobs.schedules WHERE tenant_id=$1 AND id=$2", tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return jobs.Schedule{}, jobs.ErrNotFound
	}
	return item, err
}

func (r *Postgres) CreateSchedule(ctx context.Context, p jobs.Principal, input jobs.ScheduleInput, nextRun time.Time) (jobs.Schedule, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return jobs.Schedule{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := scanSchedule(tx.QueryRow(ctx, `INSERT INTO jobs.schedules(tenant_id,code,name,handler_key,cron_expression,time_zone,payload,queue_name,overlap_policy,misfire_policy,timeout_seconds,max_attempts,status,next_run_at,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'active',$13,$14) RETURNING `+scheduleColumns,
		p.TenantID, input.Code, input.Name, input.HandlerKey, input.CronExpression, input.TimeZone, input.Payload, input.QueueName, input.OverlapPolicy, input.MisfirePolicy, input.TimeoutSeconds, input.MaxAttempts, nextRun, p.UserID))
	if isUnique(err) {
		return jobs.Schedule{}, jobs.ErrConflict
	}
	if err != nil {
		return jobs.Schedule{}, err
	}
	if err = insertAudit(ctx, tx, p, "jobs.schedule.create", item.ID, "POST", map[string]any{"code": item.Code, "handler_key": item.HandlerKey, "status": item.Status}); err != nil {
		return jobs.Schedule{}, err
	}
	return item, tx.Commit(ctx)
}

func (r *Postgres) UpdateSchedule(ctx context.Context, p jobs.Principal, id uuid.UUID, input jobs.ScheduleInput, nextRun time.Time) (jobs.Schedule, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return jobs.Schedule{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := scanSchedule(tx.QueryRow(ctx, `UPDATE jobs.schedules SET code=$1,name=$2,handler_key=$3,cron_expression=$4,time_zone=$5,payload=$6,queue_name=$7,overlap_policy=$8,misfire_policy=$9,timeout_seconds=$10,max_attempts=$11,next_run_at=CASE WHEN status='active' THEN $12 ELSE NULL END
		WHERE tenant_id=$13 AND id=$14 AND status<>'disabled' RETURNING `+scheduleColumns,
		input.Code, input.Name, input.HandlerKey, input.CronExpression, input.TimeZone, input.Payload, input.QueueName, input.OverlapPolicy, input.MisfirePolicy, input.TimeoutSeconds, input.MaxAttempts, nextRun, p.TenantID, id))
	if isUnique(err) {
		return jobs.Schedule{}, jobs.ErrConflict
	}
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.GetSchedule(ctx, p.TenantID, id); getErr != nil {
			return jobs.Schedule{}, getErr
		}
		return jobs.Schedule{}, jobs.ErrTransition
	}
	if err != nil {
		return jobs.Schedule{}, err
	}
	if err = insertAudit(ctx, tx, p, "jobs.schedule.update", id, "PATCH", map[string]any{"code": item.Code, "handler_key": item.HandlerKey, "status": item.Status}); err != nil {
		return jobs.Schedule{}, err
	}
	return item, tx.Commit(ctx)
}

func (r *Postgres) ChangeStatus(ctx context.Context, p jobs.Principal, id uuid.UUID, target string, nextRun *time.Time) (jobs.Schedule, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return jobs.Schedule{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := scanSchedule(tx.QueryRow(ctx, "SELECT "+scheduleColumns+" FROM jobs.schedules WHERE tenant_id=$1 AND id=$2 FOR UPDATE", p.TenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return jobs.Schedule{}, jobs.ErrNotFound
	}
	if err != nil {
		return jobs.Schedule{}, err
	}
	valid := current.Status == "active" && target == "paused" || current.Status == "paused" && target == "active"
	if !valid {
		return jobs.Schedule{}, jobs.ErrTransition
	}
	item, err := scanSchedule(tx.QueryRow(ctx, "UPDATE jobs.schedules SET status=$1,next_run_at=$2 WHERE tenant_id=$3 AND id=$4 RETURNING "+scheduleColumns, target, nextRun, p.TenantID, id))
	if err != nil {
		return jobs.Schedule{}, err
	}
	action := "jobs.schedule.pause"
	if target == "active" {
		action = "jobs.schedule.resume"
	}
	if err = insertAudit(ctx, tx, p, action, id, "POST", map[string]any{"before_status": current.Status, "status": target}); err != nil {
		return jobs.Schedule{}, err
	}
	return item, tx.Commit(ctx)
}

func (r *Postgres) Execute(ctx context.Context, p jobs.Principal, id uuid.UUID, idempotencyKey string, now time.Time) (jobs.Run, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return jobs.Run{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	schedule, err := scanSchedule(tx.QueryRow(ctx, "SELECT "+scheduleColumns+" FROM jobs.schedules WHERE tenant_id=$1 AND id=$2 FOR UPDATE", p.TenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return jobs.Run{}, jobs.ErrNotFound
	}
	if err != nil {
		return jobs.Run{}, err
	}
	if schedule.Status == "disabled" {
		return jobs.Run{}, jobs.ErrExecutionDenied
	}
	if existing, existingErr := scanRun(tx.QueryRow(ctx, runSelect+" WHERE run.schedule_id=$1 AND run.idempotency_key=$2", id, idempotencyKey)); existingErr == nil {
		if err = tx.Commit(ctx); err != nil {
			return jobs.Run{}, err
		}
		return existing, nil
	} else if !errors.Is(existingErr, pgx.ErrNoRows) {
		return jobs.Run{}, existingErr
	}
	if schedule.OverlapPolicy != "allow" {
		rows, queryErr := tx.Query(ctx, `SELECT id,river_job_id FROM jobs.schedule_runs WHERE schedule_id=$1 AND status IN ('queued','running') FOR UPDATE`, id)
		if queryErr != nil {
			return jobs.Run{}, queryErr
		}
		var active []struct {
			id    uuid.UUID
			river *int64
		}
		for rows.Next() {
			var item struct {
				id    uuid.UUID
				river *int64
			}
			if queryErr = rows.Scan(&item.id, &item.river); queryErr != nil {
				rows.Close()
				return jobs.Run{}, queryErr
			}
			active = append(active, item)
		}
		rows.Close()
		if queryErr = rows.Err(); queryErr != nil {
			return jobs.Run{}, queryErr
		}
		if schedule.OverlapPolicy == "skip" && len(active) > 0 {
			return jobs.Run{}, jobs.ErrConflict
		}
		for _, previous := range active {
			if previous.river != nil {
				if _, cancelErr := r.river.JobCancelTx(ctx, tx, *previous.river); cancelErr != nil {
					return jobs.Run{}, cancelErr
				}
			}
			if _, queryErr = tx.Exec(ctx, "UPDATE jobs.schedule_runs SET status='cancelled',finished_at=$1 WHERE id=$2", now, previous.id); queryErr != nil {
				return jobs.Run{}, queryErr
			}
		}
	}
	var run jobs.Run
	if err = tx.QueryRow(ctx, `INSERT INTO jobs.schedule_runs(schedule_id,trigger_type,status,scheduled_at,idempotency_key) VALUES($1,'manual','queued',$2,$3) RETURNING id,schedule_id,trigger_type,status,attempt,scheduled_at,created_at`, id, now, idempotencyKey).
		Scan(&run.ID, &run.ScheduleID, &run.TriggerType, &run.Status, &run.Attempt, &run.ScheduledAt, &run.CreatedAt); err != nil {
		return jobs.Run{}, err
	}
	inserted, err := r.river.InsertTx(ctx, tx, jobs.ScheduleRunArgs{RunID: run.ID}, &river.InsertOpts{Queue: schedule.QueueName, MaxAttempts: int(schedule.MaxAttempts)})
	if err != nil {
		return jobs.Run{}, err
	}
	run.RiverJobID = &inserted.Job.ID
	if _, err = tx.Exec(ctx, "UPDATE jobs.schedule_runs SET river_job_id=$1 WHERE id=$2", inserted.Job.ID, run.ID); err != nil {
		return jobs.Run{}, err
	}
	if _, err = tx.Exec(ctx, "UPDATE jobs.schedules SET last_enqueued_at=$1 WHERE tenant_id=$2 AND id=$3", now, p.TenantID, id); err != nil {
		return jobs.Run{}, err
	}
	if err = insertAudit(ctx, tx, p, "jobs.schedule.execute", id, "POST", map[string]any{"run_id": run.ID, "handler_key": schedule.HandlerKey, "trigger_type": "manual"}); err != nil {
		return jobs.Run{}, err
	}
	return run, tx.Commit(ctx)
}

const runSelect = `SELECT run.id,run.schedule_id,run.river_job_id,run.trigger_type,run.status,run.attempt,run.scheduled_at,run.started_at,run.finished_at,COALESCE(run.worker_id,''),run.output,COALESCE(run.error_code,''),COALESCE(run.error_message,''),run.created_at FROM jobs.schedule_runs run`

func scanRun(row scanner) (jobs.Run, error) {
	var item jobs.Run
	var errorMessage string
	if err := row.Scan(&item.ID, &item.ScheduleID, &item.RiverJobID, &item.TriggerType, &item.Status, &item.Attempt, &item.ScheduledAt, &item.StartedAt, &item.FinishedAt, &item.WorkerID, &item.Output, &item.ErrorCode, &errorMessage, &item.CreatedAt); err != nil {
		return jobs.Run{}, err
	}
	item.ErrorSummary = safeSummary(errorMessage)
	if len(item.Output) == 0 || string(item.Output) == "null" {
		item.Output = nil
	}
	return item, nil
}

func (r *Postgres) ListRuns(ctx context.Context, tenantID, scheduleID uuid.UUID, filter jobs.PageFilter) (jobs.RunPage, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM jobs.schedules WHERE tenant_id=$1 AND id=$2)", tenantID, scheduleID).Scan(&exists); err != nil {
		return jobs.RunPage{}, err
	}
	if !exists {
		return jobs.RunPage{}, jobs.ErrNotFound
	}
	args := []any{tenantID, scheduleID}
	where := []string{"schedule.tenant_id=$1", "run.schedule_id=$2"}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("run.status=$%d", len(args)))
	}
	condition := strings.Join(where, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM jobs.schedule_runs run JOIN jobs.schedules schedule ON schedule.id=run.schedule_id WHERE "+condition, args...).Scan(&total); err != nil {
		return jobs.RunPage{}, err
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := r.pool.Query(ctx, runSelect+" JOIN jobs.schedules schedule ON schedule.id=run.schedule_id WHERE "+condition+fmt.Sprintf(" ORDER BY run.scheduled_at DESC,run.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return jobs.RunPage{}, err
	}
	defer rows.Close()
	items := make([]jobs.Run, 0)
	for rows.Next() {
		item, scanErr := scanRun(rows)
		if scanErr != nil {
			return jobs.RunPage{}, scanErr
		}
		items = append(items, item)
	}
	return jobs.RunPage{Items: items, Page: filter.Page, PageSize: filter.PageSize, Total: total}, rows.Err()
}

func insertAudit(ctx context.Context, tx pgx.Tx, p jobs.Principal, action string, id uuid.UUID, method string, after any) error {
	raw, err := json.Marshal(after)
	if err != nil {
		return err
	}
	path := "/admin-api/v1/job-schedules/" + id.String()
	status := int32(200)
	resource := "jobs.schedule"
	resourceID := id.String()
	var ip *netip.Addr
	if parsed, parseErr := netip.ParseAddr(p.IPAddress); parseErr == nil {
		ip = &parsed
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit.operation_logs(tenant_id,user_id,session_id,request_id,module_code,action_name,permission_code,resource_type,resource_id,http_method,request_path,response_status,client_ip,user_agent,after_data,succeeded)
		VALUES($1,$2,nullif($3,'00000000-0000-0000-0000-000000000000'::uuid),$4,'jobs',$5,$5,$6,$7,$8,$9,$10,$11,nullif($12,''),$13,true)`,
		p.TenantID, p.UserID, p.SessionID, p.RequestID, action, resource, resourceID, method, path, status, ip, strings.TrimSpace(p.UserAgent), raw)
	return err
}

var secretPattern = regexp.MustCompile(`(?i)(bearer\s+|token[=:]\s*|password[=:]\s*|secret[=:]\s*)\S+`)

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
	value = secretPattern.ReplaceAllString(value, "[REDACTED]")
	value = strings.Join(strings.Fields(value), " ")
	if runes := []rune(value); len(runes) > 500 {
		value = string(runes[:500])
	}
	return value
}

func isUnique(err error) bool { return err != nil && strings.Contains(err.Error(), "SQLSTATE 23505") }
