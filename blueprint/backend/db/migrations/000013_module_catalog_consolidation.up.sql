BEGIN;

ALTER TABLE sys.modules
    ADD COLUMN name_key varchar(255) NOT NULL DEFAULT '',
    ADD COLUMN description_key varchar(255) NOT NULL DEFAULT '';

DELETE FROM sys.role_menus
WHERE menu_id IN (
    SELECT id FROM sys.menus
    WHERE tenant_id IS NULL AND code = 'system.settings.modules'
);

UPDATE sys.menus
SET status = 'disabled', updated_at = now()
WHERE tenant_id IS NULL AND code = 'system.settings.modules';

DELETE FROM iam.permissions
WHERE code = 'sys.module.read';

COMMIT;
