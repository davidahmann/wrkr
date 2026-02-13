package queue

import (
	"fmt"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

type Status string

const (
	StatusQueued          Status = "queued"
	StatusRunning         Status = "running"
	StatusPaused          Status = "paused"
	StatusBlockedDecision Status = "blocked_decision"
	StatusBlockedBudget   Status = "blocked_budget"
	StatusBlockedError    Status = "blocked_error"
	StatusCompleted       Status = "completed"
	StatusCanceled        Status = "canceled"
)

var allowedTransitions = map[Status]map[Status]struct{}{
	StatusQueued: {
		StatusRunning:  {},
		StatusCanceled: {},
	},
	StatusRunning: {
		StatusPaused:          {},
		StatusBlockedDecision: {},
		StatusBlockedBudget:   {},
		StatusBlockedError:    {},
		StatusCompleted:       {},
		StatusCanceled:        {},
	},
	StatusPaused: {
		StatusRunning:      {},
		StatusBlockedError: {},
		StatusCanceled:     {},
	},
	StatusBlockedDecision: {
		StatusRunning:      {},
		StatusBlockedError: {},
		StatusCanceled:     {},
	},
	StatusBlockedBudget: {
		StatusRunning:      {},
		StatusBlockedError: {},
		StatusCanceled:     {},
	},
	StatusBlockedError: {
		StatusRunning:  {},
		StatusCanceled: {},
	},
	StatusCompleted: {},
	StatusCanceled:  {},
}

func IsKnownStatus(status Status) bool {
	_, ok := allowedTransitions[status]
	return ok
}

func ValidateTransition(from, to Status) error {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return wrkrerrors.New(
			wrkrerrors.EInvalidStateTransition,
			fmt.Sprintf("unknown current status %q", from),
			map[string]any{"from": from, "to": to},
		)
	}
	if _, ok := allowed[to]; !ok {
		return wrkrerrors.New(
			wrkrerrors.EInvalidStateTransition,
			fmt.Sprintf("invalid status transition %q -> %q", from, to),
			map[string]any{"from": from, "to": to},
		)
	}
	return nil
}
