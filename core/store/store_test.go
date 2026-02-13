package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *LocalStore {
	t.Helper()

	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return s
}

func TestAppendAndLoadEvents(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)

	if _, err := s.AppendEvent("job_1", "started", map[string]any{"step": 1}, now); err != nil {
		t.Fatalf("append event 1: %v", err)
	}
	if _, err := s.AppendEvent("job_1", "progress", map[string]any{"step": 2}, now.Add(time.Second)); err != nil {
		t.Fatalf("append event 2: %v", err)
	}

	events, err := s.LoadEvents("job_1")
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Seq != 1 || events[1].Seq != 2 {
		t.Fatalf("unexpected sequence values: %+v", events)
	}
}

func TestCrashPartialLineDoesNotCorruptCommittedEvents(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)

	if _, err := s.AppendEvent("job_2", "a", nil, now); err != nil {
		t.Fatalf("append event 1: %v", err)
	}
	if _, err := s.AppendEvent("job_2", "b", nil, now.Add(time.Second)); err != nil {
		t.Fatalf("append event 2: %v", err)
	}

	path := filepath.Join(s.Root(), "jobs", "job_2", "events.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open events file: %v", err)
	}
	if _, err := f.WriteString("{\"seq\":3"); err != nil {
		t.Fatalf("write partial line: %v", err)
	}
	_ = f.Close()

	events, err := s.LoadEvents("job_2")
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 committed events, got %d", len(events))
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	state := map[string]any{"status": "running", "count": 3}

	if err := s.SaveSnapshot("job_3", 7, state, now); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	snap, err := s.LoadSnapshot("job_3")
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("expected snapshot")
	}
	if snap.LastSeq != 7 {
		t.Fatalf("expected last seq 7, got %d", snap.LastSeq)
	}
}
