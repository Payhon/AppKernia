BEGIN;

ALTER TABLE sys.api_clients
    ADD COLUMN bound_user_id uuid;

ALTER TABLE sys.api_clients
    ADD CONSTRAINT fk_api_clients_bound_member
        FOREIGN KEY (tenant_id, bound_user_id)
        REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE SET NULL (bound_user_id);

CREATE INDEX idx_api_clients_bound_user
    ON sys.api_clients (tenant_id, bound_user_id)
    WHERE bound_user_id IS NOT NULL;

ALTER TABLE audit.operation_logs
    ADD COLUMN api_client_id uuid;

ALTER TABLE audit.operation_logs
    ADD CONSTRAINT fk_operation_logs_api_client
        FOREIGN KEY (tenant_id, api_client_id)
        REFERENCES sys.api_clients(tenant_id, id)
        ON DELETE SET NULL (api_client_id);

CREATE INDEX idx_operation_logs_api_client_time
    ON audit.operation_logs (api_client_id, occurred_at DESC)
    WHERE api_client_id IS NOT NULL;

COMMENT ON COLUMN sys.api_clients.bound_user_id IS
    'Optional active tenant member whose current permissions are intersected with this API Client for explicitly agent-callable operations.';
COMMENT ON COLUMN audit.operation_logs.api_client_id IS
    'Machine actor that initiated a delegated operation; user_id remains the effective bound user.';

COMMIT;
