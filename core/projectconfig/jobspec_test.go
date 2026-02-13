package projectconfig

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitAndLoadJobSpec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobspec.yaml")
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	written, err := InitJobSpec(path, false, now, "test")
	if err != nil {
		t.Fatalf("InitJobSpec: %v", err)
	}
	if written != path {
		t.Fatalf("expected path %s, got %s", path, written)
	}
	if _, err := os.Stat(path); err != nil {
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
