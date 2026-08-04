BEGIN;

CREATE TABLE sys.modules (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    code                public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    version             varchar(64) NOT NULL,
    description         varchar(1000),
    capabilities        jsonb NOT NULL DEFAULT '{}'::jsonb,
    config_schema       jsonb NOT NULL DEFAULT '{}'::jsonb,
    status              varchar(20) NOT NULL DEFAULT 'enabled',
    installed_at        timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_modules_code UNIQUE (code),
    CONSTRAINT ck_modules_status CHECK (status IN ('enabled', 'disabled'))
);

CREATE TABLE sys.menus (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE CASCADE,
    parent_id           uuid REFERENCES sys.menus(id) ON DELETE RESTRICT,
    permission_id       uuid REFERENCES iam.permissions(id) ON DELETE SET NULL,
    code                public.citext NOT NULL,
    title               varchar(160) NOT NULL,
    menu_type           varchar(20) NOT NULL,
    route_name          varchar(160),
    route_path          varchar(500),
    component_key       varchar(255),
    redirect_path       varchar(500),
    icon                varchar(160),
    external_url        varchar(2000),
    open_mode           varchar(20) NOT NULL DEFAULT 'same_tab',
    active_menu_code    varchar(160),
    keep_alive          boolean NOT NULL DEFAULT false,
    hidden              boolean NOT NULL DEFAULT false,
    affix               boolean NOT NULL DEFAULT false,
    always_show         boolean NOT NULL DEFAULT false,
    sort_order          integer NOT NULL DEFAULT 0,
    status              varchar(20) NOT NULL DEFAULT 'active',
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT ck_menus_type CHECK (menu_type IN ('directory', 'page', 'external')),
    CONSTRAINT ck_menus_open_mode CHECK (open_mode IN ('same_tab', 'new_tab', 'iframe')),
    CONSTRAINT ck_menus_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_menus_page_fields CHECK (
        menu_type <> 'page' OR (route_path IS NOT NULL AND component_key IS NOT NULL)
    ),
    CONSTRAINT ck_menus_external_fields CHECK (
        menu_type <> 'external' OR external_url IS NOT NULL
    )
);
CREATE UNIQUE INDEX uq_menus_global_code
    ON sys.menus (code)
    WHERE tenant_id IS NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_menus_tenant_code
    ON sys.menus (tenant_id, code)
    WHERE tenant_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_menus_parent_sort ON sys.menus (tenant_id, parent_id, sort_order) WHERE deleted_at IS NULL;

CREATE TABLE sys.role_menus (
    tenant_id           uuid NOT NULL,
    role_id             uuid NOT NULL,
    menu_id             uuid NOT NULL REFERENCES sys.menus(id) ON DELETE CASCADE,
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, role_id, menu_id),
    CONSTRAINT fk_role_menus_role
        FOREIGN KEY (tenant_id, role_id) REFERENCES iam.roles(tenant_id, id)
        ON DELETE CASCADE
);
CREATE INDEX idx_role_menus_menu ON sys.role_menus (menu_id);

CREATE TABLE sys.config_items (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE CASCADE,
    module_code         varchar(64) NOT NULL DEFAULT 'core',
    config_group        varchar(96) NOT NULL DEFAULT 'default',
    config_key          public.citext NOT NULL,
    display_name        varchar(160) NOT NULL,
    value_type          varchar(24) NOT NULL DEFAULT 'string',
    value_json          jsonb,
    default_value_json  jsonb,
    is_secret           boolean NOT NULL DEFAULT false,
    secret_ciphertext   bytea,
    secret_key_version  integer,
    is_public           boolean NOT NULL DEFAULT false,
    validation_schema   jsonb NOT NULL DEFAULT '{}'::jsonb,
    description         varchar(1000),
    sort_order          integer NOT NULL DEFAULT 0,
    status              varchar(20) NOT NULL DEFAULT 'active',
    version             integer NOT NULL DEFAULT 1,
    created_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    updated_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_config_value_type CHECK (value_type IN ('string', 'integer', 'decimal', 'boolean', 'json', 'datetime')),
    CONSTRAINT ck_config_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT ck_config_version CHECK (version > 0),
    CONSTRAINT ck_config_secret_pair CHECK (
        (secret_ciphertext IS NULL AND secret_key_version IS NULL)
        OR (secret_ciphertext IS NOT NULL AND secret_key_version IS NOT NULL)
    ),
    CONSTRAINT ck_config_secret_storage CHECK (
        (is_secret AND value_json IS NULL AND default_value_json IS NULL AND NOT is_public)
        OR (NOT is_secret AND secret_ciphertext IS NULL AND secret_key_version IS NULL)
    )
);
CREATE UNIQUE INDEX uq_config_global_key
    ON sys.config_items (module_code, config_group, config_key)
    WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX uq_config_tenant_key
    ON sys.config_items (tenant_id, module_code, config_group, config_key)
    WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_config_public ON sys.config_items (tenant_id, module_code) WHERE is_public AND status = 'active';

CREATE TABLE sys.dict_types (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid REFERENCES iam.tenants(id) ON DELETE CASCADE,
    code                public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    description         varchar(500),
    is_system           boolean NOT NULL DEFAULT false,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_dict_types_status CHECK (status IN ('active', 'disabled'))
);
CREATE UNIQUE INDEX uq_dict_type_global_code
    ON sys.dict_types (code)
    WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX uq_dict_type_tenant_code
    ON sys.dict_types (tenant_id, code)
    WHERE tenant_id IS NOT NULL;

CREATE TABLE sys.dict_items (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    dict_type_id        uuid NOT NULL REFERENCES sys.dict_types(id) ON DELETE CASCADE,
    item_value          varchar(255) NOT NULL,
    label               varchar(255) NOT NULL,
    locale              varchar(32),
    color               varchar(64),
    css_class           varchar(128),
    sort_order          integer NOT NULL DEFAULT 0,
    is_default          boolean NOT NULL DEFAULT false,
    extra               jsonb NOT NULL DEFAULT '{}'::jsonb,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_dict_items_locale CHECK (locale IS NULL OR btrim(locale) <> ''),
    CONSTRAINT ck_dict_items_status CHECK (status IN ('active', 'disabled'))
);
CREATE UNIQUE INDEX uq_dict_item_value_locale
    ON sys.dict_items (dict_type_id, item_value, COALESCE(locale, ''));
CREATE INDEX idx_dict_items_type_sort ON sys.dict_items (dict_type_id, sort_order, id);

CREATE TABLE sys.regions (
    code                varchar(32) PRIMARY KEY,
    parent_code         varchar(32) REFERENCES sys.regions(code) ON DELETE RESTRICT,
    level               smallint NOT NULL,
    name                varchar(160) NOT NULL,
    full_name           varchar(500),
    postal_code         varchar(24),
    longitude           numeric(11, 7),
    latitude            numeric(10, 7),
    status              varchar(20) NOT NULL DEFAULT 'active',
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_regions_level CHECK (level BETWEEN 0 AND 10),
    CONSTRAINT ck_regions_status CHECK (status IN ('active', 'disabled'))
);
CREATE INDEX idx_regions_parent ON sys.regions (parent_code, code);
CREATE INDEX idx_regions_name_trgm ON sys.regions USING gin (name gin_trgm_ops);

CREATE TABLE sys.api_clients (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    client_id           public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    description         varchar(500),
    allowed_cidrs       cidr[] NOT NULL DEFAULT '{}'::cidr[],
    status              varchar(20) NOT NULL DEFAULT 'active',
    expires_at          timestamptz,
    created_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_api_clients_client_id UNIQUE (client_id),
    CONSTRAINT uq_api_clients_tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT ck_api_clients_status CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE sys.api_client_secrets (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    api_client_id       uuid NOT NULL REFERENCES sys.api_clients(id) ON DELETE CASCADE,
    secret_prefix       varchar(24) NOT NULL,
    secret_hash         bytea NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    expires_at          timestamptz,
    revoked_at          timestamptz,
    last_used_at        timestamptz,
    CONSTRAINT uq_api_client_secret_hash UNIQUE (secret_hash),
    CONSTRAINT ck_api_client_secret_hash_length CHECK (octet_length(secret_hash) = 32)
);
CREATE INDEX idx_api_client_secrets_active ON sys.api_client_secrets (api_client_id, created_at DESC) WHERE revoked_at IS NULL;

CREATE TABLE sys.api_client_permissions (
    tenant_id           uuid NOT NULL,
    api_client_id       uuid NOT NULL,
    permission_id       uuid NOT NULL REFERENCES iam.permissions(id) ON DELETE CASCADE,
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, api_client_id, permission_id),
    CONSTRAINT fk_api_client_permissions_client
        FOREIGN KEY (tenant_id, api_client_id) REFERENCES sys.api_clients(tenant_id, id)
        ON DELETE CASCADE
);

CREATE TABLE sys.idempotency_keys (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    identity_type       varchar(24) NOT NULL,
    identity_id         uuid NOT NULL,
    idempotency_key     varchar(255) NOT NULL,
    request_hash        bytea NOT NULL,
    response_status     integer,
    response_headers    jsonb,
    response_body       bytea,
    locked_until        timestamptz,
    expires_at          timestamptz NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    completed_at        timestamptz,
    CONSTRAINT uq_idempotency_identity_key UNIQUE (tenant_id, identity_type, identity_id, idempotency_key),
    CONSTRAINT ck_idempotency_identity_type CHECK (identity_type IN ('user', 'api_client')),
    CONSTRAINT ck_idempotency_response_status CHECK (response_status IS NULL OR response_status BETWEEN 100 AND 599),
    CONSTRAINT ck_idempotency_request_hash_length CHECK (octet_length(request_hash) = 32),
    CONSTRAINT ck_idempotency_expiry CHECK (expires_at > created_at)
);
CREATE INDEX idx_idempotency_expiry ON sys.idempotency_keys (expires_at);

CREATE TABLE sys.webhook_endpoints (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    name                varchar(160) NOT NULL,
    endpoint_url        varchar(2000) NOT NULL,
    event_types         text[] NOT NULL DEFAULT '{}'::text[],
    secret_ciphertext   bytea NOT NULL,
    secret_key_version  integer NOT NULL,
    max_attempts        integer NOT NULL DEFAULT 8,
    timeout_seconds     integer NOT NULL DEFAULT 10,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_by          uuid REFERENCES iam.users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_webhook_attempts CHECK (max_attempts BETWEEN 1 AND 100),
    CONSTRAINT ck_webhook_timeout CHECK (timeout_seconds BETWEEN 1 AND 60),
    CONSTRAINT ck_webhook_status CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE sys.webhook_deliveries (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    endpoint_id         uuid NOT NULL REFERENCES sys.webhook_endpoints(id) ON DELETE CASCADE,
    event_id            uuid NOT NULL,
    event_type          varchar(160) NOT NULL,
    payload             jsonb NOT NULL,
    status              varchar(20) NOT NULL DEFAULT 'pending',
    attempt_count       integer NOT NULL DEFAULT 0,
    next_attempt_at     timestamptz,
    response_status     integer,
    response_body       varchar(4000),
    last_error          varchar(2000),
    delivered_at        timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_webhook_event_endpoint UNIQUE (endpoint_id, event_id),
    CONSTRAINT ck_webhook_delivery_status CHECK (status IN ('pending', 'processing', 'succeeded', 'failed', 'cancelled')),
    CONSTRAINT ck_webhook_delivery_attempts CHECK (attempt_count >= 0),
    CONSTRAINT ck_webhook_delivery_response_status CHECK (response_status IS NULL OR response_status BETWEEN 100 AND 599)
);
CREATE INDEX idx_webhook_deliveries_due ON sys.webhook_deliveries (status, next_attempt_at) WHERE status IN ('pending', 'failed');

CREATE TABLE storage.files (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    owner_user_id       uuid,
    provider            varchar(32) NOT NULL,
    bucket_name         varchar(255) NOT NULL,
    object_key          varchar(1024) NOT NULL,
    original_name       varchar(1000) NOT NULL,
    media_type          varchar(255),
    extension           varchar(32),
    size_bytes          bigint NOT NULL,
    sha256              bytea,
    etag                varchar(255),
    visibility          varchar(20) NOT NULL DEFAULT 'private',
    status              varchar(20) NOT NULL DEFAULT 'pending',
    scan_status         varchar(20) NOT NULL DEFAULT 'pending',
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT uq_storage_object UNIQUE (provider, bucket_name, object_key),
    CONSTRAINT fk_storage_file_owner_member
        FOREIGN KEY (tenant_id, owner_user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT uq_storage_files_tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT ck_files_size CHECK (size_bytes >= 0),
    CONSTRAINT ck_files_sha256_length CHECK (sha256 IS NULL OR octet_length(sha256) = 32),
    CONSTRAINT ck_files_visibility CHECK (visibility IN ('private', 'public')),
    CONSTRAINT ck_files_status CHECK (status IN ('pending', 'ready', 'quarantined', 'deleted')),
    CONSTRAINT ck_files_scan_status CHECK (scan_status IN ('pending', 'clean', 'infected', 'failed', 'skipped'))
);
CREATE INDEX idx_files_tenant_time ON storage.files (tenant_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_owner ON storage.files (owner_user_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_files_tenant_dedup
    ON storage.files (tenant_id, sha256, size_bytes)
    WHERE sha256 IS NOT NULL AND status = 'ready' AND deleted_at IS NULL;
CREATE INDEX idx_files_name_trgm ON storage.files USING gin (original_name gin_trgm_ops);

CREATE TABLE storage.upload_sessions (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    user_id             uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    file_id             uuid,
    provider            varchar(32) NOT NULL,
    provider_upload_id  varchar(1000),
    bucket_name         varchar(255) NOT NULL,
    object_key          varchar(1024) NOT NULL,
    original_name       varchar(1000) NOT NULL,
    media_type          varchar(255),
    expected_size       bigint NOT NULL,
    expected_sha256     bytea,
    part_size           bigint,
    status              varchar(20) NOT NULL DEFAULT 'initiated',
    expires_at          timestamptz NOT NULL,
    completed_at        timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_upload_session_object UNIQUE (provider, bucket_name, object_key),
    CONSTRAINT fk_upload_sessions_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_upload_sessions_file
        FOREIGN KEY (tenant_id, file_id) REFERENCES storage.files(tenant_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT ck_upload_expected_size CHECK (expected_size >= 0),
    CONSTRAINT ck_upload_sha256_length CHECK (expected_sha256 IS NULL OR octet_length(expected_sha256) = 32),
    CONSTRAINT ck_upload_part_size CHECK (part_size IS NULL OR part_size > 0),
    CONSTRAINT ck_upload_status CHECK (status IN ('initiated', 'uploading', 'completed', 'aborted', 'expired')),
    CONSTRAINT ck_upload_expiry CHECK (expires_at > created_at)
);
CREATE INDEX idx_upload_sessions_user ON storage.upload_sessions (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_upload_sessions_expiry ON storage.upload_sessions (expires_at) WHERE status IN ('initiated', 'uploading');

CREATE TABLE storage.upload_parts (
    upload_session_id   uuid NOT NULL REFERENCES storage.upload_sessions(id) ON DELETE CASCADE,
    part_number         integer NOT NULL,
    etag                varchar(255) NOT NULL,
    size_bytes          bigint NOT NULL,
    checksum_sha256     bytea,
    uploaded_at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (upload_session_id, part_number),
    CONSTRAINT ck_upload_part_number CHECK (part_number > 0),
    CONSTRAINT ck_upload_part_size_bytes CHECK (size_bytes > 0),
    CONSTRAINT ck_upload_part_checksum CHECK (checksum_sha256 IS NULL OR octet_length(checksum_sha256) = 32)
);

CREATE TABLE storage.file_usages (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    file_id             uuid NOT NULL,
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE CASCADE,
    module_code         varchar(64) NOT NULL,
    entity_type         varchar(128) NOT NULL,
    entity_id           uuid NOT NULL,
    field_name          varchar(128) NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_file_usage UNIQUE (file_id, module_code, entity_type, entity_id, field_name),
    CONSTRAINT fk_file_usage_file
        FOREIGN KEY (tenant_id, file_id) REFERENCES storage.files(tenant_id, id)
        ON DELETE CASCADE
);
CREATE INDEX idx_file_usage_entity ON storage.file_usages (tenant_id, module_code, entity_type, entity_id);

ALTER TABLE iam.users
    ADD CONSTRAINT fk_users_avatar_file
    FOREIGN KEY (avatar_file_id) REFERENCES storage.files(id) ON DELETE SET NULL;

CREATE TRIGGER tr_modules_touch_updated_at
BEFORE UPDATE ON sys.modules FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_menus_touch_updated_at
BEFORE UPDATE ON sys.menus FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_config_touch_updated_at
BEFORE UPDATE ON sys.config_items FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_dict_types_touch_updated_at
BEFORE UPDATE ON sys.dict_types FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_dict_items_touch_updated_at
BEFORE UPDATE ON sys.dict_items FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_api_clients_touch_updated_at
BEFORE UPDATE ON sys.api_clients FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_webhook_endpoints_touch_updated_at
BEFORE UPDATE ON sys.webhook_endpoints FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_webhook_deliveries_touch_updated_at
BEFORE UPDATE ON sys.webhook_deliveries FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_files_touch_updated_at
BEFORE UPDATE ON storage.files FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_upload_sessions_touch_updated_at
BEFORE UPDATE ON storage.upload_sessions FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMENT ON TABLE sys.menus IS 'Admin navigation metadata. UI actions and API authorization are represented by iam.permissions, not encoded as menu rows.';
COMMENT ON TABLE sys.config_items IS 'Typed global or tenant configuration; secret values must be envelope-encrypted and never stored in value_json.';
COMMENT ON TABLE sys.modules IS 'Compile-time module registry. AK does not load or uninstall arbitrary Go code at runtime.';
COMMENT ON TABLE storage.files IS 'Object storage metadata. Uploads use presigned URLs and post-upload validation/virus scanning.';

COMMIT;
