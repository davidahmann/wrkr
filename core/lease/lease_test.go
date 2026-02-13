package lease

import (
	"testing"
	"time"
)

func TestAcquireRejectsActiveLeaseFromAnotherWorker(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	current := &Record{
		WorkerID:    "worker-a",
		LeaseID:     "lease-a",
		AcquiredAt:  now,
		HeartbeatAt: now,
		ExpiresAt:   now.Add(30 * time.Second),
	}

	_, err := Acquire(current, "worker-b", "lease-b", now, 30*time.Second)
	if err == nil {
		t.Fatal("expected lease conflict")
	}
}

func TestAcquireAfterExpiry(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	current := &Record{
		WorkerID:    "worker-a",
		LeaseID:     "lease-a",
		AcquiredAt:  now.Add(-2 * time.Minute),
		HeartbeatAt: now.Add(-2 * time.Minute),
		ExpiresAt:   now.Add(-1 * time.Minute),
	}

	rec, err := Acquire(current, "worker-b", "lease-b", now, 30*time.Second)
	if err != nil {
		t.Fatalf("expected acquire success after expiry: %v", err)
	}
	if rec.WorkerID != "worker-b" {
		t.Fatalf("expected worker-b, got %s", rec.WorkerID)
	}
}
