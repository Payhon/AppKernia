BEGIN;

DROP TABLE IF EXISTS content.sensitive_words;
DROP TABLE IF EXISTS content.video_external_hosts;
DROP TABLE IF EXISTS content.blocked_users;
DROP TABLE IF EXISTS content.comment_reports;
DROP TABLE IF EXISTS content.comments;
DROP TABLE IF EXISTS content.article_media_translations;
DROP TABLE IF EXISTS content.article_media;
DROP TABLE IF EXISTS content.article_tags;
DROP TABLE IF EXISTS content.tags;
DROP TABLE IF EXISTS content.article_categories;

ALTER TABLE content.article_translations DROP COLUMN IF EXISTS search_text;
ALTER TABLE content.article_translations ALTER COLUMN summary TYPE varchar(1000) USING left(summary, 1000);
ALTER TABLE content.article_translations
    DROP CONSTRAINT ck_content_article_translation_body,
    ADD CONSTRAINT ck_content_article_translation_body CHECK (jsonb_typeof(body) IN ('string', 'array'));

ALTER TABLE content.articles
    DROP CONSTRAINT IF EXISTS ck_content_articles_video_duration,
    DROP CONSTRAINT IF EXISTS ck_content_articles_video_source,
    DROP CONSTRAINT IF EXISTS ck_content_articles_type,
    DROP CONSTRAINT IF EXISTS fk_content_articles_topic_same_app,
    DROP CONSTRAINT IF EXISTS uq_content_articles_tenant_app_id,
    DROP COLUMN IF EXISTS video_duration_seconds,
    DROP COLUMN IF EXISTS video_external_url,
    DROP COLUMN IF EXISTS video_file_id,
    DROP COLUMN IF EXISTS video_source_type,
    DROP COLUMN IF EXISTS is_latest,
    DROP COLUMN IF EXISTS pinned,
    DROP COLUMN IF EXISTS allow_comments,
    DROP COLUMN IF EXISTS topic_id,
    DROP COLUMN IF EXISTS content_type;

DROP TABLE IF EXISTS content.topic_translations;
DROP TABLE IF EXISTS content.topics;

ALTER TABLE content.categories
    DROP CONSTRAINT IF EXISTS ck_content_categories_not_self_parent,
    DROP CONSTRAINT IF EXISTS fk_content_categories_parent_same_app,
    DROP CONSTRAINT IF EXISTS uq_content_categories_tenant_app_id,
    DROP COLUMN IF EXISTS image_file_id,
    DROP COLUMN IF EXISTS parent_id;

COMMIT;
