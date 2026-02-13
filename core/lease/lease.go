package lease

import (
	"fmt"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

type Record struct {
	WorkerID    string    `json:"worker_id"`
	LeaseID     string    `json:"lease_id"`
	AcquiredAt  time.Time `json:"acquired_at"`
	HeartbeatAt time.Time `json:"heartbeat_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func Acquire(current *Record, workerID, leaseID string, now time.Time, ttl time.Duration) (*Record, error) {
	if current != nil && current.ExpiresAt.After(now) {
		if current.WorkerID != workerID || current.LeaseID != leaseID {
			return nil, wrkrerrors.New(
				wrkrerrors.ELeaseConflict,
				"job lease already held by another worker",
				map[string]any{
					"worker_id":          workerID,
					"existing_worker_id": current.WorkerID,
					"existing_lease_id":  current.LeaseID,
				},
			)
		}
	}

	rec := &Record{
		WorkerID:    workerID,
		LeaseID:     leaseID,
		AcquiredAt:  now.UTC(),
		HeartbeatAt: now.UTC(),
		ExpiresAt:   now.UTC().Add(ttl),
	}
	return rec, nil
}

func Heartbeat(current *Record, workerID, leaseID string, now time.Time, ttl time.Duration) (*Record, error) {
	if current == nil {
		return nil, fmt.Errorf("no active lease")
	}
	if current.WorkerID != workerID || current.LeaseID != leaseID {
		return nil, wrkrerrors.New(
			wrkrerrors.ELeaseConflict,
			"heartbeat lease mismatch",
			map[string]any{"worker_id": workerID, "lease_id": leaseID},
		)
	}

	updated := *current
	updated.HeartbeatAt = now.UTC()
	updated.ExpiresAt = now.UTC().Add(ttl)
	return &updated, nil
}

func IsExpired(current *Record, now time.Time) bool {
	if current == nil {
		return true
	}
	return !current.ExpiresAt.After(now)
}
