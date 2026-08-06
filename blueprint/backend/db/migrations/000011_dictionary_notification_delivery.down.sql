BEGIN;

DROP TRIGGER IF EXISTS tr_sms_template_bindings_touch_updated_at ON notify.sms_template_bindings;
DROP TABLE IF EXISTS notify.sms_template_bindings;

DROP INDEX IF EXISTS notify.uq_notify_delivery_dedupe;
ALTER TABLE notify.deliveries
    DROP CONSTRAINT IF EXISTS ck_notify_delivery_payload_pair,
    DROP CONSTRAINT IF EXISTS ck_notify_delivery_retry_risk,
    DROP COLUMN IF EXISTS retry_risk,
    DROP COLUMN IF EXISTS retryable,
    DROP COLUMN IF EXISTS payload_key_version,
    DROP COLUMN IF EXISTS payload_ciphertext,
    DROP COLUMN IF EXISTS dedupe_key;

ALTER TABLE notify.templates
    DROP CONSTRAINT IF EXISTS ck_notify_template_body_format,
    DROP COLUMN IF EXISTS body_format;

UPDATE sys.config_items
SET status = 'active', updated_at = now()
WHERE module_code = 'notifications' AND config_group = 'sms'
  AND config_key IN ('sms.access_key_id','sms.access_key_secret','sms.sign_name','sms.template_code');

DROP INDEX IF EXISTS sys.idx_dict_items_tenant_type_sort;
DROP INDEX IF EXISTS sys.uq_dict_item_tenant_value_locale;
DROP INDEX IF EXISTS sys.uq_dict_item_global_value_locale;
DELETE FROM sys.dict_items item
USING sys.dict_types type
WHERE item.dict_type_id = type.id AND item.tenant_id IS NOT NULL AND type.tenant_id IS NULL;
ALTER TABLE sys.dict_items DROP COLUMN IF EXISTS tenant_id;
CREATE UNIQUE INDEX uq_dict_item_value_locale
    ON sys.dict_items (dict_type_id, item_value, COALESCE(locale, ''));

ALTER TABLE sys.dict_types
    DROP CONSTRAINT IF EXISTS ck_dict_types_extension_policy,
    DROP CONSTRAINT IF EXISTS ck_dict_types_visibility,
    DROP COLUMN IF EXISTS extension_policy,
    DROP COLUMN IF EXISTS visibility,
    DROP COLUMN IF EXISTS description_key,
    DROP COLUMN IF EXISTS name_key;

COMMIT;
