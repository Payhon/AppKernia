BEGIN;

DROP TRIGGER IF EXISTS tr_content_pages_protect_reserved_delete ON content.pages;
DROP FUNCTION IF EXISTS content.protect_reserved_page_delete();
DROP TRIGGER IF EXISTS tr_applications_create_reserved_pages ON app.applications;
DROP FUNCTION IF EXISTS app.create_reserved_pages_for_application();
DROP TABLE IF EXISTS app.legal_consents;
ALTER TABLE content.pages DROP CONSTRAINT IF EXISTS fk_content_pages_current_revision;
DROP TABLE IF EXISTS content.page_revision_translations;
DROP TABLE IF EXISTS content.page_revisions;
DROP TRIGGER IF EXISTS tr_content_pages_touch_updated_at ON content.pages;
DROP TABLE IF EXISTS content.pages;
DROP INDEX IF EXISTS sys.idx_mobile_releases_app_platform;
DROP INDEX IF EXISTS sys.uq_mobile_releases_active_app_platform;
-- Collapse App-local replicas deterministically: default App first, then the
-- most recently updated row, then the stable replica UUID.  This restores the
-- historical one-policy-per-platform invariant before app_id is removed.
WITH ranked AS (
    SELECT r.id, row_number() OVER (
        PARTITION BY r.platform
        ORDER BY a.is_default DESC, r.updated_at DESC, r.id
    ) AS rn
    FROM sys.mobile_releases r
    JOIN app.applications a ON a.id = r.app_id
)
DELETE FROM sys.mobile_releases r USING ranked x WHERE r.id = x.id AND x.rn > 1;
ALTER TABLE sys.mobile_releases DROP CONSTRAINT IF EXISTS fk_mobile_releases_app;
ALTER TABLE sys.mobile_releases DROP COLUMN IF EXISTS app_id;
CREATE UNIQUE INDEX IF NOT EXISTS uq_mobile_releases_active_platform ON sys.mobile_releases (platform) WHERE active;
DROP INDEX IF EXISTS audit.idx_security_events_app_user_time;
DROP INDEX IF EXISTS audit.idx_login_events_app_user_time;
ALTER TABLE audit.security_events DROP CONSTRAINT IF EXISTS fk_security_events_app;
ALTER TABLE audit.login_events DROP CONSTRAINT IF EXISTS fk_login_events_app;
ALTER TABLE audit.security_events DROP COLUMN IF EXISTS app_id;
ALTER TABLE audit.login_events DROP COLUMN IF EXISTS app_id;
DROP INDEX IF EXISTS notify.idx_push_devices_app_user;
DROP INDEX IF EXISTS notify.idx_notify_deliveries_app_created;
ALTER TABLE notify.deliveries DROP CONSTRAINT IF EXISTS fk_notify_deliveries_app;
ALTER TABLE notify.deliveries DROP COLUMN IF EXISTS app_id;
ALTER TABLE notify.push_devices DROP CONSTRAINT IF EXISTS fk_push_devices_app_member;
ALTER TABLE notify.push_devices DROP COLUMN IF EXISTS app_id;
DROP INDEX IF EXISTS notify.idx_notify_recipients_app_unread;
ALTER TABLE notify.recipients DROP CONSTRAINT IF EXISTS fk_notify_recipients_app_same_tenant;
ALTER TABLE notify.recipients DROP COLUMN IF EXISTS app_id;
DROP TRIGGER IF EXISTS tr_notify_messages_assign_default_app ON notify.messages;
DROP INDEX IF EXISTS notify.idx_notify_messages_app_time;
ALTER TABLE notify.messages DROP CONSTRAINT IF EXISTS fk_notify_messages_app_same_tenant;
ALTER TABLE notify.messages DROP COLUMN IF EXISTS app_id;
ALTER TABLE iam.user_preferences DROP CONSTRAINT IF EXISTS fk_user_preferences_app_member;
ALTER TABLE iam.user_preferences DROP CONSTRAINT IF EXISTS ck_user_preferences_locale;
-- Collapse App-local preferences deterministically before restoring the
-- historical user_id primary key: prefer the tenant default App, then newest
-- preference, then the stable App UUID. Preserve the selected locale on the
-- legacy user profile as well.
WITH ranked AS (
    SELECT p.user_id, p.app_id, p.locale,
           row_number() OVER (PARTITION BY p.user_id ORDER BY a.is_default DESC, p.updated_at DESC, p.app_id) AS rn
    FROM iam.user_preferences p
    JOIN app.applications a ON a.id = p.app_id
), chosen AS (
    SELECT user_id, locale FROM ranked WHERE rn = 1
)
UPDATE iam.users u SET locale = chosen.locale FROM chosen WHERE u.id = chosen.user_id;
WITH ranked AS (
    SELECT p.user_id, p.app_id,
           row_number() OVER (PARTITION BY p.user_id ORDER BY a.is_default DESC, p.updated_at DESC, p.app_id) AS rn
    FROM iam.user_preferences p
    JOIN app.applications a ON a.id = p.app_id
)
DELETE FROM iam.user_preferences p USING ranked r WHERE p.user_id = r.user_id AND p.app_id = r.app_id AND r.rn > 1;
ALTER TABLE iam.user_preferences DROP CONSTRAINT IF EXISTS user_preferences_pkey;
ALTER TABLE iam.user_preferences DROP COLUMN IF EXISTS locale;
ALTER TABLE iam.user_preferences DROP COLUMN IF EXISTS app_id;
ALTER TABLE iam.user_preferences ADD PRIMARY KEY (user_id);
DROP TRIGGER IF EXISTS tr_content_article_bookmarks_assign_default_app ON content.article_bookmarks;
ALTER TABLE content.article_bookmarks DROP CONSTRAINT IF EXISTS fk_content_article_bookmarks_app_same_tenant;
ALTER TABLE content.article_bookmarks DROP COLUMN IF EXISTS app_id;
DROP TRIGGER IF EXISTS tr_content_articles_assign_default_app ON content.articles;
ALTER TABLE content.articles DROP CONSTRAINT IF EXISTS fk_content_articles_app_same_tenant;
ALTER TABLE content.articles DROP COLUMN IF EXISTS app_id;
DROP TRIGGER IF EXISTS tr_content_categories_assign_default_app ON content.categories;
ALTER TABLE content.categories DROP CONSTRAINT IF EXISTS fk_content_categories_app_same_tenant;
ALTER TABLE content.categories DROP COLUMN IF EXISTS app_id;
DROP INDEX IF EXISTS iam.idx_sessions_app_active;
ALTER TABLE iam.sessions DROP CONSTRAINT IF EXISTS fk_sessions_app_same_tenant;
ALTER TABLE iam.sessions DROP COLUMN IF EXISTS app_id;
DROP TRIGGER IF EXISTS tr_app_user_memberships_touch_updated_at ON app.user_memberships;
DROP TABLE IF EXISTS app.user_memberships;
DROP TRIGGER IF EXISTS tr_iam_tenants_create_default_application ON iam.tenants;
DROP FUNCTION IF EXISTS app.create_default_application_for_tenant();
DROP FUNCTION IF EXISTS app.assign_default_application();
DROP TRIGGER IF EXISTS tr_applications_touch_updated_at ON app.applications;
DROP TABLE IF EXISTS app.applications;
DROP SCHEMA IF EXISTS app;

COMMIT;
