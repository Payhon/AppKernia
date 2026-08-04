BEGIN;

ALTER TABLE jobs.schedule_runs
    ADD COLUMN idempotency_key varchar(128);

CREATE UNIQUE INDEX uq_schedule_runs_manual_idempotency
    ON jobs.schedule_runs (schedule_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE UNIQUE INDEX uq_schedule_runs_scheduled_slot
    ON jobs.schedule_runs (schedule_id, scheduled_at)
    WHERE trigger_type = 'schedule';

COMMENT ON COLUMN jobs.schedule_runs.idempotency_key IS
    'Opaque client idempotency key for manual execution; never reused as executable input.';

COMMIT;
