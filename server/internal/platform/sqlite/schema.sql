CREATE TABLE IF NOT EXISTS ak_schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS iam_tenants (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL COLLATE NOCASE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled')),
    settings TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_iam_tenants_code_active ON iam_tenants(code) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS iam_users (
    id TEXT PRIMARY KEY,
    email TEXT COLLATE NOCASE,
    mobile TEXT,
    display_name TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT 'zh-CN' CHECK (locale IN ('zh-CN','en-US')),
    time_zone TEXT NOT NULL DEFAULT 'UTC',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled','locked')),
    avatar_file_id TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_iam_users_email_active ON iam_users(email) WHERE email IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS iam_user_credentials (
    user_id TEXT PRIMARY KEY REFERENCES iam_users(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    password_version INTEGER NOT NULL DEFAULT 1 CHECK (password_version > 0),
    password_updated_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS iam_password_history (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES iam_users(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_iam_password_history_user ON iam_password_history(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS iam_tenant_members (
    tenant_id TEXT NOT NULL REFERENCES iam_tenants(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES iam_users(id) ON DELETE CASCADE,
    display_name TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled')),
    joined_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, user_id)
);

CREATE TABLE IF NOT EXISTS iam_roles (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES iam_tenants(id) ON DELETE CASCADE,
    code TEXT NOT NULL COLLATE NOCASE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    role_type TEXT NOT NULL DEFAULT 'custom',
    data_scope TEXT NOT NULL DEFAULT 'self',
    is_default INTEGER NOT NULL DEFAULT 0 CHECK (is_default IN (0,1)),
    is_system INTEGER NOT NULL DEFAULT 0 CHECK (is_system IN (0,1)),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_iam_roles_tenant_code_active ON iam_roles(tenant_id, code) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS iam_permissions (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE COLLATE NOCASE,
    name TEXT NOT NULL,
    module_code TEXT NOT NULL,
    resource_name TEXT NOT NULL,
    action_name TEXT NOT NULL,
    permission_kind TEXT NOT NULL DEFAULT 'api',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS iam_user_roles (
    tenant_id TEXT NOT NULL REFERENCES iam_tenants(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES iam_users(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES iam_roles(id) ON DELETE CASCADE,
    valid_from TEXT NOT NULL,
    valid_until TEXT,
    granted_by TEXT REFERENCES iam_users(id) ON DELETE SET NULL,
    PRIMARY KEY (tenant_id, user_id, role_id)
);

CREATE TABLE IF NOT EXISTS iam_role_permissions (
    tenant_id TEXT NOT NULL REFERENCES iam_tenants(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES iam_roles(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES iam_permissions(id) ON DELETE CASCADE,
    granted_by TEXT REFERENCES iam_users(id) ON DELETE SET NULL,
    PRIMARY KEY (tenant_id, role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS iam_devices (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES iam_users(id) ON DELETE CASCADE,
    device_key TEXT NOT NULL,
    platform TEXT NOT NULL DEFAULT 'web',
    device_name TEXT,
    model TEXT,
    os_version TEXT,
    app_version TEXT,
    last_ip TEXT,
    last_seen_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (user_id, device_key)
);

CREATE TABLE IF NOT EXISTS iam_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES iam_users(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL REFERENCES iam_tenants(id) ON DELETE CASCADE,
    app_id TEXT,
    device_id TEXT REFERENCES iam_devices(id) ON DELETE SET NULL,
    audience TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','revoked','expired')),
    access_token_version INTEGER NOT NULL DEFAULT 1 CHECK (access_token_version > 0),
    absolute_expires_at TEXT NOT NULL,
    idle_expires_at TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    last_seen_at TEXT NOT NULL,
    revoked_at TEXT,
    revoke_reason TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_iam_sessions_scope ON iam_sessions(user_id, tenant_id, audience, created_at DESC);

CREATE TABLE IF NOT EXISTS iam_refresh_tokens (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES iam_sessions(id) ON DELETE CASCADE,
    token_hash BLOB NOT NULL UNIQUE,
    parent_token_id TEXT REFERENCES iam_refresh_tokens(id) ON DELETE SET NULL,
    replaced_by_token_id TEXT REFERENCES iam_refresh_tokens(id) ON DELETE SET NULL,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    revoked_at TEXT,
    reuse_detected_at TEXT,
    created_ip TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_iam_refresh_tokens_session ON iam_refresh_tokens(session_id);

CREATE TABLE IF NOT EXISTS iam_login_failure_states (
    scope_hash BLOB PRIMARY KEY,
    failure_count INTEGER NOT NULL CHECK (failure_count >= 0),
    first_failed_at TEXT NOT NULL,
    last_failed_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS iam_interactive_captcha_challenges (
    id TEXT PRIMARY KEY,
    scope_hash BLOB NOT NULL,
    captcha_type TEXT NOT NULL CHECK (captcha_type IN ('click','slide','drag','rotate')),
    proof_hash BLOB NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    consumed_at TEXT,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_iam_captcha_scope ON iam_interactive_captcha_challenges(scope_hash, created_at DESC);

CREATE TABLE IF NOT EXISTS iam_verification_challenges (
    id TEXT PRIMARY KEY,
    challenge_type TEXT NOT NULL,
    user_id TEXT REFERENCES iam_users(id) ON DELETE CASCADE,
    target_hash BLOB NOT NULL,
    secret_hash BLOB NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    created_ip TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_iam_verification_user ON iam_verification_challenges(user_id, challenge_type, created_at DESC);

CREATE TABLE IF NOT EXISTS app_applications (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES iam_tenants(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL,
    UNIQUE (tenant_id, code)
);

CREATE TABLE IF NOT EXISTS app_user_memberships (
    app_id TEXT NOT NULL REFERENCES app_applications(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL REFERENCES iam_tenants(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES iam_users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL,
    PRIMARY KEY (app_id, tenant_id, user_id)
);

CREATE TABLE IF NOT EXISTS app_user_login_identifiers (
    app_id TEXT NOT NULL REFERENCES app_applications(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL REFERENCES iam_tenants(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES iam_users(id) ON DELETE CASCADE,
    identifier_type TEXT NOT NULL,
    normalized_value TEXT NOT NULL COLLATE NOCASE,
    status TEXT NOT NULL DEFAULT 'active',
    verified_at TEXT,
    created_at TEXT NOT NULL,
    PRIMARY KEY (app_id, identifier_type, normalized_value)
);

CREATE TABLE IF NOT EXISTS sys_menus (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES iam_tenants(id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES sys_menus(id) ON DELETE RESTRICT,
    permission_id TEXT REFERENCES iam_permissions(id) ON DELETE SET NULL,
    code TEXT NOT NULL COLLATE NOCASE,
    i18n_key TEXT NOT NULL,
    title TEXT NOT NULL,
    menu_type TEXT NOT NULL,
    route_path TEXT,
    component_key TEXT,
    icon TEXT,
    affix INTEGER NOT NULL DEFAULT 0 CHECK (affix IN (0,1)),
    sort_order INTEGER NOT NULL DEFAULT 0,
    feature_flag TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_sys_menus_global_code ON sys_menus(code) WHERE tenant_id IS NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_sys_menus_tenant_code ON sys_menus(tenant_id, code) WHERE tenant_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS sys_role_menus (
    tenant_id TEXT NOT NULL REFERENCES iam_tenants(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES iam_roles(id) ON DELETE CASCADE,
    menu_id TEXT NOT NULL REFERENCES sys_menus(id) ON DELETE CASCADE,
    PRIMARY KEY (tenant_id, role_id, menu_id)
);

CREATE TABLE IF NOT EXISTS notify_push_devices (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    app_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    invalidated_at TEXT,
    invalid_reason TEXT,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_login_events (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    user_id TEXT,
    session_id TEXT,
    app_id TEXT,
    request_id TEXT,
    auth_method TEXT NOT NULL,
    audience TEXT NOT NULL,
    succeeded INTEGER NOT NULL CHECK (succeeded IN (0,1)),
    client_ip TEXT,
    user_agent TEXT,
    device_registered INTEGER NOT NULL DEFAULT 0 CHECK (device_registered IN (0,1)),
    occurred_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_login_tenant_time ON audit_login_events(tenant_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS audit_operation_logs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    user_id TEXT,
    session_id TEXT,
    request_id TEXT,
    module_code TEXT NOT NULL,
    action_name TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    request_path TEXT,
    client_ip TEXT,
    user_agent TEXT,
    succeeded INTEGER NOT NULL DEFAULT 1 CHECK (succeeded IN (0,1)),
    error_code TEXT,
    before_data TEXT,
    after_data TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',
    occurred_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_operations_tenant_time ON audit_operation_logs(tenant_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS audit_security_events (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    user_id TEXT,
    session_id TEXT,
    app_id TEXT,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    source TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    client_ip TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',
    occurred_at TEXT NOT NULL,
    resolved_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_audit_security_tenant_time ON audit_security_events(tenant_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS jobs_schedules (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs_schedule_runs (
    id TEXT PRIMARY KEY,
    schedule_id TEXT NOT NULL REFERENCES jobs_schedules(id) ON DELETE CASCADE,
    tenant_id TEXT,
    status TEXT NOT NULL,
    error_code TEXT,
    scheduled_at TEXT NOT NULL,
    finished_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_jobs_runs_tenant_time ON jobs_schedule_runs(tenant_id, scheduled_at DESC);

CREATE TABLE IF NOT EXISTS notify_messages (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    status TEXT NOT NULL,
    published_at TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notify_messages_tenant_time ON notify_messages(tenant_id, published_at DESC);
