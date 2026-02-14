package runner

import (
	"errors"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
)

func TestRunnerLeaseCoveragePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 21, 0, 0, 0, time.UTC)
	r := testRunner(t, now)
	if _, err := r.InitJob("job_runner_lease_cov"); err != nil {
		t.Fatalf("InitJob: %v", err)
	}
	if _, err := r.ChangeStatus("job_runner_lease_cov", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus running: %v", err)
	}

	if _, err := r.HeartbeatLease("job_runner_lease_cov", "worker", "lease"); err == nil {
		t.Fatal("expected heartbeat without lease to fail")
	}

	if _, err := r.AcquireLease("job_runner_lease_cov", "worker-a", "lease-a"); err != nil {
		t.Fatalf("AcquireLease: %v", err)
	}
	if _, err := r.HeartbeatLease("job_runner_lease_cov", "worker-b", "lease-a"); err == nil {
		t.Fatal("expected heartbeat lease mismatch failure")
	}
	if _, err := r.ReleaseLease("job_runner_lease_cov", "worker-b", "lease-a"); err == nil {
		t.Fatal("expected release lease mismatch failure")
	}
	if _, err := r.ReleaseLease("job_runner_lease_cov", "worker-a", "lease-a"); err != nil {
		t.Fatalf("ReleaseLease: %v", err)
	}
	state, err := r.ReleaseLease("job_runner_lease_cov", "worker-a", "lease-a")
	if err != nil {
		t.Fatalf("ReleaseLease idempotent path: %v", err)
	}
	if state.Lease != nil {
		t.Fatalf("expected nil lease after release, got %+v", state.Lease)
	}
}

func TestEmitCheckpointValidationCoveragePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 21, 10, 0, 0, time.UTC)
	r := testRunner(t, now)
	if _, err := r.InitJob("job_runner_emit_cov"); err != nil {
		t.Fatalf("InitJob: %v", err)
	}

	if _, err := r.EmitCheckpoint("job_runner_emit_cov", CheckpointInput{Type: "bad", Summary: "ok"}); err == nil {
		t.Fatal("expected invalid checkpoint type error")
	}
	if _, err := r.EmitCheckpoint("job_runner_emit_cov", CheckpointInput{Type: "plan", Summary: ""}); err == nil {
		t.Fatal("expected empty checkpoint summary error")
	}
	if _, err := r.EmitCheckpoint("job_runner_emit_cov", CheckpointInput{Type: "decision-needed", Summary: "approve"}); err == nil {
		t.Fatal("expected decision-needed required_action validation error")
	}

	_, err := r.EmitCheckpoint("job_runner_emit_cov", CheckpointInput{
		Type:    "progress",
		Summary: "valid",
		Status:  queue.Status("not-a-status"),
	})
	if err == nil {
		t.Fatal("expected invalid checkpoint status error")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EInvalidInputSchema {
		t.Fatalf("expected E_INVALID_INPUT_SCHEMA, got %v", err)
	}
}

