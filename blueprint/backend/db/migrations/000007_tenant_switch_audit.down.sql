BEGIN;

ALTER TABLE audit.login_events
    DROP CONSTRAINT ck_login_auth_method,
    ADD CONSTRAINT ck_login_auth_method CHECK (
        auth_method IN ('password', 'email_otp', 'sms_otp', 'oauth', 'refresh_token', 'api_secret', 'mfa')
    );

COMMIT;
