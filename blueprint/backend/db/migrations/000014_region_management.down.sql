BEGIN;

DROP INDEX IF EXISTS sys.idx_regions_active_parent;

ALTER TABLE sys.regions
    DROP CONSTRAINT IF EXISTS ck_regions_version,
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS is_manually_managed,
    DROP COLUMN IF EXISTS version;

COMMIT;
