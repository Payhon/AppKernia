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
