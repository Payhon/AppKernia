package worker

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OperationsMaintenance keeps the domain-safe task projection aligned with
// River, rolls up notification outcomes and applies the documented retention
// policy. It never reads or copies River args/errors.
type OperationsMaintenance struct {
	pool        *pgxpool.Pool
	environment string
	interval    time.Duration
}

func NewOperationsMaintenance(pool *pgxpool.Pool, environment string, interval time.Duration) *OperationsMaintenance {
	if interval < time.Minute {
		interval = time.Hour
	}
	return &OperationsMaintenance{pool: pool, environment: normalizePushEnvironment(environment), interval: interval}
}

func (m *OperationsMaintenance) Run(ctx context.Context) {
	_ = m.RunOnce(ctx)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = m.RunOnce(ctx)
		}
	}
}

func (m *OperationsMaintenance) RunOnce(ctx context.Context) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `UPDATE jobs.task_runs task SET
		status=CASE job.state::text
			WHEN 'scheduled' THEN 'scheduled'
			WHEN 'available' THEN 'queued'
			WHEN 'running' THEN 'running'
			WHEN 'retryable' THEN 'retry_wait'
			WHEN 'completed' THEN CASE WHEN task.status='failed' THEN 'failed' ELSE 'succeeded' END
			WHEN 'cancelled' THEN 'cancelled'
			WHEN 'discarded' THEN 'failed'
			ELSE task.status END,
		attempt_count=GREATEST(task.attempt_count,job.attempt),
		started_at=COALESCE(task.started_at,job.attempted_at),
		finalized_at=COALESCE(task.finalized_at,job.finalized_at)
		FROM river_job job WHERE task.river_job_id=job.id AND task.status IN ('scheduled','queued','running','retry_wait')`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO notify.delivery_daily_metrics(
		metric_date,tenant_id,app_id,environment,channel,provider,message_category,outcome,skip_reason,event_count)
		SELECT d.created_at::date,d.tenant_id,d.app_id,d.delivery_environment,d.channel,COALESCE(d.provider,''),
		COALESCE(d.metadata->>'push_category',''),outcome,'',count(*) FROM notify.deliveries d
		CROSS JOIN LATERAL (VALUES
			(CASE WHEN d.provider_result='accepted' THEN 'accepted' WHEN d.provider_result='invalid_token' THEN 'invalid_token' WHEN d.status='failed' AND NOT d.retryable THEN 'failed' ELSE 'queued' END)
		) result(outcome)
		WHERE d.app_id IS NOT NULL AND d.created_at >= current_date-interval '91 days'
		GROUP BY d.created_at::date,d.tenant_id,d.app_id,d.delivery_environment,d.channel,COALESCE(d.provider,''),COALESCE(d.metadata->>'push_category',''),outcome
		ON CONFLICT(metric_date,tenant_id,app_id,environment,channel,provider,message_category,outcome,skip_reason)
		DO UPDATE SET event_count=EXCLUDED.event_count,updated_at=now()`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO notify.delivery_daily_metrics(
		metric_date,tenant_id,app_id,environment,channel,provider,message_category,outcome,skip_reason,event_count)
		SELECT d.opened_at::date,d.tenant_id,d.app_id,d.delivery_environment,d.channel,COALESCE(d.provider,''),
		COALESCE(d.metadata->>'push_category',''),'opened','',count(*) FROM notify.deliveries d
		WHERE d.app_id IS NOT NULL AND d.opened_at IS NOT NULL AND d.opened_at >= current_date-interval '91 days'
		GROUP BY d.opened_at::date,d.tenant_id,d.app_id,d.delivery_environment,d.channel,COALESCE(d.provider,''),COALESCE(d.metadata->>'push_category','')
		ON CONFLICT(metric_date,tenant_id,app_id,environment,channel,provider,message_category,outcome,skip_reason)
		DO UPDATE SET event_count=EXCLUDED.event_count,updated_at=now()`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO notify.delivery_daily_metrics(
		metric_date,tenant_id,app_id,environment,channel,provider,message_category,outcome,skip_reason,event_count)
		SELECT r.push_evaluated_at::date,r.tenant_id,r.app_id,COALESCE(r.push_environment,$1),'push','',m.push_category,'skipped',r.push_skip_reason,count(*)
		FROM notify.recipients r JOIN notify.messages m ON m.id=r.message_id AND m.tenant_id=r.tenant_id AND m.app_id=r.app_id
		WHERE r.push_skip_reason IS NOT NULL AND r.push_evaluated_at >= current_date-interval '91 days'
		GROUP BY r.push_evaluated_at::date,r.tenant_id,r.app_id,COALESCE(r.push_environment,$1),m.push_category,r.push_skip_reason
		ON CONFLICT(metric_date,tenant_id,app_id,environment,channel,provider,message_category,outcome,skip_reason)
		DO UPDATE SET event_count=EXCLUDED.event_count,updated_at=now()`, m.environment); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM jobs.task_runs WHERE finalized_at < now()-interval '90 days' AND status IN ('succeeded','failed','cancelled')`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE notify.deliveries SET last_error=NULL,error_code=NULL
		WHERE created_at < now()-interval '90 days' AND status IN ('sent','failed','cancelled') AND (last_error IS NOT NULL OR error_code IS NOT NULL)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM notify.delivery_daily_metrics WHERE metric_date < current_date-interval '13 months'`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
