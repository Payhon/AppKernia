BEGIN;

ALTER TABLE app.application_public_web_configs
  ADD COLUMN promotion_enabled boolean NOT NULL DEFAULT true;

ALTER TABLE app.application_public_web_translations
  ADD COLUMN promotion_title varchar(160) NOT NULL DEFAULT '',
  ADD COLUMN promotion_description varchar(500) NOT NULL DEFAULT '',
  ADD COLUMN promotion_button_label varchar(80) NOT NULL DEFAULT '',
  ADD CONSTRAINT ck_application_public_web_promotion_lengths CHECK (
    length(promotion_title) <= 160
    AND length(promotion_description) <= 500
    AND length(promotion_button_label) <= 80
  );

COMMIT;
