BEGIN;
DELETE FROM iam.role_permissions WHERE permission_id IN (SELECT id FROM iam.permissions WHERE code IN ('app.public_web.read','app.public_web.update'));
DELETE FROM iam.permissions WHERE code IN ('app.public_web.read','app.public_web.update');
DROP TABLE IF EXISTS app.application_public_web_translations;
DROP TABLE IF EXISTS app.application_public_web_configs;
ALTER TABLE app.application_store_listings DROP CONSTRAINT ck_store_web_platform, DROP CONSTRAINT ck_store_web_url, DROP COLUMN platform, DROP COLUMN web_url;
COMMIT;
