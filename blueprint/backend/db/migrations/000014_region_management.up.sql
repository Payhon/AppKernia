BEGIN;

ALTER TABLE sys.regions
    ADD COLUMN version integer NOT NULL DEFAULT 1,
    ADD COLUMN is_manually_managed boolean NOT NULL DEFAULT false,
    ADD COLUMN deleted_at timestamptz;

ALTER TABLE sys.regions
    ADD CONSTRAINT ck_regions_version CHECK (version > 0);

CREATE INDEX idx_regions_active_parent
    ON sys.regions (parent_code, code)
    WHERE deleted_at IS NULL;

COMMIT;
