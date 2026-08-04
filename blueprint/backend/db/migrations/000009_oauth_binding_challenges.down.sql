BEGIN;

DROP TABLE IF EXISTS iam.oauth_binding_challenges;
DROP INDEX IF EXISTS iam.uq_oauth_account_user_provider;

COMMIT;
