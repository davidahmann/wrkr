package runner

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/budget"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/schema/validate"
)

func TestCheckpointEmitListAndShow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 14, 0, 0, 0, time.UTC)
	r := testRunner(t, now)

	if _, err := r.InitJob("job_cp"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_cp", queue.StatusRunning); err != nil {
		t.Fatalf("status: %v", err)
	}

	cp1, err := r.EmitCheckpoint("job_cp", CheckpointInput{
		Type:    "plan",
		Summary: "build a concrete execution plan",
	})
	if err != nil {
		t.Fatalf("emit checkpoint 1: %v", err)
	}
	cp2, err := r.EmitCheckpoint("job_cp", CheckpointInput{
		Type:    "progress",
		Summary: "implemented runner checkpoint persistence",
	})
	if err != nil {
		t.Fatalf("emit checkpoint 2: %v", err)
	}

	list, err := r.ListCheckpoints("job_cp")
	if err != nil {
		t.Fatalf("list checkpoints: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 checkpoints, got %d", len(list))
	}
	if list[0].CheckpointID != cp1.CheckpointID || list[1].CheckpointID != cp2.CheckpointID {
		t.Fatalf("unexpected checkpoint order: %s, %s", list[0].CheckpointID, list[1].CheckpointID)
	}

	got, err := r.GetCheckpoint("job_cp", cp2.CheckpointID)
	if err != nil {
		t.Fatalf("get checkpoint: %v", err)
	}
	if got.Summary != cp2.Summary {
		t.Fatalf("expected summary %q, got %q", cp2.Summary, got.Summary)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal checkpoint: %v", err)
	}
	if err := validate.ValidateBytes(validate.CheckpointSchemaRel, raw); err != nil {
		t.Fatalf("checkpoint should validate against schema: %v", err)
	}
}

func TestBudgetExceededEmitsBlockedCheckpoint(t *testing.T) {
	t.Parallel()

	current := time.Date(2026, 2, 13, 14, 0, 0, 0, time.UTC)
	r := testRunner(t, current)
	r.now = func() time.Time { return current }

	if _, err := r.InitJob("job_budget"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_budget", queue.StatusRunning); err != nil {
		t.Fatalf("status: %v", err)
	}
	if _, err := r.UpdateCounters("job_budget", 1, 12, 7); err != nil {
		t.Fatalf("counters: %v", err)
	}

	current = current.Add(5 * time.Minute)
	cp, err := r.CheckBudget("job_budget", budget.Limits{MaxWallTimeSeconds: 60, MaxStepCount: 10})
	if err == nil {
		t.Fatal("expected budget exceeded error")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EBudgetExceeded {
		t.Fatalf("expected E_BUDGET_EXCEEDED, got %v", err)
	}
	if cp == nil || cp.Type != "blocked" {
		t.Fatalf("expected blocked checkpoint, got %+v", cp)
	}
	if len(cp.ReasonCodes) == 0 || cp.ReasonCodes[0] != string(wrkrerrors.EBudgetExceeded) {
		t.Fatalf("expected reason code E_BUDGET_EXCEEDED, got %+v", cp.ReasonCodes)
	}
}

func TestResumeRequiresApprovalForDecisionCheckpoint(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 14, 0, 0, 0, time.UTC)
	r := testRunner(t, now)

	if _, err := r.InitJob("job_approve"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_approve", queue.StatusRunning); err != nil {
		t.Fatalf("status running: %v", err)
	}

	cp, err := r.EmitCheckpoint("job_approve", CheckpointInput{
		Type:    "decision-needed",
		Summary: "approval required to proceed",
		Status:  queue.StatusBlockedDecision,
		RequiredAction: &v1.RequiredAction{
			Kind:         "approval",
			Instructions: "confirm migration scope",
		},
	})
	if err != nil {
		t.Fatalf("emit decision checkpoint: %v", err)
	}
	if _, err := r.ChangeStatus("job_approve", queue.StatusBlockedDecision); err != nil {
		t.Fatalf("status blocked decision: %v", err)
	}

	if _, err := r.Resume("job_approve", ResumeInput{}); err == nil {
		t.Fatal("expected approval-required error")
	} else {
		var werr wrkrerrors.WrkrError
		if !errors.As(err, &werr) || werr.Code != wrkrerrors.ECheckpointApprovalRequired {
			t.Fatalf("expected E_CHECKPOINT_APPROVAL_REQUIRED, got %v", err)
		}
	}

	if _, err := r.ApproveCheckpoint("job_approve", cp.CheckpointID, "looks good", "manager"); err != nil {
		t.Fatalf("approve checkpoint: %v", err)
	}

	state, err := r.Resume("job_approve", ResumeInput{})
	if err != nil {
		t.Fatalf("resume after approval: %v", err)
	}
	if state.Status != queue.StatusRunning {
		t.Fatalf("expected running status after resume, got %s", state.Status)
	}
}

func TestResumeBlocksOnEnvMismatchUnlessOverridden(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 14, 0, 0, 0, time.UTC)
	r := testRunner(t, now)

	if _, err := r.InitJob("job_env"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_env", queue.StatusRunning); err != nil {
		t.Fatalf("running: %v", err)
	}
	if _, err := r.ChangeStatus("job_env", queue.StatusPaused); err != nil {
		t.Fatalf("pause: %v", err)
	}

	if _, err := r.store.AppendEvent("job_env", eventEnvFingerprintSet, map[string]any{
		"rules":       []string{"os"},
		"values":      map[string]string{"os": "bogus-os"},
		"hash":        "deadbeef",
		"captured_at": now.UTC(),
	}, now); err != nil {
		t.Fatalf("inject mismatch fingerprint: %v", err)
	}

	if _, err := r.Resume("job_env", ResumeInput{}); err == nil {
		t.Fatal("expected env mismatch error")
	} else {
		var werr wrkrerrors.WrkrError
		if !errors.As(err, &werr) || werr.Code != wrkrerrors.EEnvFingerprintMismatch {
			t.Fatalf("expected E_ENV_FINGERPRINT_MISMATCH, got %v", err)
		}
	}

	state, err := r.Resume("job_env", ResumeInput{OverrideEnvMismatch: true, OverrideReason: "known workstation drift", ApprovedBy: "david"})
	if err != nil {
		t.Fatalf("resume with override: %v", err)
	}
	if state.Status != queue.StatusRunning {
		t.Fatalf("expected running status after override, got %s", state.Status)
	}
	if state.EnvFingerprintHash == "deadbeef" {
		t.Fatal("expected override to persist current fingerprint hash")
	}
}

func TestResumeBlocksOnEnvMismatchFromBlockedDecision(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 14, 0, 0, 0, time.UTC)
	r := testRunner(t, now)

	if _, err := r.InitJob("job_env_blocked_decision"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_env_blocked_decision", queue.StatusRunning); err != nil {
		t.Fatalf("running: %v", err)
	}
	if _, err := r.ChangeStatus("job_env_blocked_decision", queue.StatusBlockedDecision); err != nil {
		t.Fatalf("blocked decision: %v", err)
	}

	if _, err := r.store.AppendEvent("job_env_blocked_decision", eventEnvFingerprintSet, map[string]any{
		"rules":       []string{"os"},
		"values":      map[string]string{"os": "bogus-os"},
		"hash":        "deadbeef",
		"captured_at": now.UTC(),
	}, now); err != nil {
		t.Fatalf("inject mismatch fingerprint: %v", err)
	}

	if _, err := r.Resume("job_env_blocked_decision", ResumeInput{}); err == nil {
		t.Fatal("expected env mismatch error")
	} else {
		var werr wrkrerrors.WrkrError
		if !errors.As(err, &werr) || werr.Code != wrkrerrors.EEnvFingerprintMismatch {
			t.Fatalf("expected E_ENV_FINGERPRINT_MISMATCH, got %v", err)
		}
	}

	state, err := r.Recover("job_env_blocked_decision")
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if state.Status != queue.StatusBlockedError {
		t.Fatalf("expected blocked_error status, got %s", state.Status)
	}
}

func TestEmitCheckpointRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 14, 0, 0, 0, time.UTC)
	r := testRunner(t, now)

	if _, err := r.InitJob("job_invalid_cp_status"); err != nil {
		t.Fatalf("init: %v", err)
	}
	_, err := r.EmitCheckpoint("job_invalid_cp_status", CheckpointInput{
		Type:    "progress",
		Summary: "bad status should fail",
		Status:  queue.Status("not-real"),
	})
	if err == nil {
		t.Fatal("expected invalid status error")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EInvalidInputSchema {
		t.Fatalf("expected E_INVALID_INPUT_SCHEMA, got %v", err)
	}
}
