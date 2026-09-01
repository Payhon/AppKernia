BEGIN;
ALTER TABLE app.application_store_listings
 ADD COLUMN platform varchar(16),
 ADD COLUMN web_url varchar(2048) NOT NULL DEFAULT '',
 ADD CONSTRAINT ck_store_web_platform CHECK (platform IS NULL OR platform IN ('android','ios','harmony')),
 ADD CONSTRAINT ck_store_web_url CHECK (web_url='' OR web_url ~ '^https://[^[:space:]]+$');
CREATE TABLE app.application_public_web_configs (
 tenant_id uuid NOT NULL,
 app_id uuid NOT NULL,
 enabled boolean NOT NULL DEFAULT false,
 apk_enabled boolean NOT NULL DEFAULT false,
 lock_version integer NOT NULL DEFAULT 1 CHECK (lock_version > 0),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (tenant_id, app_id),
 FOREIGN KEY (tenant_id,app_id) REFERENCES app.applications(tenant_id,id) ON DELETE CASCADE
);
CREATE TRIGGER tr_application_public_web_configs_touch_updated_at
BEFORE UPDATE ON app.application_public_web_configs
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TABLE app.application_public_web_translations (
 tenant_id uuid NOT NULL,
 app_id uuid NOT NULL,
 locale varchar(10) NOT NULL CHECK (locale IN ('zh-CN','en-US')),
 name varchar(160) NOT NULL,
 introduction text NOT NULL,
 PRIMARY KEY (tenant_id,app_id,locale),
 FOREIGN KEY (tenant_id,app_id) REFERENCES app.application_public_web_configs(tenant_id,app_id) ON DELETE CASCADE,
 CHECK (length(name) <= 160 AND length(introduction) <= 20000)
);
INSERT INTO iam.permissions(code,name,module_code,resource_name,action_name,permission_kind,http_methods,route_pattern,description,status) VALUES
 ('app.public_web.read','查看发行页配置','app','public_web','read','api',ARRAY['GET'],'/admin-api/v1/apps/{app_id}/public-web-config','Read public web configuration','active'),
 ('app.public_web.update','更新发行页配置','app','public_web','update','api',ARRAY['PUT'],'/admin-api/v1/apps/{app_id}/public-web-config','Update public web configuration','active')
ON CONFLICT(code) DO NOTHING;
INSERT INTO iam.role_permissions(tenant_id,role_id,permission_id,granted_by)
SELECT rp.tenant_id,rp.role_id,p.id,rp.granted_by FROM iam.role_permissions rp
JOIN iam.permissions source ON source.id=rp.permission_id
JOIN iam.permissions p ON p.code=CASE source.code WHEN 'app.application.read' THEN 'app.public_web.read' WHEN 'app.application.update' THEN 'app.public_web.update' END
WHERE source.code IN ('app.application.read','app.application.update')
ON CONFLICT DO NOTHING;
COMMIT;
