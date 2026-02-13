package status

import (
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/lease"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
)

func TestFromRunnerStateIncludesLease(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	state := &runner.State{
		JobID:  "job_1",
		Status: queue.StatusRunning,
		Lease: &lease.Record{
			WorkerID:  "worker-a",
			LeaseID:   "lease-a",
			ExpiresAt: now.Add(30 * time.Second),
		},
	}

	resp := FromRunnerState(state, "dev", now)
	if resp.SchemaID != "wrkr.status_response" {
		t.Fatalf("unexpected schema id: %s", resp.SchemaID)
	}
	if resp.Lease == nil || resp.Lease.WorkerID != "worker-a" {
		t.Fatalf("expected lease details in status response, got %+v", resp.Lease)
	}
}
