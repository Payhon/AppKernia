-- AppKernia PostgreSQL 18 Full Schema (57 tables, Billing optional)
-- Generated from versioned migrations. Prefer golang-migrate in real deployments.

-- ============================================================================
-- SOURCE: db/migrations/000001_extensions_and_schemas.up.sql
-- ============================================================================
-- AppKernia (AK) PostgreSQL 18 baseline
-- Target: PostgreSQL 18.x. UUIDv7 is provided by PostgreSQL core.

BEGIN;

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE SCHEMA IF NOT EXISTS iam;
CREATE SCHEMA IF NOT EXISTS org;
CREATE SCHEMA IF NOT EXISTS sys;
CREATE SCHEMA IF NOT EXISTS storage;
CREATE SCHEMA IF NOT EXISTS notify;
CREATE SCHEMA IF NOT EXISTS jobs;
CREATE SCHEMA IF NOT EXISTS audit;

CREATE OR REPLACE FUNCTION sys.touch_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

COMMIT;

-- ============================================================================
-- SOURCE: db/migrations/000002_iam.up.sql
-- ============================================================================
BEGIN;

CREATE TABLE iam.tenants (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    code                public.citext NOT NULL,
    name                varchar(120) NOT NULL,
    status              varchar(20) NOT NULL DEFAULT 'active',
    plan_code           varchar(64),
    settings            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT uq_tenants_code UNIQUE (code),
    CONSTRAINT ck_tenants_status CHECK (status IN ('active', 'disabled', 'archived'))
);

CREATE TABLE iam.users (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    username            public.citext,
    email               public.citext,
    mobile              varchar(32),
    display_name        varchar(120) NOT NULL,
    real_name           varchar(120),
    avatar_file_id      uuid,
    gender              varchar(16) NOT NULL DEFAULT 'unknown',
    birthday            date,
    bio                 varchar(500),
    locale              varchar(32) NOT NULL DEFAULT 'zh-CN',
    time_zone           varchar(64) NOT NULL DEFAULT 'UTC',
    status              varchar(20) NOT NULL DEFAULT 'pending',
    email_verified_at   timestamptz,
    mobile_verified_at  timestamptz,
    last_login_at       timestamptz,
    last_active_at      timestamptz,
    lock_version        integer NOT NULL DEFAULT 1,
    is_system           boolean NOT NULL DEFAULT false,
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT ck_users_gender CHECK (gender IN ('unknown', 'male', 'female', 'other')),
    CONSTRAINT ck_users_status CHECK (status IN ('pending', 'active', 'disabled', 'locked', 'deleted')),
    CONSTRAINT ck_users_lock_version CHECK (lock_version > 0)
);

CREATE UNIQUE INDEX uq_users_username_active
    ON iam.users (username)
    WHERE username IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_users_email_active
    ON iam.users (email)
    WHERE email IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_users_mobile_active
    ON iam.users (mobile)
    WHERE mobile IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_users_display_name_trgm
    ON iam.users USING gin (display_name gin_trgm_ops);
CREATE INDEX idx_users_status ON iam.users (status) WHERE deleted_at IS NULL;

CREATE TABLE iam.user_credentials (
    user_id                  uuid PRIMARY KEY REFERENCES iam.users(id) ON DELETE CASCADE,
    password_hash            text NOT NULL,
    password_algorithm       varchar(32) NOT NULL DEFAULT 'argon2id',
    password_version         integer NOT NULL DEFAULT 1,
    password_changed_at      timestamptz NOT NULL DEFAULT now(),
    force_password_change    boolean NOT NULL DEFAULT false,
    failed_attempts          integer NOT NULL DEFAULT 0,
    locked_until             timestamptz,
    updated_at               timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_credentials_algorithm CHECK (password_algorithm IN ('argon2id')),
    CONSTRAINT ck_credentials_password_version CHECK (password_version > 0),
    CONSTRAINT ck_credentials_failed_attempts CHECK (failed_attempts >= 0)
);

CREATE TABLE iam.password_history (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id             uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    password_hash       text NOT NULL,
    password_algorithm  varchar(32) NOT NULL DEFAULT 'argon2id',
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_password_history_algorithm CHECK (password_algorithm IN ('argon2id'))
);
CREATE INDEX idx_password_history_user_time ON iam.password_history (user_id, created_at DESC);

CREATE TABLE iam.oauth_accounts (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id             uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    provider            varchar(64) NOT NULL,
    subject             varchar(255) NOT NULL,
    union_subject       varchar(255),
    provider_username   varchar(255),
    provider_profile    jsonb NOT NULL DEFAULT '{}'::jsonb,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_oauth_provider_subject UNIQUE (provider, subject),
    CONSTRAINT ck_oauth_status CHECK (status IN ('active', 'disabled'))
);
CREATE INDEX idx_oauth_user ON iam.oauth_accounts (user_id);
CREATE UNIQUE INDEX uq_oauth_account_user_provider ON iam.oauth_accounts (user_id, provider);

CREATE TABLE iam.oauth_binding_challenges (
    id                          uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id                     uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    tenant_id                   uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    provider                    varchar(64) NOT NULL,
    state_hash                  bytea NOT NULL,
    authorization_code_hash     bytea NOT NULL,
    pkce_verifier_encrypted     bytea NOT NULL,
    pkce_challenge              varchar(128) NOT NULL,
    expires_at                  timestamptz NOT NULL,
    consumed_at                 timestamptz,
    created_at                  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_oauth_binding_state_hash UNIQUE (state_hash),
    CONSTRAINT ck_oauth_binding_provider CHECK (provider ~ '^[a-z][a-z0-9-]{1,62}$'),
    CONSTRAINT ck_oauth_binding_pkce_challenge CHECK (length(pkce_challenge) BETWEEN 43 AND 128),
    CONSTRAINT ck_oauth_binding_expiry CHECK (expires_at > created_at)
);
CREATE INDEX idx_oauth_binding_user_pending
    ON iam.oauth_binding_challenges (user_id, provider, expires_at DESC)
    WHERE consumed_at IS NULL;

CREATE TABLE iam.tenant_members (
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    user_id             uuid NOT NULL REFERENCES iam.users(id) ON DELETE RESTRICT,
    member_number       varchar(64),
    display_name        varchar(120),
    status              varchar(20) NOT NULL DEFAULT 'active',
    invited_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    invited_at          timestamptz,
    joined_at           timestamptz NOT NULL DEFAULT now(),
    left_at             timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id),
    CONSTRAINT uq_tenant_member_number UNIQUE (tenant_id, member_number),
    CONSTRAINT ck_tenant_members_status CHECK (status IN ('invited', 'active', 'suspended', 'left'))
);
CREATE INDEX idx_tenant_members_user ON iam.tenant_members (user_id, status);

CREATE TABLE iam.roles (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    parent_id           uuid,
    code                public.citext NOT NULL,
    name                varchar(120) NOT NULL,
    description         varchar(500),
    role_type           varchar(20) NOT NULL DEFAULT 'custom',
    data_scope          varchar(32) NOT NULL DEFAULT 'self',
    sort_order          integer NOT NULL DEFAULT 0,
    is_default          boolean NOT NULL DEFAULT false,
    is_system           boolean NOT NULL DEFAULT false,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT uq_roles_tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT fk_roles_parent_same_tenant
        FOREIGN KEY (tenant_id, parent_id) REFERENCES iam.roles(tenant_id, id)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT ck_roles_type CHECK (role_type IN ('system', 'custom')),
    CONSTRAINT ck_roles_data_scope CHECK (
        data_scope IN ('all', 'tenant', 'department', 'department_tree', 'self', 'custom')
    ),
    CONSTRAINT ck_roles_status CHECK (status IN ('active', 'disabled'))
);
CREATE UNIQUE INDEX uq_roles_tenant_code_active
    ON iam.roles (tenant_id, code)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_roles_tenant_status ON iam.roles (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_roles_parent ON iam.roles (tenant_id, parent_id) WHERE deleted_at IS NULL;

CREATE TABLE iam.permissions (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    code                public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    module_code         varchar(64) NOT NULL,
    resource_name       varchar(96) NOT NULL,
    action_name         varchar(64) NOT NULL,
    permission_kind     varchar(20) NOT NULL DEFAULT 'api',
    http_methods        text[] NOT NULL DEFAULT '{}'::text[],
    route_pattern       varchar(500),
    description         varchar(500),
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_permissions_code UNIQUE (code),
    CONSTRAINT ck_permissions_kind CHECK (permission_kind IN ('api', 'ui_action', 'feature')),
    CONSTRAINT ck_permissions_status CHECK (status IN ('active', 'disabled'))
);
CREATE INDEX idx_permissions_module ON iam.permissions (module_code, resource_name, action_name);

CREATE TABLE iam.user_roles (
    tenant_id           uuid NOT NULL,
    user_id             uuid NOT NULL,
    role_id             uuid NOT NULL,
    valid_from          timestamptz NOT NULL DEFAULT now(),
    valid_until         timestamptz,
    granted_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id, role_id),
    CONSTRAINT fk_user_roles_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role
        FOREIGN KEY (tenant_id, role_id) REFERENCES iam.roles(tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT ck_user_roles_validity CHECK (valid_until IS NULL OR valid_until > valid_from)
);
CREATE INDEX idx_user_roles_role ON iam.user_roles (tenant_id, role_id);

CREATE TABLE iam.role_permissions (
    tenant_id           uuid NOT NULL,
    role_id             uuid NOT NULL,
    permission_id       uuid NOT NULL REFERENCES iam.permissions(id) ON DELETE CASCADE,
    granted_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, role_id, permission_id),
    CONSTRAINT fk_role_permissions_role
        FOREIGN KEY (tenant_id, role_id) REFERENCES iam.roles(tenant_id, id)
        ON DELETE CASCADE
);
CREATE INDEX idx_role_permissions_permission ON iam.role_permissions (permission_id);

CREATE TABLE iam.devices (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id             uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    device_key          varchar(255) NOT NULL,
    device_name         varchar(160),
    platform            varchar(32) NOT NULL,
    model               varchar(120),
    os_version          varchar(64),
    app_version         varchar(64),
    trusted_until       timestamptz,
    last_ip             inet,
    last_seen_at        timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_devices_user_key UNIQUE (user_id, device_key),
    CONSTRAINT uq_devices_user_id_id UNIQUE (user_id, id),
    CONSTRAINT ck_devices_platform CHECK (platform IN ('android', 'ios', 'harmonyos', 'web', 'desktop', 'unknown'))
);

CREATE TABLE iam.sessions (
    id                      uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id                 uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    tenant_id               uuid REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    device_id               uuid REFERENCES iam.devices(id) ON DELETE SET NULL,
    audience                varchar(32) NOT NULL,
    status                  varchar(20) NOT NULL DEFAULT 'active',
    access_token_version    integer NOT NULL DEFAULT 1,
    mfa_level               smallint NOT NULL DEFAULT 0,
    ip_address              inet,
    user_agent              varchar(1000),
    last_seen_at            timestamptz NOT NULL DEFAULT now(),
    idle_expires_at         timestamptz,
    absolute_expires_at     timestamptz NOT NULL,
    revoked_at              timestamptz,
    revoke_reason           varchar(255),
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_sessions_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT ck_sessions_audience CHECK (audience IN ('ak-mobile', 'ak-admin', 'ak-api')),
    CONSTRAINT ck_sessions_status CHECK (status IN ('active', 'revoked', 'expired')),
    CONSTRAINT ck_sessions_token_version CHECK (access_token_version > 0),
    CONSTRAINT ck_sessions_mfa_level CHECK (mfa_level BETWEEN 0 AND 3),
    CONSTRAINT ck_sessions_expiry CHECK (absolute_expires_at > created_at)
);
CREATE INDEX idx_sessions_user_active ON iam.sessions (user_id, last_seen_at DESC) WHERE revoked_at IS NULL;
CREATE INDEX idx_sessions_tenant_active ON iam.sessions (tenant_id, last_seen_at DESC) WHERE revoked_at IS NULL;

CREATE TABLE iam.refresh_tokens (
    id                      uuid PRIMARY KEY DEFAULT uuidv7(),
    session_id              uuid NOT NULL REFERENCES iam.sessions(id) ON DELETE CASCADE,
    token_hash              bytea NOT NULL,
    parent_token_id         uuid REFERENCES iam.refresh_tokens(id) ON DELETE SET NULL,
    replaced_by_token_id    uuid REFERENCES iam.refresh_tokens(id) ON DELETE SET NULL,
    issued_at               timestamptz NOT NULL DEFAULT now(),
    expires_at              timestamptz NOT NULL,
    consumed_at             timestamptz,
    revoked_at              timestamptz,
    reuse_detected_at       timestamptz,
    created_ip              inet,
    CONSTRAINT uq_refresh_tokens_hash UNIQUE (token_hash),
    CONSTRAINT ck_refresh_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT ck_refresh_tokens_expiry CHECK (expires_at > issued_at)
);
CREATE INDEX idx_refresh_tokens_session ON iam.refresh_tokens (session_id, issued_at DESC);
CREATE INDEX idx_refresh_tokens_expiry ON iam.refresh_tokens (expires_at) WHERE revoked_at IS NULL;

CREATE TABLE iam.verification_challenges (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id             uuid REFERENCES iam.users(id) ON DELETE CASCADE,
    challenge_type      varchar(40) NOT NULL,
    target_hash         bytea NOT NULL,
    target_hint         varchar(160),
    secret_hash         bytea NOT NULL,
    attempts            integer NOT NULL DEFAULT 0,
    max_attempts        integer NOT NULL DEFAULT 5,
    expires_at          timestamptz NOT NULL,
    consumed_at         timestamptz,
    created_ip          inet,
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_verification_target_hash CHECK (octet_length(target_hash) = 32),
    CONSTRAINT ck_verification_type CHECK (
        challenge_type IN ('email_otp', 'sms_otp', 'password_reset', 'email_verify', 'mobile_verify', 'login_otp')
    ),
    CONSTRAINT ck_verification_attempts CHECK (attempts >= 0 AND max_attempts > 0),
    CONSTRAINT ck_verification_expiry CHECK (expires_at > created_at)
);
CREATE INDEX idx_verification_target_type ON iam.verification_challenges (target_hash, challenge_type, created_at DESC);
CREATE INDEX idx_verification_expiry ON iam.verification_challenges (expires_at) WHERE consumed_at IS NULL;

CREATE TABLE iam.mfa_factors (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id             uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    factor_type         varchar(24) NOT NULL,
    display_name        varchar(120),
    secret_encrypted    bytea,
    credential_id       bytea,
    public_key          bytea,
    sign_count          bigint NOT NULL DEFAULT 0,
    status              varchar(20) NOT NULL DEFAULT 'pending',
    verified_at         timestamptz,
    last_used_at        timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_mfa_factor_type CHECK (factor_type IN ('totp', 'webauthn')),
    CONSTRAINT ck_mfa_status CHECK (status IN ('pending', 'active', 'disabled')),
    CONSTRAINT ck_mfa_material CHECK (
        (factor_type = 'totp' AND secret_encrypted IS NOT NULL)
        OR (factor_type = 'webauthn' AND credential_id IS NOT NULL AND public_key IS NOT NULL)
    )
);
CREATE UNIQUE INDEX uq_mfa_webauthn_credential
    ON iam.mfa_factors (credential_id)
    WHERE credential_id IS NOT NULL;

CREATE TABLE iam.mfa_recovery_codes (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id             uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    code_hash           bytea NOT NULL,
    used_at             timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_mfa_recovery_code_hash UNIQUE (code_hash)
);
CREATE INDEX idx_mfa_recovery_user_unused ON iam.mfa_recovery_codes (user_id) WHERE used_at IS NULL;

CREATE TABLE iam.block_rules (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE CASCADE,
    subject_type        varchar(24) NOT NULL,
    subject_value       varchar(512) NOT NULL,
    action              varchar(24) NOT NULL DEFAULT 'deny',
    reason              varchar(500),
    starts_at           timestamptz NOT NULL DEFAULT now(),
    expires_at          timestamptz,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_block_subject_type CHECK (subject_type IN ('ip', 'cidr', 'user', 'device', 'identifier')),
    CONSTRAINT ck_block_action CHECK (action IN ('deny', 'challenge', 'rate_limit')),
    CONSTRAINT ck_block_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_block_expiry CHECK (expires_at IS NULL OR expires_at > starts_at)
);
CREATE INDEX idx_block_rules_lookup ON iam.block_rules (subject_type, subject_value, status);

CREATE TRIGGER tr_tenants_touch_updated_at
BEFORE UPDATE ON iam.tenants FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_users_touch_updated_at
BEFORE UPDATE ON iam.users FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_credentials_touch_updated_at
BEFORE UPDATE ON iam.user_credentials FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_oauth_touch_updated_at
BEFORE UPDATE ON iam.oauth_accounts FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_tenant_members_touch_updated_at
BEFORE UPDATE ON iam.tenant_members FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_roles_touch_updated_at
BEFORE UPDATE ON iam.roles FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_permissions_touch_updated_at
BEFORE UPDATE ON iam.permissions FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_devices_touch_updated_at
BEFORE UPDATE ON iam.devices FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_sessions_touch_updated_at
BEFORE UPDATE ON iam.sessions FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_mfa_factors_touch_updated_at
BEFORE UPDATE ON iam.mfa_factors FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_block_rules_touch_updated_at
BEFORE UPDATE ON iam.block_rules FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMENT ON TABLE iam.users IS 'Global user identity; tenant membership, credentials and OAuth accounts are normalized into separate tables.';
COMMENT ON TABLE iam.sessions IS 'Revocable login session for mobile, admin web or machine API audiences.';
COMMENT ON TABLE iam.refresh_tokens IS 'Opaque rotating refresh tokens stored only as SHA-256 hashes; reuse revokes the session family.';
COMMENT ON TABLE iam.permissions IS 'Stable permission catalog. Backend route code references permission codes; database route fields are documentation only.';

COMMIT;

-- ============================================================================
-- SOURCE: db/migrations/000003_org_and_scope.up.sql
-- ============================================================================
BEGIN;

CREATE TABLE org.units (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    parent_id           uuid,
    code                public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    unit_type           varchar(24) NOT NULL DEFAULT 'department',
    leader_user_id      uuid,
    phone               varchar(32),
    email               public.citext,
    sort_order          integer NOT NULL DEFAULT 0,
    status              varchar(20) NOT NULL DEFAULT 'active',
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT uq_org_units_tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT fk_org_units_leader_member
        FOREIGN KEY (tenant_id, leader_user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_org_units_parent_same_tenant
        FOREIGN KEY (tenant_id, parent_id) REFERENCES org.units(tenant_id, id)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT ck_org_units_type CHECK (unit_type IN ('company', 'division', 'department', 'team', 'group')),
    CONSTRAINT ck_org_units_status CHECK (status IN ('active', 'disabled'))
);
CREATE UNIQUE INDEX uq_org_units_tenant_code_active
    ON org.units (tenant_id, code)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_org_units_parent ON org.units (tenant_id, parent_id, sort_order) WHERE deleted_at IS NULL;
CREATE INDEX idx_org_units_name_trgm ON org.units USING gin (name gin_trgm_ops);

CREATE TABLE org.positions (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    code                public.citext NOT NULL,
    name                varchar(120) NOT NULL,
    description         varchar(500),
    sort_order          integer NOT NULL DEFAULT 0,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT uq_positions_tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT ck_positions_status CHECK (status IN ('active', 'disabled'))
);

CREATE UNIQUE INDEX uq_positions_tenant_code_active
    ON org.positions (tenant_id, code)
    WHERE deleted_at IS NULL;

CREATE TABLE org.user_units (
    tenant_id           uuid NOT NULL,
    user_id             uuid NOT NULL,
    unit_id             uuid NOT NULL,
    is_primary          boolean NOT NULL DEFAULT false,
    joined_at           timestamptz NOT NULL DEFAULT now(),
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id, unit_id),
    CONSTRAINT fk_user_units_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_user_units_unit
        FOREIGN KEY (tenant_id, unit_id) REFERENCES org.units(tenant_id, id)
        ON DELETE CASCADE
);
CREATE UNIQUE INDEX uq_user_primary_unit
    ON org.user_units (tenant_id, user_id)
    WHERE is_primary;
CREATE INDEX idx_user_units_unit ON org.user_units (tenant_id, unit_id, user_id);

CREATE TABLE org.user_positions (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL,
    user_id             uuid NOT NULL,
    position_id         uuid NOT NULL,
    unit_id             uuid,
    is_primary          boolean NOT NULL DEFAULT false,
    assigned_at         timestamptz NOT NULL DEFAULT now(),
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_positions_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_user_positions_position
        FOREIGN KEY (tenant_id, position_id) REFERENCES org.positions(tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_user_positions_unit
        FOREIGN KEY (tenant_id, unit_id) REFERENCES org.units(tenant_id, id)
        ON DELETE CASCADE
);
CREATE UNIQUE INDEX uq_user_positions_with_unit
    ON org.user_positions (tenant_id, user_id, position_id, unit_id)
    WHERE unit_id IS NOT NULL;
CREATE UNIQUE INDEX uq_user_positions_without_unit
    ON org.user_positions (tenant_id, user_id, position_id)
    WHERE unit_id IS NULL;
CREATE UNIQUE INDEX uq_user_primary_position
    ON org.user_positions (tenant_id, user_id)
    WHERE is_primary;
CREATE INDEX idx_user_positions_position ON org.user_positions (tenant_id, position_id);

CREATE TABLE iam.role_scope_units (
    tenant_id           uuid NOT NULL,
    role_id             uuid NOT NULL,
    unit_id             uuid NOT NULL,
    include_descendants boolean NOT NULL DEFAULT false,
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, role_id, unit_id),
    CONSTRAINT fk_role_scope_role
        FOREIGN KEY (tenant_id, role_id) REFERENCES iam.roles(tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_role_scope_unit
        FOREIGN KEY (tenant_id, unit_id) REFERENCES org.units(tenant_id, id)
        ON DELETE CASCADE
);

CREATE TRIGGER tr_org_units_touch_updated_at
BEFORE UPDATE ON org.units FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_positions_touch_updated_at
BEFORE UPDATE ON org.positions FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMENT ON TABLE org.units IS 'Organization tree stored as an adjacency list. Descendant queries use PostgreSQL recursive CTEs; no duplicated level/tree strings.';
COMMENT ON TABLE iam.role_scope_units IS 'Custom organization data scope assigned to a role.';

COMMIT;

-- ============================================================================
-- SOURCE: db/migrations/000004_system_and_storage.up.sql
-- ============================================================================
BEGIN;

CREATE TABLE sys.modules (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    code                public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    version             varchar(64) NOT NULL,
    description         varchar(1000),
    capabilities        jsonb NOT NULL DEFAULT '{}'::jsonb,
    config_schema       jsonb NOT NULL DEFAULT '{}'::jsonb,
    status              varchar(20) NOT NULL DEFAULT 'enabled',
    installed_at        timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_modules_code UNIQUE (code),
    CONSTRAINT ck_modules_status CHECK (status IN ('enabled', 'disabled'))
);

CREATE TABLE sys.menus (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE CASCADE,
    parent_id           uuid REFERENCES sys.menus(id) ON DELETE RESTRICT,
    permission_id       uuid REFERENCES iam.permissions(id) ON DELETE SET NULL,
    code                public.citext NOT NULL,
    title               varchar(160) NOT NULL,
    menu_type           varchar(20) NOT NULL,
    route_name          varchar(160),
    route_path          varchar(500),
    component_key       varchar(255),
    redirect_path       varchar(500),
    icon                varchar(160),
    external_url        varchar(2000),
    open_mode           varchar(20) NOT NULL DEFAULT 'same_tab',
    active_menu_code    varchar(160),
    keep_alive          boolean NOT NULL DEFAULT false,
    hidden              boolean NOT NULL DEFAULT false,
    affix               boolean NOT NULL DEFAULT false,
    always_show         boolean NOT NULL DEFAULT false,
    sort_order          integer NOT NULL DEFAULT 0,
    status              varchar(20) NOT NULL DEFAULT 'active',
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT ck_menus_type CHECK (menu_type IN ('directory', 'page', 'external')),
    CONSTRAINT ck_menus_open_mode CHECK (open_mode IN ('same_tab', 'new_tab', 'iframe')),
    CONSTRAINT ck_menus_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_menus_page_fields CHECK (
        menu_type <> 'page' OR (route_path IS NOT NULL AND component_key IS NOT NULL)
    ),
    CONSTRAINT ck_menus_external_fields CHECK (
        menu_type <> 'external' OR external_url IS NOT NULL
    )
);
CREATE UNIQUE INDEX uq_menus_global_code
    ON sys.menus (code)
    WHERE tenant_id IS NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_menus_tenant_code
    ON sys.menus (tenant_id, code)
    WHERE tenant_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_menus_parent_sort ON sys.menus (tenant_id, parent_id, sort_order) WHERE deleted_at IS NULL;

CREATE TABLE sys.role_menus (
    tenant_id           uuid NOT NULL,
    role_id             uuid NOT NULL,
    menu_id             uuid NOT NULL REFERENCES sys.menus(id) ON DELETE CASCADE,
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, role_id, menu_id),
    CONSTRAINT fk_role_menus_role
        FOREIGN KEY (tenant_id, role_id) REFERENCES iam.roles(tenant_id, id)
        ON DELETE CASCADE
);
CREATE INDEX idx_role_menus_menu ON sys.role_menus (menu_id);

CREATE TABLE sys.config_items (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE CASCADE,
    module_code         varchar(64) NOT NULL DEFAULT 'core',
    config_group        varchar(96) NOT NULL DEFAULT 'default',
    config_key          public.citext NOT NULL,
    display_name        varchar(160) NOT NULL,
    value_type          varchar(24) NOT NULL DEFAULT 'string',
    value_json          jsonb,
    default_value_json  jsonb,
    is_secret           boolean NOT NULL DEFAULT false,
    secret_ciphertext   bytea,
    secret_key_version  integer,
    is_public           boolean NOT NULL DEFAULT false,
    validation_schema   jsonb NOT NULL DEFAULT '{}'::jsonb,
    description         varchar(1000),
    sort_order          integer NOT NULL DEFAULT 0,
    status              varchar(20) NOT NULL DEFAULT 'active',
    version             integer NOT NULL DEFAULT 1,
    created_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_config_value_type CHECK (value_type IN ('string', 'integer', 'decimal', 'boolean', 'json', 'datetime')),
    CONSTRAINT ck_config_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_config_version CHECK (version > 0),
    CONSTRAINT ck_config_secret_pair CHECK (
        (secret_ciphertext IS NULL AND secret_key_version IS NULL)
        OR (secret_ciphertext IS NOT NULL AND secret_key_version IS NOT NULL)
    ),
    CONSTRAINT ck_config_secret_storage CHECK (
        (is_secret AND value_json IS NULL AND default_value_json IS NULL AND NOT is_public)
        OR (NOT is_secret AND secret_ciphertext IS NULL AND secret_key_version IS NULL)
    )
);
CREATE UNIQUE INDEX uq_config_global_key
    ON sys.config_items (module_code, config_group, config_key)
    WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX uq_config_tenant_key
    ON sys.config_items (tenant_id, module_code, config_group, config_key)
    WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_config_public ON sys.config_items (tenant_id, module_code) WHERE is_public AND status = 'active';

CREATE TABLE sys.dict_types (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE CASCADE,
    code                public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    description         varchar(500),
    is_system           boolean NOT NULL DEFAULT false,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_dict_types_status CHECK (status IN ('active', 'disabled'))
);
CREATE UNIQUE INDEX uq_dict_type_global_code
    ON sys.dict_types (code)
    WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX uq_dict_type_tenant_code
    ON sys.dict_types (tenant_id, code)
    WHERE tenant_id IS NOT NULL;

CREATE TABLE sys.dict_items (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    dict_type_id        uuid NOT NULL REFERENCES sys.dict_types(id) ON DELETE CASCADE,
    item_value          varchar(255) NOT NULL,
    label               varchar(255) NOT NULL,
    locale              varchar(32),
    color               varchar(64),
    css_class           varchar(128),
    sort_order          integer NOT NULL DEFAULT 0,
    is_default          boolean NOT NULL DEFAULT false,
    extra               jsonb NOT NULL DEFAULT '{}'::jsonb,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_dict_items_locale CHECK (locale IS NULL OR btrim(locale) <> ''),
    CONSTRAINT ck_dict_items_status CHECK (status IN ('active', 'disabled'))
);
CREATE UNIQUE INDEX uq_dict_item_value_locale
    ON sys.dict_items (dict_type_id, item_value, COALESCE(locale, ''));
CREATE INDEX idx_dict_items_type_sort ON sys.dict_items (dict_type_id, sort_order, id);

CREATE TABLE sys.regions (
    code                varchar(32) PRIMARY KEY,
    parent_code         varchar(32) REFERENCES sys.regions(code) ON DELETE RESTRICT,
    level               smallint NOT NULL,
    name                varchar(160) NOT NULL,
    full_name           varchar(500),
    postal_code         varchar(24),
    longitude           numeric(11, 7),
    latitude            numeric(10, 7),
    status              varchar(20) NOT NULL DEFAULT 'active',
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_regions_level CHECK (level BETWEEN 0 AND 10),
    CONSTRAINT ck_regions_status CHECK (status IN ('active', 'disabled'))
);
CREATE INDEX idx_regions_parent ON sys.regions (parent_code, code);
CREATE INDEX idx_regions_name_trgm ON sys.regions USING gin (name gin_trgm_ops);

CREATE TABLE sys.api_clients (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    client_id           public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    description         varchar(500),
    allowed_cidrs       cidr[] NOT NULL DEFAULT '{}'::cidr[],
    status              varchar(20) NOT NULL DEFAULT 'active',
    expires_at          timestamptz,
    created_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_api_clients_client_id UNIQUE (client_id),
    CONSTRAINT uq_api_clients_tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT ck_api_clients_status CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE sys.api_client_secrets (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    api_client_id       uuid NOT NULL REFERENCES sys.api_clients(id) ON DELETE CASCADE,
    secret_prefix       varchar(24) NOT NULL,
    secret_hash         bytea NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    expires_at          timestamptz,
    revoked_at          timestamptz,
    last_used_at        timestamptz,
    CONSTRAINT uq_api_client_secret_hash UNIQUE (secret_hash),
    CONSTRAINT ck_api_client_secret_hash_length CHECK (octet_length(secret_hash) = 32)
);
CREATE INDEX idx_api_client_secrets_active ON sys.api_client_secrets (api_client_id, created_at DESC) WHERE revoked_at IS NULL;

CREATE TABLE sys.api_client_permissions (
    tenant_id           uuid NOT NULL,
    api_client_id       uuid NOT NULL,
    permission_id       uuid NOT NULL REFERENCES iam.permissions(id) ON DELETE CASCADE,
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, api_client_id, permission_id),
    CONSTRAINT fk_api_client_permissions_client
        FOREIGN KEY (tenant_id, api_client_id) REFERENCES sys.api_clients(tenant_id, id)
        ON DELETE CASCADE
);

CREATE TABLE sys.idempotency_keys (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    identity_type       varchar(24) NOT NULL,
    identity_id         uuid NOT NULL,
    idempotency_key     varchar(255) NOT NULL,
    request_hash        bytea NOT NULL,
    response_status     integer,
    response_headers    jsonb,
    response_body       bytea,
    locked_until        timestamptz,
    expires_at          timestamptz NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    completed_at        timestamptz,
    CONSTRAINT uq_idempotency_identity_key UNIQUE (tenant_id, identity_type, identity_id, idempotency_key),
    CONSTRAINT ck_idempotency_identity_type CHECK (identity_type IN ('user', 'api_client')),
    CONSTRAINT ck_idempotency_response_status CHECK (response_status IS NULL OR response_status BETWEEN 100 AND 599),
    CONSTRAINT ck_idempotency_request_hash_length CHECK (octet_length(request_hash) = 32),
    CONSTRAINT ck_idempotency_expiry CHECK (expires_at > created_at)
);
CREATE INDEX idx_idempotency_expiry ON sys.idempotency_keys (expires_at);

CREATE TABLE sys.webhook_endpoints (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    name                varchar(160) NOT NULL,
    endpoint_url        varchar(2000) NOT NULL,
    event_types         text[] NOT NULL DEFAULT '{}'::text[],
    secret_ciphertext   bytea NOT NULL,
    secret_key_version  integer NOT NULL,
    max_attempts        integer NOT NULL DEFAULT 8,
    timeout_seconds     integer NOT NULL DEFAULT 10,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_webhook_attempts CHECK (max_attempts BETWEEN 1 AND 100),
    CONSTRAINT ck_webhook_timeout CHECK (timeout_seconds BETWEEN 1 AND 60),
    CONSTRAINT ck_webhook_status CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE sys.webhook_deliveries (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    endpoint_id         uuid NOT NULL REFERENCES sys.webhook_endpoints(id) ON DELETE CASCADE,
    event_id            uuid NOT NULL,
    event_type          varchar(160) NOT NULL,
    payload             jsonb NOT NULL,
    status              varchar(20) NOT NULL DEFAULT 'pending',
    attempt_count       integer NOT NULL DEFAULT 0,
    next_attempt_at     timestamptz,
    response_status     integer,
    response_body       varchar(4000),
    last_error          varchar(2000),
    delivered_at        timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_webhook_event_endpoint UNIQUE (endpoint_id, event_id),
    CONSTRAINT ck_webhook_delivery_status CHECK (status IN ('pending', 'processing', 'succeeded', 'failed', 'cancelled')),
    CONSTRAINT ck_webhook_delivery_attempts CHECK (attempt_count >= 0),
    CONSTRAINT ck_webhook_delivery_response_status CHECK (response_status IS NULL OR response_status BETWEEN 100 AND 599)
);
CREATE INDEX idx_webhook_deliveries_due ON sys.webhook_deliveries (status, next_attempt_at) WHERE status IN ('pending', 'failed');

CREATE TABLE storage.files (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    owner_user_id       uuid,
    provider            varchar(32) NOT NULL,
    bucket_name         varchar(255) NOT NULL,
    object_key          varchar(1024) NOT NULL,
    original_name       varchar(1000) NOT NULL,
    media_type          varchar(255),
    extension           varchar(32),
    size_bytes          bigint NOT NULL,
    sha256              bytea,
    etag                varchar(255),
    visibility          varchar(20) NOT NULL DEFAULT 'private',
    status              varchar(20) NOT NULL DEFAULT 'pending',
    scan_status         varchar(20) NOT NULL DEFAULT 'pending',
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT uq_storage_object UNIQUE (provider, bucket_name, object_key),
    CONSTRAINT fk_storage_file_owner_member
        FOREIGN KEY (tenant_id, owner_user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT uq_storage_files_tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT ck_files_size CHECK (size_bytes >= 0),
    CONSTRAINT ck_files_sha256_length CHECK (sha256 IS NULL OR octet_length(sha256) = 32),
    CONSTRAINT ck_files_visibility CHECK (visibility IN ('private', 'public')),
    CONSTRAINT ck_files_status CHECK (status IN ('pending', 'ready', 'quarantined', 'deleted')),
    CONSTRAINT ck_files_scan_status CHECK (scan_status IN ('pending', 'clean', 'infected', 'failed', 'skipped'))
);
CREATE INDEX idx_files_tenant_time ON storage.files (tenant_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_owner ON storage.files (owner_user_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_files_tenant_dedup
    ON storage.files (tenant_id, sha256, size_bytes)
    WHERE sha256 IS NOT NULL AND status = 'ready' AND deleted_at IS NULL;
CREATE INDEX idx_files_name_trgm ON storage.files USING gin (original_name gin_trgm_ops);

CREATE TABLE storage.upload_sessions (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    user_id             uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    file_id             uuid,
    provider            varchar(32) NOT NULL,
    provider_upload_id  varchar(1000),
    bucket_name         varchar(255) NOT NULL,
    object_key          varchar(1024) NOT NULL,
    original_name       varchar(1000) NOT NULL,
    media_type          varchar(255),
    expected_size       bigint NOT NULL,
    expected_sha256     bytea,
    part_size           bigint,
    status              varchar(20) NOT NULL DEFAULT 'initiated',
    expires_at          timestamptz NOT NULL,
    completed_at        timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_upload_session_object UNIQUE (provider, bucket_name, object_key),
    CONSTRAINT fk_upload_sessions_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_upload_sessions_file
        FOREIGN KEY (tenant_id, file_id) REFERENCES storage.files(tenant_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT ck_upload_expected_size CHECK (expected_size >= 0),
    CONSTRAINT ck_upload_sha256_length CHECK (expected_sha256 IS NULL OR octet_length(expected_sha256) = 32),
    CONSTRAINT ck_upload_part_size CHECK (part_size IS NULL OR part_size > 0),
    CONSTRAINT ck_upload_status CHECK (status IN ('initiated', 'uploading', 'completed', 'aborted', 'expired')),
    CONSTRAINT ck_upload_expiry CHECK (expires_at > created_at)
);
CREATE INDEX idx_upload_sessions_user ON storage.upload_sessions (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_upload_sessions_expiry ON storage.upload_sessions (expires_at) WHERE status IN ('initiated', 'uploading');

CREATE TABLE storage.upload_parts (
    upload_session_id   uuid NOT NULL REFERENCES storage.upload_sessions(id) ON DELETE CASCADE,
    part_number         integer NOT NULL,
    etag                varchar(255) NOT NULL,
    size_bytes          bigint NOT NULL,
    checksum_sha256     bytea,
    uploaded_at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (upload_session_id, part_number),
    CONSTRAINT ck_upload_part_number CHECK (part_number > 0),
    CONSTRAINT ck_upload_part_size_bytes CHECK (size_bytes > 0),
    CONSTRAINT ck_upload_part_checksum CHECK (checksum_sha256 IS NULL OR octet_length(checksum_sha256) = 32)
);

CREATE TABLE storage.file_usages (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    file_id             uuid NOT NULL,
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    module_code         varchar(64) NOT NULL,
    entity_type         varchar(128) NOT NULL,
    entity_id           uuid NOT NULL,
    field_name          varchar(128) NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_file_usage UNIQUE (file_id, module_code, entity_type, entity_id, field_name),
    CONSTRAINT fk_file_usage_file
        FOREIGN KEY (tenant_id, file_id) REFERENCES storage.files(tenant_id, id)
        ON DELETE CASCADE
);
CREATE INDEX idx_file_usage_entity ON storage.file_usages (tenant_id, module_code, entity_type, entity_id);

ALTER TABLE iam.users
    ADD CONSTRAINT fk_users_avatar_file
    FOREIGN KEY (avatar_file_id) REFERENCES storage.files(id) ON DELETE SET NULL;

CREATE TRIGGER tr_modules_touch_updated_at
BEFORE UPDATE ON sys.modules FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_menus_touch_updated_at
BEFORE UPDATE ON sys.menus FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_config_touch_updated_at
BEFORE UPDATE ON sys.config_items FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_dict_types_touch_updated_at
BEFORE UPDATE ON sys.dict_types FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_dict_items_touch_updated_at
BEFORE UPDATE ON sys.dict_items FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_api_clients_touch_updated_at
BEFORE UPDATE ON sys.api_clients FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_webhook_endpoints_touch_updated_at
BEFORE UPDATE ON sys.webhook_endpoints FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_webhook_deliveries_touch_updated_at
BEFORE UPDATE ON sys.webhook_deliveries FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_files_touch_updated_at
BEFORE UPDATE ON storage.files FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_upload_sessions_touch_updated_at
BEFORE UPDATE ON storage.upload_sessions FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMENT ON TABLE sys.menus IS 'Admin navigation metadata. UI actions and API authorization are represented by iam.permissions, not encoded as menu rows.';
COMMENT ON TABLE sys.config_items IS 'Typed global or tenant configuration; secret values must be envelope-encrypted and never stored in value_json.';
COMMENT ON TABLE sys.modules IS 'Compile-time module registry. AK does not load or uninstall arbitrary Go code at runtime.';
COMMENT ON TABLE storage.files IS 'Object storage metadata. Uploads use presigned URLs and post-upload validation/virus scanning.';

COMMIT;

-- ============================================================================
-- SOURCE: db/migrations/000005_notifications_jobs_audit.up.sql
-- ============================================================================
BEGIN;

CREATE TABLE notify.templates (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE CASCADE,
    code                public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    channel             varchar(20) NOT NULL,
    locale              varchar(32),
    subject_template    text,
    body_template       text NOT NULL,
    variables_schema    jsonb NOT NULL DEFAULT '{}'::jsonb,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_notify_templates_locale CHECK (locale IS NULL OR btrim(locale) <> ''),
    CONSTRAINT ck_notify_templates_channel CHECK (channel IN ('in_app', 'email', 'sms', 'push', 'webhook')),
    CONSTRAINT ck_notify_templates_status CHECK (status IN ('active', 'disabled'))
);
CREATE UNIQUE INDEX uq_notify_template_global
    ON notify.templates (code, channel, COALESCE(locale, ''))
    WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX uq_notify_template_tenant
    ON notify.templates (tenant_id, code, channel, COALESCE(locale, ''))
    WHERE tenant_id IS NOT NULL;

CREATE TABLE notify.messages (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    sender_user_id      uuid,
    message_type        varchar(24) NOT NULL DEFAULT 'system',
    title               varchar(300) NOT NULL,
    body                text NOT NULL,
    body_format         varchar(16) NOT NULL DEFAULT 'markdown',
    status              varchar(20) NOT NULL DEFAULT 'draft',
    scheduled_at        timestamptz,
    published_at        timestamptz,
    expires_at          timestamptz,
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT uq_notify_messages_tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT fk_notify_message_sender_member
        FOREIGN KEY (tenant_id, sender_user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT ck_notify_message_type CHECK (message_type IN ('system', 'notice', 'private', 'marketing', 'security')),
    CONSTRAINT ck_notify_body_format CHECK (body_format IN ('plain', 'markdown', 'html')),
    CONSTRAINT ck_notify_message_status CHECK (status IN ('draft', 'scheduled', 'published', 'cancelled')),
    CONSTRAINT ck_notify_message_expiry CHECK (expires_at IS NULL OR expires_at > created_at)
);
CREATE INDEX idx_notify_messages_tenant_time ON notify.messages (tenant_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_notify_messages_schedule ON notify.messages (scheduled_at) WHERE status = 'scheduled';

CREATE TABLE notify.recipients (
    tenant_id           uuid NOT NULL,
    message_id          uuid NOT NULL,
    user_id             uuid NOT NULL,
    delivery_status     varchar(20) NOT NULL DEFAULT 'pending',
    delivered_at        timestamptz,
    read_at             timestamptz,
    archived_at         timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, message_id, user_id),
    CONSTRAINT fk_notify_recipients_message
        FOREIGN KEY (tenant_id, message_id) REFERENCES notify.messages(tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_notify_recipients_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT ck_notify_recipient_status CHECK (delivery_status IN ('pending', 'delivered', 'failed'))
);
CREATE INDEX idx_notify_recipients_unread ON notify.recipients (tenant_id, user_id, created_at DESC) WHERE read_at IS NULL;
CREATE INDEX idx_notify_recipients_delivered ON notify.recipients (tenant_id, user_id, delivered_at DESC);

CREATE TABLE notify.deliveries (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    message_id          uuid,
    user_id             uuid,
    template_id         uuid REFERENCES notify.templates(id) ON DELETE SET NULL,
    channel             varchar(20) NOT NULL,
    target_ciphertext   bytea NOT NULL,
    target_hash         bytea NOT NULL,
    target_hint         varchar(160),
    target_key_version  integer NOT NULL,
    provider            varchar(64),
    provider_message_id varchar(255),
    rendered_subject    text,
    rendered_body       text,
    status              varchar(20) NOT NULL DEFAULT 'pending',
    attempt_count       integer NOT NULL DEFAULT 0,
    max_attempts        integer NOT NULL DEFAULT 5,
    scheduled_at        timestamptz NOT NULL DEFAULT now(),
    next_attempt_at     timestamptz,
    sent_at             timestamptz,
    last_error          varchar(2000),
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_notify_delivery_target_hash CHECK (octet_length(target_hash) = 32),
    CONSTRAINT fk_notify_deliveries_message
        FOREIGN KEY (tenant_id, message_id) REFERENCES notify.messages(tenant_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_notify_deliveries_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT ck_notify_delivery_channel CHECK (channel IN ('email', 'sms', 'push', 'webhook')),
    CONSTRAINT ck_notify_delivery_status CHECK (status IN ('pending', 'processing', 'sent', 'failed', 'cancelled')),
    CONSTRAINT ck_notify_delivery_attempts CHECK (attempt_count >= 0 AND max_attempts > 0)
);
CREATE INDEX idx_notify_deliveries_due ON notify.deliveries (status, next_attempt_at, scheduled_at)
    WHERE status IN ('pending', 'failed');
CREATE INDEX idx_notify_deliveries_user ON notify.deliveries (user_id, created_at DESC);
CREATE INDEX idx_notify_deliveries_target ON notify.deliveries (channel, target_hash, created_at DESC);

CREATE TABLE notify.push_devices (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id             uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    device_id           uuid,
    provider            varchar(32) NOT NULL,
    token_hash          bytea NOT NULL,
    token_ciphertext    bytea NOT NULL,
    key_version         integer NOT NULL,
    status              varchar(20) NOT NULL DEFAULT 'active',
    last_success_at     timestamptz,
    last_failure_at     timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_push_token_hash UNIQUE (provider, token_hash),
    CONSTRAINT fk_push_device_device
        FOREIGN KEY (user_id, device_id) REFERENCES iam.devices(user_id, id)
        ON DELETE CASCADE,
    CONSTRAINT ck_push_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT ck_push_device_provider CHECK (provider IN ('apns', 'fcm', 'hms', 'custom')),
    CONSTRAINT ck_push_device_status CHECK (status IN ('active', 'invalid', 'disabled'))
);

CREATE TABLE jobs.schedules (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE CASCADE,
    code                public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    handler_key         varchar(160) NOT NULL,
    cron_expression     varchar(128) NOT NULL,
    time_zone           varchar(64) NOT NULL DEFAULT 'UTC',
    payload             jsonb NOT NULL DEFAULT '{}'::jsonb,
    queue_name          varchar(64) NOT NULL DEFAULT 'default',
    overlap_policy      varchar(24) NOT NULL DEFAULT 'skip',
    misfire_policy      varchar(24) NOT NULL DEFAULT 'fire_once',
    timeout_seconds     integer NOT NULL DEFAULT 300,
    max_attempts        integer NOT NULL DEFAULT 3,
    status              varchar(20) NOT NULL DEFAULT 'active',
    last_enqueued_at    timestamptz,
    next_run_at         timestamptz,
    created_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_jobs_overlap_policy CHECK (overlap_policy IN ('allow', 'skip', 'replace')),
    CONSTRAINT ck_jobs_misfire_policy CHECK (misfire_policy IN ('ignore', 'fire_once', 'catch_up')),
    CONSTRAINT ck_jobs_timeout CHECK (timeout_seconds BETWEEN 1 AND 86400),
    CONSTRAINT ck_jobs_attempts CHECK (max_attempts BETWEEN 1 AND 100),
    CONSTRAINT ck_jobs_status CHECK (status IN ('active', 'paused', 'disabled'))
);
CREATE UNIQUE INDEX uq_job_schedule_global_code
    ON jobs.schedules (code)
    WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX uq_job_schedule_tenant_code
    ON jobs.schedules (tenant_id, code)
    WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_job_schedules_due ON jobs.schedules (next_run_at) WHERE status = 'active';

CREATE TABLE jobs.schedule_runs (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    schedule_id         uuid NOT NULL REFERENCES jobs.schedules(id) ON DELETE CASCADE,
    river_job_id        bigint,
    trigger_type        varchar(20) NOT NULL DEFAULT 'schedule',
    status              varchar(20) NOT NULL DEFAULT 'queued',
    attempt             integer NOT NULL DEFAULT 0,
    scheduled_at        timestamptz NOT NULL,
    started_at          timestamptz,
    finished_at         timestamptz,
    worker_id           varchar(255),
    output              jsonb,
    error_code          varchar(160),
    error_message       varchar(4000),
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_schedule_runs_trigger CHECK (trigger_type IN ('schedule', 'manual', 'retry')),
    CONSTRAINT ck_schedule_runs_status CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled', 'skipped')),
    CONSTRAINT ck_schedule_runs_attempt CHECK (attempt >= 0)
);
CREATE INDEX idx_schedule_runs_schedule_time ON jobs.schedule_runs (schedule_id, scheduled_at DESC);
CREATE INDEX idx_schedule_runs_status_time ON jobs.schedule_runs (status, scheduled_at DESC);
CREATE UNIQUE INDEX uq_schedule_runs_river_job ON jobs.schedule_runs (river_job_id) WHERE river_job_id IS NOT NULL;

CREATE TABLE jobs.outbox_events (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE CASCADE,
    aggregate_type      varchar(160) NOT NULL,
    aggregate_id        uuid,
    event_type          varchar(200) NOT NULL,
    event_version       integer NOT NULL DEFAULT 1,
    payload             jsonb NOT NULL,
    headers             jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at         timestamptz NOT NULL DEFAULT now(),
    available_at        timestamptz NOT NULL DEFAULT now(),
    published_at        timestamptz,
    attempt_count       integer NOT NULL DEFAULT 0,
    last_error          varchar(2000),
    CONSTRAINT ck_outbox_version CHECK (event_version > 0),
    CONSTRAINT ck_outbox_attempts CHECK (attempt_count >= 0)
);
CREATE INDEX idx_outbox_pending ON jobs.outbox_events (available_at, occurred_at) WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate ON jobs.outbox_events (aggregate_type, aggregate_id, occurred_at);

CREATE TABLE jobs.inbox_events (
    consumer_name       varchar(160) NOT NULL,
    event_id            uuid NOT NULL,
    event_type          varchar(200) NOT NULL,
    processed_at        timestamptz NOT NULL DEFAULT now(),
    result              jsonb,
    PRIMARY KEY (consumer_name, event_id)
);

CREATE TABLE audit.operation_logs (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE SET NULL,
    user_id             uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    session_id          uuid REFERENCES iam.sessions(id) ON DELETE SET NULL,
    request_id          varchar(128) NOT NULL,
    trace_id            varchar(128),
    module_code         varchar(64) NOT NULL,
    action_name         varchar(160) NOT NULL,
    permission_code     varchar(200),
    resource_type       varchar(160),
    resource_id         varchar(255),
    http_method         varchar(16),
    request_path        varchar(1500),
    response_status     integer,
    client_ip           inet,
    user_agent          varchar(1000),
    request_summary     jsonb,
    before_data         jsonb,
    after_data          jsonb,
    duration_ms         integer,
    succeeded           boolean NOT NULL,
    error_code          varchar(160),
    error_message       varchar(2000),
    occurred_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_operation_duration CHECK (duration_ms IS NULL OR duration_ms >= 0)
);
CREATE INDEX idx_operation_logs_tenant_time ON audit.operation_logs (tenant_id, occurred_at DESC);
CREATE INDEX idx_operation_logs_user_time ON audit.operation_logs (user_id, occurred_at DESC);
CREATE INDEX idx_operation_logs_request ON audit.operation_logs (request_id);
CREATE INDEX idx_operation_logs_resource ON audit.operation_logs (resource_type, resource_id, occurred_at DESC);

CREATE TABLE audit.login_events (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE SET NULL,
    user_id             uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    session_id          uuid REFERENCES iam.sessions(id) ON DELETE SET NULL,
    request_id          varchar(128),
    login_identifier_hash bytea,
    login_identifier_hint varchar(160),
    auth_method         varchar(32) NOT NULL,
    audience            varchar(32) NOT NULL,
    result              varchar(20) NOT NULL,
    failure_reason      varchar(500),
    client_ip           inet,
    user_agent          varchar(1000),
    device_info         jsonb NOT NULL DEFAULT '{}'::jsonb,
    geo_info            jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_login_identifier_hash CHECK (login_identifier_hash IS NULL OR octet_length(login_identifier_hash) = 32),
    CONSTRAINT ck_login_auth_method CHECK (auth_method IN ('password', 'email_otp', 'sms_otp', 'oauth', 'refresh_token', 'api_secret', 'mfa', 'tenant_switch')),
    CONSTRAINT ck_login_audience CHECK (audience IN ('ak-mobile', 'ak-admin', 'ak-api')),
    CONSTRAINT ck_login_result CHECK (result IN ('success', 'failure', 'blocked'))
);
CREATE INDEX idx_login_events_user_time ON audit.login_events (user_id, occurred_at DESC);
CREATE INDEX idx_login_events_identifier_time ON audit.login_events (login_identifier_hash, occurred_at DESC)
    WHERE login_identifier_hash IS NOT NULL;
CREATE INDEX idx_login_events_ip_time ON audit.login_events (client_ip, occurred_at DESC);
CREATE INDEX idx_login_events_result_time ON audit.login_events (result, occurred_at DESC);

CREATE TABLE audit.security_events (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE SET NULL,
    user_id             uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    session_id          uuid REFERENCES iam.sessions(id) ON DELETE SET NULL,
    event_type          varchar(160) NOT NULL,
    severity            varchar(16) NOT NULL,
    source              varchar(64) NOT NULL,
    client_ip           inet,
    details             jsonb NOT NULL DEFAULT '{}'::jsonb,
    resolved_at         timestamptz,
    resolved_by         uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    occurred_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_security_severity CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical'))
);
CREATE INDEX idx_security_events_open ON audit.security_events (severity, occurred_at DESC) WHERE resolved_at IS NULL;
CREATE INDEX idx_security_events_user_time ON audit.security_events (user_id, occurred_at DESC);

CREATE TRIGGER tr_notify_templates_touch_updated_at
BEFORE UPDATE ON notify.templates FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_notify_messages_touch_updated_at
BEFORE UPDATE ON notify.messages FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_notify_recipients_touch_updated_at
BEFORE UPDATE ON notify.recipients FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_notify_deliveries_touch_updated_at
BEFORE UPDATE ON notify.deliveries FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_push_devices_touch_updated_at
BEFORE UPDATE ON notify.push_devices FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_job_schedules_touch_updated_at
BEFORE UPDATE ON jobs.schedules FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMENT ON TABLE jobs.schedules IS 'Admin-managed schedules. handler_key must resolve to a worker registered in compiled Go code; no arbitrary shell command or code execution.';
COMMENT ON TABLE jobs.outbox_events IS 'Transactional outbox for external event publication. Internal background work is enqueued transactionally through River.';
COMMENT ON TABLE audit.operation_logs IS 'Immutable, redacted business-operation audit log; not a replacement for application logs in an observability backend.';
COMMENT ON TABLE audit.login_events IS 'Authentication success/failure/block events for security review.';

COMMIT;

-- ============================================================================
-- SOURCE: db/migrations/000006_billing_optional.up.sql
-- ============================================================================
-- OPTIONAL MODULE. Do not include in the first AK core release unless payments/wallets are required.
BEGIN;

CREATE SCHEMA IF NOT EXISTS billing;

CREATE TABLE billing.payment_orders (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    user_id             uuid,
    order_number        varchar(64) NOT NULL,
    business_type       varchar(96) NOT NULL,
    business_id         uuid,
    subject             varchar(300) NOT NULL,
    description         varchar(1000),
    currency            char(3) NOT NULL,
    amount_minor        bigint NOT NULL,
    status              varchar(24) NOT NULL DEFAULT 'pending',
    expires_at          timestamptz,
    paid_at             timestamptz,
    cancelled_at        timestamptz,
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_payment_order_number UNIQUE (tenant_id, order_number),
    CONSTRAINT fk_payment_order_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT ck_payment_order_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT ck_payment_order_amount CHECK (amount_minor > 0),
    CONSTRAINT ck_payment_order_status CHECK (status IN ('pending', 'processing', 'paid', 'failed', 'cancelled', 'expired', 'partially_refunded', 'refunded'))
);
CREATE INDEX idx_payment_orders_user_time ON billing.payment_orders (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_payment_orders_status_time ON billing.payment_orders (tenant_id, status, created_at DESC);

CREATE TABLE billing.payment_transactions (
    id                      uuid PRIMARY KEY DEFAULT uuidv7(),
    payment_order_id        uuid NOT NULL REFERENCES billing.payment_orders(id) ON DELETE RESTRICT,
    transaction_number      varchar(96) NOT NULL,
    gateway                 varchar(32) NOT NULL,
    channel                 varchar(32) NOT NULL,
    provider_account        varchar(128),
    provider_transaction_id varchar(255),
    currency                char(3) NOT NULL,
    amount_minor            bigint NOT NULL,
    status                  varchar(24) NOT NULL DEFAULT 'created',
    request_payload         jsonb,
    response_payload        jsonb,
    callback_payload        jsonb,
    client_ip               inet,
    paid_at                 timestamptz,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_payment_transaction_number UNIQUE (transaction_number),
    CONSTRAINT ck_payment_transaction_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT uq_provider_transaction UNIQUE (gateway, provider_transaction_id),
    CONSTRAINT ck_payment_transaction_amount CHECK (amount_minor > 0),
    CONSTRAINT ck_payment_transaction_status CHECK (status IN ('created', 'pending', 'succeeded', 'failed', 'closed'))
);
CREATE INDEX idx_payment_transactions_order ON billing.payment_transactions (payment_order_id, created_at DESC);

CREATE TABLE billing.refunds (
    id                      uuid PRIMARY KEY DEFAULT uuidv7(),
    payment_order_id        uuid NOT NULL REFERENCES billing.payment_orders(id) ON DELETE RESTRICT,
    payment_transaction_id  uuid REFERENCES billing.payment_transactions(id) ON DELETE RESTRICT,
    refund_number           varchar(96) NOT NULL,
    provider_refund_id      varchar(255),
    currency                char(3) NOT NULL,
    amount_minor            bigint NOT NULL,
    reason                  varchar(1000),
    status                  varchar(24) NOT NULL DEFAULT 'pending',
    requested_by            uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    processed_at            timestamptz,
    response_payload        jsonb,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_refund_number UNIQUE (refund_number),
    CONSTRAINT ck_refund_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT uq_provider_refund UNIQUE (provider_refund_id),
    CONSTRAINT ck_refund_amount CHECK (amount_minor > 0),
    CONSTRAINT ck_refund_status CHECK (status IN ('pending', 'processing', 'succeeded', 'failed', 'cancelled'))
);
CREATE INDEX idx_refunds_order ON billing.refunds (payment_order_id, created_at DESC);

CREATE TABLE billing.wallet_accounts (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    user_id             uuid NOT NULL,
    asset_code          varchar(32) NOT NULL,
    account_type        varchar(24) NOT NULL DEFAULT 'available',
    balance_minor       bigint NOT NULL DEFAULT 0,
    frozen_minor        bigint NOT NULL DEFAULT 0,
    lock_version        integer NOT NULL DEFAULT 1,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_wallet_account UNIQUE (tenant_id, user_id, asset_code, account_type),
    CONSTRAINT fk_wallet_account_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT ck_wallet_account_type CHECK (account_type IN ('available', 'bonus', 'points')),
    CONSTRAINT ck_wallet_balances CHECK (balance_minor >= 0 AND frozen_minor >= 0),
    CONSTRAINT ck_wallet_version CHECK (lock_version > 0),
    CONSTRAINT ck_wallet_status CHECK (status IN ('active', 'frozen', 'closed'))
);

CREATE TABLE billing.wallet_ledger_entries (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    wallet_account_id   uuid NOT NULL REFERENCES billing.wallet_accounts(id) ON DELETE RESTRICT,
    entry_number        varchar(96) NOT NULL,
    entry_type          varchar(32) NOT NULL,
    direction           varchar(8) NOT NULL,
    amount_minor        bigint NOT NULL,
    balance_before      bigint NOT NULL,
    balance_after       bigint NOT NULL,
    business_type       varchar(96) NOT NULL,
    business_id         uuid,
    idempotency_key     varchar(255) NOT NULL,
    remark              varchar(1000),
    operator_user_id    uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    occurred_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_wallet_entry_number UNIQUE (entry_number),
    CONSTRAINT uq_wallet_entry_idempotency UNIQUE (wallet_account_id, idempotency_key),
    CONSTRAINT ck_wallet_entry_direction CHECK (direction IN ('credit', 'debit')),
    CONSTRAINT ck_wallet_entry_amount CHECK (amount_minor > 0),
    CONSTRAINT ck_wallet_entry_balance CHECK (
        balance_before >= 0 AND balance_after >= 0 AND (
            (direction = 'credit' AND balance_after = balance_before + amount_minor)
            OR (direction = 'debit' AND balance_after = balance_before - amount_minor)
        )
    )
);
CREATE INDEX idx_wallet_ledger_account_time ON billing.wallet_ledger_entries (wallet_account_id, occurred_at DESC);
CREATE INDEX idx_wallet_ledger_business ON billing.wallet_ledger_entries (business_type, business_id);

CREATE TABLE billing.withdrawals (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    wallet_account_id   uuid NOT NULL REFERENCES billing.wallet_accounts(id) ON DELETE RESTRICT,
    withdrawal_number   varchar(96) NOT NULL,
    amount_minor        bigint NOT NULL,
    fee_minor           bigint NOT NULL DEFAULT 0,
    payout_minor        bigint GENERATED ALWAYS AS (amount_minor - fee_minor) STORED,
    payout_method       varchar(32) NOT NULL,
    payout_target_enc   bytea NOT NULL,
    payout_key_version  integer NOT NULL,
    status              varchar(24) NOT NULL DEFAULT 'pending',
    reason              varchar(1000),
    handled_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    handled_at          timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_withdrawal_number UNIQUE (withdrawal_number),
    CONSTRAINT ck_withdrawal_amount CHECK (amount_minor > 0 AND fee_minor >= 0 AND amount_minor >= fee_minor),
    CONSTRAINT ck_withdrawal_status CHECK (status IN ('pending', 'approved', 'processing', 'succeeded', 'rejected', 'cancelled'))
);
CREATE INDEX idx_withdrawals_account_time ON billing.withdrawals (wallet_account_id, created_at DESC);
CREATE INDEX idx_withdrawals_status_time ON billing.withdrawals (status, created_at DESC);

CREATE TRIGGER tr_payment_orders_touch_updated_at
BEFORE UPDATE ON billing.payment_orders FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_payment_transactions_touch_updated_at
BEFORE UPDATE ON billing.payment_transactions FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_refunds_touch_updated_at
BEFORE UPDATE ON billing.refunds FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_wallet_accounts_touch_updated_at
BEFORE UPDATE ON billing.wallet_accounts FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_withdrawals_touch_updated_at
BEFORE UPDATE ON billing.withdrawals FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMENT ON TABLE billing.wallet_ledger_entries IS 'Immutable wallet/points ledger. Corrections are compensating entries, never UPDATE or DELETE.';

COMMIT;
