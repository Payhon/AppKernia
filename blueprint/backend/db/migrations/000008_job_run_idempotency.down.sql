BEGIN;

DROP INDEX IF EXISTS jobs.uq_schedule_runs_scheduled_slot;
DROP INDEX IF EXISTS jobs.uq_schedule_runs_manual_idempotency;

ALTER TABLE jobs.schedule_runs
    DROP COLUMN IF EXISTS idempotency_key;

COMMIT;
