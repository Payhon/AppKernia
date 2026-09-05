BEGIN;

DROP INDEX IF EXISTS audit.idx_operation_logs_api_client_time;
ALTER TABLE audit.operation_logs DROP CONSTRAINT IF EXISTS fk_operation_logs_api_client;
ALTER TABLE audit.operation_logs DROP COLUMN IF EXISTS api_client_id;

DROP INDEX IF EXISTS sys.idx_api_clients_bound_user;
ALTER TABLE sys.api_clients DROP CONSTRAINT IF EXISTS fk_api_clients_bound_member;
ALTER TABLE sys.api_clients DROP COLUMN IF EXISTS bound_user_id;

COMMIT;
