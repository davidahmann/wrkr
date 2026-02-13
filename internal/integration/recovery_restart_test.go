package integration

import (
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
)

func TestForcedRestartResumesFromDurableCheckpoint(t *testing.T) {
	t.Parallel()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)
	r1, err := runner.New(s, runner.Options{Now: func() time.Time { return now }, LeaseTTL: 30 * time.Second})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}

	if _, err := r1.InitJob("job_resume"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r1.ChangeStatus("job_resume", queue.StatusRunning); err != nil {
		t.Fatalf("status running: %v", err)
	}
	if _, err := r1.UpdateCounters("job_resume", 2, 15, 30); err != nil {
		t.Fatalf("counters: %v", err)
	}
	if _, err := r1.RecordIdempotencyKey("job_resume", "tool-call-15"); err != nil {
		t.Fatalf("idempotency: %v", err)
	}

	// Simulate process restart by creating a new runner with same store root.
	s2, err := store.New(s.Root())
	if err != nil {
		t.Fatalf("store2.New: %v", err)
	}
	r2, err := runner.New(s2, runner.Options{Now: func() time.Time { return now.Add(10 * time.Second) }, LeaseTTL: 30 * time.Second})
	if err != nil {
		t.Fatalf("runner2.New: %v", err)
	}

	state, err := r2.Recover("job_resume")
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if state.Status != queue.StatusRunning {
		t.Fatalf("expected status running, got %s", state.Status)
	}
	if state.RetryCount != 2 || state.StepCount != 15 || state.ToolCallCount != 30 {
		t.Fatalf("unexpected counters after recovery: %+v", state)
	}
	if !state.IdempotencyKeys["tool-call-15"] {
		t.Fatal("missing idempotency key after recovery")
	}
}
