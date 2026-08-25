BEGIN;

CREATE TABLE app.application_startup_configs (
    tenant_id uuid NOT NULL,
    app_id uuid PRIMARY KEY,
    onboarding_enabled boolean NOT NULL DEFAULT false,
    draft_generation integer NOT NULL DEFAULT 1,
    published_version integer NOT NULL DEFAULT 0,
    published_revision_id uuid,
    published_at timestamptz,
    published_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_application_startup_generation CHECK (draft_generation > 0),
    CONSTRAINT ck_application_startup_version CHECK (published_version >= 0),
    CONSTRAINT ck_application_startup_publication CHECK (
        (published_version = 0 AND published_revision_id IS NULL AND published_at IS NULL)
        OR (published_version > 0 AND published_revision_id IS NOT NULL AND published_at IS NOT NULL)
    )
);

CREATE TABLE app.application_startup_translations (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    locale varchar(16) NOT NULL,
    display_name varchar(120) NOT NULL,
    subtitle varchar(240) NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (app_id, locale),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_application_startup_translation_locale CHECK (locale IN ('zh-CN', 'en-US')),
    CONSTRAINT ck_application_startup_translation_name CHECK (btrim(display_name) <> '')
);

CREATE TABLE app.application_onboarding_draft_slides (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    position integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_application_onboarding_draft_position UNIQUE (app_id, position),
    CONSTRAINT uq_application_onboarding_draft_identity UNIQUE (tenant_id, app_id, id),
    CONSTRAINT ck_application_onboarding_draft_position CHECK (position BETWEEN 0 AND 9)
);

CREATE TABLE app.application_onboarding_draft_assets (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    slide_id uuid NOT NULL,
    locale varchar(16) NOT NULL,
    file_id uuid NOT NULL,
    accessibility_label varchar(500) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (slide_id, locale),
    FOREIGN KEY (tenant_id, app_id, slide_id)
        REFERENCES app.application_onboarding_draft_slides(tenant_id, app_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, file_id) REFERENCES storage.files(tenant_id, id) ON DELETE RESTRICT,
    CONSTRAINT ck_application_onboarding_draft_asset_locale CHECK (locale IN ('zh-CN', 'en-US')),
    CONSTRAINT ck_application_onboarding_draft_asset_label CHECK (btrim(accessibility_label) <> '')
);

CREATE TABLE app.application_onboarding_revisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    version integer NOT NULL,
    source_generation integer NOT NULL,
    published_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    published_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_application_onboarding_revision_version UNIQUE (app_id, version),
    CONSTRAINT uq_application_onboarding_revision_identity UNIQUE (tenant_id, app_id, id),
    CONSTRAINT ck_application_onboarding_revision_version CHECK (version > 0),
    CONSTRAINT ck_application_onboarding_revision_generation CHECK (source_generation > 0)
);

CREATE TABLE app.application_onboarding_revision_slides (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    revision_id uuid NOT NULL,
    position integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, app_id, revision_id)
        REFERENCES app.application_onboarding_revisions(tenant_id, app_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_application_onboarding_revision_slide_position UNIQUE (revision_id, position),
    CONSTRAINT uq_application_onboarding_revision_slide_identity UNIQUE (tenant_id, app_id, revision_id, id),
    CONSTRAINT ck_application_onboarding_revision_slide_position CHECK (position BETWEEN 0 AND 9)
);

CREATE TABLE app.application_onboarding_revision_assets (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    revision_id uuid NOT NULL,
    slide_id uuid NOT NULL,
    locale varchar(16) NOT NULL,
    file_id uuid NOT NULL,
    accessibility_label varchar(500) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (slide_id, locale),
    FOREIGN KEY (tenant_id, app_id, revision_id, slide_id)
        REFERENCES app.application_onboarding_revision_slides(tenant_id, app_id, revision_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, file_id) REFERENCES storage.files(tenant_id, id) ON DELETE RESTRICT,
    CONSTRAINT ck_application_onboarding_revision_asset_locale CHECK (locale IN ('zh-CN', 'en-US')),
    CONSTRAINT ck_application_onboarding_revision_asset_label CHECK (btrim(accessibility_label) <> '')
);

ALTER TABLE app.application_startup_configs
    ADD CONSTRAINT fk_application_startup_published_revision
    FOREIGN KEY (tenant_id, app_id, published_revision_id)
    REFERENCES app.application_onboarding_revisions(tenant_id, app_id, id) ON DELETE RESTRICT;

CREATE INDEX idx_application_onboarding_draft_assets_file
    ON app.application_onboarding_draft_assets (tenant_id, file_id);
CREATE INDEX idx_application_onboarding_revision_assets_file
    ON app.application_onboarding_revision_assets (tenant_id, file_id);

CREATE TRIGGER tr_application_startup_configs_touch_updated_at
BEFORE UPDATE ON app.application_startup_configs FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_application_startup_translations_touch_updated_at
BEFORE UPDATE ON app.application_startup_translations FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_application_onboarding_draft_slides_touch_updated_at
BEFORE UPDATE ON app.application_onboarding_draft_slides FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_application_onboarding_draft_assets_touch_updated_at
BEFORE UPDATE ON app.application_onboarding_draft_assets FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

INSERT INTO app.application_startup_configs (tenant_id, app_id)
SELECT tenant_id, id FROM app.applications WHERE deleted_at IS NULL;

INSERT INTO app.application_startup_translations (tenant_id, app_id, locale, display_name, subtitle)
SELECT tenant_id, id, locale, name, ''
FROM app.applications
CROSS JOIN (VALUES ('zh-CN'), ('en-US')) AS locales(locale)
WHERE deleted_at IS NULL;

CREATE OR REPLACE FUNCTION app.create_startup_config_for_application() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO app.application_startup_configs (tenant_id, app_id)
    VALUES (NEW.tenant_id, NEW.id);
    INSERT INTO app.application_startup_translations (tenant_id, app_id, locale, display_name, subtitle)
    VALUES
        (NEW.tenant_id, NEW.id, 'zh-CN', NEW.name, ''),
        (NEW.tenant_id, NEW.id, 'en-US', NEW.name, '');
    RETURN NEW;
END;
$$;
CREATE TRIGGER tr_applications_create_startup_config
AFTER INSERT ON app.applications FOR EACH ROW EXECUTE FUNCTION app.create_startup_config_for_application();

INSERT INTO iam.permissions (
    code, name, module_code, resource_name, action_name, permission_kind,
    http_methods, route_pattern, description, status
) VALUES (
    'app.onboarding.publish', '发布应用启动介绍', 'app', 'onboarding', 'publish', 'api',
    ARRAY['POST'], '/admin-api/v1/apps/{app_id}/startup/onboarding/publish',
    'Publish an immutable App onboarding revision', 'active'
) ON CONFLICT (code) DO UPDATE SET
    name=EXCLUDED.name, module_code=EXCLUDED.module_code, resource_name=EXCLUDED.resource_name,
    action_name=EXCLUDED.action_name, permission_kind=EXCLUDED.permission_kind,
    http_methods=EXCLUDED.http_methods, route_pattern=EXCLUDED.route_pattern,
    description=EXCLUDED.description, status='active', updated_at=now();

COMMIT;
