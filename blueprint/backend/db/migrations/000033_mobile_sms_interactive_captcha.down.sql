BEGIN;

-- A downgraded server cannot safely distinguish SMS-bound challenges.
UPDATE iam.interactive_captcha_challenges
SET consumed_at = COALESCE(consumed_at, now())
WHERE consumed_at IS NULL;

UPDATE sys.config_items
SET config_key = 'admin.login_captcha.type',
    display_name = 'Admin login CAPTCHA type',
    description = 'Interactive CAPTCHA type used by the Admin login flow.'
WHERE tenant_id IS NULL
  AND module_code = 'iam'
  AND config_group = 'security'
  AND config_key = 'interactive_captcha.type';

ALTER INDEX iam.uq_interactive_captcha_scope_active
    RENAME TO uq_login_captcha_scope_active;
ALTER INDEX iam.idx_interactive_captcha_expiry
    RENAME TO idx_login_captcha_expiry;

ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT interactive_captcha_challenges_pkey TO login_captcha_challenges_pkey;
ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT ck_interactive_captcha_scope_hash TO ck_login_captcha_scope_hash;
ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT ck_interactive_captcha_attempts TO ck_login_captcha_attempts;
ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT ck_interactive_captcha_expiry TO ck_login_captcha_expiry;
ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT ck_interactive_captcha_type TO ck_login_captcha_type;
ALTER TABLE iam.interactive_captcha_challenges
    RENAME CONSTRAINT ck_interactive_captcha_proof_hash TO ck_login_captcha_proof_hash;

ALTER TABLE iam.interactive_captcha_challenges
    RENAME TO login_captcha_challenges;

COMMIT;
