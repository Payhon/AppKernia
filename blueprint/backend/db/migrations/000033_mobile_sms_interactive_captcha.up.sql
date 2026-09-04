BEGIN;

ALTER TABLE iam.login_captcha_challenges
    RENAME TO interactive_captcha_challenges;

ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT login_captcha_challenges_pkey TO interactive_captcha_challenges_pkey;
ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT ck_login_captcha_scope_hash TO ck_interactive_captcha_scope_hash;
ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT ck_login_captcha_attempts TO ck_interactive_captcha_attempts;
ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT ck_login_captcha_expiry TO ck_interactive_captcha_expiry;
ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT ck_login_captcha_type TO ck_interactive_captcha_type;
ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT ck_login_captcha_proof_hash TO ck_interactive_captcha_proof_hash;

ALTER INDEX iam.uq_login_captcha_scope_active
    RENAME TO uq_interactive_captcha_scope_active;
ALTER INDEX iam.idx_login_captcha_expiry
    RENAME TO idx_interactive_captcha_expiry;

UPDATE sys.config_items
SET config_key = 'interactive_captcha.type',
    display_name = 'Interactive CAPTCHA type',
    description = 'Interactive CAPTCHA type shared by Admin login and Mobile SMS protection.'
WHERE tenant_id IS NULL
  AND module_code = 'iam'
  AND config_group = 'security'
  AND config_key = 'admin.login_captcha.type';

COMMIT;
