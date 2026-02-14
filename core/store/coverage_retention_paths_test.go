package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreEventCoveragePaths(t *testing.T) {
	t.Parallel()

	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Date(2026, 2, 14, 15, 0, 0, 0, time.UTC)

	if _, err := s.AppendEvent("job_cov_events", "step", map[string]any{"x": 1}, now); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	if _, err := s.AppendEventCAS("job_cov_events", "step", map[string]any{"x": 2}, 0, now); !errors.Is(err, ErrCASConflict) {
		t.Fatalf("expected ErrCASConflict, got %v", err)
	}

	eventsPath := filepath.Join(s.JobDir("job_cov_events"), "events.jsonl")
	raw := []byte(
		`{"seq":2,"created_at":"2026-02-14T15:00:01Z","type":"b"}` + "\n" +
			`{"seq":1,"created_at":"2026-02-14T15:00:00Z","type":"a"}` + "\n" +
			`{"seq":3`,
	)
	if err := os.WriteFile(eventsPath, raw, 0o600); err != nil {
		t.Fatalf("write events fixture: %v", err)
	}

	events, err := s.LoadEvents("job_cov_events")
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 parsed events, got %d", len(events))
	}
	if events[0].Seq != 1 || events[1].Seq != 2 {
		t.Fatalf("expected sorted seq order, got %+v", events)
	}
}

func TestRetentionHelperCoveragePaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 2, 14, 15, 10, 0, 0, time.UTC)

	jobsRoot := filepath.Join(root, "jobs")
	if err := os.MkdirAll(filepath.Join(jobsRoot, "job_a"), 0o750); err != nil {
		t.Fatalf("mkdir job_a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jobsRoot, "job_a", "events.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write job_a events: %v", err)
	}

	checked, err := collectJobCandidates(root, 0, now, func(candidate) {})
	if err != nil {
		t.Fatalf("collectJobCandidates maxAge=0: %v", err)
	}
	if checked != 1 {
		t.Fatalf("expected checked=1, got %d", checked)
	}

	filesDir := filepath.Join(root, "files")
	if err := os.MkdirAll(filesDir, 0o750); err != nil {
		t.Fatalf("mkdir files dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filesDir, "a.zip"), []byte("a"), 0o600); err != nil {
		t.Fatalf("write a.zip: %v", err)
	}
	if _, _, err := collectFileInfos(filesDir, "["); err == nil {
		t.Fatal("expected invalid glob error")
	}

	infos, checked, err := collectFileInfos(filesDir, "*.zip")
	if err != nil {
		t.Fatalf("collectFileInfos: %v", err)
	}
	if checked != 1 || len(infos) != 1 {
		t.Fatalf("expected one checked and one match, got checked=%d len=%d", checked, len(infos))
	}

	recursive, checked, err := collectFileInfosRecursive(filepath.Join(root, "missing-dir"))
	if err != nil {
		t.Fatalf("collectFileInfosRecursive missing: %v", err)
	}
	if checked != 0 || len(recursive) != 0 {
		t.Fatalf("expected missing recursive dir to be empty, got checked=%d len=%d", checked, len(recursive))
	}

	var got []candidate
	add := func(c candidate) { got = append(got, c) }
	applyAgeCandidates(infos, time.Hour, now.Add(2*time.Hour), "jobpack", add)
	if len(got) != 1 || got[0].reason != "age" {
		t.Fatalf("expected age candidate, got %+v", got)
	}

	got = nil
	applyCountCandidates([]fileInfo{
		{path: "a", modTime: now.Add(time.Hour)},
		{path: "b", modTime: now},
	}, 1, "report", add)
	if len(got) != 1 || got[0].path != "b" || got[0].reason != "count" {
		t.Fatalf("expected count candidate for older file, got %+v", got)
	}

	emptyDir := filepath.Join(root, "empty")
	if err := os.MkdirAll(emptyDir, 0o750); err != nil {
		t.Fatalf("mkdir empty dir: %v", err)
	}
	latest, size, err := latestModAndSize(emptyDir)
	if err != nil {
		t.Fatalf("latestModAndSize empty dir: %v", err)
	}
	if !latest.Equal(time.Unix(0, 0).UTC()) || size != 0 {
		t.Fatalf("expected epoch/0 for empty dir, got latest=%s size=%d", latest, size)
	}

	filePath := filepath.Join(root, "to-remove.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write remove file: %v", err)
	}
	if err := removePath(filePath); err != nil {
		t.Fatalf("removePath file: %v", err)
	}
	if err := removePath(filepath.Join(root, "does-not-exist")); err != nil {
		t.Fatalf("removePath missing: %v", err)
	}
}

