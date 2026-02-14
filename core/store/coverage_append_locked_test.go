package store

import (
	"testing"
	"time"
)

func TestAppendEventLockedCoveragePaths(t *testing.T) {
	t.Parallel()

	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Date(2026, 2, 14, 23, 20, 0, 0, time.UTC)

	if _, err := s.appendEventLocked("bad/job", "event", nil, now, 1); err == nil {
		t.Fatal("expected appendEventLocked invalid job id failure")
	}

	if err := s.EnsureJob("job_append_locked_cov"); err != nil {
		t.Fatalf("EnsureJob: %v", err)
	}
	event, err := s.appendEventLocked("job_append_locked_cov", "event", map[string]any{"ok": true}, now, 0)
	if err != nil {
		t.Fatalf("appendEventLocked seq<=0 normalization: %v", err)
	}
	if event.Seq != 1 {
		t.Fatalf("expected normalized seq=1, got %d", event.Seq)
	}

	if _, err := s.appendEventLocked("job_append_locked_cov", "event", make(chan int), now, 2); err == nil {
		t.Fatal("expected appendEventLocked payload marshal error")
	}
}

