BEGIN;

-- Migration 23 creates this trigger conditionally so it can upgrade local
-- databases that already received the equivalent development schema. Recreate
-- it directly here so the canonical schema and static trigger inventory agree.
DROP TRIGGER IF EXISTS tr_delivery_daily_metrics_touch_updated_at
    ON notify.delivery_daily_metrics;

CREATE TRIGGER tr_delivery_daily_metrics_touch_updated_at
BEFORE UPDATE ON notify.delivery_daily_metrics
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMIT;
