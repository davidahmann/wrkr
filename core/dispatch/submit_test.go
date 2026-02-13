package dispatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/queue"
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
}
