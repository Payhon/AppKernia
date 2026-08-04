BEGIN;

DROP TABLE IF EXISTS audit.security_events;
DROP TABLE IF EXISTS audit.login_events;
DROP TABLE IF EXISTS audit.operation_logs;

DROP TABLE IF EXISTS jobs.inbox_events;
DROP TABLE IF EXISTS jobs.outbox_events;
DROP TABLE IF EXISTS jobs.schedule_runs;
DROP TABLE IF EXISTS jobs.schedules;

DROP TABLE IF EXISTS notify.push_devices;
DROP TABLE IF EXISTS notify.deliveries;
DROP TABLE IF EXISTS notify.recipients;
DROP TABLE IF EXISTS notify.messages;
DROP TABLE IF EXISTS notify.templates;

COMMIT;
