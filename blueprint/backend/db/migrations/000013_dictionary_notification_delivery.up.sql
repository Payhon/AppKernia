BEGIN;

ALTER TABLE sys.dict_types
    ADD COLUMN name_key varchar(200),
    ADD COLUMN description_key varchar(200),
    ADD COLUMN visibility varchar(20) NOT NULL DEFAULT 'internal',
    ADD COLUMN extension_policy varchar(24) NOT NULL DEFAULT 'open',
    ADD CONSTRAINT ck_dict_types_visibility CHECK (visibility IN ('internal', 'public')),
    ADD CONSTRAINT ck_dict_types_extension_policy CHECK (extension_policy IN ('fixed', 'open', 'registered', 's3_compatible'));

ALTER TABLE sys.dict_items
    ADD COLUMN tenant_id uuid REFERENCES iam.tenants(id) ON DELETE CASCADE;

UPDATE sys.dict_items item
SET tenant_id = type.tenant_id
FROM sys.dict_types type
WHERE type.id = item.dict_type_id AND type.tenant_id IS NOT NULL;

DROP INDEX sys.uq_dict_item_value_locale;
CREATE UNIQUE INDEX uq_dict_item_global_value_locale
    ON sys.dict_items (dict_type_id, item_value, COALESCE(locale, ''))
    WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX uq_dict_item_tenant_value_locale
    ON sys.dict_items (tenant_id, dict_type_id, item_value, COALESCE(locale, ''))
    WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_dict_items_tenant_type_sort
    ON sys.dict_items (tenant_id, dict_type_id, sort_order, id);

ALTER TABLE notify.templates
    ADD COLUMN body_format varchar(16) NOT NULL DEFAULT 'plain',
    ADD CONSTRAINT ck_notify_template_body_format CHECK (body_format IN ('plain', 'html'));

UPDATE sys.config_items
SET status = 'disabled', updated_at = now()
WHERE module_code = 'notifications' AND config_group = 'sms'
  AND config_key IN ('sms.access_key_id','sms.access_key_secret','sms.sign_name','sms.template_code');

ALTER TABLE notify.deliveries
    ADD COLUMN dedupe_key varchar(255),
    ADD COLUMN payload_ciphertext bytea,
    ADD COLUMN payload_key_version integer,
    ADD COLUMN retryable boolean NOT NULL DEFAULT false,
    ADD COLUMN retry_risk varchar(24) NOT NULL DEFAULT 'none',
    ADD CONSTRAINT ck_notify_delivery_retry_risk CHECK (retry_risk IN ('none', 'duplicate_possible', 'manual_review')),
    ADD CONSTRAINT ck_notify_delivery_payload_pair CHECK (
        (payload_ciphertext IS NULL AND payload_key_version IS NULL)
        OR (payload_ciphertext IS NOT NULL AND payload_key_version IS NOT NULL)
    );

CREATE UNIQUE INDEX uq_notify_delivery_dedupe
    ON notify.deliveries (tenant_id, channel, dedupe_key)
    WHERE dedupe_key IS NOT NULL;

CREATE TABLE notify.sms_template_bindings (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    template_id         uuid NOT NULL REFERENCES notify.templates(id) ON DELETE CASCADE,
    provider            varchar(64) NOT NULL,
    external_template_id varchar(255) NOT NULL,
    sign_name           varchar(120),
    parameter_order     jsonb NOT NULL DEFAULT '[]'::jsonb,
    status              varchar(20) NOT NULL DEFAULT 'active',
    version             integer NOT NULL DEFAULT 1,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_sms_template_binding UNIQUE (tenant_id, template_id, provider),
    CONSTRAINT ck_sms_template_binding_provider CHECK (provider IN ('aliyun', 'tencent')),
    CONSTRAINT ck_sms_template_binding_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_sms_template_binding_version CHECK (version > 0),
    CONSTRAINT ck_sms_template_binding_parameter_order CHECK (jsonb_typeof(parameter_order) = 'array')
);

CREATE TRIGGER tr_sms_template_bindings_touch_updated_at
BEFORE UPDATE ON notify.sms_template_bindings
FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMIT;
