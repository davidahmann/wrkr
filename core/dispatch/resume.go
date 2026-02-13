package dispatch

import (
	"strings"
	"time"

	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
)

type ResumeOptions struct {
	Now                 func() time.Time
	OverrideEnvMismatch bool
	OverrideReason      string
	ApprovedBy          string
}

type ResumeResult struct {
	JobID         string       `json:"job_id"`
	Status        queue.Status `json:"status"`
	Adapter       string       `json:"adapter,omitempty"`
	NextStepIndex int          `json:"next_step_index,omitempty"`
}

func Resume(jobID string, opts ResumeOptions) (ResumeResult, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	jobID = strings.TrimSpace(jobID)

	s, err := store.New("")
	if err != nil {
		return ResumeResult{}, err
	}
	r, err := runner.New(s, runner.Options{Now: now})
	if err != nil {
		return ResumeResult{}, err
	}
	state, err := r.Resume(jobID, runner.ResumeInput{
		OverrideEnvMismatch: opts.OverrideEnvMismatch,
		OverrideReason:      opts.OverrideReason,
		ApprovedBy:          opts.ApprovedBy,
	})
	if err != nil {
		return ResumeResult{}, err
	}

	runtimeCfg, err := LoadRuntimeConfig(s, jobID)
	if err != nil {
		return ResumeResult{}, err
	}
	if runtimeCfg == nil || state.Status != queue.StatusRunning {
		return ResumeResult{
			JobID:  jobID,
			Status: state.Status,
		}, nil
	}

	adapterName := adapterNameOrDefault(runtimeCfg.Adapter)
	adapterResult, runErr := executeWithLease(r, jobID, now, func() (adapterRunResult, error) {
		return runAdapter(adapterName, jobID, runtimeCfg, r, s, now)
	})
	if saveErr := SaveRuntimeConfig(s, jobID, *runtimeCfg, now()); saveErr != nil {
		return ResumeResult{}, saveErr
	}

	result := ResumeResult{
		JobID:         jobID,
		Status:        adapterResult.Status,
		Adapter:       adapterName,
		NextStepIndex: runtimeCfg.NextStepIndex,
	}
	if runErr != nil {
		return result, runErr
	}
	return result, nil
}
