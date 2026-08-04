BEGIN;

DROP TABLE IF EXISTS iam.block_rules;
DROP TABLE IF EXISTS iam.mfa_recovery_codes;
DROP TABLE IF EXISTS iam.mfa_factors;
DROP TABLE IF EXISTS iam.verification_challenges;
DROP TABLE IF EXISTS iam.refresh_tokens;
DROP TABLE IF EXISTS iam.sessions;
DROP TABLE IF EXISTS iam.devices;
DROP TABLE IF EXISTS iam.role_permissions;
DROP TABLE IF EXISTS iam.user_roles;
DROP TABLE IF EXISTS iam.permissions;
DROP TABLE IF EXISTS iam.roles;
DROP TABLE IF EXISTS iam.tenant_members;
DROP TABLE IF EXISTS iam.oauth_accounts;
DROP TABLE IF EXISTS iam.password_history;
DROP TABLE IF EXISTS iam.user_credentials;
DROP TABLE IF EXISTS iam.users;
DROP TABLE IF EXISTS iam.tenants;

COMMIT;
