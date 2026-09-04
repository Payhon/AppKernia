BEGIN;

ALTER TABLE iam.verification_challenges DROP CONSTRAINT ck_verification_type;
ALTER TABLE iam.verification_challenges ADD CONSTRAINT ck_verification_type CHECK (
    challenge_type IN ('email_otp', 'sms_otp', 'password_reset', 'email_verify', 'mobile_verify', 'login_otp', 'registration_otp', 'account_delete', 'step_up')
);

CREATE TABLE app.application_login_settings (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    otp_login_enabled boolean NOT NULL DEFAULT false,
    email_otp_enabled boolean NOT NULL DEFAULT true,
    sms_otp_enabled boolean NOT NULL DEFAULT false,
    lock_version integer NOT NULL DEFAULT 1,
    created_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, app_id),
    CONSTRAINT fk_application_login_settings_app
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_application_login_settings_otp_channels CHECK (
        NOT otp_login_enabled OR email_otp_enabled OR sms_otp_enabled
    ),
    CONSTRAINT ck_application_login_settings_lock_version CHECK (lock_version > 0)
);

CREATE TRIGGER tr_application_login_settings_touch_updated_at
BEFORE UPDATE ON app.application_login_settings
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

INSERT INTO iam.permissions (
    code, name, module_code, resource_name, action_name, permission_kind,
    http_methods, route_pattern, description, status
) VALUES
    ('app.login_settings.read', '查看应用登录配置', 'app', 'login_settings', 'read', 'api', ARRAY['GET'], '/admin-api/v1/apps/{app_id}/login-settings', 'Read one App login configuration', 'active'),
    ('app.login_settings.update', '更新应用登录配置', 'app', 'login_settings', 'update', 'api', ARRAY['PUT'], '/admin-api/v1/apps/{app_id}/login-settings', 'Update one App login configuration', 'active')
ON CONFLICT (code) DO UPDATE SET
    name=EXCLUDED.name,
    module_code=EXCLUDED.module_code,
    resource_name=EXCLUDED.resource_name,
    action_name=EXCLUDED.action_name,
    permission_kind=EXCLUDED.permission_kind,
    http_methods=EXCLUDED.http_methods,
    route_pattern=EXCLUDED.route_pattern,
    description=EXCLUDED.description,
    status='active',
    updated_at=now();

INSERT INTO iam.role_permissions (tenant_id, role_id, permission_id, granted_by)
SELECT existing.tenant_id, existing.role_id, login_settings.id, existing.granted_by
FROM iam.role_permissions existing
JOIN iam.permissions application_permission ON application_permission.id=existing.permission_id
JOIN iam.permissions login_settings ON login_settings.code=CASE application_permission.code
    WHEN 'app.application.read' THEN 'app.login_settings.read'
    WHEN 'app.application.update' THEN 'app.login_settings.update'
END
WHERE application_permission.code IN ('app.application.read', 'app.application.update')
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

COMMIT;
