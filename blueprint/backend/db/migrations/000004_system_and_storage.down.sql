BEGIN;

ALTER TABLE iam.users DROP CONSTRAINT IF EXISTS fk_users_avatar_file;

DROP TABLE IF EXISTS storage.file_usages;
DROP TABLE IF EXISTS storage.upload_parts;
DROP TABLE IF EXISTS storage.upload_sessions;
DROP TABLE IF EXISTS storage.files;

DROP TABLE IF EXISTS sys.webhook_deliveries;
DROP TABLE IF EXISTS sys.webhook_endpoints;
DROP TABLE IF EXISTS sys.idempotency_keys;
DROP TABLE IF EXISTS sys.api_client_permissions;
DROP TABLE IF EXISTS sys.api_client_secrets;
DROP TABLE IF EXISTS sys.api_clients;
DROP TABLE IF EXISTS sys.regions;
DROP TABLE IF EXISTS sys.dict_items;
DROP TABLE IF EXISTS sys.dict_types;
DROP TABLE IF EXISTS sys.config_items;
DROP TABLE IF EXISTS sys.role_menus;
DROP TABLE IF EXISTS sys.menus;
DROP TABLE IF EXISTS sys.modules;

COMMIT;
