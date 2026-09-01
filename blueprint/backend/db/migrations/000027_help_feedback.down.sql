BEGIN;
DELETE FROM storage.file_usages WHERE module_code='app' AND entity_type='app.feedback';
DROP TABLE IF EXISTS app.feedback_events;
DROP TABLE IF EXISTS app.feedback_replies;
DROP TABLE IF EXISTS app.feedback_attachments;
DROP TABLE IF EXISTS app.feedbacks;
DELETE FROM iam.role_permissions WHERE permission_id IN(SELECT id FROM iam.permissions WHERE code LIKE 'app.feedback.%');
DELETE FROM iam.permissions WHERE code LIKE 'app.feedback.%';
ALTER TABLE storage.upload_sessions DROP COLUMN purpose;
-- Preserve newly authored help pages as custom pages on rollback.
UPDATE content.pages SET page_type='custom' WHERE page_type IN ('faq','contact-support');
ALTER TABLE content.pages DROP CONSTRAINT ck_content_pages_type;
ALTER TABLE content.pages ADD CONSTRAINT ck_content_pages_type CHECK(page_type IN ('privacy-policy','terms-of-service','about-us','custom'));
-- Preserve feedback objects and their checksum metadata without violating legacy deduplication.
UPDATE storage.files SET metadata=metadata || jsonb_build_object('original_sha256',encode(sha256,'hex')),sha256=NULL WHERE metadata->>'purpose'='feedback';
DROP INDEX storage.uq_files_tenant_dedup_shared;
CREATE UNIQUE INDEX uq_files_tenant_dedup ON storage.files(tenant_id,sha256,size_bytes) WHERE sha256 IS NOT NULL AND status='ready' AND deleted_at IS NULL;
CREATE OR REPLACE FUNCTION content.protect_reserved_page_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.page_type IN ('privacy-policy', 'terms-of-service', 'about-us') THEN
        RAISE EXCEPTION 'reserved core pages cannot be deleted';
    END IF;
    RETURN OLD;
END;
$$;


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



COMMIT;
