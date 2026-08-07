BEGIN;

CREATE SCHEMA IF NOT EXISTS app;

CREATE TABLE app.applications (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    code varchar(64) NOT NULL,
    name varchar(120) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'active',
    default_locale varchar(16) NOT NULL DEFAULT 'zh-CN',
    registration_enabled boolean NOT NULL DEFAULT true,
    registration_verification varchar(16) NOT NULL DEFAULT 'email_otp',
    is_default boolean NOT NULL DEFAULT false,
    lock_version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_applications_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT uq_applications_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT ck_applications_code CHECK (code ~ '^[a-z][a-z0-9-]{1,62}$'),
    CONSTRAINT ck_applications_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_applications_locale CHECK (default_locale IN ('zh-CN', 'en-US')),
    CONSTRAINT ck_applications_verification CHECK (registration_verification IN ('none', 'email_otp')),
    CONSTRAINT ck_applications_lock_version CHECK (lock_version > 0)
);
CREATE UNIQUE INDEX uq_applications_default_per_tenant ON app.applications (tenant_id) WHERE is_default;
CREATE INDEX idx_applications_tenant_status ON app.applications (tenant_id, status, created_at DESC);
CREATE TRIGGER tr_applications_touch_updated_at BEFORE UPDATE ON app.applications
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

-- Tenant creation normally precedes App provisioning (for example bootstrap
-- on an empty database), so the default App is created atomically with the
-- tenant. The `local` tenant has a documented deterministic development App
-- UUID consumed by the checked-in simulator manifest; production tenants use
-- UUIDv7 and their build-time App ID instead.
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
CREATE TRIGGER tr_iam_tenants_create_default_application
AFTER INSERT ON iam.tenants FOR EACH ROW EXECUTE FUNCTION app.create_default_application_for_tenant();

CREATE TABLE app.user_memberships (
    app_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    user_id uuid NOT NULL REFERENCES iam.users(id) ON DELETE RESTRICT,
    source varchar(32) NOT NULL,
    status varchar(32) NOT NULL DEFAULT 'active',
    invited_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    verified_at timestamptz,
    disabled_at timestamptz,
    lock_version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (app_id, user_id),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id) ON DELETE RESTRICT,
    CONSTRAINT ck_app_user_memberships_source CHECK (source IN ('self_registration', 'admin_created', 'legacy')),
    CONSTRAINT ck_app_user_memberships_status CHECK (status IN ('pending_verification', 'active', 'disabled')),
    CONSTRAINT ck_app_user_memberships_lock_version CHECK (lock_version > 0)
);
CREATE INDEX idx_app_user_memberships_tenant_user ON app.user_memberships (tenant_id, user_id, status);
CREATE TRIGGER tr_app_user_memberships_touch_updated_at BEFORE UPDATE ON app.user_memberships
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

-- Legacy tenant-scoped mobile records are deterministically assigned to that
-- tenant's default App. The trigger preserves compatibility for legacy Admin
-- endpoints while all new App endpoints must supply their explicit app scope.
CREATE OR REPLACE FUNCTION app.assign_default_application() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.app_id IS NULL THEN
        SELECT id INTO NEW.app_id FROM app.applications
        WHERE tenant_id = NEW.tenant_id AND is_default;
    END IF;
    IF NEW.app_id IS NULL THEN
        RAISE EXCEPTION 'no default application for tenant %', NEW.tenant_id;
    END IF;
    RETURN NEW;
END;
$$;

-- Every existing tenant receives one immutable-by-convention default application.
-- New code scopes mobile data by this public UUID, never by a caller-provided tenant id.
INSERT INTO app.applications (id, tenant_id, code, name, is_default)
SELECT CASE WHEN t.code::text = 'local'
            THEN '00000000-0000-4000-8000-000000000001'::uuid
            ELSE uuidv7()
       END,
       t.id, 'default-app', t.name || ' App', true
FROM iam.tenants t
ON CONFLICT (tenant_id, code) DO NOTHING;

-- Conservative legacy backfill: only identities with pre-existing mobile
-- evidence are included. This deliberately excludes Admin-only users while
-- preserving users that have preferences, push registration, bookmarks, or a
-- materialized mobile inbox but no currently-live mobile session.
WITH mobile_evidence AS (
    SELECT s.tenant_id, s.user_id FROM iam.sessions s WHERE s.audience = 'ak-mobile'
    UNION
    SELECT tm.tenant_id, p.user_id FROM iam.user_preferences p
    JOIN iam.tenant_members tm ON tm.user_id = p.user_id
    UNION
    SELECT tm.tenant_id, d.user_id FROM notify.push_devices d
    JOIN iam.tenant_members tm ON tm.user_id = d.user_id
    UNION
    SELECT tenant_id, user_id FROM content.article_bookmarks
    UNION
    SELECT tenant_id, user_id FROM notify.recipients
)
INSERT INTO app.user_memberships (app_id, tenant_id, user_id, source, status, verified_at)
SELECT DISTINCT a.id, e.tenant_id, e.user_id, 'legacy', 'active', u.email_verified_at
FROM mobile_evidence e
JOIN app.applications a ON a.tenant_id = e.tenant_id AND a.is_default
JOIN iam.users u ON u.id = e.user_id
ON CONFLICT (app_id, user_id) DO NOTHING;

ALTER TABLE iam.sessions ADD COLUMN app_id uuid;
ALTER TABLE iam.sessions
    ADD CONSTRAINT fk_sessions_app_same_tenant
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE RESTRICT;
UPDATE iam.sessions s SET app_id = a.id
FROM app.applications a
WHERE s.audience = 'ak-mobile' AND a.tenant_id = s.tenant_id AND a.is_default;
CREATE INDEX idx_sessions_app_active ON iam.sessions (app_id, user_id, last_seen_at DESC)
WHERE audience = 'ak-mobile' AND revoked_at IS NULL;

ALTER TABLE content.categories ADD COLUMN app_id uuid;
UPDATE content.categories c SET app_id = a.id
FROM app.applications a WHERE a.tenant_id = c.tenant_id AND a.is_default;
ALTER TABLE content.categories ALTER COLUMN app_id SET NOT NULL;
ALTER TABLE content.categories ADD CONSTRAINT fk_content_categories_app_same_tenant
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE RESTRICT;
CREATE INDEX idx_content_categories_app ON content.categories (app_id, status, sort_order, id);
CREATE TRIGGER tr_content_categories_assign_default_app BEFORE INSERT ON content.categories
FOR EACH ROW EXECUTE FUNCTION app.assign_default_application();

ALTER TABLE content.articles ADD COLUMN app_id uuid;
UPDATE content.articles x SET app_id = a.id
FROM app.applications a WHERE a.tenant_id = x.tenant_id AND a.is_default;
ALTER TABLE content.articles ALTER COLUMN app_id SET NOT NULL;
ALTER TABLE content.articles ADD CONSTRAINT fk_content_articles_app_same_tenant
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE RESTRICT;
CREATE INDEX idx_content_articles_app ON content.articles (app_id, status, published_at DESC, id DESC);
CREATE TRIGGER tr_content_articles_assign_default_app BEFORE INSERT ON content.articles
FOR EACH ROW EXECUTE FUNCTION app.assign_default_application();

ALTER TABLE content.article_bookmarks ADD COLUMN app_id uuid;
UPDATE content.article_bookmarks x SET app_id = a.id
FROM app.applications a WHERE a.tenant_id = x.tenant_id AND a.is_default;
ALTER TABLE content.article_bookmarks ALTER COLUMN app_id SET NOT NULL;
ALTER TABLE content.article_bookmarks ADD CONSTRAINT fk_content_article_bookmarks_app_same_tenant
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE;
CREATE INDEX idx_content_article_bookmarks_app_user ON content.article_bookmarks (app_id, user_id, created_at DESC);
CREATE TRIGGER tr_content_article_bookmarks_assign_default_app BEFORE INSERT ON content.article_bookmarks
FOR EACH ROW EXECUTE FUNCTION app.assign_default_application();

-- Mobile profile, inbox and release policy state is App-owned.  Existing
-- tenant-scoped records are adopted by that tenant's default App, so legacy
-- endpoints retain deterministic behaviour while new App endpoints never
-- read across an App boundary.
ALTER TABLE iam.user_preferences DROP CONSTRAINT IF EXISTS user_preferences_pkey;
ALTER TABLE iam.user_preferences ADD COLUMN app_id uuid;
ALTER TABLE iam.user_preferences ADD COLUMN locale varchar(16);
INSERT INTO iam.user_preferences (user_id, app_id, appearance, notification_preferences, updated_at)
SELECT p.user_id, m.app_id, p.appearance, p.notification_preferences, p.updated_at
FROM iam.user_preferences p
JOIN app.user_memberships m ON m.user_id = p.user_id
WHERE p.app_id IS NULL
ON CONFLICT DO NOTHING;
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM iam.user_preferences p
        WHERE p.app_id IS NULL
          AND NOT EXISTS (SELECT 1 FROM app.user_memberships m WHERE m.user_id = p.user_id)
    ) THEN
        RAISE EXCEPTION 'cannot App-scope mobile preferences without an App membership';
    END IF;
END $$;
DELETE FROM iam.user_preferences WHERE app_id IS NULL;
UPDATE iam.user_preferences p SET locale = u.locale FROM iam.users u WHERE u.id = p.user_id;
ALTER TABLE iam.user_preferences ALTER COLUMN app_id SET NOT NULL;
ALTER TABLE iam.user_preferences ALTER COLUMN locale SET NOT NULL;
ALTER TABLE iam.user_preferences ADD CONSTRAINT ck_user_preferences_locale CHECK (locale IN ('zh-CN', 'en-US'));
ALTER TABLE iam.user_preferences ADD PRIMARY KEY (app_id, user_id);
ALTER TABLE iam.user_preferences ADD CONSTRAINT fk_user_preferences_app_member
    FOREIGN KEY (app_id, user_id) REFERENCES app.user_memberships(app_id, user_id) ON DELETE CASCADE;
CREATE INDEX idx_user_preferences_user_app ON iam.user_preferences (user_id, app_id);

ALTER TABLE notify.messages ADD COLUMN app_id uuid;
UPDATE notify.messages x SET app_id = a.id
FROM app.applications a WHERE a.tenant_id = x.tenant_id AND a.is_default;
ALTER TABLE notify.messages ALTER COLUMN app_id SET NOT NULL;
ALTER TABLE notify.messages ADD CONSTRAINT fk_notify_messages_app_same_tenant
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE;
CREATE INDEX idx_notify_messages_app_time ON notify.messages (app_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE TRIGGER tr_notify_messages_assign_default_app BEFORE INSERT ON notify.messages
FOR EACH ROW EXECUTE FUNCTION app.assign_default_application();

ALTER TABLE notify.recipients ADD COLUMN app_id uuid;
UPDATE notify.recipients r SET app_id = m.app_id
FROM notify.messages m WHERE m.tenant_id = r.tenant_id AND m.id = r.message_id;
ALTER TABLE notify.recipients ALTER COLUMN app_id SET NOT NULL;
ALTER TABLE notify.recipients ADD CONSTRAINT fk_notify_recipients_app_same_tenant
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE;
CREATE INDEX idx_notify_recipients_app_unread ON notify.recipients (app_id, user_id, created_at DESC) WHERE read_at IS NULL;

ALTER TABLE notify.push_devices ADD COLUMN app_id uuid;
UPDATE notify.push_devices d SET app_id = (
    SELECT m.app_id FROM app.user_memberships m
    JOIN app.applications a ON a.id = m.app_id AND a.tenant_id = m.tenant_id
    WHERE m.user_id = d.user_id AND m.status = 'active' AND a.is_default
    ORDER BY m.tenant_id, m.app_id LIMIT 1
)
WHERE d.app_id IS NULL;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM notify.push_devices WHERE app_id IS NULL) THEN
        RAISE EXCEPTION 'cannot App-scope push device without an App membership';
    END IF;
END $$;
ALTER TABLE notify.push_devices ALTER COLUMN app_id SET NOT NULL;
ALTER TABLE notify.push_devices ADD CONSTRAINT fk_push_devices_app_member
    FOREIGN KEY (app_id, user_id) REFERENCES app.user_memberships(app_id, user_id) ON DELETE CASCADE;
CREATE INDEX idx_push_devices_app_user ON notify.push_devices (app_id, user_id, status);

ALTER TABLE notify.deliveries ADD COLUMN app_id uuid;
ALTER TABLE notify.deliveries ADD CONSTRAINT fk_notify_deliveries_app FOREIGN KEY (app_id) REFERENCES app.applications(id) ON DELETE SET NULL;
CREATE INDEX idx_notify_deliveries_app_created ON notify.deliveries (app_id, created_at DESC) WHERE app_id IS NOT NULL;

-- A pre-App release was global.  Preserve that behaviour for every App rather
-- than silently assigning it to one arbitrary/default App.  Every replica has
-- a fresh identity so a subsequent App-local update cannot alias legacy data.
ALTER TABLE sys.mobile_releases ADD COLUMN app_id uuid;
DROP INDEX IF EXISTS sys.uq_mobile_releases_active_platform;
INSERT INTO sys.mobile_releases (
    id, app_id, platform, current_version, minimum_version, upgrade_url,
    release_notes, active, lock_version, created_at, updated_at
)
SELECT uuidv7(), a.id, r.platform, r.current_version, r.minimum_version,
       r.upgrade_url, r.release_notes, r.active, r.lock_version,
       r.created_at, r.updated_at
FROM sys.mobile_releases r
CROSS JOIN app.applications a
WHERE r.app_id IS NULL;
DELETE FROM sys.mobile_releases WHERE app_id IS NULL;
ALTER TABLE sys.mobile_releases ALTER COLUMN app_id SET NOT NULL;
ALTER TABLE sys.mobile_releases ADD CONSTRAINT fk_mobile_releases_app
    FOREIGN KEY (app_id) REFERENCES app.applications(id) ON DELETE CASCADE;
CREATE UNIQUE INDEX uq_mobile_releases_active_app_platform ON sys.mobile_releases (app_id, platform) WHERE active;
CREATE INDEX idx_mobile_releases_app_platform ON sys.mobile_releases (app_id, platform, updated_at DESC);

ALTER TABLE audit.login_events ADD COLUMN app_id uuid;
ALTER TABLE audit.security_events ADD COLUMN app_id uuid;
UPDATE audit.login_events e SET app_id = s.app_id
FROM iam.sessions s WHERE e.session_id = s.id AND s.audience = 'ak-mobile' AND s.app_id IS NOT NULL;
UPDATE audit.security_events e SET app_id = s.app_id
FROM iam.sessions s WHERE e.session_id = s.id AND s.audience = 'ak-mobile' AND s.app_id IS NOT NULL;
-- Mobile events created before sessions carried an App ID use the deterministic
-- tenant default App; admin and API events intentionally remain NULL.
UPDATE audit.login_events e SET app_id = (
    SELECT a.id FROM app.applications a WHERE a.tenant_id = e.tenant_id
    ORDER BY a.is_default DESC, a.created_at, a.id LIMIT 1
) WHERE e.app_id IS NULL AND e.audience = 'ak-mobile' AND e.tenant_id IS NOT NULL;
UPDATE audit.security_events e SET app_id = (
    SELECT a.id FROM app.applications a WHERE a.tenant_id = e.tenant_id
    ORDER BY a.is_default DESC, a.created_at, a.id LIMIT 1
) WHERE e.app_id IS NULL AND EXISTS (
    SELECT 1 FROM iam.sessions s WHERE s.id = e.session_id AND s.audience = 'ak-mobile'
);
ALTER TABLE audit.login_events ADD CONSTRAINT fk_login_events_app FOREIGN KEY (app_id) REFERENCES app.applications(id) ON DELETE SET NULL;
ALTER TABLE audit.security_events ADD CONSTRAINT fk_security_events_app FOREIGN KEY (app_id) REFERENCES app.applications(id) ON DELETE SET NULL;
CREATE INDEX idx_login_events_app_user_time ON audit.login_events (app_id, user_id, occurred_at DESC) WHERE app_id IS NOT NULL;
CREATE INDEX idx_security_events_app_user_time ON audit.security_events (app_id, user_id, occurred_at DESC) WHERE app_id IS NOT NULL;

CREATE TABLE content.pages (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    app_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    slug varchar(120) NOT NULL,
    page_type varchar(32) NOT NULL DEFAULT 'custom',
    status varchar(16) NOT NULL DEFAULT 'draft',
    current_revision_id uuid,
    lock_version integer NOT NULL DEFAULT 1,
    created_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_content_pages_app_slug UNIQUE (app_id, slug),
    CONSTRAINT uq_content_pages_tenant_id UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_content_pages_slug CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    CONSTRAINT ck_content_pages_type CHECK (page_type IN ('privacy-policy', 'terms-of-service', 'about-us', 'custom')),
    CONSTRAINT ck_content_pages_status CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT ck_content_pages_lock_version CHECK (lock_version > 0)
);
CREATE INDEX idx_content_pages_app_status ON content.pages (app_id, status, updated_at DESC);
CREATE TRIGGER tr_content_pages_touch_updated_at BEFORE UPDATE ON content.pages
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

CREATE TABLE content.page_revisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    page_id uuid NOT NULL REFERENCES content.pages(id) ON DELETE CASCADE,
    app_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    revision_number integer NOT NULL,
    content_hash bytea NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'draft',
    published_at timestamptz,
    created_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_content_page_revisions_number UNIQUE (page_id, revision_number),
    CONSTRAINT uq_content_page_revisions_id_app UNIQUE (id, app_id),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_content_page_revisions_status CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT ck_content_page_revisions_number CHECK (revision_number > 0),
    CONSTRAINT ck_content_page_revisions_hash CHECK (octet_length(content_hash) = 32)
);
CREATE INDEX idx_content_page_revisions_app_page ON content.page_revisions (app_id, page_id, revision_number DESC);
ALTER TABLE content.pages
    ADD CONSTRAINT fk_content_pages_current_revision
    FOREIGN KEY (current_revision_id, app_id) REFERENCES content.page_revisions(id, app_id) DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE content.page_revision_translations (
    revision_id uuid NOT NULL REFERENCES content.page_revisions(id) ON DELETE CASCADE,
    locale varchar(16) NOT NULL,
    title varchar(300) NOT NULL,
    body_format varchar(16) NOT NULL DEFAULT 'markdown',
    body jsonb NOT NULL,
    PRIMARY KEY (revision_id, locale),
    CONSTRAINT ck_content_page_revision_translation_locale CHECK (locale IN ('zh-CN', 'en-US')),
    CONSTRAINT ck_content_page_revision_translation_format CHECK (body_format IN ('markdown', 'blocks')),
    CONSTRAINT ck_content_page_revision_translation_body CHECK (jsonb_typeof(body) IN ('string', 'array'))
);

CREATE TABLE app.legal_consents (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    app_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    user_id uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    page_type varchar(32) NOT NULL,
    revision_id uuid NOT NULL,
    content_hash bytea NOT NULL,
    locale varchar(16) NOT NULL,
    accepted_at timestamptz NOT NULL DEFAULT now(),
    ip_address inet,
    user_agent varchar(1000),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (revision_id, app_id) REFERENCES content.page_revisions(id, app_id) ON DELETE RESTRICT,
    CONSTRAINT uq_legal_consents_version UNIQUE (app_id, user_id, page_type, revision_id),
    CONSTRAINT ck_legal_consents_type CHECK (page_type IN ('privacy-policy', 'terms-of-service')),
    CONSTRAINT ck_legal_consents_locale CHECK (locale IN ('zh-CN', 'en-US')),
    CONSTRAINT ck_legal_consents_hash CHECK (octet_length(content_hash) = 32)
);
CREATE INDEX idx_legal_consents_app_user ON app.legal_consents (app_id, user_id, accepted_at DESC);

CREATE OR REPLACE FUNCTION content.protect_reserved_page_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.page_type IN ('privacy-policy', 'terms-of-service', 'about-us') THEN
        RAISE EXCEPTION 'reserved core pages cannot be deleted';
    END IF;
    RETURN OLD;
END;
$$;
CREATE TRIGGER tr_content_pages_protect_reserved_delete BEFORE DELETE ON content.pages
FOR EACH ROW EXECUTE FUNCTION content.protect_reserved_page_delete();

-- Create the three reserved page identities for every existing App. Their
-- translations/revisions are authored and versioned through the Admin API.
INSERT INTO content.pages (app_id, tenant_id, slug, page_type)
SELECT a.id, a.tenant_id, v.slug, v.page_type
FROM app.applications a
CROSS JOIN (VALUES
    ('privacy-policy', 'privacy-policy'),
    ('terms-of-service', 'terms-of-service'),
    ('about-us', 'about-us')
) AS v(slug, page_type)
ON CONFLICT (app_id, slug) DO NOTHING;

CREATE OR REPLACE FUNCTION app.create_reserved_pages_for_application() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO content.pages (app_id, tenant_id, slug, page_type)
    VALUES
        (NEW.id, NEW.tenant_id, 'privacy-policy', 'privacy-policy'),
        (NEW.id, NEW.tenant_id, 'terms-of-service', 'terms-of-service'),
        (NEW.id, NEW.tenant_id, 'about-us', 'about-us')
    ON CONFLICT (app_id, slug) DO NOTHING;
    RETURN NEW;
END;
$$;
CREATE TRIGGER tr_applications_create_reserved_pages
AFTER INSERT ON app.applications FOR EACH ROW EXECUTE FUNCTION app.create_reserved_pages_for_application();

COMMIT;
