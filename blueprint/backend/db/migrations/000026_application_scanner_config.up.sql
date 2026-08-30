BEGIN;

CREATE TABLE app.application_scanner_configs (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    webview_enabled boolean NOT NULL DEFAULT false,
    allowed_host_patterns text[] NOT NULL DEFAULT '{}'::text[],
    lock_version integer NOT NULL DEFAULT 1,
    created_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, app_id),
    CONSTRAINT fk_application_scanner_config_app
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_application_scanner_config_hosts CHECK (cardinality(allowed_host_patterns) <= 100),
    CONSTRAINT ck_application_scanner_config_enabled CHECK (
        NOT webview_enabled OR cardinality(allowed_host_patterns) > 0
    ),
    CONSTRAINT ck_application_scanner_config_lock_version CHECK (lock_version > 0)
);

CREATE TRIGGER tr_application_scanner_configs_touch_updated_at
BEFORE UPDATE ON app.application_scanner_configs
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

INSERT INTO iam.permissions (
    code, name, module_code, resource_name, action_name, permission_kind,
    http_methods, route_pattern, description, status
) VALUES
    ('app.scanner_config.read', '查看应用扫码配置', 'app', 'scanner_config', 'read', 'api', ARRAY['GET'], '/admin-api/v1/apps/{app_id}/scanner-config', 'Read one App scanner configuration', 'active'),
    ('app.scanner_config.update', '更新应用扫码配置', 'app', 'scanner_config', 'update', 'api', ARRAY['PUT'], '/admin-api/v1/apps/{app_id}/scanner-config', 'Update one App scanner configuration', 'active')
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
SELECT existing.tenant_id, existing.role_id, scanner.id, existing.granted_by
FROM iam.role_permissions existing
JOIN iam.permissions application_permission
  ON application_permission.id=existing.permission_id
JOIN iam.permissions scanner
  ON scanner.code=CASE application_permission.code
      WHEN 'app.application.read' THEN 'app.scanner_config.read'
      WHEN 'app.application.update' THEN 'app.scanner_config.update'
  END
WHERE application_permission.code IN ('app.application.read', 'app.application.update')
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

COMMIT;
