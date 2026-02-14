package runner

import (
	"errors"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/budget"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
)

func TestRunnerHelperCoveragePaths(t *testing.T) {
	t.Parallel()

	if _, err := parseCheckpointID("bad"); err == nil {
		t.Fatal("expected invalid checkpoint id error")
	}
	if seq, err := parseCheckpointID("cp_3"); err != nil || seq != 3 {
		t.Fatalf("expected cp_3 parse success, got seq=%d err=%v", seq, err)
	}
	if !isCheckpointType("blocked") || isCheckpointType("other") {
		t.Fatal("unexpected checkpoint type classifier behavior")
	}

	event := store.Event{
		Seq:       10,
		CreatedAt: time.Date(2026, 2, 14, 17, 0, 0, 0, time.UTC),
		Type:      eventCheckpointEmitted,
		Payload:   []byte(`{"type":"progress","summary":"ok","status":"running"}`),
	}
	cp, err := checkpointFromEvent("job_x", event)
	if err != nil {
		t.Fatalf("checkpointFromEvent: %v", err)
	}
	if cp.CheckpointID != "cp_10" || cp.Type != "progress" {
		t.Fatalf("unexpected checkpoint: %+v", cp)
	}

	event.Payload = []byte("{bad")
	if _, err := checkpointFromEvent("job_x", event); err == nil {
		t.Fatal("expected checkpoint payload decode error")
	}
}

func TestRunnerApprovalAndCheckpointCoveragePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 17, 10, 0, 0, time.UTC)
	r := testRunner(t, now)

	if _, err := r.InitJob("job_runner_cov"); err != nil {
		t.Fatalf("InitJob: %v", err)
	}
	if _, err := r.ChangeStatus("job_runner_cov", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus running: %v", err)
	}

	if _, err := r.GetCheckpoint("job_runner_cov", "bad"); err == nil {
		t.Fatal("expected invalid checkpoint id path")
	}
	if _, err := r.GetCheckpoint("job_runner_cov", "cp_99"); err == nil {
		t.Fatal("expected checkpoint not found path")
	}

	progressCP, err := r.EmitCheckpoint("job_runner_cov", CheckpointInput{
		Type:    "progress",
		Summary: "progress",
	})
	if err != nil {
		t.Fatalf("EmitCheckpoint progress: %v", err)
	}
	if _, err := r.ApproveCheckpoint("job_runner_cov", progressCP.CheckpointID, "ok", "lead"); err == nil {
		t.Fatal("expected non decision checkpoint approval failure")
	}

	decisionCP, err := r.EmitCheckpoint("job_runner_cov", CheckpointInput{
		Type:    "decision-needed",
		Summary: "approve",
		Status:  queue.StatusBlockedDecision,
		RequiredAction: &v1.RequiredAction{
			Kind:         "approval",
			Instructions: "approve",
		},
	})
	if err != nil {
		t.Fatalf("EmitCheckpoint decision-needed: %v", err)
	}
	if _, err := r.ApproveCheckpoint("job_runner_cov", decisionCP.CheckpointID, "", "lead"); err == nil {
		t.Fatal("expected empty reason approval failure")
	}
	if _, err := r.ApproveCheckpoint("job_runner_cov", decisionCP.CheckpointID, "ok", ""); err == nil {
		t.Fatal("expected empty approved_by approval failure")
	}
	if _, err := r.ApproveCheckpoint("job_runner_cov", decisionCP.CheckpointID, "ok", "lead"); err != nil {
		t.Fatalf("ApproveCheckpoint success: %v", err)
	}
}

func TestRunnerBudgetAndResumeCoveragePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 17, 20, 0, 0, time.UTC)
	r := testRunner(t, now)

	if _, err := r.InitJob("job_runner_resume_cov"); err != nil {
		t.Fatalf("InitJob: %v", err)
	}
	if _, err := r.ChangeStatus("job_runner_resume_cov", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus running: %v", err)
	}
	if cp, err := r.CheckBudget("job_runner_resume_cov", budget.Limits{MaxStepCount: 100}); err != nil || cp != nil {
		t.Fatalf("expected budget within limits, cp=%+v err=%v", cp, err)
	}

	if _, err := r.ChangeStatus("job_runner_resume_cov", queue.StatusPaused); err != nil {
		t.Fatalf("ChangeStatus paused: %v", err)
	}
	state, err := r.Resume("job_runner_resume_cov", ResumeInput{})
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if state.Status != queue.StatusRunning {
		t.Fatalf("expected running status after resume, got %s", state.Status)
	}
	if state.EnvFingerprintHash == "" || len(state.EnvFingerprintRules) == 0 {
		t.Fatalf("expected env fingerprint to be set during resume: %+v", state)
	}
}

func TestRunnerListApprovalsDecodeErrorPath(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 17, 30, 0, 0, time.UTC)
	r := testRunner(t, now)
	if _, err := r.InitJob("job_runner_bad_approval"); err != nil {
		t.Fatalf("InitJob: %v", err)
	}
	if _, err := r.store.AppendEvent("job_runner_bad_approval", eventApprovalRecorded, "bad", now); err != nil {
		t.Fatalf("AppendEvent malformed approval payload: %v", err)
	}
	if _, err := r.ListApprovals("job_runner_bad_approval"); err == nil {
		t.Fatal("expected approval decode error")
	}
}

func TestRunnerResumeApprovalRequiredCode(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 17, 40, 0, 0, time.UTC)
	r := testRunner(t, now)
	if _, err := r.InitJob("job_runner_approval_required"); err != nil {
		t.Fatalf("InitJob: %v", err)
	}
	if _, err := r.ChangeStatus("job_runner_approval_required", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus running: %v", err)
	}
	if _, err := r.EmitCheckpoint("job_runner_approval_required", CheckpointInput{
		Type:    "decision-needed",
		Summary: "needs approval",
		Status:  queue.StatusBlockedDecision,
		RequiredAction: &v1.RequiredAction{
			Kind:         "approval",
			Instructions: "approve",
		},
	}); err != nil {
		t.Fatalf("EmitCheckpoint decision-needed: %v", err)
	}
	if _, err := r.ChangeStatus("job_runner_approval_required", queue.StatusBlockedDecision); err != nil {
		t.Fatalf("ChangeStatus blocked_decision: %v", err)
	}
	_, err := r.Resume("job_runner_approval_required", ResumeInput{})
	if err == nil {
		t.Fatal("expected approval required error")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.ECheckpointApprovalRequired {
		t.Fatalf("expected E_CHECKPOINT_APPROVAL_REQUIRED, got %v", err)
	}
}
