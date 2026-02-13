package dispatch

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
)

func TestSubmitReferenceJobSpec(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 2, 0, 0, 0, time.UTC)

	specPath := filepath.Join(t.TempDir(), "jobspec.yaml")
	if err := os.WriteFile(specPath, []byte(`schema_id: wrkr.jobspec
schema_version: v1
created_at: "2026-02-14T02:00:00Z"
producer_version: test
name: submit-demo
objective: test submit
inputs:
  steps:
    - id: build
      summary: run step
      command: "true"
      artifacts:
        - reports/out.md
      executed: true
expected_artifacts:
  - reports/out.md
adapter:
  name: reference
budgets:
  max_wall_time_seconds: 100
  max_retries: 1
  max_step_count: 5
  max_tool_calls: 5
checkpoint_policy:
  min_interval_seconds: 1
  required_types: [plan, progress, completed]
environment_fingerprint:
  rules: [go_version]
`), 0o600); err != nil {
		t.Fatalf("write jobspec: %v", err)
	}

	result, err := Submit(specPath, SubmitOptions{Now: func() time.Time { return now }, JobID: "job_submit_ok"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if result.Status != queue.StatusCompleted {
		t.Fatalf("expected completed, got %s", result.Status)
	}
	if result.JobID != "job_submit_ok" {
		t.Fatalf("unexpected job id: %s", result.JobID)
	}

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	state, err := r.Recover("job_submit_ok")
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if len(state.EnvFingerprintRules) != 1 || state.EnvFingerprintRules[0] != "go_version" {
		t.Fatalf("expected env rules from jobspec, got %+v", state.EnvFingerprintRules)
	}
}

func TestSubmitResumeContinuesRemainingSteps(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 2, 10, 0, 0, time.UTC)

	specPath := filepath.Join(t.TempDir(), "jobspec_resume.yaml")
	if err := os.WriteFile(specPath, []byte(`schema_id: wrkr.jobspec
schema_version: v1
created_at: "2026-02-14T02:10:00Z"
producer_version: test
name: submit-resume
objective: test submit resume continuation
inputs:
  steps:
    - id: build
      summary: run step
      command: "true"
      artifacts: [reports/build.md]
      executed: true
    - id: review
      summary: approval needed
      decision_needed: true
      required_action: approval
      executed: false
    - id: finalize
      summary: finalize output
      command: "true"
      artifacts: [reports/final.md]
      executed: true
expected_artifacts: [reports/final.md]
adapter: { name: reference }
budgets:
  max_wall_time_seconds: 100
  max_retries: 1
  max_step_count: 10
  max_tool_calls: 10
checkpoint_policy:
  min_interval_seconds: 1
  required_types: [plan, progress, decision-needed, completed]
environment_fingerprint:
  rules: [go_version]
`), 0o600); err != nil {
		t.Fatalf("write jobspec: %v", err)
	}

	submitResult, err := Submit(specPath, SubmitOptions{Now: func() time.Time { return now }, JobID: "job_submit_resume"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if submitResult.Status != queue.StatusBlockedDecision {
		t.Fatalf("expected blocked_decision, got %s", submitResult.Status)
	}

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	checkpoints, err := r.ListCheckpoints("job_submit_resume")
	if err != nil {
		t.Fatalf("list checkpoints: %v", err)
	}
	decisionID := ""
	for _, cp := range checkpoints {
		if cp.Type == "decision-needed" {
			decisionID = cp.CheckpointID
			break
		}
	}
	if decisionID == "" {
		t.Fatal("expected decision checkpoint")
	}
	if _, err := r.ApproveCheckpoint("job_submit_resume", decisionID, "approved", "lead"); err != nil {
		t.Fatalf("approve checkpoint: %v", err)
	}

	resumeResult, err := Resume("job_submit_resume", ResumeOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resumeResult.Status != queue.StatusCompleted {
		t.Fatalf("expected completed after resume, got %s", resumeResult.Status)
	}

	state, err := r.Recover("job_submit_resume")
	if err != nil {
		t.Fatalf("recover resumed job: %v", err)
	}
	if state.Status != queue.StatusCompleted {
		t.Fatalf("expected completed state after resume, got %s", state.Status)
	}

	cfg, err := LoadRuntimeConfig(s, "job_submit_resume")
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected runtime config")
	}
	if cfg.NextStepIndex != 3 {
		t.Fatalf("expected next_step_index=3, got %d", cfg.NextStepIndex)
	}
}

func TestSubmitEnforcesBudgetFromJobSpec(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 2, 20, 0, 0, time.UTC)

	specPath := filepath.Join(t.TempDir(), "jobspec_budget.yaml")
	if err := os.WriteFile(specPath, []byte(`schema_id: wrkr.jobspec
schema_version: v1
created_at: "2026-02-14T02:20:00Z"
producer_version: test
name: submit-budget
objective: test budget enforcement
inputs:
  steps:
    - id: one
      summary: first
      command: "true"
      executed: true
    - id: two
      summary: second
      command: "true"
      executed: true
expected_artifacts: []
adapter: { name: reference }
budgets:
  max_wall_time_seconds: 100
  max_retries: 1
  max_step_count: 1
  max_tool_calls: 10
checkpoint_policy:
  min_interval_seconds: 1
  required_types: [plan, progress, blocked]
environment_fingerprint:
  rules: [go_version]
`), 0o600); err != nil {
		t.Fatalf("write jobspec: %v", err)
	}

	_, err := Submit(specPath, SubmitOptions{Now: func() time.Time { return now }, JobID: "job_submit_budget"})
	if err == nil {
		t.Fatal("expected budget exceeded error")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) {
		t.Fatalf("expected WrkrError, got %T", err)
	}
	if werr.Code != wrkrerrors.EBudgetExceeded {
		t.Fatalf("expected E_BUDGET_EXCEEDED, got %s", werr.Code)
	}
}
