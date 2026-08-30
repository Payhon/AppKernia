// Package jobqueue isolates application modules from the concrete River client
// and records a tenant-scoped, payload-free execution projection.
package jobqueue

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

var (
	ErrInvalidSpec     = errors.New("job queue specification is invalid")
	ErrUnknownKind     = errors.New("job queue kind is not registered")
	ErrSpecMismatch    = errors.New("job queue specification does not match its registered definition")
	ErrInvalidRegistry = errors.New("job queue registry is invalid")
)

const (
	RetryClassTransient = "transient"
	RetryClassThrottled = "throttled"
)

type Args interface {
	Kind() string
}

type Scope struct {
	TenantID      uuid.UUID
	AppID         *uuid.UUID
	ModuleCode    string
	ResourceType  string
	ResourceID    *uuid.UUID
	CorrelationID *uuid.UUID
}

type Spec struct {
	Scope        Scope
	Args         Args
	Queue        string
	MaxAttempts  int
	ScheduledAt  *time.Time
	UniqueByArgs bool
}

// Definition is the compile-time execution policy for one task type. Runtime
// producers may request fewer attempts, but cannot select another queue or
// exceed the registered maximum.
type Definition struct {
	Kind         string
	Queue        string
	MaxAttempts  int
	Timeout      time.Duration
	RetryClasses []string
}

type Registry struct {
	definitions map[string]Definition
}

func NewRegistry(definitions ...Definition) (*Registry, error) {
	registry := &Registry{definitions: make(map[string]Definition, len(definitions))}
	for _, definition := range definitions {
		definition.Kind = strings.TrimSpace(definition.Kind)
		definition.Queue = strings.TrimSpace(definition.Queue)
		if !code(definition.Kind, 100) || !code(definition.Queue, 80) || definition.MaxAttempts < 1 ||
			definition.MaxAttempts > 100 || definition.Timeout <= 0 || definition.Timeout > 24*time.Hour {
			return nil, ErrInvalidRegistry
		}
		if _, exists := registry.definitions[definition.Kind]; exists {
			return nil, ErrInvalidRegistry
		}
		seenRetryClasses := make(map[string]struct{}, len(definition.RetryClasses))
		for index, retryClass := range definition.RetryClasses {
			retryClass = strings.TrimSpace(retryClass)
			if retryClass != RetryClassTransient && retryClass != RetryClassThrottled {
				return nil, ErrInvalidRegistry
			}
			if _, exists := seenRetryClasses[retryClass]; exists {
				return nil, ErrInvalidRegistry
			}
			seenRetryClasses[retryClass] = struct{}{}
			definition.RetryClasses[index] = retryClass
		}
		definition.RetryClasses = append([]string(nil), definition.RetryClasses...)
		registry.definitions[definition.Kind] = definition
	}
	if len(registry.definitions) == 0 {
		return nil, ErrInvalidRegistry
	}
	return registry, nil
}

func MustRegistry(definitions ...Definition) *Registry {
	registry, err := NewRegistry(definitions...)
	if err != nil {
		panic(err)
	}
	return registry
}

func (r *Registry) Definition(kind string) (Definition, bool) {
	if r == nil {
		return Definition{}, false
	}
	definition, ok := r.definitions[strings.TrimSpace(kind)]
	definition.RetryClasses = append([]string(nil), definition.RetryClasses...)
	return definition, ok
}

func (r *Registry) ValidateSpec(spec Spec) error {
	if r == nil || spec.Args == nil {
		return ErrInvalidSpec
	}
	definition, ok := r.Definition(spec.Args.Kind())
	if !ok {
		return ErrUnknownKind
	}
	if strings.TrimSpace(spec.Queue) != definition.Queue || spec.MaxAttempts < 1 || spec.MaxAttempts > definition.MaxAttempts {
		return ErrSpecMismatch
	}
	return nil
}

type Run struct {
	ID         uuid.UUID
	RiverJobID int64
	Status     string
}

type Enqueuer interface {
	Enqueue(context.Context, Spec) (Run, error)
	EnqueueTx(context.Context, pgx.Tx, Spec) (Run, error)
	GetState(context.Context, uuid.UUID, uuid.UUID) (Run, error)
}

type RiverAdapter struct {
	pool     *pgxpool.Pool
	client   *river.Client[pgx.Tx]
	registry *Registry
}

func NewRiverAdapter(pool *pgxpool.Pool, client *river.Client[pgx.Tx], registry *Registry) *RiverAdapter {
	return &RiverAdapter{pool: pool, client: client, registry: registry}
}

func (a *RiverAdapter) Enqueue(ctx context.Context, spec Spec) (Run, error) {
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Run{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	run, err := a.EnqueueTx(ctx, tx, spec)
	if err != nil {
		return Run{}, err
	}
	return run, tx.Commit(ctx)
}

func (a *RiverAdapter) EnqueueTx(ctx context.Context, tx pgx.Tx, spec Spec) (Run, error) {
	if a == nil || a.client == nil || a.registry == nil || tx == nil || spec.Args == nil || spec.Scope.TenantID == uuid.Nil ||
		!code(spec.Scope.ModuleCode, 64) || !code(spec.Scope.ResourceType, 80) ||
		strings.TrimSpace(spec.Queue) == "" || len(spec.Queue) > 80 || spec.MaxAttempts < 1 || spec.MaxAttempts > 100 {
		return Run{}, ErrInvalidSpec
	}
	if err := a.registry.ValidateSpec(spec); err != nil {
		return Run{}, err
	}
	opts := &river.InsertOpts{Queue: spec.Queue, MaxAttempts: spec.MaxAttempts}
	if spec.ScheduledAt != nil {
		opts.ScheduledAt = spec.ScheduledAt.UTC()
	}
	if spec.UniqueByArgs {
		opts.UniqueOpts = river.UniqueOpts{ByArgs: true}
	}
	inserted, err := a.client.InsertTx(ctx, tx, spec.Args, opts)
	if err != nil {
		return Run{}, err
	}
	status := "queued"
	scheduledAt := time.Now().UTC()
	if spec.ScheduledAt != nil {
		status, scheduledAt = "scheduled", spec.ScheduledAt.UTC()
	}
	var run Run
	err = tx.QueryRow(ctx, `INSERT INTO jobs.task_runs(
		tenant_id,app_id,module_code,task_kind,queue_name,resource_type,resource_id,correlation_id,
		river_job_id,status,scheduled_at,max_attempts)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT(river_job_id) WHERE river_job_id IS NOT NULL DO UPDATE SET updated_at=jobs.task_runs.updated_at
		RETURNING id,river_job_id,status`, spec.Scope.TenantID, spec.Scope.AppID, spec.Scope.ModuleCode,
		spec.Args.Kind(), spec.Queue, spec.Scope.ResourceType, spec.Scope.ResourceID, spec.Scope.CorrelationID,
		inserted.Job.ID, status, scheduledAt, spec.MaxAttempts).Scan(&run.ID, &run.RiverJobID, &run.Status)
	return run, err
}

func (a *RiverAdapter) GetState(ctx context.Context, tenantID, id uuid.UUID) (Run, error) {
	var out Run
	err := a.pool.QueryRow(ctx, `SELECT id,COALESCE(river_job_id,0),status FROM jobs.task_runs WHERE tenant_id=$1 AND id=$2`, tenantID, id).
		Scan(&out.ID, &out.RiverJobID, &out.Status)
	return out, err
}

func code(value string, maxLen int) bool {
	value = strings.TrimSpace(value)
	if len(value) < 2 || len(value) > maxLen || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for _, r := range value[1:] {
		if r < 'a' || r > 'z' {
			if r < '0' || r > '9' {
				if r != '_' && r != '.' && r != '-' {
					return false
				}
			}
		}
	}
	return true
}

type Completion struct {
	Status            string
	ResultClass       string
	ErrorCode         string
	ErrorSummary      string
	ExternalRequestID string
	TraceID           string
	NextRetryAt       *time.Time
}

// StartAttempt is called by worker adapters before domain work begins. It never
// receives job args, so sensitive payloads cannot enter the projection.
func StartAttempt(ctx context.Context, pool *pgxpool.Pool, riverJobID int64, attempt int) error {
	if pool == nil || riverJobID < 1 || attempt < 1 || attempt > 100 {
		return ErrInvalidSpec
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var runID, tenantID uuid.UUID
	var appID *uuid.UUID
	err = tx.QueryRow(ctx, `UPDATE jobs.task_runs SET status='running',started_at=COALESCE(started_at,now()),
		attempt_count=GREATEST(attempt_count,$2),next_retry_at=NULL
		WHERE river_job_id=$1 RETURNING id,tenant_id,app_id`, riverJobID, attempt).Scan(&runID, &tenantID, &appID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO jobs.task_attempts(tenant_id,app_id,task_run_id,attempt_number,status)
		VALUES($1,$2,$3,$4,'running') ON CONFLICT(task_run_id,attempt_number) DO NOTHING`, tenantID, appID, runID, attempt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func FinishAttempt(ctx context.Context, pool *pgxpool.Pool, riverJobID int64, attempt int, completion Completion) error {
	if pool == nil || riverJobID < 1 || attempt < 1 || attempt > 100 ||
		completion.Status != "retry_wait" && completion.Status != "succeeded" && completion.Status != "failed" && completion.Status != "cancelled" {
		return ErrInvalidSpec
	}
	completion.ErrorSummary = safe(completion.ErrorSummary, 500)
	completion.ErrorCode = safe(completion.ErrorCode, 160)
	completion.ResultClass = safe(completion.ResultClass, 40)
	completion.ExternalRequestID = safe(completion.ExternalRequestID, 255)
	completion.TraceID = safe(completion.TraceID, 64)
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var runID uuid.UUID
	err = tx.QueryRow(ctx, `UPDATE jobs.task_runs SET status=$2,finalized_at=CASE WHEN $2 IN ('succeeded','failed','cancelled') THEN now() ELSE NULL END,
		next_retry_at=$3,last_result_class=NULLIF($4,''),last_error_code=NULLIF($5,''),last_error_summary=NULLIF($6,''),trace_id=NULLIF($7,'')
		WHERE river_job_id=$1 RETURNING id`, riverJobID, completion.Status, completion.NextRetryAt,
		completion.ResultClass, completion.ErrorCode, completion.ErrorSummary, completion.TraceID).Scan(&runID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE jobs.task_attempts SET status=$3,result_class=NULLIF($4,''),error_code=NULLIF($5,''),
		error_summary=NULLIF($6,''),external_request_id=NULLIF($7,''),trace_id=NULLIF($8,''),finished_at=now(),
		duration_ms=GREATEST(0,(extract(epoch FROM (now()-started_at))*1000)::bigint),next_retry_at=$9
		WHERE task_run_id=$1 AND attempt_number=$2`, runID, attempt, completion.Status, completion.ResultClass,
		completion.ErrorCode, completion.ErrorSummary, completion.ExternalRequestID, completion.TraceID, completion.NextRetryAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func safe(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > limit {
		value = string(runes[:limit])
	}
	return value
}
