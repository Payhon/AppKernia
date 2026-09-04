package application

import "testing"

func TestValidateLoginSettings(t *testing.T) {
	if err := validateLoginSettings(LoginSettingsInput{OTPEnabled: true}); err == nil {
		t.Fatal("OTP login accepted without an enabled channel")
	}
	for _, input := range []LoginSettingsInput{
		{},
		{OTPEnabled: true, EmailOTPEnabled: true},
		{OTPEnabled: true, SMSOTPEnabled: true},
		{EmailOTPEnabled: true, LockVersion: 2},
	} {
		if err := validateLoginSettings(input); err != nil {
			t.Fatalf("valid login settings rejected: %#v: %v", input, err)
		}
	}
}
