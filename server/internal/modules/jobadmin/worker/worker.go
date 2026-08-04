package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	jobapp "github.com/appkernia/appkernia/server/internal/modules/jobadmin/application"
	jobs "github.com/appkernia/appkernia/server/internal/modules/jobadmin/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type RunWorker struct {
	river.WorkerDefaults[jobs.ScheduleRunArgs]
	pool *pgxpool.Pool
}

func NewRunWorker(pool *pgxpool.Pool) *RunWorker { return &RunWorker{pool: pool} }

func (w *RunWorker) Work(ctx context.Context, job *river.Job[jobs.ScheduleRunArgs]) error {
	var handlerKey string
	var payload json.RawMessage
	var timeoutSeconds int32
	var status string
	err := w.pool.QueryRow(ctx, `UPDATE jobs.schedule_runs run SET status='running',attempt=$2,started_at=COALESCE(started_at,now()),worker_id='appkernia-worker'
		FROM jobs.schedules schedule WHERE run.id=$1 AND schedule.id=run.schedule_id AND run.status IN ('queued','failed','running')
		RETURNING schedule.handler_key,schedule.payload,schedule.timeout_seconds,run.status`, job.Args.RunID, job.Attempt).
		Scan(&handlerKey, &payload, &timeoutSeconds, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(errors.New("JOBS.RUN.NOT_EXECUTABLE"))
	}
	if err != nil {
		return errors.New("JOBS.RUN.STATE_UPDATE_FAILED")
	}
	if _, ok := jobs.FindCompiledHandler(handlerKey); !ok {
		w.failRun(ctx, job.Args.RunID, "JOBS.HANDLER.UNKNOWN")
		return river.JobCancel(errors.New("JOBS.HANDLER.UNKNOWN"))
	}
	handlerContext, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	output, err := w.execute(handlerContext, handlerKey, payload)
	if err != nil {
		code := "JOBS.HANDLER.FAILED"
		if errors.Is(err, context.DeadlineExceeded) {
			code = "JOBS.HANDLER.TIMEOUT"
		}
		w.failRun(ctx, job.Args.RunID, code)
		return errors.New(code)
	}
	if _, err = w.pool.Exec(ctx, `UPDATE jobs.schedule_runs SET status='succeeded',finished_at=now(),output=$2,error_code=NULL,error_message=NULL WHERE id=$1`, job.Args.RunID, output); err != nil {
		return errors.New("JOBS.RUN.FINALIZE_FAILED")
	}
	return nil
}

func (w *RunWorker) execute(ctx context.Context, handlerKey string, _ json.RawMessage) (json.RawMessage, error) {
	switch handlerKey {
	case "system.health.snapshot":
		if err := w.pool.Ping(ctx); err != nil {
			return nil, errors.New("JOBS.HEALTH.UNAVAILABLE")
		}
		return json.RawMessage(`{"result_code":"JOBS.HEALTH.OK"}`), nil
	default:
		return nil, jobs.ErrHandlerUnknown
	}
}

func (w *RunWorker) failRun(ctx context.Context, id uuid.UUID, code string) {
	_, _ = w.pool.Exec(ctx, `UPDATE jobs.schedule_runs SET status='failed',finished_at=now(),error_code=$2,error_message=NULL WHERE id=$1`, id, code)
}

type Scheduler struct {
	pool     *pgxpool.Pool
	river    *river.Client[pgx.Tx]
	interval time.Duration
}

func NewScheduler(pool *pgxpool.Pool, riverClient *river.Client[pgx.Tx], interval time.Duration) *Scheduler {
	return &Scheduler{pool: pool, river: riverClient, interval: interval}
}

func (s *Scheduler) Run(ctx context.Context) {
	if s.interval <= 0 {
		s.interval = 15 * time.Second
	}
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		if err := s.enqueueDue(ctx, time.Now().UTC()); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("schedule scan failed", "error_code", "JOBS.SCHEDULER.SCAN_FAILED")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type dueSchedule struct {
	ID             uuid.UUID
	CronExpression string
	TimeZone       string
	QueueName      string
	OverlapPolicy  string
	MisfirePolicy  string
	MaxAttempts    int32
	NextRunAt      time.Time
}

func (s *Scheduler) enqueueDue(ctx context.Context, now time.Time) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT id,cron_expression,time_zone,queue_name,overlap_policy,misfire_policy,max_attempts,next_run_at
		FROM jobs.schedules WHERE status='active' AND next_run_at IS NOT NULL AND next_run_at<=$1 ORDER BY next_run_at,id FOR UPDATE SKIP LOCKED LIMIT 50`, now)
	if err != nil {
		return err
	}
	due := make([]dueSchedule, 0)
	for rows.Next() {
		var item dueSchedule
		if err = rows.Scan(&item.ID, &item.CronExpression, &item.TimeZone, &item.QueueName, &item.OverlapPolicy, &item.MisfirePolicy, &item.MaxAttempts, &item.NextRunAt); err != nil {
			rows.Close()
			return err
		}
		due = append(due, item)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	for _, schedule := range due {
		if err = s.enqueueSchedule(ctx, tx, schedule, now); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Scheduler) enqueueSchedule(ctx context.Context, tx pgx.Tx, schedule dueSchedule, now time.Time) error {
	preview, err := jobapp.PreviewCron(schedule.CronExpression, schedule.TimeZone, schedule.NextRunAt)
	if err != nil {
		_, updateErr := tx.Exec(ctx, `UPDATE jobs.schedules SET status='disabled',next_run_at=NULL WHERE id=$1`, schedule.ID)
		return updateErr
	}
	occurrences := []time.Time{schedule.NextRunAt}
	if schedule.MisfirePolicy == "ignore" && schedule.NextRunAt.Before(now.Add(-time.Minute)) {
		occurrences = nil
	}
	if schedule.MisfirePolicy == "catch_up" {
		for _, candidate := range preview.NextRuns {
			if candidate.After(now) || len(occurrences) >= 10 {
				break
			}
			occurrences = append(occurrences, candidate)
		}
	}
	last := schedule.NextRunAt
	if len(occurrences) > 0 {
		last = occurrences[len(occurrences)-1]
	}
	nextPreview, err := jobapp.PreviewCron(schedule.CronExpression, schedule.TimeZone, last)
	if err != nil {
		return err
	}
	nextRun := nextPreview.NextRuns[0]
	if schedule.MisfirePolicy != "catch_up" {
		for !nextRun.After(now) {
			nextPreview, err = jobapp.PreviewCron(schedule.CronExpression, schedule.TimeZone, nextRun)
			if err != nil {
				return err
			}
			nextRun = nextPreview.NextRuns[0]
		}
	}
	active, err := s.activeRuns(ctx, tx, schedule.ID)
	if err != nil {
		return err
	}
	if schedule.OverlapPolicy == "skip" && len(active) > 0 {
		occurrences = nil
	}
	if schedule.OverlapPolicy == "replace" {
		for _, run := range active {
			if run.riverJobID != nil {
				if _, err = s.river.JobCancelTx(ctx, tx, *run.riverJobID); err != nil {
					return err
				}
			}
			if _, err = tx.Exec(ctx, `UPDATE jobs.schedule_runs SET status='cancelled',finished_at=$2 WHERE id=$1`, run.id, now); err != nil {
				return err
			}
		}
	}
	var enqueued bool
	for _, occurrence := range occurrences {
		var runID uuid.UUID
		err = tx.QueryRow(ctx, `INSERT INTO jobs.schedule_runs(schedule_id,trigger_type,status,scheduled_at) VALUES($1,'schedule','queued',$2)
			ON CONFLICT (schedule_id,scheduled_at) WHERE trigger_type='schedule' DO NOTHING RETURNING id`, schedule.ID, occurrence).Scan(&runID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		inserted, insertErr := s.river.InsertTx(ctx, tx, jobs.ScheduleRunArgs{RunID: runID}, &river.InsertOpts{Queue: schedule.QueueName, MaxAttempts: int(schedule.MaxAttempts)})
		if insertErr != nil {
			return insertErr
		}
		if _, err = tx.Exec(ctx, `UPDATE jobs.schedule_runs SET river_job_id=$2 WHERE id=$1`, runID, inserted.Job.ID); err != nil {
			return err
		}
		enqueued = true
	}
	if enqueued {
		_, err = tx.Exec(ctx, `UPDATE jobs.schedules SET last_enqueued_at=$2,next_run_at=$3 WHERE id=$1`, schedule.ID, now, nextRun)
	} else {
		_, err = tx.Exec(ctx, `UPDATE jobs.schedules SET next_run_at=$2 WHERE id=$1`, schedule.ID, nextRun)
	}
	return err
}

type activeRun struct {
	id         uuid.UUID
	riverJobID *int64
}

func (s *Scheduler) activeRuns(ctx context.Context, tx pgx.Tx, scheduleID uuid.UUID) ([]activeRun, error) {
	rows, err := tx.Query(ctx, `SELECT id,river_job_id FROM jobs.schedule_runs WHERE schedule_id=$1 AND status IN ('queued','running') FOR UPDATE`, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]activeRun, 0)
	for rows.Next() {
		var item activeRun
		if err = rows.Scan(&item.id, &item.riverJobID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func RegisteredHandlerKeys() []string {
	items := jobs.CompiledHandlers()
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}

func ValidateRegistry() error {
	for _, key := range RegisteredHandlerKeys() {
		if key != "system.health.snapshot" {
			return fmt.Errorf("compiled handler %q has no worker", key)
		}
	}
	return nil
}
