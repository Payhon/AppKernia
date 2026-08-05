BEGIN;

CREATE SCHEMA IF NOT EXISTS content;

CREATE TABLE content.categories (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    slug varchar(120) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'active',
    sort_order integer NOT NULL DEFAULT 0,
    lock_version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_content_category_tenant_slug UNIQUE (tenant_id, slug),
    CONSTRAINT uq_content_category_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT ck_content_category_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_content_category_lock_version CHECK (lock_version > 0),
    CONSTRAINT ck_content_category_slug CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$')
);

CREATE TABLE content.category_translations (
    category_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    locale varchar(16) NOT NULL,
    name varchar(160) NOT NULL,
    description varchar(500) NOT NULL DEFAULT '',
    PRIMARY KEY (category_id, locale),
    FOREIGN KEY (tenant_id, category_id) REFERENCES content.categories(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_content_category_translation_locale CHECK (locale IN ('zh-CN', 'en-US'))
);

CREATE TABLE content.articles (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    category_id uuid,
    slug varchar(160) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'draft',
    featured boolean NOT NULL DEFAULT false,
    sort_order integer NOT NULL DEFAULT 0,
    cover_file_id uuid REFERENCES storage.files(id) ON DELETE SET NULL,
    reading_minutes smallint NOT NULL DEFAULT 1,
    lock_version integer NOT NULL DEFAULT 1,
    published_at timestamptz,
    created_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_content_article_tenant_slug UNIQUE (tenant_id, slug),
    CONSTRAINT uq_content_article_tenant_id UNIQUE (tenant_id, id),
    CONSTRAINT ck_content_article_status CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT ck_content_article_slug CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    CONSTRAINT ck_content_article_reading_minutes CHECK (reading_minutes BETWEEN 1 AND 120),
    CONSTRAINT ck_content_article_lock_version CHECK (lock_version > 0),
    CONSTRAINT fk_content_article_category FOREIGN KEY (tenant_id, category_id) REFERENCES content.categories(tenant_id, id) ON DELETE SET NULL (category_id),
    CONSTRAINT ck_content_article_published_at CHECK ((status <> 'published') OR published_at IS NOT NULL)
);

CREATE TABLE content.article_translations (
    article_id uuid NOT NULL REFERENCES content.articles(id) ON DELETE CASCADE,
    locale varchar(16) NOT NULL,
    title varchar(300) NOT NULL,
    summary varchar(1000) NOT NULL DEFAULT '',
    body_format varchar(16) NOT NULL DEFAULT 'markdown',
    body jsonb NOT NULL,
    PRIMARY KEY (article_id, locale),
    CONSTRAINT ck_content_article_translation_locale CHECK (locale IN ('zh-CN', 'en-US')),
    CONSTRAINT ck_content_article_translation_format CHECK (body_format IN ('markdown', 'blocks')),
    CONSTRAINT ck_content_article_translation_body CHECK (jsonb_typeof(body) IN ('string', 'array'))
);

CREATE TABLE content.article_bookmarks (
    tenant_id uuid NOT NULL,
    user_id uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    article_id uuid NOT NULL REFERENCES content.articles(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id, article_id),
    FOREIGN KEY (tenant_id, article_id) REFERENCES content.articles(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_content_articles_public ON content.articles (tenant_id, featured DESC, sort_order DESC, published_at DESC, id DESC) WHERE status = 'published';
CREATE INDEX idx_content_articles_category_public ON content.articles (tenant_id, category_id, sort_order DESC, published_at DESC, id DESC) WHERE status = 'published';
CREATE INDEX idx_content_article_translations_title_trgm ON content.article_translations USING gin (title gin_trgm_ops);
CREATE TRIGGER tr_content_categories_touch_updated_at BEFORE UPDATE ON content.categories FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_content_articles_touch_updated_at BEFORE UPDATE ON content.articles FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMIT;
