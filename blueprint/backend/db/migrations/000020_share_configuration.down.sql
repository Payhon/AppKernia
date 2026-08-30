BEGIN;

DELETE FROM sys.role_menus
WHERE menu_id = (SELECT id FROM sys.menus WHERE tenant_id IS NULL AND code = 'system.settings.share-configs');
DELETE FROM sys.menus WHERE tenant_id IS NULL AND code = 'system.settings.share-configs';

DELETE FROM iam.role_permissions
WHERE permission_id IN (
    SELECT id FROM iam.permissions WHERE code IN (
        'sys.share_config.read','sys.share_config.create','sys.share_config.update',
        'sys.share_config.delete','sys.share_config.rotate_secret',
        'app.share_binding.read','app.share_binding.update'
    )
);
DELETE FROM iam.permissions WHERE code IN (
    'sys.share_config.read','sys.share_config.create','sys.share_config.update',
    'sys.share_config.delete','sys.share_config.rotate_secret',
    'app.share_binding.read','app.share_binding.update'
);

DROP TABLE IF EXISTS app.application_share_bindings;
DROP TABLE IF EXISTS sys.share_configs;

COMMIT;
