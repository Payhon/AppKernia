BEGIN;

-- Push providers are protocol/security capabilities compiled into the server and
-- native clients.  They intentionally do not use the extensible dictionary
-- catalog as an execution registry.
CREATE TABLE notify.push_provider_configs (
    id                      uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id               uuid NOT NULL,
    app_id                  uuid NOT NULL,
    environment             varchar(20) NOT NULL,
    provider                varchar(32) NOT NULL,
    config_schema_version   integer NOT NULL DEFAULT 1,
    public_config           jsonb NOT NULL DEFAULT '{}'::jsonb,
    secret_ciphertext       bytea,
    secret_key_version      integer,
    secret_field_names      text[] NOT NULL DEFAULT '{}',
    credential_fingerprint  varchar(64),
    status                  varchar(20) NOT NULL DEFAULT 'draft',
    last_preflight_at       timestamptz,
    last_preflight_status   varchar(20),
    last_preflight_issues   jsonb NOT NULL DEFAULT '[]'::jsonb,
    lock_version            integer NOT NULL DEFAULT 1,
    created_by              uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by              uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_push_provider_config UNIQUE (tenant_id, app_id, environment, provider),
    CONSTRAINT fk_push_provider_config_app
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_push_provider_environment CHECK (environment IN ('development', 'test', 'staging', 'production')),
    CONSTRAINT ck_push_provider_code CHECK (provider IN (
        'apns', 'fcm', 'huawei_android', 'honor', 'xiaomi', 'oppo', 'vivo', 'meizu', 'harmony'
    )),
    CONSTRAINT ck_push_provider_schema_version CHECK (config_schema_version > 0),
    CONSTRAINT ck_push_provider_public_config CHECK (jsonb_typeof(public_config) = 'object'),
    CONSTRAINT ck_push_provider_secret_pair CHECK (
        (secret_ciphertext IS NULL AND secret_key_version IS NULL AND cardinality(secret_field_names) = 0)
        OR (secret_ciphertext IS NOT NULL AND secret_key_version IS NOT NULL AND cardinality(secret_field_names) > 0)
    ),
    CONSTRAINT ck_push_provider_status CHECK (status IN ('draft', 'active', 'disabled', 'faulted')),
    CONSTRAINT ck_push_provider_preflight_status CHECK (
        last_preflight_status IS NULL OR last_preflight_status IN ('ready', 'failed')
    ),
    CONSTRAINT ck_push_provider_preflight_issues CHECK (jsonb_typeof(last_preflight_issues) = 'array'),
    CONSTRAINT ck_push_provider_lock_version CHECK (lock_version > 0)
);
CREATE INDEX idx_push_provider_configs_app_status
    ON notify.push_provider_configs (app_id, environment, status, provider);
CREATE TRIGGER tr_push_provider_configs_touch_updated_at
BEFORE UPDATE ON notify.push_provider_configs
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

ALTER TABLE notify.push_devices DROP CONSTRAINT uq_push_token_hash;
ALTER TABLE notify.push_devices DROP CONSTRAINT ck_push_device_provider;
ALTER TABLE notify.push_devices
    ADD COLUMN tenant_id uuid,
    ADD COLUMN platform varchar(16) NOT NULL DEFAULT 'unknown',
    ADD COLUMN build_variant varchar(24) NOT NULL DEFAULT 'legacy',
    ADD COLUMN locale varchar(16) NOT NULL DEFAULT 'zh-CN',
    ADD COLUMN sdk_version varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN app_version varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN registered_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN token_updated_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN invalidated_at timestamptz,
    ADD COLUMN invalid_reason varchar(120);

UPDATE notify.push_devices d
SET tenant_id = a.tenant_id,
    provider = CASE d.provider WHEN 'hms' THEN 'huawei_android' ELSE d.provider END,
    platform = CASE d.provider WHEN 'apns' THEN 'ios' WHEN 'fcm' THEN 'android' WHEN 'hms' THEN 'android' ELSE 'unknown' END,
    build_variant = CASE d.provider WHEN 'apns' THEN 'ios' WHEN 'fcm' THEN 'android_google' WHEN 'hms' THEN 'android_china' ELSE 'legacy' END
FROM app.applications a
WHERE a.id = d.app_id;

ALTER TABLE notify.push_devices ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE notify.push_devices
    ADD CONSTRAINT fk_push_devices_app_same_tenant
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    ADD CONSTRAINT ck_push_device_provider CHECK (provider IN (
        'apns', 'fcm', 'huawei_android', 'honor', 'xiaomi', 'oppo', 'vivo', 'meizu', 'harmony', 'custom'
    )),
    ADD CONSTRAINT ck_push_device_platform CHECK (platform IN ('ios', 'android', 'harmony', 'unknown')),
    ADD CONSTRAINT ck_push_device_build_variant CHECK (build_variant IN ('ios', 'android_google', 'android_china', 'harmony', 'legacy')),
    ADD CONSTRAINT ck_push_device_locale CHECK (locale IN ('zh-CN', 'en-US')),
    ADD CONSTRAINT ck_push_device_provider_platform CHECK (
        (provider = 'apns' AND platform = 'ios' AND build_variant = 'ios')
        OR (provider = 'fcm' AND platform = 'android' AND build_variant = 'android_google')
        OR (provider IN ('huawei_android', 'honor', 'xiaomi', 'oppo', 'vivo', 'meizu') AND platform = 'android' AND build_variant = 'android_china')
        OR (provider = 'harmony' AND platform = 'harmony' AND build_variant = 'harmony')
        OR (provider = 'custom' AND platform = 'unknown' AND build_variant = 'legacy')
    );
WITH ranked AS (
    SELECT id, row_number() OVER (
        PARTITION BY app_id, device_id
        ORDER BY updated_at DESC, id DESC
    ) AS position
    FROM notify.push_devices
    WHERE status = 'active' AND device_id IS NOT NULL
)
UPDATE notify.push_devices d
SET status = 'disabled', invalidated_at = now(), invalid_reason = 'migration_replaced_duplicate'
FROM ranked r
WHERE d.id = r.id AND r.position > 1;
CREATE UNIQUE INDEX uq_push_devices_app_provider_token
    ON notify.push_devices (app_id, provider, token_hash);
CREATE UNIQUE INDEX uq_push_devices_active_app_device
    ON notify.push_devices (app_id, device_id)
    WHERE status = 'active' AND device_id IS NOT NULL;
CREATE INDEX idx_push_devices_active_fanout
    ON notify.push_devices (app_id, user_id, provider)
    WHERE status = 'active';

ALTER TABLE notify.templates
    ADD COLUMN push_category varchar(32) NOT NULL DEFAULT 'service_security',
    ADD COLUMN push_ttl_seconds integer NOT NULL DEFAULT 86400,
    ADD COLUMN push_collapse_key varchar(128),
    ADD COLUMN push_route_key varchar(96),
    ADD CONSTRAINT ck_notify_template_push_category CHECK (push_category IN ('service_security', 'news_operations')),
    ADD CONSTRAINT ck_notify_template_push_ttl CHECK (push_ttl_seconds BETWEEN 300 AND 604800),
    ADD CONSTRAINT ck_notify_template_push_route CHECK (
        push_route_key IS NULL OR push_route_key ~ '^[a-z][a-z0-9_.-]{1,95}$'
    );

ALTER TABLE notify.messages
    ADD COLUMN push_category varchar(32) NOT NULL DEFAULT 'service_security',
    ADD COLUMN push_ttl_seconds integer NOT NULL DEFAULT 86400,
    ADD COLUMN push_collapse_key varchar(128),
    ADD COLUMN push_route_key varchar(96),
    ADD COLUMN push_route_params jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD CONSTRAINT ck_notify_message_push_category CHECK (push_category IN ('service_security', 'news_operations')),
    ADD CONSTRAINT ck_notify_message_push_ttl CHECK (push_ttl_seconds BETWEEN 300 AND 604800),
    ADD CONSTRAINT ck_notify_message_push_route CHECK (
        push_route_key IS NULL OR push_route_key ~ '^[a-z][a-z0-9_.-]{1,95}$'
    ),
    ADD CONSTRAINT ck_notify_message_push_route_params CHECK (jsonb_typeof(push_route_params) = 'object');

ALTER TABLE notify.recipients
    ADD COLUMN push_skip_reason varchar(40),
    ADD COLUMN push_evaluated_at timestamptz,
    ADD CONSTRAINT ck_notify_recipient_push_skip_reason CHECK (
        push_skip_reason IS NULL OR push_skip_reason IN (
            'membership_inactive', 'push_disabled', 'category_disabled',
            'no_active_device', 'provider_unavailable', 'message_expired'
        )
    );

ALTER TABLE notify.deliveries
    ADD COLUMN push_device_id uuid REFERENCES notify.push_devices(id) ON DELETE SET NULL,
    ADD COLUMN accepted_at timestamptz,
    ADD COLUMN opened_at timestamptz,
    ADD COLUMN provider_result varchar(32),
    ADD COLUMN error_code varchar(160),
    ADD CONSTRAINT ck_notify_delivery_provider_result CHECK (
        provider_result IS NULL OR provider_result IN (
            'accepted', 'invalid_token', 'throttled', 'transient', 'permanent',
            'auth_config_error', 'unknown_after_write'
        )
    );
CREATE UNIQUE INDEX uq_notify_push_delivery_recipient_device
    ON notify.deliveries (tenant_id, message_id, user_id, push_device_id)
    WHERE channel = 'push' AND message_id IS NOT NULL AND user_id IS NOT NULL AND push_device_id IS NOT NULL;
CREATE INDEX idx_notify_push_deliveries_opened
    ON notify.deliveries (app_id, accepted_at DESC, opened_at)
    WHERE channel = 'push';

UPDATE iam.user_preferences
SET notification_preferences = jsonb_set(
    jsonb_set(
        notification_preferences,
        '{push_service}',
        COALESCE(notification_preferences -> 'push_service', 'true'::jsonb),
        true
    ),
    '{push_operations}',
    COALESCE(notification_preferences -> 'push_operations', 'false'::jsonb),
    true
)
WHERE NOT (notification_preferences ? 'push_service')
   OR NOT (notification_preferences ? 'push_operations');

ALTER TABLE iam.user_preferences ALTER COLUMN notification_preferences
    SET DEFAULT '{"in_app":true,"push":false,"push_service":true,"push_operations":false,"email":true}'::jsonb;

ALTER TABLE iam.user_preferences DROP CONSTRAINT ck_user_preferences_notification_preferences;
ALTER TABLE iam.user_preferences ADD CONSTRAINT ck_user_preferences_notification_preferences CHECK (
    jsonb_typeof(notification_preferences) = 'object'
    AND jsonb_typeof(notification_preferences -> 'in_app') = 'boolean'
    AND jsonb_typeof(notification_preferences -> 'push') = 'boolean'
    AND jsonb_typeof(notification_preferences -> 'push_service') = 'boolean'
    AND jsonb_typeof(notification_preferences -> 'push_operations') = 'boolean'
    AND jsonb_typeof(notification_preferences -> 'email') = 'boolean'
);

INSERT INTO iam.permissions (
    code, name, module_code, resource_name, action_name, permission_kind,
    http_methods, route_pattern, description, status
) VALUES
    ('notify.push_provider.read', '查看推送渠道', 'notify', 'push_provider', 'read', 'api', ARRAY['GET'], '/admin-api/v1/apps/{app_id}/push-provider-configs', 'Read application push provider configurations without secret values', 'active'),
    ('notify.push_provider.manage', '管理推送渠道', 'notify', 'push_provider', 'manage', 'api', ARRAY['PUT','POST'], '/admin-api/v1/apps/{app_id}/push-provider-configs/{provider}', 'Create and change application push provider configurations', 'active'),
    ('notify.push_provider.rotate_secret', '轮换推送凭据', 'notify', 'push_provider', 'rotate_secret', 'api', ARRAY['POST'], '/admin-api/v1/apps/{app_id}/push-provider-configs/{id}/rotate-secret', 'Replace encrypted push provider credentials', 'active'),
    ('notify.push.preflight', '预检推送渠道', 'notify', 'push', 'preflight', 'api', ARRAY['POST'], '/admin-api/v1/apps/{app_id}/push-provider-configs/{id}/preflight', 'Validate push provider credentials and public configuration', 'active'),
    ('notify.push.test', '测试推送渠道', 'notify', 'push', 'test', 'api', ARRAY['POST'], '/admin-api/v1/apps/{app_id}/push-provider-configs/{id}/test', 'Send a test notification to a registered application device', 'active'),
    ('notify.operations.publish', '发布资讯运营推送', 'notify', 'operations', 'publish', 'api', ARRAY['POST'], '/admin-api/v1/apps/{app_id}/messages/{id}/publish', 'Publish news and operations push notifications', 'active')
ON CONFLICT (code) DO UPDATE SET
    name=EXCLUDED.name, module_code=EXCLUDED.module_code, resource_name=EXCLUDED.resource_name,
    action_name=EXCLUDED.action_name, permission_kind=EXCLUDED.permission_kind,
    http_methods=EXCLUDED.http_methods, route_pattern=EXCLUDED.route_pattern,
    description=EXCLUDED.description, status='active', updated_at=now();

COMMIT;
