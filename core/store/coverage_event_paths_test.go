package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreEventCoverageAdditionalPaths(t *testing.T) {
	t.Parallel()

	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Date(2026, 2, 14, 21, 30, 0, 0, time.UTC)

	if _, err := s.AppendEvent("bad/job", "event", map[string]any{}, now); err == nil {
		t.Fatal("expected invalid job id append failure")
	}

	if err := s.EnsureJob("job_empty_events"); err != nil {
		t.Fatalf("EnsureJob: %v", err)
	}
	events, err := s.LoadEvents("job_empty_events")
	if err != nil {
		t.Fatalf("LoadEvents missing file path: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected empty events on missing file, got %d", len(events))
	}
}

func TestStoreMiscCoveragePaths(t *testing.T) {
	t.Parallel()

	if _, err := New("bad\x00root"); err == nil {
		t.Fatal("expected invalid store root failure")
	}

	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dirPath := filepath.Join(s.Root(), "remove-dir")
	if err := os.MkdirAll(filepath.Join(dirPath, "nested"), 0o750); err != nil {
		t.Fatalf("mkdir nested remove dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, "nested", "x.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	if err := removePath(dirPath); err != nil {
		t.Fatalf("removePath dir: %v", err)
	}
}
