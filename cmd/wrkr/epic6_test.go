package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
)

func TestHelpIncludesEpic6Commands(t *testing.T) {
	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"help"}, &out, &errBuf, time.Now)
	if code != 0 {
		t.Fatalf("help failed: %d %s", code, errBuf.String())
	}
	for _, cmd := range []string{"demo", "init", "submit", "pause", "cancel", "wrap", "bridge work-item", "serve", "doctor"} {
		if !strings.Contains(out.String(), cmd) {
			t.Fatalf("missing command %q in help output: %s", cmd, out.String())
		}
	}
}

func TestDemoAndDoctorCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 3, 0, 0, 0, time.UTC)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"demo", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("demo failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"job_id\":") || !strings.Contains(out.String(), "\"jobpack\":") {
		t.Fatalf("unexpected demo output: %s", out.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"doctor", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("doctor failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"checks\"") {
		t.Fatalf("unexpected doctor output: %s", out.String())
	}
}

func TestSubmitAndBridgeWorkItemDryRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 3, 15, 0, 0, time.UTC)

	specPath := filepath.Join(t.TempDir(), "jobspec.yaml")
	spec := `schema_id: wrkr.jobspec
schema_version: v1
created_at: "2026-02-14T03:15:00Z"
producer_version: test
name: epic6-submit
objective: run steps
inputs:
  steps:
    - id: build
      summary: build
      command: "true"
      artifacts: [reports/out.md]
      executed: true
    - id: review
      summary: approve me
      decision_needed: true
      required_action: approval
      executed: false
expected_artifacts: [reports/out.md]
adapter: { name: reference }
budgets:
  max_wall_time_seconds: 100
  max_retries: 1
  max_step_count: 10
  max_tool_calls: 10
checkpoint_policy:
  min_interval_seconds: 1
  required_types: [plan, progress, decision-needed]
environment_fingerprint:
  rules: [go_version]
`
	if err := os.WriteFile(specPath, []byte(spec), 0o600); err != nil {
		t.Fatalf("write jobspec: %v", err)
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"submit", specPath, "--job-id", "job_epic6_submit", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("submit failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"status\": \"blocked_decision\"") {
		t.Fatalf("expected blocked_decision submit output: %s", out.String())
	}

	r, _, err := openRunner(func() time.Time { return now })
	if err != nil {
		t.Fatalf("openRunner: %v", err)
	}
	checkpoints, err := r.ListCheckpoints("job_epic6_submit")
	if err != nil {
		t.Fatalf("ListCheckpoints: %v", err)
	}
	var decisionID string
	for _, cp := range checkpoints {
		if cp.Type == "decision-needed" {
			decisionID = cp.CheckpointID
			break
		}
	}
	if decisionID == "" {
		t.Fatal("expected decision-needed checkpoint")
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"bridge", "work-item", "job_epic6_submit", "--checkpoint", decisionID, "--dry-run", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("bridge dry-run failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"next_commands\"") {
		t.Fatalf("expected next_commands in bridge output: %s", out.String())
	}
}

func TestPauseCancelAndWrap(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 3, 30, 0, 0, time.UTC)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_epic6_pause"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_epic6_pause", queue.StatusRunning); err != nil {
		t.Fatalf("running: %v", err)
	}
	if _, err := r.EmitCheckpoint("job_epic6_pause", runner.CheckpointInput{
		Type:    "progress",
		Summary: "running",
		ArtifactsDelta: v1.ArtifactsDelta{
			Added: []string{"reports/out.md"},
		},
	}); err != nil {
		t.Fatalf("emit checkpoint: %v", err)
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"pause", "job_epic6_pause", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("pause failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"status\": \"paused\"") {
		t.Fatalf("unexpected pause output: %s", out.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"cancel", "job_epic6_pause", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("cancel failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"status\": \"canceled\"") {
		t.Fatalf("unexpected cancel output: %s", out.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"wrap", "--job-id", "job_epic6_wrap", "--", "sh", "-lc", "printf wrapped", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("wrap failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"jobpack_path\":") {
		t.Fatalf("unexpected wrap output: %s", out.String())
	}

	script := filepath.Join(t.TempDir(), "fixture.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env sh\nprintf fixture\n"), 0o700); err != nil {
		t.Fatalf("write fixture script: %v", err)
	}
	out.Reset()
	errBuf.Reset()
	code = run([]string{"wrap", "--job-id", "job_epic6_wrap_fixture", "--", script, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("wrap fixture failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"job_id\": \"job_epic6_wrap_fixture\"") {
		t.Fatalf("unexpected wrap fixture output: %s", out.String())
	}
}
