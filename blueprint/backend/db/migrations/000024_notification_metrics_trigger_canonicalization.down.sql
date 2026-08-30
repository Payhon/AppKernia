BEGIN;

-- Version 23 owns the effective trigger. A 24 -> 23 rollback restores the same
-- trigger definition instead of silently removing version 23 behavior.
DROP TRIGGER IF EXISTS tr_delivery_daily_metrics_touch_updated_at
    ON notify.delivery_daily_metrics;

CREATE TRIGGER tr_delivery_daily_metrics_touch_updated_at
BEFORE UPDATE ON notify.delivery_daily_metrics
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMIT;
