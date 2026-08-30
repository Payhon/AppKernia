BEGIN;

DELETE FROM iam.role_permissions
WHERE permission_id IN (
    SELECT id FROM iam.permissions
    WHERE code IN ('app.scanner_config.read', 'app.scanner_config.update')
);

DELETE FROM iam.permissions
WHERE code IN ('app.scanner_config.read', 'app.scanner_config.update');

DROP TABLE IF EXISTS app.application_scanner_configs;

COMMIT;
