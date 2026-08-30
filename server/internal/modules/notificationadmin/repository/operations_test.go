package repository

import "testing"

func TestDeliveryManualRetryRejection(t *testing.T) {
	tests := []struct {
		name         string
		retryable    bool
		risk         string
		resultClass  string
		count        int
		acknowledged bool
		want         string
	}{
		{name: "transient after automatic attempts exhausted", resultClass: "transient", count: 3},
		{name: "provider throttling", resultClass: "throttled", count: 100},
		{name: "auth requires a separate preflight check", resultClass: "auth_config_error", count: 1},
		{name: "unknown needs acknowledgement", resultClass: "unknown_after_write", count: 1, want: "duplicate_risk_requires_single_acknowledgement"},
		{name: "unknown cannot be retried in a batch", resultClass: "unknown_after_write", count: 2, acknowledged: true, want: "duplicate_risk_requires_single_acknowledgement"},
		{name: "unknown accepted once", resultClass: "unknown_after_write", count: 1, acknowledged: true},
		{name: "permanent failure", resultClass: "permanent", count: 1, acknowledged: true, want: "not_retryable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deliveryManualRetryRejection(tt.retryable, tt.risk, tt.resultClass, tt.count, tt.acknowledged); got != tt.want {
				t.Fatalf("deliveryManualRetryRejection() = %q, want %q", got, tt.want)
			}
		})
	}
}
