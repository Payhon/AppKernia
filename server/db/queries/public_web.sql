-- name: PublicWebGetConfig :one
SELECT a.id AS app_id, COALESCE(c.enabled,false)::boolean AS enabled, COALESCE(c.apk_enabled,false)::boolean AS apk_enabled,
 COALESCE(c.promotion_enabled,true)::boolean AS promotion_enabled, COALESCE(c.lock_version,0)::integer AS lock_version
FROM app.applications a LEFT JOIN app.application_public_web_configs c ON c.tenant_id=a.tenant_id AND c.app_id=a.id
WHERE a.tenant_id=$1 AND a.id=$2 AND a.deleted_at IS NULL;

-- name: PublicWebLockApp :one
SELECT id FROM app.applications WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE;

-- name: PublicWebWriteConfig :exec
INSERT INTO app.application_public_web_configs(tenant_id,app_id,enabled,apk_enabled,promotion_enabled)
VALUES($1,$2,$3,$4,$5) ON CONFLICT(tenant_id,app_id) DO UPDATE SET enabled=EXCLUDED.enabled,apk_enabled=EXCLUDED.apk_enabled,promotion_enabled=EXCLUDED.promotion_enabled,lock_version=app.application_public_web_configs.lock_version+1,updated_at=now();

-- name: PublicWebTranslations :many
SELECT locale,name,introduction,promotion_title,promotion_description,promotion_button_label FROM app.application_public_web_translations WHERE tenant_id=$1 AND app_id=$2 ORDER BY locale;

-- name: PublicWebWriteTranslation :exec
INSERT INTO app.application_public_web_translations(tenant_id,app_id,locale,name,introduction,promotion_title,promotion_description,promotion_button_label) VALUES($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT(tenant_id,app_id,locale) DO UPDATE SET name=EXCLUDED.name,introduction=EXCLUDED.introduction,promotion_title=EXCLUDED.promotion_title,promotion_description=EXCLUDED.promotion_description,promotion_button_label=EXCLUDED.promotion_button_label;

-- name: PublicWebStores :many
SELECT id,name,scheme,enabled,priority,COALESCE(platform,'')::text AS platform,web_url
FROM app.application_store_listings WHERE tenant_id=$1 AND app_id=$2 ORDER BY priority DESC,id;

-- name: PublicWebWriteStore :execrows
UPDATE app.application_store_listings SET platform=NULLIF(sqlc.arg(platform)::text,''),web_url=sqlc.arg(web_url)
WHERE tenant_id=sqlc.arg(tenant_id) AND app_id=sqlc.arg(app_id) AND id=sqlc.arg(id);

-- name: PublicWebAsset :one
SELECT f.id,f.provider,f.bucket_name,f.object_key,COALESCE(f.media_type,'')::text AS media_type,f.size_bytes
FROM storage.files f JOIN app.applications a ON a.tenant_id=f.tenant_id AND a.id=$2
WHERE f.tenant_id=$1 AND f.id=$3 AND a.status='active' AND a.deleted_at IS NULL
 AND f.deleted_at IS NULL AND f.status='ready' AND f.scan_status IN ('clean','skipped')
 AND COALESCE(f.metadata->>'purpose','')<>'feedback' AND f.media_type IN ('image/png','image/jpeg','image/webp')
 AND (a.icon_file_id=f.id OR EXISTS (
 SELECT 1 FROM app.application_assets asset JOIN app.application_public_web_configs c ON c.tenant_id=asset.tenant_id AND c.app_id=asset.app_id
 WHERE asset.tenant_id=a.tenant_id AND asset.app_id=a.id AND asset.file_id=f.id AND asset.asset_type='screenshot' AND c.enabled));

-- name: PublicWebScreenshots :many
SELECT asset.file_id FROM app.application_assets asset JOIN storage.files f ON f.tenant_id=asset.tenant_id AND f.id=asset.file_id
WHERE asset.tenant_id=$1 AND asset.app_id=$2 AND asset.asset_type='screenshot'
 AND f.deleted_at IS NULL AND f.status='ready' AND f.scan_status IN ('clean','skipped') AND COALESCE(f.metadata->>'purpose','')<>'feedback' AND f.media_type IN ('image/png','image/jpeg','image/webp')
ORDER BY asset.position,asset.id;

-- name: PublicWebPageLinks :many
SELECT p.slug,t.title FROM content.pages p JOIN content.page_revision_translations t ON t.revision_id=p.current_revision_id AND t.locale=$3
WHERE p.tenant_id=$1 AND p.app_id=$2 AND p.status='published' ORDER BY p.slug;

-- name: PublicWebPageImageCandidates :many
SELECT t.body,t.body_format FROM content.pages p
JOIN content.page_revisions r ON r.id=p.current_revision_id AND r.app_id=p.app_id AND r.status='published'
JOIN content.page_revision_translations t ON t.revision_id=r.id
WHERE p.tenant_id=$1 AND p.app_id=$2 AND p.status='published' AND strpos(t.body::text,sqlc.arg(file_id_text)::text)>0;

-- name: PublicWebPageImageFile :one
SELECT f.id,f.provider,f.bucket_name,f.object_key,COALESCE(f.media_type,'')::text AS media_type,f.size_bytes
FROM storage.files f WHERE f.tenant_id=$1 AND f.id=$2
AND f.deleted_at IS NULL AND f.status='ready' AND f.scan_status IN ('clean','skipped')
AND COALESCE(f.metadata->>'purpose','')<>'feedback' AND f.media_type IN ('image/png','image/jpeg','image/webp');
