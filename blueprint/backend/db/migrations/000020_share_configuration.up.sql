BEGIN;

CREATE TABLE sys.share_configs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    name varchar(160) NOT NULL,
    description varchar(2000) NOT NULL DEFAULT '',
    provider_code varchar(32) NOT NULL,
    external_app_id varchar(255) NOT NULL,
    config_schema_version integer NOT NULL DEFAULT 1,
    public_config jsonb NOT NULL DEFAULT '{}'::jsonb,
    secret_ciphertext bytea,
    secret_key_version integer,
    secret_field_names text[] NOT NULL DEFAULT '{}'::text[],
    status varchar(16) NOT NULL DEFAULT 'draft',
    lock_version integer NOT NULL DEFAULT 1,
    created_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_share_configs_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT uq_share_configs_tenant_id_provider UNIQUE (tenant_id, id, provider_code),
    CONSTRAINT ck_share_configs_name CHECK (name = btrim(name) AND name <> ''),
    CONSTRAINT ck_share_configs_provider CHECK (provider_code ~ '^[a-z][a-z0-9_]{1,31}$'),
    CONSTRAINT ck_share_configs_external_app CHECK (external_app_id = btrim(external_app_id) AND external_app_id <> ''),
    CONSTRAINT ck_share_configs_schema_version CHECK (config_schema_version > 0),
    CONSTRAINT ck_share_configs_public_object CHECK (jsonb_typeof(public_config) = 'object'),
    CONSTRAINT ck_share_configs_secret_pair CHECK (
        (secret_ciphertext IS NULL AND secret_key_version IS NULL AND cardinality(secret_field_names) = 0)
        OR (secret_ciphertext IS NOT NULL AND secret_key_version IS NOT NULL AND cardinality(secret_field_names) > 0)
    ),
    CONSTRAINT ck_share_configs_status CHECK (status IN ('draft', 'active', 'disabled')),
    CONSTRAINT ck_share_configs_lock_version CHECK (lock_version > 0),
    CONSTRAINT ck_share_configs_wechat_app_id CHECK (
        provider_code <> 'wechat' OR external_app_id ~ '^wx[A-Za-z0-9]{16}$'
    )
);

CREATE UNIQUE INDEX uq_share_configs_tenant_provider_name
    ON sys.share_configs (tenant_id, provider_code, lower(name))
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_share_configs_tenant_provider_external_app
    ON sys.share_configs (tenant_id, provider_code, external_app_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_share_configs_tenant_status
    ON sys.share_configs (tenant_id, provider_code, status, updated_at DESC, id DESC)
    WHERE deleted_at IS NULL;
CREATE TRIGGER tr_share_configs_touch_updated_at
BEFORE UPDATE ON sys.share_configs FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

CREATE TABLE app.application_share_bindings (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    provider_code varchar(32) NOT NULL,
    share_config_id uuid NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    scenes text[] NOT NULL DEFAULT ARRAY['session','timeline','favorite']::text[],
    share_origin varchar(2048) NOT NULL,
    fallback_mode varchar(16) NOT NULL DEFAULT 'system',
    lock_version integer NOT NULL DEFAULT 1,
    created_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_application_share_binding UNIQUE (tenant_id, app_id, provider_code),
    CONSTRAINT fk_application_share_binding_app
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_application_share_binding_config
        FOREIGN KEY (tenant_id, share_config_id, provider_code)
        REFERENCES sys.share_configs(tenant_id, id, provider_code) ON DELETE RESTRICT,
    CONSTRAINT ck_application_share_binding_provider CHECK (provider_code ~ '^[a-z][a-z0-9_]{1,31}$'),
    CONSTRAINT ck_application_share_binding_scenes CHECK (
        cardinality(scenes) BETWEEN 1 AND 3
        AND scenes <@ ARRAY['session','timeline','favorite']::text[]
    ),
    CONSTRAINT ck_application_share_binding_origin CHECK (
        share_origin ~ '^https://[A-Za-z0-9.-]+(:[0-9]+)?$'
        AND share_origin !~* '^https://(localhost|127\\.|0\\.|10\\.|192\\.168\\.|172\\.(1[6-9]|2[0-9]|3[01])\\.)'
    ),
    CONSTRAINT ck_application_share_binding_fallback CHECK (fallback_mode IN ('system')),
    CONSTRAINT ck_application_share_binding_lock_version CHECK (lock_version > 0)
);

CREATE INDEX idx_application_share_bindings_config
    ON app.application_share_bindings (tenant_id, share_config_id, app_id);
CREATE INDEX idx_application_share_bindings_runtime
    ON app.application_share_bindings (app_id, enabled, provider_code);
CREATE TRIGGER tr_application_share_bindings_touch_updated_at
BEFORE UPDATE ON app.application_share_bindings FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

INSERT INTO iam.permissions (
    code, name, module_code, resource_name, action_name, permission_kind,
    http_methods, route_pattern, description, status
) VALUES
    ('sys.share_config.read', '查看分享配置', 'sys', 'share_config', 'read', 'api', ARRAY['GET'], '/admin-api/v1/share-configs', 'Read tenant share provider configurations', 'active'),
    ('sys.share_config.create', '创建分享配置', 'sys', 'share_config', 'create', 'api', ARRAY['POST'], '/admin-api/v1/share-configs', 'Create tenant share provider configurations', 'active'),
    ('sys.share_config.update', '更新分享配置', 'sys', 'share_config', 'update', 'api', ARRAY['PATCH','POST'], '/admin-api/v1/share-configs/{id}', 'Update and change share configuration lifecycle', 'active'),
    ('sys.share_config.delete', '删除分享配置', 'sys', 'share_config', 'delete', 'api', ARRAY['DELETE'], '/admin-api/v1/share-configs/{id}', 'Delete an unused share configuration', 'active'),
    ('sys.share_config.rotate_secret', '轮换分享配置秘密', 'sys', 'share_config', 'rotate_secret', 'api', ARRAY['POST'], '/admin-api/v1/share-configs/{id}/rotate-secret', 'Rotate encrypted provider secret values', 'active'),
    ('app.share_binding.read', '查看应用分享绑定', 'app', 'share_binding', 'read', 'api', ARRAY['GET','POST'], '/admin-api/v1/apps/{app_id}/share-bindings', 'Read and preflight App share bindings', 'active'),
    ('app.share_binding.update', '更新应用分享绑定', 'app', 'share_binding', 'update', 'api', ARRAY['PUT','DELETE'], '/admin-api/v1/apps/{app_id}/share-bindings/{provider_code}', 'Bind or remove an App share configuration', 'active')
ON CONFLICT (code) DO UPDATE SET
    name=EXCLUDED.name, module_code=EXCLUDED.module_code, resource_name=EXCLUDED.resource_name,
    action_name=EXCLUDED.action_name, permission_kind=EXCLUDED.permission_kind,
    http_methods=EXCLUDED.http_methods, route_pattern=EXCLUDED.route_pattern,
    description=EXCLUDED.description, status='active', updated_at=now();

INSERT INTO sys.menus (
    parent_id, permission_id, code, title, menu_type, route_path, component_key,
    icon, sort_order, status, metadata
)
SELECT parent.id, permission.id, 'system.settings.share-configs', '分享配置', 'page',
       '/system/settings/share-configs', 'system.settings.share-configs',
       'ShareAltOutlined', 15, 'active', '{"i18n_key":"menu.system.settings.share_configs"}'::jsonb
FROM sys.menus parent
JOIN iam.permissions permission ON permission.code = 'sys.share_config.read'
WHERE parent.code = 'system.settings' AND parent.tenant_id IS NULL
ON CONFLICT (code) WHERE tenant_id IS NULL AND deleted_at IS NULL DO UPDATE SET
    parent_id=EXCLUDED.parent_id, permission_id=EXCLUDED.permission_id,
    title=EXCLUDED.title, menu_type=EXCLUDED.menu_type, route_path=EXCLUDED.route_path,
    component_key=EXCLUDED.component_key, icon=EXCLUDED.icon, sort_order=EXCLUDED.sort_order,
    status='active', metadata=EXCLUDED.metadata, updated_at=now();

COMMIT;
