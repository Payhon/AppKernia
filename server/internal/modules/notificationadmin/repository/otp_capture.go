package repository

import (
	"context"
	"errors"
	"strings"

	login "github.com/appkernia/appkernia/server/internal/modules/loginprovider/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseCaptureOTP is deliberately development/test-only. It stores the
// plaintext code without the target so local API flows can be tested without
// configuring email or SMS providers.
type DatabaseCaptureOTP struct{}

func NewDatabaseCaptureOTP(ctx context.Context, pool *pgxpool.Pool) (*DatabaseCaptureOTP, error) {
	if pool == nil {
		return nil, errors.New("OTP capture database is unavailable")
	}
	_, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS test_support;
CREATE UNLOGGED TABLE IF NOT EXISTS test_support.otp_captures (
    challenge_id uuid PRIMARY KEY,
    app_id uuid NOT NULL,
    purpose varchar(32) NOT NULL,
    identifier_type varchar(16) NOT NULL,
    target_hint varchar(160) NOT NULL,
    code varchar(6) NOT NULL CHECK (code ~ '^[0-9]{6}$'),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_otp_captures_app_created ON test_support.otp_captures(app_id,created_at DESC);`)
	if err != nil {
		return nil, err
	}
	return &DatabaseCaptureOTP{}, nil
}

func (n *DatabaseCaptureOTP) Queue(ctx context.Context, tx pgx.Tx, input login.OTPChallenge) error {
	if tx == nil || input.ID == uuid.Nil || input.AppID == uuid.Nil || len(input.Code) != 6 || strings.TrimSpace(input.DisplayHint) == "" {
		return errors.New("OTP capture input is invalid")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM test_support.otp_captures WHERE expires_at<=now()`); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO test_support.otp_captures(
challenge_id,app_id,purpose,identifier_type,target_hint,code,expires_at)
VALUES($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT(challenge_id) DO UPDATE SET code=EXCLUDED.code,expires_at=EXCLUDED.expires_at`,
		input.ID, input.AppID, input.Purpose, input.IdentifierType, input.DisplayHint, input.Code, input.ExpiresAt)
	return err
}
