BEGIN;

ALTER TABLE iam.verification_challenges
    DROP CONSTRAINT ck_verification_type,
    ADD CONSTRAINT ck_verification_type CHECK (
        challenge_type IN ('email_otp', 'sms_otp', 'password_reset', 'email_verify', 'mobile_verify', 'login_otp', 'account_delete')
    );

ALTER TABLE content.comments
    DROP CONSTRAINT comments_parent_id_fkey,
    DROP CONSTRAINT comments_root_id_fkey,
    ADD CONSTRAINT comments_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES content.comments(id) ON DELETE SET NULL,
    ADD CONSTRAINT comments_root_id_fkey FOREIGN KEY (root_id) REFERENCES content.comments(id) ON DELETE SET NULL;

ALTER TABLE app.legal_consents
    DROP CONSTRAINT legal_consents_user_id_fkey,
    ALTER COLUMN user_id DROP NOT NULL,
    ADD COLUMN anonymized_at timestamptz,
    ADD CONSTRAINT legal_consents_user_id_fkey FOREIGN KEY (user_id) REFERENCES iam.users(id) ON DELETE SET NULL,
    ADD CONSTRAINT ck_legal_consents_anonymized CHECK (
        (user_id IS NOT NULL AND anonymized_at IS NULL)
        OR (user_id IS NULL AND anonymized_at IS NOT NULL AND ip_address IS NULL AND user_agent IS NULL)
    );

ALTER TABLE audit.operation_logs
    ADD COLUMN app_id uuid REFERENCES app.applications(id) ON DELETE SET NULL;
UPDATE audit.operation_logs AS log
SET app_id = session.app_id
FROM iam.sessions AS session
WHERE log.session_id = session.id AND session.app_id IS NOT NULL;
CREATE INDEX idx_operation_logs_app_user_time
    ON audit.operation_logs (app_id, user_id, occurred_at DESC)
    WHERE app_id IS NOT NULL;
CREATE FUNCTION audit.assign_operation_log_app_id() RETURNS trigger AS $$
BEGIN
    IF NEW.app_id IS NULL AND NEW.session_id IS NOT NULL THEN
        SELECT app_id INTO NEW.app_id FROM iam.sessions WHERE id = NEW.session_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER tr_operation_logs_assign_app
BEFORE INSERT ON audit.operation_logs
FOR EACH ROW EXECUTE FUNCTION audit.assign_operation_log_app_id();

ALTER TABLE storage.files
    ADD COLUMN app_id uuid,
    ADD CONSTRAINT fk_storage_files_app_same_tenant
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE SET NULL (app_id);
CREATE INDEX idx_storage_files_app_owner
    ON storage.files (app_id, owner_user_id, created_at DESC)
    WHERE app_id IS NOT NULL;

ALTER TABLE storage.upload_sessions
    ADD COLUMN app_id uuid,
    ADD CONSTRAINT fk_storage_upload_sessions_app_same_tenant
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE SET NULL (app_id);
CREATE INDEX idx_storage_upload_sessions_app_user
    ON storage.upload_sessions (app_id, user_id, created_at DESC)
    WHERE app_id IS NOT NULL;

CREATE TABLE audit.privacy_erasure_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    app_id uuid NOT NULL REFERENCES app.applications(id) ON DELETE RESTRICT,
    action_code varchar(200) NOT NULL DEFAULT 'iam.user.delete_self',
    scope varchar(32) NOT NULL DEFAULT 'current_app',
    status varchar(24) NOT NULL,
    global_identity_deleted boolean NOT NULL DEFAULT false,
    erased_counts jsonb NOT NULL DEFAULT '{}'::jsonb,
    requested_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT ck_privacy_erasure_scope CHECK (scope = 'current_app'),
    CONSTRAINT ck_privacy_erasure_action CHECK (action_code = 'iam.user.delete_self'),
    CONSTRAINT ck_privacy_erasure_status CHECK (status IN ('pending_objects', 'completed', 'failed')),
    CONSTRAINT ck_privacy_erasure_completion CHECK (
        (status = 'pending_objects' AND completed_at IS NULL)
        OR (status IN ('completed', 'failed') AND completed_at IS NOT NULL)
    )
);
CREATE INDEX idx_privacy_erasure_events_app_time
    ON audit.privacy_erasure_events (app_id, requested_at DESC);

CREATE TABLE audit.privacy_erasure_objects (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    event_id uuid NOT NULL REFERENCES audit.privacy_erasure_events(id) ON DELETE CASCADE,
    tenant_id uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    provider varchar(32) NOT NULL,
    bucket_name varchar(255) NOT NULL,
    object_key varchar(1024) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'pending',
    attempt_count integer NOT NULL DEFAULT 0,
    last_error_code varchar(80),
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_privacy_erasure_object UNIQUE (provider, bucket_name, object_key),
    CONSTRAINT ck_privacy_erasure_object_status CHECK (status IN ('pending', 'succeeded', 'failed')),
    CONSTRAINT ck_privacy_erasure_object_attempts CHECK (attempt_count >= 0),
    CONSTRAINT ck_privacy_erasure_object_completion CHECK (
        (status = 'pending' AND completed_at IS NULL)
        OR (status IN ('succeeded', 'failed') AND completed_at IS NOT NULL)
    )
);
CREATE INDEX idx_privacy_erasure_objects_pending
    ON audit.privacy_erasure_objects (status, created_at)
    WHERE status = 'pending';

COMMIT;
