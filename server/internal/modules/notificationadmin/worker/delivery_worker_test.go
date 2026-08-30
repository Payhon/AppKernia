package worker

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestPushRetryDelayUsesBackoffProviderHintAndJitter(t *testing.T) {
	id := uuid.MustParse("018f08d0-3b00-7000-8000-000000000001")
	first := pushRetryDelay(id, 1, 0)
	second := pushRetryDelay(id, 2, 0)
	hinted := pushRetryDelay(id, 1, 2*time.Minute)
	if first < 30*time.Second || second < 60*time.Second || second <= first {
		t.Fatalf("retry backoff did not increase: first=%s second=%s", first, second)
	}
	if hinted < 2*time.Minute || hinted > 3*time.Minute {
		t.Fatalf("provider Retry-After was not honored with bounded jitter: %s", hinted)
	}
}

func TestSafeErrorBoundsAndRemovesControlCharacters(t *testing.T) {
	got := safeError(errors.New(strings.Repeat("x", 600) + "\nsecret\tvalue"))
	if len(got) != 500 || strings.ContainsAny(got, "\n\r\t") {
		t.Fatalf("safe error is not bounded: length=%d value=%q", len(got), got)
	}
}
