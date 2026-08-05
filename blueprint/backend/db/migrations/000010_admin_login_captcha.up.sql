BEGIN;

CREATE TABLE iam.login_failure_states (
    scope_hash      bytea PRIMARY KEY,
    failure_count   integer NOT NULL DEFAULT 0,
    last_failed_at  timestamptz NOT NULL,
    expires_at      timestamptz NOT NULL,
    CONSTRAINT ck_login_failure_scope_hash CHECK (octet_length(scope_hash) = 32),
    CONSTRAINT ck_login_failure_count CHECK (failure_count BETWEEN 0 AND 1000000),
    CONSTRAINT ck_login_failure_expiry CHECK (expires_at > last_failed_at)
);

CREATE INDEX idx_login_failure_states_expiry
    ON iam.login_failure_states (expires_at);

CREATE TABLE iam.login_captcha_challenges (
    id              uuid PRIMARY KEY DEFAULT uuidv7(),
    scope_hash      bytea NOT NULL,
    answer_salt     bytea NOT NULL,
    answer_hash     bytea NOT NULL,
    attempt_count   smallint NOT NULL DEFAULT 0,
    expires_at      timestamptz NOT NULL,
    consumed_at     timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_login_captcha_scope_hash CHECK (octet_length(scope_hash) = 32),
    CONSTRAINT ck_login_captcha_salt CHECK (octet_length(answer_salt) = 16),
    CONSTRAINT ck_login_captcha_answer_hash CHECK (octet_length(answer_hash) = 32),
    CONSTRAINT ck_login_captcha_attempts CHECK (attempt_count BETWEEN 0 AND 5),
    CONSTRAINT ck_login_captcha_expiry CHECK (expires_at > created_at)
);

CREATE INDEX idx_login_captcha_scope_active
    ON iam.login_captcha_challenges (scope_hash, expires_at DESC)
    WHERE consumed_at IS NULL;

CREATE INDEX idx_login_captcha_expiry
    ON iam.login_captcha_challenges (expires_at);

COMMIT;
