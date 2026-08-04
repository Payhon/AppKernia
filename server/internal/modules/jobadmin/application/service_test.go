package application

import (
	"encoding/json"
	"testing"
	"time"

	jobs "github.com/appkernia/appkernia/server/internal/modules/jobadmin/domain"
)

func TestPreviewCronHonorsIANAZoneAndDST(t *testing.T) {
	preview, err := PreviewCron("30 1 * * *", "America/New_York", time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC))
	if err != nil || len(preview.NextRuns) != 5 {
		t.Fatalf("preview failed: %#v %v", preview, err)
	}
	first := preview.NextRuns[0].In(mustLocation(t, "America/New_York"))
	if first.Hour() != 1 || first.Minute() != 30 {
		t.Fatalf("expected local 01:30, got %s", first)
	}
	if !preview.NextRuns[1].After(preview.NextRuns[0]) {
		t.Fatal("next runs must be strictly ordered")
	}
}

func TestNormalizeInputRejectsUnknownHandlerAndExecutablePayload(t *testing.T) {
	base := jobs.ScheduleInput{Code: "health.snapshot", Name: "Health snapshot", HandlerKey: "unknown.handler", CronExpression: "*/5 * * * *", TimeZone: "UTC"}
	if _, _, err := normalizeInput(base, time.Now()); err != jobs.ErrHandlerUnknown {
		t.Fatalf("expected unknown handler, got %v", err)
	}
	base.HandlerKey = "system.health.snapshot"
	base.Payload = json.RawMessage(`{"command":"rm -rf /"}`)
	if _, _, err := normalizeInput(base, time.Now()); err != jobs.ErrInvalid {
		t.Fatalf("expected payload rejection, got %v", err)
	}
}

func TestNormalizeInputAppliesBoundedPolicies(t *testing.T) {
	input, preview, err := normalizeInput(jobs.ScheduleInput{
		Code: "health.snapshot", Name: "Health snapshot", HandlerKey: "system.health.snapshot", CronExpression: "0 * * * *", Payload: json.RawMessage(`{}`),
	}, time.Date(2026, 8, 3, 1, 2, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if input.TimeZone != "UTC" || input.QueueName != "default" || input.OverlapPolicy != "skip" || input.MisfirePolicy != "fire_once" || input.TimeoutSeconds != 300 || input.MaxAttempts != 3 {
		t.Fatalf("unexpected defaults: %#v", input)
	}
	if got := preview.NextRuns[0]; !got.Equal(time.Date(2026, 8, 3, 2, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected next run %s", got)
	}
}

func mustLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	location, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return location
}
