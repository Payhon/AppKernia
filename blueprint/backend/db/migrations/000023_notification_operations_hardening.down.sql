BEGIN;

ALTER TABLE notify.recipients
    DROP CONSTRAINT IF EXISTS ck_notify_recipient_push_environment,
    DROP COLUMN IF EXISTS push_environment;

DROP TRIGGER IF EXISTS tr_delivery_daily_metrics_touch_updated_at
    ON notify.delivery_daily_metrics;

COMMIT;
