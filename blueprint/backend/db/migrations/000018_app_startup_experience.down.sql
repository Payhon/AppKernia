BEGIN;

DELETE FROM iam.role_permissions
WHERE permission_id = (SELECT id FROM iam.permissions WHERE code = 'app.onboarding.publish');
DELETE FROM iam.permissions WHERE code = 'app.onboarding.publish';

DROP TRIGGER IF EXISTS tr_applications_create_startup_config ON app.applications;
DROP FUNCTION IF EXISTS app.create_startup_config_for_application();
DROP TRIGGER IF EXISTS tr_application_onboarding_draft_assets_touch_updated_at ON app.application_onboarding_draft_assets;
DROP TRIGGER IF EXISTS tr_application_onboarding_draft_slides_touch_updated_at ON app.application_onboarding_draft_slides;
DROP TRIGGER IF EXISTS tr_application_startup_translations_touch_updated_at ON app.application_startup_translations;
DROP TRIGGER IF EXISTS tr_application_startup_configs_touch_updated_at ON app.application_startup_configs;

ALTER TABLE app.application_startup_configs
    DROP CONSTRAINT IF EXISTS fk_application_startup_published_revision;
DROP TABLE IF EXISTS app.application_onboarding_revision_assets;
DROP TABLE IF EXISTS app.application_onboarding_revision_slides;
DROP TABLE IF EXISTS app.application_onboarding_revisions;
DROP TABLE IF EXISTS app.application_onboarding_draft_assets;
DROP TABLE IF EXISTS app.application_onboarding_draft_slides;
DROP TABLE IF EXISTS app.application_startup_translations;
DROP TABLE IF EXISTS app.application_startup_configs;

COMMIT;
