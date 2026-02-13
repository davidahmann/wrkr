package accept

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/schema/validate"
	"github.com/davidahmann/wrkr/core/store"
)

func TestRunWritesAcceptanceResult(t *testing.T) {
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
	now := time.Date(2026, 2, 13, 21, 0, 0, 0, time.UTC)

	setupAcceptJob(t, "job_accept_run", now)

	configPath := "accept.yaml"
	if err := os.WriteFile(filepath.Join(workspace, configPath), []byte(`schema_id: wrkr.accept_config
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

	result, err := Run("job_accept_run", RunOptions{Now: func() time.Time { return now }, ProducerVersion: "test", ConfigPath: configPath, WorkDir: workspace})
	if err != nil {
		t.Fatalf("run accept: %v", err)
	}
	if Failed(result.Result) {
		t.Fatalf("expected passing acceptance result: %+v", result.Result)
	}

	raw, err := os.ReadFile(result.ResultPath)
	if err != nil {
		t.Fatalf("read accept_result: %v", err)
	}
	if err := validate.ValidateBytes(validate.AcceptResultSchemaRel, raw); err != nil {
		t.Fatalf("accept_result schema invalid: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode accept_result: %v", err)
	}
	if decoded["job_id"] != "job_accept_run" {
		t.Fatalf("expected job_accept_run in accept_result, got %v", decoded["job_id"])
	}
}

func TestRunFailureCode(t *testing.T) {
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
	now := time.Date(2026, 2, 13, 21, 0, 0, 0, time.UTC)

	setupAcceptJob(t, "job_accept_fail", now)

	configPath := "accept.yaml"
	if err := os.WriteFile(filepath.Join(workspace, configPath), []byte(`schema_id: wrkr.accept_config
schema_version: v1
required_artifacts:
  - reports/missing.md
test_command: "true"
lint_command: "true"
path_rules:
  max_artifact_paths: 0
  forbidden_prefixes: []
  allowed_prefixes: []
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	result, err := Run("job_accept_fail", RunOptions{Now: func() time.Time { return now }, ProducerVersion: "test", ConfigPath: configPath, WorkDir: workspace})
	if err != nil {
		t.Fatalf("run accept: %v", err)
	}
	if !Failed(result.Result) {
		t.Fatalf("expected acceptance failures")
	}
	if got := FailureCode(result.Result); got != wrkrerrors.EAcceptMissingArtifact {
		t.Fatalf("expected E_ACCEPT_MISSING_ARTIFACT, got %s", got)
	}
}

func setupAcceptJob(t *testing.T, jobID string, now time.Time) {
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
		Summary: "created artifact",
		ArtifactsDelta: v1.ArtifactsDelta{
			Added:   []string{},
			Changed: []string{},
			Removed: []string{},
		},
	}); err != nil {
		t.Fatalf("emit checkpoint: %v", err)
	}
	if _, err := r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:    "progress",
		Summary: "artifact path update",
		ArtifactsDelta: v1.ArtifactsDelta{
			Added:   []string{"reports/out.md"},
			Changed: []string{},
			Removed: []string{},
		},
	}); err != nil {
		t.Fatalf("emit artifact checkpoint: %v", err)
	}
}
