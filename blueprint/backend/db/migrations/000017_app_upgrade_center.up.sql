BEGIN;

-- Application identity and descriptive metadata. The UUID remains the routing
-- identity used by X-AppID; appid is the immutable manifest identity.
ALTER TABLE app.applications
    ADD COLUMN appid citext,
    ADD COLUMN appid_configured_at timestamptz,
    ADD COLUMN app_type varchar(16) NOT NULL DEFAULT 'uni_app_x',
    ADD COLUMN description text NOT NULL DEFAULT '',
    ADD COLUMN introduction text NOT NULL DEFAULT '',
    ADD COLUMN remark text NOT NULL DEFAULT '',
    ADD COLUMN creator_user_id uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    ADD COLUMN owner_type varchar(16) NOT NULL DEFAULT 'tenant',
    ADD COLUMN owner_user_id uuid REFERENCES iam.users(id) ON DELETE RESTRICT,
    ADD COLUMN owner_tenant_id uuid REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    ADD COLUMN icon_file_id uuid,
    ADD COLUMN deleted_at timestamptz,
    ADD CONSTRAINT ck_applications_app_type CHECK (app_type IN ('uni_app', 'uni_app_x')),
    ADD CONSTRAINT ck_applications_owner_type CHECK (owner_type IN ('user', 'tenant')),
    ADD CONSTRAINT ck_applications_appid CHECK (
        appid IS NULL OR appid::text ~ '^__UNI__[A-Za-z0-9_]{2,120}$'
    ),
    ADD CONSTRAINT fk_applications_icon_file
        FOREIGN KEY (tenant_id, icon_file_id) REFERENCES storage.files(tenant_id, id) ON DELETE RESTRICT;

UPDATE app.applications
SET owner_tenant_id = tenant_id,
    appid = CASE WHEN id = '00000000-0000-4000-8000-000000000001'::uuid
                 THEN '__UNI__APPKERNIA'::citext ELSE NULL END,
    appid_configured_at = CASE WHEN id = '00000000-0000-4000-8000-000000000001'::uuid
                               THEN now() ELSE NULL END;

ALTER TABLE app.applications ADD CONSTRAINT ck_applications_owner_reference CHECK (
    (owner_type = 'user' AND owner_user_id IS NOT NULL AND owner_tenant_id IS NULL)
    OR (owner_type = 'tenant' AND owner_user_id IS NULL AND owner_tenant_id IS NOT NULL)
);

CREATE UNIQUE INDEX uq_applications_manifest_appid
    ON app.applications (appid) WHERE appid IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_applications_tenant_live
    ON app.applications (tenant_id, app_type, status, created_at DESC) WHERE deleted_at IS NULL;

CREATE OR REPLACE FUNCTION app.protect_application_identity() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.app_type IS DISTINCT FROM OLD.app_type THEN
        RAISE EXCEPTION 'application app_type is immutable';
    END IF;
    IF OLD.appid IS NOT NULL AND NEW.appid IS DISTINCT FROM OLD.appid THEN
        RAISE EXCEPTION 'application manifest appid is immutable once configured';
    END IF;
    IF OLD.appid IS NULL AND NEW.appid IS NOT NULL THEN
        NEW.appid_configured_at = now();
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER tr_applications_protect_identity
BEFORE UPDATE OF appid, app_type ON app.applications
FOR EACH ROW EXECUTE FUNCTION app.protect_application_identity();

-- New tenant defaults are created with the same known local manifest identity.
CREATE OR REPLACE FUNCTION app.create_default_application_for_tenant() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO app.applications (
        id, tenant_id, code, name, is_default, appid, appid_configured_at,
        app_type, owner_type, owner_tenant_id
    ) VALUES (
        CASE WHEN NEW.code::text = 'local'
            THEN '00000000-0000-4000-8000-000000000001'::uuid
            ELSE uuidv7()
        END,
        NEW.id, 'default-app', NEW.name || ' App', true,
        CASE WHEN NEW.code::text = 'local' THEN '__UNI__APPKERNIA'::citext ELSE NULL END,
        CASE WHEN NEW.code::text = 'local' THEN now() ELSE NULL END,
        'uni_app_x', 'tenant', NEW.id
    ) ON CONFLICT (tenant_id, code) DO NOTHING;
    RETURN NEW;
END;
$$;

CREATE TABLE app.application_team_members (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role varchar(16) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (app_id, user_id, role),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id) ON DELETE RESTRICT,
    CONSTRAINT ck_application_team_role CHECK (role IN ('manager', 'member'))
);
CREATE INDEX idx_application_team_user ON app.application_team_members (tenant_id, user_id, role);

CREATE TABLE app.application_assets (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    file_id uuid NOT NULL,
    asset_type varchar(16) NOT NULL,
    position integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, file_id) REFERENCES storage.files(tenant_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_application_asset_position UNIQUE (app_id, asset_type, position),
    CONSTRAINT ck_application_asset_type CHECK (asset_type IN ('screenshot', 'qrcode')),
    CONSTRAINT ck_application_asset_position CHECK (position >= 0)
);
CREATE INDEX idx_application_assets_order ON app.application_assets (app_id, asset_type, position, id);

CREATE TABLE app.application_channels (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    channel_code varchar(24) NOT NULL,
    name varchar(160) NOT NULL DEFAULT '',
    url varchar(2000),
    abm_url varchar(2000),
    qrcode_file_id uuid,
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, qrcode_file_id) REFERENCES storage.files(tenant_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_application_channel UNIQUE (app_id, channel_code),
    CONSTRAINT ck_application_channel_code CHECK (channel_code IN (
        'android','ios','harmony','h5','quickapp','mp_weixin','mp_alipay','mp_baidu',
        'mp_toutiao','mp_qq','mp_kuaishou','mp_lark','mp_jd','mp_dingtalk'
    )),
    CONSTRAINT ck_application_channel_url CHECK (url IS NULL OR url ~ '^https://[^[:space:]]+$'),
    CONSTRAINT ck_application_channel_abm_url CHECK (abm_url IS NULL OR abm_url ~ '^https://[^[:space:]]+$')
);
CREATE TRIGGER tr_application_channels_touch_updated_at BEFORE UPDATE ON app.application_channels
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

CREATE TABLE app.application_store_listings (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    name varchar(160) NOT NULL,
    scheme varchar(255) NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT true,
    priority integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_application_store_name UNIQUE (app_id, name),
    CONSTRAINT uq_application_store_tenant_app_id UNIQUE (tenant_id, app_id, id),
    CONSTRAINT ck_application_store_priority CHECK (priority BETWEEN -100000 AND 100000),
    CONSTRAINT ck_application_store_scheme CHECK (scheme = '' OR scheme ~ '^[A-Za-z][A-Za-z0-9+.-]*://')
);
CREATE INDEX idx_application_store_priority ON app.application_store_listings (app_id, enabled, priority DESC, id);
CREATE TRIGGER tr_application_store_listings_touch_updated_at BEFORE UPDATE ON app.application_store_listings
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

-- Expand releases while keeping the legacy columns as a compatibility
-- projection. New code treats targets/translations/publications as authority.
ALTER TABLE sys.mobile_releases
    ADD COLUMN tenant_id uuid,
    ADD COLUMN package_type varchar(16) NOT NULL DEFAULT 'native_app',
    ADD COLUMN version varchar(64),
    ADD COLUMN minimum_native_version varchar(64),
    ADD COLUMN package_file_id uuid,
    ADD COLUMN external_url varchar(2000),
    ADD COLUMN create_env varchar(24) NOT NULL DEFAULT 'upgrade_center',
    ADD COLUMN is_silently boolean NOT NULL DEFAULT false,
    ADD COLUMN is_mandatory boolean NOT NULL DEFAULT false,
    ADD COLUMN created_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    ADD COLUMN ever_published_at timestamptz,
    ADD COLUMN last_published_at timestamptz,
    ADD COLUMN unpublished_at timestamptz,
    ADD COLUMN deleted_at timestamptz;

UPDATE sys.mobile_releases r
SET tenant_id = a.tenant_id,
    version = r.current_version,
    minimum_native_version = r.minimum_version,
    external_url = r.upgrade_url,
    ever_published_at = r.created_at,
    last_published_at = CASE WHEN r.active THEN r.updated_at ELSE r.created_at END
FROM app.applications a WHERE a.id = r.app_id;

ALTER TABLE sys.mobile_releases
    ALTER COLUMN tenant_id SET NOT NULL,
    ALTER COLUMN version SET NOT NULL,
    DROP CONSTRAINT IF EXISTS ck_mobile_release_active_upgrade_url,
    DROP CONSTRAINT IF EXISTS ck_mobile_release_notes,
    DROP CONSTRAINT IF EXISTS fk_mobile_releases_app,
    ADD CONSTRAINT fk_mobile_releases_app_same_tenant
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_mobile_releases_package_file
        FOREIGN KEY (tenant_id, package_file_id) REFERENCES storage.files(tenant_id, id) ON DELETE RESTRICT,
    ADD CONSTRAINT ck_mobile_release_package_type CHECK (package_type IN ('native_app', 'wgt')),
    ADD CONSTRAINT ck_mobile_release_version CHECK (version ~ '^[0-9]+\.[0-9]+\.[0-9]+$'),
    ADD CONSTRAINT ck_mobile_release_minimum_native_version CHECK (
        minimum_native_version IS NULL OR minimum_native_version ~ '^[0-9]+\.[0-9]+\.[0-9]+$'
    ),
    ADD CONSTRAINT ck_mobile_release_create_env CHECK (create_env IN ('uni_stat', 'upgrade_center')),
    ADD CONSTRAINT ck_mobile_release_external_url CHECK (external_url IS NULL OR external_url ~ '^https://[^[:space:]]+$'),
    ADD CONSTRAINT ck_mobile_release_source CHECK (package_file_id IS NULL OR external_url IS NULL),
    ADD CONSTRAINT ck_mobile_release_notes_json CHECK (jsonb_typeof(release_notes) = 'object'),
    ADD CONSTRAINT uq_mobile_release_tenant_app_id UNIQUE (tenant_id, app_id, id),
    ADD CONSTRAINT uq_mobile_release_tenant_app_package_id UNIQUE (tenant_id, app_id, package_type, id);

DROP INDEX IF EXISTS sys.uq_mobile_releases_active_app_platform;
CREATE INDEX idx_mobile_releases_history
    ON sys.mobile_releases (app_id, package_type, created_at DESC, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE sys.mobile_release_targets (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    release_id uuid NOT NULL,
    package_type varchar(16) NOT NULL,
    platform varchar(16) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (release_id, platform),
    CONSTRAINT uq_mobile_release_target_identity UNIQUE (tenant_id, app_id, package_type, release_id, platform),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, app_id, package_type, release_id)
        REFERENCES sys.mobile_releases(tenant_id, app_id, package_type, id) ON DELETE CASCADE,
    CONSTRAINT ck_mobile_release_target_package CHECK (package_type IN ('native_app', 'wgt')),
    CONSTRAINT ck_mobile_release_target_platform CHECK (platform IN ('android', 'ios', 'harmony'))
);

CREATE TABLE sys.mobile_release_translations (
    release_id uuid NOT NULL REFERENCES sys.mobile_releases(id) ON DELETE CASCADE,
    locale varchar(16) NOT NULL,
    title varchar(200) NOT NULL,
    contents text NOT NULL,
    PRIMARY KEY (release_id, locale),
    CONSTRAINT ck_mobile_release_translation_locale CHECK (locale IN ('zh-CN', 'en-US')),
    CONSTRAINT ck_mobile_release_translation_values CHECK (btrim(title) <> '' AND btrim(contents) <> '')
);

CREATE TABLE sys.mobile_release_store_listings (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    release_id uuid NOT NULL,
    store_listing_id uuid NOT NULL,
    FOREIGN KEY (tenant_id, app_id, release_id)
        REFERENCES sys.mobile_releases(tenant_id, app_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, app_id, store_listing_id)
        REFERENCES app.application_store_listings(tenant_id, app_id, id) ON DELETE RESTRICT,
    PRIMARY KEY (release_id, store_listing_id)
);

CREATE TABLE sys.mobile_release_publications (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    package_type varchar(16) NOT NULL,
    platform varchar(16) NOT NULL,
    release_id uuid NOT NULL REFERENCES sys.mobile_releases(id) ON DELETE RESTRICT,
    published_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    published_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (app_id, package_type, platform),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, app_id, package_type, release_id, platform)
        REFERENCES sys.mobile_release_targets(tenant_id, app_id, package_type, release_id, platform) ON DELETE RESTRICT,
    CONSTRAINT ck_mobile_release_publication_package CHECK (package_type IN ('native_app', 'wgt')),
    CONSTRAINT ck_mobile_release_publication_platform CHECK (platform IN ('android', 'ios', 'harmony'))
);
CREATE INDEX idx_mobile_release_publications_release ON sys.mobile_release_publications (release_id, platform);

INSERT INTO sys.mobile_release_targets (tenant_id, app_id, release_id, package_type, platform)
SELECT tenant_id, app_id, id, 'native_app', platform FROM sys.mobile_releases;

INSERT INTO sys.mobile_release_translations (release_id, locale, title, contents)
SELECT id, 'zh-CN', '版本更新', release_notes ->> 'zh-CN' FROM sys.mobile_releases
UNION ALL
SELECT id, 'en-US', 'Version update', release_notes ->> 'en-US' FROM sys.mobile_releases;

INSERT INTO sys.mobile_release_publications (tenant_id, app_id, package_type, platform, release_id, published_at)
SELECT tenant_id, app_id, 'native_app', platform, id, updated_at
FROM sys.mobile_releases WHERE active;

COMMIT;
