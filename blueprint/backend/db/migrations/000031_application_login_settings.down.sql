BEGIN;

ALTER TABLE iam.verification_challenges DROP CONSTRAINT ck_verification_type;
ALTER TABLE iam.verification_challenges ADD CONSTRAINT ck_verification_type CHECK (
    challenge_type IN ('email_otp', 'sms_otp', 'password_reset', 'email_verify', 'mobile_verify', 'login_otp', 'account_delete', 'step_up')
);

DELETE FROM iam.role_permissions
WHERE permission_id IN (
    SELECT id FROM iam.permissions WHERE code IN ('app.login_settings.read', 'app.login_settings.update')
);
DELETE FROM iam.permissions WHERE code IN ('app.login_settings.read', 'app.login_settings.update');
DROP TABLE IF EXISTS app.application_login_settings;

COMMIT;
