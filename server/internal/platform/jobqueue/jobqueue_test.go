package jobqueue

import (
	"errors"
	"testing"
	"time"
)

type testArgs struct{ kind string }

func (a testArgs) Kind() string { return a.kind }

func TestCodeValidation(t *testing.T) {
	for _, value := range []string{"notify", "notification.delivery", "push-fanout", "task_2"} {
		if !code(value, 80) {
			t.Fatalf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"", "N", "Notification", "notify/unsafe", "1notify"} {
		if code(value, 80) {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func TestSafeSummary(t *testing.T) {
	got := safe(" provider\n error\twith details ", 16)
	if got != "provider error w" {
		t.Fatalf("unexpected summary: %q", got)
	}
}

func TestRegistryValidatesCompiledExecutionPolicy(t *testing.T) {
	registry, err := NewRegistry(Definition{
		Kind: "notification-delivery", Queue: "notifications", MaxAttempts: 5, Timeout: time.Minute,
		RetryClasses: []string{RetryClassThrottled, RetryClassTransient},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = registry.ValidateSpec(Spec{Args: testArgs{kind: "notification-delivery"}, Queue: "notifications", MaxAttempts: 3}); err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}
	if err = registry.ValidateSpec(Spec{Args: testArgs{kind: "notification-delivery"}, Queue: "default", MaxAttempts: 3}); !errors.Is(err, ErrSpecMismatch) {
		t.Fatalf("queue mismatch error=%v", err)
	}
	if err = registry.ValidateSpec(Spec{Args: testArgs{kind: "notification-delivery"}, Queue: "notifications", MaxAttempts: 6}); !errors.Is(err, ErrSpecMismatch) {
		t.Fatalf("attempt limit error=%v", err)
	}
	if err = registry.ValidateSpec(Spec{Args: testArgs{kind: "unknown"}, Queue: "notifications", MaxAttempts: 1}); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("unknown kind error=%v", err)
	}
}

func TestRegistryRejectsInvalidOrDuplicateDefinitions(t *testing.T) {
	cases := [][]Definition{
		{{Kind: "notification-delivery", Queue: "notifications", MaxAttempts: 0, Timeout: time.Minute}},
		{{Kind: "notification-delivery", Queue: "notifications", MaxAttempts: 5, Timeout: 0}},
		{{Kind: "notification-delivery", Queue: "notifications", MaxAttempts: 5, Timeout: time.Minute, RetryClasses: []string{"unknown_after_write"}}},
		{
			{Kind: "notification-delivery", Queue: "notifications", MaxAttempts: 5, Timeout: time.Minute},
			{Kind: "notification-delivery", Queue: "notifications", MaxAttempts: 5, Timeout: time.Minute},
		},
	}
	for index, definitions := range cases {
		if _, err := NewRegistry(definitions...); !errors.Is(err, ErrInvalidRegistry) {
			t.Fatalf("case %d error=%v", index, err)
		}
	}
}

func TestRegistryDefinitionIsImmutableFromCaller(t *testing.T) {
	retryClasses := []string{RetryClassTransient}
	registry := MustRegistry(Definition{Kind: "notification-delivery", Queue: "notifications", MaxAttempts: 5, Timeout: time.Minute, RetryClasses: retryClasses})
	retryClasses[0] = RetryClassThrottled
	definition, ok := registry.Definition("notification-delivery")
	if !ok || definition.RetryClasses[0] != RetryClassTransient {
		t.Fatalf("registry was mutated by input: %#v", definition)
	}
	definition.RetryClasses[0] = RetryClassThrottled
	again, _ := registry.Definition("notification-delivery")
	if again.RetryClasses[0] != RetryClassTransient {
		t.Fatalf("registry was mutated by output: %#v", again)
	}
}
