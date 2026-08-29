BEGIN;

ALTER TABLE content.categories
    ADD COLUMN parent_id uuid,
    ADD COLUMN image_file_id uuid REFERENCES storage.files(id) ON DELETE SET NULL;

ALTER TABLE content.categories
    ADD CONSTRAINT uq_content_categories_tenant_app_id UNIQUE (tenant_id, app_id, id),
    ADD CONSTRAINT fk_content_categories_parent_same_app
        FOREIGN KEY (tenant_id, app_id, parent_id)
        REFERENCES content.categories(tenant_id, app_id, id) ON DELETE RESTRICT,
    ADD CONSTRAINT ck_content_categories_not_self_parent CHECK (parent_id IS NULL OR parent_id <> id);

CREATE TABLE content.topics (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    app_id uuid NOT NULL,
    slug varchar(160) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'active',
    sort_order integer NOT NULL DEFAULT 0,
    cover_file_id uuid REFERENCES storage.files(id) ON DELETE SET NULL,
    lock_version integer NOT NULL DEFAULT 1,
    created_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_content_topics_app_same_tenant
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_content_topics_app_slug UNIQUE (app_id, slug),
    CONSTRAINT uq_content_topics_tenant_app_id UNIQUE (tenant_id, app_id, id),
    CONSTRAINT ck_content_topics_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_content_topics_slug CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    CONSTRAINT ck_content_topics_sort_order CHECK (sort_order >= 0),
    CONSTRAINT ck_content_topics_lock_version CHECK (lock_version > 0)
);

CREATE TABLE content.topic_translations (
    topic_id uuid NOT NULL REFERENCES content.topics(id) ON DELETE CASCADE,
    locale varchar(16) NOT NULL,
    name varchar(160) NOT NULL,
    description varchar(2000) NOT NULL DEFAULT '',
    PRIMARY KEY (topic_id, locale),
    CONSTRAINT ck_content_topic_translation_locale CHECK (locale IN ('zh-CN', 'en-US'))
);

ALTER TABLE content.articles
    ADD COLUMN content_type varchar(16) NOT NULL DEFAULT 'article',
    ADD COLUMN topic_id uuid,
    ADD COLUMN allow_comments boolean NOT NULL DEFAULT true,
    ADD COLUMN pinned boolean NOT NULL DEFAULT false,
    ADD COLUMN is_latest boolean NOT NULL DEFAULT false,
    ADD COLUMN video_source_type varchar(16),
    ADD COLUMN video_file_id uuid REFERENCES storage.files(id) ON DELETE SET NULL,
    ADD COLUMN video_external_url varchar(2048),
    ADD COLUMN video_duration_seconds integer,
    ADD CONSTRAINT uq_content_articles_tenant_app_id UNIQUE (tenant_id, app_id, id),
    ADD CONSTRAINT fk_content_articles_topic_same_app
        FOREIGN KEY (tenant_id, app_id, topic_id)
        REFERENCES content.topics(tenant_id, app_id, id) ON DELETE SET NULL,
    ADD CONSTRAINT ck_content_articles_type CHECK (content_type IN ('article', 'gallery', 'video')),
    ADD CONSTRAINT ck_content_articles_video_source CHECK (
        (content_type <> 'video' AND video_source_type IS NULL AND video_file_id IS NULL AND video_external_url IS NULL AND video_duration_seconds IS NULL)
        OR
        (content_type = 'video' AND video_source_type = 'upload' AND video_file_id IS NOT NULL AND video_external_url IS NULL)
        OR
        (content_type = 'video' AND video_source_type = 'external' AND video_file_id IS NULL AND video_external_url IS NOT NULL)
    ),
    ADD CONSTRAINT ck_content_articles_video_duration CHECK (video_duration_seconds IS NULL OR video_duration_seconds BETWEEN 1 AND 86400);

ALTER TABLE content.article_translations
    ALTER COLUMN summary TYPE varchar(3000),
    ADD COLUMN search_text text NOT NULL DEFAULT '';

-- Information articles use a controlled Tiptap document object while legacy
-- block payloads remain arrays. Markdown continues to be stored as a JSON
-- string. Keep the database constraint aligned with the application validator.
ALTER TABLE content.article_translations
    DROP CONSTRAINT ck_content_article_translation_body,
    ADD CONSTRAINT ck_content_article_translation_body CHECK (
        (body_format = 'blocks' AND jsonb_typeof(body) IN ('object', 'array'))
        OR
        (body_format = 'markdown' AND jsonb_typeof(body) = 'string')
    );

UPDATE content.article_translations
SET search_text = lower(concat_ws(' ', title, summary, body::text));

CREATE TABLE content.article_categories (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    article_id uuid NOT NULL,
    category_id uuid NOT NULL,
    sort_order smallint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (article_id, category_id),
    FOREIGN KEY (tenant_id, app_id, article_id)
        REFERENCES content.articles(tenant_id, app_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, app_id, category_id)
        REFERENCES content.categories(tenant_id, app_id, id) ON DELETE RESTRICT,
    CONSTRAINT ck_content_article_categories_sort CHECK (sort_order BETWEEN 0 AND 9)
);

INSERT INTO content.article_categories (tenant_id, app_id, article_id, category_id)
SELECT tenant_id, app_id, id, category_id
FROM content.articles
WHERE category_id IS NOT NULL
ON CONFLICT DO NOTHING;

CREATE TABLE content.tags (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    app_id uuid NOT NULL,
    name varchar(80) NOT NULL,
    normalized_name varchar(80) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'active',
    lock_version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_content_tags_app_same_tenant
        FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_content_tags_app_normalized UNIQUE (app_id, normalized_name),
    CONSTRAINT uq_content_tags_tenant_app_id UNIQUE (tenant_id, app_id, id),
    CONSTRAINT ck_content_tags_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_content_tags_name CHECK (name = btrim(name) AND name <> '' AND name !~ '^#'),
    CONSTRAINT ck_content_tags_normalized CHECK (normalized_name = lower(normalized_name) AND normalized_name = btrim(normalized_name) AND normalized_name <> ''),
    CONSTRAINT ck_content_tags_lock_version CHECK (lock_version > 0)
);

CREATE TABLE content.article_tags (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    article_id uuid NOT NULL,
    tag_id uuid NOT NULL,
    sort_order smallint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (article_id, tag_id),
    FOREIGN KEY (tenant_id, app_id, article_id)
        REFERENCES content.articles(tenant_id, app_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, app_id, tag_id)
        REFERENCES content.tags(tenant_id, app_id, id) ON DELETE RESTRICT,
    CONSTRAINT ck_content_article_tags_sort CHECK (sort_order BETWEEN 0 AND 9)
);

CREATE TABLE content.article_media (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    article_id uuid NOT NULL,
    file_id uuid NOT NULL,
    role varchar(16) NOT NULL,
    sort_order smallint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, app_id, article_id)
        REFERENCES content.articles(tenant_id, app_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, file_id)
        REFERENCES storage.files(tenant_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_content_article_media_position UNIQUE (article_id, role, sort_order),
    CONSTRAINT ck_content_article_media_role CHECK (role IN ('gallery', 'inline')),
    CONSTRAINT ck_content_article_media_sort CHECK (sort_order BETWEEN 0 AND 99)
);

CREATE TABLE content.article_media_translations (
    media_id uuid NOT NULL REFERENCES content.article_media(id) ON DELETE CASCADE,
    locale varchar(16) NOT NULL,
    alt_text varchar(500) NOT NULL,
    PRIMARY KEY (media_id, locale),
    CONSTRAINT ck_content_article_media_translation_locale CHECK (locale IN ('zh-CN', 'en-US'))
);

CREATE TABLE content.comments (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    article_id uuid NOT NULL,
    author_id uuid NOT NULL REFERENCES iam.users(id) ON DELETE RESTRICT,
    parent_id uuid REFERENCES content.comments(id) ON DELETE RESTRICT,
    root_id uuid REFERENCES content.comments(id) ON DELETE RESTRICT,
    status varchar(16) NOT NULL DEFAULT 'pending',
    body varchar(500) NOT NULL,
    body_fingerprint bytea NOT NULL,
    moderation_reason varchar(500),
    moderated_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    moderated_at timestamptz,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, app_id, article_id)
        REFERENCES content.articles(tenant_id, app_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_content_comments_tenant_app_id UNIQUE (tenant_id, app_id, id),
    CONSTRAINT ck_content_comments_status CHECK (status IN ('pending', 'approved', 'rejected', 'hidden', 'deleted')),
    CONSTRAINT ck_content_comments_body CHECK (body = btrim(body) AND body <> ''),
    CONSTRAINT ck_content_comments_fingerprint CHECK (octet_length(body_fingerprint) = 32),
    CONSTRAINT ck_content_comments_moderation CHECK (
        (status = 'pending' AND moderated_at IS NULL)
        OR (status <> 'pending' AND moderated_at IS NOT NULL)
    )
);

CREATE TABLE content.comment_reports (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    comment_id uuid NOT NULL,
    reporter_id uuid NOT NULL REFERENCES iam.users(id) ON DELETE RESTRICT,
    reason varchar(24) NOT NULL,
    details varchar(500) NOT NULL DEFAULT '',
    status varchar(16) NOT NULL DEFAULT 'open',
    resolution varchar(500) NOT NULL DEFAULT '',
    resolved_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    resolved_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, app_id, comment_id)
        REFERENCES content.comments(tenant_id, app_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_content_comment_reporter UNIQUE (comment_id, reporter_id),
    CONSTRAINT ck_content_comment_reports_reason CHECK (reason IN ('spam', 'abuse', 'illegal', 'privacy', 'other')),
    CONSTRAINT ck_content_comment_reports_status CHECK (status IN ('open', 'resolved', 'dismissed')),
    CONSTRAINT ck_content_comment_reports_resolution CHECK ((status = 'open' AND resolved_at IS NULL) OR (status <> 'open' AND resolved_at IS NOT NULL))
);

CREATE TABLE content.blocked_users (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    blocker_id uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    blocked_id uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (app_id, blocker_id, blocked_id),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_content_blocked_users_not_self CHECK (blocker_id <> blocked_id)
);

CREATE TABLE content.video_external_hosts (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    hostname varchar(253) NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (app_id, hostname),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_content_video_external_host CHECK (hostname = lower(btrim(hostname)) AND hostname <> '' AND hostname !~ '[/@:?#]')
);

CREATE TABLE content.sensitive_words (
    tenant_id uuid NOT NULL,
    app_id uuid NOT NULL,
    word varchar(120) NOT NULL,
    normalized_word varchar(120) NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (app_id, normalized_word),
    FOREIGN KEY (tenant_id, app_id) REFERENCES app.applications(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_content_sensitive_word CHECK (word = btrim(word) AND word <> '' AND normalized_word = lower(btrim(normalized_word)))
);

CREATE INDEX idx_content_categories_parent ON content.categories (app_id, parent_id, sort_order, id) WHERE status = 'active';
CREATE INDEX idx_content_topics_public ON content.topics (app_id, sort_order, updated_at DESC, id) WHERE status = 'active';
CREATE INDEX idx_content_articles_home ON content.articles (app_id, pinned DESC, featured DESC, is_latest DESC, sort_order DESC, published_at DESC, id DESC) WHERE status = 'published';
CREATE INDEX idx_content_articles_type_time ON content.articles (app_id, content_type, published_at DESC, id DESC) WHERE status = 'published';
CREATE INDEX idx_content_article_translations_search_trgm ON content.article_translations USING gin (search_text gin_trgm_ops);
CREATE INDEX idx_content_article_categories_category ON content.article_categories (app_id, category_id, article_id);
CREATE INDEX idx_content_article_tags_tag ON content.article_tags (app_id, tag_id, article_id);
CREATE INDEX idx_content_comments_public ON content.comments (app_id, article_id, created_at DESC, id DESC) WHERE status = 'approved';
CREATE INDEX idx_content_comments_moderation ON content.comments (app_id, status, created_at, id);
CREATE INDEX idx_content_comment_reports_queue ON content.comment_reports (app_id, status, created_at, id);

CREATE TRIGGER tr_content_topics_touch_updated_at BEFORE UPDATE ON content.topics FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_content_tags_touch_updated_at BEFORE UPDATE ON content.tags FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_content_comments_touch_updated_at BEFORE UPDATE ON content.comments FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMIT;
