package lease

import (
	"errors"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func TestHeartbeatCoveragePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 9, 30, 0, 0, time.UTC)

	if _, err := Heartbeat(nil, "worker-a", "lease-a", now, 30*time.Second); err == nil {
		t.Fatal("expected nil lease heartbeat to fail")
	}

	current := &Record{
		WorkerID:    "worker-a",
		LeaseID:     "lease-a",
		AcquiredAt:  now.Add(-time.Minute),
		HeartbeatAt: now.Add(-time.Minute),
		ExpiresAt:   now.Add(time.Minute),
	}

	_, err := Heartbeat(current, "worker-b", "lease-a", now, 30*time.Second)
	if err == nil {
		t.Fatal("expected heartbeat mismatch to fail")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.ELeaseConflict {
		t.Fatalf("expected E_LEASE_CONFLICT, got %v", err)
	}

	updated, err := Heartbeat(current, "worker-a", "lease-a", now, 45*time.Second)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if !updated.HeartbeatAt.Equal(now.UTC()) {
		t.Fatalf("expected heartbeat timestamp update, got %s", updated.HeartbeatAt)
	}
	if !updated.ExpiresAt.Equal(now.UTC().Add(45 * time.Second)) {
		t.Fatalf("expected ttl extension, got %s", updated.ExpiresAt)
	}
}

func TestIsExpiredCoveragePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC)
	if !IsExpired(nil, now) {
		t.Fatal("expected nil lease to be expired")
	}

	rec := &Record{ExpiresAt: now.Add(time.Second)}
	if IsExpired(rec, now) {
		t.Fatal("expected future lease to be active")
	}

	rec.ExpiresAt = now
	if !IsExpired(rec, now) {
		t.Fatal("expected boundary lease to be expired")
	}
}

