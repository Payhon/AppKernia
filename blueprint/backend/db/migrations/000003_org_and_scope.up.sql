BEGIN;

CREATE TABLE org.units (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    parent_id           uuid,
    code                public.citext NOT NULL,
    name                varchar(160) NOT NULL,
    unit_type           varchar(24) NOT NULL DEFAULT 'department',
    leader_user_id      uuid,
    phone               varchar(32),
    email               public.citext,
    sort_order          integer NOT NULL DEFAULT 0,
    status              varchar(20) NOT NULL DEFAULT 'active',
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT uq_org_units_tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT fk_org_units_leader_member
        FOREIGN KEY (tenant_id, leader_user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_org_units_parent_same_tenant
        FOREIGN KEY (tenant_id, parent_id) REFERENCES org.units(tenant_id, id)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT ck_org_units_type CHECK (unit_type IN ('company', 'division', 'department', 'team', 'group')),
    CONSTRAINT ck_org_units_status CHECK (status IN ('active', 'disabled'))
);
CREATE UNIQUE INDEX uq_org_units_tenant_code_active
    ON org.units (tenant_id, code)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_org_units_parent ON org.units (tenant_id, parent_id, sort_order) WHERE deleted_at IS NULL;
CREATE INDEX idx_org_units_name_trgm ON org.units USING gin (name gin_trgm_ops);

CREATE TABLE org.positions (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL REFERENCES iam.tenants(id) ON DELETE RESTRICT,
    code                public.citext NOT NULL,
    name                varchar(120) NOT NULL,
    description         varchar(500),
    sort_order          integer NOT NULL DEFAULT 0,
    status              varchar(20) NOT NULL DEFAULT 'active',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT uq_positions_tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT ck_positions_status CHECK (status IN ('active', 'disabled'))
);

CREATE UNIQUE INDEX uq_positions_tenant_code_active
    ON org.positions (tenant_id, code)
    WHERE deleted_at IS NULL;

CREATE TABLE org.user_units (
    tenant_id           uuid NOT NULL,
    user_id             uuid NOT NULL,
    unit_id             uuid NOT NULL,
    is_primary          boolean NOT NULL DEFAULT false,
    joined_at           timestamptz NOT NULL DEFAULT now(),
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id, unit_id),
    CONSTRAINT fk_user_units_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_user_units_unit
        FOREIGN KEY (tenant_id, unit_id) REFERENCES org.units(tenant_id, id)
        ON DELETE CASCADE
);
CREATE UNIQUE INDEX uq_user_primary_unit
    ON org.user_units (tenant_id, user_id)
    WHERE is_primary;
CREATE INDEX idx_user_units_unit ON org.user_units (tenant_id, unit_id, user_id);

CREATE TABLE org.user_positions (
    id                  uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id           uuid NOT NULL,
    user_id             uuid NOT NULL,
    position_id         uuid NOT NULL,
    unit_id             uuid,
    is_primary          boolean NOT NULL DEFAULT false,
    assigned_at         timestamptz NOT NULL DEFAULT now(),
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_positions_member
        FOREIGN KEY (tenant_id, user_id) REFERENCES iam.tenant_members(tenant_id, user_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_user_positions_position
        FOREIGN KEY (tenant_id, position_id) REFERENCES org.positions(tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_user_positions_unit
        FOREIGN KEY (tenant_id, unit_id) REFERENCES org.units(tenant_id, id)
        ON DELETE CASCADE
);
CREATE UNIQUE INDEX uq_user_positions_with_unit
    ON org.user_positions (tenant_id, user_id, position_id, unit_id)
    WHERE unit_id IS NOT NULL;
CREATE UNIQUE INDEX uq_user_positions_without_unit
    ON org.user_positions (tenant_id, user_id, position_id)
    WHERE unit_id IS NULL;
CREATE UNIQUE INDEX uq_user_primary_position
    ON org.user_positions (tenant_id, user_id)
    WHERE is_primary;
CREATE INDEX idx_user_positions_position ON org.user_positions (tenant_id, position_id);

CREATE TABLE iam.role_scope_units (
    tenant_id           uuid NOT NULL,
    role_id             uuid NOT NULL,
    unit_id             uuid NOT NULL,
    include_descendants boolean NOT NULL DEFAULT false,
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, role_id, unit_id),
    CONSTRAINT fk_role_scope_role
        FOREIGN KEY (tenant_id, role_id) REFERENCES iam.roles(tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_role_scope_unit
        FOREIGN KEY (tenant_id, unit_id) REFERENCES org.units(tenant_id, id)
        ON DELETE CASCADE
);

CREATE TRIGGER tr_org_units_touch_updated_at
BEFORE UPDATE ON org.units FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();
CREATE TRIGGER tr_positions_touch_updated_at
BEFORE UPDATE ON org.positions FOR EACH ROW EXECUTE FUNCTION sys.touch_updated_at();

COMMENT ON TABLE org.units IS 'Organization tree stored as an adjacency list. Descendant queries use PostgreSQL recursive CTEs; no duplicated level/tree strings.';
COMMENT ON TABLE iam.role_scope_units IS 'Custom organization data scope assigned to a role.';

COMMIT;
