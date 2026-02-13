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

func TestAcceptInitWritesConfig(t *testing.T) {
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
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	now := time.Date(2026, 2, 13, 23, 0, 0, 0, time.UTC)

	configPath := filepath.Join(workspace, "accept.yaml")
	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"accept", "init", "--path", configPath, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("accept init failed: %d %s", code, errBuf.String())
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config not written: %v", err)
	}
}

func TestAcceptRunCIAndReport(t *testing.T) {
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
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	now := time.Date(2026, 2, 13, 23, 0, 0, 0, time.UTC)

	setupEpic5Job(t, "job_cli_accept", now)

	configPath := filepath.Join(workspace, "accept.yaml")
	if err := os.WriteFile(configPath, []byte(`schema_id: wrkr.accept_config
schema_version: v1
required_artifacts:
  - reports/out.md
test_command: "true"
lint_command: "true"
path_rules:
  max_artifact_paths: 10
  forbidden_prefixes: []
  allowed_prefixes:
    - reports/
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	outDir := filepath.Join(workspace, "out")
	junitPath := filepath.Join(workspace, "report", "accept.junit.xml")
	stepSummary := filepath.Join(workspace, "gh", "step-summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", stepSummary)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"accept", "run", "job_cli_accept", "--config", configPath, "--ci", "--junit", junitPath, "--out-dir", outDir, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("accept run failed: %d %s", code, errBuf.String())
	}
	for _, needle := range []string{"\"job_id\": \"job_cli_accept\"", "\"summary_json_path\":", "\"summary_markdown_path\":"} {
		if !strings.Contains(out.String(), needle) {
			t.Fatalf("expected %s in output: %s", needle, out.String())
		}
	}
	if _, err := os.Stat(junitPath); err != nil {
		t.Fatalf("expected junit output: %v", err)
	}
	if _, err := os.Stat(stepSummary); err != nil {
		t.Fatalf("expected github step summary output: %v", err)
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"report", "github", "job_cli_accept", "--out-dir", outDir, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("report github failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"summary_markdown_path\":") {
		t.Fatalf("expected summary output: %s", out.String())
	}
}

func TestAcceptRunFailureExitCode(t *testing.T) {
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
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	now := time.Date(2026, 2, 13, 23, 0, 0, 0, time.UTC)

	setupEpic5Job(t, "job_cli_accept_fail", now)

	configPath := filepath.Join(workspace, "accept.yaml")
	if err := os.WriteFile(configPath, []byte(`schema_id: wrkr.accept_config
schema_version: v1
required_artifacts:
  - reports/out.md
test_command: "false"
lint_command: "true"
path_rules:
  max_artifact_paths: 0
  forbidden_prefixes: []
  allowed_prefixes: []
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"accept", "run", "job_cli_accept_fail", "--config", configPath, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 5 {
		t.Fatalf("expected exit 5 on acceptance failure, got %d (stderr=%s)", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "E_ACCEPT_TEST_FAIL") {
		t.Fatalf("expected E_ACCEPT_TEST_FAIL in output: %s", out.String())
	}
}

func setupEpic5Job(t *testing.T, jobID string, now time.Time) {
	t.Helper()

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob(jobID); err != nil {
		t.Fatalf("init job: %v", err)
	}
	if _, err := r.ChangeStatus(jobID, queue.StatusRunning); err != nil {
		t.Fatalf("status running: %v", err)
	}
	if _, err := r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:    "progress",
		Summary: "produced report",
		ArtifactsDelta: v1.ArtifactsDelta{
			Added:   []string{"reports/out.md"},
			Changed: []string{},
			Removed: []string{},
		},
	}); err != nil {
		t.Fatalf("emit checkpoint: %v", err)
	}
}
