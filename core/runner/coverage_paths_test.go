package runner

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/envfp"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/lease"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/store"
)

func TestNewWithNilStoreUsesDefaultRoot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r, err := New(nil, Options{
		Now:      func() time.Time { return time.Date(2026, 2, 14, 8, 0, 0, 0, time.UTC) },
		LeaseTTL: 15 * time.Second,
	})
	if err != nil {
		t.Fatalf("New with nil store: %v", err)
	}
	if r.store == nil {
		t.Fatal("expected runner store to be initialized")
	}
}

func TestHeartbeatLeaseAndReleaseConflictPaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 8, 5, 0, 0, time.UTC)
	r := testRunner(t, now)

	if _, err := r.InitJob("job_lease_paths"); err != nil {
		t.Fatalf("InitJob: %v", err)
	}
	if _, err := r.AcquireLease("job_lease_paths", "worker-a", "lease-a"); err != nil {
		t.Fatalf("AcquireLease: %v", err)
	}

	state, err := r.HeartbeatLease("job_lease_paths", "worker-a", "lease-a")
	if err != nil {
		t.Fatalf("HeartbeatLease: %v", err)
	}
	if state.Lease == nil || state.Lease.WorkerID != "worker-a" {
		t.Fatalf("expected lease owned by worker-a, got %+v", state.Lease)
	}

	if _, err := r.HeartbeatLease("job_lease_paths", "worker-b", "lease-b"); err == nil {
		t.Fatal("expected lease conflict heartbeat error")
	} else {
		var werr wrkrerrors.WrkrError
		if !errors.As(err, &werr) || werr.Code != wrkrerrors.ELeaseConflict {
			t.Fatalf("expected E_LEASE_CONFLICT from heartbeat, got %v", err)
		}
	}

	if _, err := r.ReleaseLease("job_lease_paths", "worker-b", "lease-b"); err == nil {
		t.Fatal("expected lease conflict release error")
	} else {
		var werr wrkrerrors.WrkrError
		if !errors.As(err, &werr) || werr.Code != wrkrerrors.ELeaseConflict {
			t.Fatalf("expected E_LEASE_CONFLICT from release, got %v", err)
		}
	}
}

func TestApplyEventCoversEventTypes(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 2, 14, 8, 10, 0, 0, time.UTC)
	state := defaultState("job_apply")

	startedAt := base.Add(-2 * time.Minute)
	jobInitPayload, _ := json.Marshal(map[string]any{"started_at": startedAt})
	if err := applyEvent(&state, store.Event{Type: eventJobInitialized, Payload: jobInitPayload}); err != nil {
		t.Fatalf("apply job_initialized: %v", err)
	}
	if state.StartedAt == nil {
		t.Fatal("expected started_at to be set")
	}

	statusPayload, _ := json.Marshal(map[string]any{"to": queue.StatusRunning})
	if err := applyEvent(&state, store.Event{Type: eventStatusChanged, Payload: statusPayload}); err != nil {
		t.Fatalf("apply status_changed: %v", err)
	}
	if state.Status != queue.StatusRunning {
		t.Fatalf("expected running status, got %s", state.Status)
	}

	countersPayload, _ := json.Marshal(map[string]any{
		"retry_count":     2,
		"step_count":      3,
		"tool_call_count": 4,
	})
	if err := applyEvent(&state, store.Event{Type: eventCountersUpdated, Payload: countersPayload}); err != nil {
		t.Fatalf("apply counters_updated: %v", err)
	}
	if state.RetryCount != 2 || state.StepCount != 3 || state.ToolCallCount != 4 {
		t.Fatalf("unexpected counters: %+v", state)
	}

	idempotencyPayload, _ := json.Marshal(map[string]any{"key": "step-1"})
	if err := applyEvent(&state, store.Event{Type: eventIdempotencyRecorded, Payload: idempotencyPayload}); err != nil {
		t.Fatalf("apply idempotency: %v", err)
	}
	if !state.IdempotencyKeys["step-1"] {
		t.Fatal("expected idempotency key to be recorded")
	}

	leasePayload, _ := json.Marshal(lease.Record{
		WorkerID:   "worker-a",
		LeaseID:    "lease-a",
		AcquiredAt: base.UTC(),
		ExpiresAt:  base.Add(30 * time.Second).UTC(),
	})
	if err := applyEvent(&state, store.Event{Type: eventLeaseSet, Payload: leasePayload}); err != nil {
		t.Fatalf("apply lease_set: %v", err)
	}
	if state.Lease == nil || state.Lease.LeaseID != "lease-a" {
		t.Fatalf("expected lease to be set, got %+v", state.Lease)
	}
	if err := applyEvent(&state, store.Event{Type: eventLeaseReleased}); err != nil {
		t.Fatalf("apply lease_released: %v", err)
	}
	if state.Lease != nil {
		t.Fatalf("expected lease to be cleared, got %+v", state.Lease)
	}

	checkpointPayload, _ := json.Marshal(map[string]any{"reason_codes": []string{"E_BUDGET_EXCEEDED"}})
	if err := applyEvent(&state, store.Event{Type: eventCheckpointEmitted, Payload: checkpointPayload}); err != nil {
		t.Fatalf("apply checkpoint_emitted: %v", err)
	}
	if len(state.LastReasonCodes) != 1 || state.LastReasonCodes[0] != "E_BUDGET_EXCEEDED" {
		t.Fatalf("unexpected reason codes: %+v", state.LastReasonCodes)
	}

	fp := envfp.Fingerprint{
		Rules:      []string{"go_version"},
		Values:     map[string]string{"go_version": "go1.25.7"},
		Hash:       "hash-a",
		CapturedAt: base.UTC(),
	}
	fpPayload, _ := json.Marshal(fp)
	if err := applyEvent(&state, store.Event{Type: eventEnvFingerprintSet, Payload: fpPayload}); err != nil {
		t.Fatalf("apply env_fingerprint_set: %v", err)
	}
	if state.EnvFingerprintHash != "hash-a" {
		t.Fatalf("expected env hash to be updated, got %s", state.EnvFingerprintHash)
	}

	overridePayload, _ := json.Marshal(map[string]any{
		"actual_hash": "hash-b",
		"rules":       []string{"go_version"},
		"values":      map[string]string{"go_version": "go1.25.8"},
	})
	if err := applyEvent(&state, store.Event{Type: eventEnvOverrideRecorded, Payload: overridePayload}); err != nil {
		t.Fatalf("apply env_override_recorded: %v", err)
	}
	if state.EnvFingerprintHash != "hash-b" {
		t.Fatalf("expected override env hash, got %s", state.EnvFingerprintHash)
	}

	if err := applyEvent(&state, store.Event{Type: eventApprovalRecorded}); err != nil {
		t.Fatalf("apply approval_recorded: %v", err)
	}
	if err := applyEvent(&state, store.Event{Type: eventAdapterStep}); err != nil {
		t.Fatalf("apply adapter_step: %v", err)
	}
}

func TestApplyEventDecodeAndUnknownErrors(t *testing.T) {
	t.Parallel()

	state := defaultState("job_apply_err")
	cases := []store.Event{
		{Type: eventStatusChanged, Payload: []byte("{")},
		{Type: eventCountersUpdated, Payload: []byte("{")},
		{Type: eventIdempotencyRecorded, Payload: []byte("{")},
		{Type: eventLeaseSet, Payload: []byte("{")},
		{Type: eventCheckpointEmitted, Payload: []byte("{")},
		{Type: eventEnvFingerprintSet, Payload: []byte("{")},
		{Type: eventEnvOverrideRecorded, Payload: []byte("{")},
		{Type: "unexpected_event_type"},
	}
	for _, evt := range cases {
		if err := applyEvent(&state, evt); err == nil {
			t.Fatalf("expected applyEvent error for type=%s", evt.Type)
		}
	}
}
