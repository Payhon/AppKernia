BEGIN;

INSERT INTO iam.permissions (
    code, name, module_code, resource_name, action_name, permission_kind, status
)
VALUES ('sys.module.read', 'View modules', 'sys', 'module', 'read', 'api', 'active')
ON CONFLICT (code) DO UPDATE SET status = 'active', updated_at = now();

UPDATE sys.menus
SET permission_id = (
        SELECT id FROM iam.permissions WHERE code = 'sys.module.read'
    ),
    status = 'active',
    updated_at = now()
WHERE tenant_id IS NULL AND code = 'system.settings.modules';

ALTER TABLE sys.modules
    DROP COLUMN description_key,
    DROP COLUMN name_key;

COMMIT;
