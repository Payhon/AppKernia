package application

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeSelfProfileAcceptsSupportedValues(t *testing.T) {
	displayName := "  AppKernia Admin  "
	locale := "en-US"
	timeZone := "Asia/Shanghai"
	input, err := normalizeSelfProfile(UpdateSelfProfileInput{
		DisplayName: &displayName, Locale: &locale, TimeZone: &timeZone, RequestID: "request-1",
	})
	if err != nil {
		t.Fatalf("normalize self profile: %v", err)
	}
	if *input.DisplayName != "AppKernia Admin" || *input.Locale != "en-US" || *input.TimeZone != "Asia/Shanghai" {
		t.Fatalf("unexpected normalized profile: %#v", input)
	}
}

func TestNormalizeSelfProfileRejectsInvalidValues(t *testing.T) {
	validName := "Admin"
	unsupportedLocale := "fr-FR"
	invalidTimeZone := "Mars/Olympus"
	tooLongName := strings.Repeat("名", 161)
	tests := []UpdateSelfProfileInput{
		{RequestID: "request-1"},
		{DisplayName: &validName},
		{Locale: &unsupportedLocale, RequestID: "request-1"},
		{TimeZone: &invalidTimeZone, RequestID: "request-1"},
		{DisplayName: &tooLongName, RequestID: "request-1"},
	}
	for index, input := range tests {
		if _, err := normalizeSelfProfile(input); !errors.Is(err, ErrProfileValidation) {
			t.Fatalf("case %d: expected profile validation error, got %v", index, err)
		}
	}
}
