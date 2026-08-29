package http

import (
	"testing"
	"time"
)

func TestOptionalRFC3339(t *testing.T) {
	if value, err := optionalRFC3339(""); err != nil || value != nil {
		t.Fatalf("empty value=%v err=%v", value, err)
	}
	value, err := optionalRFC3339("2026-08-28T12:30:00Z")
	if err != nil || value == nil || !value.Equal(time.Date(2026, 8, 28, 12, 30, 0, 0, time.UTC)) {
		t.Fatalf("parsed value=%v err=%v", value, err)
	}
	if _, err = optionalRFC3339("2026-08-28"); err == nil {
		t.Fatal("date-only value must be rejected")
	}
}
