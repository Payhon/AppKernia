BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger
        WHERE tgname = 'tr_delivery_daily_metrics_touch_updated_at'
          AND tgrelid = 'notify.delivery_daily_metrics'::regclass
          AND NOT tgisinternal
    ) THEN
        CREATE TRIGGER tr_delivery_daily_metrics_touch_updated_at
        BEFORE UPDATE ON notify.delivery_daily_metrics
        FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
    END IF;
END $$;

ALTER TABLE notify.recipients
    ADD COLUMN IF NOT EXISTS push_environment varchar(20);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_notify_recipient_push_environment'
          AND conrelid = 'notify.recipients'::regclass
    ) THEN
        ALTER TABLE notify.recipients
            ADD CONSTRAINT ck_notify_recipient_push_environment CHECK (
                push_environment IS NULL OR push_environment IN ('development', 'test', 'staging', 'production')
            );
    END IF;
END $$;

COMMIT;
