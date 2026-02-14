package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultRootAndNewEmptyRootPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root, err := DefaultRoot()
	if err != nil {
		t.Fatalf("DefaultRoot: %v", err)
	}
	if root != filepath.Join(home, ".wrkr") {
		t.Fatalf("unexpected default root: %s", root)
	}

	s, err := New("")
	if err != nil {
		t.Fatalf("New with empty root: %v", err)
	}
	if s.Root() != root {
		t.Fatalf("expected root %s, got %s", root, s.Root())
	}
}

func TestJobExistsAndValidationPaths(t *testing.T) {
	t.Parallel()

	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	exists, err := s.JobExists("job_exists_paths")
	if err != nil {
		t.Fatalf("JobExists missing: %v", err)
	}
	if exists {
		t.Fatal("expected missing job to report not exists")
	}

	if err := s.EnsureJob("job_exists_paths"); err != nil {
		t.Fatalf("EnsureJob: %v", err)
	}
	exists, err = s.JobExists("job_exists_paths")
	if err != nil {
		t.Fatalf("JobExists existing: %v", err)
	}
	if !exists {
		t.Fatal("expected ensured job to exist")
	}

	if _, err := s.JobExists("../bad"); err == nil {
		t.Fatal("expected JobExists to reject invalid job id")
	}
	if err := s.EnsureJob("bad/path"); err == nil {
		t.Fatal("expected EnsureJob to reject path separators")
	}
	if _, err := s.safeJobPath("job_exists_paths", ""); err == nil {
		t.Fatal("expected safeJobPath empty leaf error")
	}
}

func TestAppendEventPayloadMarshalErrorAndLoadEventsDecodeError(t *testing.T) {
	t.Parallel()

	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := s.AppendEvent("job_payload_err", "bad", map[string]any{"x": make(chan int)}, time.Now().UTC()); err == nil {
		t.Fatal("expected append payload marshal error")
	}

	if err := s.EnsureJob("job_decode_err"); err != nil {
		t.Fatalf("EnsureJob: %v", err)
	}
	eventsPath := filepath.Join(s.Root(), "jobs", "job_decode_err", "events.jsonl")
	if err := os.WriteFile(eventsPath, []byte("{not-json}\n"), 0o600); err != nil {
		t.Fatalf("write events fixture: %v", err)
	}
	if _, err := s.LoadEvents("job_decode_err"); err == nil {
		t.Fatal("expected decode error from malformed events line")
	}

	if err := s.EnsureJob("job_empty_events"); err != nil {
		t.Fatalf("EnsureJob empty events: %v", err)
	}
	events, err := s.LoadEvents("job_empty_events")
	if err != nil {
		t.Fatalf("LoadEvents empty events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected empty events, got %d", len(events))
	}
}

func TestSnapshotCoveragePaths(t *testing.T) {
	t.Parallel()

	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := s.SaveSnapshot("job_snapshot_paths", 1, map[string]any{"bad": make(chan int)}, time.Now().UTC()); err == nil {
		t.Fatal("expected snapshot marshal state error")
	}

	missing, err := s.LoadSnapshot("job_snapshot_paths")
	if err != nil {
		t.Fatalf("LoadSnapshot missing: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil missing snapshot, got %+v", missing)
	}

	if err := s.EnsureJob("job_snapshot_paths"); err != nil {
		t.Fatalf("EnsureJob: %v", err)
	}
	rawPath := filepath.Join(s.Root(), "jobs", "job_snapshot_paths", "snapshot.json")
	if err := os.WriteFile(rawPath, []byte("{bad-json}"), 0o600); err != nil {
		t.Fatalf("write invalid snapshot: %v", err)
	}
	if _, err := s.LoadSnapshot("job_snapshot_paths"); err == nil || !strings.Contains(err.Error(), "decode snapshot") {
		t.Fatalf("expected decode snapshot error, got %v", err)
	}
}
