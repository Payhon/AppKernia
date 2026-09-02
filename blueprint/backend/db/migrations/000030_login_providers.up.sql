BEGIN;

-- Third-party login providers are compile-time capabilities. Tenant operators
-- may configure credentials and bind them to Apps, but cannot install code or
-- override provider endpoints through database data.
CREATE TABLE sys.login_provider_configs (
    id                      uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id               uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    name                    varchar(160) NOT NULL,
    description             varchar(2000) NOT NULL DEFAULT '',
    provider_code           varchar(32) NOT NULL,
    external_client_id      varchar(255) NOT NULL,
    config_schema_version   integer NOT NULL DEFAULT 1,
    public_config           jsonb NOT NULL DEFAULT '{}'::jsonb,
    secret_ciphertext       bytea,
    secret_key_version      integer,
    secret_field_names      text[] NOT NULL DEFAULT '{}'::text[],
    credential_fingerprint  varchar(64),
    status                  varchar(16) NOT NULL DEFAULT 'draft',
    last_preflight_at       timestamptz,
    last_preflight_status   varchar(16),
    last_preflight_issues   jsonb NOT NULL DEFAULT '[]'::jsonb,
    lock_version            integer NOT NULL DEFAULT 1,
    created_by              uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by              uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    deleted_at              timestamptz,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_login_provider_configs_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT uq_login_provider_configs_tenant_provider UNIQUE (tenant_id, id, provider_code),
    CONSTRAINT ck_login_provider_configs_name CHECK (name = btrim(name) AND name <> ''),
    CONSTRAINT ck_login_provider_configs_description CHECK (description = btrim(description)),
    CONSTRAINT ck_login_provider_configs_provider CHECK (provider_code ~ '^[a-z][a-z0-9_]{1,31}$'),
    CONSTRAINT ck_login_provider_configs_external_client CHECK (
        external_client_id = btrim(external_client_id) AND external_client_id <> ''
    ),
    CONSTRAINT ck_login_provider_configs_schema_version CHECK (config_schema_version = 1),
    CONSTRAINT ck_login_provider_configs_public_object CHECK (jsonb_typeof(public_config) = 'object'),
    CONSTRAINT ck_login_provider_configs_secret_pair CHECK (
        (secret_ciphertext IS NULL AND secret_key_version IS NULL AND cardinality(secret_field_names) = 0 AND credential_fingerprint IS NULL)
        OR (secret_ciphertext IS NOT NULL AND secret_key_version > 0 AND cardinality(secret_field_names) > 0
            AND credential_fingerprint ~ '^[a-f0-9]{64}$')
    ),
    CONSTRAINT ck_login_provider_configs_status CHECK (status IN ('draft', 'active', 'disabled')),
    CONSTRAINT ck_login_provider_configs_preflight_status CHECK (
        last_preflight_status IS NULL OR last_preflight_status IN ('ready', 'failed')
    ),
    CONSTRAINT ck_login_provider_configs_preflight_issues CHECK (jsonb_typeof(last_preflight_issues) = 'array'),
    CONSTRAINT ck_login_provider_configs_active_ready CHECK (
        status <> 'active' OR last_preflight_status = 'ready'
    ),
    CONSTRAINT ck_login_provider_configs_lock_version CHECK (lock_version > 0)
);

CREATE UNIQUE INDEX uq_login_provider_configs_tenant_provider_name
    ON sys.login_provider_configs (tenant_id, provider_code, lower(name))
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_login_provider_configs_tenant_provider_client
    ON sys.login_provider_configs (tenant_id, provider_code, external_client_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_login_provider_configs_tenant_status
    ON sys.login_provider_configs (tenant_id, provider_code, status, updated_at DESC, id DESC)
    WHERE deleted_at IS NULL;
CREATE TRIGGER tr_login_provider_configs_touch_updated_at
BEFORE UPDATE ON sys.login_provider_configs
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

CREATE TABLE app.application_login_provider_bindings (
    id                          uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id                   uuid NOT NULL,
    app_id                      uuid NOT NULL,
    provider_code               varchar(32) NOT NULL,
    login_provider_config_id    uuid NOT NULL,
    enabled                     boolean NOT NULL DEFAULT false,
    sort_order                  integer NOT NULL DEFAULT 100,
    lock_version                integer NOT NULL DEFAULT 1,
    created_by                  uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by                  uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at                  timestamptz NOT NULL DEFAULT now(),
    updated_at                  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_application_login_provider_binding UNIQUE (tenant_id, app_id, provider_code),
    CONSTRAINT fk_application_login_provider_binding_app
        FOREIGN KEY (tenant_id, app_id)
        REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_application_login_provider_binding_config
        FOREIGN KEY (tenant_id, login_provider_config_id, provider_code)
        REFERENCES sys.login_provider_configs(tenant_id, id, provider_code) ON DELETE RESTRICT,
    CONSTRAINT ck_application_login_provider_binding_provider CHECK (provider_code ~ '^[a-z][a-z0-9_]{1,31}$'),
    CONSTRAINT ck_application_login_provider_binding_sort CHECK (sort_order BETWEEN 0 AND 1000),
    CONSTRAINT ck_application_login_provider_binding_lock CHECK (lock_version > 0)
);

CREATE INDEX idx_application_login_provider_bindings_config
    ON app.application_login_provider_bindings (tenant_id, login_provider_config_id, app_id);
CREATE INDEX idx_application_login_provider_bindings_runtime
    ON app.application_login_provider_bindings (app_id, enabled, sort_order, provider_code);
CREATE TRIGGER tr_application_login_provider_bindings_touch_updated_at
BEFORE UPDATE ON app.application_login_provider_bindings
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

-- Mobile identities are deliberately App-scoped. The existing
-- iam.oauth_accounts table remains the Admin self-binding/local-adapter model.
CREATE TABLE iam.app_oauth_accounts (
    id                      uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id               uuid NOT NULL,
    app_id                  uuid NOT NULL,
    user_id                 uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    provider_code           varchar(32) NOT NULL,
    issuer                  varchar(255) NOT NULL,
    external_client_id      varchar(255) NOT NULL,
    subject                 varchar(512) NOT NULL,
    union_subject           varchar(512),
    provider_username       varchar(255),
    provider_profile        jsonb NOT NULL DEFAULT '{}'::jsonb,
    status                  varchar(16) NOT NULL DEFAULT 'active',
    bound_at                timestamptz NOT NULL DEFAULT now(),
    last_authenticated_at   timestamptz,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_app_oauth_accounts_app
        FOREIGN KEY (tenant_id, app_id)
        REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_app_oauth_accounts_member
        FOREIGN KEY (app_id, user_id)
        REFERENCES app.user_memberships(app_id, user_id) ON DELETE CASCADE,
    CONSTRAINT ck_app_oauth_accounts_provider CHECK (provider_code ~ '^[a-z][a-z0-9_]{1,31}$'),
    CONSTRAINT ck_app_oauth_accounts_issuer CHECK (issuer = btrim(issuer) AND issuer <> ''),
    CONSTRAINT ck_app_oauth_accounts_client CHECK (external_client_id = btrim(external_client_id) AND external_client_id <> ''),
    CONSTRAINT ck_app_oauth_accounts_subject CHECK (subject = btrim(subject) AND subject <> ''),
    CONSTRAINT ck_app_oauth_accounts_profile CHECK (jsonb_typeof(provider_profile) = 'object'),
    CONSTRAINT ck_app_oauth_accounts_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT uq_app_oauth_accounts_subject UNIQUE (app_id, provider_code, issuer, external_client_id, subject),
    CONSTRAINT uq_app_oauth_accounts_user_provider UNIQUE (app_id, user_id, provider_code)
);

CREATE INDEX idx_app_oauth_accounts_user
    ON iam.app_oauth_accounts (app_id, user_id, status, provider_code);
CREATE INDEX idx_app_oauth_accounts_union_subject
    ON iam.app_oauth_accounts (app_id, provider_code, union_subject)
    WHERE union_subject IS NOT NULL;
CREATE TRIGGER tr_app_oauth_accounts_touch_updated_at
BEFORE UPDATE ON iam.app_oauth_accounts
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

-- Authorization state is server-owned, short-lived and one-time. Raw provider
-- authorization codes, state values, ID tokens and provider access tokens are
-- never persisted.
CREATE TABLE iam.oauth_authorization_flows (
    id                              uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id                       uuid NOT NULL,
    app_id                          uuid NOT NULL,
    login_provider_config_id        uuid NOT NULL,
    provider_code                   varchar(32) NOT NULL,
    mode                            varchar(16) NOT NULL,
    platform                        varchar(16) NOT NULL,
    build_variant                   varchar(24) NOT NULL,
    user_id                         uuid REFERENCES iam.users(id) ON DELETE CASCADE,
    session_id                      uuid REFERENCES iam.sessions(id) ON DELETE CASCADE,
    reauth_purpose                  varchar(32),
    target_oauth_account_id         uuid REFERENCES iam.app_oauth_accounts(id) ON DELETE CASCADE,
    state_hash                      bytea NOT NULL,
    nonce_hash                      bytea,
    pkce_verifier_ciphertext        bytea,
    pkce_key_version                integer,
    device_key_hash                 bytea NOT NULL,
    verified_identity_ciphertext    bytea,
    verified_identity_key_version   integer,
    completion_ticket_hash          bytea,
    completion_ticket_expires_at    timestamptz,
    expires_at                      timestamptz NOT NULL,
    provider_verified_at            timestamptz,
    consumed_at                     timestamptz,
    failure_count                   integer NOT NULL DEFAULT 0,
    last_failure_code               varchar(160),
    created_at                      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_oauth_authorization_flows_app
        FOREIGN KEY (tenant_id, app_id)
        REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_oauth_authorization_flows_config
        FOREIGN KEY (tenant_id, login_provider_config_id, provider_code)
        REFERENCES sys.login_provider_configs(tenant_id, id, provider_code) ON DELETE CASCADE,
    CONSTRAINT ck_oauth_authorization_flows_provider CHECK (provider_code ~ '^[a-z][a-z0-9_]{1,31}$'),
    CONSTRAINT ck_oauth_authorization_flows_mode CHECK (mode IN ('login', 'bind', 'reauth')),
    CONSTRAINT ck_oauth_authorization_flows_platform CHECK (platform IN ('ios', 'android', 'harmony')),
    CONSTRAINT ck_oauth_authorization_flows_build_variant CHECK (
        build_variant IN ('ios', 'android_google', 'android_china', 'harmony')
    ),
    CONSTRAINT ck_oauth_authorization_flows_platform_variant CHECK (
        (platform = 'ios' AND build_variant = 'ios')
        OR (platform = 'android' AND build_variant IN ('android_google', 'android_china'))
        OR (platform = 'harmony' AND build_variant = 'harmony')
    ),
    CONSTRAINT ck_oauth_authorization_flows_actor CHECK (
        (mode = 'login' AND user_id IS NULL AND session_id IS NULL)
        OR (mode IN ('bind', 'reauth') AND user_id IS NOT NULL AND session_id IS NOT NULL)
    ),
    CONSTRAINT ck_oauth_authorization_flows_reauth_target CHECK (
        (mode IN ('login','bind') AND reauth_purpose IS NULL AND target_oauth_account_id IS NULL)
        OR (mode='reauth' AND reauth_purpose IN ('oauth_unbind','account_delete') AND target_oauth_account_id IS NOT NULL)
    ),
    CONSTRAINT ck_oauth_authorization_flows_state_hash CHECK (octet_length(state_hash) = 32),
    CONSTRAINT ck_oauth_authorization_flows_nonce_hash CHECK (nonce_hash IS NULL OR octet_length(nonce_hash) = 32),
    CONSTRAINT ck_oauth_authorization_flows_device_hash CHECK (octet_length(device_key_hash) = 32),
    CONSTRAINT ck_oauth_authorization_flows_pkce_pair CHECK (
        (pkce_verifier_ciphertext IS NULL AND pkce_key_version IS NULL)
        OR (pkce_verifier_ciphertext IS NOT NULL AND pkce_key_version > 0)
    ),
    CONSTRAINT ck_oauth_authorization_flows_identity_pair CHECK (
        (verified_identity_ciphertext IS NULL AND verified_identity_key_version IS NULL AND provider_verified_at IS NULL)
        OR (verified_identity_ciphertext IS NOT NULL AND verified_identity_key_version > 0 AND provider_verified_at IS NOT NULL)
    ),
    CONSTRAINT ck_oauth_authorization_flows_ticket CHECK (
        (completion_ticket_hash IS NULL AND completion_ticket_expires_at IS NULL)
        OR (octet_length(completion_ticket_hash) = 32
            AND completion_ticket_expires_at IS NOT NULL
            AND completion_ticket_expires_at <= expires_at
            AND completion_ticket_expires_at > created_at)
    ),
    CONSTRAINT ck_oauth_authorization_flows_expiry CHECK (expires_at > created_at),
    CONSTRAINT ck_oauth_authorization_flows_failures CHECK (failure_count BETWEEN 0 AND 20),
    CONSTRAINT uq_oauth_authorization_flows_state UNIQUE (state_hash)
);

CREATE UNIQUE INDEX uq_oauth_authorization_flows_ticket
    ON iam.oauth_authorization_flows (completion_ticket_hash)
    WHERE completion_ticket_hash IS NOT NULL;
CREATE INDEX idx_oauth_authorization_flows_open
    ON iam.oauth_authorization_flows (app_id, provider_code, expires_at, id)
    WHERE consumed_at IS NULL;
CREATE INDEX idx_oauth_authorization_flows_cleanup
    ON iam.oauth_authorization_flows (expires_at, id)
    WHERE consumed_at IS NULL;

-- Email and mobile identifiers become App-scoped facts. Existing global fields
-- remain for Admin compatibility and are copied for every existing App
-- membership. New Mobile authentication reads this table first.
CREATE TABLE app.user_login_identifiers (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL,
    app_id              uuid NOT NULL,
    user_id             uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    identifier_type     varchar(16) NOT NULL,
    normalized_value    public.citext NOT NULL,
    display_hint        varchar(160) NOT NULL,
    verified_at         timestamptz,
    status              varchar(16) NOT NULL DEFAULT 'active',
    lock_version        integer NOT NULL DEFAULT 1,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_app_user_identifiers_app
        FOREIGN KEY (tenant_id, app_id)
        REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_app_user_identifiers_member
        FOREIGN KEY (app_id, user_id)
        REFERENCES app.user_memberships(app_id, user_id) ON DELETE CASCADE,
    CONSTRAINT ck_app_user_identifiers_type CHECK (identifier_type IN ('email', 'mobile')),
    CONSTRAINT ck_app_user_identifiers_value CHECK (
        normalized_value::text = btrim(normalized_value::text)
        AND normalized_value::text <> ''
        AND length(normalized_value::text) <= 320
        AND (
            identifier_type <> 'mobile'
            OR normalized_value::text ~ '^\+[1-9][0-9]{7,14}$'
        )
    ),
    CONSTRAINT ck_app_user_identifiers_hint CHECK (
        display_hint = btrim(display_hint) AND display_hint <> '' AND length(display_hint) <= 160
    ),
    CONSTRAINT ck_app_user_identifiers_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_app_user_identifiers_lock CHECK (lock_version > 0),
    CONSTRAINT uq_app_user_identifiers_user_type UNIQUE (app_id, user_id, identifier_type),
    CONSTRAINT uq_app_user_identifiers_value_type UNIQUE (app_id, identifier_type, normalized_value)
);

CREATE INDEX idx_user_login_identifiers_user
    ON app.user_login_identifiers (app_id, user_id, status, identifier_type);
CREATE TRIGGER tr_user_login_identifiers_touch_updated_at
BEFORE UPDATE ON app.user_login_identifiers
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

INSERT INTO app.user_login_identifiers (
    tenant_id, app_id, user_id, identifier_type, normalized_value, display_hint,
    verified_at, status
)
SELECT m.tenant_id, m.app_id, m.user_id, 'email', lower(btrim(u.email::text)),
       CASE
           WHEN position('@' IN u.email::text) > 1
               THEN left(u.email::text, 1) || '***@' || split_part(u.email::text, '@', 2)
           ELSE '***'
       END,
       u.email_verified_at, 'active'
FROM app.user_memberships m
JOIN iam.users u ON u.id = m.user_id
WHERE u.email IS NOT NULL
  AND length(lower(btrim(u.email::text))) <= 254
  AND lower(btrim(u.email::text)) ~ '^[a-z0-9][a-z0-9._%+~-]{0,63}@[a-z0-9][a-z0-9.-]*[a-z0-9]$'
  AND split_part(lower(btrim(u.email::text)), '@', 2) LIKE '%.%'
  AND lower(btrim(u.email::text)) NOT LIKE '%..%'
ON CONFLICT (app_id, user_id, identifier_type) DO NOTHING;

INSERT INTO app.user_login_identifiers (
    tenant_id, app_id, user_id, identifier_type, normalized_value, display_hint,
    verified_at, status
)
SELECT m.tenant_id, m.app_id, m.user_id, 'mobile', btrim(u.mobile),
       CASE
           WHEN length(btrim(u.mobile)) >= 7
               THEN left(btrim(u.mobile), 3) || '****' || right(btrim(u.mobile), 4)
           ELSE '***'
       END,
       u.mobile_verified_at, 'active'
FROM app.user_memberships m
JOIN iam.users u ON u.id = m.user_id
WHERE u.mobile IS NOT NULL AND btrim(u.mobile) ~ '^\+[1-9][0-9]{7,14}$'
ON CONFLICT (app_id, user_id, identifier_type) DO NOTHING;

DO $$
DECLARE
    skipped_email_count bigint;
    skipped_mobile_count bigint;
BEGIN
    SELECT count(*) INTO skipped_email_count
    FROM app.user_memberships m
    JOIN iam.users u ON u.id=m.user_id
    WHERE u.email IS NOT NULL AND btrim(u.email::text)<>''
      AND NOT (
        length(lower(btrim(u.email::text))) <= 254
        AND lower(btrim(u.email::text)) ~ '^[a-z0-9][a-z0-9._%+~-]{0,63}@[a-z0-9][a-z0-9.-]*[a-z0-9]$'
        AND split_part(lower(btrim(u.email::text)), '@', 2) LIKE '%.%'
        AND lower(btrim(u.email::text)) NOT LIKE '%..%'
      );
    SELECT count(*) INTO skipped_mobile_count
    FROM app.user_memberships m
    JOIN iam.users u ON u.id=m.user_id
    WHERE u.mobile IS NOT NULL AND btrim(u.mobile)<>''
      AND btrim(u.mobile) !~ '^\+[1-9][0-9]{7,14}$';
    RAISE NOTICE 'login-provider identifier backfill skipped % suspicious email memberships and % non-E.164 mobile memberships', skipped_email_count, skipped_mobile_count;
END $$;

ALTER TABLE app.user_memberships DROP CONSTRAINT ck_app_user_memberships_source;
ALTER TABLE app.user_memberships ADD CONSTRAINT ck_app_user_memberships_source
    CHECK (source IN ('self_registration', 'federated_registration', 'admin_created', 'legacy'));

ALTER TABLE iam.verification_challenges DROP CONSTRAINT ck_verification_type;
ALTER TABLE iam.verification_challenges ADD CONSTRAINT ck_verification_type CHECK (
    challenge_type IN ('email_otp', 'sms_otp', 'password_reset', 'email_verify', 'mobile_verify', 'login_otp', 'account_delete', 'step_up')
);

INSERT INTO iam.permissions (
    code, name, module_code, resource_name, action_name, permission_kind,
    http_methods, route_pattern, description, status
) VALUES
    ('sys.login_provider_config.read', '查看第三方登录配置', 'sys', 'login_provider_config', 'read', 'api', ARRAY['GET'], '/admin-api/v1/login-provider-configs', 'Read tenant login provider configurations without secret values', 'active'),
    ('sys.login_provider_config.create', '创建第三方登录配置', 'sys', 'login_provider_config', 'create', 'api', ARRAY['POST'], '/admin-api/v1/login-provider-configs', 'Create tenant login provider configurations', 'active'),
    ('sys.login_provider_config.update', '更新第三方登录配置', 'sys', 'login_provider_config', 'update', 'api', ARRAY['PATCH','POST'], '/admin-api/v1/login-provider-configs/{id}', 'Update and change login provider configuration lifecycle', 'active'),
    ('sys.login_provider_config.delete', '删除第三方登录配置', 'sys', 'login_provider_config', 'delete', 'api', ARRAY['DELETE'], '/admin-api/v1/login-provider-configs/{id}', 'Delete an unused login provider configuration', 'active'),
    ('sys.login_provider_config.rotate_secret', '轮换第三方登录凭据', 'sys', 'login_provider_config', 'rotate_secret', 'api', ARRAY['POST'], '/admin-api/v1/login-provider-configs/{id}/rotate-secret', 'Rotate encrypted login provider credentials', 'active'),
    ('sys.login_provider_config.preflight', '预检第三方登录配置', 'sys', 'login_provider_config', 'preflight', 'api', ARRAY['POST'], '/admin-api/v1/login-provider-configs/{id}/preflight', 'Validate registered provider fields and credential readiness', 'active'),
    ('app.login_provider_binding.read', '查看应用第三方登录绑定', 'app', 'login_provider_binding', 'read', 'api', ARRAY['GET'], '/admin-api/v1/apps/{app_id}/login-provider-bindings', 'Read App login provider bindings', 'active'),
    ('app.login_provider_binding.update', '更新应用第三方登录绑定', 'app', 'login_provider_binding', 'update', 'api', ARRAY['PUT'], '/admin-api/v1/apps/{app_id}/login-provider-bindings', 'Atomically replace App login provider bindings', 'active')
ON CONFLICT (code) DO UPDATE SET
    name=EXCLUDED.name, module_code=EXCLUDED.module_code, resource_name=EXCLUDED.resource_name,
    action_name=EXCLUDED.action_name, permission_kind=EXCLUDED.permission_kind,
    http_methods=EXCLUDED.http_methods, route_pattern=EXCLUDED.route_pattern,
    description=EXCLUDED.description, status='active', updated_at=now();

INSERT INTO iam.role_permissions(tenant_id,role_id,permission_id,granted_by)
SELECT existing.tenant_id,existing.role_id,target.id,existing.granted_by
FROM iam.role_permissions existing
JOIN iam.permissions source ON source.id=existing.permission_id
JOIN (VALUES
    ('sys.config.read','sys.login_provider_config.read'),
    ('sys.config.create','sys.login_provider_config.create'),
    ('sys.config.update','sys.login_provider_config.update'),
    ('sys.config.update','sys.login_provider_config.delete'),
    ('sys.config.update','sys.login_provider_config.preflight'),
    ('sys.config.rotate_secret','sys.login_provider_config.rotate_secret'),
    ('app.application.read','app.login_provider_binding.read'),
    ('app.application.update','app.login_provider_binding.update')
) AS inheritance(source_code,target_code) ON inheritance.source_code=source.code
JOIN iam.permissions target ON target.code=inheritance.target_code
ON CONFLICT (tenant_id,role_id,permission_id) DO NOTHING;

INSERT INTO sys.menus (
    parent_id, permission_id, code, title, menu_type, route_path, component_key,
    icon, sort_order, status, metadata
)
SELECT parent.id, permission.id, 'system.settings.login-providers', '第三方登录配置', 'page',
       '/system/settings/login-providers', 'system.settings.login-providers',
       'LoginOutlined', 16, 'active', '{"i18n_key":"menu.system.settings.login_providers"}'::jsonb
FROM sys.menus parent
JOIN iam.permissions permission ON permission.code = 'sys.login_provider_config.read'
WHERE parent.code = 'system.settings' AND parent.tenant_id IS NULL
ON CONFLICT (code) WHERE tenant_id IS NULL AND deleted_at IS NULL DO UPDATE SET
    parent_id=EXCLUDED.parent_id, permission_id=EXCLUDED.permission_id,
    title=EXCLUDED.title, menu_type=EXCLUDED.menu_type, route_path=EXCLUDED.route_path,
    component_key=EXCLUDED.component_key, icon=EXCLUDED.icon, sort_order=EXCLUDED.sort_order,
    status='active', metadata=EXCLUDED.metadata, updated_at=now();

COMMENT ON TABLE sys.login_provider_configs IS 'Tenant-owned, code-registered third-party login configurations; secret plaintext is never returned or logged.';
COMMENT ON TABLE app.application_login_provider_bindings IS 'App selection and runtime enablement of tenant login-provider configurations.';
COMMENT ON TABLE iam.app_oauth_accounts IS 'App-scoped external identities for Mobile; independent from Admin iam.oauth_accounts.';
COMMENT ON TABLE iam.oauth_authorization_flows IS 'Short-lived hash/encrypted OAuth state with single-consumption semantics.';
COMMENT ON TABLE app.user_login_identifiers IS 'App-scoped email/mobile identifiers used by Mobile login and account binding.';

COMMIT;
