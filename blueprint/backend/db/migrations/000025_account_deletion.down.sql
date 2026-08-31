BEGIN;

DROP TABLE IF EXISTS audit.privacy_erasure_objects;
DROP TABLE IF EXISTS audit.privacy_erasure_events;

DROP INDEX IF EXISTS storage.idx_storage_upload_sessions_app_user;
ALTER TABLE storage.upload_sessions
    DROP CONSTRAINT IF EXISTS fk_storage_upload_sessions_app_same_tenant,
    DROP COLUMN IF EXISTS app_id;

DROP INDEX IF EXISTS storage.idx_storage_files_app_owner;
ALTER TABLE storage.files
    DROP CONSTRAINT IF EXISTS fk_storage_files_app_same_tenant,
    DROP COLUMN IF EXISTS app_id;

DELETE FROM app.legal_consents WHERE user_id IS NULL;
ALTER TABLE app.legal_consents
    DROP CONSTRAINT ck_legal_consents_anonymized,
    DROP CONSTRAINT legal_consents_user_id_fkey,
    DROP COLUMN anonymized_at,
    ALTER COLUMN user_id SET NOT NULL,
    ADD CONSTRAINT legal_consents_user_id_fkey FOREIGN KEY (user_id) REFERENCES iam.users(id) ON DELETE CASCADE;

DROP INDEX IF EXISTS audit.idx_operation_logs_app_user_time;
DROP TRIGGER IF EXISTS tr_operation_logs_assign_app ON audit.operation_logs;
DROP FUNCTION IF EXISTS audit.assign_operation_log_app_id();
ALTER TABLE audit.operation_logs DROP COLUMN IF EXISTS app_id;

ALTER TABLE content.comments
    DROP CONSTRAINT comments_parent_id_fkey,
    DROP CONSTRAINT comments_root_id_fkey,
    ADD CONSTRAINT comments_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES content.comments(id) ON DELETE RESTRICT,
    ADD CONSTRAINT comments_root_id_fkey FOREIGN KEY (root_id) REFERENCES content.comments(id) ON DELETE RESTRICT;

ALTER TABLE iam.verification_challenges
    DROP CONSTRAINT ck_verification_type,
    ADD CONSTRAINT ck_verification_type CHECK (
        challenge_type IN ('email_otp', 'sms_otp', 'password_reset', 'email_verify', 'mobile_verify', 'login_otp')
    );

COMMIT;
