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

func setupReferenceRunner(t *testing.T, now time.Time) *runner.Runner {
	t.Helper()
	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	return r
}

func initReferenceJob(t *testing.T, r *runner.Runner, jobID string) {
	t.Helper()
	if _, err := r.InitJob(jobID); err != nil {
		t.Fatalf("InitJob(%s): %v", jobID, err)
	}
	if _, err := r.ChangeStatus(jobID, queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus(%s): %v", jobID, err)
	}
}

func TestRunCoveragePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 16, 0, 0, 0, time.UTC)

	if _, err := Run("job_ref_none", nil, RunOptions{Now: func() time.Time { return now }}); err == nil {
		t.Fatal("expected empty steps error")
	}

	r := setupReferenceRunner(t, now)
	initReferenceJob(t, r, "job_ref_start_index")

	advanced := false
	result, err := Run("job_ref_start_index", []Step{
		{ID: "only", Summary: "only", Executed: false},
	}, RunOptions{
		Now:        func() time.Time { return now },
		StartIndex: 99,
		OnAdvance: func(nextStepIndex int) error {
			advanced = true
			if nextStepIndex != 1 {
				t.Fatalf("unexpected next index: %d", nextStepIndex)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run start index path: %v", err)
	}
	if result.Status != queue.StatusCompleted || result.NextStepIndex != 1 || !advanced {
		t.Fatalf("unexpected start index result: %+v advanced=%t", result, advanced)
	}

	initReferenceJob(t, r, "job_ref_command_fail")
	_, err = Run("job_ref_command_fail", []Step{
		{ID: "bad", Summary: "bad command", Command: "exit 3", Executed: true},
	}, RunOptions{Now: func() time.Time { return now }})
	if err == nil {
		t.Fatal("expected command failure")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EAdapterFail {
		t.Fatalf("expected E_ADAPTER_FAIL, got %v", err)
	}

	initReferenceJob(t, r, "job_ref_onadvance_fail")
	_, err = Run("job_ref_onadvance_fail", []Step{
		{ID: "ok", Summary: "ok", Executed: false},
	}, RunOptions{
		Now: func() time.Time { return now },
		OnAdvance: func(nextStepIndex int) error {
			return errors.New("advance failure")
		},
	})
	if err == nil {
		t.Fatal("expected onAdvance error to propagate")
	}

	initReferenceJob(t, r, "job_ref_budget_block")
	_, err = Run("job_ref_budget_block", []Step{
		{ID: "step1", Summary: "s1", Executed: false},
		{ID: "step2", Summary: "s2", Executed: false},
	}, RunOptions{
		Now:          func() time.Time { return now },
		BudgetLimits: budget.Limits{MaxStepCount: 1},
	})
	if err == nil {
		t.Fatal("expected budget block")
	}
}

func TestStepsAndFieldHelpersCoveragePaths(t *testing.T) {
	t.Parallel()

	if _, err := StepsFromInputs(map[string]any{}); err == nil {
		t.Fatal("expected missing steps error")
	}
	if _, err := StepsFromInputs(map[string]any{"steps": "not-list"}); err == nil {
		t.Fatal("expected non-list steps error")
	}
	if _, err := StepsFromInputs(map[string]any{"steps": []any{"bad"}}); err == nil {
		t.Fatal("expected non-object step error")
	}

	steps, err := StepsFromInputs(map[string]any{
		"steps": []any{
			map[string]any{
				"id":              " a ",
				"summary":         " ",
				"command":         " true ",
				"artifacts":       []any{"out/a.md", "out/a.md", 5},
				"decision_needed": true,
				"required_action": " ",
				"executed":        true,
			},
		},
	})
	if err != nil {
		t.Fatalf("StepsFromInputs: %v", err)
	}
	if len(steps) != 1 || steps[0].RequiredAction != "approval" {
		t.Fatalf("unexpected normalized step: %+v", steps)
	}
	if action := requiredAction(steps[0]); action == nil || action.Kind != "approval" {
		t.Fatalf("expected required action, got %+v", action)
	}
	if action := requiredAction(Step{DecisionNeeded: false}); action != nil {
		t.Fatalf("expected nil required action for non-decision step, got %+v", action)
	}

	if got := boolField(map[string]any{"v": "x"}, "v"); got {
		t.Fatal("expected non-bool field to decode false")
	}
	if got := boolFieldWithDefault(map[string]any{"v": "x"}, "v", true); !got {
		t.Fatal("expected boolFieldWithDefault fallback=true")
	}
	if got := stringField(map[string]any{"v": 123}, "v"); got != "" {
		t.Fatalf("expected empty string for non-string field, got %q", got)
	}
	if got := stringSliceField(map[string]any{"v": "x"}, "v"); len(got) != 0 {
		t.Fatalf("expected empty slice for non-list field, got %v", got)
	}
}

