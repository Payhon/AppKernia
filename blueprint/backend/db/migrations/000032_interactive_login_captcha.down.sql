BEGIN;

-- Interactive proof tokens cannot be converted back to numeric answers.
UPDATE iam.login_captcha_challenges
SET consumed_at = COALESCE(consumed_at, now())
WHERE consumed_at IS NULL;

DROP INDEX iam.uq_login_captcha_scope_active;

ALTER TABLE iam.login_captcha_challenges
    ADD COLUMN answer_salt bytea,
    ADD COLUMN answer_hash bytea;

UPDATE iam.login_captcha_challenges
SET answer_salt = decode(repeat('00', 16), 'hex'),
    answer_hash = proof_hash;

ALTER TABLE iam.login_captcha_challenges
    ALTER COLUMN answer_salt SET NOT NULL,
    ALTER COLUMN answer_hash SET NOT NULL,
    DROP CONSTRAINT ck_login_captcha_type,
    DROP CONSTRAINT ck_login_captcha_proof_hash,
    DROP COLUMN captcha_type,
    DROP COLUMN proof_hash,
    ADD CONSTRAINT ck_login_captcha_salt
        CHECK (octet_length(answer_salt) = 16),
    ADD CONSTRAINT ck_login_captcha_answer_hash
        CHECK (octet_length(answer_hash) = 32);

CREATE INDEX idx_login_captcha_scope_active
    ON iam.login_captcha_challenges (scope_hash, expires_at DESC)
    WHERE consumed_at IS NULL;

COMMIT;
