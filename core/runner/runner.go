package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/davidahmann/wrkr/core/budget"
	"github.com/davidahmann/wrkr/core/envfp"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/fsx"
	"github.com/davidahmann/wrkr/core/lease"
	"github.com/davidahmann/wrkr/core/queue"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
)

const (
	eventJobInitialized      = "job_initialized"
	eventStatusChanged       = "status_changed"
	eventCountersUpdated     = "counters_updated"
	eventIdempotencyRecorded = "idempotency_recorded"
	eventLeaseSet            = "lease_set"
	eventCheckpointEmitted   = "checkpoint_emitted"
	eventApprovalRecorded    = "approval_recorded"
	eventEnvFingerprintSet   = "env_fingerprint_set"
	eventEnvOverrideRecorded = "env_override_recorded"
	maxCASAttempts           = 64
	maxSummaryLength         = 2000
)

type State struct {
	JobID                string            `json:"job_id"`
	Status               queue.Status      `json:"status"`
	RetryCount           int               `json:"retry_count"`
	StepCount            int               `json:"step_count"`
	ToolCallCount        int               `json:"tool_call_count"`
	IdempotencyKeys      map[string]bool   `json:"idempotency_keys"`
	Lease                *lease.Record     `json:"lease,omitempty"`
	LastAppliedSeq       int64             `json:"last_applied_seq"`
	StartedAt            *time.Time        `json:"started_at,omitempty"`
	LastReasonCodes      []string          `json:"last_reason_codes,omitempty"`
	EnvFingerprintHash   string            `json:"env_fingerprint_hash,omitempty"`
	EnvFingerprintRules  []string          `json:"env_fingerprint_rules,omitempty"`
	EnvFingerprintValues map[string]string `json:"env_fingerprint_values,omitempty"`
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

type CheckpointInput struct {
	Type           string
	Summary        string
	Status         queue.Status
	BudgetState    v1.BudgetState
	ArtifactsDelta v1.ArtifactsDelta
	RequiredAction *v1.RequiredAction
	ReasonCodes    []string
}

type ResumeInput struct {
	OverrideEnvMismatch bool
	OverrideReason      string
	ApprovedBy          string
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
	startedAt := r.now().UTC()
	event, err := r.store.AppendEvent(
		jobID,
		eventJobInitialized,
		map[string]any{"status": state.Status, "started_at": startedAt},
		startedAt,
	)
	if err != nil {
		return nil, err
	}
	state.LastAppliedSeq = event.Seq
	state.StartedAt = &startedAt

	fp, err := envfp.Capture(nil, startedAt)
	if err != nil {
		return nil, err
	}
	event, err = r.store.AppendEvent(jobID, eventEnvFingerprintSet, fp, r.now())
	if err != nil {
		return nil, err
	}
	state.EnvFingerprintHash = fp.Hash
	state.EnvFingerprintRules = fp.Rules
	state.EnvFingerprintValues = fp.Values
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
	for attempt := 0; attempt < maxCASAttempts; attempt++ {
		state, err := r.Recover(jobID)
		if err != nil {
			return nil, err
		}
		if err := queue.ValidateTransition(state.Status, to); err != nil {
			return nil, err
		}

		event, err := r.store.AppendEventCAS(
			jobID,
			eventStatusChanged,
			map[string]any{"from": state.Status, "to": to},
			state.LastAppliedSeq,
			r.now(),
		)
		if err != nil {
			if errors.Is(err, store.ErrCASConflict) || errors.Is(err, fsx.ErrLockBusy) {
				time.Sleep(1 * time.Millisecond)
				continue
			}
			return nil, err
		}

		state.Status = to
		state.LastAppliedSeq = event.Seq
		if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
			return nil, err
		}
		return state, nil
	}

	return nil, wrkrerrors.New(
		wrkrerrors.EStoreCorrupt,
		"status update contention exceeded retry budget",
		map[string]any{"job_id": jobID},
	)
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
	for attempt := 0; attempt < maxCASAttempts; attempt++ {
		state, err := r.Recover(jobID)
		if err != nil {
			return nil, err
		}

		rec, err := lease.Acquire(state.Lease, workerID, leaseID, r.now(), r.leaseTTL)
		if err != nil {
			return nil, err
		}

		event, err := r.store.AppendEventCAS(jobID, eventLeaseSet, rec, state.LastAppliedSeq, r.now())
		if err != nil {
			if errors.Is(err, store.ErrCASConflict) || errors.Is(err, fsx.ErrLockBusy) {
				time.Sleep(1 * time.Millisecond)
				continue
			}
			return nil, err
		}
		state.Lease = rec
		state.LastAppliedSeq = event.Seq
		if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
			return nil, err
		}
		return state, nil
	}

	return nil, wrkrerrors.New(
		wrkrerrors.EStoreCorrupt,
		"lease acquire contention exceeded retry budget",
		map[string]any{"job_id": jobID, "worker_id": workerID, "lease_id": leaseID},
	)
}

func (r *Runner) HeartbeatLease(jobID, workerID, leaseID string) (*State, error) {
	for attempt := 0; attempt < maxCASAttempts; attempt++ {
		state, err := r.Recover(jobID)
		if err != nil {
			return nil, err
		}

		rec, err := lease.Heartbeat(state.Lease, workerID, leaseID, r.now(), r.leaseTTL)
		if err != nil {
			return nil, err
		}

		event, err := r.store.AppendEventCAS(jobID, eventLeaseSet, rec, state.LastAppliedSeq, r.now())
		if err != nil {
			if errors.Is(err, store.ErrCASConflict) || errors.Is(err, fsx.ErrLockBusy) {
				time.Sleep(1 * time.Millisecond)
				continue
			}
			return nil, err
		}

		state.Lease = rec
		state.LastAppliedSeq = event.Seq
		if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
			return nil, err
		}
		return state, nil
	}

	return nil, wrkrerrors.New(
		wrkrerrors.EStoreCorrupt,
		"lease heartbeat contention exceeded retry budget",
		map[string]any{"job_id": jobID, "worker_id": workerID, "lease_id": leaseID},
	)
}

func (r *Runner) EmitCheckpoint(jobID string, input CheckpointInput) (*v1.Checkpoint, error) {
	state, err := r.Recover(jobID)
	if err != nil {
		return nil, err
	}

	cpType := strings.TrimSpace(input.Type)
	if !isCheckpointType(cpType) {
		return nil, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"invalid checkpoint type",
			map[string]any{"type": cpType},
		)
	}

	summary := strings.TrimSpace(input.Summary)
	if summary == "" || len(summary) > maxSummaryLength {
		return nil, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"checkpoint summary must be 1..2000 chars",
			map[string]any{"summary_length": len(summary)},
		)
	}

	if cpType == "decision-needed" && input.RequiredAction == nil {
		return nil, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"decision-needed checkpoint requires required_action",
			nil,
		)
	}

	status := input.Status
	if status == "" {
		status = state.Status
	}

	bs := input.BudgetState
	if bs.WallTimeSeconds == 0 && bs.RetryCount == 0 && bs.StepCount == 0 && bs.ToolCallCount == 0 {
		bs = budgetUsageFromState(state, r.now())
	}
	delta := input.ArtifactsDelta
	if delta.Added == nil {
		delta.Added = []string{}
	}
	if delta.Changed == nil {
		delta.Changed = []string{}
	}
	if delta.Removed == nil {
		delta.Removed = []string{}
	}
	reasonCodes := append([]string(nil), input.ReasonCodes...)
	if reasonCodes == nil {
		reasonCodes = []string{}
	}

	payload := map[string]any{
		"type":            cpType,
		"summary":         summary,
		"status":          string(status),
		"budget_state":    bs,
		"artifacts_delta": delta,
		"required_action": input.RequiredAction,
		"reason_codes":    reasonCodes,
	}

	event, err := r.store.AppendEvent(jobID, eventCheckpointEmitted, payload, r.now())
	if err != nil {
		return nil, err
	}

	cp, err := checkpointFromEvent(jobID, event)
	if err != nil {
		return nil, err
	}

	state.LastReasonCodes = append([]string(nil), cp.ReasonCodes...)
	state.LastAppliedSeq = event.Seq
	if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
		return nil, err
	}

	return cp, nil
}

func (r *Runner) ListCheckpoints(jobID string) ([]v1.Checkpoint, error) {
	events, err := r.store.LoadEvents(jobID)
	if err != nil {
		return nil, err
	}

	out := make([]v1.Checkpoint, 0, 8)
	for _, event := range events {
		if event.Type != eventCheckpointEmitted {
			continue
		}
		cp, err := checkpointFromEvent(jobID, event)
		if err != nil {
			return nil, err
		}
		out = append(out, *cp)
	}
	return out, nil
}

func (r *Runner) GetCheckpoint(jobID, checkpointID string) (*v1.Checkpoint, error) {
	seq, err := parseCheckpointID(checkpointID)
	if err != nil {
		return nil, err
	}

	events, err := r.store.LoadEvents(jobID)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.Seq == seq && event.Type == eventCheckpointEmitted {
			return checkpointFromEvent(jobID, event)
		}
	}

	return nil, wrkrerrors.New(
		wrkrerrors.EInvalidInputSchema,
		"checkpoint not found",
		map[string]any{"job_id": jobID, "checkpoint_id": checkpointID},
	)
}

func (r *Runner) ApproveCheckpoint(jobID, checkpointID, reason, approvedBy string) (*v1.ApprovalRecord, error) {
	cp, err := r.GetCheckpoint(jobID, checkpointID)
	if err != nil {
		return nil, err
	}
	if cp.Type != "decision-needed" {
		return nil, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"only decision-needed checkpoints can be approved",
			map[string]any{"checkpoint_id": checkpointID, "type": cp.Type},
		)
	}

	if strings.TrimSpace(reason) == "" {
		return nil, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "approval reason is required", nil)
	}
	if strings.TrimSpace(approvedBy) == "" {
		return nil, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "approved_by is required", nil)
	}

	record := v1.ApprovalRecord{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.approval_record",
			SchemaVersion:   "v1",
			CreatedAt:       r.now().UTC(),
			ProducerVersion: "dev",
		},
		JobID:        jobID,
		CheckpointID: checkpointID,
		Reason:       strings.TrimSpace(reason),
		ApprovedBy:   strings.TrimSpace(approvedBy),
	}

	event, err := r.store.AppendEvent(jobID, eventApprovalRecorded, record, r.now())
	if err != nil {
		return nil, err
	}

	state, err := r.Recover(jobID)
	if err != nil {
		return nil, err
	}
	state.LastAppliedSeq = event.Seq
	if err := r.store.SaveSnapshot(jobID, state.LastAppliedSeq, state, r.now()); err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *Runner) ListApprovals(jobID string) ([]v1.ApprovalRecord, error) {
	events, err := r.store.LoadEvents(jobID)
	if err != nil {
		return nil, err
	}

	out := make([]v1.ApprovalRecord, 0, 4)
	for _, event := range events {
		if event.Type != eventApprovalRecorded {
			continue
		}
		var rec v1.ApprovalRecord
		if err := json.Unmarshal(event.Payload, &rec); err != nil {
			return nil, fmt.Errorf("decode approval payload: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *Runner) CheckBudget(jobID string, limits budget.Limits) (*v1.Checkpoint, error) {
	state, err := r.Recover(jobID)
	if err != nil {
		return nil, err
	}

	usage := budget.Usage{
		WallTimeSeconds: budgetUsageFromState(state, r.now()).WallTimeSeconds,
		RetryCount:      state.RetryCount,
		StepCount:       state.StepCount,
		ToolCallCount:   state.ToolCallCount,
	}
	result := budget.Evaluate(limits, usage)
	if !result.Exceeded {
		return nil, nil
	}

	if state.Status != queue.StatusBlockedBudget {
		if _, err := r.ChangeStatus(jobID, queue.StatusBlockedBudget); err != nil {
			return nil, err
		}
	}

	cp, err := r.EmitCheckpoint(jobID, CheckpointInput{
		Type:        "blocked",
		Summary:     "budget exceeded: " + strings.Join(result.Violations, ", "),
		Status:      queue.StatusBlockedBudget,
		BudgetState: budgetUsageFromState(state, r.now()),
		ReasonCodes: []string{string(wrkrerrors.EBudgetExceeded)},
	})
	if err != nil {
		return nil, err
	}

	return cp, wrkrerrors.New(
		wrkrerrors.EBudgetExceeded,
		"job stopped because budget limits were exceeded",
		map[string]any{"job_id": jobID, "violations": result.Violations},
	)
}

func (r *Runner) Resume(jobID string, input ResumeInput) (*State, error) {
	state, err := r.Recover(jobID)
	if err != nil {
		return nil, err
	}

	rules := state.EnvFingerprintRules
	if len(rules) == 0 {
		rules = envfp.DefaultRules()
	}
	currentFP, err := envfp.Capture(rules, r.now())
	if err != nil {
		return nil, err
	}

	if state.EnvFingerprintHash == "" {
		if _, err := r.store.AppendEvent(jobID, eventEnvFingerprintSet, currentFP, r.now()); err != nil {
			return nil, err
		}
		state.EnvFingerprintHash = currentFP.Hash
		state.EnvFingerprintRules = currentFP.Rules
		state.EnvFingerprintValues = currentFP.Values
	}

	if state.EnvFingerprintHash != "" && state.EnvFingerprintHash != currentFP.Hash {
		if !input.OverrideEnvMismatch {
			if state.Status != queue.StatusBlockedError {
				if _, err := r.ChangeStatus(jobID, queue.StatusBlockedError); err != nil {
					var werr wrkrerrors.WrkrError
					if !(errors.As(err, &werr) && werr.Code == wrkrerrors.EInvalidStateTransition) {
						return nil, err
					}
				}
			}
			if _, err := r.EmitCheckpoint(jobID, CheckpointInput{
				Type:        "blocked",
				Summary:     "environment fingerprint mismatch; resume blocked",
				Status:      queue.StatusBlockedError,
				BudgetState: budgetUsageFromState(state, r.now()),
				ReasonCodes: []string{string(wrkrerrors.EEnvFingerprintMismatch)},
			}); err != nil {
				return nil, err
			}
			return nil, wrkrerrors.New(
				wrkrerrors.EEnvFingerprintMismatch,
				"environment fingerprint mismatch",
				map[string]any{"job_id": jobID, "expected_hash": state.EnvFingerprintHash, "actual_hash": currentFP.Hash},
			)
		}

		overridePayload := map[string]any{
			"expected_hash": state.EnvFingerprintHash,
			"actual_hash":   currentFP.Hash,
			"reason":        strings.TrimSpace(input.OverrideReason),
			"approved_by":   strings.TrimSpace(input.ApprovedBy),
			"rules":         currentFP.Rules,
			"values":        currentFP.Values,
			"captured_at":   currentFP.CapturedAt.UTC(),
		}
		if _, err := r.store.AppendEvent(jobID, eventEnvOverrideRecorded, overridePayload, r.now()); err != nil {
			return nil, err
		}
	}

	latestDecisionID, err := r.latestDecisionCheckpoint(jobID)
	if err != nil {
		return nil, err
	}
	if latestDecisionID != "" {
		approved, err := r.hasApproval(jobID, latestDecisionID)
		if err != nil {
			return nil, err
		}
		if !approved {
			return nil, wrkrerrors.New(
				wrkrerrors.ECheckpointApprovalRequired,
				"approval required before resume",
				map[string]any{"job_id": jobID, "checkpoint_id": latestDecisionID},
			)
		}
	}

	state, err = r.Recover(jobID)
	if err != nil {
		return nil, err
	}
	if state.Status != queue.StatusRunning {
		if _, err := r.ChangeStatus(jobID, queue.StatusRunning); err != nil {
			return nil, err
		}
	}
	return r.Recover(jobID)
}

func checkpointIDForSeq(seq int64) string {
	return fmt.Sprintf("cp_%d", seq)
}

func parseCheckpointID(id string) (int64, error) {
	parts := strings.SplitN(strings.TrimSpace(id), "_", 2)
	if len(parts) != 2 || parts[0] != "cp" {
		return 0, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"invalid checkpoint id",
			map[string]any{"checkpoint_id": id},
		)
	}
	seq, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || seq <= 0 {
		return 0, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"invalid checkpoint id",
			map[string]any{"checkpoint_id": id},
		)
	}
	return seq, nil
}

func isCheckpointType(v string) bool {
	switch v {
	case "plan", "progress", "decision-needed", "blocked", "completed":
		return true
	default:
		return false
	}
}

func checkpointFromEvent(jobID string, event store.Event) (*v1.Checkpoint, error) {
	var payload struct {
		Type           string             `json:"type"`
		Summary        string             `json:"summary"`
		Status         string             `json:"status"`
		BudgetState    v1.BudgetState     `json:"budget_state"`
		ArtifactsDelta v1.ArtifactsDelta  `json:"artifacts_delta"`
		RequiredAction *v1.RequiredAction `json:"required_action,omitempty"`
		ReasonCodes    []string           `json:"reason_codes"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode checkpoint payload: %w", err)
	}

	return &v1.Checkpoint{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.checkpoint",
			SchemaVersion:   "v1",
			CreatedAt:       event.CreatedAt.UTC(),
			ProducerVersion: "dev",
		},
		CheckpointID:   checkpointIDForSeq(event.Seq),
		JobID:          jobID,
		Type:           payload.Type,
		Summary:        payload.Summary,
		Status:         payload.Status,
		BudgetState:    payload.BudgetState,
		ArtifactsDelta: payload.ArtifactsDelta,
		RequiredAction: payload.RequiredAction,
		ReasonCodes:    payload.ReasonCodes,
	}, nil
}

func budgetUsageFromState(state *State, now time.Time) v1.BudgetState {
	wall := 0
	if state.StartedAt != nil {
		d := now.UTC().Sub(state.StartedAt.UTC())
		if d > 0 {
			wall = int(d.Seconds())
		}
	}
	return v1.BudgetState{
		WallTimeSeconds: wall,
		RetryCount:      state.RetryCount,
		StepCount:       state.StepCount,
		ToolCallCount:   state.ToolCallCount,
	}
}

func (r *Runner) latestDecisionCheckpoint(jobID string) (string, error) {
	checkpoints, err := r.ListCheckpoints(jobID)
	if err != nil {
		return "", err
	}
	for i := len(checkpoints) - 1; i >= 0; i-- {
		if checkpoints[i].Type == "decision-needed" {
			return checkpoints[i].CheckpointID, nil
		}
	}
	return "", nil
}

func (r *Runner) hasApproval(jobID, checkpointID string) (bool, error) {
	approvals, err := r.ListApprovals(jobID)
	if err != nil {
		return false, err
	}
	for _, rec := range approvals {
		if rec.CheckpointID == checkpointID {
			return true, nil
		}
	}
	return false, nil
}

func applyEvent(state *State, event store.Event) error {
	switch event.Type {
	case eventJobInitialized:
		var payload struct {
			StartedAt *time.Time `json:"started_at,omitempty"`
		}
		if len(event.Payload) > 0 {
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return fmt.Errorf("decode job_initialized payload: %w", err)
			}
		}
		state.Status = queue.StatusQueued
		if payload.StartedAt != nil {
			at := payload.StartedAt.UTC()
			state.StartedAt = &at
		}
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
	case eventCheckpointEmitted:
		var payload struct {
			ReasonCodes []string `json:"reason_codes"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode checkpoint payload: %w", err)
		}
		state.LastReasonCodes = append([]string(nil), payload.ReasonCodes...)
		return nil
	case eventEnvFingerprintSet:
		var fp envfp.Fingerprint
		if err := json.Unmarshal(event.Payload, &fp); err != nil {
			return fmt.Errorf("decode env fingerprint payload: %w", err)
		}
		state.EnvFingerprintHash = fp.Hash
		state.EnvFingerprintRules = append([]string(nil), fp.Rules...)
		state.EnvFingerprintValues = make(map[string]string, len(fp.Values))
		for k, v := range fp.Values {
			state.EnvFingerprintValues[k] = v
		}
		return nil
	case eventEnvOverrideRecorded:
		var payload struct {
			ActualHash string            `json:"actual_hash"`
			Rules      []string          `json:"rules"`
			Values     map[string]string `json:"values"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode env override payload: %w", err)
		}
		state.EnvFingerprintHash = payload.ActualHash
		state.EnvFingerprintRules = append([]string(nil), payload.Rules...)
		state.EnvFingerprintValues = make(map[string]string, len(payload.Values))
		for k, v := range payload.Values {
			state.EnvFingerprintValues[k] = v
		}
		return nil
	case eventApprovalRecorded:
		return nil
	default:
		return wrkrerrors.New(
			wrkrerrors.EStoreCorrupt,
			"unknown event type",
			map[string]any{"type": event.Type, "seq": event.Seq},
		)
	}
}
