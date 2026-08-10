BEGIN;

-- Refuse a lossy downgrade. The 000016 model cannot represent WGT, multi-target
-- releases, internal package references, application teams/assets/channels or
-- store metadata.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM sys.mobile_releases r
        WHERE r.package_type <> 'native_app' OR r.package_file_id IS NOT NULL
           OR (SELECT count(*) FROM sys.mobile_release_targets t WHERE t.release_id = r.id) <> 1
           OR NOT EXISTS (
               SELECT 1 FROM sys.mobile_release_translations zh
               JOIN sys.mobile_release_translations en ON en.release_id=zh.release_id AND en.locale='en-US'
               WHERE zh.release_id=r.id AND zh.locale='zh-CN'
                 AND btrim(zh.contents) <> '' AND btrim(en.contents) <> ''
           )
    ) OR EXISTS (SELECT 1 FROM app.application_team_members)
      OR EXISTS (SELECT 1 FROM app.application_assets)
      OR EXISTS (SELECT 1 FROM app.application_channels)
      OR EXISTS (SELECT 1 FROM app.application_store_listings)
      OR EXISTS (
          SELECT 1 FROM app.applications
          WHERE description <> '' OR introduction <> '' OR remark <> '' OR icon_file_id IS NOT NULL
             OR creator_user_id IS NOT NULL OR owner_type <> 'tenant' OR deleted_at IS NOT NULL
      ) THEN
        RAISE EXCEPTION '000017 downgrade refused: data exists that 000016 cannot represent';
    END IF;
END $$;

UPDATE sys.mobile_releases r
SET platform = t.platform,
    current_version = r.version,
    minimum_version = COALESCE(r.minimum_native_version, '0.0.0'),
    upgrade_url = r.external_url,
    release_notes = jsonb_build_object(
        'zh-CN', (SELECT x.contents FROM sys.mobile_release_translations x WHERE x.release_id=r.id AND x.locale='zh-CN'),
        'en-US', (SELECT x.contents FROM sys.mobile_release_translations x WHERE x.release_id=r.id AND x.locale='en-US')
    ),
    active = EXISTS (SELECT 1 FROM sys.mobile_release_publications p WHERE p.release_id=r.id)
FROM sys.mobile_release_targets t WHERE t.release_id=r.id;

DROP INDEX IF EXISTS sys.idx_mobile_release_publications_release;
DROP TABLE IF EXISTS sys.mobile_release_publications;
DROP TABLE IF EXISTS sys.mobile_release_store_listings;
DROP TABLE IF EXISTS sys.mobile_release_translations;
DROP TABLE IF EXISTS sys.mobile_release_targets;
DROP INDEX IF EXISTS sys.idx_mobile_releases_history;

ALTER TABLE sys.mobile_releases
    DROP CONSTRAINT IF EXISTS ck_mobile_release_source,
    DROP CONSTRAINT IF EXISTS ck_mobile_release_notes_json,
    DROP CONSTRAINT IF EXISTS ck_mobile_release_notes,
    DROP CONSTRAINT IF EXISTS ck_mobile_release_external_url,
    DROP CONSTRAINT IF EXISTS ck_mobile_release_create_env,
    DROP CONSTRAINT IF EXISTS ck_mobile_release_minimum_native_version,
    DROP CONSTRAINT IF EXISTS ck_mobile_release_version,
    DROP CONSTRAINT IF EXISTS ck_mobile_release_package_type,
    DROP CONSTRAINT IF EXISTS fk_mobile_releases_package_file,
    DROP CONSTRAINT IF EXISTS fk_mobile_releases_app_same_tenant,
    ADD CONSTRAINT fk_mobile_releases_app FOREIGN KEY (app_id) REFERENCES app.applications(id) ON DELETE CASCADE,
    ADD CONSTRAINT ck_mobile_release_active_upgrade_url CHECK (NOT active OR upgrade_url IS NOT NULL),
    ADD CONSTRAINT ck_mobile_release_notes CHECK (
        jsonb_typeof(release_notes) = 'object'
        AND jsonb_typeof(release_notes -> 'zh-CN') = 'string'
        AND jsonb_typeof(release_notes -> 'en-US') = 'string'
        AND btrim(release_notes ->> 'zh-CN') <> ''
        AND btrim(release_notes ->> 'en-US') <> ''
    ),
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS unpublished_at,
    DROP COLUMN IF EXISTS last_published_at,
    DROP COLUMN IF EXISTS ever_published_at,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS is_mandatory,
    DROP COLUMN IF EXISTS is_silently,
    DROP COLUMN IF EXISTS create_env,
    DROP COLUMN IF EXISTS external_url,
    DROP COLUMN IF EXISTS package_file_id,
    DROP COLUMN IF EXISTS minimum_native_version,
    DROP COLUMN IF EXISTS version,
    DROP COLUMN IF EXISTS package_type,
    DROP COLUMN IF EXISTS tenant_id;
CREATE UNIQUE INDEX uq_mobile_releases_active_app_platform
    ON sys.mobile_releases (app_id, platform) WHERE active;

DROP TRIGGER IF EXISTS tr_application_store_listings_touch_updated_at ON app.application_store_listings;
DROP TABLE IF EXISTS app.application_store_listings;
DROP TRIGGER IF EXISTS tr_application_channels_touch_updated_at ON app.application_channels;
DROP TABLE IF EXISTS app.application_channels;
DROP TABLE IF EXISTS app.application_assets;
DROP TABLE IF EXISTS app.application_team_members;
DROP TRIGGER IF EXISTS tr_applications_protect_identity ON app.applications;
DROP FUNCTION IF EXISTS app.protect_application_identity();
DROP INDEX IF EXISTS app.idx_applications_tenant_live;
DROP INDEX IF EXISTS app.uq_applications_manifest_appid;

ALTER TABLE app.applications
    DROP CONSTRAINT IF EXISTS fk_applications_icon_file,
    DROP CONSTRAINT IF EXISTS ck_applications_appid,
    DROP CONSTRAINT IF EXISTS ck_applications_owner_reference,
    DROP CONSTRAINT IF EXISTS ck_applications_owner_type,
    DROP CONSTRAINT IF EXISTS ck_applications_app_type,
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS icon_file_id,
    DROP COLUMN IF EXISTS owner_tenant_id,
    DROP COLUMN IF EXISTS owner_user_id,
    DROP COLUMN IF EXISTS owner_type,
    DROP COLUMN IF EXISTS creator_user_id,
    DROP COLUMN IF EXISTS remark,
    DROP COLUMN IF EXISTS introduction,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS app_type,
    DROP COLUMN IF EXISTS appid_configured_at,
    DROP COLUMN IF EXISTS appid;

CREATE OR REPLACE FUNCTION app.create_default_application_for_tenant() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO app.applications (id, tenant_id, code, name, is_default)
    VALUES (
        CASE WHEN NEW.code::text = 'local'
            THEN '00000000-0000-4000-8000-000000000001'::uuid
            ELSE uuidv7()
        END,
        NEW.id, 'default-app', NEW.name || ' App', true
    ) ON CONFLICT (tenant_id, code) DO NOTHING;
    RETURN NEW;
END;
$$;

COMMIT;
