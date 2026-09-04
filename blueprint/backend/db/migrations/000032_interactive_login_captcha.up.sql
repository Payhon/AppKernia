BEGIN;

-- Numeric CAPTCHA answers cannot be upgraded into opaque interactive proofs.
-- Consume every pre-upgrade challenge so no legacy credential remains usable.
UPDATE iam.login_captcha_challenges
SET consumed_at = COALESCE(consumed_at, now())
WHERE consumed_at IS NULL;

ALTER TABLE iam.login_captcha_challenges
    ADD COLUMN captcha_type text,
    ADD COLUMN proof_hash bytea;

UPDATE iam.login_captcha_challenges
SET captcha_type = 'slide',
    proof_hash = answer_hash;

ALTER TABLE iam.login_captcha_challenges
    ALTER COLUMN captcha_type SET NOT NULL,
    ALTER COLUMN proof_hash SET NOT NULL,
    DROP CONSTRAINT ck_login_captcha_salt,
    DROP CONSTRAINT ck_login_captcha_answer_hash,
    DROP COLUMN answer_salt,
    DROP COLUMN answer_hash,
    ADD CONSTRAINT ck_login_captcha_type
        CHECK (captcha_type IN ('click', 'slide', 'drag', 'rotate')),
    ADD CONSTRAINT ck_login_captcha_proof_hash
        CHECK (octet_length(proof_hash) = 32);

DROP INDEX iam.idx_login_captcha_scope_active;

CREATE UNIQUE INDEX uq_login_captcha_scope_active
    ON iam.login_captcha_challenges (scope_hash)
    WHERE consumed_at IS NULL;

COMMIT;
