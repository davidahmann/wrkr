package store

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func TestAppendEventCASConflict(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)

	ev1, err := s.AppendEvent("job_4", "a", nil, now)
	if err != nil {
		t.Fatalf("append event 1: %v", err)
	}

	if _, err := s.AppendEventCAS("job_4", "b", nil, ev1.Seq-1, now.Add(time.Second)); !errors.Is(err, ErrCASConflict) {
		t.Fatalf("expected ErrCASConflict, got %v", err)
	}
}

func TestAppendEventReclaimsStaleLock(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)

	if err := s.EnsureJob("job_5"); err != nil {
		t.Fatalf("ensure job: %v", err)
	}

	lockPath := filepath.Join(s.Root(), "jobs", "job_5", "append.lock")
	cmd := exec.Command("sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatalf("create dead pid: %v", err)
	}
	if cmd.Process == nil {
		t.Fatal("expected process info")
	}
	owner := "pid=" + strconv.Itoa(cmd.Process.Pid) + ";ts=1\n"
	if err := os.WriteFile(lockPath, []byte(owner), 0o600); err != nil {
		t.Fatalf("write stale lock: %v", err)
	}
	stale := time.Now().Add(-3 * time.Minute)
	if err := os.Chtimes(lockPath, stale, stale); err != nil {
		t.Fatalf("chtimes stale lock: %v", err)
	}

	if _, err := s.AppendEvent("job_5", "recovered", nil, now); err != nil {
		t.Fatalf("append with stale lock should succeed, got %v", err)
	}
}
