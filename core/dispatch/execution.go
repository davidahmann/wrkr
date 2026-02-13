package dispatch

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/davidahmann/wrkr/core/adapters/reference"
	"github.com/davidahmann/wrkr/core/budget"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
)

const leaseHeartbeatInterval = 10 * time.Second

type adapterRunResult struct {
	Status        queue.Status
	NextStepIndex int
}

func executeWithLease(
	r *runner.Runner,
	jobID string,
	now func() time.Time,
	run func() (adapterRunResult, error),
) (adapterRunResult, error) {
	workerID := fmt.Sprintf("dispatch-%d", os.Getpid())
	leaseID := fmt.Sprintf("lease-%d-%d", os.Getpid(), now().UTC().UnixNano())

	if _, err := r.AcquireLease(jobID, workerID, leaseID); err != nil {
		return adapterRunResult{}, err
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	var heartbeatErr error
	var heartbeatErrMu sync.Mutex
	recordHeartbeatErr := func(err error) {
		heartbeatErrMu.Lock()
		defer heartbeatErrMu.Unlock()
		if heartbeatErr == nil {
			heartbeatErr = err
		}
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(leaseHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if _, err := r.HeartbeatLease(jobID, workerID, leaseID); err != nil {
					recordHeartbeatErr(err)
					return
				}
			}
		}
	}()

	result, runErr := run()
	close(stop)
	wg.Wait()
	_, releaseErr := r.ReleaseLease(jobID, workerID, leaseID)

	heartbeatErrMu.Lock()
	hbErr := heartbeatErr
	heartbeatErrMu.Unlock()

	if runErr != nil {
		return result, errors.Join(runErr, releaseErr)
	}
	if hbErr != nil {
		return result, errors.Join(hbErr, releaseErr)
	}
	if releaseErr != nil {
		return result, releaseErr
	}

	return result, nil
}

func runAdapter(
	adapterName, jobID string,
	runtimeCfg *RuntimeConfig,
	r *runner.Runner,
	s *store.LocalStore,
	now func() time.Time,
) (adapterRunResult, error) {
	switch adapterName {
	case "reference":
		steps, err := reference.StepsFromInputs(runtimeCfg.Inputs)
		if err != nil {
			return adapterRunResult{}, err
		}
		result, err := reference.Run(jobID, steps, reference.RunOptions{
			Now:          now,
			StartIndex:   runtimeCfg.NextStepIndex,
			BudgetLimits: runtimeCfg.Budgets,
			OnAdvance: func(nextStepIndex int) error {
				runtimeCfg.NextStepIndex = nextStepIndex
				return SaveRuntimeConfig(s, jobID, *runtimeCfg, now())
			},
		})
		runResult := adapterRunResult{
			Status:        result.Status,
			NextStepIndex: result.NextStepIndex,
		}
		if err != nil {
			return runResult, err
		}
		return runResult, nil

	case "noop":
		_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
			Type:    "completed",
			Summary: "noop adapter completed",
			Status:  queue.StatusCompleted,
		})
		if _, err := r.ChangeStatus(jobID, queue.StatusCompleted); err != nil {
			return adapterRunResult{}, err
		}
		runtimeCfg.NextStepIndex = 0
		return adapterRunResult{Status: queue.StatusCompleted, NextStepIndex: runtimeCfg.NextStepIndex}, nil

	default:
		return adapterRunResult{}, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			fmt.Sprintf("unsupported adapter %q", adapterName),
			map[string]any{"adapter": adapterName},
		)
	}
}

func budgetFromSpec(spec v1.BudgetSpec) budget.Limits {
	limits := budget.Limits{
		MaxWallTimeSeconds: spec.MaxWallTimeSeconds,
		MaxRetries:         spec.MaxRetries,
		MaxStepCount:       spec.MaxStepCount,
		MaxToolCalls:       spec.MaxToolCalls,
	}
	if spec.MaxEstimatedCost != nil {
		value := *spec.MaxEstimatedCost
		limits.MaxEstimatedCost = &value
	}
	if spec.MaxTokens != nil {
		value := *spec.MaxTokens
		limits.MaxTokens = &value
	}
	return limits
}

func adapterNameOrDefault(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return "reference"
	}
	return normalized
}
