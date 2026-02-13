package dispatch

import (
	"fmt"
	"strings"
	"time"

	"github.com/davidahmann/wrkr/core/adapters/reference"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/projectconfig"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
)

type SubmitOptions struct {
	Now       func() time.Time
	JobID     string
	FromServe bool
}

type SubmitResult struct {
	JobID     string       `json:"job_id"`
	Status    queue.Status `json:"status"`
	Adapter   string       `json:"adapter"`
	SpecName  string       `json:"spec_name"`
	Objective string       `json:"objective"`
	SpecPath  string       `json:"spec_path"`
}

func Submit(specPath string, opts SubmitOptions) (SubmitResult, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	spec, err := projectconfig.LoadJobSpec(specPath)
	if err != nil {
		return SubmitResult{}, err
	}

	jobID := strings.TrimSpace(opts.JobID)
	if jobID == "" {
		jobID = inferJobID(spec.Name, now())
	}
	jobID = projectconfig.NormalizeJobID(jobID)

	s, err := store.New("")
	if err != nil {
		return SubmitResult{}, err
	}
	exists, err := s.JobExists(jobID)
	if err != nil {
		return SubmitResult{}, err
	}
	if exists {
		return SubmitResult{}, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"job already exists",
			map[string]any{"job_id": jobID},
		)
	}

	r, err := runner.New(s, runner.Options{Now: now})
	if err != nil {
		return SubmitResult{}, err
	}
	if _, err := r.InitJob(jobID); err != nil {
		return SubmitResult{}, err
	}
	if _, err := r.ChangeStatus(jobID, queue.StatusRunning); err != nil {
		return SubmitResult{}, err
	}
	_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:    "plan",
		Summary: spec.Objective,
		Status:  queue.StatusRunning,
	})

	adapterName := strings.ToLower(strings.TrimSpace(spec.Adapter.Name))
	if adapterName == "" {
		adapterName = "reference"
	}
	status, runErr := runAdapter(adapterName, jobID, spec.Inputs, now)
	if runErr != nil {
		return SubmitResult{}, runErr
	}

	return SubmitResult{
		JobID:     jobID,
		Status:    status,
		Adapter:   adapterName,
		SpecName:  spec.Name,
		Objective: spec.Objective,
		SpecPath:  specPath,
	}, nil
}

func runAdapter(adapterName, jobID string, inputs map[string]any, now func() time.Time) (queue.Status, error) {
	switch adapterName {
	case "reference":
		steps, err := reference.StepsFromInputs(inputs)
		if err != nil {
			return "", err
		}
		result, err := reference.Run(jobID, steps, reference.RunOptions{Now: now})
		if err != nil {
			return "", err
		}
		return result.Status, nil
	case "noop":
		s, err := store.New("")
		if err != nil {
			return "", err
		}
		r, err := runner.New(s, runner.Options{Now: now})
		if err != nil {
			return "", err
		}
		_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
			Type:    "completed",
			Summary: "noop adapter completed",
			Status:  queue.StatusCompleted,
		})
		if _, err := r.ChangeStatus(jobID, queue.StatusCompleted); err != nil {
			return "", err
		}
		return queue.StatusCompleted, nil
	default:
		return "", wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			fmt.Sprintf("unsupported adapter %q", adapterName),
			map[string]any{"adapter": adapterName},
		)
	}
}

func inferJobID(name string, now time.Time) string {
	base := projectconfig.NormalizeJobID(name)
	if base == "" {
		base = "job"
	}
	return fmt.Sprintf("%s_%d", base, now.UTC().Unix())
}
