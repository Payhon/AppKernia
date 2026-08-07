package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type captureOTPNotifier struct{ calls []OTPNotification }

func (n *captureOTPNotifier) QueueOTP(_ context.Context, _ pgx.Tx, input OTPNotification) error {
	n.calls = append(n.calls, input)
	return nil
}

func TestHashDocumentIsDeterministicAndLocaleBound(t *testing.T) {
	body := json.RawMessage(`"privacy text"`)
	first := HashDocument("zh-CN", "隐私政策", "markdown", body)
	second := HashDocument("zh-CN", "隐私政策", "markdown", body)
	english := HashDocument("en-US", "Privacy policy", "markdown", body)
	if string(first) != string(second) {
		t.Fatal("same content must produce same hash")
	}
	if string(first) == string(english) {
		t.Fatal("locale and translated content must be bound into consent hash")
	}
}

func TestOTPAndEmailValidation(t *testing.T) {
	if !validOTP("012345") || validOTP("12345x") || validOTP("12345") {
		t.Fatal("OTP validation must be exact six digits")
	}
	if _, ok := normalizedEmail("USER@example.test"); !ok {
		t.Fatal("normalized valid email rejected")
	}
	if _, ok := normalizedEmail("not-an-email"); ok {
		t.Fatal("invalid email accepted")
	}
}

func TestAppStatusUsesLifecyclePermissionForEnableAndDisable(t *testing.T) {
	for _, status := range []string{"active", "disabled"} {
		if got := appStatusPermission(status); got != "app.application.disable" {
			t.Fatalf("%s permission = %q, want app.application.disable", status, got)
		}
	}
}

func TestAdminListFilterBoundsAndStatus(t *testing.T) {
	filter, err := normalizedAdminListFilter(AdminListFilter{Query: "  mobile  ", Status: "active"}, "active", "disabled")
	if err != nil || filter.Query != "mobile" || filter.Page != 1 || filter.PageSize != 20 {
		t.Fatalf("default normalized filter = %#v, err=%v", filter, err)
	}
	for _, invalid := range []AdminListFilter{
		{Page: -1}, {PageSize: 101}, {Status: "archived"}, {Query: string(make([]rune, 161))},
	} {
		if _, err := normalizedAdminListFilter(invalid, "active", "disabled"); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("invalid filter %#v error=%v, want ErrInvalidInput", invalid, err)
		}
	}
}

func TestServiceReceivesOTPNotifierAndFailsClosedWithoutOne(t *testing.T) {
	capture := &captureOTPNotifier{}
	service := NewService(nil, nil, WithOTPNotifier(capture))
	if service.otp != capture {
		t.Fatal("OTP notifier option was not retained")
	}
	if NewService(nil, nil).otp != nil {
		t.Fatal("OTP notifier must not default to a fake success adapter")
	}
}
