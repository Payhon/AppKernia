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
    CONSTRAINT ck_login_auth_method CHECK (auth_method IN ('password', 'email_otp', 'sms_otp', 'oauth', 'refresh_token', 'api_secret', 'mfa')),
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
