BEGIN;
ALTER TABLE content.pages DROP CONSTRAINT ck_content_pages_type;
ALTER TABLE content.pages ADD CONSTRAINT ck_content_pages_type CHECK (page_type IN ('privacy-policy','terms-of-service','about-us','faq','contact-support','custom'));
-- Preserve existing content and revisions for the newly reserved slugs.
UPDATE content.pages SET page_type=slug WHERE slug IN ('faq','contact-support');
ALTER TABLE storage.upload_sessions ADD COLUMN purpose varchar(32) NOT NULL DEFAULT 'general';
ALTER TABLE storage.upload_sessions ADD CONSTRAINT ck_upload_purpose CHECK (purpose IN ('general','feedback'));
-- Private feedback images cannot deduplicate across owners or business scopes.
DROP INDEX storage.uq_files_tenant_dedup;
CREATE UNIQUE INDEX uq_files_tenant_dedup_shared ON storage.files(tenant_id,sha256,size_bytes)
 WHERE sha256 IS NOT NULL AND status='ready' AND deleted_at IS NULL AND metadata->>'purpose' IS DISTINCT FROM 'feedback';
CREATE TABLE app.feedbacks (
 id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL, app_id uuid NOT NULL,
 user_id uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
 description text NOT NULL CHECK (char_length(description) BETWEEN 1 AND 2000),
 contact varchar(200) NOT NULL DEFAULT '', platform varchar(20) NOT NULL CHECK(platform IN ('android','ios','harmony','unknown')),
 app_version varchar(64) NOT NULL DEFAULT '', status varchar(20) NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','processing','resolved')),
 idempotency_key uuid NOT NULL, request_hash bytea NOT NULL CHECK(octet_length(request_hash)=32),
 lock_version integer NOT NULL DEFAULT 1 CHECK(lock_version>0),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(tenant_id,app_id,id), UNIQUE(tenant_id,app_id,user_id,idempotency_key),
 FOREIGN KEY(tenant_id,app_id) REFERENCES app.applications(tenant_id,id) ON DELETE CASCADE
);
CREATE TRIGGER tr_feedbacks_touch_updated_at BEFORE UPDATE ON app.feedbacks FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE INDEX idx_feedbacks_app_status_time ON app.feedbacks(tenant_id,app_id,status,created_at DESC,id DESC);
CREATE INDEX idx_feedbacks_owner_time ON app.feedbacks(tenant_id,app_id,user_id,created_at DESC,id DESC);
CREATE TABLE app.feedback_attachments (
 tenant_id uuid NOT NULL, app_id uuid NOT NULL, feedback_id uuid NOT NULL, file_id uuid NOT NULL,
 position integer NOT NULL CHECK(position BETWEEN 0 AND 2), PRIMARY KEY(feedback_id,file_id), UNIQUE(feedback_id,position),
 FOREIGN KEY(tenant_id,app_id,feedback_id) REFERENCES app.feedbacks(tenant_id,app_id,id) ON DELETE CASCADE,
 FOREIGN KEY(tenant_id,file_id) REFERENCES storage.files(tenant_id,id) ON DELETE RESTRICT
);
CREATE TABLE app.feedback_replies (
 id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL, app_id uuid NOT NULL, feedback_id uuid NOT NULL,
 author_id uuid REFERENCES iam.users(id) ON DELETE SET NULL,
 body text NOT NULL CHECK(char_length(body) BETWEEN 1 AND 2000),
 idempotency_key uuid NOT NULL, request_hash bytea NOT NULL CHECK(octet_length(request_hash)=32),
 created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(feedback_id,idempotency_key),
 FOREIGN KEY(tenant_id,app_id,feedback_id) REFERENCES app.feedbacks(tenant_id,app_id,id) ON DELETE CASCADE
);
CREATE TABLE app.feedback_events (
 id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL, app_id uuid NOT NULL, feedback_id uuid NOT NULL,
 actor_id uuid REFERENCES iam.users(id) ON DELETE SET NULL,
 from_status varchar(20) NOT NULL CHECK(from_status IN ('pending','processing','resolved')),
 to_status varchar(20) NOT NULL CHECK(to_status IN ('pending','processing','resolved')),
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(tenant_id,app_id,feedback_id) REFERENCES app.feedbacks(tenant_id,app_id,id) ON DELETE CASCADE
);
CREATE OR REPLACE FUNCTION content.protect_reserved_page_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.page_type IN ('privacy-policy', 'terms-of-service', 'about-us', 'faq', 'contact-support') THEN
        RAISE EXCEPTION 'reserved core pages cannot be deleted';
    END IF;
    RETURN OLD;
END;
$$;


-- Create the five reserved page identities for every existing App. Their
-- translations/revisions are authored and versioned through the Admin API.
INSERT INTO content.pages (app_id, tenant_id, slug, page_type)
SELECT a.id, a.tenant_id, v.slug, v.page_type
FROM app.applications a
CROSS JOIN (VALUES
    ('privacy-policy', 'privacy-policy'),
    ('terms-of-service', 'terms-of-service'),
    ('about-us', 'about-us'), ('faq', 'faq'), ('contact-support', 'contact-support')
) AS v(slug, page_type)
ON CONFLICT (app_id, slug) DO NOTHING;

CREATE OR REPLACE FUNCTION app.create_reserved_pages_for_application() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO content.pages (app_id, tenant_id, slug, page_type)
    VALUES
        (NEW.id, NEW.tenant_id, 'privacy-policy', 'privacy-policy'),
        (NEW.id, NEW.tenant_id, 'terms-of-service', 'terms-of-service'),
        (NEW.id, NEW.tenant_id, 'about-us', 'about-us'),
        (NEW.id, NEW.tenant_id, 'faq', 'faq'), (NEW.id, NEW.tenant_id, 'contact-support', 'contact-support')
    ON CONFLICT (app_id, slug) DO NOTHING;
    RETURN NEW;
END;
$$;



INSERT INTO iam.permissions(code,name,module_code,resource_name,action_name,permission_kind,http_methods,route_pattern,description,status)
VALUES
 ('app.feedback.read','查看问题反馈','app','feedback','read','api',ARRAY['GET'],'/admin-api/v1/apps/{app_id}/feedbacks','Read App feedback','active'),
 ('app.feedback.update','处理问题反馈','app','feedback','update','api',ARRAY['PATCH'],'/admin-api/v1/apps/{app_id}/feedbacks/{id}','Update App feedback status','active'),
 ('app.feedback.reply','回复问题反馈','app','feedback','reply','api',ARRAY['POST'],'/admin-api/v1/apps/{app_id}/feedbacks/{id}/replies','Reply to App feedback','active')
ON CONFLICT(code) DO NOTHING;
INSERT INTO iam.role_permissions(tenant_id,role_id,permission_id)
SELECT r.tenant_id,r.id,p.id FROM iam.roles r CROSS JOIN iam.permissions p
WHERE r.code IN ('super_admin','tenant_admin') AND p.code LIKE 'app.feedback.%'
ON CONFLICT DO NOTHING;
COMMIT;
