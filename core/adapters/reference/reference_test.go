package reference

import (
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
)

func TestRunDecisionCheckpoint(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 1, 30, 0, 0, time.UTC)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_ref_decision"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_ref_decision", queue.StatusRunning); err != nil {
		t.Fatalf("running: %v", err)
	}

	result, err := Run("job_ref_decision", []Step{
		{ID: "analyze", Summary: "analyze", Command: "true", Executed: true},
		{ID: "approve", Summary: "need approval", DecisionNeeded: true, Executed: false},
	}, RunOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != queue.StatusBlockedDecision {
		t.Fatalf("expected blocked_decision, got %s", result.Status)
	}
}

func TestStepsFromInputs(t *testing.T) {
	steps, err := StepsFromInputs(map[string]any{
		"steps": []any{
			map[string]any{"id": "s1", "summary": "x", "command": "true", "artifacts": []any{"a", "b"}, "executed": true},
		},
	})
	if err != nil {
		t.Fatalf("StepsFromInputs: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].ID != "s1" {
		t.Fatalf("unexpected step: %+v", steps[0])
	}
}
