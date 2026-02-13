package runner

import (
	"encoding/json"
	"fmt"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/lease"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/store"
)

const (
	eventJobInitialized      = "job_initialized"
	eventStatusChanged       = "status_changed"
	eventCountersUpdated     = "counters_updated"
	eventIdempotencyRecorded = "idempotency_recorded"
	eventLeaseSet            = "lease_set"
)

type State struct {
	JobID           string          `json:"job_id"`
	Status          queue.Status    `json:"status"`
	RetryCount      int             `json:"retry_count"`
	StepCount       int             `json:"step_count"`
	ToolCallCount   int             `json:"tool_call_count"`
	IdempotencyKeys map[string]bool `json:"idempotency_keys"`
	Lease           *lease.Record   `json:"lease,omitempty"`
	LastAppliedSeq  int64           `json:"last_applied_seq"`
}

type Options struct {
	Now      func() time.Time
	LeaseTTL time.Duration
}

type Runner struct {
	store    *store.LocalStore
	now      func() time.Time
	leaseTTL time.Duration
}

func New(s *store.LocalStore, opts Options) (*Runner, error) {
	if s == nil {
		var err error
		s, err = store.New("")
		if err != nil {
			return nil, err
		}
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	leaseTTL := opts.LeaseTTL
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Second
	}

	return &Runner{store: s, now: now, leaseTTL: leaseTTL}, nil
}

func defaultState(jobID string) State {
	return State{
		JobID:           jobID,
		Status:          queue.StatusQueued,
		IdempotencyKeys: map[string]bool{},
	}
}

func (r *Runner) InitJob(jobID string) (*State, error) {
	state := defaultState(jobID)
	event, err := r.store.AppendEvent(jobID, eventJobInitialized, map[string]any{"status": state.Status}, r.now())
	if err != nil {
		return nil, err
	}
	state.LastAppliedSeq = event.Seq
	if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *Runner) Recover(jobID string) (*State, error) {
	state := defaultState(jobID)

	snap, err := r.store.LoadSnapshot(jobID)
	if err != nil {
		return nil, err
	}
	if snap != nil && len(snap.State) > 0 {
		if err := json.Unmarshal(snap.State, &state); err != nil {
			return nil, wrkrerrors.New(
				wrkrerrors.EStoreCorrupt,
				"invalid snapshot payload",
				map[string]any{"job_id": jobID},
			)
		}
		state.LastAppliedSeq = snap.LastSeq
	}
	if state.IdempotencyKeys == nil {
		state.IdempotencyKeys = map[string]bool{}
	}

	events, err := r.store.LoadEvents(jobID)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.Seq <= state.LastAppliedSeq {
			continue
		}
		if err := applyEvent(&state, event); err != nil {
			return nil, err
		}
		state.LastAppliedSeq = event.Seq
	}

	state.JobID = jobID
	return &state, nil
}

func (r *Runner) ChangeStatus(jobID string, to queue.Status) (*State, error) {
	state, err := r.Recover(jobID)
	if err != nil {
		return nil, err
	}
	if err := queue.ValidateTransition(state.Status, to); err != nil {
		return nil, err
	}

	event, err := r.store.AppendEvent(
		jobID,
		eventStatusChanged,
		map[string]any{"from": state.Status, "to": to},
		r.now(),
	)
	if err != nil {
		return nil, err
	}

	state.Status = to
	state.LastAppliedSeq = event.Seq
	if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
		return nil, err
	}
	return state, nil
}

func (r *Runner) UpdateCounters(jobID string, retryCount, stepCount, toolCallCount int) (*State, error) {
	state, err := r.Recover(jobID)
	if err != nil {
		return nil, err
	}

	event, err := r.store.AppendEvent(
		jobID,
		eventCountersUpdated,
		map[string]any{
			"retry_count":     retryCount,
			"step_count":      stepCount,
			"tool_call_count": toolCallCount,
		},
		r.now(),
	)
	if err != nil {
		return nil, err
	}

	state.RetryCount = retryCount
	state.StepCount = stepCount
	state.ToolCallCount = toolCallCount
	state.LastAppliedSeq = event.Seq
	if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
		return nil, err
	}
	return state, nil
}

func (r *Runner) RecordIdempotencyKey(jobID, key string) (*State, error) {
	state, err := r.Recover(jobID)
	if err != nil {
		return nil, err
	}

	event, err := r.store.AppendEvent(jobID, eventIdempotencyRecorded, map[string]any{"key": key}, r.now())
	if err != nil {
		return nil, err
	}
	state.IdempotencyKeys[key] = true
	state.LastAppliedSeq = event.Seq
	if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
		return nil, err
	}
	return state, nil
}

func (r *Runner) AcquireLease(jobID, workerID, leaseID string) (*State, error) {
	state, err := r.Recover(jobID)
	if err != nil {
		return nil, err
	}

	rec, err := lease.Acquire(state.Lease, workerID, leaseID, r.now(), r.leaseTTL)
	if err != nil {
		return nil, err
	}

	event, err := r.store.AppendEvent(jobID, eventLeaseSet, rec, r.now())
	if err != nil {
		return nil, err
	}
	state.Lease = rec
	state.LastAppliedSeq = event.Seq
	if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
		return nil, err
	}

	return state, nil
}

func (r *Runner) HeartbeatLease(jobID, workerID, leaseID string) (*State, error) {
	state, err := r.Recover(jobID)
	if err != nil {
		return nil, err
	}

	rec, err := lease.Heartbeat(state.Lease, workerID, leaseID, r.now(), r.leaseTTL)
	if err != nil {
		return nil, err
	}

	event, err := r.store.AppendEvent(jobID, eventLeaseSet, rec, r.now())
	if err != nil {
		return nil, err
	}

	state.Lease = rec
	state.LastAppliedSeq = event.Seq
	if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
		return nil, err
	}
	return state, nil
}

func applyEvent(state *State, event store.Event) error {
	switch event.Type {
	case eventJobInitialized:
		state.Status = queue.StatusQueued
		return nil
	case eventStatusChanged:
		var payload struct {
			To queue.Status `json:"to"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode status_changed payload: %w", err)
		}
		state.Status = payload.To
		return nil
	case eventCountersUpdated:
		var payload struct {
			RetryCount    int `json:"retry_count"`
			StepCount     int `json:"step_count"`
			ToolCallCount int `json:"tool_call_count"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode counters_updated payload: %w", err)
		}
		state.RetryCount = payload.RetryCount
		state.StepCount = payload.StepCount
		state.ToolCallCount = payload.ToolCallCount
		return nil
	case eventIdempotencyRecorded:
		var payload struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode idempotency payload: %w", err)
		}
		if state.IdempotencyKeys == nil {
			state.IdempotencyKeys = map[string]bool{}
		}
		state.IdempotencyKeys[payload.Key] = true
		return nil
	case eventLeaseSet:
		var rec lease.Record
		if err := json.Unmarshal(event.Payload, &rec); err != nil {
			return fmt.Errorf("decode lease payload: %w", err)
		}
		state.Lease = &rec
		return nil
	default:
		return wrkrerrors.New(
			wrkrerrors.EStoreCorrupt,
			"unknown event type",
			map[string]any{"type": event.Type, "seq": event.Seq},
		)
	}
}
