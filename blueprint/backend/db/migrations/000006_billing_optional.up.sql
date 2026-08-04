-- OPTIONAL MODULE. Do not include in the first AK core release unless payments/wallets are required.
BEGIN;

CREATE SCHEMA IF NOT EXISTS billing;

CREATE TABLE billing.payment_orders (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    user_id             uuid,
    order_number        varchar(64) NOT NULL,
    business_type       varchar(96) NOT NULL,
    business_id         uuid,
    subject             varchar(300) NOT NULL,
    description         varchar(1000),
    currency            char(3) NOT NULL,
    amount_minor        bigint NOT NULL,
    status              varchar(24) NOT NULL DEFAULT 'pending',
    expires_at          timestamptz,
    paid_at             timestamptz,
    cancelled_at        timestamptz,
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_payment_order_number UNIQUE (tenant_id, order_number),
    CONSTRAINT fk_payment_order_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT ck_payment_order_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT ck_payment_order_amount CHECK (amount_minor > 0),
    CONSTRAINT ck_payment_order_status CHECK (status IN ('pending', 'processing', 'paid', 'failed', 'cancelled', 'expired', 'partially_refunded', 'refunded'))
);
CREATE INDEX idx_payment_orders_user_time ON billing.payment_orders (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_payment_orders_status_time ON billing.payment_orders (tenant_id, status, created_at DESC);

CREATE TABLE billing.payment_transactions (
    id                      uuid PRIMARY KEY DEFAULT uuidv7(),
    payment_order_id        uuid NOT NULL REFERENCES billing.payment_orders(id) ON DELETE RESTRICT,
    transaction_number      varchar(96) NOT NULL,
    gateway                 varchar(32) NOT NULL,
    channel                 varchar(32) NOT NULL,
    provider_account        varchar(128),
    provider_transaction_id varchar(255),
    currency                char(3) NOT NULL,
    amount_minor            bigint NOT NULL,
    status                  varchar(24) NOT NULL DEFAULT 'created',
    request_payload         jsonb,
    response_payload        jsonb,
    callback_payload        jsonb,
    client_ip               inet,
    paid_at                 timestamptz,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_payment_transaction_number UNIQUE (transaction_number),
    CONSTRAINT ck_payment_transaction_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT uq_provider_transaction UNIQUE (gateway, provider_transaction_id),
    CONSTRAINT ck_payment_transaction_amount CHECK (amount_minor > 0),
    CONSTRAINT ck_payment_transaction_status CHECK (status IN ('created', 'pending', 'succeeded', 'failed', 'closed'))
);
CREATE INDEX idx_payment_transactions_order ON billing.payment_transactions (payment_order_id, created_at DESC);

CREATE TABLE billing.refunds (
    id                      uuid PRIMARY KEY DEFAULT uuidv7(),
    payment_order_id        uuid NOT NULL REFERENCES billing.payment_orders(id) ON DELETE RESTRICT,
    payment_transaction_id  uuid REFERENCES billing.payment_transactions(id) ON DELETE RESTRICT,
    refund_number           varchar(96) NOT NULL,
    provider_refund_id      varchar(255),
    currency                char(3) NOT NULL,
    amount_minor            bigint NOT NULL,
    reason                  varchar(1000),
    status                  varchar(24) NOT NULL DEFAULT 'pending',
    requested_by            uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    processed_at            timestamptz,
    response_payload        jsonb,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_refund_number UNIQUE (refund_number),
    CONSTRAINT ck_refund_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT uq_provider_refund UNIQUE (provider_refund_id),
    CONSTRAINT ck_refund_amount CHECK (amount_minor > 0),
    CONSTRAINT ck_refund_status CHECK (status IN ('pending', 'processing', 'succeeded', 'failed', 'cancelled'))
);
CREATE INDEX idx_refunds_order ON billing.refunds (payment_order_id, created_at DESC);

CREATE TABLE billing.wallet_accounts (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    user_id             uuid NOT NULL,
    asset_code          varchar(32) NOT NULL,
    account_type        varchar(24) NOT NULL DEFAULT 'available',
    balance_minor       bigint NOT NULL DEFAULT 0,
    frozen_minor        bigint NOT NULL DEFAULT 0,
    lock_version        integer NOT NULL DEFAULT 1,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_wallet_account UNIQUE (tenant_id, user_id, asset_code, account_type),
    CONSTRAINT fk_wallet_account_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT ck_wallet_account_type CHECK (account_type IN ('available', 'bonus', 'points')),
    CONSTRAINT ck_wallet_balances CHECK (balance_minor >= 0 AND frozen_minor >= 0),
    CONSTRAINT ck_wallet_version CHECK (lock_version > 0),
    CONSTRAINT ck_wallet_status CHECK (status IN ('active', 'frozen', 'closed'))
);

CREATE TABLE billing.wallet_ledger_entries (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    wallet_account_id   uuid NOT NULL REFERENCES billing.wallet_accounts(id) ON DELETE RESTRICT,
    entry_number        varchar(96) NOT NULL,
    entry_type          varchar(32) NOT NULL,
    direction           varchar(8) NOT NULL,
    amount_minor        bigint NOT NULL,
    balance_before      bigint NOT NULL,
    balance_after       bigint NOT NULL,
    business_type       varchar(96) NOT NULL,
    business_id         uuid,
    idempotency_key     varchar(255) NOT NULL,
    remark              varchar(1000),
    operator_user_id    uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    occurred_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_wallet_entry_number UNIQUE (entry_number),
    CONSTRAINT uq_wallet_entry_idempotency UNIQUE (wallet_account_id, idempotency_key),
    CONSTRAINT ck_wallet_entry_direction CHECK (direction IN ('credit', 'debit')),
    CONSTRAINT ck_wallet_entry_amount CHECK (amount_minor > 0),
    CONSTRAINT ck_wallet_entry_balance CHECK (
        balance_before >= 0 AND balance_after >= 0 AND (
            (direction = 'credit' AND balance_after = balance_before + amount_minor)
            OR (direction = 'debit' AND balance_after = balance_before - amount_minor)
        )
    )
);
CREATE INDEX idx_wallet_ledger_account_time ON billing.wallet_ledger_entries (wallet_account_id, occurred_at DESC);
CREATE INDEX idx_wallet_ledger_business ON billing.wallet_ledger_entries (business_type, business_id);

CREATE TABLE billing.withdrawals (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    wallet_account_id   uuid NOT NULL REFERENCES billing.wallet_accounts(id) ON DELETE RESTRICT,
    withdrawal_number   varchar(96) NOT NULL,
    amount_minor        bigint NOT NULL,
    fee_minor           bigint NOT NULL DEFAULT 0,
    payout_minor        bigint GENERATED ALWAYS AS (amount_minor - fee_minor) STORED,
    payout_method       varchar(32) NOT NULL,
    payout_target_enc   bytea NOT NULL,
    payout_key_version  integer NOT NULL,
    status              varchar(24) NOT NULL DEFAULT 'pending',
    reason              varchar(1000),
    handled_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    handled_at          timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_withdrawal_number UNIQUE (withdrawal_number),
    CONSTRAINT ck_withdrawal_amount CHECK (amount_minor > 0 AND fee_minor >= 0 AND amount_minor >= fee_minor),
    CONSTRAINT ck_withdrawal_status CHECK (status IN ('pending', 'approved', 'processing', 'succeeded', 'rejected', 'cancelled'))
);
CREATE INDEX idx_withdrawals_account_time ON billing.withdrawals (wallet_account_id, created_at DESC);
CREATE INDEX idx_withdrawals_status_time ON billing.withdrawals (status, created_at DESC);

CREATE TRIGGER tr_payment_orders_touch_updated_at
BEFORE UPDATE ON billing.payment_orders FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_payment_transactions_touch_updated_at
BEFORE UPDATE ON billing.payment_transactions FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_refunds_touch_updated_at
BEFORE UPDATE ON billing.refunds FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_wallet_accounts_touch_updated_at
BEFORE UPDATE ON billing.wallet_accounts FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_withdrawals_touch_updated_at
BEFORE UPDATE ON billing.withdrawals FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMENT ON TABLE billing.wallet_ledger_entries IS 'Immutable wallet/points ledger. Corrections are compensating entries, never UPDATE or DELETE.';

COMMIT;
