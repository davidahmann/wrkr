package queue

import "testing"

func TestValidTransitions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		from Status
		to   Status
	}{
		{StatusQueued, StatusRunning},
		{StatusRunning, StatusPaused},
		{StatusPaused, StatusRunning},
		{StatusRunning, StatusCompleted},
		{StatusBlockedDecision, StatusRunning},
	}

	for _, tc := range cases {
		if err := ValidateTransition(tc.from, tc.to); err != nil {
			t.Fatalf("expected transition %s->%s to be valid: %v", tc.from, tc.to, err)
		}
	}
}

func TestInvalidTransitions(t *testing.T) {
	t.Parallel()

	if err := ValidateTransition(StatusQueued, StatusCompleted); err == nil {
		t.Fatal("expected queued->completed to fail")
	}
	if err := ValidateTransition(StatusCompleted, StatusRunning); err == nil {
		t.Fatal("expected completed->running to fail")
	}
}
