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

func writeJobSpec(t *testing.T, path string, adapter string) {
	t.Helper()
	raw := `schema_id: wrkr.jobspec
schema_version: v1
created_at: "2026-02-14T21:00:00Z"
producer_version: test
name: dispatch-adapter-test
objective: adapter coverage
inputs:
  steps:
    - id: one
      summary: one
      command: "true"
      executed: true
expected_artifacts: []
adapter:
  name: ` + adapter + `
budgets:
  max_wall_time_seconds: 100
  max_retries: 2
  max_step_count: 5
  max_tool_calls: 5
checkpoint_policy:
  min_interval_seconds: 1
  required_types: [plan, progress, completed]
environment_fingerprint:
  rules: [go_version]
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write jobspec %s: %v", path, err)
	}
}

func TestSubmitAdapterCoveragePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	workspace := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	now := time.Date(2026, 2, 14, 21, 0, 0, 0, time.UTC)
	writeJobSpec(t, "jobspec_noop.yaml", "noop")
	writeJobSpec(t, "jobspec_unsupported.yaml", "not-real")

	okResult, err := Submit("jobspec_noop.yaml", SubmitOptions{Now: func() time.Time { return now }, JobID: "job_submit_noop"})
	if err != nil {
		t.Fatalf("Submit noop: %v", err)
	}
	if okResult.Status != queue.StatusCompleted {
		t.Fatalf("expected noop submit completed, got %+v", okResult)
	}

	_, err = Submit("jobspec_unsupported.yaml", SubmitOptions{Now: func() time.Time { return now }, JobID: "job_submit_bad_adapter"})
	if err == nil {
		t.Fatal("expected unsupported adapter submit error")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EInvalidInputSchema {
		t.Fatalf("expected E_INVALID_INPUT_SCHEMA, got %v", err)
	}
}

func TestResumeAdapterCoveragePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 21, 10, 0, 0, time.UTC)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_resume_bad_adapter"); err != nil {
		t.Fatalf("InitJob: %v", err)
	}
	if _, err := r.ChangeStatus("job_resume_bad_adapter", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus running: %v", err)
	}
	if _, err := r.ChangeStatus("job_resume_bad_adapter", queue.StatusPaused); err != nil {
		t.Fatalf("ChangeStatus paused: %v", err)
	}
	if err := SaveRuntimeConfig(s, "job_resume_bad_adapter", RuntimeConfig{
		Adapter: "not-real",
		Inputs:  map[string]any{},
	}, now); err != nil {
		t.Fatalf("SaveRuntimeConfig: %v", err)
	}

	result, err := Resume("job_resume_bad_adapter", ResumeOptions{Now: func() time.Time { return now }})
	if err == nil {
		t.Fatal("expected resume error for unsupported adapter")
	}
	if result.JobID != "job_resume_bad_adapter" || result.Adapter != "not-real" {
		t.Fatalf("unexpected resume result on adapter error: %+v", result)
	}

	cfgPath := filepath.Join(s.JobDir("job_resume_bad_adapter"), "runtime_config.json")
	if err := os.WriteFile(cfgPath, []byte("{bad"), 0o600); err != nil {
		t.Fatalf("write malformed runtime config: %v", err)
	}
	if _, err := Resume("job_resume_bad_adapter", ResumeOptions{Now: func() time.Time { return now }}); err == nil {
		t.Fatal("expected resume error for malformed runtime config")
	}
}

