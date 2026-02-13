package reference

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
)

type Step struct {
	ID             string
	Summary        string
	Command        string
	Artifacts      []string
	DecisionNeeded bool
	RequiredAction string
	Executed       bool
}

type RunOptions struct {
	Now func() time.Time
}

type RunResult struct {
	Status          queue.Status `json:"status"`
	DecisionStepID  string       `json:"decision_step_id,omitempty"`
	DecisionSummary string       `json:"decision_summary,omitempty"`
}

func Run(jobID string, steps []Step, opts RunOptions) (RunResult, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	if len(steps) == 0 {
		return RunResult{}, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "reference adapter requires at least one step", nil)
	}

	s, err := store.New("")
	if err != nil {
		return RunResult{}, err
	}
	r, err := runner.New(s, runner.Options{Now: now})
	if err != nil {
		return RunResult{}, err
	}

	for _, step := range steps {
		normalized := normalizeStep(step)
		payload := map[string]any{
			"adapter":   "reference",
			"step_id":   normalized.ID,
			"summary":   normalized.Summary,
			"command":   normalized.Command,
			"executed":  normalized.Executed,
			"artifacts": normalized.Artifacts,
		}
		if _, err := s.AppendEvent(jobID, "adapter_step", payload, now()); err != nil {
			return RunResult{}, err
		}

		if normalized.Executed && normalized.Command != "" {
			// #nosec G204 -- reference adapter executes explicit step command from jobspec.
			cmd := exec.Command("sh", "-lc", normalized.Command)
			if runErr := cmd.Run(); runErr != nil {
				code := 1
				var exitErr *exec.ExitError
				if errors.As(runErr, &exitErr) {
					code = exitErr.ExitCode()
				}
				_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
					Type:        "blocked",
					Summary:     fmt.Sprintf("reference step %s failed (exit=%d)", normalized.ID, code),
					Status:      queue.StatusBlockedError,
					ReasonCodes: []string{string(wrkrerrors.EAdapterFail)},
				})
				_, _ = r.ChangeStatus(jobID, queue.StatusBlockedError)
				return RunResult{Status: queue.StatusBlockedError}, wrkrerrors.New(
					wrkrerrors.EAdapterFail,
					"reference adapter step failed",
					map[string]any{"job_id": jobID, "step_id": normalized.ID, "exit_code": code},
				)
			}
		}

		checkpointType := "progress"
		if normalized.DecisionNeeded {
			checkpointType = "decision-needed"
		}
		status := queue.StatusRunning
		if normalized.DecisionNeeded {
			status = queue.StatusBlockedDecision
		}
		_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
			Type:    checkpointType,
			Summary: normalized.Summary,
			Status:  status,
			ArtifactsDelta: v1.ArtifactsDelta{
				Added: normalized.Artifacts,
			},
			RequiredAction: requiredAction(normalized),
		})

		if normalized.DecisionNeeded {
			if _, err := r.ChangeStatus(jobID, queue.StatusBlockedDecision); err != nil {
				return RunResult{}, err
			}
			return RunResult{
				Status:          queue.StatusBlockedDecision,
				DecisionStepID:  normalized.ID,
				DecisionSummary: normalized.Summary,
			}, nil
		}
	}

	_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:    "completed",
		Summary: "reference adapter completed",
		Status:  queue.StatusCompleted,
	})
	if _, err := r.ChangeStatus(jobID, queue.StatusCompleted); err != nil {
		return RunResult{}, err
	}
	return RunResult{Status: queue.StatusCompleted}, nil
}

func StepsFromInputs(inputs map[string]any) ([]Step, error) {
	raw, ok := inputs["steps"]
	if !ok {
		return nil, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "jobspec.inputs.steps is required for reference adapter", nil)
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "jobspec.inputs.steps must be a list", nil)
	}

	steps := make([]Step, 0, len(items))
	for idx, item := range items {
		asMap, ok := item.(map[string]any)
		if !ok {
			return nil, wrkrerrors.New(
				wrkrerrors.EInvalidInputSchema,
				"jobspec step must be an object",
				map[string]any{"index": idx},
			)
		}
		step := Step{
			ID:             stringField(asMap, "id"),
			Summary:        stringField(asMap, "summary"),
			Command:        stringField(asMap, "command"),
			DecisionNeeded: boolField(asMap, "decision_needed"),
			RequiredAction: stringField(asMap, "required_action"),
			Executed:       boolFieldWithDefault(asMap, "executed", true),
		}
		step.Artifacts = stringSliceField(asMap, "artifacts")
		steps = append(steps, normalizeStep(step))
	}
	return steps, nil
}

func normalizeStep(step Step) Step {
	step.ID = strings.TrimSpace(step.ID)
	if step.ID == "" {
		step.ID = "step"
	}
	step.Summary = strings.TrimSpace(step.Summary)
	if step.Summary == "" {
		step.Summary = "reference adapter step " + step.ID
	}
	step.Command = strings.TrimSpace(step.Command)
	step.RequiredAction = strings.TrimSpace(step.RequiredAction)
	if step.DecisionNeeded && step.RequiredAction == "" {
		step.RequiredAction = "approval"
	}
	if step.Artifacts == nil {
		step.Artifacts = []string{}
	}
	step.Artifacts = uniqueStrings(step.Artifacts)
	return step
}

func requiredAction(step Step) *v1.RequiredAction {
	if !step.DecisionNeeded {
		return nil
	}
	return &v1.RequiredAction{
		Kind:         step.RequiredAction,
		Instructions: "review and approve step " + step.ID,
	}
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func stringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func boolField(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func boolFieldWithDefault(m map[string]any, key string, fallback bool) bool {
	v, ok := m[key]
	if !ok {
		return fallback
	}
	b, ok := v.(bool)
	if !ok {
		return fallback
	}
	return b
}

func stringSliceField(m map[string]any, key string) []string {
	raw, ok := m[key]
	if !ok {
		return []string{}
	}
	items, ok := raw.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
