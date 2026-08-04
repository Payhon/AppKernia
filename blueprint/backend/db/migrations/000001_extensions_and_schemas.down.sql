BEGIN;

DROP FUNCTION IF EXISTS sys.touch_updated_at();
DROP SCHEMA IF EXISTS audit;
DROP SCHEMA IF EXISTS jobs;
DROP SCHEMA IF EXISTS notify;
DROP SCHEMA IF EXISTS storage;
DROP SCHEMA IF EXISTS sys;
DROP SCHEMA IF EXISTS org;
DROP SCHEMA IF EXISTS iam;

-- These extensions live in public and may be shared. Leave them installed on rollback.

COMMIT;
