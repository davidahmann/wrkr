package runner

import (
	"errors"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/store"
)

func testRunner(t *testing.T, now time.Time) *Runner {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := New(s, Options{Now: func() time.Time { return now }, LeaseTTL: 30 * time.Second})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	return r
}

func TestRecoverPreservesCountersAndIdempotencyAcrossRestart(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	r1 := testRunner(t, now)

	if _, err := r1.InitJob("job_1"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r1.ChangeStatus("job_1", queue.StatusRunning); err != nil {
		t.Fatalf("change status: %v", err)
	}
	if _, err := r1.UpdateCounters("job_1", 1, 4, 7); err != nil {
		t.Fatalf("update counters: %v", err)
	}
	if _, err := r1.RecordIdempotencyKey("job_1", "step-abc"); err != nil {
		t.Fatalf("record idempotency key: %v", err)
	}

	store2, err := store.New(r1.store.Root())
	if err != nil {
		t.Fatalf("store2.New: %v", err)
	}
	r2, err := New(store2, Options{Now: func() time.Time { return now.Add(time.Minute) }, LeaseTTL: 30 * time.Second})
	if err != nil {
		t.Fatalf("runner2.New: %v", err)
	}

	state, err := r2.Recover("job_1")
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if state.Status != queue.StatusRunning {
		t.Fatalf("expected running status, got %s", state.Status)
	}
	if state.RetryCount != 1 || state.StepCount != 4 || state.ToolCallCount != 7 {
		t.Fatalf("unexpected counters: %+v", state)
	}
	if !state.IdempotencyKeys["step-abc"] {
		t.Fatal("idempotency key missing after recovery")
	}
}

func TestInvalidStatusTransitionReturnsStableReasonCode(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	r := testRunner(t, now)

	if _, err := r.InitJob("job_2"); err != nil {
		t.Fatalf("init: %v", err)
	}
	_, err := r.ChangeStatus("job_2", queue.StatusCompleted)
	if err == nil {
		t.Fatal("expected invalid transition error")
	}

	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) {
		t.Fatalf("expected WrkrError, got %T", err)
	}
	if werr.Code != wrkrerrors.EInvalidStateTransition {
		t.Fatalf("expected E_INVALID_STATE_TRANSITION, got %s", werr.Code)
	}
}

func TestLeaseConflictAndExpiryClaim(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	current := base
	r := testRunner(t, current)
	r.now = func() time.Time { return current }

	if _, err := r.InitJob("job_3"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.AcquireLease("job_3", "worker-a", "lease-a"); err != nil {
		t.Fatalf("acquire first lease: %v", err)
	}

	_, err := r.AcquireLease("job_3", "worker-b", "lease-b")
	if err == nil {
		t.Fatal("expected lease conflict")
	}

	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) {
		t.Fatalf("expected WrkrError, got %T", err)
	}
	if werr.Code != wrkrerrors.ELeaseConflict {
		t.Fatalf("expected E_LEASE_CONFLICT, got %s", werr.Code)
	}

	current = current.Add(31 * time.Second)
	state, err := r.AcquireLease("job_3", "worker-b", "lease-b")
	if err != nil {
		t.Fatalf("expected acquire after ttl expiry: %v", err)
	}
	if state.Lease == nil || state.Lease.WorkerID != "worker-b" {
		t.Fatalf("expected worker-b lease after expiry, got %+v", state.Lease)
	}
}
