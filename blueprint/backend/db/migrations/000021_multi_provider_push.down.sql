BEGIN;

DELETE FROM iam.permissions WHERE code IN (
    'notify.push_provider.read',
    'notify.push_provider.manage',
    'notify.push_provider.rotate_secret',
    'notify.push.preflight',
    'notify.push.test',
    'notify.operations.publish'
);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM notify.push_provider_configs) THEN
        RAISE EXCEPTION 'cannot roll back multi-provider push while provider configurations exist';
    END IF;
    IF EXISTS (
        SELECT 1 FROM notify.push_devices
        WHERE provider IN ('honor', 'xiaomi', 'oppo', 'vivo', 'meizu', 'harmony')
    ) THEN
        RAISE EXCEPTION 'cannot roll back multi-provider push while new provider devices exist';
    END IF;
    IF EXISTS (
        SELECT 1 FROM notify.push_devices
        GROUP BY CASE provider WHEN 'huawei_android' THEN 'hms' ELSE provider END, token_hash
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'cannot roll back multi-provider push while provider token hashes overlap across Apps';
    END IF;
END $$;

ALTER TABLE iam.user_preferences DROP CONSTRAINT ck_user_preferences_notification_preferences;
UPDATE iam.user_preferences
SET notification_preferences = notification_preferences - 'push_service' - 'push_operations';
ALTER TABLE iam.user_preferences ALTER COLUMN notification_preferences
    SET DEFAULT '{"in_app":true,"push":false,"email":true}'::jsonb;
ALTER TABLE iam.user_preferences ADD CONSTRAINT ck_user_preferences_notification_preferences
    CHECK (jsonb_typeof(notification_preferences) = 'object');

DROP INDEX IF EXISTS notify.idx_notify_push_deliveries_opened;
DROP INDEX IF EXISTS notify.uq_notify_push_delivery_recipient_device;
ALTER TABLE notify.deliveries
    DROP CONSTRAINT IF EXISTS ck_notify_delivery_provider_result,
    DROP COLUMN IF EXISTS error_code,
    DROP COLUMN IF EXISTS provider_result,
    DROP COLUMN IF EXISTS opened_at,
    DROP COLUMN IF EXISTS accepted_at,
    DROP COLUMN IF EXISTS push_device_id;

ALTER TABLE notify.messages
    DROP CONSTRAINT IF EXISTS ck_notify_message_push_route_params,
    DROP CONSTRAINT IF EXISTS ck_notify_message_push_route,
    DROP CONSTRAINT IF EXISTS ck_notify_message_push_ttl,
    DROP CONSTRAINT IF EXISTS ck_notify_message_push_category,
    DROP COLUMN IF EXISTS push_route_params,
    DROP COLUMN IF EXISTS push_route_key,
    DROP COLUMN IF EXISTS push_collapse_key,
    DROP COLUMN IF EXISTS push_ttl_seconds,
    DROP COLUMN IF EXISTS push_category;

ALTER TABLE notify.recipients
    DROP CONSTRAINT IF EXISTS ck_notify_recipient_push_skip_reason,
    DROP COLUMN IF EXISTS push_evaluated_at,
    DROP COLUMN IF EXISTS push_skip_reason;

ALTER TABLE notify.templates
    DROP CONSTRAINT IF EXISTS ck_notify_template_push_route,
    DROP CONSTRAINT IF EXISTS ck_notify_template_push_ttl,
    DROP CONSTRAINT IF EXISTS ck_notify_template_push_category,
    DROP COLUMN IF EXISTS push_route_key,
    DROP COLUMN IF EXISTS push_collapse_key,
    DROP COLUMN IF EXISTS push_ttl_seconds,
    DROP COLUMN IF EXISTS push_category;

DROP INDEX IF EXISTS notify.idx_push_devices_active_fanout;
DROP INDEX IF EXISTS notify.uq_push_devices_active_app_device;
DROP INDEX IF EXISTS notify.uq_push_devices_app_provider_token;
ALTER TABLE notify.push_devices
    DROP CONSTRAINT IF EXISTS ck_push_device_provider_platform,
    DROP CONSTRAINT IF EXISTS ck_push_device_locale,
    DROP CONSTRAINT IF EXISTS ck_push_device_build_variant,
    DROP CONSTRAINT IF EXISTS ck_push_device_platform,
    DROP CONSTRAINT IF EXISTS ck_push_device_provider,
    DROP CONSTRAINT IF EXISTS fk_push_devices_app_same_tenant;
UPDATE notify.push_devices SET provider = 'hms' WHERE provider = 'huawei_android';
ALTER TABLE notify.push_devices
    DROP COLUMN IF EXISTS invalid_reason,
    DROP COLUMN IF EXISTS invalidated_at,
    DROP COLUMN IF EXISTS token_updated_at,
    DROP COLUMN IF EXISTS registered_at,
    DROP COLUMN IF EXISTS app_version,
    DROP COLUMN IF EXISTS sdk_version,
    DROP COLUMN IF EXISTS locale,
    DROP COLUMN IF EXISTS build_variant,
    DROP COLUMN IF EXISTS platform,
    DROP COLUMN IF EXISTS tenant_id,
    ADD CONSTRAINT ck_push_device_provider CHECK (provider IN ('apns', 'fcm', 'hms', 'custom')),
    ADD CONSTRAINT uq_push_token_hash UNIQUE (provider, token_hash);

DROP TABLE IF EXISTS notify.push_provider_configs;

COMMIT;
