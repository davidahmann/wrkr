package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/pack"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/schema/validate"
	"github.com/davidahmann/wrkr/core/store"
)

func TestBuildAndWriteGitHubSummary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 22, 0, 0, 0, time.UTC)

	setupReportJob(t, "job_report", now)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	acceptPath := filepath.Join(s.JobDir("job_report"), "accept_result.json")
	acceptRaw := []byte(`{"schema_id":"wrkr.accept_result","schema_version":"v1","created_at":"2026-02-13T22:00:00Z","producer_version":"test","job_id":"job_report","checks_run":3,"checks_passed":2,"failures":[{"check":"test_command","message":"failed"}],"reason_codes":["E_ACCEPT_TEST_FAIL"]}`)
	if err := os.WriteFile(acceptPath, acceptRaw, 0o600); err != nil {
		t.Fatalf("write accept_result: %v", err)
	}

	exported, err := pack.ExportJobpack("job_report", pack.ExportOptions{OutDir: t.TempDir(), ProducerVersion: "test", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("export jobpack: %v", err)
	}

	a, err := BuildGitHubSummaryFromJobpack(exported.Path, SummaryOptions{Now: func() time.Time { return now }, ProducerVersion: "test"})
	if err != nil {
		t.Fatalf("build summary A: %v", err)
	}
	b, err := BuildGitHubSummaryFromJobpack(exported.Path, SummaryOptions{Now: func() time.Time { return now }, ProducerVersion: "test"})
	if err != nil {
		t.Fatalf("build summary B: %v", err)
	}
	if a.Markdown != b.Markdown {
		t.Fatal("expected deterministic markdown")
	}
	if !strings.Contains(a.Markdown, "Final Checkpoint") {
		t.Fatalf("expected final checkpoint section: %s", a.Markdown)
	}

	stepSummaryPath := filepath.Join(t.TempDir(), "step", "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", stepSummaryPath)
	written, err := WriteGitHubSummary(a, t.TempDir())
	if err != nil {
		t.Fatalf("write summary: %v", err)
	}
	if written.JSONPath == "" || written.MarkdownPath == "" || written.StepSummaryPath == "" {
		t.Fatalf("expected output paths, got %+v", written)
	}

	jsonRaw, err := os.ReadFile(written.JSONPath)
	if err != nil {
		t.Fatalf("read summary json: %v", err)
	}
	if err := validate.ValidateBytes(validate.GitHubSummarySchemaRel, jsonRaw); err != nil {
		t.Fatalf("summary schema invalid: %v", err)
	}
	mdRaw, err := os.ReadFile(written.MarkdownPath)
	if err != nil {
		t.Fatalf("read summary markdown: %v", err)
	}
	if !strings.Contains(string(mdRaw), "Wrkr GitHub Summary") {
		t.Fatalf("unexpected markdown content: %s", string(mdRaw))
	}
	stepRaw, err := os.ReadFile(stepSummaryPath)
	if err != nil {
		t.Fatalf("read step summary: %v", err)
	}
	if !strings.Contains(string(stepRaw), "Wrkr GitHub Summary") {
		t.Fatalf("unexpected step summary content: %s", string(stepRaw))
	}

	var decoded map[string]any
	if err := json.Unmarshal(jsonRaw, &decoded); err != nil {
		t.Fatalf("decode summary json: %v", err)
	}
	if decoded["job_id"] != "job_report" {
		t.Fatalf("expected job_report, got %v", decoded["job_id"])
	}
}

func setupReportJob(t *testing.T, jobID string, now time.Time) {
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
		t.Fatalf("status: %v", err)
	}
	if _, err := r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:    "progress",
		Summary: "generated reports",
		ArtifactsDelta: v1.ArtifactsDelta{
			Added:   []string{"reports/a.md", "reports/b.md"},
			Changed: []string{},
			Removed: []string{},
		},
	}); err != nil {
		t.Fatalf("emit checkpoint 1: %v", err)
	}
	if _, err := r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:    "completed",
		Summary: "done",
		ArtifactsDelta: v1.ArtifactsDelta{
			Added:   []string{},
			Changed: []string{"reports/a.md"},
			Removed: []string{},
		},
	}); err != nil {
		t.Fatalf("emit checkpoint 2: %v", err)
	}
}
