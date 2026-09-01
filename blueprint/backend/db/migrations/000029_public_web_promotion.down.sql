BEGIN;

ALTER TABLE app.application_public_web_translations
  DROP CONSTRAINT ck_application_public_web_promotion_lengths,
  DROP COLUMN promotion_button_label,
  DROP COLUMN promotion_description,
  DROP COLUMN promotion_title;

ALTER TABLE app.application_public_web_configs
  DROP COLUMN promotion_enabled;

COMMIT;
