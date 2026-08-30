BEGIN;

-- API clients are tenant principals, but machine notification submission is
-- explicitly scoped to applications.  An empty mapping denies App access.
CREATE TABLE sys.api_client_apps (
    tenant_id       uuid NOT NULL,
    api_client_id   uuid NOT NULL,
    app_id          uuid NOT NULL,
    created_by      uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, api_client_id, app_id),
    CONSTRAINT fk_api_client_apps_client
        FOREIGN KEY (tenant_id, api_client_id)
        REFERENCES sys.api_clients(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_api_client_apps_app
        FOREIGN KEY (tenant_id, app_id)
        REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_api_client_apps_app ON sys.api_client_apps (tenant_id, app_id, api_client_id);

-- Domain-safe projection of River jobs. River remains the execution source of
-- truth; this table provides tenant/App isolation, correlation and retained
-- history without exposing raw args or stack traces.
CREATE TABLE jobs.task_runs (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    app_id              uuid,
    module_code         varchar(64) NOT NULL,
    task_kind           varchar(160) NOT NULL,
    queue_name          varchar(80) NOT NULL,
    resource_type       varchar(80) NOT NULL,
    resource_id         uuid,
    correlation_id      uuid,
    river_job_id        bigint,
    status              varchar(24) NOT NULL DEFAULT 'queued',
    scheduled_at        timestamptz NOT NULL DEFAULT now(),
    started_at          timestamptz,
    finalized_at        timestamptz,
    next_retry_at       timestamptz,
    attempt_count       integer NOT NULL DEFAULT 0,
    max_attempts        integer NOT NULL DEFAULT 1,
    last_result_class   varchar(40),
    last_error_code     varchar(160),
    last_error_summary  varchar(500),
    trace_id            varchar(64),
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_task_runs_app
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_task_runs_module CHECK (module_code ~ '^[a-z][a-z0-9_.-]{1,63}$'),
    CONSTRAINT ck_task_runs_status CHECK (status IN (
        'scheduled', 'queued', 'running', 'retry_wait', 'succeeded', 'failed', 'cancelled'
    )),
    CONSTRAINT ck_task_runs_attempts CHECK (
        attempt_count >= 0 AND max_attempts BETWEEN 1 AND 100 AND attempt_count <= max_attempts
    ),
    CONSTRAINT ck_task_runs_error_summary CHECK (
        last_error_summary IS NULL OR length(last_error_summary) <= 500
    )
);
CREATE UNIQUE INDEX uq_task_runs_river_job ON jobs.task_runs (river_job_id) WHERE river_job_id IS NOT NULL;
CREATE INDEX idx_task_runs_scope_state ON jobs.task_runs (tenant_id, app_id, module_code, status, scheduled_at DESC, id DESC);
CREATE INDEX idx_task_runs_correlation ON jobs.task_runs (tenant_id, app_id, correlation_id, created_at, id);
CREATE INDEX idx_task_runs_retention ON jobs.task_runs (finalized_at, id)
    WHERE status IN ('succeeded', 'failed', 'cancelled');
CREATE TRIGGER tr_task_runs_touch_updated_at BEFORE UPDATE ON jobs.task_runs
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

CREATE TABLE jobs.task_attempts (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    app_id              uuid,
    task_run_id         uuid NOT NULL REFERENCES jobs.task_runs(id) ON DELETE CASCADE,
    attempt_number      integer NOT NULL,
    status              varchar(24) NOT NULL,
    result_class        varchar(40),
    error_code          varchar(160),
    error_summary       varchar(500),
    external_request_id varchar(255),
    trace_id            varchar(64),
    started_at          timestamptz NOT NULL DEFAULT now(),
    finished_at         timestamptz,
    duration_ms         bigint,
    next_retry_at       timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_task_attempts_app
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_task_attempt_number UNIQUE (task_run_id, attempt_number),
    CONSTRAINT ck_task_attempt_number CHECK (attempt_number BETWEEN 1 AND 100),
    CONSTRAINT ck_task_attempt_status CHECK (status IN ('running', 'retry_wait', 'succeeded', 'failed', 'cancelled')),
    CONSTRAINT ck_task_attempt_duration CHECK (duration_ms IS NULL OR duration_ms >= 0),
    CONSTRAINT ck_task_attempt_error_summary CHECK (error_summary IS NULL OR length(error_summary) <= 500)
);
CREATE INDEX idx_task_attempts_scope ON jobs.task_attempts (tenant_id, app_id, task_run_id, attempt_number DESC);

CREATE TABLE notify.message_runs (
    id                      uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id               uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    app_id                  uuid NOT NULL,
    message_id              uuid NOT NULL REFERENCES notify.messages(id) ON DELETE CASCADE,
    trigger_type            varchar(24) NOT NULL,
    status                  varchar(32) NOT NULL,
    recipient_count         bigint NOT NULL DEFAULT 0,
    evaluated_count         bigint NOT NULL DEFAULT 0,
    delivery_count          bigint NOT NULL DEFAULT 0,
    accepted_count          bigint NOT NULL DEFAULT 0,
    failed_count            bigint NOT NULL DEFAULT 0,
    invalid_token_count     bigint NOT NULL DEFAULT 0,
    opened_count            bigint NOT NULL DEFAULT 0,
    skipped_count           bigint NOT NULL DEFAULT 0,
    started_at              timestamptz,
    completed_at            timestamptz,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_message_runs_app
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_message_runs_message UNIQUE (tenant_id, app_id, message_id),
    CONSTRAINT uq_message_runs_scope_id UNIQUE (tenant_id, app_id, id),
    CONSTRAINT ck_message_runs_trigger CHECK (trigger_type IN ('admin', 'scheduled', 'api_client', 'internal')),
    CONSTRAINT ck_message_runs_status CHECK (status IN (
        'scheduled', 'queued', 'running', 'completed', 'completed_with_failures',
        'failed', 'cancelled', 'expired'
    )),
    CONSTRAINT ck_message_runs_counts CHECK (
        recipient_count >= 0 AND evaluated_count >= 0 AND delivery_count >= 0
        AND accepted_count >= 0 AND failed_count >= 0 AND invalid_token_count >= 0
        AND opened_count >= 0 AND skipped_count >= 0
    )
);
CREATE INDEX idx_message_runs_scope_status ON notify.message_runs (tenant_id, app_id, status, created_at DESC, id DESC);
CREATE TRIGGER tr_message_runs_touch_updated_at BEFORE UPDATE ON notify.message_runs
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

ALTER TABLE notify.deliveries
    ADD COLUMN message_run_id uuid,
    ADD COLUMN task_run_id uuid REFERENCES jobs.task_runs(id) ON DELETE SET NULL,
    ADD COLUMN delivery_environment varchar(20) NOT NULL DEFAULT 'development',
    ADD CONSTRAINT fk_notify_delivery_message_run
        FOREIGN KEY (tenant_id, app_id, message_run_id)
        REFERENCES notify.message_runs(tenant_id, app_id, id) ON DELETE SET NULL,
    ADD CONSTRAINT ck_notify_delivery_environment CHECK (
        delivery_environment IN ('development', 'test', 'staging', 'production')
    );
CREATE INDEX idx_notify_deliveries_operations
    ON notify.deliveries (tenant_id, app_id, message_run_id, status, created_at DESC, id DESC);

CREATE TABLE notify.delivery_daily_metrics (
    metric_date         date NOT NULL,
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    app_id              uuid NOT NULL,
    environment         varchar(20) NOT NULL,
    channel             varchar(24) NOT NULL,
    provider            varchar(32) NOT NULL DEFAULT '',
    message_category    varchar(32) NOT NULL DEFAULT '',
    outcome             varchar(40) NOT NULL,
    skip_reason         varchar(40) NOT NULL DEFAULT '',
    event_count         bigint NOT NULL DEFAULT 0,
    updated_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (
        metric_date, tenant_id, app_id, environment, channel,
        provider, message_category, outcome, skip_reason
    ),
    CONSTRAINT fk_delivery_metrics_app
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_delivery_metrics_environment CHECK (environment IN ('development', 'test', 'staging', 'production')),
    CONSTRAINT ck_delivery_metrics_channel CHECK (channel IN ('in_app', 'email', 'sms', 'push', 'webhook')),
    CONSTRAINT ck_delivery_metrics_outcome CHECK (outcome IN (
        'queued', 'accepted', 'failed', 'invalid_token', 'opened', 'skipped'
    )),
    CONSTRAINT ck_delivery_metrics_count CHECK (event_count >= 0)
);
CREATE INDEX idx_delivery_metrics_scope_date
    ON notify.delivery_daily_metrics (tenant_id, app_id, metric_date DESC);
INSERT INTO iam.permissions (
    code, name, module_code, resource_name, action_name, permission_kind,
    http_methods, route_pattern, description, status
) VALUES
    ('notify.observability.read', '查看消息运营监控', 'notify', 'observability', 'read', 'api', ARRAY['GET'], '/admin-api/v1/apps/{app_id}/notification-operations/*', 'Read application-scoped notification runs, tasks, failures and metrics', 'active'),
    ('notify.task.retry', '重试通知任务', 'notify', 'task', 'retry', 'api', ARRAY['POST'], '/admin-api/v1/apps/{app_id}/notification-retries', 'Retry eligible notification pipeline tasks', 'active'),
    ('notify.message.submit', '提交应用消息', 'notify', 'message', 'submit', 'api', ARRAY['POST'], '/api/v1/apps/{app_id}/notifications', 'Submit application notifications using an API client', 'active'),
    ('notify.message.status.read', '查询应用消息状态', 'notify', 'message_status', 'read', 'api', ARRAY['GET'], '/api/v1/apps/{app_id}/notifications/{message_id}', 'Read notification submission status using an API client', 'active'),
    ('notify.message.broadcast', '广播应用消息', 'notify', 'message', 'broadcast', 'api', ARRAY['POST'], '/api/v1/apps/{app_id}/notifications', 'Submit a notification to all active application members', 'active')
ON CONFLICT (code) DO UPDATE SET
    name=EXCLUDED.name, module_code=EXCLUDED.module_code, resource_name=EXCLUDED.resource_name,
    action_name=EXCLUDED.action_name, permission_kind=EXCLUDED.permission_kind,
    http_methods=EXCLUDED.http_methods, route_pattern=EXCLUDED.route_pattern,
    description=EXCLUDED.description, status='active', updated_at=now();

COMMIT;
