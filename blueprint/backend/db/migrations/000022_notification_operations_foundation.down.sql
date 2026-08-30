BEGIN;

DELETE FROM iam.permissions WHERE code IN (
    'notify.observability.read',
    'notify.task.retry',
    'notify.message.submit',
    'notify.message.status.read',
    'notify.message.broadcast'
);

DROP TABLE IF EXISTS notify.delivery_daily_metrics;
DROP INDEX IF EXISTS notify.idx_notify_deliveries_operations;
ALTER TABLE notify.deliveries
    DROP CONSTRAINT IF EXISTS ck_notify_delivery_environment,
    DROP CONSTRAINT IF EXISTS fk_notify_delivery_message_run,
    DROP COLUMN IF EXISTS delivery_environment,
    DROP COLUMN IF EXISTS task_run_id,
    DROP COLUMN IF EXISTS message_run_id;
DROP TABLE IF EXISTS notify.message_runs;
DROP TABLE IF EXISTS jobs.task_attempts;
DROP TABLE IF EXISTS jobs.task_runs;
DROP TABLE IF EXISTS sys.api_client_apps;

COMMIT;
