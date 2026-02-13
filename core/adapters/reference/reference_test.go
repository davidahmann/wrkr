package reference

import (
	"errors"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/budget"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
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
	if result.NextStepIndex != 2 {
		t.Fatalf("expected next step index 2, got %d", result.NextStepIndex)
	}
}

func TestRunAppliesStartIndexAndCompletes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 1, 35, 0, 0, time.UTC)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_ref_resume"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_ref_resume", queue.StatusRunning); err != nil {
		t.Fatalf("running: %v", err)
	}

	var advances []int
	result, err := Run("job_ref_resume", []Step{
		{ID: "approve", Summary: "need approval", DecisionNeeded: true, Executed: false},
		{ID: "finalize", Summary: "finalize", Command: "true", Executed: true},
	}, RunOptions{
		Now:        func() time.Time { return now },
		StartIndex: 1,
		OnAdvance: func(nextStepIndex int) error {
			advances = append(advances, nextStepIndex)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != queue.StatusCompleted {
		t.Fatalf("expected completed, got %s", result.Status)
	}
	if result.NextStepIndex != 2 {
		t.Fatalf("expected next step index 2, got %d", result.NextStepIndex)
	}
	if len(advances) == 0 || advances[len(advances)-1] != 2 {
		t.Fatalf("unexpected advance sequence: %+v", advances)
	}
}

func TestRunStopsWhenBudgetExceeded(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 1, 40, 0, 0, time.UTC)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_ref_budget"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_ref_budget", queue.StatusRunning); err != nil {
		t.Fatalf("running: %v", err)
	}

	_, err = Run("job_ref_budget", []Step{
		{ID: "s1", Summary: "s1", Command: "true", Executed: true},
		{ID: "s2", Summary: "s2", Command: "true", Executed: true},
	}, RunOptions{
		Now:          func() time.Time { return now },
		BudgetLimits: budget.Limits{MaxStepCount: 1},
	})
	if err == nil {
		t.Fatal("expected budget exceeded error")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) {
		t.Fatalf("expected WrkrError, got %T", err)
	}
	if werr.Code != wrkrerrors.EBudgetExceeded {
		t.Fatalf("expected E_BUDGET_EXCEEDED, got %s", werr.Code)
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
