BEGIN;

DELETE FROM sys.role_menus
WHERE menu_id = (
    SELECT id FROM sys.menus
    WHERE tenant_id IS NULL AND code = 'system.settings.login-providers'
);
DELETE FROM sys.menus
WHERE tenant_id IS NULL AND code = 'system.settings.login-providers';

DELETE FROM iam.role_permissions
WHERE permission_id IN (
    SELECT id FROM iam.permissions WHERE code IN (
        'sys.login_provider_config.read', 'sys.login_provider_config.create',
        'sys.login_provider_config.update', 'sys.login_provider_config.delete',
        'sys.login_provider_config.rotate_secret', 'sys.login_provider_config.preflight',
        'app.login_provider_binding.read', 'app.login_provider_binding.update'
    )
);
DELETE FROM iam.permissions WHERE code IN (
    'sys.login_provider_config.read', 'sys.login_provider_config.create',
    'sys.login_provider_config.update', 'sys.login_provider_config.delete',
    'sys.login_provider_config.rotate_secret', 'sys.login_provider_config.preflight',
    'app.login_provider_binding.read', 'app.login_provider_binding.update'
);

UPDATE app.user_memberships
SET source = 'legacy'
WHERE source = 'federated_registration';

ALTER TABLE app.user_memberships DROP CONSTRAINT ck_app_user_memberships_source;
ALTER TABLE app.user_memberships ADD CONSTRAINT ck_app_user_memberships_source
    CHECK (source IN ('self_registration', 'admin_created', 'legacy'));

DELETE FROM iam.verification_challenges
WHERE challenge_type = 'step_up';

ALTER TABLE iam.verification_challenges DROP CONSTRAINT ck_verification_type;
ALTER TABLE iam.verification_challenges ADD CONSTRAINT ck_verification_type CHECK (
    challenge_type IN ('email_otp', 'sms_otp', 'password_reset', 'email_verify', 'mobile_verify', 'login_otp', 'account_delete')
);

DROP TABLE IF EXISTS app.user_login_identifiers;
DROP TABLE IF EXISTS iam.oauth_authorization_flows;
DROP TABLE IF EXISTS iam.app_oauth_accounts;
DROP TABLE IF EXISTS app.application_login_provider_bindings;
DROP TABLE IF EXISTS sys.login_provider_configs;

COMMIT;
