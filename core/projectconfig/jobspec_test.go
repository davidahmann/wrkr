package projectconfig

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitAndLoadJobSpec(t *testing.T) {
	wd := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})

	path := "jobspec.yaml"
	absPath := filepath.Join(wd, path)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	written, err := InitJobSpec(path, false, now, "test")
	if err != nil {
		t.Fatalf("InitJobSpec: %v", err)
	}
	expectedResolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		expectedResolved = absPath
	}
	writtenResolved, err := filepath.EvalSymlinks(written)
	if err != nil {
		writtenResolved = written
	}
	if writtenResolved != expectedResolved {
		t.Fatalf("expected path %s, got %s", expectedResolved, writtenResolved)
	}
	if _, err := os.Stat(absPath); err != nil {
		t.Fatalf("jobspec missing: %v", err)
	}

	spec, err := LoadJobSpec(path)
	if err != nil {
		t.Fatalf("LoadJobSpec: %v", err)
	}
	if spec.SchemaID != "wrkr.jobspec" {
		t.Fatalf("unexpected schema: %s", spec.SchemaID)
	}
	if spec.Adapter.Name != "reference" {
		t.Fatalf("unexpected adapter: %s", spec.Adapter.Name)
	}
}

func TestNormalizeJobID(t *testing.T) {
	if got := NormalizeJobID("  Foo Bar/1 "); got != "foo_bar_1" {
		t.Fatalf("unexpected normalized id: %s", got)
	}
}
