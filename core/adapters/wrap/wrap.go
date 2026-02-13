package wrap

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
)

type RunOptions struct {
	Now            func() time.Time
	ExpectedOutput []string
}

type RunResult struct {
	JobID    string       `json:"job_id"`
	Status   queue.Status `json:"status"`
	ExitCode int          `json:"exit_code"`
	Stdout   string       `json:"stdout,omitempty"`
	Stderr   string       `json:"stderr,omitempty"`
}

func Run(jobID string, command []string, opts RunOptions) (RunResult, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	command = trimCommand(command)
	if len(command) == 0 {
		return RunResult{}, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"wrap command is required",
			nil,
		)
	}

	s, err := store.New("")
	if err != nil {
		return RunResult{}, err
	}
	r, err := runner.New(s, runner.Options{Now: now})
	if err != nil {
		return RunResult{}, err
	}

	if _, err := r.InitJob(jobID); err != nil {
		return RunResult{}, err
	}
	if _, err := r.ChangeStatus(jobID, queue.StatusRunning); err != nil {
		return RunResult{}, err
	}

	_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:    "plan",
		Summary: "wrap mode executing command: " + strings.Join(command, " "),
		Status:  queue.StatusRunning,
	})

	// #nosec G204 -- wrap intentionally executes user-supplied adapter command.
	cmd := exec.Command(command[0], command[1:]...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	exitCode := 0
	status := queue.StatusCompleted
	if runErr != nil {
		exitCode = 1
		status = queue.StatusBlockedError
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	progressSummary := fmt.Sprintf("wrap command finished (exit=%d)", exitCode)
	_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:    "progress",
		Summary: progressSummary,
		Status:  queue.StatusRunning,
		ArtifactsDelta: v1.ArtifactsDelta{
			Added: opts.ExpectedOutput,
		},
	})

	if runErr != nil {
		if _, err := r.ChangeStatus(jobID, queue.StatusBlockedError); err == nil {
			_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
				Type:        "blocked",
				Summary:     "wrap command failed",
				Status:      queue.StatusBlockedError,
				ReasonCodes: []string{string(wrkrerrors.EAdapterFail)},
			})
		}
		return RunResult{
				JobID:    jobID,
				Status:   status,
				ExitCode: exitCode,
				Stdout:   strings.TrimSpace(stdout.String()),
				Stderr:   strings.TrimSpace(stderr.String()),
			}, wrkrerrors.New(
				wrkrerrors.EAdapterFail,
				"wrap command failed",
				map[string]any{"job_id": jobID, "command": strings.Join(command, " "), "exit_code": exitCode},
			)
	}

	_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:    "completed",
		Summary: "wrap mode completed successfully",
		Status:  queue.StatusCompleted,
	})
	if _, err := r.ChangeStatus(jobID, queue.StatusCompleted); err != nil {
		return RunResult{}, err
	}

	return RunResult{
		JobID:    jobID,
		Status:   status,
		ExitCode: exitCode,
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
	}, nil
}

func trimCommand(command []string) []string {
	out := make([]string, 0, len(command))
	for _, part := range command {
		if strings.TrimSpace(part) == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
