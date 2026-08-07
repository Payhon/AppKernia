//go:build integration

package application

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type integrationOTPNotifier struct {
	calls []OTPNotification
	err   error
}

func (n *integrationOTPNotifier) QueueOTP(_ context.Context, _ pgx.Tx, input OTPNotification) error {
	n.calls = append(n.calls, input)
	return n.err
}

func TestMobileOTPFlowsInvokeNotifierAndRollBack(t *testing.T) {
	dsn := os.Getenv("AK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AK_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	suffix := uuid.NewString()
	var tenantID, appID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO iam.tenants(code,name) VALUES($1,$2) RETURNING id`, "otp-"+suffix, "OTP Integration").Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = pool.Exec(context.Background(), `DELETE FROM iam.tenants WHERE id=$1`, tenantID) }()
	if err = pool.QueryRow(ctx, `SELECT id FROM app.applications WHERE tenant_id=$1 AND is_default`, tenantID).Scan(&appID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE app.applications SET registration_enabled=true,registration_verification='email_otp' WHERE id=$1`, appID); err != nil {
		t.Fatal(err)
	}
	var templateExists bool
	if err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notify.templates WHERE code='verification_code' AND channel='email' AND status='active')`).Scan(&templateExists); err != nil || !templateExists {
		t.Skip("verification_code template is not seeded")
	}
	app := Application{ID: appID, TenantID: tenantID, RegistrationEnabled: true, RegistrationVerification: "email_otp", DefaultLocale: "en-US"}
	n := &integrationOTPNotifier{}
	service := NewService(pool, nil, WithOTPNotifier(n))
	email := "otp-" + suffix + "@example.test"
	if err = service.RegisterMobile(ctx, app, email, "OTP User", "otp integration password 2026!", "en-US"); err != nil {
		t.Fatal(err)
	}
	if len(n.calls) != 1 || n.calls[0].Purpose != "email_otp" || n.calls[0].Code == "" {
		t.Fatalf("register notifier calls=%#v", n.calls)
	}
	if _, err = pool.Exec(ctx, `UPDATE iam.verification_challenges SET created_at=now()-interval '2 minutes' WHERE user_id=(SELECT id FROM iam.users WHERE email=$1)`, email); err != nil {
		t.Fatal(err)
	}
	if _, err = service.ResendRegistrationEmail(ctx, app, email); err != nil {
		t.Fatal(err)
	}
	if len(n.calls) != 2 || n.calls[1].Purpose != "email_otp" {
		t.Fatalf("resend notifier calls=%#v", n.calls)
	}
	if _, err = service.ForgotMobilePassword(ctx, app, email); err != nil {
		t.Fatal(err)
	}
	if len(n.calls) != 2 {
		t.Fatalf("forgot pending account must remain non-enumerating, calls=%d", len(n.calls))
	}
	if _, err = pool.Exec(ctx, `UPDATE app.user_memberships SET status='active' WHERE app_id=$1 AND user_id=(SELECT id FROM iam.users WHERE email=$2)`, appID, email); err != nil {
		t.Fatal(err)
	}
	if _, err = service.ForgotMobilePassword(ctx, app, email); err != nil {
		t.Fatal(err)
	}
	if len(n.calls) != 3 || n.calls[2].Purpose != "password_reset" {
		t.Fatalf("forgot notifier calls=%#v", n.calls)
	}
	n.err = errors.New("queue failed")
	failedEmail := "rollback-" + suffix + "@example.test"
	if err = service.RegisterMobile(ctx, app, failedEmail, "Rollback User", "otp integration password 2026!", "en-US"); err == nil {
		t.Fatal("notifier failure must fail registration")
	}
	var users int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM iam.users WHERE email=$1`, failedEmail).Scan(&users); err != nil || users != 0 {
		t.Fatalf("notifier failure leaked user count=%d err=%v", users, err)
	}
	app.RegistrationVerification = "none"
	n.err = nil
	before := len(n.calls)
	if err = service.RegisterMobile(ctx, app, "none-"+suffix+"@example.test", "None User", "otp integration password 2026!", "en-US"); err != nil {
		t.Fatal(err)
	}
	if len(n.calls) != before {
		t.Fatal("none verification must not queue OTP")
	}
}
