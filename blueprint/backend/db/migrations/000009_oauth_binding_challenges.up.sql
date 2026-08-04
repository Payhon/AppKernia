BEGIN;

CREATE TABLE iam.oauth_binding_challenges (
    id                          uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id                     uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    tenant_id                   uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    provider                    varchar(64) NOT NULL,
    state_hash                  bytea NOT NULL,
    authorization_code_hash     bytea NOT NULL,
    pkce_verifier_encrypted     bytea NOT NULL,
    pkce_challenge              varchar(128) NOT NULL,
    expires_at                  timestamptz NOT NULL,
    consumed_at                 timestamptz,
    created_at                  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_oauth_binding_state_hash UNIQUE (state_hash),
    CONSTRAINT ck_oauth_binding_provider CHECK (provider ~ '^[a-z][a-z0-9-]{1,62}$'),
    CONSTRAINT ck_oauth_binding_pkce_challenge CHECK (length(pkce_challenge) BETWEEN 43 AND 128),
    CONSTRAINT ck_oauth_binding_expiry CHECK (expires_at > created_at)
);

CREATE INDEX idx_oauth_binding_user_pending
    ON iam.oauth_binding_challenges (user_id, provider, expires_at DESC)
    WHERE consumed_at IS NULL;

CREATE UNIQUE INDEX uq_oauth_account_user_provider
    ON iam.oauth_accounts (user_id, provider);

COMMIT;
