BEGIN;

CREATE TABLE iam.user_preferences (
    user_id uuid PRIMARY KEY REFERENCES iam.users(id) ON DELETE CASCADE,
    appearance varchar(16) NOT NULL DEFAULT 'system',
    notification_preferences jsonb NOT NULL DEFAULT '{"in_app":true,"push":false,"email":true}'::jsonb,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_user_preferences_appearance CHECK (appearance IN ('system', 'light', 'dark')),
    CONSTRAINT ck_user_preferences_notification_preferences CHECK (jsonb_typeof(notification_preferences) = 'object')
);

CREATE TRIGGER tr_iam_user_preferences_touch_updated_at
BEFORE UPDATE ON iam.user_preferences FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

CREATE TABLE sys.mobile_releases (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    platform varchar(16) NOT NULL,
    current_version varchar(64) NOT NULL,
    minimum_version varchar(64) NOT NULL,
    upgrade_url varchar(2000),
    release_notes jsonb NOT NULL,
    active boolean NOT NULL DEFAULT false,
    lock_version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_mobile_release_platform CHECK (platform IN ('android', 'ios', 'harmony')),
    CONSTRAINT ck_mobile_release_versions CHECK (
        current_version ~ '^[0-9]+\.[0-9]+\.[0-9]+$'
        AND minimum_version ~ '^[0-9]+\.[0-9]+\.[0-9]+$'
    ),
    CONSTRAINT ck_mobile_release_upgrade_url CHECK (
        upgrade_url IS NULL OR upgrade_url ~ '^https://[^[:space:]]+$'
    ),
    CONSTRAINT ck_mobile_release_active_upgrade_url CHECK (
        NOT active OR upgrade_url IS NOT NULL
    ),
    CONSTRAINT ck_mobile_release_notes CHECK (
        jsonb_typeof(release_notes) = 'object'
        AND jsonb_typeof(release_notes -> 'zh-CN') = 'string'
        AND jsonb_typeof(release_notes -> 'en-US') = 'string'
        AND btrim(release_notes ->> 'zh-CN') <> ''
        AND btrim(release_notes ->> 'en-US') <> ''
    ),
    CONSTRAINT ck_mobile_release_lock_version CHECK (lock_version > 0)
);
CREATE UNIQUE INDEX uq_mobile_releases_active_platform ON sys.mobile_releases (platform) WHERE active;
CREATE TRIGGER tr_sys_mobile_releases_touch_updated_at
BEFORE UPDATE ON sys.mobile_releases FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMIT;
