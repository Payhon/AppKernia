package worker

import (
	"errors"
	"strings"
	"testing"
)

func TestSMSFailureClassificationsNeverAutoRetryUncertainResults(t *testing.T) {
	uncertainResult := uncertain(errors.New("timeout after request write"))
	if uncertainResult.retryable || uncertainResult.risk != "duplicate_possible" {
		t.Fatalf("uncertain SMS result must require manual duplicate-risk confirmation: %#v", uncertainResult)
	}
	permanentResult := permanent(errors.New("template rejected"))
	if permanentResult.retryable || permanentResult.risk != "manual_review" {
		t.Fatalf("permanent rejection must require manual review: %#v", permanentResult)
	}
}

func TestSafeErrorBoundsAndRemovesControlCharacters(t *testing.T) {
	got := safeError(errors.New(strings.Repeat("x", 600) + "\nsecret\tvalue"))
	if len(got) != 500 || strings.ContainsAny(got, "\n\r\t") {
		t.Fatalf("safe error is not bounded: length=%d value=%q", len(got), got)
	}
}
