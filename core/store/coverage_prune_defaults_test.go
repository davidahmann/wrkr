package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneDefaultRootsCoveragePaths(t *testing.T) {
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

	now := time.Date(2026, 2, 14, 22, 20, 0, 0, time.UTC)
	defaultStoreRoot := filepath.Join(home, ".wrkr")
	defaultOutRoot := filepath.Join(workspace, "wrkr-out")

	if err := os.MkdirAll(filepath.Join(defaultStoreRoot, "jobs", "job_default"), 0o750); err != nil {
		t.Fatalf("mkdir default store job: %v", err)
	}
	if err := os.WriteFile(filepath.Join(defaultStoreRoot, "jobs", "job_default", "events.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write default store events: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(defaultOutRoot, "jobpacks"), 0o750); err != nil {
		t.Fatalf("mkdir default out jobpacks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(defaultOutRoot, "jobpacks", "jp.zip"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write default out jobpack: %v", err)
	}

	report, err := Prune(PruneOptions{
		Now:           func() time.Time { return now },
		DryRun:        true,
		JobMaxAge:     time.Hour,
		JobpackMaxAge: time.Hour,
		MaxJobpacks:   0,
	})
	if err != nil {
		t.Fatalf("Prune default roots: %v", err)
	}
	if report.Checked == 0 {
		t.Fatalf("expected checked paths from default roots, got %+v", report)
	}
}

