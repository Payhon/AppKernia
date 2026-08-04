BEGIN;

DROP TABLE IF EXISTS billing.withdrawals;
DROP TABLE IF EXISTS billing.wallet_ledger_entries;
DROP TABLE IF EXISTS billing.wallet_accounts;
DROP TABLE IF EXISTS billing.refunds;
DROP TABLE IF EXISTS billing.payment_transactions;
DROP TABLE IF EXISTS billing.payment_orders;
DROP SCHEMA IF EXISTS billing;

COMMIT;
