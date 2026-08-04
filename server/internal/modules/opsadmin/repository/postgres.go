package repository

import (
	"context"
	"runtime"
	"time"

	ops "github.com/appkernia/appkernia/server/internal/modules/opsadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct{ ObjectStorageConfigured bool }
type Postgres struct {
	pool   *pgxpool.Pool
	config Config
	now    func() time.Time
}

func NewPostgres(p *pgxpool.Pool, c Config) *Postgres {
	return &Postgres{pool: p, config: c, now: time.Now}
}
func (r *Postgres) Health(ctx context.Context) ops.Health {
	checked := r.now().UTC()
	start := time.Now()
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	e := r.pool.Ping(pingCtx)
	dbStatus, detail := "ready", "ops.health.ok"
	if e != nil {
		dbStatus, detail = "unavailable", "ops.health.probe_failed"
	}
	deps := []ops.Dependency{{Code: "api", Status: "ready", DetailCode: "ops.health.ok", CheckedAt: checked}, {Code: "postgresql", Status: dbStatus, LatencyMS: time.Since(start).Milliseconds(), DetailCode: detail, CheckedAt: checked}, {Code: "redis", Status: "not_configured", DetailCode: "ops.health.not_configured", CheckedAt: checked}}
	storeStatus := "not_configured"
	if r.config.ObjectStorageConfigured {
		storeStatus = "ready"
	}
	deps = append(deps, ops.Dependency{Code: "object_storage", Status: storeStatus, DetailCode: "ops.health." + storeStatus, CheckedAt: checked})
	status := "ready"
	if dbStatus != "ready" {
		status = "unavailable"
	}
	return ops.Health{Status: status, Dependencies: deps, CheckedAt: checked}
}
func (r *Postgres) Runtime(ctx context.Context, tenant uuid.UUID, started time.Time) (ops.RuntimeSummary, error) {
	now := r.now().UTC()
	out := ops.RuntimeSummary{AppVersion: "0.1.0", GoVersion: runtime.Version(), UptimeSeconds: int64(now.Sub(started).Seconds()), Modules: []ops.Module{}, Queue: ops.Queue{Status: "unknown"}, GeneratedAt: now}
	rows, e := r.pool.Query(ctx, `SELECT code::text,version,status FROM sys.modules ORDER BY code`)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var m ops.Module
		if e = rows.Scan(&m.Code, &m.Version, &m.Status); e != nil {
			return out, e
		}
		out.Modules = append(out.Modules, m)
	}
	if e = rows.Err(); e != nil {
		return out, e
	}
	var heartbeat *time.Time
	e = r.pool.QueryRow(ctx, `SELECT max(updated_at) FROM river_queue`).Scan(&heartbeat)
	if e != nil {
		return out, e
	}
	out.Queue.LastHeartbeatAt = heartbeat
	if heartbeat != nil && now.Sub(*heartbeat) <= 45*time.Second {
		out.Queue.Status = "ready"
	}
	e = r.pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE state='available'),count(*) FILTER(WHERE state='running'),count(*) FILTER(WHERE state='retryable'),count(*) FILTER(WHERE state='scheduled'),count(*) FILTER(WHERE state='completed'),count(*) FILTER(WHERE state IN ('discarded','cancelled')) FROM river_job`).Scan(&out.Queue.Available, &out.Queue.Running, &out.Queue.Retryable, &out.Queue.Scheduled, &out.Queue.Completed, &out.Queue.Failed)
	if e != nil {
		return out, e
	}
	e = r.pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE r.status='queued'),count(*) FILTER(WHERE r.status='running'),count(*) FILTER(WHERE r.status='succeeded'),count(*) FILTER(WHERE r.status='failed'),count(*) FILTER(WHERE r.status='skipped') FROM jobs.schedule_runs r JOIN jobs.schedules s ON s.id=r.schedule_id WHERE s.tenant_id=$1 AND r.created_at>=now()-interval '24 hours'`, tenant).Scan(&out.ScheduleRuns24H.Queued, &out.ScheduleRuns24H.Running, &out.ScheduleRuns24H.Succeeded, &out.ScheduleRuns24H.Failed, &out.ScheduleRuns24H.Skipped)
	return out, e
}
