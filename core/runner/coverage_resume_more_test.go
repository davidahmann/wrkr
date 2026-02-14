package runner

import (
	"errors"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
)

func injectEnvFingerprint(t *testing.T, r *Runner, jobID, hash string, now time.Time) {
	t.Helper()
	if _, err := r.store.AppendEvent(jobID, eventEnvFingerprintSet, map[string]any{
		"rules":       []string{"os"},
		"values":      map[string]string{"os": "bogus-os"},
		"hash":        hash,
		"captured_at": now.UTC(),
	}, now); err != nil {
		t.Fatalf("inject env fingerprint: %v", err)
	}
}

func TestResumeCoverageMorePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 23, 0, 0, 0, time.UTC)
	r := testRunner(t, now)

	if _, err := r.InitJob("job_resume_blocked_error"); err != nil {
		t.Fatalf("InitJob: %v", err)
	}
	if _, err := r.ChangeStatus("job_resume_blocked_error", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus running: %v", err)
	}
	if _, err := r.ChangeStatus("job_resume_blocked_error", queue.StatusBlockedError); err != nil {
		t.Fatalf("ChangeStatus blocked_error: %v", err)
	}
	injectEnvFingerprint(t, r, "job_resume_blocked_error", "deadbeef", now)
	_, err := r.Resume("job_resume_blocked_error", ResumeInput{})
	if err == nil {
		t.Fatal("expected env mismatch error")
	}
	var mismatch wrkrerrors.WrkrError
	if !errors.As(err, &mismatch) || mismatch.Code != wrkrerrors.EEnvFingerprintMismatch {
		t.Fatalf("expected E_ENV_FINGERPRINT_MISMATCH, got %v", err)
	}

	if _, err := r.InitJob("job_resume_override_gate"); err != nil {
		t.Fatalf("InitJob override gate: %v", err)
	}
	if _, err := r.ChangeStatus("job_resume_override_gate", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus override gate running: %v", err)
	}
	decision, err := r.EmitCheckpoint("job_resume_override_gate", CheckpointInput{
		Type:    "decision-needed",
		Summary: "approve before resume",
		Status:  queue.StatusBlockedDecision,
		RequiredAction: &v1.RequiredAction{
			Kind:         "approval",
			Instructions: "approve",
		},
	})
	if err != nil {
		t.Fatalf("EmitCheckpoint decision-needed: %v", err)
	}
	if _, err := r.ChangeStatus("job_resume_override_gate", queue.StatusBlockedDecision); err != nil {
		t.Fatalf("ChangeStatus blocked_decision: %v", err)
	}
	injectEnvFingerprint(t, r, "job_resume_override_gate", "deadbeef", now)

	_, err = r.Resume("job_resume_override_gate", ResumeInput{
		OverrideEnvMismatch: true,
		OverrideReason:      "known drift",
		ApprovedBy:          "lead",
	})
	if err == nil {
		t.Fatal("expected approval-required error after env override without approval")
	}
	var approvalErr wrkrerrors.WrkrError
	if !errors.As(err, &approvalErr) || approvalErr.Code != wrkrerrors.ECheckpointApprovalRequired {
		t.Fatalf("expected E_CHECKPOINT_APPROVAL_REQUIRED, got %v", err)
	}

	if _, err := r.ApproveCheckpoint("job_resume_override_gate", decision.CheckpointID, "ok", "lead"); err != nil {
		t.Fatalf("ApproveCheckpoint: %v", err)
	}
	state, err := r.Resume("job_resume_override_gate", ResumeInput{
		OverrideEnvMismatch: true,
		OverrideReason:      "known drift",
		ApprovedBy:          "lead",
	})
	if err != nil {
		t.Fatalf("Resume with approval: %v", err)
	}
	if state.Status != queue.StatusRunning {
		t.Fatalf("expected running after resume, got %s", state.Status)
	}
}

